package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	if limits.MaxOutputTokens != 2500 {
		t.Errorf("expected default MaxOutputTokens to be 2500, got %d", limits.MaxOutputTokens)
	}
}

func TestGenerateAnswerMaxTokensAndTruncation(t *testing.T) {
	var capturedPayload openAIRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedPayload)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.Header.Get("X-Test-Case"), "length") {
			_, _ = w.Write([]byte(`{
				"choices": [
					{
						"message": {"content": "Truncated content"},
						"finish_reason": "length"
					}
				]
			}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {"content": "OK response"},
					"finish_reason": "stop"
				}
			]
		}`))
	}))
	defer server.Close()

	cfg := &Config{
		BaseURL: server.URL,
		APIKey:  "sk-test",
		Model:   "test-model",
		Limits:  ModelLimits{MaxInputTokens: 4000, MaxOutputTokens: 2500},
	}
	provider := NewProvider(cfg)

	resp, err := provider.GenerateAnswer("Test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "OK response" {
		t.Fatalf("expected 'OK response', got %q", resp)
	}
	if capturedPayload.MaxTokens != 0 {
		t.Fatalf("expected max_tokens in payload to be 0 (omitted for free-flow output), got %d", capturedPayload.MaxTokens)
	}

	// Test truncation error when finish_reason == "length"
	req, _ := http.NewRequest("GET", server.URL, nil)
	_ = req
	serverLength := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {"content": "Truncated content"},
					"finish_reason": "length"
				}
			]
		}`))
	}))
	defer serverLength.Close()

	cfgLength := &Config{
		BaseURL: serverLength.URL,
		APIKey:  "sk-test",
		Model:   "test-model",
		Limits:  ModelLimits{MaxInputTokens: 4000, MaxOutputTokens: 2500},
	}
	providerLength := NewProvider(cfgLength)

	_, errLength := providerLength.GenerateAnswer("Test prompt")
	if errLength == nil {
		t.Fatalf("expected truncation error for finish_reason=length, got nil")
	}
	if !strings.Contains(errLength.Error(), "finish_reason=length") {
		t.Fatalf("expected error mentioning finish_reason=length, got: %v", errLength)
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

func TestKeyFormatValidation(t *testing.T) {
	geminiCfg := &Config{
		BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
		APIKey:  "gsk_invalid_groq_key_sent_to_gemini",
		Model:   "gemini-flash-lite-latest",
	}
	pGemini := NewProvider(geminiCfg)
	_, errGemini := pGemini.GenerateAnswer("hi")
	if errGemini == nil || !strings.Contains(errGemini.Error(), "invalid API key for Gemini provider") {
		t.Fatalf("expected Gemini key validation error, got: %v", errGemini)
	}

	groqCfg := &Config{
		BaseURL: "https://api.groq.com/openai/v1",
		APIKey:  "AIza_invalid_gemini_key_sent_to_groq",
		Model:   "openai/gpt-oss-120b",
	}
	pGroq := NewProvider(groqCfg)
	_, errGroq := pGroq.GenerateAnswer("hi")
	if errGroq == nil || !strings.Contains(errGroq.Error(), "invalid API key for Groq provider") {
		t.Fatalf("expected Groq key validation error, got: %v", errGroq)
	}
}
