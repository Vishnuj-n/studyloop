# Production LLM Output Budget, Dual Settings Tests & Truncation Handling

## Problem

1. **Unbounded Output Tokens**: Providers were not passed explicit completion token limits (`max_tokens`), resulting in unpredictable completion lengths and potential provider defaults clipping output mid-JSON.
2. **Silent Half-Written JSON**: Truncated responses (`finish_reason: "length"`) were passed directly to `json.Unmarshal`, producing generic unmarshalling errors instead of clear truncation feedback.
3. **Ambiguous LLM Testing**: A single "Test Connection" button only checked basic auth with `"Hi"`, providing no feedback on whether the provider endpoint accepted the app's configured token budgets (`MaxInputTokens` / `MaxOutputTokens`).
4. **Static LLM Routing**: Task routing in `selectLLM` only checked input token limits, ignoring task output budget requirements (e.g. 2500 tokens for comprehensive quiz generation).

## Solution Architecture

### 1. Free-Flow Provider Output (No Artificial Output Token Capping)
- Removed `max_tokens` payload capping in `internal/llm/provider.go` to allow output to flow freely until natural completion (`finish_reason == "stop"`).
- Retained input token limits (`MaxInputTokens`) to protect user TPM/RPM free tier quotas.
- `finish_reason == "length"` remains handled in case provider/model sequence limits are reached natively.

### 2. Dual Testing & Input Validation Controls
- **`Test Connection`**: Quick ping checking auth & reachability with a basic `"Hi"` prompt.
- **`Test Limits`**: Validates whether provider API endpoints cleanly accept StudyLoop's configured `MaxInputTokens`.
- UI fields focus purely on **Max Input Tokens** prompt protection.

### 3. Capability-Aware LLM Routing
- `selectLLM(contextText string)` routes requests based on input token limits (`MaxInputTokens`) and rate-limit cooldown tracking.
- The LLM writes all requested quiz questions and flashcards freely without artificial output truncation or downscaling.

---

## File Changes Map

| Module / Path | Description |
|---|---|
| `internal/llm/provider.go` | Added `max_tokens` payload budgeting, `finish_reason` handling, and updated 2500 default output limit |
| `internal/llm/provider_test.go` | Added unit tests for `max_tokens` payload serialization and `finish_reason=length` error handling |
| `internal/app/app_settings.go` | Added `TestLLMLimits` handler and updated settings `MaxOutputTokens` default to 2500 |
| `internal/app/app_settings_test.go` | Added unit tests for `TestLLMConnection` and `TestLLMLimits` |
| `internal/study/service.go` | Updated `selectLLM` to accept `requiredOutputBudget` and evaluate output capacity |
| `internal/study/quiz_sync.go` | Set `quizTaskOutputBudget = 2500`, passed budget to `selectLLM`, and added `targetCount` fallback |
| `internal/study/*` | Updated callers of `selectLLM` across `examiner.go`, `audio_overview.go`, `flashcard.go`, `reader_ai.go`, `simplifier.go` |
| `frontend/src/services/appApi.js` | Exported `testLLMLimits` Wails bridge function |
| `frontend/src/composables/useLLM.js` | Updated `max_output_tokens` default to 2500 tokens in reactive settings state |
| `frontend/src/components/SettingsAIProvider.vue` | Added **Max Output Tokens** input fields for both Fast and Heavy provider settings panels and **[ Test Limits ]** buttons |

---

## Verification

- **Automated Tests**:
  - `go test -short ./internal/...` passed across all packages (`app`, `db`, `embeddings`, `extension`, `llm`, `notebook`, `retrieval`, `runtime`, `scheduler`, `study`, `utils`).
  - `go test -run=^$ ./internal/...` passed for instant compilation and type-checking verification.
