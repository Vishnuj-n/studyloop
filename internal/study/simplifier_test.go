package study

import (
	"strings"
	"testing"
)

func TestSimplifierLevelDirectives(t *testing.T) {
	tests := []struct {
		level    string
		expected string
	}{
		{"very_simple", "everyday language"},
		{"eli10", "everyday language"},
		{"summary", "essential ideas"},
		{"bullet", "essential ideas"},
		{"academic", "precise academic"},
		{"simple", "straightforward"},
		{"eli15", "straightforward"},
		{"unknown", "straightforward"},
		{"", "straightforward"},
	}

	for _, tt := range tests {
		directive := getSimplifierLevelDirective(tt.level)
		if !strings.Contains(strings.ToLower(directive), strings.ToLower(tt.expected)) {
			t.Errorf("expected directive for level %q to contain %q, got %q", tt.level, tt.expected, directive)
		}
	}
}

func TestBuildSimplifierPromptWithStyle(t *testing.T) {
	template := "Intro\n\nStyle: {{style}}\n\nReading:\n\"\"\"\n{{content}}\n\"\"\""
	content := "Gradient descent is an optimization algorithm."
	directive := "Explain using simple everyday analogies."

	prompt := buildSimplifierPrompt(template, content, directive)

	if !strings.Contains(prompt, "Style: Explain using simple everyday analogies.") {
		t.Errorf("expected prompt to contain substituted style, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Gradient descent is an optimization algorithm.") {
		t.Errorf("expected prompt to contain reading content, got: %s", prompt)
	}
}
