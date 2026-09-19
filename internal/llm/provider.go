package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"
)

var promptLoggingEnabled atomic.Bool

// SetPromptLoggingEnabled sets whether LLM prompt logging to file is active.
func SetPromptLoggingEnabled(enabled bool) {
	promptLoggingEnabled.Store(enabled)
}

// IsPromptLoggingEnabled returns true if prompt logging is enabled at runtime or via APP_ENV=dev.
func IsPromptLoggingEnabled() bool {
	return promptLoggingEnabled.Load() || os.Getenv("APP_ENV") == "dev"
}

// RateLimitError indicates an HTTP 429 rate limit / TPM exceeded error from the provider.
type RateLimitError struct {
	StatusCode int
	Message    string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("status 429: rate limit reached (%s)", e.Message)
}

// IsRateLimitError returns true if the error represents an HTTP 429 Rate Limit error.
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	var rErr *RateLimitError
	if errors.As(err, &rErr) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 429") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "tpm limit")
}

// ModelLimits defines token limits for specific models.
type ModelLimits struct {
	MaxInputTokens  int
	MaxOutputTokens int
}

// Config holds LLM provider configuration.
type Config struct {
	BaseURL   string
	APIKey    string
	Model     string
	TimeoutMs int
	Limits    ModelLimits
}

// LoadConfigFromEnvForPrefix loads provider config for a named tier.
// Prefix examples: FAST_LLM. Pass empty string for the default single-provider config.
func LoadConfigFromEnvForPrefix(prefix string) *Config {
	prefix = strings.TrimSpace(prefix)
	if prefix != "" {
		prefix = strings.TrimSuffix(prefix, "_")
	}

	baseURLKeys := prefixedKeys(prefix, "LLM_BASE_URL", "OPENAI_ENDPOINT", "OPENAI_BASE_URL", "BASE_URL")
	apiKeyKeys := prefixedKeys(prefix, "LLM_API_KEY", "OPENAI_API_KEY", "API_KEY")
	modelKeys := prefixedKeys(prefix, "LLM_MODEL", "BASE_MODEL", "OPENAI_MODEL", "MODEL")
	timeoutKeys := prefixedKeys(prefix, "LLM_TIMEOUT_MS", "OPENAI_TIMEOUT_MS", "TIMEOUT_MS")

	config := &Config{
		BaseURL:   firstEnv(baseURLKeys...),
		APIKey:    firstEnv(apiKeyKeys...),
		Model:     firstEnv(modelKeys...),
		TimeoutMs: firstEnvInt(30000, timeoutKeys...),
	}

	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:8000"
	}
	if config.Model == "" {
		config.Model = "openai/gpt-oss-120b"
	}
	if config.APIKey == "" {
		config.APIKey = "sk-test"
	}

	config.Limits = getModelLimits()
	applyEnvLimitsOverride(prefix, &config.Limits)

	return config
}

// LoadConfigFromSettingsForPrefix resolves config as:
// env override -> SQLite non-secret settings + keyring secret -> provider defaults.
func LoadConfigFromSettingsForPrefix(prefix string, settings models.LLMTierSettings, apiKey string) *Config {
	prefix = strings.TrimSpace(prefix)
	if prefix != "" {
		prefix = strings.TrimSuffix(prefix, "_")
	}

	baseURLKeys := prefixedKeys(prefix, "LLM_BASE_URL", "OPENAI_ENDPOINT", "OPENAI_BASE_URL", "BASE_URL")
	apiKeyKeys := prefixedKeys(prefix, "LLM_API_KEY", "OPENAI_API_KEY", "API_KEY")
	modelKeys := prefixedKeys(prefix, "LLM_MODEL", "BASE_MODEL", "OPENAI_MODEL", "MODEL")
	timeoutKeys := prefixedKeys(prefix, "LLM_TIMEOUT_MS", "OPENAI_TIMEOUT_MS", "TIMEOUT_MS")

	config := &Config{
		BaseURL:   firstNonEmpty(settings.BaseURL, firstEnv(baseURLKeys...), defaultBaseURLForProvider(settings.Provider)),
		APIKey:    firstNonEmpty(apiKey, firstEnv(apiKeyKeys...)),
		Model:     firstNonEmpty(settings.Model, firstEnv(modelKeys...), defaultModelForProvider(settings.Provider)),
		TimeoutMs: firstEnvInt(settings.TimeoutMs, timeoutKeys...),
	}
	if config.TimeoutMs <= 0 {
		config.TimeoutMs = 30000
	}
	config.Limits = getModelLimits()
	if settings.MaxInputTokens > 0 {
		config.Limits.MaxInputTokens = settings.MaxInputTokens
	}
	if settings.MaxOutputTokens > 0 {
		config.Limits.MaxOutputTokens = settings.MaxOutputTokens
	}
	applyEnvLimitsOverride(prefix, &config.Limits)
	return config
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func defaultBaseURLForProvider(provider string) string {
	switch strings.TrimSpace(strings.ToLower(provider)) {
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta/openai"
	case "groq":
		return "https://api.groq.com/openai/v1"
	case "openai":
		return "https://api.openai.com/v1"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	default:
		return ""
	}
}

func defaultModelForProvider(provider string) string {
	switch strings.TrimSpace(strings.ToLower(provider)) {
	case "gemini":
		return "gemini-flash-lite-latest"
	case "groq":
		return "openai/gpt-oss-120b"
	case "openai":
		return "gpt-4.1-mini"
	case "openrouter":
		return "openai/gpt-4.1-mini"
	default:
		return ""
	}
}

// getModelLimits returns token limits with safe conservative defaults.
func getModelLimits() ModelLimits {
	return ModelLimits{
		MaxInputTokens:  4000,
		MaxOutputTokens: 2500,
	}
}

func applyEnvLimitsOverride(prefix string, limits *ModelLimits) {
	maxInputKeys := prefixedKeys(prefix, "LLM_MAX_INPUT_TOKENS", "MAX_INPUT_TOKENS")
	maxOutputKeys := prefixedKeys(prefix, "LLM_MAX_OUTPUT_TOKENS", "MAX_OUTPUT_TOKENS")

	for _, key := range maxInputKeys {
		raw := strings.TrimSpace(os.Getenv(key))
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			utils.Warnf("[LLM_CONFIG] ignoring invalid %s=%q", key, raw)
			continue
		}
		limits.MaxInputTokens = value
		break
	}

	for _, key := range maxOutputKeys {
		raw := strings.TrimSpace(os.Getenv(key))
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			utils.Warnf("[LLM_CONFIG] ignoring invalid %s=%q", key, raw)
			continue
		}
		limits.MaxOutputTokens = value
		break
	}
}

func prefixedKeys(prefix string, keys ...string) []string {
	if prefix == "" {
		return keys
	}

	result := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		result = append(result, prefix+"_"+key)
	}
	result = append(result, keys...)
	return result
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
}

func firstEnvInt(defaultValue int, keys ...string) int {
	for _, key := range keys {
		raw := strings.TrimSpace(os.Getenv(key))
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err == nil {
			return value
		}
	}
	return defaultValue
}

// Provider handles communication with OpenAI-compatible APIs.
type Provider struct {
	config *Config
}

// NewProvider creates a new LLM provider.
func NewProvider(config *Config) *Provider {
	return &Provider{config: config}
}

// ModelName returns the configured model identifier sent to the provider API.
func (p *Provider) ModelName() string {
	if p == nil || p.config == nil {
		return ""
	}
	return strings.TrimSpace(p.config.Model)
}

// GetLimits returns the token limits for the configured model.
func (p *Provider) GetLimits() ModelLimits {
	if p == nil || p.config == nil {
		return ModelLimits{
			MaxInputTokens:  30000,
			MaxOutputTokens: 3000,
		}
	}
	return p.config.Limits
}

// openAIRequest follows the OpenAI API format.
type openAIRequest struct {
	Model     string          `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

// openAIMessage represents a message in the OpenAI API.
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse follows the OpenAI API response format.
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// GenerateAnswer calls the LLM to generate an answer.
func (p *Provider) GenerateAnswer(prompt string) (string, error) {
	if p == nil || p.config == nil || p.config.BaseURL == "" {
		return "", fmt.Errorf("LLM config not configured")
	}
	apiKey := strings.TrimSpace(p.config.APIKey)
	if apiKey == "" {
		return "", fmt.Errorf("LLM API key not configured")
	}

	baseURLCheck := strings.ToLower(p.config.BaseURL)
	if strings.Contains(baseURLCheck, "googleapis.com") {
		if strings.HasPrefix(apiKey, "gsk_") {
			return "", fmt.Errorf("invalid API key for Gemini provider: key starts with 'gsk_' (Groq key format). Please check your AI provider settings in Settings.")
		}
	} else if strings.Contains(baseURLCheck, "groq.com") {
		if strings.HasPrefix(apiKey, "AIza") {
			return "", fmt.Errorf("invalid API key for Groq provider: key starts with 'AIza' (Gemini key format). Please check your AI provider settings in Settings.")
		}
	}

	words := len(strings.Fields(prompt))
	estPromptTokens := int(float64(words) * 1.3)
	if estPromptTokens == 0 && len(prompt) > 0 {
		estPromptTokens = (len(prompt) + 3) / 4
	}

	limits := p.GetLimits()
	utils.Warnf("[LLM_REQUEST] model=%s base_url=%s max_input_tokens=%d est_prompt_tokens=%d prompt_chars=%d prompt_words=%d",
		p.config.Model, p.config.BaseURL, limits.MaxInputTokens, estPromptTokens, len(prompt), words)

	requestBody := openAIRequest{
		Model: p.config.Model,
		Messages: []openAIMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	if IsPromptLoggingEnabled() {
		logDir := resolveLogsDir()
		_ = os.MkdirAll(logDir, 0755)
		debugLog := fmt.Sprintf("\n--- PROMPT @ %s [model: %s | max_input: %d | est_tokens: %d | chars: %d] ---\n%s\n--- END PROMPT ---\n",
			time.Now().Format("2006-01-02 15:04:05"), p.config.Model, limits.MaxInputTokens, estPromptTokens, len(prompt), prompt)
		if f, err := os.OpenFile(filepath.Join(logDir, "llm_prompt.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			_, _ = f.WriteString(debugLog)
			_ = f.Close()
		}
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	if strings.EqualFold(baseURL, "https://api.groq.com/openai") {
		baseURL = "https://api.groq.com/openai/v1"
	} else if strings.EqualFold(baseURL, "https://openrouter.ai/api") {
		baseURL = "https://openrouter.ai/api/v1"
	} else if strings.EqualFold(baseURL, "https://api.openai.com") {
		baseURL = "https://api.openai.com/v1"
	}
	var url string
	if strings.HasSuffix(baseURL, "/chat/completions") {
		url = baseURL
	} else {
		url = baseURL + "/chat/completions"
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	effectiveTimeoutMs := p.config.TimeoutMs
	if effectiveTimeoutMs <= 0 {
		effectiveTimeoutMs = 30000
	}
	client := &http.Client{
		Timeout: time.Duration(effectiveTimeoutMs) * time.Millisecond,
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		utils.Warnf("[LLM_ERROR] model=%s duration_ms=%d err=%v", p.config.Model, time.Since(startTime).Milliseconds(), err)
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respDuration := time.Since(startTime)
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		utils.Warnf("[LLM_ERROR] model=%s duration_ms=%d status=%d err_body=%s", p.config.Model, respDuration.Milliseconds(), resp.StatusCode, string(bodyBytes))
		if resp.StatusCode == http.StatusTooManyRequests {
			return "", &RateLimitError{StatusCode: resp.StatusCode, Message: string(bodyBytes)}
		}
		return "", fmt.Errorf("LLM API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", err
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	if apiResp.Choices[0].FinishReason == "length" {
		return "", fmt.Errorf("LLM output truncated: max output tokens reached (finish_reason=length)")
	}

	if apiResp.Usage.TotalTokens > 0 {
		utils.Warnf("[LLM_RESPONSE] model=%s duration_ms=%d prompt_tokens=%d completion_tokens=%d total_tokens=%d configured_max_input=%d",
			p.config.Model, respDuration.Milliseconds(), apiResp.Usage.PromptTokens, apiResp.Usage.CompletionTokens, apiResp.Usage.TotalTokens, limits.MaxInputTokens)
	} else {
		outWords := len(strings.Fields(apiResp.Choices[0].Message.Content))
		estCompletionTokens := int(float64(outWords) * 1.3)
		utils.Warnf("[LLM_RESPONSE] model=%s duration_ms=%d est_completion_tokens=%d resp_chars=%d configured_max_input=%d",
			p.config.Model, respDuration.Milliseconds(), estCompletionTokens, len(apiResp.Choices[0].Message.Content), limits.MaxInputTokens)
	}

	return apiResp.Choices[0].Message.Content, nil
}

func resolveLogsDir() string {
	if os.Getenv("APP_ENV") == "dev" {
		return filepath.Join("dev_data", "logs")
	}
	if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
		return filepath.Join(cfgDir, "Studyloop", "logs")
	}
	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		return filepath.Join(homeDir, ".Studyloop", "logs")
	}
	return filepath.Join("dev_data", "logs")
}

