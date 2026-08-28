# Solution: Deep Structured PDF Extraction Zero-Copy Path & Freeze Elimination

**Date:** 2026-08-28  
**Module:** Backend Desktop App Ingestion (`notebook_endpoints.go`, `deep_pdf.go`, `upload.go`), Extension (`extensions/deep_pdf`), Frontend UI (`Notebook.vue`, `NotebookUpload.vue`, `App.vue`, `appApi.js`)

---

## 1. Problem & Root Cause

1. **Chromium V8 Heap Exhaustion & Main Thread Freeze**:
   - In `Notebook.vue`, reading large multi-megabyte PDFs (e.g. 50MB) via `file.arrayBuffer()` and converting to a plain JavaScript number array with `Array.from(new Uint8Array(arrayBuffer))` created tens of millions of boxed JS number objects.
   - Serializing this huge array across Wails IPC (`JSON.stringify([37, 80, ...])`) locked the Chromium UI message loop, prompting Windows to mark the application window as **`Studyloop (Not Responding)`** at 5% progress.
2. **Confusing & Inconsistent Naming**:
   - The deep structured parser was ambiguously named `fast_pdf` in multiple directories, files, and manifests, causing confusion with standard linear PDF ingestion.
3. **Scoped Progress Listener & Missed Completion Notifications**:
   - The `EventsOn('ingestion-progress')` listener was mounted only in `Notebook.vue`. If a user navigated to the Dashboard or Reader while a 500-page book was extracting in the background, the completion event was lost.

---

## 2. Solutions Implemented

### A. Zero-Copy Native OS File Dialog (`internal/app/notebook_endpoints.go`)
- Completely removed the legacy `UploadFastPDFNotebook(fileData []byte, ...)` buffer handler.
- Added `SelectAndUploadDeepStructuredPDF(isPro bool)`:
  - Invokes native Wails `wailsruntime.OpenFileDialog`.
  - Directly copies and reads the local file path on disk via `notebookService.SaveUploadedFileFromPath(selectedPath)` with **0 MB JavaScript heap overhead** and zero IPC serialization lag.

### B. End-to-End `deep_pdf` Naming Refactor
- Renamed extension directory: `extensions/fast_pdf` → `extensions/deep_pdf`.
- Renamed Go service files: `internal/notebook/fast_pdf.go` → `internal/notebook/deep_pdf.go` and `deep_pdf_test.go`.
- Renamed Go types and methods:
  - `FastPDFIngestResult` → `DeepPDFIngestResult`
  - `FastPDFProgress` → `DeepPDFProgress`
  - `IngestFastPDF` / `IngestFastPDFWithProgress` → `IngestDeepPDF` / `IngestDeepPDFWithProgress`
- Updated extension tier mappings in `internal/extension/tiers.go` to `"deep_pdf": "pro"`.
- Updated frontend references in `importerRegistry.js`, `useExtensions.js`, `NotebookUpload.vue`, `Notebook.vue`, and `Extensions.vue`.

### C. Modal Notification on Start & Global In-App Alert on Complete
- **On Start (`Notebook.vue`)**: Replaced the brief auto-dismiss toast with an explicit modal dialog (`alertDialog` from `useDialog.js`) explaining that multi-page books extract in the background and that users can continue studying.
- **On Complete (`App.vue`)**: Added a top-level global `EventsOn('ingestion-progress')` listener in `App.vue` that triggers the in-app notice (`showNotice`) on `status: 'draft_ready'` from any screen.

---

## 3. Verification

- **Database Integrity**: Verified in `dev_data/Studyloop.db` that the 393-page *Grokking Artificial Intelligence Algorithms* extracted 348 structured study chunks across all 6 chapters with clean headings.
- **Backend Tests**: `go test -short ./internal/...` passed 100%.
- **Frontend Tests**: `vitest run` passed 100% across all 12 test suites (42 tests).
