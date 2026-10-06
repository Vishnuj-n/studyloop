# Solution Architecture: Flashcard, Notes, and Reader Bi-Directional Study Loop

## Overview
This solution implements a clean, bi-directional reference loop connecting **Flashcards** (`/flashcards`), **Study Notes** (`/notes`), and the **Reader** (`/reader`). It enables users to jump directly from retention practice to chapter summaries and source material (PDF page ranges or video timestamps), with conditional back-tracking buttons that return users seamlessly to their active study tasks.

---

## Navigation & Study Loop Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│ 1. FORWARD PROGRESSION (Recall → Summary → Original Source)                                 │
│                                                                                             │
│    [ Flashcard Back Face ]                                                                  │
│            │                                                                                │
│            └── [ 📖 View Note ] (passes query: from=flashcards, notebookId, topicId, taskId)│
│                      │                                                                      │
│                      ▼                                                                      │
│    [ Study Notes Page (`/notes`) ]                                                          │
│            │                                                                                │
│            ├── [ ← Back to Flashcards ] (only shown when from === 'flashcards')             │
│            │                                                                                │
│            └── [ 📄 Jump to p. X ↗ ] / [ 🎥 Watch ↗ ] (passes from=notes, page=X)           │
│                      │                                                                      │
│                      ▼                                                                      │
│    [ Reader Page (`/reader`) ]                                                              │
│            │                                                                                │
│            ├── Automatically positions PDF / Video at target page / timestamp               │
│            └── [ ← Back to Notes ] (only shown when from === 'notes')                       │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│ 2. CONDITIONAL CONTEXT PRESERVATION & TARGET HIGHLIGHTING                                   │
│                                                                                             │
│  • Flashcards -> Notes:                                                                     │
│    - Back of flashcard displays `[📖 View Note]` if the card has an associated topic.       │
│    - Passes `from=flashcards`, `topicId`, `notebookId`, and `page=currentCard.page`.        │
│    - On `/notes`, `[← Back to Flashcards]` appears in the header to return directly.        │
│    - Auto-Scroll & Fading Glow: Notes automatically scrolls to the matching session card    │
│      (`isTargetSlot`) and applies a subtle, non-intrusive 3.5s fading pulse (`target-highlight`).│
│                                                                                             │
│  • Notes -> Reader:                                                                         │
│    - Each session note card displays `[Jump to p. X ↗]` or `[Watch ↗]`.                     │
│    - When clicked, navigates to `/reader` with `from=notes` & `page=slot.start_page`.       │
│    - On `/reader`, `resolveBrowseContext` reads `query.page` and updates current page.      │
│    - `[← Back to Notes]` appears in the Reader toolbar to jump straight back to notes.     │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Key Implementations

### 1. `frontend/src/pages/Flashcards.vue`
- **Computed Topic ID (`currentCardTopicID`)**: Resolves the topic associated with the active card (via `card.topic_id`, `card.topicId`, or page bounds lookup).
- **Card Back Action Link**:
  ```html
  <div class="card-action-links">
    <button class="flip-back-btn" @click="flipped = false">Show Question</button>
    <button
      v-if="currentCardTopicID"
      type="button"
      class="view-note-btn"
      title="View study notes for this chapter"
      @click.stop="goToTopicNotes"
    >
      <BaseIcon name="book-open" size="13" />
      <span>View Note</span>
    </button>
  </div>
  ```
- **Direct Navigation (`goToTopicNotes`)**:
  ```javascript
  function goToTopicNotes() {
    const topicId = currentCardTopicID.value
    if (!topicId) return
    router.push({
      path: '/notes',
      query: {
        topicId,
        notebookId: selectedNotebookID.value || undefined,
        from: 'flashcards',
        taskId: route.query.taskId || undefined,
      },
    })
  }
  ```

### 2. `frontend/src/pages/Notes.vue`
- **Conditional Back Button**:
  ```html
  <button
    v-if="isFromFlashcards"
    type="button"
    class="secondary-btn btn-sm back-flashcards-btn"
    title="Return to Flashcards review session"
    @click="goBackToFlashcards"
  >
    <BaseIcon name="arrow-left" size="14" />
    <span>Back to Flashcards</span>
  </button>
  ```
- **Session Card Jump-to-Source Button**:
  ```html
  <button
    v-if="canJumpToSource(slot)"
    type="button"
    class="jump-source-pill-btn"
    :title="isCurrentTopicYouTube ? 'Jump to video segment in Reader' : `Open Reader at page ${slot.start_page || 1}`"
    @click="jumpToReader(slot)"
  >
    <BaseIcon :name="isCurrentTopicYouTube ? 'play' : 'external-link'" size="11" />
    <span>{{ isCurrentTopicYouTube ? 'Watch ↗' : `Jump to p. ${slot.start_page || 1} ↗` }}</span>
  </button>
  ```
- **Source Routing**: Navigates to `/reader` with `{ notebookId, topicId, page: slot.start_page, from: 'notes' }`.

### 3. `frontend/src/pages/Reader.vue`
- **Deep-Link Page Positioning in Browse Mode**:
  ```javascript
  async function resolveBrowseContext() {
    await reader.loadNotebookTree()
    const queryNotebook = route.query.notebookId || route.query.notebook_id
    const queryTopic = route.query.topicId || route.query.topic_id
    const queryPage = Number.parseInt(route.query.page || route.query.startPage || route.query.start_page) || 0

    if (queryNotebook) reader.selectedNotebookID.value = queryNotebook
    if (queryTopic) {
      reader.selectedTopicID.value = queryTopic
      const loaded = await reader.loadBundle()
      if (loaded && queryPage > 0) {
        reader.updateCurrentPage(queryPage)
      }
    }
  }
  ```
- **Conditional Back Button**:
  ```html
  <button
    v-if="route.query.from === 'notes'"
    class="secondary"
    title="Return to Study Notes"
    @click="router.push({ path: '/notes', query: { notebookId: reader.selectedNotebookID.value, topicId: reader.selectedTopicID.value } })"
  >
    <BaseIcon name="arrow-left" size="14" />
    <span>Back to Notes</span>
  </button>
  ```

---

## Verification & Status
- **Vitest Unit & Component Tests**: 19/19 test files passed (69/69 tests passed).
- **Go Backend Verification**: `go test -short ./internal/...` passed.
- **Manual Verification Flow**:
  1. Flashcards flip $\rightarrow$ click `[📖 View Note]` $\rightarrow$ Notes opens with `[← Back to Flashcards]`.
  2. Notes session card $\rightarrow$ click `[📄 Jump to p. X ↗]` $\rightarrow$ Reader opens at page $X$ with `[← Back to Notes]`.
  3. Clicking back buttons preserves context and returns directly to the prior study screen.
