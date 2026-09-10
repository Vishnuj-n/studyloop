package llm

import (
	"strings"
	"testing"

	"ai-tutor/internal/models"
)

func TestLoadConfigFromEnvForPrefixUsesPrefixedValues(t *testing.T) {
	t.Setenv("FAST_LLM_BASE_URL", "https://fast.example.com")
	t.Setenv("FAST_LLM_API_KEY", "fast-key")
	t.Setenv("FAST_LLM_MODEL", "fast-model")
	t.Setenv("FAST_LLM_TIMEOUT_MS", "1234")

	config := LoadConfigFromEnvForPrefix("FAST_LLM")

	if config.BaseURL != "https://fast.example.com" {
		t.Fatalf("unexpected BaseURL: %s", config.BaseURL)
	}
	if config.APIKey != "fast-key" {
		t.Fatalf("unexpected APIKey: %s", config.APIKey)
	}
	if config.Model != "fast-model" {
		t.Fatalf("unexpected Model: %s", config.Model)
	}
	if config.TimeoutMs != 1234 {
		t.Fatalf("unexpected TimeoutMs: %d", config.TimeoutMs)
	}
}

func TestLoadConfigFromEnvForPrefixFallsBackToLegacyVars(t *testing.T) {
	t.Setenv("LLM_BASE_URL", "https://legacy.example.com")
	t.Setenv("LLM_API_KEY", "legacy-key")
	t.Setenv("LLM_MODEL", "legacy-model")
	t.Setenv("LLM_TIMEOUT_MS", "2345")

	config := LoadConfigFromEnvForPrefix("HEAVY_LLM")

	if config.BaseURL != "https://legacy.example.com" {
		t.Fatalf("unexpected BaseURL: %s", config.BaseURL)
	}
	if config.APIKey != "legacy-key" {
		t.Fatalf("unexpected APIKey: %s", config.APIKey)
	}
	if config.Model != "legacy-model" {
		t.Fatalf("unexpected Model: %s", config.Model)
	}
	if config.TimeoutMs != 2345 {
		t.Fatalf("unexpected TimeoutMs: %d", config.TimeoutMs)
	}
}

func TestGetModelLimitsDefault(t *testing.T) {
	limits := getModelLimits()
	if limits.MaxInputTokens != 4000 {
		t.Errorf("expected default MaxInputTokens to be 4000, got %d", limits.MaxInputTokens)
	}
	if limits.MaxOutputTokens != 1000 {
		t.Errorf("expected default MaxOutputTokens to be 1000, got %d", limits.MaxOutputTokens)
	}
}

func TestLoadConfigLimitsOverride(t *testing.T) {
	t.Setenv("FAST_LLM_MODEL", "openai/gpt-oss-120b")
	t.Setenv("FAST_LLM_MAX_INPUT_TOKENS", "12345")
	t.Setenv("FAST_LLM_MAX_OUTPUT_TOKENS", "54321")

	config := LoadConfigFromEnvForPrefix("FAST_LLM")
	if config.Limits.MaxInputTokens != 12345 {
		t.Fatalf("expected MaxInputTokens override 12345, got %d", config.Limits.MaxInputTokens)
	}
	if config.Limits.MaxOutputTokens != 54321 {
		t.Fatalf("expected MaxOutputTokens override 54321, got %d", config.Limits.MaxOutputTokens)
	}
}

func TestLoadConfigFromSettingsLimitsOverride(t *testing.T) {
	settings := models.LLMTierSettings{
		Tier:            "fast",
		Provider:        "groq",
		Model:           "openai/gpt-oss-120b",
		MaxInputTokens:  5000,
		MaxOutputTokens: 2000,
	}
	config := LoadConfigFromSettingsForPrefix("", settings, "test-key")
	if config.Limits.MaxInputTokens != 5000 {
		t.Fatalf("expected settings MaxInputTokens 5000, got %d", config.Limits.MaxInputTokens)
	}
	if config.Limits.MaxOutputTokens != 2000 {
		t.Fatalf("expected settings MaxOutputTokens 2000, got %d", config.Limits.MaxOutputTokens)
	}
}

func TestDefaultBaseURLAndModelForGemini(t *testing.T) {
	if got := defaultBaseURLForProvider("gemini"); got != "https://generativelanguage.googleapis.com/v1beta/openai" {
		t.Fatalf("unexpected Gemini baseURL: %s", got)
	}
	if got := defaultModelForProvider("gemini"); got != "gemini-flash-lite-latest" {
		t.Fatalf("unexpected Gemini model: %s", got)
	}
}

func TestCanonicalBaseURLsAndURLConstruction(t *testing.T) {
	tests := []struct {
		provider    string
		expectedBase string
		expectedURL  string
	}{
		{
			provider:     "gemini",
			expectedBase: "https://generativelanguage.googleapis.com/v1beta/openai",
			expectedURL:  "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		},
		{
			provider:     "groq",
			expectedBase: "https://api.groq.com/openai/v1",
			expectedURL:  "https://api.groq.com/openai/v1/chat/completions",
		},
		{
			provider:     "openai",
			expectedBase: "https://api.openai.com/v1",
			expectedURL:  "https://api.openai.com/v1/chat/completions",
		},
		{
			provider:     "openrouter",
			expectedBase: "https://openrouter.ai/api/v1",
			expectedURL:  "https://openrouter.ai/api/v1/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			base := defaultBaseURLForProvider(tt.provider)
			if base != tt.expectedBase {
				t.Fatalf("expected base %s, got %s", tt.expectedBase, base)
			}
			baseURL := strings.TrimSuffix(base, "/")
			var url string
			if strings.HasSuffix(baseURL, "/chat/completions") {
				url = baseURL
			} else {
				url = baseURL + "/chat/completions"
			}
			if url != tt.expectedURL {
				t.Fatalf("expected endpoint %s, got %s", tt.expectedURL, url)
			}
		})
	}
}

func TestProviderGenerateAnswerNilReceiver(t *testing.T) {
	var p *Provider
	_, err := p.GenerateAnswer("hello")
	if err == nil || err.Error() != "LLM config not configured" {
		t.Fatalf("expected 'LLM config not configured', got %v", err)
	}

	p2 := &Provider{}
	_, err2 := p2.GenerateAnswer("hello")
	if err2 == nil || err2.Error() != "LLM config not configured" {
		t.Fatalf("expected 'LLM config not configured', got %v", err2)
	}
}

func TestApplyEnvLimitsOverrideHandling(t *testing.T) {
	t.Setenv("LLM_MAX_INPUT_TOKENS", "notanumber")
	t.Setenv("LLM_MAX_OUTPUT_TOKENS", "0")

	limits := getModelLimits()
	applyEnvLimitsOverride("", &limits)
	if limits.MaxInputTokens != 4000 {
		t.Fatalf("expected MaxInputTokens to remain default 4000, got %d", limits.MaxInputTokens)
	}
	if limits.MaxOutputTokens != 1000 {
		t.Fatalf("expected MaxOutputTokens to remain default 1000, got %d", limits.MaxOutputTokens)
	}

	t.Setenv("LLM_MAX_INPUT_TOKENS", "8000")
	t.Setenv("LLM_MAX_OUTPUT_TOKENS", "2000")
	applyEnvLimitsOverride("", &limits)
	if limits.MaxInputTokens != 8000 {
		t.Fatalf("expected MaxInputTokens 8000, got %d", limits.MaxInputTokens)
	}
	if limits.MaxOutputTokens != 2000 {
		t.Fatalf("expected MaxOutputTokens 2000, got %d", limits.MaxOutputTokens)
	}
}



