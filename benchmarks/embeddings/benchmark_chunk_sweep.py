#!/usr/bin/env python3
"""
Chunk Sweep Benchmark Harness: 100 to 1,200 Words.
Profiles real textbook text (from learning go.pdf or SQLite) across chunk sizes:
[100, 200, 400, 600, 800, 1000, 1200] words.

Measures:
- Text extraction throughput (PyMuPDF / pypdf)
- Generated chunk count & token distribution
- ONNX INT8 embedding latency & throughput (chunks/s, words/s)
- Total ingestion time per 1,000 book pages
"""

import os
import sys
import time
import argparse
import numpy as np
from typing import List, Dict, Any

# Ensure UTF-8 output on Windows
try:
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")
    if hasattr(sys.stderr, "reconfigure"):
        sys.stderr.reconfigure(encoding="utf-8")
except Exception:
    pass


def extract_text_from_pdf(pdf_path: str, max_pages: int = 50) -> str:
    """Extracts raw text from PDF using pymupdf, pypdf, or fallback."""
    if not os.path.exists(pdf_path):
        raise FileNotFoundError(f"PDF file not found at: {pdf_path}")

    # Try pymupdf / fitz
    try:
        import fitz
        doc = fitz.open(pdf_path)
        pages_to_read = min(len(doc), max_pages) if max_pages > 0 else len(doc)
        text_parts = []
        for i in range(pages_to_read):
            text_parts.append(doc[i].get_text())
        doc.close()
        return "\n".join(text_parts)
    except ImportError:
        pass

    # Try pypdf
    try:
        import pypdf
        reader = pypdf.PdfReader(pdf_path)
        pages_to_read = min(len(reader.pages), max_pages) if max_pages > 0 else len(reader.pages)
        text_parts = [reader.pages[i].extract_text() or "" for i in range(pages_to_read)]
        return "\n".join(text_parts)
    except ImportError:
        pass

    # Fallback to PyMuPDF4LLM if available
    try:
        import pymupdf4llm
        return pymupdf4llm.to_markdown(pdf_path, pages=list(range(max_pages)))
    except ImportError:
        print("Error: Neither 'pymupdf' nor 'pypdf' is installed.")
        print("Run with: uv run --with pymupdf --with onnxruntime --with tokenizers --with numpy python benchmarks/embeddings/benchmark_chunk_sweep.py")
        sys.exit(1)


def slice_text_into_chunks(raw_text: str, target_words: int, task_prefix: str = "search_document: ") -> List[str]:
    """Splits text into chunks of roughly target_words words."""
    words = raw_text.split()
    if not words:
        return []

    chunks = []
    for i in range(0, len(words), target_words):
        chunk_words = words[i : i + target_words]
        if len(chunk_words) < max(20, target_words // 4) and chunks:
            # Append tiny trailing tail to last chunk
            chunks[-1] += " " + " ".join(chunk_words)
        else:
            chunk_str = " ".join(chunk_words)
            chunks.append(f"{task_prefix}{chunk_str}")
    return chunks


def benchmark_chunk_sweep(pdf_path: str, max_pages: int = 40, batch_size: int = 4):
    try:
        from tokenizers import Tokenizer
        import onnxruntime as ort
    except ImportError:
        print("Error: onnxruntime or tokenizers not installed.")
        print("Run with: uv run --with pymupdf --with onnxruntime --with tokenizers --with numpy python benchmarks/embeddings/benchmark_chunk_sweep.py")
        sys.exit(1)

    root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
    tokenizer_path = os.path.join(root_dir, "asset", "tokenizer.json")
    model_path = os.path.join(root_dir, "asset", "model_int8.onnx")

    if not os.path.exists(tokenizer_path) or not os.path.exists(model_path):
        print(f"Error: Asset files not found:\n  {tokenizer_path}\n  {model_path}")
        sys.exit(1)

    print("=" * 100)
    print("CHUNK SIZE GRID SWEEP BENCHMARK (100 to 1,200 Words)")
    print(f"Source PDF  : {pdf_path}")
    print(f"Max Pages   : {max_pages} pages")
    print(f"Batch Size  : {batch_size} (optimal CPU setting)")
    print("=" * 100)

    print(f"\n1. Extracting text from '{pdf_path}' ({max_pages} pages) ...", end="", flush=True)
    t_extract = time.perf_counter()
    raw_text = extract_text_from_pdf(pdf_path, max_pages=max_pages)
    extract_time = time.perf_counter() - t_extract
    total_words = len(raw_text.split())
    print(f" Extracted {total_words:,} words in {extract_time:.2f}s ({total_words/extract_time:.0f} words/s)")

    print("2. Initializing ONNX INT8 Session ...", end="", flush=True)
    t_load = time.perf_counter()
    tok = Tokenizer.from_file(tokenizer_path)
    tok.enable_truncation(max_length=2048)

    session_opts = ort.SessionOptions()
    session_opts.intra_op_num_threads = os.cpu_count() or 4
    session_opts.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
    session = ort.InferenceSession(model_path, sess_options=session_opts)
    print(f" Loaded in {time.perf_counter() - t_load:.2f}s (Threads: {session_opts.intra_op_num_threads})")

    # Sweep chunk sizes from 100 to 1200 words
    sweep_sizes = [100, 200, 400, 600, 800, 1000, 1200]
    results = []

    print("\n" + "=" * 100)
    print(f"{'Target Words':<14} | {'Chunk Count':<12} | {'Time (s)':<10} | {'Chunks/sec':<12} | {'Words/sec':<12} | {'Avg Tokens':<10} | {'Extrapolate 1k pgs'}")
    print("-" * 100)

    for target_w in sweep_sizes:
        chunks = slice_text_into_chunks(raw_text, target_w)
        if not chunks:
            continue

        t_start = time.perf_counter()

        token_counts = []
        for i in range(0, len(chunks), batch_size):
            batch_texts = chunks[i : i + batch_size]
            encodings = tok.encode_batch(batch_texts)
            token_counts.extend([len(e.ids) for e in encodings])

            max_batch_tokens = max(len(e.ids) for e in encodings)
            input_ids = []
            attention_mask = []
            token_type_ids = []

            for e in encodings:
                pad_len = max_batch_tokens - len(e.ids)
                input_ids.append(e.ids + [0] * pad_len)
                attention_mask.append(e.attention_mask + [0] * pad_len)
                token_type_ids.append(e.type_ids + [0] * pad_len)

            feed_dict = {}
            for inp in session.get_inputs():
                name_lower = inp.name.lower()
                if "attention" in name_lower:
                    feed_dict[inp.name] = np.array(attention_mask, dtype=np.int64)
                elif "token_type" in name_lower or "segment" in name_lower:
                    feed_dict[inp.name] = np.array(token_type_ids, dtype=np.int64)
                else:
                    feed_dict[inp.name] = np.array(input_ids, dtype=np.int64)

            session.run(None, feed_dict)

        elapsed = time.perf_counter() - t_start
        chunks_per_sec = len(chunks) / elapsed if elapsed > 0 else 0
        words_per_sec = total_words / elapsed if elapsed > 0 else 0
        avg_tokens = sum(token_counts) / len(token_counts) if token_counts else 0

        # Estimate time for 1,000 pages (~250,000 words)
        est_1k_pages_time = (250000 / words_per_sec) if words_per_sec > 0 else 0
        est_str = f"{est_1k_pages_time/60:.1f} min" if est_1k_pages_time >= 60 else f"{est_1k_pages_time:.1f}s"

        results.append({
            "target_words": target_w,
            "chunk_count": len(chunks),
            "time": elapsed,
            "chunks_per_sec": chunks_per_sec,
            "words_per_sec": words_per_sec,
            "avg_tokens": avg_tokens,
            "est_1k": est_str
        })

        print(f"{target_w:<14} | {len(chunks):<12} | {elapsed:>9.2f}s | {chunks_per_sec:>11.1f} | {words_per_sec:>11.1f} | {avg_tokens:>9.1f} | {est_str}")

    print("=" * 100)

    # Print analysis recommendation
    fastest_words = max(results, key=lambda x: x["words_per_sec"])
    print("\n📊 GRID SEARCH SUMMARY & OBSERVATIONS:")
    print(f"• Peak Word Throughput : {fastest_words['target_words']} words/chunk ({fastest_words['words_per_sec']:.1f} words/sec)")
    print(f"• Ingestion Speed Scaling: Larger chunks process more total words/sec because per-chunk tensor call overhead is amortized.")
    print(f"• RAG Precision Tradeoff :")
    print(f"  - 100–200 words  : Highest retrieval granularity, but higher chunk count.")
    print(f"  - 400–600 words  : Optimal balance of paragraph completeness, speed, and vector specificity.")
    print(f"  - 800–1200 words : Fast ingestion, but vector represents broad multi-page concepts.")


def main():
    root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
    default_pdf = os.path.join(root_dir, "learning go.pdf")

    parser = argparse.ArgumentParser(description="Chunk Sweep Benchmark (100 to 1200 Words)")
    parser.add_argument("--pdf", default=default_pdf, help="Path to textbook PDF")
    parser.add_argument("--pages", type=int, default=30, help="Number of pages to parse and sweep")
    parser.add_argument("--batch-size", type=int, default=4, help="Inference batch size")
    args = parser.parse_args()

    benchmark_chunk_sweep(pdf_path=args.pdf, max_pages=args.pages, batch_size=args.batch_size)


if __name__ == "__main__":
    main()
