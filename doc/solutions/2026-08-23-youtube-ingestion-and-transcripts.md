# Solution: YouTube Ingestion, Transcripts & Video Reader

## Overview
Implemented YouTube video lecture ingestion into StudyLoop using **`yt-dlp`** inside a **Free-tiered** extension (`extensions/youtube/`), feeding directly into StudyLoop's canonical `ExtractedDocument` pipeline, and providing an embedded video player with 1-click default browser quick-launch fallback (`OpenURLInBrowser`) and collapsible transcript drawer in `YouTubeReader.vue`.

---

## Changes Made

### 1. YouTube Free Extension (`extensions/youtube/`)
- **[extensions/youtube/manifest.json](file:///c:/Users/vishn/PROJECT/ai-tutor/extensions/youtube/manifest.json)**:
  - Extension manifest registering `id: "youtube"`, `tier: "free"`, `runtime: "python"`, and `entrypoint: "ingest.py"`.
  ```json
  {
    "id": "youtube",
    "name": "YouTube Ingestion & Transcripts",
    "version": "0.1.0",
    "runtime": "python",
    "entrypoint": "ingest.py",
    "tier": "free",
    "category": "ingestion",
    "description": "Ingest YouTube video lectures, extract timestamped transcripts with chapters, and study with embedded video player and quizzes."
  }
  ```
- **[extensions/youtube/requirements.txt](file:///c:/Users/vishn/PROJECT/ai-tutor/extensions/youtube/requirements.txt)**:
  - Minimal dependency on `yt-dlp>=2024.0.0`.
- **[extensions/youtube/ingest.py](file:///c:/Users/vishn/PROJECT/ai-tutor/extensions/youtube/ingest.py)**:
  - Uses `yt-dlp` to extract video metadata, duration, native chapters/timestamps, and captions.
  - Automatically attempts manual/creator subtitles first, falling back to auto-generated subtitles.
  - Slices video into chapters (or ~10-minute parts if no native chapters) and outputs clean UTF-8 JSON.

### 2. Backend Go Integration
- **[internal/extension/tiers.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/extension/tiers.go)**:
  - Configured `"youtube": "free"` in `officialExtensionTiers` so all users can ingest and study with YouTube lectures.
- **[internal/app/app_update.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/app/app_update.go)**:
  - Added `OpenURLInBrowser(urlStr)` bridge helper utilizing `wailsruntime.BrowserOpenURL` for reliable OS browser launching.
- **[internal/notebook/youtube.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/notebook/youtube.go)**:
  - `IngestYouTubeVideo`: Executes the YouTube extension and normalizes video chapters into standard `ExtractedDocument` and `ExtractedSection` models.
- **[internal/notebook/upload.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/notebook/upload.go)**:
  - Added `case "youtube":` handling to `ExtractDocumentRange` and `ExtractDocumentSample`.
- **[internal/app/notebook_endpoints.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/app/notebook_endpoints.go)**:
  - `UploadYouTubeNotebook(url, isPro)`: Ingests video, stores video metadata to disk, creates notebook record, and pre-populates chapter syllabus draft.
- **[internal/db/reader_bundle_repo.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/db/reader_bundle_repo.go)**:
  - For YouTube notebooks, inspects the active chapter's `start_seconds` and `end_seconds` from metadata JSON on disk.
  - Generates the authoritative embed URL with query parameters: `https://www.youtube-nocookie.com/embed/<video_id>?enablejsapi=1&start=<start>&end=<end>`.
  - Populates `bundle.RawContent` with the chapter's actual transcript text rather than the raw JSON string.

### 3. Frontend Video Reader & Ingestion UI
- **[frontend/src/services/appApi.js](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/services/appApi.js)**:
  - Exported `uploadYouTubeNotebook(url, isPro)` and `openURLInBrowser(url)`.
- **[frontend/src/services/importerRegistry.js](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/services/importerRegistry.js)**:
  - Registered YouTube lecture importer in the free community tray.
- **[frontend/src/components/NotebookUpload.vue](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/components/NotebookUpload.vue)**:
  - Added "OR IMPORT VIDEO" YouTube input box with direct paste and ingestion trigger.
- **[frontend/src/pages/Notebook.vue](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/pages/Notebook.vue)**:
  - Handles `@upload-youtube` event and automatically opens syllabus draft confirmation modal.
- **[frontend/src/composables/useReaderBase.js](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/composables/useReaderBase.js)**:
  - Added `isYouTube` flag and streamlined `youtubeEmbedUrl` computed helper that preserves the authoritative backend query parameters.
- **[frontend/src/components/YouTubeReader.vue](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/components/YouTubeReader.vue)**:
  - Self-contained timestamp parsing from `embedUrl` query parameters (`start`, `end`).
  - Added timecode & duration banner with **"↗ Open in Browser"** quick-launch button for 1-click fallback to the user's logged-in default browser (bypassing embed/sign-in blocks).
  - Added clear study session scope hints and collapsible transcript drawer.
- **[frontend/src/pages/Reader.vue](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/pages/Reader.vue)**:
  - Renders `YouTubeReader` seamlessly in task and browse modes.

---

## Verification Results

### Backend Unit & Integration Tests
```powershell
go test ./...
```
Output:
```text
ok  	ai-tutor/internal/app	14.851s
ok  	ai-tutor/internal/db	(cached)
ok  	ai-tutor/internal/embeddings	(cached)
ok  	ai-tutor/internal/extension	(cached)
ok  	ai-tutor/internal/llm	(cached)
ok  	ai-tutor/internal/notebook	(cached)
ok  	ai-tutor/internal/runtime	(cached)
ok  	ai-tutor/internal/scheduler	(cached)
ok  	ai-tutor/internal/study	(cached)
ok  	ai-tutor/internal/utils	(cached)
PASS
```

### Frontend Production Build
```powershell
cd frontend
npm run build
```
Output:
```text
vite v6.4.3 building for production...
✓ 417 modules transformed.
dist/index.html                           0.96 kB
dist/assets/index-LJui0vV0.js         4,196.29 kB
✓ built in 22.93s
```
