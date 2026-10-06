# Solution Architecture: Sequential Background Compression & Study Note Pipeline

## Overview
This document outlines the sequential execution lifecycle between **LLMLingua-2 context compression** and **automated study note generation** during active reading sessions in Studyloop. 

It details the resolution of the race condition between parallel background goroutines, the progressive batching of chunks to adhere to transformer token limits, and the fallback behaviors across edge cases.

---

## The Core Problem: Parallel Race Condition vs Sequential Dependency

### Previous Parallel Race Condition
When a reading session started (`InitializeReadingSession`), two asynchronous tasks were spawned concurrently:
1. `CompressTopicChunksAsync(task.TopicID)` — Ran the Python LLMLingua-2 sidecar in the background.
2. `GenerateTopicStudyNoteAsync(task.TopicID, ...)` — Read chunks from SQLite immediately to assemble the note prompt.

Because note generation ran in parallel without waiting for compression to write to SQLite:
- `GenerateTopicStudyNote` read chunks when `chunk.compressed_text` was still empty (`NULL`).
- It fell back to raw uncompressed text (`chunk.chunk_text`), defeating token budget optimizations.
- The LLM context window received 20–30% less textbook material than intended.

---

## Architectural Solution: Sequential Handshake Pipeline

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ 1. User Opens Reading Session (`InitializeReadingSession` in app_study_reading.go)      │
└───────────────────────────────────────────┬────────────────────────────────────────────┘
                                            │
                                            ▼
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ 2. `CompressTopicChunksAsync(ctx, topicID, onCompleteCallback)`                        │
│    - Evaluates `prompt_compression_mode` (OVER_LIMIT / ALWAYS / DISABLED)              │
│    - Batches uncompressed topic chunks in groups of 10 (`batchSize = 10`)              │
│    - Commits each batch to SQLite (`chunks.compressed_text`) immediately               │
│    - Keeps latency predictable and fits BERT's 512-token limit per chunk               │
└───────────────────────────────────────────┬────────────────────────────────────────────┘
                                            │
                                            ▼ (Executes strictly after all batches complete)
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ 3. `onCompleteCallback()`                                                              │
│    - Checks if `user_settings.auto_generate_study_notes == true`                       │
│    - Checks if note for (topic, startPage, endPage) already exists                     │
│    - Spawns `GenerateTopicStudyNoteAsync`                                              │
│    - Reads pre-compressed chunks from SQLite (`~20% smaller token footprint`)          │
│    - Generates crisp structured note and saves to `dev_data/notes/...`                 │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Batching & Transformer 512-Token Window Invariant

1. **Transformer Token Limit**: The underlying model (`llmlingua-2-bert-base-multilingual-cased-meetingbank`) has a physical architecture ceiling of 512 tokens (`max_position_embeddings = 512`). Large single payloads risk token truncation or index errors.
2. **Progressive Batch Processing**:
   - Chunks are dispatched in groups of 10.
   - Each batch writes to SQLite (`UpdateChunkCompressedText`) immediately upon return.
   - If user navigates away or background job is stopped, completed batches remain persisted in SQLite and are never recomputed.
3. **Decoupled Background Context**:
   - `CompressTopicChunksAsync` uses `context.WithTimeout(context.Background(), 10*time.Minute)` rather than the transient HTTP/RPC request context.

---

## Edge Cases & Non-Happy Paths

| Scenario / Edge Case | System Behavior | Safety & Invariant |
| :--- | :--- | :--- |
| **Compression Mode = `DISABLED`** | `CompressTopicChunks` evaluates mode and returns `nil` in 0ms. `onCompleteCallback()` fires immediately, generating notes from raw text. | **Safe & Graceful**: Zero blocking delay. |
| **Auto Note Generation = `false`** | `onCompleteCallback()` checks `AutoGenerateStudyNotes`, sees `false`, and exits without calling LLM. | **Safe & Graceful**: Respects user setting. |
| **User Manually Clicks "Generate Note" Before Compression Finishes** | `GenerateTopicStudyNoteForRange` reads currently available chunks; uses `compressed_text` for finished chunks and falls back to `chunk_text` for pending chunks. When background job finishes, it sees existing note and skips overwrite. | **Safe & Graceful**: No crash, no duplicate LLM calls, user manual action respected. |
| **Short Chunks (`< 25` words)** | Skipped from compression to avoid BERT overhead on headings/bullet headers; raw text used. | **Safe & Graceful**: No wasted computation. |
| **Python Sidecar / Model Unavailable** | Sidecar returns error; batch updates are skipped; note generator gracefully falls back to raw text. | **Fail-Safe**: App remains completely functional. |

---

## Component Reference

- **Reading Session RPC**: [`internal/app/app_study_reading.go`](../../internal/app/app_study_reading.go)
- **Compression Service**: [`internal/study/compression_service.go`](../../internal/study/compression_service.go)
- **Study Notes Service**: [`internal/study/study_notes_service.go`](../../internal/study/study_notes_service.go)
- **Sidecar Entrypoint**: [`extensions/prompt_compressor/compress.py`](../../extensions/prompt_compressor/compress.py)
- **Compression Popover UI**: [`frontend/src/components/CompressionBadge.vue`](../../frontend/src/components/CompressionBadge.vue)
