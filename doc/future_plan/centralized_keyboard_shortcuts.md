# Centralized Keyboard Shortcut Architecture Specification

## 1. Overview & Objective
Establish a unified, reactive keyboard shortcut subsystem (`useKeyboardShortcuts.js`) for Studyloop that replaces scattered, ad-hoc `window.addEventListener('keydown')` listeners across individual Vue components.

This system guarantees:
1. **Desktop Webview Focus Reliability**: Automatic document focus restoration on window launch and route transitions so WebView2 never drops keystrokes on desktop cold-boot.
2. **Global & Contextual Layering**: Seamless coexistence of app-wide shortcuts (e.g. `Ctrl+Shift+P`, `Ctrl+,`) and page-specific hotkeys (e.g. Flashcards `1..4`, Profile Chooser `1..9`).
3. **Modal & Form Input Isolation**: Automatic suppression of page hotkeys when typing in form inputs (`<input>`, `<textarea>`, `contenteditable`) or when a modal/dialog overlay is active.

---

## 2. Background & Problem Statement

Prior to centralization, keyboard shortcuts were wired independently in each component:
- `Flashcards.vue` bound `window.addEventListener('keydown')` for `Space`, `1..4`, and `Shift+S`.
- `ProfileSelect.vue` bound `window.addEventListener('keydown')` for `1..9`, `Enter`, and Arrow keys.
- `ConfirmModal.vue`, `FlashcardDeckManagerModal.vue`, and others registered their own isolated listeners.

### Known Pain Points & Root Causes:
1. **WebView2 Cold Launch Unfocused Document**:
   On Windows desktop (`Studyloop.exe`), Wails creates a native Win32 window host. Edge WebView2 loads inside it without granting DOM focus to the document until a user clicks inside the page. At cold startup into `/select-profile`, number keys (`1`, `2`) had no response because keystrokes were absorbed by the native window frame.
2. **Modal Bleed-Through**:
   When a modal (like Settings or ConfirmModal) opened over `Flashcards.vue`, pressing keys could still inadvertently trigger card flipping or rating actions in the background.
3. **Code Duplication**:
   Every component had to re-implement input target checks (`e.target.tagName === 'INPUT' || e.target.isContentEditable`) and manually clean up event listeners on unmount.

---

## 3. Architecture & Design

### Composable: `useKeyboardShortcuts.js`

```text
┌────────────────────────────────────────────────────────────────┐
│               Global Hotkey Manager (useKeyboardShortcuts)     │
│                 Installed at App root (App.vue)                │
└────────────────────────────────┬───────────────────────────────┘
                                 │
         ┌───────────────────────┴───────────────────────┐
         ▼                                               ▼
┌──────────────────────────────┐        ┌──────────────────────────────┐
│       Global Shortcuts       │        │   Active Scoped Shortcut     │
│  - Ctrl+Shift+P (Profiles)   │        │   Stack (Top-of-Stack Only)  │
│  - Ctrl+, (Settings)         │        └──────────────┬───────────────┘
│  - Ctrl+/ (Shortcuts Help)   │                       │
└──────────────────────────────┘                       │
                                   ┌───────────────────┴───────────────────┐
                                   ▼                                       ▼
                     ┌───────────────────────────┐           ┌───────────────────────────┐
                     │     Page-Level Scope      │           │     Modal-Level Scope     │
                     │  - Flashcards (1..4, Spc) │           │  (Pauses Page Scope)      │
                     │  - Profile Chooser (1..9) │           │  - Esc (Dismiss)          │
                     │  - Reader (Arrows, J/K)   │           │  - Enter (Confirm)        │
                     └───────────────────────────┘           └───────────────────────────┘
```

### Core Architecture Rules:
1. **Single Event Listener**: One capture-phase listener on `window` dispatches events through an active handler stack.
2. **Scope Stack**:
   - Pushing a modal scope onto the stack suspends lower scopes (preventing background card reviews or page navigation while modals are open).
   - Popping a scope on unmount automatically restores the underlying page scope.
3. **Input Guarding**: Keystrokes targeting `INPUT`, `TEXTAREA`, `SELECT`, or elements with `isContentEditable` are ignored unless explicitly designated as `allowInInputs: true`.
4. **WebView2 Focus Recovery**: Automatically calls `window.focus()` and sets DOM focus on the active root container on route navigation.

---

## 4. Planned Shortcut Registry

### 4.1 Global Shortcuts (Available Everywhere)
| Shortcut | Action | Description |
| :--- | :--- | :--- |
| `Ctrl+Shift+P` / `Cmd+Shift+P` | Open Profile Chooser | Quickly switch study profile without leaving current task |
| `Ctrl+,` / `Ctrl+Shift+S` | Open Settings | Jump straight to application preferences |
| `Ctrl+/` / `?` | Keyboard Shortcuts Cheatsheet | Open an overlay modal showing all available shortcuts |
| `Escape` | Dismiss Overlay / Cancel | Close active modal, drawer, or search dropdown |

### 4.2 Profile Chooser (`/select-profile`)
| Shortcut | Action | Description |
| :--- | :--- | :--- |
| `1` .. `9` | Select Profile | Instantly switch to Profile #1 through #9 |
| `ArrowUp` / `ArrowDown` | Navigate Profiles | Move focus between profile cards |
| `ArrowLeft` / `ArrowRight` | Navigate Profiles | Move horizontal focus across grid |
| `Enter` | Select Focused | Launch selected profile workspace |

### 4.3 Flashcards Mode (`/flashcards`)
| Shortcut | Action | Condition | Description |
| :--- | :--- | :--- | :--- |
| `Space` | Flip Card | Question or Answer | Toggle between prompt and answer |
| `1` | Rate Again | Answer revealed | FSRS rating: Again (forget, review soon) |
| `2` | Rate Hard | Answer revealed | FSRS rating: Hard (struggled) |
| `3` | Rate Good | Answer revealed | FSRS rating: Good (expected recall) |
| `4` | Rate Easy | Answer revealed | FSRS rating: Easy (immediate recall) |
| `Shift+S` | Suspend Card | Any | Remove poor card from active study queue |
| `V` | View Chapter Note | Any | Open chapter summary in Study Notes |
| `P` | Open PDF Page | Card has page | Jump to textbook page in Reader |

### 4.4 Reader Mode (`/reader`)
| Shortcut | Action | Description |
| :--- | :--- | :--- |
| `ArrowLeft` / `ArrowRight` | Prev / Next Page | Step through textbook pages |
| `J` / `K` | Scroll Segment Down / Up | Smooth reading increment |
| `Ctrl+F` | Find in Textbook | Focus in-book search bar |

---

## 5. Implementation Roadmap (PR Scope)
- [ ] Create `frontend/src/composables/useKeyboardShortcuts.js` with stack-based scope registration.
- [ ] Register global shortcuts in `App.vue` (`Ctrl+Shift+P`, `Ctrl+,`).
- [ ] Migrate `ProfileSelect.vue` hotkeys to `useKeyboardShortcuts` scope.
- [ ] Migrate `Flashcards.vue` hotkeys to `useKeyboardShortcuts` scope.
- [ ] Migrate `ConfirmModal.vue` and `FlashcardDeckManagerModal.vue` to push modal scopes onto the stack.
- [ ] Add unit tests in `frontend/src/composables/useKeyboardShortcuts.spec.js`.
