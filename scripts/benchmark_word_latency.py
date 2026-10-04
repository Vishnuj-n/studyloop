import os
import sys
import time
import json
import warnings

warnings.filterwarnings("ignore")

# Add the prompt_compressor extension directory to path
EXT_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "extensions", "prompt_compressor")
sys.path.insert(0, EXT_DIR)

from compress import get_compressor, compress_chunk_text

BASE_ACADEMIC_PARAGRAPHS = [
    (
        "Machine learning is a subset of artificial intelligence focused on building systems that learn from data. "
        "Supervised learning algorithms infer a function from labeled training data consisting of input-output pairs. "
        "In contrast, unsupervised learning discovers hidden patterns or intrinsic structures within unlabelled datasets. "
        "Reinforcement learning operates on an agent-environment paradigm where actions yield rewards or penalties. "
        "Deep neural networks utilize multiple processing layers composed of linear transformations and non-linear activation functions. "
    ),
    (
        "Cellular biology investigates the structural and functional units of all living organisms. "
        "Mitochondria generate the majority of cellular adenosine triphosphate through oxidative phosphorylation. "
        "The endoplasmic reticulum facilitates protein synthesis and lipid metabolism across eukaryotic structures. "
        "Ribosomes translate messenger RNA transcripts into polypeptide sequences with high thermodynamic fidelity. "
        "Membrane potential is maintained by active ion transport via sodium-potassium adenosine triphosphatase pumps. "
    ),
    (
        "Macroeconomics evaluates economy-wide phenomena such as inflation, price indices, gross domestic product, and unemployment. "
        "Central banks manipulate short-term interest rates and open market operations to stabilize monetary expansion. "
        "Fiscal policy adjustments directly influence aggregate demand through government expenditure and taxation frameworks. "
        "Supply-side economics emphasizes deregulation, tax rate reductions, and free trade to stimulate capital investment. "
        "Equilibrium prices occur when quantity supplied strictly equals quantity demanded within competitive open markets. "
    ),
    (
        "Quantum computing leverages quantum mechanical principles such as superposition, entanglement, and interference. "
        "Unlike classical bits constrained to binary states, qubits represent linear combinations of orthogonal basis states. "
        "Quantum gate operations perform unitary transformations on state vectors within complex Hilbert spaces. "
        "Decoherence caused by thermal fluctuations remains a fundamental engineering hurdle in fault-tolerant quantum hardware. "
        "Shor's algorithm efficiently computes prime factorizations in polynomial time, posing challenges to RSA encryption. "
    )
]

def generate_text(target_words: int) -> str:
    """Generates synthetic educational text of exact word count."""
    words = []
    idx = 0
    while len(words) < target_words:
        para = BASE_ACADEMIC_PARAGRAPHS[idx % len(BASE_ACADEMIC_PARAGRAPHS)]
        words.extend(para.split())
        idx += 1
    return " ".join(words[:target_words])

def run_benchmarks():
    print("=========================================================================")
    print("  STUDYLOOP LIGHT BERT (LLMLingua-2) COMPRESSION BENCHMARK")
    print("=========================================================================")
    print("Model: microsoft/llmlingua-2-bert-base-multilingual-cased-meetingbank")
    print("Device: CPU (Single Thread / Multi-Core Native Inference)")
    print("Target Compression Rate: 0.80 (20% reduction preserving vital semantic tokens)\n")

    # 1. Warm-up & Model Load
    print("[1/3] Loading model & executing 1st-run warm-up...")
    t_warm_start = time.time()
    compressor = get_compressor()
    _ = compress_chunk_text("Model initialization warm up test text.", rate=0.8)
    t_warm_end = time.time()
    print(f"      Model load + warm-up time: {(t_warm_end - t_warm_start):.3f}s\n")

    # Word counts to test
    word_targets = [2000, 3000, 5000]
    results = []

    print("[2/3] Benchmarking Compression Latencies:")
    print("-------------------------------------------------------------------------")
    print(f"{'Word Count':<12} | {'Origin Tok':<10} | {'Comp Tok':<10} | {'Saved %':<9} | {'Latency (s)':<12} | {'Throughput':<14}")
    print("-------------------------------------------------------------------------")

    for count in word_targets:
        text = generate_text(count)
        
        # Measure latency
        t0 = time.perf_counter()
        res = compress_chunk_text(text, rate=0.80)
        t1 = time.perf_counter()
        
        latency_sec = t1 - t0
        words_per_sec = count / latency_sec if latency_sec > 0 else 0
        ms_per_word = (latency_sec * 1000) / count if count > 0 else 0

        orig_tokens = res.get("origin_tokens", count)
        comp_tokens = res.get("compressed_tokens", 0)
        saving = res.get("saving", "N/A")

        results.append({
            "words": count,
            "origin_tokens": orig_tokens,
            "compressed_tokens": comp_tokens,
            "saving": saving,
            "latency_seconds": round(latency_sec, 3),
            "words_per_second": round(words_per_sec, 1),
            "ms_per_word": round(ms_per_word, 2)
        })

        print(f"{count:<12} | {orig_tokens:<10} | {comp_tokens:<10} | {saving:<9} | {latency_sec:<11.3f}s | {words_per_sec:<8.1f} w/s")

    print("-------------------------------------------------------------------------\n")

    # 3. Batch Chunk Simulation (e.g. 500 words/chunk as used in the reader pipeline)
    print("[3/3] Simulating Chunker Pipeline (500 words per chunk in parallel/sequential):")
    print("-------------------------------------------------------------------------")
    for count in word_targets:
        chunk_size = 500
        full_doc = generate_text(count)
        words = full_doc.split()
        chunks = [" ".join(words[i:i+chunk_size]) for i in range(0, len(words), chunk_size)]
        num_chunks = len(chunks)
        
        t0 = time.perf_counter()
        for ch in chunks:
            _ = compress_chunk_text(ch, rate=0.80)
        t1 = time.perf_counter()
        
        chunk_latency = t1 - t0
        print(f"  • {count} words across {num_chunks} chunks ({chunk_size} words/chunk): {chunk_latency:.3f}s total ({count/chunk_latency:.1f} words/sec)")

    print("\n=========================================================================")
    print("  BENCHMARK COMPLETE")
    print("=========================================================================")

if __name__ == "__main__":
    run_benchmarks()
