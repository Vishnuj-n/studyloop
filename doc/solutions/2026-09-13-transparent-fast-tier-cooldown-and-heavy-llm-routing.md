# Transparent Fast Tier Cooldown & Heavy LLM Routing

## Problem

1. **Silent HTTP 429 Failures**: When the Fast LLM provider returned HTTP 429 (Rate Limit / TPM limit reached), the app presented generic or raw API errors to the user without indicating why the request failed or what to do next.
2. **Lack of Dynamic Cooldown & Failover**: If Fast LLM hit rate limits, subsequent requests continued to blindly hit the same rate-limited provider endpoint rather than temporarily cooldown and route to a configured Heavy LLM tier.

## Solution Architecture

### 1. HTTP 429 Rate Limit Detection & Classification
- Added `RateLimitError` struct and `IsRateLimitError(err error) bool` helper in `internal/llm/provider.go`.
- Updated `GenerateAnswer` to detect `http.StatusTooManyRequests` (429) and return a structured `RateLimitError`.

### 2. Fast Tier Cooldown Tracking & Capability Routing
- Added `fastRateLimitedUntil time.Time` and mutex protection to `StudyService` in `internal/study/service.go`.
- Added `MarkFastRateLimited(cooldownDuration time.Duration)` (default 45s) and `IsFastRateLimited() bool`.
- Updated `selectLLM` to check `IsFastRateLimited()`: if Fast tier is in active cooldown and `heavyLLMProvider` is available, requests route automatically to `heavyLLMProvider` (tier `"heavy"`).

### 3. Transparent User Error Formatting
- Added `FormatLLMError(err error, tier string) error` on `StudyService`.
- If `heavyLLMProvider` is configured and distinct:
  `"Fast provider rate-limited (status 429). Please try again — next attempt will use Heavy provider (<ModelName>)."`
- If `heavyLLMProvider` is NOT configured or identical to Fast:
  `"Fast provider rate-limited (status 429: TPM limit reached). Please try again in a few seconds, or configure a Heavy provider (e.g. Gemini AI Studio) in Settings."`
- Integrated `FormatLLMError` across `examiner.go`, `quiz_sync.go`, `flashcard.go`, `socratic.go`, `simplifier.go`, and `reader_ai.go`.

---

## File Changes Map

| File Path | Description |
|---|---|
| `internal/llm/provider.go` | Added `RateLimitError`, `IsRateLimitError`, and HTTP 429 status handling in `GenerateAnswer` |
| `internal/study/service.go` | Added `fastRateLimitedUntil`, `MarkFastRateLimited`, `IsFastRateLimited`, `FormatLLMError`, and updated `selectLLM` |
| `internal/study/examiner.go` | Updated LLM generation error paths to use `FormatLLMError` |
| `internal/study/quiz_sync.go` | Updated `GenerateQuizSync` error handling to use `FormatLLMError` |
| `internal/study/flashcard.go` | Updated flashcard core generation error handling to use `FormatLLMError` |
| `internal/study/socratic.go` | Updated socratic prompt and response error handling to use `FormatLLMError` |
| `internal/study/simplifier.go` | Updated text simplifier error handling to use `FormatLLMError` and bound `tier` in `SimplifyReadingContent` |
| `internal/study/reader_ai.go` | Updated reader sidebar error handling to use `FormatLLMError` |
| `internal/study/service_test.go` | Added unit tests `TestFastTierCooldownRouting` and `TestFormatLLMErrorMessages` |

---

## Verification

- **Automated Tests**:
  - `go test -short ./internal/...` passed across all internal packages.
  - Full `go test ./internal/...` passed 100% cleanly without errors.
