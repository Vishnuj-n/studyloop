# StudyLoop Performance & Benchmarking Suite

This directory contains standalone benchmark harnesses to profile, measure, and optimize performance-critical paths across StudyLoop without modifying production application code.

---

## 📁 Directory Structure

```text
benchmarks/
├── README.md                          # Suite documentation & usage instructions
├── pdf_ingest/                        # PDF ingestion concurrency tests
│   └── benchmark_ingest.py
├── embeddings/                        # SentenceTransformer / ONNX embedding batch sizing
│   └── benchmark_embeddings.py
├── db_bulk_writes/                    # SQLite transaction & insert batching
│   └── benchmark_sqlite_writes.go
└── retrieval/                         # Lexical, Vector, and Hybrid RRF search latency
    └── benchmark_hybrid_search.go
```

---

## 🚀 How to Run the Benchmarks

### 1. PDF Ingestion Concurrency
Evaluates 10 concurrency models (Sequential, ThreadPools, 1-Page/Thread, Adaptive Degradation, ProcessPool) on real PDF documents.

```bash
# Run on sample PDF
uv run --with pymupdf4llm python benchmarks/pdf_ingest/benchmark_ingest.py "learning go.pdf" --start-page 1 --end-page 20

# Run on custom upload
uv run --with pymupdf4llm python benchmarks/pdf_ingest/benchmark_ingest.py "dev_data/uploads/<filename>.pdf"
```

---

### 2. Embedding Inference Batch Sizing
Tests embedding generation throughput across batch sizes (1, 4, 8, 16, 32, 64, 128) for 384-dimensional text vectors.

```bash
uv run --with sentence-transformers python benchmarks/embeddings/benchmark_embeddings.py --chunks 256
```

---

### 3. SQLite Bulk Insert & Transaction Strategies
Compares Auto-commit vs. Prepared Statements vs. Single Transactions vs. Chunked Transactions vs. WAL mode on thousands of chunk records.

```bash
go run benchmarks/db_bulk_writes/benchmark_sqlite_writes.go
```

---

### 4. Hybrid Retrieval Latency (Lexical vs. Vector vs. RRF)
Measures query latency across 5,000 document chunks comparing Table Scans, Inverted Indexes, 384-dimensional Cosine similarity, and Reciprocal Rank Fusion (RRF).

```bash
go run benchmarks/retrieval/benchmark_hybrid_search.go
```
