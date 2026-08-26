# StudyLoop Performance & Benchmark Report

This document records the empirical benchmark results across StudyLoop's core performance-critical pipelines: **PDF Ingestion**, **SQLite Database Operations**, and **Hybrid Vector Retrieval**.

---

## 1. PDF Ingestion Concurrency Benchmark

Evaluated on `learning go.pdf` across 10 pages on a **16-core CPU** machine comparing different worker and chunking strategies against `PyMuPDF4LLM`.

### 📊 Results Matrix

| Strategy / Concurrency Mode | Total Time | Throughput | Speedup | Notes |
| :--- | :---: | :---: | :---: | :--- |
| **Adaptive Degradation (16 workers)** | **`7.446s`** | **1.3 p/s** | **5.62x** | 🏆 **Fastest overall (Eliminates stragglers)** |
| **CPU Count ThreadPool (16 workers, even split)** | **`8.004s`** | **1.2 p/s** | **5.23x** | Near-optimal batching |
| **1 Page / Thread (Pool = 16 workers)** | **`8.393s`** | **1.2 p/s** | **4.99x** | Consistent, zero tail latency |
| **1 Page / Thread (Pool = 10 workers)** | **`9.428s`** | **1.1 p/s** | **4.44x** | 1 worker per active page |
| **Overcommit ThreadPool (32 workers)** | **`13.785s`** | **0.7 p/s** | **3.04x** | Thread switching overhead |
| **Fixed ThreadPool (8 workers)** | **`21.936s`** | **0.5 p/s** | **1.91x** | Sub-optimal core utilization |
| **Fixed ThreadPool (2 workers)** | **`24.565s`** | **0.4 p/s** | **1.70x** | Baseline multi-worker |
| **Sequential Baseline (1-pass)** | **`41.865s`** | **0.2 p/s** | **1.00x** | Baseline reference |
| **ProcessPool (16 processes)** | **`53.969s`** | **0.2 p/s** | **0.78x** | ⚠️ Process spawn overhead on Windows |

### ⚡ Architectural Decision
* Production `extensions/fast_pdf/ingest.py` has been updated to use **Adaptive Degradation with `max(2, os.cpu_count())` workers**.
* Large upfront chunks amortize Python $\leftrightarrow$ C++ call overhead, while decaying tail chunks prevent worker idle time caused by complex pages.

---

## 2. Default Go PDF Extractor vs. Fast PDF Deep Parser

| Feature / Metric | Go Native (`ledongthuc/pdf`) | Fast PDF Engine (`PyMuPDF4LLM`) |
| :--- | :---: | :---: |
| **Execution Time (10 pages)** | **`1.35s` (5.5x faster)** | **`7.45s`** |
| **Runtime Dependency** | **Zero** (Compiled into `.exe`) | Python Extension Sidecar |
| **Output Format** | Plain unformatted text | Semantic Markdown (`#`, `##`, `\| Table \|`) |
| **Table Extraction** | ❌ Interleaved / Broken | ✅ Structured Markdown Tables |
| **Code Block Detection** | ❌ Flat unindented lines | ✅ Fenced Code Blocks (`` ```go ``) |
| **Target Use-case** | Fast / Lightweight Syllabus import | High-precision Deep Study & RAG |

---

## 3. SQLite Bulk Write & Transaction Strategy (5,000 Chunks)

Evaluated writing 5,000 chunk vector records into SQLite across different transaction modes.

### 📊 Results Matrix

| Strategy / Transaction Mode | Total Time | Write Speed | Speedup |
| :--- | :---: | :---: | :---: |
| **Auto-Commit (1 statement / transaction)** | `29.088s` | 171.9 rows/s | 1.00x |
| **Multi-Row `VALUES` Batches (`batch=250`)** | `108ms` | 46,301 rows/s | 269.4x |
| **Chunked Transactions (`batch=500, WAL`)** | `80ms` | 62,575 rows/s | 364.0x |
| **Single Transaction + Prepared Stmt (DELETE mode)** | `59ms` | 84,865 rows/s | 493.7x |
| **Single Transaction + Prepared Stmt (WAL mode)** | **`57ms`** | **88,378 rows/s** | 🏆 **514.1x faster** |

### ⚡ Architectural Decision
* `internal/db/vector_repo.go` uses `r.withTx` + prepared statements in WAL mode (`UpsertChunkVectorsBatch`), achieving **sub-60ms bulk writes for entire textbooks**.

---

## 4. Hybrid Search & RAG Retrieval Latency (5,000 Chunks, 100 Queries)

Evaluated retrieval speed across 5,000 vector chunks (384-dimensional dense embeddings + lexical indexing) in Go.

### 📊 Results Matrix

| Search Strategy | Time (100 Queries) | Avg Latency / Query | Throughput |
| :--- | :---: | :---: | :---: |
| **In-Memory Inverted Index (Lexical Match)** | `< 1ms` | `< 0.001 ms` | 100,000+ q/s |
| **SQLite `LIKE %term%` Scan** | `6ms` | `0.063 ms` | 15,970 q/s |
| **Hybrid RRF (Lexical + Vector + Reciprocal Rank Fusion)** | **`534ms`** | **`5.337 ms`** | **187.4 q/s** |
| **Brute-Force Vector Cosine Scan (384d)** | `641ms` | `6.407 ms` | 156.1 q/s |

### ⚡ Architectural Decision
* Hybrid RRF provides deep semantic search combined with exact keyword precision at **~5.3ms per query** with zero external vector database servers or Docker containers required.

---

## 5. Summary of Key Guidelines for StudyLoop

1. **Keep PDF Ingestion Adaptive**: Use thread pools matching CPU cores with decaying chunk sizes.
2. **Never Use Multiprocessing on Windows**: Process pools add significant startup latency; thread pools are faster because MuPDF releases the GIL.
3. **Always Batch DB Writes**: Route chunk ingestion through `UpsertChunkVectorsBatch` inside single transactions.
4. **Embedded Vector RAG is Optimal**: In-process `onnx.dll` + `vec0.dll` + SQLite gives $<10\text{ms}$ search without external server bloat.
