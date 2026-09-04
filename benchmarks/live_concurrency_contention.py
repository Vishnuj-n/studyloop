import time
import os
import threading
import numpy as np

def generate_synthetic_pages(num_pages=1000, words_per_page=300):
    text_unit = "Go Wails SQLite WAL deterministic state machine queue ONNX INT8 RAG. "
    page_str = text_unit * (words_per_page // len(text_unit.split()))
    page_bytes = page_str.encode("utf-8")
    return [page_bytes for _ in range(num_pages)]

def parse_page(page_bytes: bytes) -> int:
    decoded = page_bytes.decode("utf-8")
    words = decoded.split()
    return sum(len(w) for w in words)

def run_foreground_pdf_extraction(pages, num_workers=None):
    if num_workers is None:
        num_workers = min(len(pages), (os.cpu_count() or 4) * 2)

    start_time = time.perf_counter()
    num_pages = len(pages)
    results = [0] * num_pages

    def worker(indices):
        for idx in indices:
            results[idx] = parse_page(pages[idx])

    chunks = [[] for _ in range(num_workers)]
    for i in range(num_pages):
        chunks[i % num_workers].append(i)

    threads = [threading.Thread(target=worker, args=(chunks[w],)) for w in range(num_workers)]
    for t in threads:
        t.start()
    for t in threads:
        t.join()

    elapsed = time.perf_counter() - start_time
    throughput = num_pages / elapsed
    return elapsed, throughput

def run_onnx_single_batch(batch_size=64, dim=768, reps=15):
    a = np.random.randn(batch_size, dim).astype(np.float32)
    b = np.random.randn(dim, dim).astype(np.float32)
    for _ in range(reps):
        c = np.dot(a, b)
        a = c / (np.linalg.norm(c, axis=1, keepdims=True) + 1e-6)
    return a

def run_background_onnx_worker(num_batches=20, batch_size=64, inter_batch_sleep=0.0):
    start_time = time.perf_counter()
    for _ in range(num_batches):
        _ = run_onnx_single_batch(batch_size=batch_size)
        if inter_batch_sleep > 0:
            time.sleep(inter_batch_sleep)
    elapsed = time.perf_counter() - start_time
    return elapsed

def main():
    print("=" * 85)
    print("LIVE CONTENTION BENCHMARK: FOREGROUND PDF EXTRACTION VS BACKGROUND ONNX EMBEDDINGS")
    print(f"Host CPU Cores: {os.cpu_count()} | Target: 1,000 PDF Pages vs 20 ONNX Batches (Batch=64, 768-d)")
    print("=" * 85)

    print("\nPre-generating 1,000 synthetic PDF pages in memory...")
    pages = generate_synthetic_pages(1000, 300)

    print("\n--- 1. ISOLATED BASELINES (Zero Contention) ---")
    pdf_iso_time, pdf_iso_tp = run_foreground_pdf_extraction(pages)
    print(f"Foreground PDF Extraction Alone : {pdf_iso_time:.3f}s ({pdf_iso_tp:.1f} pages/sec)")

    onnx_iso_time = run_background_onnx_worker(num_batches=20, batch_size=64, inter_batch_sleep=0.0)
    print(f"Background ONNX Embedding Alone : {onnx_iso_time:.3f}s")

    print("\n--- 2. UNGATED CONCURRENCY (Individual Optimums Colliding) ---")
    print("Executing full-throttle PDF extraction + full-throttle ONNX embedding simultaneously...")

    pdf_res = {}
    onnx_res = {}

    def pdf_task_wrapper():
        t, tp = run_foreground_pdf_extraction(pages)
        pdf_res["time"] = t
        pdf_res["tp"] = tp

    def onnx_ungated_wrapper():
        t = run_background_onnx_worker(num_batches=20, batch_size=64, inter_batch_sleep=0.0)
        onnx_res["time"] = t

    t_start = time.perf_counter()
    t1 = threading.Thread(target=pdf_task_wrapper)
    t2 = threading.Thread(target=onnx_ungated_wrapper)

    t1.start()
    t2.start()
    t1.join()
    t2.join()
    total_ungated_time = time.perf_counter() - t_start

    pdf_deg = ((pdf_res["time"] - pdf_iso_time) / pdf_iso_time) * 100
    print(f"Foreground PDF Extraction Time  : {pdf_res['time']:.3f}s ({pdf_res['tp']:.1f} p/s) [+{pdf_deg:.1f}% Latency Spike]")
    print(f"Background ONNX Embedding Time  : {onnx_res['time']:.3f}s")
    print(f"Total Wall-Clock Time           : {total_ungated_time:.3f}s")

    print("\n--- 3. SYSTEM-OPTIMAL COORDINATION (Background Queue with Micro-Yield) ---")
    print("Executing PDF extraction + Background ONNX with 15ms inter-batch yield...")

    pdf_paced_res = {}
    onnx_paced_res = {}

    def pdf_paced_task_wrapper():
        t, tp = run_foreground_pdf_extraction(pages)
        pdf_paced_res["time"] = t
        pdf_paced_res["tp"] = tp

    def onnx_paced_wrapper():
        t = run_background_onnx_worker(num_batches=20, batch_size=64, inter_batch_sleep=0.015)
        onnx_paced_res["time"] = t

    t_start_paced = time.perf_counter()
    t1_p = threading.Thread(target=pdf_paced_task_wrapper)
    t2_p = threading.Thread(target=onnx_paced_wrapper)

    t1_p.start()
    t2_p.start()
    t1_p.join()
    t2_p.join()
    total_paced_time = time.perf_counter() - t_start_paced

    pdf_recovery = ((pdf_res["time"] - pdf_paced_res["time"]) / pdf_res["time"]) * 100
    print(f"Foreground PDF Extraction Time  : {pdf_paced_res['time']:.3f}s ({pdf_paced_res['tp']:.1f} p/s) [{pdf_recovery:.1f}% Recovery]")
    print(f"Background ONNX Embedding Time  : {onnx_paced_res['time']:.3f}s")
    print(f"Total Wall-Clock Time           : {total_paced_time:.3f}s")

    print("\n" + "=" * 85)
    print("CONCURRENCY BENCHMARK SUMMARY & SYSTEM TRADE-OFF MATRIX")
    print("=" * 85)
    print(f"{'Execution Mode':<36} | {'PDF Time (FG)':<14} | {'PDF Throughput':<16} | {'ONNX Time (BG)':<14}")
    print("-" * 85)
    print(f"{'Isolated PDF Baseline':<36} | {pdf_iso_time:<12.3f}s | {pdf_iso_tp:<14.1f}p/s | {'-':<14}")
    print(f"{'Ungated Concurrency (Contention)':<36} | {pdf_res['time']:<12.3f}s | {pdf_res['tp']:<14.1f}p/s | {onnx_res['time']:<12.3f}s")
    print(f"{'Paced Background Queue (15ms Yield)':<36} | {pdf_paced_res['time']:<12.3f}s | {pdf_paced_res['tp']:<14.1f}p/s | {onnx_paced_res['time']:<12.3f}s")
    print("-" * 85)

if __name__ == '__main__':
    main()
