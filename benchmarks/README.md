# StudyLoop Performance & Benchmarking Suite

This directory contains standalone benchmark harnesses to profile, measure, and optimize performance-critical paths across StudyLoop without modifying production application code.

---

## 📁 Directory Structure

```text
benchmarks/
├── README.md                          # Suite documentation & usage instructions
├── pdf_ingest/                        # PDF ingestion concurrency tests
│   └── benchmark_ingest.py
├── pdf_text_extraction/              # Go-native PDF text extraction strategies
│   └── benchmark_go_pdf.go
├── embeddings/                        # SentenceTransformer / ONNX embedding batch sizing
│   └── benchmark_embeddings.py
├── db_bulk_writes/                    # SQLite transaction & insert batching
│   └── benchmark_sqlite_writes.go
└── retrieval/                         # Lexical, Vector, and Hybrid RRF search latency
    └── benchmark_hybrid_search.go
```

---

## 🚀 How to Run the Benchmarks

### 1. Go Native PDF Text Extraction Strategies
Evaluates Go concurrency models (Sequential Page-by-Page, Stream Readers, Shared Readers, Independent File Handles, In-Memory Preload + Parallel Readers, and Dynamic/Adaptive Decaying Chunks) using Go's native PDF reader.

```bash
# Run on sample PDF
go run benchmarks/pdf_text_extraction/benchmark_go_pdf.go -pdf "learning go.pdf" -start-page 1 -end-page 30 -runs 2

# Run on custom upload
go run benchmarks/pdf_text_extraction/benchmark_go_pdf.go -pdf "dev_data/uploads/<filename>.pdf" -runs 3
```

---

### 2. PDF Ingestion Concurrency (Python / Fast PDF Extension)
Evaluates 10 concurrency models (Sequential, ThreadPools, 1-Page/Thread, Adaptive Degradation, ProcessPool) on real PDF documents using PyMuPDF4LLM.

```bash
# Run on sample PDF
uv run --with pymupdf4llm python benchmarks/pdf_ingest/benchmark_ingest.py "learning go.pdf" --start-page 1 --end-page 20

# Run on custom upload
uv run --with pymupdf4llm python benchmarks/pdf_ingest/benchmark_ingest.py "dev_data/uploads/<filename>.pdf"
```

---

### 3. Embedding Inference Batch Sizing
Tests embedding generation throughput across batch sizes (1, 4, 8, 16, 32, 64, 128) for 384-dimensional text vectors.

```bash
uv run --with sentence-transformers python benchmarks/embeddings/benchmark_embeddings.py --chunks 256
```

---

### 4. SQLite Bulk Insert & Transaction Strategies
Compares Auto-commit vs. Prepared Statements vs. Single Transactions vs. Chunked Transactions vs. WAL mode on thousands of chunk records.

```bash
go run benchmarks/db_bulk_writes/benchmark_sqlite_writes.go
```

---

### 5. Hybrid Retrieval Latency (Lexical vs. Vector vs. RRF)
Measures query latency across 5,000 document chunks comparing Table Scans, Inverted Indexes, 384-dimensional Cosine similarity, and Reciprocal Rank Fusion (RRF).

```bash
go run benchmarks/retrieval/benchmark_hybrid_search.go
```
