# Flashcard Generation Bias Mitigation & Empirical Benchmarking

## Problem

1. **Distractor Poisoning (Negative Priming)**: Flashcard generation previously included the user's incorrect quiz selection (`User's wrong selection: [X]`). This primed LLM attention on the faulty premise, causing models to generate negative-formulation cards ("Why is X wrong?") rather than testing ground-truth textbook concepts.
2. **Attention Clumping & Topic Starvation**: When passed failed quiz questions, the LLM heavily anchored on the lexical neighborhood of the missed question. In multi-page chapters (e.g., pages 1–10), missed concepts on page 2 led to 3–4 flashcards clustered around page 2, starving pages 3–10 of foundational spaced-repetition coverage.
3. **Hard-Slice Truncation Mismatch**: The backend enforces `maxCardsAllowed = 5 + len(failedQuestions)`. When biased upfront generation filled all card slots with narrow misconception cards, broad chapter coverage cards were pruned by the array slice.

---

## Solution Architecture

### 1. Distractor-Free Reinforcement Prompting
- Removed raw `UserAnswer` strings from the prompt payload.
- Formatted misconception directives around the tested concept and correct ground truth (`- Tested Concept: %s | Correct Ground Truth: %s`).

### 2. Explicit Quota Partitioning
- Partitioned prompt generation targets into two distinct buckets:
  - **Baseline Coverage**: 5 evenly distributed cards across the entire document page range (`startPage`-`endPage`).
  - **Targeted Remediation**: Exactly 1 supplementary card per missed quiz concept.
- Added explicit prompt rule: *"Do NOT allow these targeted topics to crowd out or replace baseline coverage for the remaining pages."*

### 3. Standalone Benchmark & Unit Testing
- Created `scripts/test_flashcard_bias.go` to test and evaluate chunk spread, distractor isolation, and attention non-interference metrics across simulated multi-page chapters.
- Added `TestBuildMarathonFlashcardPromptWithBudget_MitigatesMisconceptionBias` in `internal/study/flashcard_test.go`.

---

## File Changes Map

| Module / Path | Description |
|---|---|
| `internal/study/flashcard.go` | Updated `buildMarathonFlashcardPromptWithBudget` and `buildFlashcardStaticTemplate` to partition card quotas and eliminate distractor poisoning |
| `internal/study/flashcard_test.go` | Added unit test verifying bias mitigation, balanced coverage directives, and distractor isolation |
| `scripts/test_flashcard_bias.go` | Added standalone benchmark script to evaluate flashcard prompt bias across multi-page document chunks |

---

## Verification

- **Standalone Benchmark Script**:
  - `go run scripts/test_flashcard_bias.go` passed with full metric scorecard verification.
- **Unit Tests**:
  - `go test -v ./internal/study -run TestBuildMarathonFlashcard` passed.
  - `go test -short ./internal/...` passed across all internal packages.
