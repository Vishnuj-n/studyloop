# Solution Report: Unified Shortcut Controller & Command Palette Actions

## 1. Quick Pomodoro & Settings Actions via `Ctrl + Shift + P`

The Profile Bar has been enhanced into a full **Quick Action & Profile Command Palette** ([`ProfileCommandPalette.vue`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/components/ProfileCommandPalette.vue)):

### ⏱ Start / Pause 25m Pomodoro:
- Press <kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>P</kbd>.
- The first action selected by default is **"Start 25m Pomodoro Focus"** (or **"Pause Pomodoro Focus"** when active).
- Press <kbd>Enter</kbd> immediately:
  - Toggles the Pomodoro focus timer without touching the mouse or searching for the sidebar widget.
  - Dynamically updates subtitle and badge with live time remaining (e.g. `Timer active (24:45 left) - Running`).

### ⚙ Direct Access to Settings:
- Press <kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>P</kbd>.
- Type `set` or hit <kbd>↓</kbd> to reach **"Open Settings"**.
- Press <kbd>Enter</kbd>:
  - Immediately navigates to `/settings` without sidebar navigation.

---

## 2. Updated Interaction Model & Direct Profile Switching

```
Press Ctrl + Shift + P
 ├── ⏱ [Item 0] Start / Pause 25m Pomodoro Focus  ──> [Enter] (Toggle focus mode)
 ├── ⚙ [Item 1] Open Settings                     ──> [Enter] (Direct jump to config)
 ├── 👤 [Item 2] Study Profile 1                  ──> [1 or Enter] (Direct profile switch)
 └── 👤 [Item 3] Study Profile 2                  ──> [2 or Enter] (Direct profile switch)
```

- **Instant 1 & 2 Profile Shortcuts**:
  - In both [`ProfileCommandPalette.vue`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/components/ProfileCommandPalette.vue) and [`ProfileSelect.vue`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/pages/ProfileSelect.vue), pressing `1` or `2` (either from the standard number row or the numeric keypad `Numpad1`/`Numpad2`) directly selects that study profile without polluting the search input or getting swallowed.
  - Profile switching dispatches both `'profile-switched'` and `'settings-updated'`, ensuring Dashboard agenda, active task queue, and widgets reactively re-fetch without manual reloads.
- **Instant Search**: Typing `pomo`, `focus`, `timer`, `set`, `config`, or any profile name instantly isolates the item.
- **Key Navigation**: <kbd>↑</kbd> / <kbd>↓</kbd> wraps around all items seamlessly, and <kbd>Esc</kbd> closes.

---

## 3. Test & Build Status

| Suite | Result | Details |
| :--- | :--- | :--- |
| `src/components/ProfileCommandPalette.spec.js` | **7/7 passed** | Verifies 1/2 key shortcut in input, Numpad code, settings-updated event |
| `src/composables/useShortcuts.spec.js` | **9/9 passed** | Supports key, physical code (`Digit`/`Numpad`), modifiers, scope stack |
| `src/pages/ProfileSelect.spec.js` | **5/5 passed** | Top-row and Numpad digits, arrow navigation, Enter/Space |
| `src/pages/Flashcards.spec.js` | **5/5 passed** | Unified shortcut scope integration |
| **Vite Production Build** | **`✓ built in 21.64s` (0 errors)** | Clean compilation |
