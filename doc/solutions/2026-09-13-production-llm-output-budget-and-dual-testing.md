# Production LLM Output Budget, Dual Settings Tests & Truncation Handling

## Problem

1. **Unbounded Output Tokens**: Providers were not passed explicit completion token limits (`max_tokens`), resulting in unpredictable completion lengths and potential provider defaults clipping output mid-JSON.
2. **Silent Half-Written JSON**: Truncated responses (`finish_reason: "length"`) were passed directly to `json.Unmarshal`, producing generic unmarshalling errors instead of clear truncation feedback.
3. **Ambiguous LLM Testing**: A single "Test Connection" button only checked basic auth with `"Hi"`, providing no feedback on whether the provider endpoint accepted the app's configured token budgets (`MaxInputTokens` / `MaxOutputTokens`).
4. **Static LLM Routing**: Task routing in `selectLLM` only checked input token limits, ignoring task output budget requirements (e.g. 2500 tokens for comprehensive quiz generation).

## Solution Architecture

### 1. Provider Output Budgeting & Truncation Guards
- Added `MaxTokens int` (`json:"max_tokens,omitempty"`) to `openAIRequest` in `internal/llm/provider.go`.
- Added `FinishReason string` (`json:"finish_reason,omitempty"`) to `openAIResponse` choice structure.
- Updated safe default `MaxOutputTokens` fallback in `getModelLimits()` and `app_settings.go` from 1000 to 2500 tokens.
- Populated `requestBody.MaxTokens = limits.MaxOutputTokens` in every API call.
- Added immediate error return when `finish_reason == "length"`:
  `fmt.Errorf("LLM output truncated: max output tokens reached (finish_reason=length)")`.

### 2. Dual Testing Controls
- **`Test Connection`**: Quick ping checking auth & reachability with a basic `"Hi"` prompt.
- **`Test Limits`**: Added `TestLLMLimits` in `internal/app/app_settings.go` and `appApi.js` to validate whether provider API endpoints cleanly accept StudyLoop's configured `MaxInputTokens` and `MaxOutputTokens`.
- Added **[ Test Limits ]** UI buttons alongside **[ Test Connection ]** in `SettingsAIProvider.vue` for Fast and Heavy tiers.

### 3. Capability-Aware LLM Routing & Quiz Target Fallbacks
- Updated `selectLLM(contextText string, requiredOutputBudget int)` in `internal/study/service.go`.
- Check if `fastLLMProvider` meets both input limit AND required output budget (`requiredOutputBudget <= fastLimits.MaxOutputTokens`). If either fails, evaluate and escalate to `heavyLLMProvider`.
- Set `quizTaskOutputBudget = 2500` in `internal/study/quiz_sync.go`. If the selected model's `MaxOutputTokens < quizTaskOutputBudget`, `targetCount` (number of quiz questions) is adjusted proportionally down as a deliberate fallback.

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
| `frontend/src/components/SettingsAIProvider.vue` | Added **[ Test Limits ]** buttons and result state handlers for Fast and Heavy tiers |

---

## Verification

- **Automated Tests**:
  - `go test -short ./internal/...` passed across all packages (`app`, `db`, `embeddings`, `extension`, `llm`, `notebook`, `retrieval`, `runtime`, `scheduler`, `study`, `utils`).
  - `go test -run=^$ ./internal/...` passed for instant compilation and type-checking verification.
