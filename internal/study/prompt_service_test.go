package study

import (
	"strings"
	"testing"

	"ai-tutor/internal/models"
)

type mockPromptRepo struct {
	prefLang        string
	customPrompts   map[string]string
	assignedPrompts map[string]string
	variants        map[string][]models.CustomTaskPromptVariant
}

func newMockPromptRepo() *mockPromptRepo {
	return &mockPromptRepo{
		prefLang:        "english",
		customPrompts:   make(map[string]string),
		assignedPrompts: make(map[string]string),
		variants:        make(map[string][]models.CustomTaskPromptVariant),
	}
}

func (m *mockPromptRepo) GetPreferredLanguage() (string, error) {
	return m.prefLang, nil
}

func (m *mockPromptRepo) SetPreferredLanguage(lang string) error {
	m.prefLang = lang
	return nil
}

func (m *mockPromptRepo) GetActiveCustomPrompt(category string) (string, error) {
	return m.customPrompts[category], nil
}

func (m *mockPromptRepo) GetAssignedPromptTemplate(notebookID, category string) (string, error) {
	if m.assignedPrompts != nil {
		if tmpl, ok := m.assignedPrompts[notebookID+":"+category]; ok && tmpl != "" {
			return tmpl, nil
		}
	}
	return m.GetActiveCustomPrompt(category)
}

func (m *mockPromptRepo) GetPromptVariants(category string) ([]models.CustomTaskPromptVariant, error) {
	return m.variants[category], nil
}

func (m *mockPromptRepo) GetAllPromptVariants() ([]models.CustomTaskPromptVariant, error) {
	var all []models.CustomTaskPromptVariant
	for _, vars := range m.variants {
		all = append(all, vars...)
	}
	return all, nil
}

func (m *mockPromptRepo) SavePromptVariant(v models.CustomTaskPromptVariant) error {
	vars := m.variants[v.Category]
	found := false
	for i, existing := range vars {
		if existing.ID == v.ID {
			vars[i] = v
			found = true
			break
		}
	}
	if !found {
		vars = append(vars, v)
	}
	m.variants[v.Category] = vars
	if v.IsActive {
		m.customPrompts[v.Category] = v.Template
	}
	return nil
}

func (m *mockPromptRepo) SetActivePromptVariant(category, variantID string) error {
	vars := m.variants[category]
	for i := range vars {
		vars[i].IsActive = (vars[i].ID == variantID)
		if vars[i].IsActive {
			m.customPrompts[category] = vars[i].Template
		}
	}
	m.variants[category] = vars
	return nil
}

func (m *mockPromptRepo) DeletePromptVariant(variantID string) error {
	for cat, vars := range m.variants {
		var filtered []models.CustomTaskPromptVariant
		for _, v := range vars {
			if v.ID != variantID {
				filtered = append(filtered, v)
			}
		}
		m.variants[cat] = filtered
	}
	return nil
}

func (m *mockPromptRepo) GetCustomPrompt(category string) (string, error) {
	return m.customPrompts[category], nil
}

func (m *mockPromptRepo) SetCustomPrompt(category string, template string) error {
	m.customPrompts[category] = template
	return nil
}

func (m *mockPromptRepo) DeleteCustomPrompt(category string) error {
	delete(m.customPrompts, category)
	return nil
}

func (m *mockPromptRepo) GetAllCustomPrompts() (map[string]string, error) {
	cp := make(map[string]string)
	for k, v := range m.customPrompts {
		cp[k] = v
	}
	return cp, nil
}

func TestPromptService_Validation(t *testing.T) {
	repo := newMockPromptRepo()
	svc := NewPromptService(repo)

	// Valid notes prompt
	err := svc.SetCustomTemplate(PromptCategoryNotes, "Notes on {CONTENT} and {TOPIC}")
	if err != nil {
		t.Fatalf("expected valid notes template to succeed: %v", err)
	}

	// Invalid quiz prompt (missing {NUM_QUESTIONS})
	err = svc.SetCustomTemplate(PromptCategoryQuiz, "Quiz on {CONTENT}")
	if err == nil {
		t.Fatalf("expected error on missing mandatory placeholder {NUM_QUESTIONS}")
	}

	// Invalid flashcard prompt (missing {MAX_CARDS})
	err = svc.SetCustomTemplate(PromptCategoryFlashcard, "Cards for {CONTENT}")
	if err == nil {
		t.Fatalf("expected error on missing mandatory placeholder {MAX_CARDS}")
	}
}

func TestPromptService_BuildPrompt_QuizGuardrails(t *testing.T) {
	repo := newMockPromptRepo()
	svc := NewPromptService(repo)

	prompt, err := svc.BuildPrompt(PromptCategoryQuiz, map[string]string{
		"{CONTENT}":       "Operating systems manage CPU and RAM memory.",
		"{TOPIC}":         "Operating Systems",
		"{NUM_QUESTIONS}": "3",
		"{DIFFICULTY}":    "Hard",
	})
	if err != nil {
		t.Fatalf("failed to build quiz prompt: %v", err)
	}

	if !strings.Contains(prompt, "CRITICAL FORMAT RULES:") {
		t.Fatalf("quiz prompt must contain syntax guardrail footer")
	}
	if !strings.Contains(prompt, "Operating systems manage CPU and RAM memory.") {
		t.Fatalf("quiz prompt must contain interpolated content")
	}
}

func TestPromptService_BuildPrompt_FlashcardGuardrails(t *testing.T) {
	repo := newMockPromptRepo()
	svc := NewPromptService(repo)

	prompt, err := svc.BuildPrompt(PromptCategoryFlashcard, map[string]string{
		"{CONTENT}":   "Photosynthesis converts light into chemical energy.",
		"{TOPIC}":     "Biology",
		"{MAX_CARDS}": "5",
	})
	if err != nil {
		t.Fatalf("failed to build flashcard prompt: %v", err)
	}

	if !strings.Contains(prompt, "=== JSON FORMAT ===") {
		t.Fatalf("flashcard prompt must contain JSON schema enforcement wrapper")
	}
	if !strings.Contains(prompt, "Photosynthesis converts light into chemical energy.") {
		t.Fatalf("flashcard prompt must contain interpolated content")
	}
}

func TestPromptService_MultilingualInjection(t *testing.T) {
	repo := newMockPromptRepo()
	_ = repo.SetPreferredLanguage("kannada")
	svc := NewPromptService(repo)

	prompt, err := svc.BuildPrompt(PromptCategoryNotes, map[string]string{
		"{CONTENT}": "Newton's laws of motion form the basis of classical mechanics.",
		"{TOPIC}":   "Physics",
	})
	if err != nil {
		t.Fatalf("failed to build multilingual notes prompt: %v", err)
	}

	if !strings.Contains(prompt, "[LANGUAGE INSTRUCTION]") || !strings.Contains(prompt, "Kannada") {
		t.Fatalf("multilingual prompt must contain language instruction layer: %s", prompt)
	}
}

func TestPromptService_ResetToDefault(t *testing.T) {
	repo := newMockPromptRepo()
	svc := NewPromptService(repo)

	custom := "Custom Note on {CONTENT}"
	_ = svc.SetCustomTemplate(PromptCategoryNotes, custom)

	active, _ := svc.GetTemplate(PromptCategoryNotes)
	if active != custom {
		t.Fatalf("expected custom template to be active, got %s", active)
	}

	_ = svc.ResetToDefault(PromptCategoryNotes)
	defTmpl, _ := svc.GetDefaultTemplate(PromptCategoryNotes)
	activeAfter, _ := svc.GetTemplate(PromptCategoryNotes)
	if activeAfter != defTmpl {
		t.Fatalf("expected default template after reset, got %s", activeAfter)
	}
}

func TestPromptService_GetTemplate_NotebookOverride(t *testing.T) {
	repo := newMockPromptRepo()
	svc := NewPromptService(repo)

	// Setup assigned override for a notebook
	repo.assignedPrompts["book-123:" + PromptCategoryFlashcard] = "Custom Flashcards for book: {CONTENT} {MAX_CARDS}"
	repo.customPrompts[PromptCategoryFlashcard] = "Global Flashcards: {CONTENT} {MAX_CARDS}"

	tmpl, err := svc.GetTemplate(PromptCategoryFlashcard, "book-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(tmpl, "Custom Flashcards for book") {
		t.Errorf("expected book override, got: %s", tmpl)
	}

	// Different book should fallback to global active
	globalTmpl, err := svc.GetTemplate(PromptCategoryFlashcard, "other-book")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(globalTmpl, "Global Flashcards") {
		t.Errorf("expected global template, got: %s", globalTmpl)
	}
}
