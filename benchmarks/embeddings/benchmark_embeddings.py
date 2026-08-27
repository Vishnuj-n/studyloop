#!/usr/bin/env python3
"""
Benchmarking Suite for Text Embeddings Generation.
Compares sequential vs batched inference (Batch sizes: 1, 4, 8, 16, 32, 64, 128)
measuring throughput (chunks/sec, words/sec).
"""

import sys
import time
import argparse
from typing import List

# Ensure UTF-8 output on Windows
try:
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")
    if hasattr(sys.stderr, "reconfigure"):
        sys.stderr.reconfigure(encoding="utf-8")
except Exception:
    pass


SAMPLE_TEXTS = [
    "Go is a statically typed, compiled high-level programming language designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson.",
    "Memory allocation in Go uses a concurrent tri-color mark and sweep garbage collector to reclaim unused heap objects efficiently.",
    "Goroutines are lightweight execution threads managed by the Go runtime rather than the operating system kernel.",
    "Channels are typed conduits through which you can send and receive values with the channel operator <- to synchronize goroutines.",
    "The Free Spaced Repetition Scheduler (FSRS) algorithm computes optimal review intervals using four-component memory states: stability, difficulty, retrievability, and elapsed time.",
    "Reciprocal Rank Fusion (RRF) is an algorithmic technique for combining the ranked result lists of multiple information retrieval systems without score normalization.",
    "SQLite FTS5 full-text search engine leverages inverted indexes and BM25 ranking for ultra-low latency lexical retrieval across document chunks.",
    "Vector embeddings map textual tokens into high-dimensional semantic spaces where cosine distance correlates with conceptual similarity."
]


def generate_synthetic_corpus(target_count: int) -> List[str]:
    corpus = []
    for i in range(target_count):
        base = SAMPLE_TEXTS[i % len(SAMPLE_TEXTS)]
        corpus.append(f"[Chunk #{i+1}] {base} Additional context padding for realistic chunk density and token distributions.")
    return corpus


def run_embedding_benchmark(model_name: str = "all-MiniLM-L6-v2", chunk_count: int = 256):
    try:
        from sentence_transformers import SentenceTransformer
    except ImportError:
        print("Note: 'sentence_transformers' is not installed in the active environment.")
        print("Run with: uv run --with sentence-transformers python benchmarks/embeddings/benchmark_embeddings.py")
        sys.exit(1)

    print("=" * 80)
    print("EMBEDDING INFERENCE BATCHING BENCHMARK")
    print(f"Model Name  : {model_name}")
    print(f"Total Chunks: {chunk_count}")
    print("=" * 80)

    print("Loading embedding model ...", end="", flush=True)
    t0 = time.perf_counter()
    model = SentenceTransformer(model_name)
    load_time = time.perf_counter() - t0
    print(f" Loaded in {load_time:.2f}s")

    corpus = generate_synthetic_corpus(chunk_count)
    total_words = sum(len(c.split()) for c in corpus)

    batch_sizes = [1, 4, 8, 16, 32, 64, 128]
    results = []

    baseline_time = None

    for bs in batch_sizes:
        print(f"Testing Batch Size = {bs:>3} ...", end="", flush=True)
        t_start = time.perf_counter()
        
        # Run batched encoding
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

    # Summary table
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
    parser = argparse.ArgumentParser(description="Benchmark Embedding Batch Sizing")
    parser.add_argument("--chunks", type=int, default=256, help="Number of text chunks to embed")
    parser.add_argument("--model", default="all-MiniLM-L6-v2", help="SentenceTransformer model name")
    args = parser.parse_args()

    run_embedding_benchmark(model_name=args.model, chunk_count=args.chunks)


if __name__ == "__main__":
    main()
