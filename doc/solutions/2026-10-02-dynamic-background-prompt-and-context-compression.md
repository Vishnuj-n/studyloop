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
└─────────────────────────────────────────────────────────────┘
```

---

## Core Component Reference

- **Database Migrations & Schema**: [`internal/db/schema.go`](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/db/schema.go), [`doc/SCHEMA.md`](file:///c:/Users/vishn/PROJECT/ai-tutor/doc/SCHEMA.md)
- **Extension Sidecar**: `extensions/prompt_compressor/compress.py`, [`internal/extension/tiers.go`](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/extension/tiers.go)
- **Study Compression Service**: [`internal/study/compression_service.go`](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/study/compression_service.go), [`internal/study/compression_test.go`](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/study/compression_test.go)
- **Queue Transition Trigger**: [`internal/study/queue_transition.go`](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/study/queue_transition.go)
