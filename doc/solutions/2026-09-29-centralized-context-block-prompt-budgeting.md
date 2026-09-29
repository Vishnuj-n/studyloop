# Solution: Centralized String Block & Citation Prompt Budgeting

**Date:** 2026-09-29  
**Module:** Study Pipelines & Prompt Budgeting (`internal/study/prompt_budget.go`, `internal/study/reader_ai.go`, `internal/study/socratic.go`, `internal/study/prompt_budget_test.go`)

---

## 1. Problem & Context

When centralized prompt budgeting was initially implemented for database models (`[]models.ChunkWithContext` via `BudgetChunksToLimit`), lightweight retrieval endpoints (`socratic.go` and `reader_ai.go`) still operated on raw string slices and synchronized citation tags (`blocks []string, citations []string`).

Because there was no slice-of-strings helper in `prompt_budget.go`, both `reader_ai.go` and `socratic.go` ended up duplicating ~35–40 lines of manual token accumulation, boundary truncation, and fallback logic inline.

### Architectural Divergence
- **Redundant Loop Arithmetic**: Manual `embeddings.CountTokens` loops were duplicated across `reader_ai.go` and `socratic.go`.
- **Parallel Slice Drift**: Risk of slice misalignment between `blocks`, `citations`, and `chunkTexts` during boundary truncation.
- **Untested Invariants**: The ad-hoc token slicing logic in `reader_ai.go` and `socratic.go` could not be tested directly in isolation without spinning up full LLM/retrieval fixtures.

---

## 2. Solution: `BudgetContextBlocks`

A canonical helper was added to [`internal/study/prompt_budget.go`](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/study/prompt_budget.go) that fulfills the SSoT prompt budgeting invariant for string blocks and parallel citations:

```go
// BudgetContextBlocks fits string blocks and synchronized citations/metadata into available tokens,
// cleanly truncating the boundary chunk when partial budget remains.
func BudgetContextBlocks(blocks []string, citations []string, tokenBudget int) ([]string, []string, error)
```

### Key Capabilities
1. **Accurate Token Accumulation**: Uses the shared tokenizer to measure each block against the available token budget.
2. **Boundary Chunk Truncation**: When partial budget remains for the boundary block, it cleanly truncates using `embeddings.TruncateToTokens` while keeping citations synchronized.
3. **Graceful Fallback**: If the budget is very tight, it retains a truncated portion of the first block within the safe token limit rather than discarding all context.

---

## 3. Changes Applied

### A. Centralized Prompt Budgeting Engine (`internal/study/prompt_budget.go`)
- **[ADDED]** `BudgetContextBlocks(blocks []string, citations []string, tokenBudget int) ([]string, []string, error)`

### B. Reader AI (`internal/study/reader_ai.go`)
- **[DELETED]** 38 lines of inline token counting and truncation loop.
- **[ADDED]** Single call to `BudgetContextBlocks`:
  ```go
  newBlocks, newCitations, err := BudgetContextBlocks(blocks, citations, available)
  if err != nil {
      return map[string]interface{}{"error": err.Error()}
  }
  ```

### C. Socratic Rescue & Tutor (`internal/study/socratic.go`)
- **[DELETED]** 42 lines of inline block loop and truncation logic.
- **[ADDED]** Clean call to `BudgetContextBlocks` and synchronized `chunkTexts` slice bounds:
  ```go
  newBlocks, newCitations, err := BudgetContextBlocks(blocks, citations, available)
  if err != nil {
      return nil, fmt.Errorf("error budgeting context blocks: %w", err)
  }

  newChunkTexts := chunkTexts
  if len(newBlocks) < len(chunkTexts) {
      newChunkTexts = chunkTexts[:len(newBlocks)]
  }
  ```

### D. Unit Tests (`internal/study/prompt_budget_test.go`)
- **[ADDED]** Test suite `TestBudgetContextBlocks` covering:
  - Empty input handling
  - Zero/negative budget handling
  - Full fit within generous budget
  - Boundary block partial truncation
  - Minimal budget fallback retention

---

## 4. Verification

- `go test -v ./internal/study -run=TestBudgetContextBlocks` passed.
- `go test -short ./internal/...` passed across all packages.
