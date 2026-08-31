# Solution: Complete Elimination of Hardcoded Token Limits & Unified Dynamic Prompt Budgeting

**Date:** 2026-08-31  
**Module:** AI Tutor Study Pipelines & Prompt Budgeting (`internal/study/prompt_budget.go`, `internal/study/quiz_sync.go`, `internal/study/flashcard.go`, `internal/study/socratic.go`, `internal/study/examiner.go`, `internal/study/simplifier.go`, `internal/study/audio_overview.go`, `internal/notebook/syllabus.go`)

---

## 1. Problem & Architectural Rationale

Prior to this refactor, several AI generation endpoints relied on ad-hoc, hardcoded token subtractions, magic numbers, or character-length heuristics to prevent exceeding LLM context windows:
- **Magic number arithmetic**: Pipelines subtracted hardcoded values (e.g. `limit - 300 - 500` or `limit - 1500`) to account for instructions, schemas, and question requirements.
- **Character length approximations**: Certain generators divided character counts by 4 (`len(text) / 4`) instead of using the tokenizer.
- **Code duplication**: Reverse-turn conversation budgeting, chunk-slicing, and prompt template overhead measurement were independently implemented across multiple files.
- **Invariant violation**: Hardcoded numbers violate the AGENTS.md invariant (*"No Hardcoded Fallbacks (Fail Fast) — Always use user-configured or database-persisted settings dynamically"*).

---

## 2. Centralized Architecture: `internal/study/prompt_budget.go`

All token budgeting algorithms were unified into a single source of truth in `internal/study/prompt_budget.go`:

| Function | Purpose | Implementation Detail |
| :--- | :--- | :--- |
| `CalculateAvailableContextBudget(maxInput, template)` | Computes remaining token budget for dynamic content | Measures exact tokens of the prompt template (system instructions, JSON schema, rules) via `embeddings.CountTokens` and subtracts from `MaxInputTokens`. Fails fast if template exceeds input limit. |
| `BudgetChunksToLimit(chunks, budget)` | Token-accurate chunk budgeting | Sequentially adds retrieval chunks while cumulative tokens fit strictly within `budget`. |
| `BudgetConversationHistory(history, budget)` | Reverse-turn conversation budgeting | Iterates backwards from newest turn to oldest turn, accumulating message tokens until budget is reached, preserving recent dialogue. |
| `BudgetTextSample(sample, budget)` | Exact text budgeting | Uses `embeddings.TruncateToTokens` to strictly fit raw text samples within budget. |

---

## 3. What Was Added vs What Was Deleted

### A. Centralized Prompt Budgeting Engine (`internal/study/prompt_budget.go`)
- **[ADDED]** `CalculateAvailableContextBudget(maxInputTokens int, templateText string) (int, error)`
- **[ADDED]** `BudgetChunksToLimit(chunks []models.ChunkWithContext, tokenBudget int) ([]models.ChunkWithContext, error)`
- **[ADDED]** `BudgetConversationHistory(history []map[string]string, tokenBudget int) ([]map[string]string, error)`
- **[ADDED]** `BudgetTextSample(sample string, tokenBudget int) (string, error)`

---

### B. Quiz Generation (`internal/study/quiz_sync.go`)
- **[DELETED]** Hardcoded overhead subtractions:
  ```go
  // DELETED
  tokenBudget := limits.MaxInputTokens - 300 - 500
  ```
- **[DELETED]** Duplicate chunk truncation loop.
- **[ADDED]** Dynamic template measurement and unified chunk budgeting:
  ```go
  templatePrompt := buildDeterministicQuizPrompt("", req.QuestionCount, req.PassingScore, nil)
  availableBudget, err := CalculateAvailableContextBudget(limits.MaxInputTokens, templatePrompt)
  budgetedChunks, err := BudgetChunksToLimit(chunks, availableBudget)
  ```

---

### C. Flashcard Generation (`internal/study/flashcard.go`)
- **[DELETED]** Arbitrary overhead numbers:
  ```go
  // DELETED
  overhead := 1500
  tokenBudget := limits.MaxInputTokens - overhead
  ```
- **[ADDED]** Tokenizer-accurate template budget calculation:
  ```go
  templatePrompt := buildDynamicPromptWithTemplate("", topicName, "")
  availableBudget, err := CalculateAvailableContextBudget(limits.MaxInputTokens, templatePrompt)
  budgetedChunks, err := BudgetChunksToLimit(chunks, availableBudget)
  ```

---

### D. Socratic Remedial Dialogue (`internal/study/socratic.go`)
- **[DELETED]** Ad-hoc reverse-turn string building and inline token budget arithmetic.
- **[ADDED]** Unified conversation history budgeting:
  ```go
  selectedHistory, err := BudgetConversationHistory(history, historyBudget)
  ```

---

### E. Syllabus Generation (`internal/notebook/syllabus.go`)
- **[DELETED]** Character-length heuristics (`maxChars = tokenBudget * 4`) and string slicing `[:maxChars]`.
- **[ADDED]** Token-accurate sample truncation:
  ```go
  templatePrompt := buildSyllabusPrompt("", "")
  availableBudget, err := study.CalculateAvailableContextBudget(limits.MaxInputTokens, templatePrompt)
  budgetedSample, err := study.BudgetTextSample(sample, availableBudget)
  ```

---

### F. Extensions & Auxiliary Services
- **Audio Overview (`internal/study/audio_overview.go`)**: Uses `CalculateAvailableContextBudget` + `embeddings.TruncateToTokens`.
- **Text Simplifier (`internal/study/simplifier.go`)**: Uses `CalculateAvailableContextBudget` + `embeddings.TruncateToTokens`.
- **Comprehensive Examiner (`internal/study/examiner.go`)**: Uses `CalculateAvailableContextBudget` + `BudgetChunksToLimit`.

---

## 4. Verification

1. **Codebase Grep Audit**: Verified that no arbitrary `000` / hardcoded token constants remain in any AI pipeline.
2. **Unit & Integration Tests**:
   - `go test -short ./internal/...` passed with zero errors.
   - All study, notebook, LLM, and app test suites passed cleanly.
