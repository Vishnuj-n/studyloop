# Solution: PDF Extraction Acceleration & Duplicate Upload Prevention

## Problem
1. **Redundant Process Execution in PDF Bookmark Extraction**: In `internal/notebook/syllabus.go`, PDF chapter extraction called `extractPDFCPUBookmarkDraft()` (which invoked `pdfcpu bookmarks export <path> <temp_json>`) and immediately followed it with an identical call to `runPDFCPUBookmarksExport()`. On Windows, launching two sequential `pdfcpu.exe` processes created noticeable disk I/O, process startup, and parser latency.
2. **Duplicate Uploads in SQLite & File System**: The backend generated a new random UUID for every upload and lacked a `file_hash` deduplication check in `internal/app/notebook_endpoints.go`. Uploading the same textbook multiple times created duplicate rows in the database and redundant PDF files in `dev_data/uploads/`, confusing users with duplicate entries showing different chapter splits and chunk counts.

---

## Architecture & Implementation

### 1. Single-Export Bookmark Parsing (`internal/notebook/syllabus.go` & `pdfcpu.go`)
- Replaced the double invocation in `DraftSyllabusChapters` with a single call:
  ```go
  if strings.EqualFold(strings.TrimSpace(fileType), "pdf") && strings.TrimSpace(filePath) != "" {
      if raw, err := runPDFCPUBookmarksExport(filePath, s.config.UploadDir); err == nil && len(raw) > 0 {
          rawBookmarkJSON = raw
          bookmarkLikeDraft = ParsePDFCPUBookmarkDraftFromJSON(raw, doc.PageCount)
      }
  }
  ```
- Reused the resulting JSON bytes directly for both level-1 chapter extraction and full LLM context prompt generation.
- Pruned the redundant wrapper `extractPDFCPUBookmarkDraft()` from `pdfcpu.go`.

### 2. File Hash Lookup & Interception (`internal/db/notebooks_repo.go` & `internal/app/notebook_endpoints.go`)
- Added `FindNotebookByFileHash(fileHash, profileID string) (*models.Notebook, error)` to query existing records in the active profile.
- In `finalizeNotebookUpload` and `finalizeDeepStructuredPDFUpload`:
  - Computes the SHA-256 hash upon upload.
  - Queries `repo.FindNotebookByFileHash(fileHash, profileID)`.
  - If a matching notebook is found:
    1. Immediately removes the uploaded temp file using `a.notebookService.DeleteFile(...)`.
    2. Returns `{ "duplicate": true, "existing_id": existingNb.ID, "id": existingNb.ID, "file_name": existingNb.Title, ... }`.
    3. Prevents any redundant `INSERT INTO notebooks` in SQLite.

### 3. Frontend Deduplication Handling (`frontend/src/pages/Notebook.vue`)
- In `uploadFile()`:
  - Detects `result?.duplicate`.
  - Dispatches a toast notification: `Document already uploaded as "<name>"`.
  - Automatically routes to `openSyllabusDraft(result.id, ...)` to review the existing book's syllabus without creating duplicate entries.
- In `handleDeepStructuredUpload()`:
  - Displays an informational alert modal stating that the book is already in the library.

---

## Verification
1. **Automated Unit Tests**:
   - Added `TestUploadNotebookDuplicateFileDetection` in `internal/app/notebook_upload_test.go` verifying that second uploads of the same content are flagged as duplicates and return the original notebook's ID.
   - Verified via `go test -v -run=TestUploadNotebookDuplicateFileDetection ./internal/app/...` (passed in 0.07s).
2. **Compilation & Integrity**:
   - `go test -run=^$ ./internal/...` passed with 0 compile or type errors.
