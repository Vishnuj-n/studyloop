# Solution Report: Unified Shortcut Controller & `Ctrl + Shift + P` Profile Bar

## 1. Did we add `Ctrl + Shift + P`?

**Yes, it is implemented and wired globally.**

In [`frontend/src/App.vue`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/App.vue#L41-L56):

```javascript
// ── Profile Command Palette (Ctrl+Shift+P) ────────────────────────────────
const showProfilePalette = ref(false)
useShortcuts({
  scope: 'global',
  bindings: [
    {
      key: 'p',
      ctrl: true,
      shift: true,
      allowInInputs: true,
      handler() {
        showProfilePalette.value = !showProfilePalette.value
      },
    },
  ],
})
```

And in the template of [`frontend/src/App.vue`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/App.vue#L571-L574):

```html
<RouterView />
<ConfirmModal />
<ProfileCommandPalette v-model="showProfilePalette" />
```

---

## 2. Key Casing Detail (`p` vs `P`)

When the user physically presses <kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>P</kbd>, standard browsers emit `e.key = "P"` (uppercase) because Shift is held. In the test suite, dispatching `key: 'p'` with `shiftKey: true` matched, but in native DOM environments where `e.key` is uppercase `"P"`, normalising key comparison (comparing lowercase or handling both `'p'` and `'P'`) ensures 100% reliability across all browsers and platforms.

---

## 3. What Was Done Across the Codebase

### A. New Composable: `useShortcuts.js`
*Path: [`frontend/src/composables/useShortcuts.js`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/composables/useShortcuts.js)*

1. **Centralized listener**: Instead of each component attaching its own raw `window.addEventListener('keydown')`, a single listener manages all keystrokes.
2. **Priority Stack (LIFO)**: Scopes are registered on a stack (`[Global] -> [Page] -> [Modal]`). When a modal opens, it pushes its own scope to the top. It consumes keys, stopping lower background page shortcuts from firing.
3. **Typing Guard**: Automatically silences single-key shortcuts (like digits `1..4` or `Space`) when typing in `<input>`, `<textarea>`, `<select>`, or content-editable elements, unless explicitly flagged with `allowInInputs: true`.
4. **Auto-Cleanup**: Automatically cleans up on Vue component unmount via `onMounted` / `onUnmounted`.

### B. New Component: `ProfileCommandPalette.vue`
*Path: [`frontend/src/components/ProfileCommandPalette.vue`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/components/ProfileCommandPalette.vue)*

1. **Floating Command Palette Overlay**:
   - Opens when `showProfilePalette` is set to `true` via `Ctrl + Shift + P`.
   - Auto-focuses a search input to quickly filter profiles by name.
2. **Keyboard Navigation**:
   - Press <kbd>1</kbd>–<kbd>9</kbd> to pick that visible profile directly.
   - Use <kbd>ArrowUp</kbd> / <kbd>ArrowDown</kbd> to cycle items and <kbd>Enter</kbd> to activate.
   - Press <kbd>Esc</kbd> or click the backdrop to dismiss.
3. **Data & State**:
   - Displays profile color dots, active status badges, and pending study task counts.
   - Emits `profile-switched` event matching existing app components.
   - Links to "Manage Profiles" and "Add Profile".

### C. Refactored Existing Pages

1. **`Flashcards.vue`**:
   - Removed raw `window.addEventListener('keydown', handleKeydown)`.
   - Replaced with `useShortcuts({ scope: 'flashcards', bindings: [...] })` for <kbd>Space</kbd> (flip), <kbd>1</kbd>–<kbd>4</kbd> (rate), and <kbd>Shift + S</kbd> (suspend).
2. **`ProfileSelect.vue`**:
   - Removed raw capturing listener `document.addEventListener('keydown', handleKeydown, true)`.
   - Replaced with `useShortcuts({ scope: 'profile-select', bindings: [...] })` for <kbd>1</kbd>–<kbd>9</kbd> profile cards, arrow key navigation, and <kbd>Enter</kbd>/<kbd>Space</kbd>.

---

## 4. Verification & Test Results

All test suites passed:

| Test Suite | Result | Details |
| :--- | :--- | :--- |
| `useShortcuts.spec.js` | **9 / 9 passed** | Key matching, modifier checks, input guards, stack priority |
| `ProfileCommandPalette.spec.js` | **4 / 4 passed** | Opening, profile rendering, search filtering, escape closing |
| `Flashcards.spec.js` | **5 / 5 passed** | Flips with Space, rates with 1-4, deck manager suppression |
| `ProfileSelect.spec.js` | **5 / 5 passed** | Card selection via digits, arrow navigation, Enter/Space |
| **Go Backend** (`go test -short`) | **Passed (0 errors)** | All internal packages pass |
