# AI Tutor RAG Architecture

**Legacy note:** Older docs used `blocks`/`block_vectors`. Live schema uses `chunks` and RAG embedding store (sqlite-vec). See `doc/SCHEMA.md`.

---

## 1. Purpose

RAG powers contextual AI for the current topic only. Answers stay grounded in active material, no cross-topic drift, predictable latency and cost.

Retrieval from active `topic_id` only. Single-turn prompt, one stateless LLM request.

## 2. Scope

Used for contextual explanation in Reader and Flashcards flows. This is a guided tutor, not a chatbot.

**Allowed:** clarification on active topic, explain flashcard, summarize section, answer content questions.
**Not allowed:** free-form chat, long-lived memory, cross-topic search, autonomous multi-step research.

## 3. Retrieval Inputs

Consumes: active `block_id` from task context, user question, topic content from `chunks` table (sliding window chunks), token budget, output constraints.

UI sends `block_id` with request. Backend validates block exists. Retrieval queries sqlite-vec filtered by `block_id` scope.

## 4. Content Structure

**Sliding Window & Markdown-Aware Chunking:**
- **RAG Embedding Chunk Target**: ~500 words (bounds: [350, 650] words)
- **Reading Task Session Window**: ~2,500 – 3,000 words (`TargetSessionWords`)
- **Storage**: `chunks` table linked to topics
- **Retrieval**: top-k chunks via sqlite-vec within `topic_id` / `block_id` scope
- **Production Model**: Nomic Embed Text v1.5 (INT8 ONNX quantized, 768-d)

### 4.1 Pro Extension Ingestion (Deep Structured PDF & Markdown)
- **Engine**: PyMuPDF4LLM (`fast_pdf` extension) converts documents to structured Markdown.
- **Markdown-Aware Chunking**: Uses `SplitMarkdownIntoChunks` (500-word target) to align splits with `#` / `##` headings while preserving tables (`|---|`) and code fences (` ``` `) as unbroken semantic units.
- **Heading Context**: Chunk text retains heading metadata for higher retrieval precision.

## 5. Retrieval Pipeline

```text
User question
  → validate active topic/notebook scope
  → embed query (ONNX) + tokenize query (lexical TF)
  → execute dense vector search + lexical keyword search concurrently
  → Reciprocal Rank Fusion (Hybrid RRF, K=60):
      Score(chunk) = sum( 1 / (60 + rank_m + 1) )
  → select top-k matches
  → assemble prompt within token budget
  → call OpenAI-compatible model once
  → return answer with citations
```

RRF combines exact keyword matching (for code symbols, numbers, definitions) with dense semantic similarity, with automatic graceful degradation if either branch produces 0 matches.

## 5.1 Vector Storage

Embeddings stored in sqlite-vec virtual table. Single persistent connection required (extensions are connection-scoped).

**Storage:** Single SQLite connection with vec0 loaded at `db.Init()`. Embeddings in sqlite-vec; relational rows reference chunk ids.

**Retrieval:** Get `block_id` from task → query sqlite-vec for vector → calculate similarity → return chunk content.

**Constraints:** Connection pool = 1. Embeddings JSON-serialized before SQL binding.

## 6. Prompt Assembly

Combines user request with minimum supporting context. Prompt includes: user question, topic metadata, retrieved chunks, output instructions.

**Rules:** Keep only most relevant chunks. Strict max token budget. Prefer concise answers unless UI requests longer explanation.

## 7. Token Budgeting

Reserve tokens for response first. Allocate remainder to context. Drop lower-ranked chunks when exceeded. Prefer fewer high-signal chunks. Don't partially force in chunks that don't fit.

## 8. Answer Behavior

Grounded explanations, not chat history. Cite sections when possible. Focus on user question. Ask user to return to Reader if context insufficient. Don't invent knowledge not in retrieved context.

## 9. Failure Modes

- No active topic → clear guidance message
- Retrieval returns nothing → state topic content insufficient
- AI unavailable → explicit online-required error
- Never fabricate or switch topics silently

## 10. What RAG Does Not Do

No global knowledge search. No chat memory. No agent planning. No background autonomous retriever.

## 11. Related Data

- `chunks` table: content
- sqlite-vec virtual table: embeddings (referenced from `chunks` via `embedding_ref`)
- Current task provides `block_id` for scoped retrieval
- UI shows chunk reference for traceability

## 12. Local Embedding Pipeline
 
1. Tokenize with `asset/tokenizer.json` (Hugging Face format)
2. Generate embeddings with ONNX (`yalue/onnxruntime_go`, `asset/model_int8.onnx`):
   - Single inference: `Embed(text)`
   - Mini-batch inference: `EmbedBatch(texts)` with dynamic mini-batch padding (default batch size: 32 chunks)
3. Persist: chunk text in relational tables, vectors in sqlite-vec virtual table
4. Retrieve top-k: embed query → pre-filter by topic_id/page_num → hybrid vector + lexical search via Reciprocal Rank Fusion (RRF, K=60)
5. Build token-budgeted prompt → call LLM once → return answer + citations

## 13. Windows Runtime Assets

Required in `asset/`: `onnxruntime.dll`, `vec0.dll`. Missing = explicit setup error, no synthetic fallback.

## 14. Build Constraints

CGO required (`CGO_ENABLED=1`). SQLite extension support: `go build -tags sqlite_extension .`
