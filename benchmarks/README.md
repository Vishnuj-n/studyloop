# StudyLoop Performance & Benchmarking Suite

This directory contains standalone benchmark harnesses to profile, measure, and optimize performance-critical paths across StudyLoop without modifying production application code.

---

## 📁 Directory Structure

```text
benchmarks/
├── README.md                          # Suite documentation & usage instructions
├── pdf_ingest/                        # PDF ingestion concurrency tests (PyMuPDF4LLM)
│   └── benchmark_ingest.py
├── pdf_text_extraction/              # Go-native PDF text extraction concurrency strategies
│   └── benchmark_go_pdf.go
├── embeddings/                        # Nomic INT8 ONNX & SentenceTransformer benchmarks
│   ├── benchmark_embeddings.py        # Inference engine & batch sizing benchmark (SQLite / Synthetic)
│   ├── benchmark_chunk_sweep.py       # Chunk word sweep (100 to 1,200 words) throughput profiling
│   └── benchmark_retrieval_quality.py # Scientific RAG Hit@1, Hit@3, MRR & Cosine margin evaluation
├── db_bulk_writes/                    # SQLite transaction & insert batching
│   └── benchmark_sqlite_writes.go
└── retrieval/                         # Lexical, Vector, and Hybrid RRF search latency
    └── benchmark_hybrid_search.go
```

---

## 🚀 How to Run the Benchmarks

### 1. Go Native PDF Text Extraction Strategies
Evaluates 10 Go concurrency models (Sequential Page-by-Page, Stream Readers, Shared Readers, Independent File Handles, In-Memory Preload + Parallel Readers, and Dynamic/Adaptive Decaying Chunks) using Go's native PDF reader.

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

### 3. Embedding Inference & Batch Sizing Benchmark
Profiles text embedding inference using the local **INT8 quantized ONNX Nomic model (`asset/model_int8.onnx`, 768 dimensions)** or HuggingFace SentenceTransformers across batch sizes (`1, 4, 8, 16, 32, 64, 128`). Supports loading real chunks from `dev_data/Studyloop.db`.

```bash
# Run ONNX INT8 on real database chunks from dev_data/Studyloop.db
uv run --with onnxruntime --with tokenizers --with numpy python benchmarks/embeddings/benchmark_embeddings.py --engine onnx --use-db --chunks 64

# Run ONNX INT8 on synthetic corpus with custom word density
uv run --with onnxruntime --with tokenizers --with numpy python benchmarks/embeddings/benchmark_embeddings.py --engine onnx --words-per-chunk 200 --chunks 64

# Run HuggingFace SentenceTransformer Nomic v1.5 model
uv run --with sentence-transformers --with einops python benchmarks/embeddings/benchmark_embeddings.py --engine st --model nomic-ai/nomic-embed-text-v1.5 --chunks 64
```

---

### 4. Chunk Size Grid Sweep (100 to 1,200 Words)
Profiles how chunk sizes across `[100, 200, 400, 600, 800, 1000, 1200]` words affect token counts, embedding throughput (`words/sec`, `chunks/sec`), and projected full-textbook ingestion time on CPU.

```bash
uv run --with pypdf --with onnxruntime --with tokenizers --with numpy python benchmarks/embeddings/benchmark_chunk_sweep.py --pdf "learning go.pdf" --pages 30
```

---

### 5. Empirical RAG Retrieval Quality & Accuracy Benchmark
Scientifically evaluates search accuracy across chunk sizes (`100` to `1,200` words) by extracting textbook chapters, dynamically synthesizing ground-truth probe questions, and measuring:
- **Hit@1 (%)**: Does the #1 ranked chunk contain the ground-truth answer?
- **Hit@3 (%)**: Is the answer within the top-3 retrieved chunks?
- **MRR**: Mean Reciprocal Rank across queries.
- **Cosine Separation Margin ($\Delta$)**: Score difference between target chunk and nearest distractor chunk.

```bash
uv run --with pypdf --with onnxruntime --with tokenizers --with numpy python benchmarks/embeddings/benchmark_retrieval_quality.py --pdf "learning go.pdf" --start-page 30 --end-page 90
```

---

### 6. SQLite Bulk Insert & Transaction Strategies
Compares Auto-commit vs. Prepared Statements vs. Single Transactions vs. Chunked Transactions vs. WAL mode on thousands of chunk records.

```bash
go run benchmarks/db_bulk_writes/benchmark_sqlite_writes.go
```

---

### 7. Hybrid Retrieval Latency (Lexical vs. Vector vs. RRF)
Measures query latency across 5,000 document chunks comparing Table Scans, Inverted Indexes, 384/768-dimensional Cosine similarity, and Reciprocal Rank Fusion (RRF).

```bash
go run benchmarks/retrieval/benchmark_hybrid_search.go
```
