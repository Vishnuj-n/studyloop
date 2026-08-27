# Non-Blocking Deep PDF Extraction & Ingestion Architecture Reuse

**Date:** 2026-08-25  
**Module:** Backend Desktop App Ingestion (`notebook_endpoints.go`, `notebooks_repo.go`, `fast_pdf.go`), Frontend UI (`Notebook.vue`, `NotebookCard.vue`, `Dashboard.vue`)

---

## Problem & Architectural Goals

1. **Eliminate Main Thread & UI Blocking**: Deep structured PDF parsing (PyMuPDF4LLM) on multi-page books can take minutes. Synchronously awaiting extraction across the Wails bridge blocked the frontend Promise, trapping the user inside an open upload dialog with erratic progress polling.
2. **Leave-Anytime Asynchronous Flow**: Users should be acknowledged immediately (<50ms), allowing them to navigate to the Study Queue, review cards, or leave the app open without waiting on a modal.
3. **100% Reuse of Existing Ingestion Architecture**: The app already possessed a proven chapter-draft notification and review pipeline for downloaded teacher assignments (documented in `2026-08-08-ingestion-notification-status-banner.md`). Deep PDF extraction should plug directly into this mechanism rather than reinventing custom state stores or notification dialogs.
4. **Standard Linear PDF Isolation**: Standard lightweight Go linear PDF ingestion (`UploadNotebook`) was kept intact.

---

## Key Solutions Implemented

### 1. Asynchronous Fast PDF Upload Endpoint (`internal/app/notebook_endpoints.go`)
- In `UploadFastPDFNotebook` & `finalizeFastPDFUpload`:
  - Instantly writes file bytes to storage, computes SHA256, and inserts the notebook row with `status: "uploaded"`.
  - Immediately returns `{ "id": id, "file_name": title, "status": "uploaded" }` to the frontend (<50ms).
  - Launches PyMuPDF extraction in a background Go goroutine (`go func()`).
  - Upon background completion, updates the notebook page count, persists the draft syllabus JSON, and transitions the notebook to `status: "draft_ready"`, emitting an `ingestion-progress` event.

### 2. Database Helper (`internal/db/notebooks_repo.go`)
- Added `UpdateNotebookPageCount(notebookID string, count int) error` to safely persist the extracted document page count once the background Python worker completes.

### 3. Immediate UI Feedback & Ingestion Reuse (`Notebook.vue`, `NotebookCard.vue`, `Dashboard.vue`)
- **`Notebook.vue`**: When uploading via `options.engine === 'fast_pdf'`, the upload dialog closes immediately and displays an informative toast (*"PDF uploaded. Deep extraction running in background..."*).
- **`Dashboard.vue`**: Dynamically computes `pendingIngestionBannerTitle` (`"New Assignment — Ingestion Needed"` for cloud classroom profiles vs. `"New Book — Ingestion Needed"` for local uploads) and displays `StatusBanner` with the `"✨ Ingest Book"` button.
- **`NotebookCard.vue`**: Formats the badge label (`ingestionBadgeLabel`) and renders the `"✨ Ingest Book"` button to launch `openSyllabusDraft` directly whenever the user is ready.

---

## Verification

- **Backend Tests**: `go test ./internal/...` passed 100% across all packages.
- **Frontend Tests**: `npm test -- --run` passed cleanly across all 12 test suites (40 tests).
