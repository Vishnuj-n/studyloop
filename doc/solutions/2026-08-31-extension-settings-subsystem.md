# Solution: Streamlined Extension Settings Subsystem

## Context & Motivation
As StudyLoop's extension ecosystem expanded (AI Audio Overview, Text Simplifier, YouTube Ingestion, and Deep Structured PDF Parser), users needed persistent control over extension parameters (e.g. TTS voice personas, speech playback rates, reading simplification tiers, and ingestion options).

Following StudyLoop's **Deterministic Backend / SQLite Single Source of Truth** and **Ponytail (Minimal Code, Maximum Value)** architecture rules, we implemented a lean, schema-flexible settings subsystem that avoids heavy multi-table migrations while providing desktop persistence across compiled `.exe` builds.

---

## Solution Architecture

### 1. Database & Persistence Layer (SQLite Single Source of Truth)
- Added `extension_settings TEXT DEFAULT '{}'` column to the `user_settings` table in `internal/db/schema.go`.
- Added migration entry in `schemaMigrations` ensuring automatic schema updates without database repair loops or manual migrations.
- Implemented `GetExtensionConfig() (string, error)` and `SaveExtensionConfig(configJSON string) error` on `*Repository` in `internal/db/store.go`.
- Added comprehensive unit tests in `internal/db/profile_test.go` (`TestExtensionConfig`) to verify default initialization, custom JSON updates, and persistence across database reconnects.

### 2. Backend Bridge APIs
- Exposed `GetExtensionConfig()` and `SaveExtensionConfig(configJSON string)` via `App` methods in `internal/app/app_extension.go` to the Wails runtime bridge.

### 3. Extension Subprocess & Python Integration
- Updated `extensions/audio_overview/edge_tts_stream.py` to parse `rate` multipliers (e.g. `1.25` / `"+25%"`) from incoming JSON payloads and pass `rate=rate_str` to `edge_tts.Communicate`.

### 4. Frontend UI & Settings Workspace Integration
- **`frontend/src/composables/useExtensions.js`**:
  - Implemented persistent reactive configuration loading from SQLite on startup.
  - Provided `getExtensionSetting(extensionId, key, fallback)` and `setExtensionSetting(extensionId, key, value)`.
- **`frontend/src/components/SettingsExtensions.vue`**:
  - Created a modular configuration component using the existing theme tokens (`--surface-container-low`, `--outline-variant`, `--on-surface`, `--muted-text`).
  - Implemented settings for:
    - **AI Audio Overview**: Voice persona selector (`Christopher`, `Jenny`, `Guy`, `Sonia`, `Aria`, `Eric`) and speech tempo (`0.85x`, `1.0x`, `1.25x`, `1.5x`, `1.75x`).
    - **Text Simplifier**: Target reading comprehension level (`ELI15 High School`, `ELI10 Middle School`, `Executive Bullet Summary`, `Academic Precision`).
    - **YouTube Ingestion**: Preferred subtitle language track (`en`, `es`, `fr`, `de`, `auto`) and video chapter auto-segmentation toggle.
    - **Deep Structured PDF Parser**: Table formatting format (Markdown Grid Tables vs Hierarchical Lists) and KaTeX formula/equation extraction toggle.
- **`frontend/src/pages/Settings.vue`**:
  - Added **Extensions & Tools** category to the left navigation rail.
  - Supported URL query deep-linking (`/settings?category=extensions`).
- **`frontend/src/pages/Extensions.vue`**:
  - Added discrete `⚙️` configure buttons on extension cards routing directly to the settings workspace.
- **`frontend/src/components/AudioOverviewBar.vue`**:
  - Automatically loads and initializes playback with the user's saved voice persona and speed.

---

## Modified & Created Files

- `internal/db/schema.go`
- `internal/db/store.go`
- `internal/db/profile_test.go`
- `internal/app/app_extension.go`
- `extensions/audio_overview/edge_tts_stream.py`
- `frontend/src/services/appApi.js`
- `frontend/src/composables/useExtensions.js`
- `frontend/src/components/SettingsExtensions.vue` [NEW]
- `frontend/src/pages/Settings.vue`
- `frontend/src/pages/Extensions.vue`
- `frontend/src/components/AudioOverviewBar.vue`
- `doc/solutions/2026-08-31-extension-settings-subsystem.md` [NEW]

---

## Verification

### Backend Automated Test Suite
```powershell
go test -v -run=TestExtensionConfig ./internal/db/...
go test ./internal/...
```
**Results:** All unit tests pass with code 0 (`TestExtensionConfig`, `internal/app`, `internal/db`, `internal/notebook`, `internal/runtime`, `internal/study`).
