#!/usr/bin/env python3
"""
Benchmarking Suite for Fast PDF Ingestion.
Compares various concurrency models (Sequential, ThreadPool, ProcessPool,
1-Page-Per-Worker, Max-Hardware Threads, Adaptive/Gradual Degradation)
to measure latency, throughput, and output consistency.
"""

import os
import sys
import time
import argparse
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor, ProcessPoolExecutor, as_completed
from typing import List, Dict, Any

# Ensure UTF-8 output on Windows
try:
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")
    if hasattr(sys.stderr, "reconfigure"):
        sys.stderr.reconfigure(encoding="utf-8")
except Exception:
    pass


def convert_slice(file_path: str, page_slice: List[int]) -> str:
    """Worker function to convert a specific slice of pages using PyMuPDF4LLM."""
    import pymupdf4llm
    if not page_slice:
        return ""
    return pymupdf4llm.to_markdown(
        file_path,
        pages=page_slice,
        page_chunks=False,
        write_images=False
    )


# --- STRATEGY 1: Sequential / Baseline ---
def strategy_sequential(file_path: str, pages: List[int]) -> str:
    import pymupdf4llm
    return pymupdf4llm.to_markdown(file_path, pages=pages, page_chunks=False, write_images=False)


# --- STRATEGY 2 & 3: ThreadPool with Fixed / Max Worker Count ---
def strategy_thread_pool(file_path: str, pages: List[int], workers: int, chunk_size: int = None) -> str:
    total = len(pages)
    if chunk_size is None:
        chunk_size = max(1, (total + workers - 1) // workers)
    chunks = [pages[i:i + chunk_size] for i in range(0, total, chunk_size)]
    
    results = {}
    with ThreadPoolExecutor(max_workers=workers) as executor:
        future_map = {executor.submit(convert_slice, file_path, c): idx for idx, c in enumerate(chunks)}
        for fut in as_completed(future_map):
            idx = future_map[fut]
            results[idx] = fut.result()
            
    ordered = [results[i] for i in range(len(chunks))]
    return "\n\n".join(p.strip() for p in ordered if p and p.strip())


# --- STRATEGY 4: 1 Page Per Thread ---
def strategy_one_page_per_thread(file_path: str, pages: List[int], workers: int) -> str:
    chunks = [[p] for p in pages]
    results = {}
    with ThreadPoolExecutor(max_workers=workers) as executor:
        future_map = {executor.submit(convert_slice, file_path, c): idx for idx, c in enumerate(chunks)}
        for fut in as_completed(future_map):
            idx = future_map[fut]
            results[idx] = fut.result()
            
    ordered = [results[i] for i in range(len(chunks))]
    return "\n\n".join(p.strip() for p in ordered if p and p.strip())


# --- STRATEGY 5: Gradual / Adaptive Degradation Chunking ---
def strategy_adaptive_degradation(file_path: str, pages: List[int], workers: int) -> str:
    """
    Adaptive degradation: Distributes larger chunks initially to maximize batch efficiency,
    then degrades to smaller chunks towards the tail to minimize straggler latency.
    """
    total = len(pages)
    chunks = []
    curr = 0
    # Start with chunk size ~ total // (workers / 2), gradually degrading down to chunk size of 1-2
    current_chunk_size = max(4, total // (workers * 2)) if total > workers * 4 else max(1, total // workers)
    
    while curr < total:
        take = min(current_chunk_size, total - curr)
        chunks.append(pages[curr:curr + take])
        curr += take
        if current_chunk_size > 2 and (total - curr) < (total // 3):
            current_chunk_size = max(1, current_chunk_size // 2)

    results = {}
    with ThreadPoolExecutor(max_workers=workers) as executor:
        future_map = {executor.submit(convert_slice, file_path, c): idx for idx, c in enumerate(chunks)}
        for fut in as_completed(future_map):
            idx = future_map[fut]
            results[idx] = fut.result()
            
    ordered = [results[i] for i in range(len(chunks))]
    return "\n\n".join(p.strip() for p in ordered if p and p.strip())


# --- STRATEGY 6: Multiprocessing (ProcessPoolExecutor) ---
def strategy_process_pool(file_path: str, pages: List[int], workers: int, chunk_size: int = None) -> str:
    total = len(pages)
    if chunk_size is None:
        chunk_size = max(1, (total + workers - 1) // workers)
    chunks = [pages[i:i + chunk_size] for i in range(0, total, chunk_size)]
    
    results = {}
    with ProcessPoolExecutor(max_workers=workers) as executor:
        future_map = {executor.submit(convert_slice, file_path, c): idx for idx, c in enumerate(chunks)}
        for fut in as_completed(future_map):
            idx = future_map[fut]
            results[idx] = fut.result()
            
    ordered = [results[i] for i in range(len(chunks))]
    return "\n\n".join(p.strip() for p in ordered if p and p.strip())


def run_benchmark(pdf_path: str, start_page: int = 0, end_page: int = 0, runs: int = 1):
    import pymupdf
    
    doc = pymupdf.open(pdf_path)
    total_doc_pages = len(doc)
    doc.close()
    
    min_p = max(0, start_page - 1) if start_page > 0 else 0
    max_p = min(total_doc_pages, end_page) if end_page > 0 else total_doc_pages
    pages = list(range(min_p, max_p))
    num_pages = len(pages)
    
    cpu_count = os.cpu_count() or 4
    
    print("=" * 80)
    print("PDF INGESTION CONCURRENCY BENCHMARK")
    print(f"Target File : {pdf_path}")
    print(f"Page Range  : {min_p + 1} to {max_p} (Total Pages: {num_pages} of {total_doc_pages})")
    print(f"Host System : {cpu_count} Logical CPU Cores")
    print(f"Iterations  : {runs} per strategy")
    print("=" * 80)
    
    test_cases: List[Dict[str, Any]] = [
        {
            "name": "Sequential (1-pass Baseline)",
            "fn": lambda: strategy_sequential(pdf_path, pages)
        },
        {
            "name": "Fixed ThreadPool (2 workers, even split)",
            "fn": lambda: strategy_thread_pool(pdf_path, pages, workers=2)
        },
        {
            "name": "Fixed ThreadPool (4 workers, even split)",
            "fn": lambda: strategy_thread_pool(pdf_path, pages, workers=4)
        },
        {
            "name": "Fixed ThreadPool (8 workers, even split)",
            "fn": lambda: strategy_thread_pool(pdf_path, pages, workers=8)
        },
        {
            "name": f"CPU Count ThreadPool ({cpu_count} workers, even split)",
            "fn": lambda: strategy_thread_pool(pdf_path, pages, workers=cpu_count)
        },
        {
            "name": f"Overcommit ThreadPool ({cpu_count * 2} workers, even split)",
            "fn": lambda: strategy_thread_pool(pdf_path, pages, workers=cpu_count * 2)
        },
        {
            "name": f"1 Page / Thread (Pool={cpu_count} workers)",
            "fn": lambda: strategy_one_page_per_thread(pdf_path, pages, workers=cpu_count)
        },
        {
            "name": f"1 Page / Thread (Pool={min(num_pages, 32)} workers)",
            "fn": lambda: strategy_one_page_per_thread(pdf_path, pages, workers=min(num_pages, 32))
        },
        {
            "name": f"Adaptive Degradation ({cpu_count} workers)",
            "fn": lambda: strategy_adaptive_degradation(pdf_path, pages, workers=cpu_count)
        },
        {
            "name": f"ProcessPool ({cpu_count} processes, even split)",
            "fn": lambda: strategy_process_pool(pdf_path, pages, workers=cpu_count)
        },
    ]

    results_table = []
    baseline_time = None

    for idx, tc in enumerate(test_cases, 1):
        name = tc["name"]
        fn = tc["fn"]
        print(f"\n[{idx}/{len(test_cases)}] Running: {name} ...", end="", flush=True)
        
        times = []
        words = 0
        text_len = 0
        error = None
        
        for r in range(runs):
            t0 = time.perf_counter()
            try:
                out = fn()
                elapsed = time.perf_counter() - t0
                times.append(elapsed)
                words = len(out.split())
                text_len = len(out)
            except Exception as e:
                error = str(e)
                break

        if error:
            print(f" FAILED ({error})")
            results_table.append({
                "name": name,
                "time": float("inf"),
                "speed": 0,
                "words": 0,
                "speedup": "ERR",
                "error": error
            })
            continue

        avg_time = sum(times) / len(times)
        if baseline_time is None and idx == 1:
            baseline_time = avg_time

        pages_per_sec = num_pages / avg_time if avg_time > 0 else 0
        speedup = (baseline_time / avg_time) if baseline_time and avg_time > 0 else 1.0
        
        print(f" Done in {avg_time:.3f}s ({pages_per_sec:.1f} p/s, {speedup:.2f}x)")
        
        results_table.append({
            "name": name,
            "time": avg_time,
            "speed": pages_per_sec,
            "words": words,
            "chars": text_len,
            "speedup": f"{speedup:.2f}x"
        })

    # Print Summary Table
    print("\n" + "=" * 95)
    print("BENCHMARK RESULTS SUMMARY")
    print("=" * 95)
    header = f"{'Strategy / Concurrency Mode':<48} | {'Time (s)':<9} | {'Pages/s':<8} | {'Speedup':<8} | {'Words':<8}"
    print(header)
    print("-" * 95)
    
    best_time = min(r["time"] for r in results_table if r.get("time", float("inf")) > 0)
    
    for r in results_table:
        is_best = " 🏆" if r["time"] == best_time else ""
        print(f"{r['name']:<48} | {r['time']:>8.3f}s | {r['speed']:>7.1f} | {r['speedup']:>7} | {r.get('words', 0):>8}{is_best}")
    print("=" * 95)


def main():
    parser = argparse.ArgumentParser(description="Benchmark PDF Ingestion Strategies")
    parser.add_argument("pdf_path", nargs="?", default="learning go.pdf", help="Path to PDF file")
    parser.add_argument("--start-page", type=int, default=1, help="Start page (1-based)")
    parser.add_argument("--end-page", type=int, default=20, help="End page (1-based, default 20 pages for rapid test)")
    parser.add_argument("--runs", type=int, default=1, help="Number of benchmark runs per strategy")
    
    args = parser.parse_args()
    
    pdf_path = Path(args.pdf_path)
    if not pdf_path.exists():
        candidates = list(Path("dev_data/uploads").glob("*.pdf"))
        if candidates:
            pdf_path = candidates[0]
            print(f"Notice: '{args.pdf_path}' not found, using fallback: {pdf_path}")
        else:
            print(f"Error: PDF file '{args.pdf_path}' not found and no uploads available in dev_data/uploads.")
            sys.exit(1)

    run_benchmark(str(pdf_path), args.start_page, args.end_page, args.runs)


if __name__ == "__main__":
    main()
