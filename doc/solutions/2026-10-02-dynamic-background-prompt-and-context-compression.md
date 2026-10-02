# Solution Architecture: Dynamic Background Prompt & Context Compression (LLMLingua-2)

## Overview
This solution implements non-blocking, dynamic context and prompt compression using **LLMLingua-2 (110M BERT base)** packaged as a Python sidecar extension (`extensions/prompt_compressor`). Compressed chunk text is cached in SQLite (`chunks.compressed_text`) and dynamically passed into downstream LLM workflows (Quiz generation, Flashcard generation, RAG retrieval, Socratic tutoring).

---

## Dynamic Triggering Policies
Compression is controlled dynamically via `user_settings.prompt_compression_mode`:

1. **`OVER_LIMIT` (Default)**: Compresses text dynamically when total chunk tokens exceed the safe input target threshold:
   $$\text{Threshold} = \min(\text{Session Word Target} \times 1.33, 0.70 \times \text{Active Model Max Input Tokens})$$
2. **`ALWAYS`**: Compresses all topic chunks asynchronously upon topic activation.
3. **`DISABLED`**: Bypasses sidecar compression entirely (`COALESCE(compressed_text, chunk_text)` falls back to raw chunk text).

---

## Architecture & Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. SQLite Storage & Schema                                  │
│    - chunks.compressed_text & compressed_token_count        │
│    - user_settings.prompt_compression_mode & rate           │
└──────────────────────────────┬──────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────┐
│ 2. Python Sidecar (`extensions/prompt_compressor`)          │
│    - Uses llmlingua-2-bert-base-multilingual-cased          │
│    - Standard single-process CPU execution                  │
│    - Registered in internal/extension/tiers.go ("free")    │
└──────────────────────────────┬──────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────┐
│ 3. Asynchronous Study Service                               │
│    - compression_service.go: CompressTopicChunksAsync()     │
│    - Queue transition trigger on active READING task        │
└──────────────────────────────┬──────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────┐
│ 4. Downstream Integration & UI                              │
│    - COALESCE(compressed_text, chunk_text) in prompt builds │
│    - UI token saving indicator / badge in Reader drawer     │
└──────────────────────────────┘
```

---

## Empirical Benchmarks

Empirical performance measured via `scripts/benchmark_prompt_compression.py`:

| Compression Rate | Token Savings (% Reduction) | Keyword / Term Retention | Per-Chunk Latency (CPU) |
| :--- | :--- | :--- | :--- |
| **80% Target (Default)** | **23.2% Token Savings** | **88.8%** term retention | **~750 ms** / chunk |
| **60% Target (Aggressive)** | **43.5% Token Savings** | **75.0%** term retention | **~700 ms** / chunk |

---

## Process & Pipeline Invariants

1. **Single-Process Execution**: The Python sidecar (`compress.py`) runs as a standard single-process helper via Go's stream runner (`runner.RunStreamWithInput`), matching all other Studyloop extensions without complex thread manipulation.
2. **Vector Index Precision**: Embeddings are generated 100% from raw `chunk_text`, keeping vector search rankings and retrieval accuracy completely untouched.
3. **SQLite WAL Concurrency**: SQLite operates in WAL mode with a 5000ms busy handler, gracefully serializing background single-row `UPDATE` operations without UI lockups.

---

## Core Component Reference

- **Database Migrations & Schema**: [`internal/db/schema.go`](../../internal/db/schema.go), [`doc/SCHEMA.md`](../SCHEMA.md)
- **Extension Sidecar**: [`extensions/prompt_compressor/compress.py`](../../extensions/prompt_compressor/compress.py), [`internal/extension/tiers.go`](../../internal/extension/tiers.go)
- **Benchmark Script**: [`scripts/benchmark_prompt_compression.py`](../../scripts/benchmark_prompt_compression.py)
- **Study Compression Service**: [`internal/study/compression_service.go`](../../internal/study/compression_service.go), [`internal/study/compression_test.go`](../../internal/study/compression_test.go)
- **Queue Transition Trigger**: [`internal/study/queue_transition.go`](../../internal/study/queue_transition.go)
