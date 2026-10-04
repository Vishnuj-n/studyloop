package study

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-tutor/internal/extension"
	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"
)

type CompressionChunkInput struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type CompressionPayload struct {
	Rate   float64                 `json:"rate"`
	Chunks []CompressionChunkInput `json:"chunks"`
}

type CompressionChunkResult struct {
	ID               string `json:"id"`
	CompressedText   string `json:"compressed_text"`
	OriginTokens     int    `json:"origin_tokens"`
	CompressedTokens int    `json:"compressed_tokens"`
	Error            string `json:"error,omitempty"`
}

type CompressionOutput struct {
	Results []CompressionChunkResult `json:"results"`
	Error   string                   `json:"error,omitempty"`
}

// CompressTopicChunksAsync triggers background compression for all chunks in a topic
// without blocking the caller. It accepts an optional onComplete callback that is executed
// strictly after compression finishes, guaranteeing sequential execution for dependent tasks like note generation.
func (s *StudyService) CompressTopicChunksAsync(ctx context.Context, topicID string, onComplete func()) {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	s.inFlightCompressMu.Lock()
	if s.inFlightCompress == nil {
		s.inFlightCompress = make(map[string]bool)
	}
	if s.inFlightCompress[topicID] {
		s.inFlightCompressMu.Unlock()
		return
	}
	s.inFlightCompress[topicID] = true
	s.inFlightCompressMu.Unlock()

	go func() {
		defer func() {
			s.inFlightCompressMu.Lock()
			delete(s.inFlightCompress, topicID)
			s.inFlightCompressMu.Unlock()

			if r := recover(); r != nil {
				utils.Errorf("[COMPRESSION] panic in CompressTopicChunksAsync: %v", r)
			}
		}()

		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := s.CompressTopicChunks(bgCtx, topicID); err != nil {
			utils.Warnf("[COMPRESSION] CompressTopicChunks failed for topic %s: %v", topicID, err)
		}

		if onComplete != nil {
			onComplete()
		}
	}()
}

// CompressTopicChunks evaluates whether compression is needed and executes it for the topic.
func (s *StudyService) CompressTopicChunks(ctx context.Context, topicID string) error {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return fmt.Errorf("topic ID is required")
	}

	userSettings, err := s.repo.GetUserSettings()
	if err != nil {
		return fmt.Errorf("failed to load user settings: %w", err)
	}

	mode := strings.ToUpper(strings.TrimSpace(userSettings.PromptCompressionMode))
	if mode == "DISABLED" {
		utils.Infof("[COMPRESSION] compression disabled by user setting")
		return nil
	}

	// Fetch all chunks for this topic
	chunks, err := s.repo.GetChunksForTopic(topicID)
	if err != nil {
		return fmt.Errorf("failed to load topic chunks: %w", err)
	}
	if len(chunks) == 0 {
		return nil
	}

	// Check if all substantive chunks are already compressed
	var uncompressedChunks []models.Chunk
	totalTokens := 0
	for _, c := range chunks {
		words := len(strings.Fields(c.Text))
		totalTokens += int(float64(words) * 1.33)
		if strings.TrimSpace(c.CompressedText) == "" && words >= 25 {
			uncompressedChunks = append(uncompressedChunks, c)
		}
	}

	if len(uncompressedChunks) == 0 {
		utils.Debugf("[COMPRESSION] topic %s already fully compressed or chunks too short", topicID)
		return nil
	}

	// Dynamic Threshold Calculation (Zero magic numbers):
	// 1. Session Budget Limit: TargetSessionWords * 1.33 (approx 4/3 tokens per word)
	// 2. Model Context Window Limit: 70% of active model's MaxInputTokens
	if mode != "ALWAYS" {
		targetSessionWords := userSettings.TargetSessionWords
		if targetSessionWords <= 0 {
			targetSessionWords = 3000
		}
		sessionBudgetThreshold := int(float64(targetSessionWords) * 1.33)

		dynamicThreshold := sessionBudgetThreshold

		llmSettings, err := s.repo.GetLLMSettings()
		if err == nil && llmSettings != nil {
			maxInput := llmSettings.Fast.MaxInputTokens
			if maxInput > 0 {
				modelSafeLimit := int(float64(maxInput) * 0.70)
				if modelSafeLimit < dynamicThreshold {
					dynamicThreshold = modelSafeLimit
				}
			}
		}

		if totalTokens <= dynamicThreshold {
			utils.Infof("[COMPRESSION] total tokens %d <= dynamic threshold %d, skipping compression under %s mode", totalTokens, dynamicThreshold, mode)
			return nil
		}
	}

	rate := userSettings.PromptCompressionRate
	if rate <= 0 || rate > 1.0 {
		rate = 0.80
	}

	utils.Infof("[COMPRESSION] running background compression on %d chunks for topic %s (rate=%.2f)", len(uncompressedChunks), topicID, rate)

	extDir := extension.ResolveExtensionsDir("")
	compressExt := &extension.Extension{
		Manifest: extension.Manifest{ID: "prompt_compressor", Name: "Prompt & Context Compressor", Runtime: "python", Entrypoint: "compress.py"},
		Dir:      filepath.Join(extDir, "prompt_compressor"),
	}

	pythonPath, err := extension.FindExtensionPython(compressExt)
	if err != nil {
		// If python is not installed / ready yet, fail gracefully
		return fmt.Errorf("python runtime not ready for prompt_compressor: %w", err)
	}

	scriptCandidates := []string{
		filepath.Join("extensions", "prompt_compressor", "compress.py"),
		filepath.Join("extensions", "compress.py"),
	}
	if extDir != "" {
		scriptCandidates = append(scriptCandidates,
			filepath.Join(extDir, "prompt_compressor", "compress.py"),
			filepath.Join(extDir, "compress.py"),
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
		scriptPath = filepath.Join("extensions", "prompt_compressor", "compress.py")
	}

	runner := extension.NewRunner()
	const batchSize = 10
	totalProcessed := 0

	for i := 0; i < len(uncompressedChunks); i += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		end := i + batchSize
		if end > len(uncompressedChunks) {
			end = len(uncompressedChunks)
		}
		batch := uncompressedChunks[i:end]

		var chunkInputs []CompressionChunkInput
		for _, c := range batch {
			chunkInputs = append(chunkInputs, CompressionChunkInput{
				ID:   c.ID,
				Text: c.Text,
			})
		}

		payload := CompressionPayload{
			Rate:   rate,
			Chunks: chunkInputs,
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			utils.Warnf("[COMPRESSION] failed to marshal batch payload for topic %s: %v", topicID, err)
			continue
		}

		outputBytes, err := runExtensionWithInput(ctx, runner, pythonPath, scriptPath, payloadBytes)
		if err != nil {
			utils.Warnf("[COMPRESSION] failed to execute batch %d-%d for topic %s: %v", i, end, topicID, err)
			continue
		}

		var output CompressionOutput
		if err := json.Unmarshal(outputBytes, &output); err != nil {
			utils.Warnf("[COMPRESSION] failed to parse batch output for topic %s: %v", topicID, err)
			continue
		}

		if output.Error != "" {
			utils.Warnf("[COMPRESSION] batch error for topic %s: %s", topicID, output.Error)
			continue
		}

		// Update SQLite chunks immediately for this batch
		for _, res := range output.Results {
			if res.Error == "" && res.CompressedText != "" {
				if err := s.repo.UpdateChunkCompressedText(res.ID, res.CompressedText, res.CompressedTokens); err != nil {
					utils.Warnf("[COMPRESSION] failed to update chunk %s compressed text: %v", res.ID, err)
				} else {
					totalProcessed++
				}
			}
		}
	}

	utils.Infof("[COMPRESSION] successfully compressed and persisted %d chunks for topic %s", totalProcessed, topicID)

	return nil
}

func runExtensionWithInput(ctx context.Context, runner *extension.Runner, pythonPath, scriptPath string, input []byte) ([]byte, error) {
	var outBuilder strings.Builder
	err := runner.RunStreamWithInput(ctx, "", pythonPath, input, func(line string) error {
		outBuilder.WriteString(line)
		return nil
	}, scriptPath)
	if err != nil {
		return nil, err
	}
	return []byte(outBuilder.String()), nil
}
