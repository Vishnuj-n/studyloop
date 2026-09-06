# Solution: Anki (.apkg / .colpkg) Import Extension & Standalone Flashcard Decks

## Overview
Added support for importing Anki flashcard decks (`.apkg`, `.colpkg`) into StudyLoop. Built as a lightweight **Free-tiered** Python extension (`extensions/anki_importer/`) running on Astral's `uv` engine, it handles modern zstd-compressed SQLite collections (`collection.anki21b`) and legacy schemas, normalizes Cloze deletions (`{{c1::...}}`), strips HTML markup into clean prompt/answer pairs, and seeds immediate FSRS review flashcards (`fsrs_cards`) via the Go backend.

---

## Key Invariants & Architectural Design

1. **No False Reading Tasks (Deterministic Queue Invariant)**:
   - Standalone Anki decks are saved with `page_count = 0` and topic status `completed`.
   - StudyLoop's queue scheduler (`EnsurePendingReadingTaskForNotebook` and `QueryNextReadingTopic`) strictly requires `end_page > 0` and `status != 'completed'` to create `READING` tasks.
   - Standalone Anki decks therefore **never generate empty reading tasks**; they only generate `FLASHCARD_REVIEW` tasks when cards are due under FSRS.
2. **Zero DOM Binary Serialization**:
   - Uses desktop-native file picker (`SelectAnkiFile` / `wailsruntime.OpenFileDialog`) passing only file paths across the bridge.
3. **Clean Separation of Concerns**:
   - Python unpacks zip/zstd, parses Anki schema variations, normalizes cloze into Q&A, and outputs clean JSON to STDOUT.
   - Go backend receives the JSON, validates records, initializes FSRS calibration states (`StateCode: 2`, `Reps: 0`, `due_at: now`), and inserts records via `repo.GetOrCreateFlashcardsForTopic(...)`.

---

## Changes Made

### 1. Python Extension (`extensions/anki_importer/`)
- **[extensions/anki_importer/manifest.json](../../extensions/anki_importer/manifest.json)**:
  ```json
  {
    "id": "anki_importer",
    "name": "Anki Deck Importer",
    "version": "0.1.0",
    "runtime": "python",
    "entrypoint": "ingest.py",
    "tier": "free",
    "category": "study",
    "download_size": "~1 MB",
    "setup_notice": "Lightweight Anki package parser (~1 MB). Installs in seconds.",
    "description": "Import flashcard decks (.apkg, .colpkg) from Anki into StudyLoop with full cloze and HTML normalization."
  }
  ```
- **[extensions/anki_importer/requirements.txt](../../extensions/anki_importer/requirements.txt)**: Minimal requirements `zstandard>=0.22.0` and `beautifulsoup4>=4.12.0`.
- **[extensions/anki_importer/ingest.py](../../extensions/anki_importer/ingest.py)**:
  - Supports `--test` / `--smoke-test` probe for environment verification and self-test.
  - Automatically unpacks archive, decompresses `collection.anki21b` via `zstandard` or reads `collection.anki2` / `.anki21`.
  - Normalizes `{{c1::...}}` Cloze deletions into prompt/answer pairs.
  - Strips HTML markup into clean text.
  - Emits normalized JSON schema `{"deck_name": "...", "cards": [...]}` to STDOUT.

### 2. Backend Go Layer
- **[internal/app/app_anki.go](../../internal/app/app_anki.go)**:
  - `SelectAnkiFile()`: Native desktop OS file dialog filtering for `.apkg` and `.colpkg`.
  - `ImportAnkiDeck(filePath, targetNotebookID, targetTopicID)`: Invokes the Python runner, unmarshals flashcards, seeds initial FSRS states (`StateCode: 2`, `Reps: 0`, `due_at: now`), and persists cards via `repo.GetOrCreateFlashcardsForTopic(...)`.
- **[internal/db/topics_repo.go](../../internal/db/topics_repo.go)**:
  - Added `EnsureTopicWithStatus(topicID, title, status)` to allow creating topics with custom initial statuses (`completed` for standalone flashcards).
- **[internal/app/app_anki_test.go](../../internal/app/app_anki_test.go)**:
  - Unit tests verifying standalone deck creation, `0` reading tasks generated, and extension discovery.

### 3. Frontend UI
- **[frontend/src/components/AnkiImportModal.vue](../../frontend/src/components/AnkiImportModal.vue)**: Clean popup modal allowing file selection and choice between "Create Standalone Notebook" or "Add to Existing Notebook".
- **[frontend/src/services/importerRegistry.js](../../frontend/src/services/importerRegistry.js)**: Registered `anki_importer` in `NOTEBOOK_IMPORTERS`.
- **[frontend/src/services/appApi.js](../../frontend/src/services/appApi.js)**: Exported `selectAnkiFile` and `importAnkiDeck`.
- **[frontend/src/components/NotebookUpload.vue](../../frontend/src/components/NotebookUpload.vue)**: Bound `AnkiImportModal` to the extension tray and emitted `upload-anki`.
- **[frontend/src/pages/Notebook.vue](../../frontend/src/pages/Notebook.vue)**: Wired `handleAnkiUpload` with progress tracking, notification toasts, and notebook list reload.

### 4. Documentation
- **[doc/ARCHITECTURE.md](../../doc/ARCHITECTURE.md)**: Documented `anki_importer` in Ingestion Engines (§1.4).
- **[doc/DATA_API.md](../../doc/DATA_API.md)**: Added `SelectAnkiFile` and `ImportAnkiDeck` API contracts.

---

## Verification

- **Smoke test probe**:
  ```bash
  extensions/anki_importer/.venv/Scripts/python.exe extensions/anki_importer/ingest.py --smoke-test
  # {"status": "ready", "message": "Anki Importer (zstandard 0.25.0, bs4 4.15.0) is ready."}
  ```
- **Go unit tests**:
  ```bash
  go test -v -run="TestImportAnkiDeck|TestExtensionDiscoveryAnki" ./internal/app/...
  # PASS
  ```
- **Whole-repo test suite**:
  ```bash
  go test -short ./internal/...
  # PASS (all internal packages)
  ```
