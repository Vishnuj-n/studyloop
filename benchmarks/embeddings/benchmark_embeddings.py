#!/usr/bin/env python3
"""
Benchmarking Suite for Text Embeddings Generation.
Supports:
1. Real data from SQLite database (dev_data/Studyloop.db) or Synthetic corpus.
2. Local ONNX INT8 Runtime (asset/model_int8.onnx) vs SentenceTransformers.
3. Word-length scaling & distribution profiling (Short ~50w, Medium ~150w, Long ~500w, Book Section ~2500w).
4. Batch size scaling: [1, 4, 8, 16, 32, 64, 128].
"""

import os
import sys
import time
import sqlite3
import argparse
from typing import List, Optional, Tuple

# Ensure UTF-8 output on Windows
try:
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")
    if hasattr(sys.stderr, "reconfigure"):
        sys.stderr.reconfigure(encoding="utf-8")
except Exception:
    pass


def load_chunks_from_sqlite(db_path: str, limit: Optional[int] = None, task_prefix: str = "search_document: ") -> List[str]:
    """Loads actual real chunks from dev_data/Studyloop.db."""
    if not os.path.exists(db_path):
        raise FileNotFoundError(f"Database not found at: {db_path}")

    conn = sqlite3.connect(db_path)
    cur = conn.cursor()
    query = "SELECT chunk_text FROM chunks WHERE chunk_text IS NOT NULL AND trim(chunk_text) != '' ORDER BY page_num ASC, id ASC"
    params: tuple = ()
    if limit and limit > 0:
        query += " LIMIT ?"
        params = (limit,)

    cur.execute(query, params)
    rows = cur.fetchall()
    conn.close()

    corpus = [f"{task_prefix}{row[0].strip()}" for row in rows]
    return corpus


SAMPLE_SENTENCES = [
    "Go is a statically typed, compiled high-level programming language designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson.",
    "Memory allocation in Go uses a concurrent tri-color mark and sweep garbage collector to reclaim unused heap objects efficiently.",
    "Goroutines are lightweight execution threads managed by the Go runtime rather than the operating system kernel.",
    "Channels are typed conduits through which you can send and receive values with the channel operator <- to synchronize goroutines.",
    "The Free Spaced Repetition Scheduler (FSRS) algorithm computes optimal review intervals using four-component memory states: stability, difficulty, retrievability, and elapsed time.",
    "Reciprocal Rank Fusion (RRF) is an algorithmic technique for combining the ranked result lists of multiple information retrieval systems without score normalization.",
    "SQLite FTS5 full-text search engine leverages inverted indexes and BM25 ranking for ultra-low latency lexical retrieval across document chunks.",
    "Vector embeddings map textual tokens into high-dimensional semantic spaces where cosine distance correlates with conceptual similarity.",
    "Cellular respiration is a set of metabolic reactions that take place in the cells of organisms to convert biochemical energy from nutrients into adenosine triphosphate.",
    "Transformer architectures process entire sequences of text in parallel using multi-head self-attention mechanisms to capture long-range contextual relationships."
]


def generate_synthetic_corpus(target_chunks: int, words_per_chunk: int, task_prefix: str = "search_document: ") -> List[str]:
    """Generates synthetic text chunks with exact controlled word counts and task prefix."""
    corpus = []
    sentence_pool_len = len(SAMPLE_SENTENCES)
    
    for i in range(target_chunks):
        words = []
        sentence_idx = i % sentence_pool_len
        while len(words) < words_per_chunk:
            words.extend(SAMPLE_SENTENCES[sentence_idx % sentence_pool_len].split())
            sentence_idx += 1
        
        chunk_text = " ".join(words[:words_per_chunk])
        corpus.append(f"{task_prefix}{chunk_text}")
    return corpus


def run_onnx_benchmark(corpus: List[str], max_seq_len: int = 512, corpus_source_name: str = "Synthetic"):
    """Benchmarks local ONNX INT8 engine with realistic dynamic sequence lengths."""
    try:
        import numpy as np
        from tokenizers import Tokenizer
        import onnxruntime as ort
    except ImportError:
        print("Note: Required packages missing. Run with:")
        print("uv run --with onnxruntime --with tokenizers --with numpy python benchmarks/embeddings/benchmark_embeddings.py --engine onnx")
        sys.exit(1)

    root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
    tokenizer_path = os.path.join(root_dir, "asset", "tokenizer.json")
    model_path = os.path.join(root_dir, "asset", "model_int8.onnx")

    if not os.path.exists(tokenizer_path) or not os.path.exists(model_path):
        print(f"Error: Asset files missing:\n  {tokenizer_path}\n  {model_path}")
        sys.exit(1)

    chunk_count = len(corpus)
    total_words = sum(len(c.split()) for c in corpus)
    avg_words = total_words / chunk_count if chunk_count > 0 else 0

    print("=" * 96)
    print("EMBEDDING INFERENCE BENCHMARK — LOCAL ONNX INT8 RUNTIME")
    print(f"Engine          : ONNX Runtime (CPU)")
    print(f"Model Asset     : asset/model_int8.onnx")
    print(f"Tokenizer       : asset/tokenizer.json")
    print(f"Corpus Source   : {corpus_source_name} ({chunk_count} total chunks)")
    print(f"Avg Words/Chunk : {avg_words:.1f} words (Total: {total_words:,} words)")
    print(f"Max Seq Length  : {max_seq_len} tokens")
    print("=" * 96)

    print("Loading tokenizer and ONNX session ...", end="", flush=True)
    t0 = time.perf_counter()
    tok = Tokenizer.from_file(tokenizer_path)
    tok.enable_truncation(max_length=max_seq_len)

    session_opts = ort.SessionOptions()
    session_opts.intra_op_num_threads = os.cpu_count() or 4
    session_opts.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
    session = ort.InferenceSession(model_path, sess_options=session_opts)
    load_time = time.perf_counter() - t0
    print(f" Loaded in {load_time:.2f}s (Threads: {session_opts.intra_op_num_threads})")

    batch_sizes = [1, 4, 8, 16, 32, 64, 128]
    batch_sizes = [b for b in batch_sizes if b <= chunk_count or b == batch_sizes[0]]
    results = []
    baseline_time = None

    def mean_pooling(token_embeddings: np.ndarray, attention_mask: np.ndarray) -> np.ndarray:
        input_mask_expanded = np.expand_dims(attention_mask, axis=-1).astype(np.float32)
        sum_embeddings = np.sum(token_embeddings * input_mask_expanded, axis=1)
        sum_mask = np.clip(input_mask_expanded.sum(axis=1), a_min=1e-9, a_max=None)
        return sum_embeddings / sum_mask

    def normalize_l2(vector: np.ndarray) -> np.ndarray:
        norm = np.linalg.norm(vector, axis=-1, keepdims=True)
        norm = np.where(norm == 0, 1e-12, norm)
        return vector / norm

    for bs in batch_sizes:
        print(f"Testing Batch Size = {bs:>3} ...", end="", flush=True)
        t_start = time.perf_counter()

        dim = 0
        for i in range(0, len(corpus), bs):
            batch_texts = corpus[i : i + bs]
            # Encode batch with dynamic padding per mini-batch
            encodings = tok.encode_batch(batch_texts)
            max_batch_tokens = max(len(e.ids) for e in encodings)

            # Pad explicitly to batch maximum (not global max)
            input_ids_list = []
            attention_mask_list = []
            token_type_ids_list = []

            for e in encodings:
                pad_len = max_batch_tokens - len(e.ids)
                input_ids_list.append(e.ids + [0] * pad_len)
                attention_mask_list.append(e.attention_mask + [0] * pad_len)
                token_type_ids_list.append(e.type_ids + [0] * pad_len)

            input_ids = np.array(input_ids_list, dtype=np.int64)
            attention_mask = np.array(attention_mask_list, dtype=np.int64)
            token_type_ids = np.array(token_type_ids_list, dtype=np.int64)

            feed_dict = {}
            for inp in session.get_inputs():
                name_lower = inp.name.lower()
                if "attention" in name_lower:
                    feed_dict[inp.name] = attention_mask
                elif "token_type" in name_lower or "segment" in name_lower:
                    feed_dict[inp.name] = token_type_ids
                else:
                    feed_dict[inp.name] = input_ids

            outputs = session.run(None, feed_dict)
            raw = outputs[0]
            if len(raw.shape) == 3:
                pooled = mean_pooling(raw, attention_mask)
            else:
                pooled = raw
            normed = normalize_l2(pooled)
            dim = normed.shape[-1]

        elapsed = time.perf_counter() - t_start
        if baseline_time is None:
            baseline_time = elapsed

        chunks_per_sec = chunk_count / elapsed if elapsed > 0 else 0
        words_per_sec = total_words / elapsed if elapsed > 0 else 0
        speedup = baseline_time / elapsed if elapsed > 0 else 1.0

        print(f" {elapsed:>6.3f}s | {chunks_per_sec:>6.1f} chunks/s | {words_per_sec:>7.1f} words/s | {speedup:>5.2f}x speedup")

        results.append({
            "batch_size": bs,
            "time": elapsed,
            "chunks_per_sec": chunks_per_sec,
            "words_per_sec": words_per_sec,
            "speedup": f"{speedup:.2f}x",
            "dim": dim
        })

    print_summary_table(results)


def run_sentence_transformers_benchmark(corpus: List[str], model_name: str, corpus_source_name: str = "Synthetic"):
    """Benchmarks HuggingFace SentenceTransformer model."""
    try:
        from sentence_transformers import SentenceTransformer
    except ImportError:
        print("Note: 'sentence_transformers' is not installed in the active environment.")
        print("Run with: uv run --with sentence-transformers --with einops python benchmarks/embeddings/benchmark_embeddings.py --engine st")
        sys.exit(1)

    chunk_count = len(corpus)
    total_words = sum(len(c.split()) for c in corpus)
    avg_words = total_words / chunk_count if chunk_count > 0 else 0

    print("=" * 96)
    print("EMBEDDING INFERENCE BENCHMARK — SENTENCETRANSFORMERS")
    print(f"Model Name      : {model_name}")
    print(f"Corpus Source   : {corpus_source_name} ({chunk_count} total chunks)")
    print(f"Avg Words/Chunk : {avg_words:.1f} words (Total: {total_words:,} words)")
    print("=" * 96)

    print("Loading embedding model ...", end="", flush=True)
    t0 = time.perf_counter()
    model = SentenceTransformer(model_name, trust_remote_code=True)
    load_time = time.perf_counter() - t0
    print(f" Loaded in {load_time:.2f}s")

    batch_sizes = [1, 4, 8, 16, 32, 64, 128]
    batch_sizes = [b for b in batch_sizes if b <= chunk_count or b == batch_sizes[0]]
    results = []
    baseline_time = None

    for bs in batch_sizes:
        print(f"Testing Batch Size = {bs:>3} ...", end="", flush=True)
        t_start = time.perf_counter()

        embeddings = model.encode(corpus, batch_size=bs, show_progress_bar=False, normalize_embeddings=True)
        elapsed = time.perf_counter() - t_start

        if baseline_time is None:
            baseline_time = elapsed

        chunks_per_sec = chunk_count / elapsed if elapsed > 0 else 0
        words_per_sec = total_words / elapsed if elapsed > 0 else 0
        speedup = baseline_time / elapsed if elapsed > 0 else 1.0

        print(f" {elapsed:>6.3f}s | {chunks_per_sec:>6.1f} chunks/s | {words_per_sec:>7.1f} words/s | {speedup:>5.2f}x speedup")

        results.append({
            "batch_size": bs,
            "time": elapsed,
            "chunks_per_sec": chunks_per_sec,
            "words_per_sec": words_per_sec,
            "speedup": f"{speedup:.2f}x",
            "dim": len(embeddings[0]) if len(embeddings) > 0 else 0
        })

    print_summary_table(results)


def print_summary_table(results: List[dict]):
    print("\n" + "=" * 92)
    print("EMBEDDING BATCH RESULTS SUMMARY")
    print("=" * 92)
    print(f"{'Batch Size':<12} | {'Time (s)':<10} | {'Chunks/sec':<12} | {'Words/sec':<12} | {'Speedup':<10} | {'Dim':<6}")
    print("-" * 92)
    best_time = min(r["time"] for r in results)
    for r in results:
        is_best = " 🏆" if r["time"] == best_time else ""
        print(f"{r['batch_size']:<12} | {r['time']:>9.3f}s | {r['chunks_per_sec']:>11.1f} | {r['words_per_sec']:>11.1f} | {r['speedup']:>9} | {r['dim']:<6}{is_best}")
    print("=" * 92)


def main():
    root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
    default_db = os.path.join(root_dir, "dev_data", "Studyloop.db")

    parser = argparse.ArgumentParser(description="Benchmark Embedding Inference Strategies with SQLite Data & Synthetic Scaling")
    parser.add_argument("--engine", choices=["onnx", "st"], default="onnx",
                        help="Inference engine: 'onnx' (Local INT8 ONNX asset) or 'st' (SentenceTransformers)")
    parser.add_argument("--model", default="nomic-ai/nomic-embed-text-v1.5",
                        help="SentenceTransformer model name (used if --engine st)")
    parser.add_argument("--use-db", action="store_true", default=False,
                        help="Load real textbook chunks from dev_data/Studyloop.db instead of synthetic data")
    parser.add_argument("--db-path", default=default_db,
                        help="Path to SQLite database containing chunks table")
    parser.add_argument("--chunks", type=int, default=256,
                        help="Number of text chunks to embed (if --use-db, limits chunk query count)")
    parser.add_argument("--words-per-chunk", type=int, default=150,
                        help="Target words per chunk for synthetic generation (e.g. 50, 150, 500, 2500)")
    parser.add_argument("--max-seq-len", type=int, default=512,
                        help="Max token truncation length for ONNX tokenizer (e.g. 256, 512, 1024, 2048)")
    parser.add_argument("--prefix", default="search_document: ",
                        help="Task prefix prepended to text chunks")
    args = parser.parse_args()

    if args.use_db:
        if not os.path.exists(args.db_path):
            print(f"Error: SQLite database not found at {args.db_path}")
            sys.exit(1)
        corpus = load_chunks_from_sqlite(args.db_path, limit=args.chunks, task_prefix=args.prefix)
        corpus_source = f"SQLite ({args.db_path})"
    else:
        corpus = generate_synthetic_corpus(args.chunks, args.words_per_chunk, task_prefix=args.prefix)
        corpus_source = f"Synthetic ({args.words_per_chunk} words/chunk)"

    if len(corpus) == 0:
        print("Error: Corpus is empty. No chunks to embed.")
        sys.exit(1)

    if args.engine == "onnx":
        run_onnx_benchmark(corpus, max_seq_len=args.max_seq_len, corpus_source_name=corpus_source)
    else:
        run_sentence_transformers_benchmark(corpus, model_name=args.model, corpus_source_name=corpus_source)


if __name__ == "__main__":
    main()
