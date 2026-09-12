# Size-Based Log Rotation (5MB Cap)

## Problem & Motivation

The desktop app initializes three multi-file loggers (`app.log`, `queue.log`, and `rag_engine.log`) in `dev_data/logs/` (or the user app data directory). In previous implementations, files were opened with `os.O_APPEND` without any file size limits or truncation rules. 

Over extended periods of use (e.g. processing large PDFs, running multiple RAG vector searches, and processing thousands of queue items), these log files would grow indefinitely on disk, posing a disk bloat risk on user machines.

---

## Solution Architecture

Instead of complex timestamp parsing or multi-condition hybrid time/size cleanup rules, we implemented a simple, robust size-based rotation strategy with a 1-backup history (`.old`).

### Key Parameters:
- **Max Log File Size**: `5 MB` (`5 * 1024 * 1024` bytes)
- **Log Files Capped**: `queue.log`, `rag_engine.log`, `app.log`
- **Max Storage Bound**: $3 \text{ files} \times 2 \text{ (active + .old)} \times 5\text{MB} = 30\text{MB}$ maximum on disk forever.

---

## Implementation Details

### `internal/utils/logging.go`

1. **`rotateLogFile(logPath string, maxSize int64)` Helper**:
   - Uses `os.Stat(logPath)` to inspect file size.
   - If `size >= maxSize`:
     - Deletes any pre-existing `logPath + ".old"` file (required to avoid Windows `os.Rename` overwrite errors).
     - Renames `logPath` $\rightarrow$ `logPath + ".old"`.

2. **Integration in `InitMultiFileLogger`**:
   - Before opening file handles for `queue.log`, `rag_engine.log`, and `app.log`, `InitMultiFileLogger` calls `rotateLogFile` on each path.

```go
const maxLogSizeBytes int64 = 5 * 1024 * 1024 // 5 MB per log file

func rotateLogFile(logPath string, maxSize int64) {
	info, err := os.Stat(logPath)
	if err != nil || info.Size() < maxSize {
		return
	}
	oldPath := logPath + ".old"
	_ = os.Remove(oldPath)
	_ = os.Rename(logPath, oldPath)
}
```

---

## Verification & Testing

### `internal/utils/logging_test.go`

Added unit tests:
- `TestRotateLogFile`: Verifies that missing files are skipped, files under 5MB are untouched, and files exceeding 5MB are safely rotated to `.old`.
- `TestInitMultiFileLoggerRotation`: Verifies end-to-end logger initialization and automatic rotation when starting `InitMultiFileLogger` with pre-existing large log files.

### Automated Tests
- Run `go test -v ./internal/utils` $\rightarrow$ `PASS`
- Run `go test -short ./internal/...` $\rightarrow$ `PASS`
