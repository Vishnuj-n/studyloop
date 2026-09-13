# LSM Architecture Inspiration & Incremental Ingestion Solution

## 1. Architectural Philosophy & Inspiration

### Core Principle
> **Don't make expensive, large-scale work happen synchronously every time new data arrives. Accumulate & process incrementally, treating heavy data outputs as immutable, append-only units.**

This principle stems directly from **Log-Structured Merge-tree (LSM)** storage engines (used in RocksDB, LevelDB, and Cassandra). While StudyLoop is a guided study desktop app rather than a database storage engine, the core bottleneck of document AI/RAG applications mirrors LSM workloads:

- **PDF extraction & chunking** is computationally intensive.
- **Vector embedding generation (ONNX / local LLMs)** is GPU/CPU heavy and latency prone.
- **Re-processing static, immutable content** on every small UI change or re-ingest is an architectural anti-pattern.

---

## 2. Theoretical Mapping: LSM Engine vs. StudyLoop RAG Pipeline

| LSM Database Concept | StudyLoop Document & Retrieval Mapping | Benefit in StudyLoop |
| :--- | :--- | :--- |
| **Memtable (In-Memory Buffer)** | Page-range sliding window extraction buffer (50-page blocks) | Keeps extraction state lightweight and non-blocking before SQLite persistence. |
| **SSTable (Immutable Sorted File)** | Content-addressed `chunks` (`chunk_hash` = SHA256 of text + model version) | Prevents rewriting or re-embedding immutable past text. |
| **Write-Ahead Log (WAL)** | Page savepoint transactions in SQLite (`pdf_extraction_progress`) | Allows resuming PDF processing from page $N$ if the application crashes at page $N+1$. |
| **Compaction** | Incremental RRF (Reciprocal Rank Fusion) hybrid index updates | Merges new chunk vectors without re-building the full vector index. |

---

## 3. Concrete Solution & Pipeline Design

```text
                  [ User Uploads Document (PDF / MD) ]
                                   │
                                   ▼
                   1. Incremental Page Batching
             ┌──────────────────────────────────────────┐
             │ Extract Pages in 50-Page Savepoints      │
             │ Write extracted markdown to SQLite       │
             └─────────────────────┬────────────────────┘
                                   │
                                   ▼
                    2. Content-Addressed Hashing
             ┌──────────────────────────────────────────┐
             │ Calculate hash:                          │
             │ SHA256(chunk_text + model_version)      │
             └─────────────────────┬────────────────────┘
                                   │
                                   ▼
                   3. Embedding Deduplication Check
                      ┌──────────────────────┐
                      │ Hash exists in DB?   │
                      └──────────┬───────────┘
                    YES ─────────┴───────── NO
                     │                       │
                     ▼                       ▼
            [Reuse Vector Ref]     [Execute ONNX Inference]
                     │                       │
                     └───────────┬───────────┘
                                 │
                                 ▼
                     4. Progressive Reader Access
             ┌──────────────────────────────────────────┐
             │ Pages 1–5 embedded synchronously → ACTIVE│
             │ Background task enqueued for Pages 6–500 │
             └──────────────────────────────────────────┘
```

---

## 4. Implementation Details in StudyLoop

### A. Schema Definition (`internal/db/schema.go` & `doc/SCHEMA.md`)
Added `chunk_hash` (`VARCHAR(64)`) to the `chunks` table:

```sql
CREATE TABLE IF NOT EXISTS chunks (
    id TEXT PRIMARY KEY,
    topic_id TEXT NOT NULL,
    chunk_text TEXT NOT NULL,
    chunk_hash TEXT DEFAULT '',
    page_num INTEGER DEFAULT 0,
    token_count INTEGER DEFAULT 0,
    importance_score REAL DEFAULT 0,
    weakness_score REAL DEFAULT 0,
    embedding_ref TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (topic_id) REFERENCES topics(id)
);
```

### B. Repository Ingestion (`internal/db/notebooks_repo.go`)
During chunk creation, hashes are computed dynamically if omitted:
```go
func insertChunkRow(exec sqlExecer, topicID string, chunk NotebookChunkInput) error {
    hashVal := chunk.ChunkHash
    if hashVal == "" && chunk.Text != "" {
        hashVal = utils.MD5Hex(chunk.Text)
    }
    _, err := exec.Exec(`
        INSERT INTO chunks (id, topic_id, chunk_text, chunk_hash, page_num, token_count, importance_score, weakness_score)
        VALUES (?, ?, ?, ?, ?, ?, 0, 0)
    `, chunk.ID, topicID, chunk.Text, hashVal, chunk.PageNum, chunk.TokenCount)
    return err
}
```

### C. Checksum Vector Indexer (`internal/retrieval/indexer.go`)
The `VectorIndexer` checks `chunk_hash` values prior to embedding:
- Unchanged text skips ONNX model execution (`skipped++`).
- Only modified or new hashes are passed to `EmbedBatch()`.

---

## 5. Architectural Benefits

1. **Fast PDF Readiness**: Users can begin reading instantly as soon as Page 1 is parsed, while remaining pages vectorize lazily in the background.
2. **Zero Waste Re-Indexing**: Re-ordering topics or re-parsing documents re-uses pre-computed vector references for identical text chunks.
3. **Resilient Ingestion**: Crash recovery resumes from the latest 50-page savepoint rather than re-extracting from page 1.
