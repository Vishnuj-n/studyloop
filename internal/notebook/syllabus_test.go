package notebook

import (
	"strings"
	"testing"

	"ai-tutor/internal/llm"
)

type mockLLMProvider struct {
	limits        llm.ModelLimits
	capturedPrompt string
}

func (m *mockLLMProvider) GenerateAnswer(prompt string) (string, error) {
	m.capturedPrompt = prompt
	return `{"chapters":[{"title":"Chapter 1","start_page":1,"end_page":2}]}`, nil
}

func (m *mockLLMProvider) GetLimits() llm.ModelLimits {
	return m.limits
}

func TestDraftSyllabusChapters_YouTubePromptRetainsRulesUnderLowMaxTokens(t *testing.T) {
	svc := NewService(t.TempDir())
	
	// Create a document with several sections that generate a large sample text
	doc := &ExtractedDocument{
		PageCount: 5,
		Sections: []ExtractedSection{
			{PageNum: 1, Heading: "Intro", Text: strings.Repeat("Introductory topic content discussing fundamentals. ", 50)},
			{PageNum: 2, Heading: "Deep Dive", Text: strings.Repeat("Deep dive content into specific mechanics and math. ", 50)},
			{PageNum: 3, Heading: "Advanced", Text: strings.Repeat("Advanced topics and examples of architecture. ", 50)},
		},
	}

	// Constrained LLM limits
	mockLLM := &mockLLMProvider{
		limits: llm.ModelLimits{
			MaxInputTokens: 500,
		},
	}

	res, err := svc.DraftSyllabusChapters("youtube", "machine_learning.mp4", doc, mockLLM)
	if err != nil {
		t.Fatalf("unexpected error drafting syllabus: %v", err)
	}
	if res == nil || len(res.Chapters) != 1 {
		t.Fatalf("expected 1 parsed chapter, got %+v", res)
	}

	requiredRules := []string{
		`Output strict JSON only: {"chapters":[{"title":"...","start_page":1,"end_page":4}]}`,
		`"start_page" and "end_page" are 1-based segment indices`,
		`Aim for 4–8 chapters total`,
		`Any segment shorter than 3 minutes MUST be merged`,
		`Group related micro-segments into substantive study chapters`,
		`Omit or merge trivial segments`,
		`Ensure sequential, contiguous segment ranges without gaps or overlaps.`,
	}

	for _, rule := range requiredRules {
		if !strings.Contains(mockLLM.capturedPrompt, rule) {
			t.Errorf("expected prompt to retain rule %q, but was truncated or missing.\nFull prompt:\n%s", rule, mockLLM.capturedPrompt)
		}
	}
}
