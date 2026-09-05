# Background Vector Indexing Throttling and Foreground Prioritization

## Problem
When a notebook syllabus was confirmed, background ONNX vector indexing spawned embedding batches of up to 32 chunks across `NumCPU * 2` threads with tight loops and zero cooperative yields. 

When users subsequently performed interactive operations—such as multi-threaded Go PDF extraction or bookmark parsing—the ONNX SIMD/BLAS matrix calculations consumed all host CPU cycles. This starved the Go runtime scheduler, delayed Wails event dispatching, and caused the UI to appear frozen or stuck at intermediate milestones (e.g. 50% lecture/extraction).

## Solution

### 1. Foreground Extraction Speed Preserved
- `optimalPDFWorkers` in `internal/notebook/upload.go` remains unthrottled and multi-worker (`runtime.NumCPU() * 2`), ensuring users waiting on book extraction receive maximal parsing throughput.
- Bookmark parsing via `pdfcpu` remains instantaneous.

### 2. Deprioritized Background ONNX Indexer
In `internal/retrieval/indexer.go`:
- **Modest Batch Sizing**: Changed `optimalBatchSize()` from `[8, 32]` to `runtime.NumCPU() / 2`, clamped strictly between 4 and 16. This prevents long, uninterrupted CPU-heavy embedding bursts.
- **Cooperative Scheduler Yield**: Injected `time.Sleep(10 * time.Millisecond)` and `runtime.Gosched()` after every embedding batch in `IndexChunks`. This gives the Go runtime, operating system, and foreground extraction goroutines regular time slices to execute without lag.

## Key Files
- `internal/retrieval/indexer.go`: Throttled `optimalBatchSize()` and added 10ms cooperative pause + `runtime.Gosched()`.
- `internal/notebook/upload.go`: Maintained fast multi-worker PDF extraction.

## Verification
- Compilation verified: `go test -run=^$ ./internal/...`
- Unit tests verified: `go test -short ./internal/retrieval/...` (1.492s)
