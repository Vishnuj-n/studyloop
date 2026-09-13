package study

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCleanOverviewText(t *testing.T) {
	// 0xED 0xB2 0x9D is UTF-8 encoding of surrogate \udc9d
	surrogateBytes := []byte{0xed, 0xb2, 0x9d}
	input := "# Heading 1\nThis is **bold** text with a [link](http://example.com) and `code` " + string(surrogateBytes) + "."
	cleaned := CleanOverviewText(input)

	if strings.Contains(cleaned, "#") || strings.Contains(cleaned, "**") || strings.Contains(cleaned, "`") {
		t.Fatalf("markdown symbols not cleaned: %s", cleaned)
	}

	if strings.Contains(cleaned, string(surrogateBytes)) {
		t.Fatalf("surrogate bytes were not stripped from output: %s", cleaned)
	}

	if !strings.Contains(cleaned, "Heading 1") || !strings.Contains(cleaned, "This is bold text") {
		t.Fatalf("unexpected cleaned output: %s", cleaned)
	}
}

func TestSplitIntoSentences(t *testing.T) {
	input := "Welcome to the study overview! In this chapter, we explore how neural networks learn from data. They adjust their internal weights through backpropagation. Finally, gradient descent minimizes the error rate."
	sentences := SplitIntoSentences(input)

	if len(sentences) == 0 {
		t.Fatalf("expected sentences, got empty slice")
	}

	if len(sentences) < 2 {
		t.Fatalf("expected at least 2 sentence chunks, got %d: %v", len(sentences), sentences)
	}
}

func TestBuildAudioOverviewPrompt(t *testing.T) {
	prompt := BuildAudioOverviewPrompt("Machine Learning Basics", "Neural networks are models composed of layers.")
	if !strings.Contains(prompt, "Machine Learning Basics") {
		t.Fatalf("prompt missing topic title")
	}
	if !strings.Contains(prompt, "Neural networks are models") {
		t.Fatalf("prompt missing material context")
	}
}

func TestAudioChunkUnmarshal(t *testing.T) {
	malformedLine := "not-json-line-error"
	var chunk AudioChunk
	err := json.Unmarshal([]byte(malformedLine), &chunk)
	if err == nil {
		t.Fatalf("expected unmarshal error for malformed json stdout, got nil")
	}

	validJSON := `{"generation_id":"gen-123","status":"success","chunk_id":1,"text":"Hello world","audio_base64":"abc123"}`
	var validChunk AudioChunk
	err = json.Unmarshal([]byte(validJSON), &validChunk)
	if err != nil {
		t.Fatalf("expected valid unmarshal, got error: %v", err)
	}
	if validChunk.GenerationID != "gen-123" || validChunk.ChunkID != 1 {
		t.Fatalf("unexpected unmarshaled chunk values: %+v", validChunk)
	}
}
