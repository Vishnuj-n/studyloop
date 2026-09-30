# Solution: Settings RAG Status Feedback & Instant Search Filter

## Context & Motivation
As the Settings interface grew with numerous configuration options across LLMs, local RAG retrieval, study budgets, profiles, extensions, themes, and developer diagnostics, users needed:
1. **Clear visual feedback on local ONNX vector retrieval (RAG) state**: Distinguishing whether embeddings are dormant in storage, actively loading into RAM, or ready for immediate Socratic Q&A.
2. **Instant search and navigation across configuration options**: Finding specific settings (e.g., API keys, ONNX models, pomodoro timers, dark themes, voice persona) without manually clicking through every category tab.

## Solution Architecture

### 1. Dynamic RAG Status Badges & On-Demand Setup Launcher
- **Dynamic Status Pill in [`SettingsRagRetrieval.vue`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/components/SettingsRagRetrieval.vue)**:
  - 🟢 **Active in RAM / Ready**: When RAG is enabled and ONNX vector models are active in memory.
  - 🟡 **Setting Up (`ragStatus`)**: Animated pulsating indicator with live stage status (`checking`, `acquiring`, `extracting`, `initializing`) during background download and staging.
  - ⚪ **Dormant / Off**: Clean neutral indicator when RAG is disabled.
- **Direct Setup Trigger**:
  - Added a **Re-index / Setup** action button next to the status badge when RAG is enabled.
  - Exported `openRagModal()` in [`useRAG.js`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/composables/useRAG.js) to trigger `SettingsRagModal` on demand without having to toggle settings off and back on.

### 2. Instant Category Keyword Search & Filter Rail
- **Search Bar in [`Settings.vue`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/pages/Settings.vue)**:
  - Added a search input (`🔍 Search settings...`) at the top of the category rail with an instant clear (`✕`) button.
  - Deep keyword index mapped to each of the 6 settings categories:
    - `study`: budgets, tokens, duration, quiz, rescue, reading, schedule, pomodoro.
    - `ai`: llm, retrieval, rag, onnx, vector, embeddings, api keys, gemini, openai, groq, tutor persona, socratic.
    - `profiles`: goals, notebooks, textbooks, exam, deadline, pace, syllabus.
    - `extensions`: podcast, voice personas, audio, simplifier, comprehension, tts.
    - `system`: theme, dark/light mode, cloud sync, student login, classroom, updates.
    - `dev`: diagnostics, dev logs, vector db, reading task history.
- **Intelligent Navigation**:
  - Highlights matched categories with a subtle `Match` badge.
  - Automatically switches the active viewport pane to the highest-matching category if the current tab doesn't match the search query.
  - Displays a clean empty state message when no categories match.

## Modified Files
- `frontend/src/composables/useRAG.js`
- `frontend/src/components/SettingsRagRetrieval.vue`
- `frontend/src/pages/Settings.vue`

## Verification
- `go test -short ./internal/...` passed with 0 errors.
- `npm run build` in `frontend/` succeeded with clean production bundle compilation.
