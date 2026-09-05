# StudyLoop Performance & Benchmark Report

This document records the empirical benchmark results across StudyLoop's core performance-critical pipelines: **Go Native PDF Text Extraction**, **Deep Structured PDF Ingestion**, **SQLite Bulk Database Operations**, **Hybrid Vector Retrieval Latency**, and **RAG Embedding Quality & Sizing**.

---

## 1. Summary of Pipeline Winners

| Performance Pipeline | Evaluated Methods | 🏆 Winner / Best Strategy | Key Performance Characteristic |
| :--- | :--- | :--- | :--- |
| **Go Native PDF Extraction** | Sequential, Stream Reader, Parallel Shared, Independent Handles (2/4/16w), In-Memory Preload (4/16/32w), Adaptive Chunks | **In-Memory Preload + 2x CPU Workers / Parallel Handles** | **`~2.87s` for 1,000 pages (~347.6 p/s, 8.6x–25.6x speedup)** |
| **Deep PDF Ingestion (PyMuPDF)** | Sequential, ThreadPools (2/4/8/16/32w), 1-Page/Thread, Adaptive Degradation, ProcessPool | **Adaptive Degradation (`max(2, CPU)` workers)** | **`7.45s` (1.3 p/s, 5.62x speedup)**; eliminates straggler pages |
| **SQLite Bulk Writes (5,000 rows)** | Auto-Commit, Single Tx (DELETE mode), Single Tx (WAL mode), Chunked Tx (500), Multi-Row VALUES (250) | **Single Tx + Prepared Statements (WAL mode)** | **`45ms – 57ms` (88,000–111,000 rows/s, >500x speedup)** |
| **Search / RAG Query Latency** | Table Scan (`LIKE`), Inverted Index, Brute-Force Vector Cosine (384d), Hybrid RRF | **Hybrid RRF (Lexical + Vector + RRF)** | **`5.33ms / query` (187.4 q/s)** with zero server/Docker overhead |
| **RAG Retrieval Accuracy (Hit@k)** | Word Sweeps: 100, 200, 400, 600, 800, 1000, 1200 words | **500 words target (bounds [350, 650])** | **Optimal balance of Hit@1 accuracy, cosine margin, and token budget** |

---

## 2. Go-Native PDF Text Extraction Concurrency

The optimal Go concurrency strategy shifts depending on **document size / page count**:

* **Small Documents / Syllabus (< 50 pages)**: **Parallel Independent Handles (4 workers)** wins (`46ms`, ~654 pages/sec) because in-memory whole-file byte buffer copying has minor setup overhead.
* **Large Textbooks (300 to 1,000+ pages)**: **In-Memory Preload + 2x CPU Workers (32 goroutines)** wins (`2.877s` for 1,000 pages, **347.6 pages/sec**, 8.6x–25.6x speedup) because single-pass in-memory buffering completely eliminates repeated disk seek/lock contention across thousands of pages.

### 📊 Results Matrix (Large 1,000-Page Textbook vs. 30-Page Section)

| Strategy / Concurrency Mode | 1,000-Page Time | 1,000-Page Speed | 30-Page Time | 30-Page Speed | Notes |
| :--- | :---: | :---: | :---: | :---: | :--- |
| **In-Memory Preload + 2x CPU Workers** | **`2.877s`** | **`347.6 p/s`** 🏆 | `96ms` | `311.8 p/s` | **🏆 Best for Large Books (Work-stealing)** |
| **Parallel Independent Handles (4 Workers)** | `5.999s` | `166.7 p/s` | **`46ms`** | **`654.5 p/s`** 🏆 | **🏆 Best for Small Docs / Syllabus** |
| **In-Memory Preload + 1x CPU Workers** | `3.271s` | `305.7 p/s` | `77ms` | `391.1 p/s` | CPU hardware core alignment |
| **Dynamic Adaptive Decaying Chunks** | `3.370s` | `296.7 p/s` | `65ms` | `463.4 p/s` | Chunk-decay tail optimization |
| **Parallel File Handles (16 Workers)** | `4.768s` | `209.7 p/s` | `48ms` | `619.5 p/s` | OS file descriptor overhead |
| **Sequential Single Reader** | `24.76s – 1m23s` | `12.0 – 40.4 p/s` | `92ms` | `324.5 p/s` | Baseline reference |
| **Sequential Stream Reader** | `37.95s` | `26.3 p/s` | `1.234s` | `24.3 p/s` | Full document stream overhead |

### ⚡ Architectural Decision & Dynamic Worker Scaling
* **Implemented in `internal/notebook/upload.go`**: Dynamic worker scaling `optimalPDFWorkers(pageCount)`:
  ```go
  func optimalPDFWorkers(pageCount int) int {
      if pageCount <= 1 {
          return 1
      }
      maxWorkers := runtime.NumCPU() * 2
      if maxWorkers < 2 {
          maxWorkers = 2
      }
      if pageCount < maxWorkers {
          return pageCount // Automatically adapts: spawns fewer workers for small books
      }
      return maxWorkers
  }
  ```
* Caps workers to `pageCount` for short documents (preventing worker spawn overhead) while saturating `2x CPU` goroutines with in-memory preloading for 1,000+ page books.


---

## 3. Deep Structured PDF Ingestion Concurrency (PyMuPDF4LLM)

Evaluated on `learning go.pdf` across 10 pages on a 16-core CPU machine comparing worker and chunking strategies:

### 📊 Results Matrix

| Strategy / Concurrency Mode | Total Time | Throughput | Speedup | Notes |
| :--- | :---: | :---: | :---: | :--- |
| **Adaptive Degradation (16 workers)** | **`7.446s`** | **1.3 p/s** | **5.62x** | 🏆 **Fastest overall (Eliminates tail stragglers)** |
| **CPU Count ThreadPool (16 workers, even split)** | **`8.004s`** | **1.2 p/s** | **5.23x** | Near-optimal batching |
| **1 Page / Thread (Pool = 16 workers)** | **`8.393s`** | **1.2 p/s** | **4.99x** | Consistent, zero tail latency |
| **1 Page / Thread (Pool = 10 workers)** | **`9.428s`** | **1.1 p/s** | **4.44x** | 1 worker per active page |
| **Overcommit ThreadPool (32 workers)** | **`13.785s`** | **0.7 p/s** | **3.04x** | Thread switching overhead |
| **Fixed ThreadPool (8 workers)** | **`21.936s`** | **0.5 p/s** | **1.91x** | Sub-optimal core utilization |
| **Fixed ThreadPool (2 workers)** | **`24.565s`** | **0.4 p/s** | **1.70x** | Baseline multi-worker |
| **Sequential Baseline (1-pass)** | **`41.865s`** | **0.2 p/s** | **1.00x** | Baseline reference |
| **ProcessPool (16 processes)** | **`53.969s`** | **0.2 p/s** | **0.78x** | ⚠️ Process spawn overhead on Windows |

### ⚡ Architectural Decision & Solution
* **Implemented in `extensions/deep_pdf/ingest.py`**: Adaptive Degradation with `max(2, os.cpu_count())` workers.
* **Non-Blocking Background Flow & Zero-Copy Path** (`doc/solutions/2026-08-25-non-blocking-deep-pdf-extraction-and-ingestion-reuse.md`, `2026-08-28-deep-structured-pdf-zero-copy-and-freeze-fix.md`): Native OS file dialog (`SelectAndUploadDeepStructuredPDF`), zero JavaScript heap serialization, background execution with `ingestion-progress` events, and explicit `extraction_engine: 'deep_structured'` tracking.

---

## 4. Go Native Extractor vs. Deep Structured Markdown Parser

| Feature / Metric | Go Native (`ledongthuc/pdf`) | Deep PDF Engine (`PyMuPDF4LLM`) |
| :--- | :---: | :---: |
| **Execution Time (10 pages)** | **`~0.046s – 1.35s` (5.5x–20x faster)** | **`7.45s`** |
| **Runtime Dependency** | **Zero** (Compiled into Go `.exe`) | Python Extension Sidecar (`uv`-managed) |
| **Output Format** | Plain unformatted text | Semantic Markdown (`#`, `##`, `\| Table \|`) |
| **Table Extraction** | ❌ Interleaved / Broken | ✅ Structured Markdown Tables |
| **Code Block Detection** | ❌ Flat unindented lines | ✅ Fenced Code Blocks (`` ```go ``) |
| **Target Use-case** | Fast / Lightweight Syllabus import | High-precision Deep Study, Flashcards & RAG |

---

## 5. SQLite Bulk Writes & Transaction Strategy (5,000 Chunks)

Evaluated writing 5,000 chunk vector records into SQLite across different transaction modes:

### 📊 Results Matrix

| Strategy / Transaction Mode | Total Time | Write Speed | Speedup |
| :--- | :---: | :---: | :---: |
| **Auto-Commit (1 statement / transaction)** | `25.258s` | 198.0 rows/s | 1.00x |
| **Multi-Row `VALUES` Batches (`batch=250`)** | `87ms` | 57,460 rows/s | 290.3x |
| **Chunked Transactions (`batch=500, WAL`)** | `57ms` | 88,438 rows/s | 446.8x |
| **Single Transaction + Prepared Stmt (DELETE mode)** | `55ms` | 91,145 rows/s | 460.4x |
| **Single Transaction + Prepared Stmt (WAL mode)** | **`45ms`** | **111,435 rows/s** | 🏆 **562.9x faster** |

### ⚡ Architectural Decision
* `internal/db/vector_repo.go` routes bulk chunk ingestion through `r.withTx` + prepared statements in WAL mode (`UpsertChunkVectorsBatch`), achieving **sub-50ms bulk writes for entire textbooks**.

---

## 6. Hybrid Search & RAG Query Latency (5,000 Chunks, 100 Queries)

Evaluated retrieval speed across 5,000 vector chunks (384/768-dimensional dense embeddings + lexical indexing) in Go:

### 📊 Results Matrix

| Search Strategy | Time (100 Queries) | Avg Latency / Query | Throughput |
| :--- | :---: | :---: | :---: |
| **In-Memory Inverted Index (Lexical Match)** | `< 1ms` | `< 0.001 ms` | 100,000+ q/s |
| **SQLite `LIKE %term%` Scan** | `7ms` | `0.071 ms` | 14,045 q/s |
| **Hybrid RRF (Lexical + Vector + Reciprocal Rank Fusion)** | **`534ms`** | **`5.336 ms`** | **187.4 q/s** |
| **Brute-Force Vector Cosine Scan (384d)** | `545ms` | `5.448 ms` | 183.6 q/s |

### ⚡ Architectural Decision
* **Hybrid RRF** combines deep semantic search with exact keyword precision at **~5.3ms per query** with zero external vector database servers or Docker containers required.

---

## 7. RAG Chunk Sizing & Retrieval Accuracy Metrics

Evaluated across chunk size variations (`100` to `1,200` words) using `asset/model_int8.onnx` (768-d Nomic INT8):

### 📊 Metrics Matrix

| Chunk Target Size | Granularity / Chunk Count | Hit@1 / Hit@3 Precision | Cosine Margin ($\Delta$) | Token & Session Fit | Tradeoff Assessment |
| :--- | :---: | :---: | :---: | :---: | :--- |
| **100 – 200 words** | High chunk count | High local Hit@1 | High noise / fragmented context | Requires multi-chunk assembly | Code / tables get split |
| **400 – 600 words (Target: 500w)** | **Optimal** | **High Hit@1 & Hit@3 (>85%)** | **Strong signal separation (+0.25Δ)** | **Fits single prompt context window** | 🏆 **Production Standard (`doc/RAG.md`)** |
| **800 – 1200 words** | Low chunk count | Lower Hit@1 precision | Lower discrimination margin | Fast ingestion, high token footprint | Multi-topic vector dilution |

### ⚡ Architectural Rules for RAG
1. **Target Chunk Size**: ~500 words (bounds: `[350, 650]` words).
2. **Markdown-Aware Splitting**: `SplitMarkdownIntoChunks` preserves `#`/`##` headings, tables (`|---|`), and code blocks (`` ``` ``) as indivisible semantic units.
3. **Session Window**: ~2,500 – 3,000 words per reading task session.
4. **Local Runtime**: ONNX INT8 + SQLite `vec0.dll` executing in-process with zero network dependency.

---

## 8. Local ONNX INT8 Embedding Batch Scaling (768-d Nomic)

The optimal embedding batch strategy also depends on the **total chunk count and size of the book**:

* **Small Articles / Syllabus (< 10 pages, < 16 chunks)**:
  - Mini-batching at **Batch = 4 to 8** runs immediately (`~10–20s total`) without memory spiking or thread coordination delay.
* **Full Textbooks (300 to 1,000+ pages, 300–800 chunks)**:
  - **Batch = 32 to 64 (historical benchmark)**: Demonstrated a **~9x speedup** (`333 words/sec` vs `37 words/sec` for unbatched) in unconstrained standalone benchmarks, though current production limits background indexing batches to 4–16 to avoid foreground thread contention.

### 📊 Results Matrix (Evaluated on 64 Chunks / 9,664 Words across 16 CPU Threads)

| Batch Size | Total Time (s) | Chunks / sec | Words / sec | Speedup | Dimensions | Best Fit by Book Size |
| :---: | :---: | :---: | :---: | :---: | :---: | :--- |
| **Batch = 64** | **`29.024s`** | **`2.2 c/s`** | **`333.0 w/s`** | 🏆 **`8.97x`** | 768-d | **🏆 Large Textbooks (300–1,000+ pages) [Historical Benchmark]** |
| **Batch = 32** | `34.620s` | `1.8 c/s` | `279.1 w/s` | `7.52x` | 768-d | **Medium Coursebooks (50–300 pages) [Historical Benchmark]** |
| **Batch = 16** | `45.540s` | `1.4 c/s` | `212.2 w/s` | `5.72x` | 768-d | Short Chapters / Booklets |
| **Batch = 8** | `97.948s` | `0.7 c/s` | `98.7 w/s` | `2.66x` | 768-d | Small Syllabus / Articles (<15 pages) |
| **Batch = 4** | `136.168s` | `0.5 c/s` | `71.0 w/s` | `1.91x` | 768-d | Micro-extracts / Single Page |
| **Batch = 1** | `260.439s` | `0.2 c/s` | `37.1 w/s` | `1.00x` | 768-d | Baseline single-item inference |

### ⚡ Architectural Decision & Dynamic Scaling
* **Background Worker Queue (`doc/solutions/21062026_background_rag.md`)**:
  - Embedding runs in the background `VectorIndexQueue` without blocking the desktop UI.
  - Chunks are dispatched in mini-batches dynamically bounded by `4–16` (`runtime.NumCPU() / 2` clamped), ensuring rapid feedback on small assignments while achieving near-instant insertion rates (>100k rows/s) into SQLite WAL via `UpsertChunkVectorsBatch`.



