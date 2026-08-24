package study

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ai-tutor/internal/extension"
)

// AudioChunk represents a synthesized audio chunk streamed to the client.
type AudioChunk struct {
	Status      string `json:"status"`
	ChunkID     int    `json:"chunk_id"`
	TotalChunks int    `json:"total_chunks,omitempty"`
	Text        string `json:"text"`
	AudioBase64 string `json:"audio_base64"`
	Error       string `json:"error,omitempty"`
}

// SentenceItem holds an individual numbered sentence for TTS processing.
type SentenceItem struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

// AudioOverviewPayload is passed via stdin to extensions/edge_tts_stream.py.
type AudioOverviewPayload struct {
	Voice     string         `json:"voice"`
	Sentences []SentenceItem `json:"sentences"`
}

var markdownCleanupRegex = regexp.MustCompile(`[#*` + "`" + `_~>\[\]\(\)]+`)

// CleanOverviewText removes markdown formatting and invalid/surrogate unicode to make text speech-friendly.
func CleanOverviewText(text string) string {
	var b strings.Builder
	for _, r := range text {
		// Filter out UTF-16 surrogates (0xD800-0xDFFF) and replacement rune errors
		if (r >= 0xD800 && r <= 0xDFFF) || r == '\uFFFD' {
			continue
		}
		b.WriteRune(r)
	}

	cleaned := markdownCleanupRegex.ReplaceAllString(b.String(), " ")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	return strings.TrimSpace(cleaned)
}

// SplitIntoSentences splits spoken text into clean, natural sentence chunks.
func SplitIntoSentences(text string) []string {
	cleaned := CleanOverviewText(text)
	if cleaned == "" {
		return nil
	}

	var sentences []string
	var current strings.Builder

	runes := []rune(cleaned)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		current.WriteRune(r)

		if r == '.' || r == '!' || r == '?' {
			if i+1 == len(runes) || runes[i+1] == ' ' || runes[i+1] == '\n' || runes[i+1] == '\t' || runes[i+1] == '"' || runes[i+1] == '\'' {
				chunk := strings.TrimSpace(current.String())
				if len(chunk) >= 40 || i+1 == len(runes) {
					sentences = append(sentences, chunk)
					current.Reset()
				}
			}
		}
	}

	if current.Len() > 0 {
		remaining := strings.TrimSpace(current.String())
		if remaining != "" {
			if len(sentences) > 0 && len(remaining) < 30 {
				sentences[len(sentences)-1] += " " + remaining
			} else {
				sentences = append(sentences, remaining)
			}
		}
	}

	return sentences
}

// BuildAudioOverviewPrompt creates a prompt for the conversational overview script.
func BuildAudioOverviewPrompt(topicTitle string, contextContent string) string {
	return fmt.Sprintf(`You are an engaging, insightful study host providing a spoken audio overview of this topic.
Write a clear, concise 2 to 3 paragraph audio briefing that sounds natural, warm, and engaging when spoken aloud.

Rules:
- Do NOT use markdown headers, bullet lists, asterisks, citations, or formatting symbols.
- Write in plain, conversational English suitable for text-to-speech narration.
- Use natural punctuation (. , ? !) to guide the speech rhythm and pauses.
- Summarize key concepts, why they matter, and core takeaways.
- Total length: between 120 and 220 words.

Topic: %s

Material:
%s

Spoken Overview:`, topicTitle, contextContent)
}

// GenerateAudioOverview generates a spoken audio overview for a topic and streams chunks via onChunk callback.
func (s *StudyService) GenerateAudioOverview(
	ctx context.Context,
	topicID string,
	notebookID string,
	voice string,
	onChunk func(chunk AudioChunk) error,
) error {
	topicID = strings.TrimSpace(topicID)
	notebookID = strings.TrimSpace(notebookID)
	if voice == "" {
		voice = "en-US-ChristopherNeural"
	}

	if topicID == "" {
		return fmt.Errorf("topic ID is required")
	}
	if s.repo == nil {
		return fmt.Errorf("repository is required")
	}

	bundle, err := s.repo.GetReaderTopicBundle(topicID, notebookID)
	if err != nil {
		return fmt.Errorf("failed to load topic content: %w", err)
	}

	var contentBuilder strings.Builder
	for _, section := range bundle.Sections {
		if strings.TrimSpace(section.Content) != "" {
			contentBuilder.WriteString(section.Content)
			contentBuilder.WriteString("\n\n")
		}
	}

	topicContent := strings.TrimSpace(contentBuilder.String())
	if topicContent == "" {
		return fmt.Errorf("no topic text found to generate audio overview")
	}

	// Prioritize heavy LLM for rich script overview, fallback to fast LLM
	llmProvider := s.heavyLLMProvider
	if llmProvider == nil {
		llmProvider, _ = s.selectLLM(topicContent)
	}
	if llmProvider == nil {
		return fmt.Errorf("no LLM provider available")
	}

	prompt := BuildAudioOverviewPrompt(bundle.TopicTitle, topicContent)
	script, err := llmProvider.GenerateAnswer(prompt)
	if err != nil {
		return fmt.Errorf("failed to generate audio script: %w", err)
	}

	sentences := SplitIntoSentences(script)
	if len(sentences) == 0 {
		return fmt.Errorf("failed to extract sentences from audio script")
	}

	extDir := extension.ResolveExtensionsDir("")
	audioExt := &extension.Extension{
		Manifest: extension.Manifest{ID: "audio_overview", Name: "Audio Overview", Runtime: "python", Entrypoint: "edge_tts_stream.py"},
		Dir:      filepath.Join(extDir, "audio_overview"),
	}
	pythonPath, err := extension.FindExtensionPython(audioExt)
	if err != nil {
		return fmt.Errorf("Python executable not found for Audio Overview: %w", err)
	}

	// Resolve edge_tts_stream.py script path
	scriptCandidates := []string{
		filepath.Join("extensions", "audio_overview", "edge_tts_stream.py"),
		filepath.Join("extensions", "edge_tts_stream.py"),
	}
	if extDir != "" {
		scriptCandidates = append(scriptCandidates,
			filepath.Join(extDir, "audio_overview", "edge_tts_stream.py"),
			filepath.Join(extDir, "edge_tts_stream.py"),
		)
	}

	scriptPath := ""
	for _, candidate := range scriptCandidates {
		if _, err := os.Stat(candidate); err == nil {
			scriptPath = candidate
			break
		}
	}
	if scriptPath == "" {
		scriptPath = filepath.Join("extensions", "audio_overview", "edge_tts_stream.py")
	}

	var sentenceItems []SentenceItem
	for i, sentence := range sentences {
		sentenceItems = append(sentenceItems, SentenceItem{
			ID:   i + 1,
			Text: sentence,
		})
	}

	payload := AudioOverviewPayload{
		Voice:     voice,
		Sentences: sentenceItems,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize audio payload: %w", err)
	}

	runner := extension.NewRunner()
	totalChunks := len(sentenceItems)

	return runner.RunStreamWithInput(ctx, "", pythonPath, payloadBytes, func(line string) error {
		var chunk AudioChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return nil
		}
		chunk.TotalChunks = totalChunks
		if onChunk != nil {
			return onChunk(chunk)
		}
		return nil
	}, scriptPath)
}
