package study

import (
	"strings"
	"testing"
)

func TestCleanOverviewText(t *testing.T) {
	input := "# Heading 1\nThis is **bold** text with a [link](http://example.com) and `code`."
	cleaned := CleanOverviewText(input)

	if strings.Contains(cleaned, "#") || strings.Contains(cleaned, "**") || strings.Contains(cleaned, "`") {
		t.Fatalf("markdown symbols not cleaned: %s", cleaned)
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
