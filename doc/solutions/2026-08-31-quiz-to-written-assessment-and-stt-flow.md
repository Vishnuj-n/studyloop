# Solution: Quiz to Written Assessment Flow & Speech-to-Text Integration

**Date:** 2026-08-31  
**Module:** Frontend Assessment (`frontend/src/pages/Quiz.vue`, `frontend/src/pages/WrittenAssessment.vue`)

---

## 1. Problem & Context

1. **Underutilized Examiner Synthesis Feature**:
   - The backend contains a robust `GenerateComprehensiveExam` and `ScoreShortAnswer` pipeline in `internal/study/examiner.go`, which generates short-answer questions across page ranges and grades them ($1\text{–}10$) with concise qualitative feedback.
   - However, Examiner was only accessible via manual page range input on the `/examiner` route, disconnected from the primary queue progression loop ($\text{Reading} \rightarrow \text{Quiz} \rightarrow \text{Flashcards}$).

2. **High Friction for Written/Spoken Synthesis**:
   - Long-form written synthesis answers can feel tedious to type manually.
   - A frictionless way to verbally dictate answers (oral defense / viva) using offline speech-to-text tools like [Handy](https://github.com/cjpais/Handy) or OS dictation (`Win + H`) without introducing complex audio driver/CGO dependencies into the Go backend was needed.

---

## 2. Architectural Solution

### A. Non-Blocking Optional Transition from Quiz (`frontend/src/pages/Quiz.vue`)
- Added a **"Written Assessment"** button to the Quiz result view when a quiz passes.
- Implemented `handleGoToExaminer()`:
  - If flashcard generation is pending, it completes flashcard generation first (`generateFlashcardsForQuizTask`), keeping the queue deterministic and preserving FSRS state machine integrity.
  - Routes to `/examiner` with query parameters:
    ```javascript
    router.push({
      path: '/examiner',
      query: {
        notebookID: nbID,
        startPage: String(startP),
        endPage: String(endP),
        autoGenerate: 'true',
      },
    })
    ```

### B. Auto-Generation & Navigation Return (`frontend/src/pages/WrittenAssessment.vue`)
- In `onMounted()`, parses incoming route query params (`notebookID`, `startPage`, `endPage`, `autoGenerate`) and automatically triggers `generate()` for the active slice.
- Added a **"Back to Dashboard"** button in the score result panel (`goToDashboard`), allowing users to return directly to their study queue after receiving assessment feedback.

### C. Speech-to-Text Guidance
- Added an informative tip banner above the answer field recommending built-in OS dictation (`Win + H`) or local offline dictation via [Handy](https://github.com/cjpais/Handy) to dictate explanations directly into the textarea without any backend bloat or audio driver coupling.

---

## 3. Verification

1. **Fast Unit Tests**:
   ```bash
   go test -short ./internal/...
   ```
   Passed across all packages.

2. **Frontend Build**:
   ```bash
   cd frontend && npm run build
   ```
   Passed with zero bundle errors.
