# Partial Quiz Question Recovery on LLM Output Truncation

## Problem

When an LLM response reaches output token limits (`max_tokens`) during quiz generation, the trailing JSON payload gets truncated mid-question (e.g. while outputting question 6 of 8). Previously, `json.Unmarshal` failed completely with a syntax error, discarding all previously generated valid questions and presenting a generation failure to the user.

## Solution Architecture

### 1. Truncated JSON Repair (`recoverTruncatedQuizJSON`)
- Detects the last complete question object (`}`) inside the `"questions"` array when standard JSON parsing fails.
- Trims any trailing commas or malformed partial tokens following the last valid closing brace.
- Automatically repairs and balances unclosed array `]` and object `}` brackets to form valid JSON.

### 2. Partial Question Fallback Parsing (`parseQuizLLMResponse`)
- In `internal/study/parsers.go`, `parseQuizLLMResponse` first attempts standard JSON unmarshalling.
- If standard unmarshalling fails, it runs `recoverTruncatedQuizJSON` to salvage all fully completed questions.
- Logs a warning with the count of recovered questions (`[QUIZ_RECOVERY] recovered N valid questions from truncated LLM response`).
- If at least 1 valid question is recovered, the request succeeds and returns the salvaged questions.

---

## File Changes Map

| Module / Path | Description |
|---|---|
| `internal/study/parsers.go` | Added `recoverTruncatedQuizJSON` helper and updated `parseQuizLLMResponse` to salvage partial quiz questions |
| `internal/study/parsers_test.go` | Added unit tests (`TestParseQuizLLMResponse_ValidComplete`, `TestParseQuizLLMResponse_PartialRecovery`, `TestParseQuizLLMResponse_PartialRecoveryWithTrailingComma`) |

---

## Verification

- **Automated Tests**:
  - Ran `go test -short ./internal/study/...` — all parser unit tests passed cleanly.
  - Ran `go test ./internal/...` — full repository test suite passed 100% green without regressions.
