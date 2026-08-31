# Solution: Streamlined Extension Settings Subsystem

## Context & Motivation
As StudyLoop's extension ecosystem expanded, users needed persistent control over key extension parameters (Edge-TTS voice personas, speech playback rates, and reading simplification comprehension styles).

Following StudyLoop's **Deterministic Backend / SQLite Single Source of Truth** and **Ponytail (Minimal Code, Maximum Value, Zero Over-engineering)** rules, we implemented a lean, schema-flexible settings subsystem that avoids heavy multi-table schemas while ensuring 100% desktop persistence in `Studyloop.db` across `.exe` builds.

---

## Solution Architecture

### 1. Database & Persistence Layer (SQLite Single Source of Truth)
- Added `extension_settings TEXT DEFAULT '{}'` column to the `user_settings` table in `internal/db/schema.go`.
- Added automatic migration entry in `schemaMigrations` ensuring seamless schema updates without migration loops.
- Implemented [`GetExtensionConfig()`](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/db/store.go) and [`SaveExtensionConfig(configJSON string)`](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/db/store.go) in `internal/db/store.go`.
- Added unit tests in `internal/db/profile_test.go` (`TestExtensionConfig`) to verify default initialization, custom JSON updates, and persistence across reconnects.

### 2. Backend Bridge & Execution Wiring
- **Wails Bridge APIs**: Exposed `GetExtensionConfig()` and `SaveExtensionConfig(configJSON string)` in `internal/app/app_extension.go`.
- **AI Audio Overview**:
  - `extensions/audio_overview/edge_tts_stream.py` accepts `voice` and `rate` multipliers (e.g. `1.25` / `"+25%"`) in its stdin payload and passes `rate=rate_str` to `edge_tts.Communicate`.
  - `frontend/src/components/AudioOverviewBar.vue` automatically loads and initializes playback with the user's saved voice persona and speed.
- **Text Simplifier**:
  - `extensions/text_simplifier/prompt.md` includes dynamic placeholder `{{style}}`.
  - `internal/study/simplifier.go` formats prompt directives based on target comprehension level (`eli10`, `bullet`, `academic`, `eli15`).
  - `internal/app/app_extension.go` parses `text_simplifier.level` from SQLite settings and injects it into LLM simplification requests.

### 3. Frontend UI & Settings Workspace Integration
- **`frontend/src/composables/useExtensions.js`**:
  - Manages reactive extension config state loaded from SQLite on startup.
- **`frontend/src/components/SettingsExtensions.vue`**:
  - Compact, dedicated settings pane styled with the app's existing dark/light CSS variables (`--surface-container-low`, `--outline-variant`, `--on-surface`, `--muted-text`).
  - Exposes settings for:
    - **🎙️ AI Audio Overview**: Voice persona selector (`Christopher`, `Jenny`, `Guy`, `Sonia`, `Aria`, `Eric`) and speech tempo (`0.85x`, `1.0x`, `1.25x`, `1.5x`, `1.75x`).
    - **📝 Text Simplifier**: Target reading comprehension level (`ELI15 High School`, `ELI10 Middle School`, `Executive Bullet Summary`, `Academic Precision`).
- **`frontend/src/pages/Settings.vue`**:
  - Added **Extensions & Tools** to the left category rail with deep-linking support (`/settings?category=extensions`).
- **`frontend/src/pages/Extensions.vue`**:
  - Added `⚙️` configure buttons on configurable cards to route directly to Settings.

---

## Modified & Created Files

- `internal/db/schema.go`
- `internal/db/store.go`
- `internal/db/profile_test.go`
- `internal/app/app_extension.go`
- `internal/study/simplifier.go`
- `internal/study/simplifier_test.go` [NEW]
- `extensions/audio_overview/edge_tts_stream.py`
- `extensions/text_simplifier/prompt.md`
- `frontend/src/services/appApi.js`
- `frontend/src/composables/useExtensions.js`
- `frontend/src/components/SettingsExtensions.vue` [NEW]
- `frontend/src/pages/Settings.vue`
- `frontend/src/pages/Extensions.vue`
- `frontend/src/components/AudioOverviewBar.vue`
- `doc/solutions/2026-08-31-extension-settings-subsystem.md` [NEW]

---

## Verification

### Automated Test Suite
```powershell
go test -short ./internal/...
go test ./internal/...
```
**Results:** All tests pass with code 0 across all internal packages (`internal/app`, `internal/db`, `internal/embeddings`, `internal/extension`, `internal/llm`, `internal/notebook`, `internal/runtime`, `internal/scheduler`, `internal/study`, `internal/utils`).
