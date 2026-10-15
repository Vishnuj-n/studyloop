# Solution: Dedicated Profile Selection Launch Page & Chrome-Inspired UX

## Overview

This solution documents the motivation, design decisions, and architectural implementation of transitioning the app's startup profile selection from a popup modal into a **dedicated launch page (`/select-profile`)** inspired by the **Google Chrome "Who's using Chrome?" profile picker**.

---

## Why This Page Was Added

### 1. Eliminating Startup FOUC and Black-and-White Dashboard Flashing
Previously, profile selection was implemented as an overlay modal (`ProfileLauncherModal.vue`) mounted on top of `App.vue` or `Dashboard.vue`.
- **The Problem**: Before the modal rendered, the router had already navigated to `/dashboard`. This caused `Dashboard.vue` to mount prematurely, initiate queries for the active profile's study queue and analytics, and flash the dashboard shell on screen before the modal overlay appeared.
- **The Fix**: Introducing a dedicated route [`/select-profile`](../../frontend/src/router/index.js) intercepted at the router navigation guard level (`router.beforeEach`). When a user with multiple profiles launches the application, the app navigates directly to `/select-profile` before the dashboard or sidebar are ever mounted.

### 2. Matching the Polished Google Chrome Experience
Modal dialogs with heavy borders, dark backdrops, and boxed containers feel like disruptive popups rather than a welcoming launch gate.
- **Chrome Aesthetic**: The new page adopts Google Chrome's clean profile picker architecture:
  - Clean, open, edge-to-edge canvas with subtle ambient background lighting.
  - Large circular avatar glyphs (84px) with vibrant, harmonious pastel gradient palettes.
  - Centered hero typography ("*Who is studying today?*").
  - Subtle card lift, elevation shadows, and active profile status badge.
  - Keyboard number shortcuts (`1`, `2`, `...`), arrow-key navigation, and `Enter` key activation for lightning-fast profile switching.

### 3. Clear Separation of "Add Profile" vs "Manage Profiles"
- **"Add Profile" Tile**: Placed directly within the profile card grid as an inviting dashed card. Clicking it opens the settings workspace and immediately displays the **Create New Profile** modal (`/settings?category=profiles&action=add`).
- **"Manage Profiles"**: Pinned to the bottom-right corner of the viewport, routing to `/settings?category=profiles` where users can view all profiles, edit target exam dates, reassign textbooks, or delete workspaces.
- **"Don't Ask on Startup"**: Pinned to the bottom-left corner of the viewport as an unobtrusive checkbox toggle, persisting user preference to `user_settings.ask_profile_on_startup` in SQLite.

---

## Architectural Changes & Key Files

### 1. Dedicated Route & Route Guard Interception
- **[`frontend/src/router/index.js`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/router/index.js)**:
  - Registered route `/select-profile` pointing to [`ProfileSelect.vue`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/pages/ProfileSelect.vue).
  - Enhanced `router.beforeEach`:
    - Checks whether `studyloop_profile_picker_shown` has been recorded in `sessionStorage`.
    - Evaluates `ask_profile_on_startup !== false` and checks profile count.
    - If multiple profiles exist and selection is enabled, intercepts startup and routes immediately to `/select-profile`.

### 2. Viewport-Pinned Bottom Bar & Chrome Layout
- **[`frontend/src/pages/ProfileSelect.vue`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/pages/ProfileSelect.vue)**:
  - Center stage: Profile cards grid with circular avatars, pace chips (`d ahead`, `d behind`, `On track`), and target exam deadlines.
  - Pinned footer: Fixed to viewport corners (`.footer-corner-left` for startup preference, `.footer-corner-right` for profile management and optional cancel).
  - Responsive fallback: Transitions cleanly to a stacked flow on mobile screens (`<= 640px`).

### 3. Direct Modal Triggering via URL Query
- **[`frontend/src/pages/Settings.vue`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/pages/Settings.vue)**:
  - Added support for `route.query.action === 'add' || route.query.add === 'true'`, automatically opening `SettingsProfileModal.vue` when arriving from the "Add Profile" card.

### 4. App Shell & Test Suite
- **[`frontend/src/App.vue`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/App.vue)**:
  - Excluded the navigation sidebar from rendering on `/select-profile` (`route.path !== '/select-profile'`), giving the page a full-bleed, focused startup feel.
- **[`frontend/src/pages/ProfileSelect.spec.js`](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/pages/ProfileSelect.spec.js)**:
  - Comprehensive unit test suite covering profile rendering, number key shortcuts, active profile selection, and routing to both Manage Profiles and Add Profile modal actions.
