package llm

import (
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
	limits := getModelLimits("any-model")
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

