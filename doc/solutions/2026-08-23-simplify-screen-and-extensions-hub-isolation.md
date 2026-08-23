# Solution: Dedicated Simplify Page, YouTube Sub-Component & Extensions Hub Isolation

## Overview
Resolved architectural coupling and feature discovery issues across the Extensions Hub and Reader view:
1. **AI Audio Overview** was properly placed and registered as a **Pro Extension** in `extensions/audio_overview/manifest.json`.
2. Removed broken placeholder `extensions/example/` and fixed the extension process runner in `internal/app/app_extension.go`.
3. Created a standalone **Simplify Page** (`frontend/src/pages/Simplify.vue` at `/simplify`) as a 100% free native Markdown reading breakdown tool with a direct `← Back to Reader` button.
4. Extracted YouTube video playback and chapter transcript drawer out of `Reader.vue` into an isolated sub-component: `frontend/src/components/YouTubeReader.vue`.

---

## Changes Made

### 1. Dedicated Simplify Screen (`/simplify`)
- **[frontend/src/pages/Simplify.vue](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/pages/Simplify.vue)**:
  - Standalone Vue page rendering the simplified output using `<MarkdownReader>` (with KaTeX math, tables, lists, and code block support).
  - Includes a `← Back to Reader` button to return to the active task, a `📋 Copy Markdown` action, and session caching.
  - 100% free for all users.
- **[frontend/src/router/index.js](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/router/index.js)**:
  - Registered route `/simplify`.
- **[frontend/src/pages/Reader.vue](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/pages/Reader.vue)**:
  - Simplified the `✨ Simplify` button in the toolbar to extract session content (matching the logic of `copySessionContent`) and navigate to `/simplify`.
  - Removed all transient in-page modal state logic.

### 2. Isolated YouTube Player Component
- **[frontend/src/components/YouTubeReader.vue](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/components/YouTubeReader.vue)**:
  - Self-contained component managing the YouTube iframe embed, aspect ratio wrapper, and transcript drawer.
- **[frontend/src/pages/Reader.vue](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/pages/Reader.vue)**:
  - Replaced inline iframe and transcript DOM/CSS with `<YouTubeReader>` invocation. Keeps `Reader.vue` focused strictly on PDF & Markdown document reading.

### 3. Extensions Hub & Runner Fixes
- **[extensions/audio_overview/manifest.json](file:///c:/Users/vishn/PROJECT/ai-tutor/extensions/audio_overview/manifest.json)**:
  - Created manifest registering AI Audio Overview as a **Pro** extension.
- **[internal/extension/tiers.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/extension/tiers.go)**:
  - Authoritative tier mapping configured for `audio_overview` as `pro`.
- **[internal/app/app_extension.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/app/app_extension.go)**:
  - Fixed `RunExtension` to resolve the Python runtime interpreter executable path and pass the entrypoint script path and arguments (preventing `exec: no command` errors).
- **[extensions/example/](file:///c:/Users/vishn/PROJECT/ai-tutor/extensions/example/)**:
  - Deleted example extension directory.

---

## Verification Results

### Automated Tests
- **Frontend**: All 36 Vitest tests passing (`npm test`):
  ```text
  Test Files  11 passed (11)
  Tests       36 passed (36)
  ```
- **Backend**: All Go packages passing (`go test ./...`):
  ```text
  ok   ai-tutor/internal/app
  ok   ai-tutor/internal/db
  ok   ai-tutor/internal/extension
  ok   ai-tutor/internal/study
  ok   ai-tutor/internal/runtime
  ```
