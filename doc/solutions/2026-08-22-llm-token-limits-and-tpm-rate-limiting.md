# LLM Token Limits, Groq TPM 413 Fix & Configurable Prompt Budgets

## Problem Statement

When completing a quiz or generating flashcards with Groq free-tier models (`openai/gpt-oss-120b`), generation failed with a 413 rate limit error:

```json
{"error":{"message":"Request too large for model `openai/gpt-oss-120b` ... on tokens per minute (TPM): Limit 8000, Requested 8138, please reduce your message size and try again.","type":"tokens","code":"rate_limit_exceeded"}}
```

### Root Causes
1. **Groq's 8K TPM Rolling Window:** Groq's free tier limits `gpt-oss-120b` to 8,000 Tokens Per Minute across a rolling 60-second window.
2. **High Default Limits:** The backend configured `MaxInputTokens = 7500` + `MaxOutputTokens = 1500` ($9,000 > 8,000$ TPM).
3. **Sequential Task Execution:** Quiz grading immediately followed by flashcard generation in the same minute saturated the rolling TPM quota.
4. **Tokenizer Drift:** Local WordPiece token counts differed by 10–20% from remote LLM Byte-Pair Encoding (BPE).

---

## Solution

### 1. Safe Conservative Defaults
- Default token limits in `internal/llm/provider.go` are set to `MaxInputTokens: 4000`, `MaxOutputTokens: 1000`.
- This ensures prompt, output, and sequential quiz-to-flashcard generations stay safely under 8,000 TPM without rate limit collisions.

### 2. User-Configurable Token Limits in Settings
- Added `max_input_tokens` and `max_output_tokens` columns to the `llm_settings` SQLite table.
- Added **Max Input Tokens** input field directly to the **AI Provider** panel in `Settings.vue` (`frontend/src/components/SettingsAIProvider.vue`).
- Supports both Fast AI tier and Heavy AI tier with a default of `4000`.

### 3. Open OpenAI-Compatible Architecture
- Users can connect any OpenAI-compatible provider (Gemini, OpenRouter, Groq, Ollama, LMStudio) by entering the Base URL and Model Name.
- High-capacity models (e.g. Gemini with 1M context or paid tiers) can have their `Max Input Tokens` raised in the Settings UI or via `FAST_LLM_MAX_INPUT_TOKENS` / `LLM_MAX_INPUT_TOKENS` environment variables.

---

## Files Modified

| File | Changes |
|------|---------|
| `internal/llm/provider.go` | Simplified `getModelLimits` default to 4000/1000; reads settings/env overrides. |
| `internal/models/models.go` | Added `MaxInputTokens` and `MaxOutputTokens` to `LLMTierSettings`. |
| `internal/db/schema.go` | Added `max_input_tokens` and `max_output_tokens` to `llm_settings` and `alterStatements`. |
| `internal/db/store.go` | Updated `GetLLMSettings` and `UpdateLLMSettings` to persist and load token limits. |
| `internal/app/app_settings.go` | Updated `UpdateLLMSettings` to enforce token defaults when saving. |
| `frontend/src/components/SettingsAIProvider.vue` | Added "Max Input Tokens" input field with placeholder `4000 (Default)`. |
| `frontend/src/composables/useLLM.js` | Added default `max_input_tokens: 4000` to reactive state. |
