# Solution: Deterministic Markdown Chunking & In-Card Deep PDF Upgrade

## Problem
1. **Fragile Chunking Heuristics**: Detection of Markdown sections previously relied on substring regex (`#`, `|---|`), which could falsely match C/C++ code (`#include`, `#define`) in plain text or standard PDF extractions.
2. **Missing In-App Upgrade Action**: Books originally parsed with standard Go (`pdfcpu`) had no simple in-app upgrade path to re-extract with PyMuPDF structured tables and code blocks.
3. **Extraction State & Crash Vulnerability**: If extraction was running and the user closed the window, notebooks could remain permanently stuck in `processing`.

---

## Architecture & Implementation

### 1. Deterministic Format Dispatch
- Added `IsMarkdown bool` to `ExtractedDocument` in `internal/notebook/upload.go`.
- Native `.md` files and PyMuPDF extractions explicitly set `IsMarkdown = true` $\rightarrow$ dispatched to `SplitMarkdownIntoChunks(..., 500)`.
- Standard Go `pdfcpu` extractions set `IsMarkdown = false` $\rightarrow$ dispatched to `SplitPageIntoChunks(..., 500)`.
- Removed dead `isMarkdownSection` and unused `regexp` dependencies from `internal/notebook/ingestion.go`.

### 2. Backend Re-Extraction Endpoint (`UpgradeNotebookToDeepPDF`)
- Exposed `UpgradeNotebookToDeepPDF(notebookID string)` in `internal/app/notebook_endpoints.go`.
- Runs PyMuPDF background extraction on the existing `notebook.file_path`.
- Automatically transitions notebook to `status = 'processing'` and `study_status = 'dormant'` to prevent assignment of outdated queue tasks during re-parsing.
- Generates structured chapter bounds and emits `ingestion-progress` events with status `draft_ready`.

### 3. Startup Crash Recovery (`ResetInterruptedNotebookStatuses`)
- Added `ResetInterruptedNotebookStatuses()` in `internal/db/notebooks_repo.go` and wired into `internal/runtime/boot.go`.
- On application bootstrap, any notebooks stuck in `processing` or `analyzing` due to an unexpected quit/crash are safely recovered:
  - If chunks exist: reverts to `chunked`.
  - If un-chunked upload: reverts to `uploaded`.

### 4. Frontend UI State & In-Card Action
- Added `[ ⚡ Deep Extract ]` action button in `NotebookCard.vue` (gated on `isPro`).
- Replaced button with `[ ⏳ Deep Extracting... ]` and rotating spinner while extraction is active.
- Flips directly to `[ ✨ Review Chapters ]` upon completion.
- Added friendly non-blocking warning in the action toast to keep StudyLoop open during background extraction.

---

## Verification
- `go test -short ./internal/...` passed with 0 errors across all internal packages.
- Frontend build `npm run build` passed with 0 errors.
