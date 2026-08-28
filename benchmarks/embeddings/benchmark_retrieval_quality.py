#!/usr/bin/env python3
"""
Scientific RAG Retrieval Accuracy & Quality Benchmark across Chunk Sizes.
Extracts real chapters from the textbook and runs dynamic semantic needle-in-a-haystack
evaluation on concepts that are ACTUALLY present in the selected pages.
"""

import os
import sys
import re
import time
import argparse
import numpy as np
from typing import List, Dict, Any, Tuple

# Ensure UTF-8 output on Windows
try:
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")
    if hasattr(sys.stderr, "reconfigure"):
        sys.stderr.reconfigure(encoding="utf-8")
except Exception:
    pass


def extract_pdf_pages(pdf_path: str, start_page: int = 1, end_page: int = 100) -> List[Tuple[int, str]]:
    """Extracts raw text per page from PDF."""
    try:
        import pypdf
        reader = pypdf.PdfReader(pdf_path)
        total = len(reader.pages)
        min_p = max(0, start_page - 1)
        max_p = min(total, end_page)
        pages = []
        for i in range(min_p, max_p):
            t = reader.pages[i].extract_text() or ""
            if len(t.strip()) > 30:
                pages.append((i + 1, t))
        return pages
    except ImportError:
        import fitz
        doc = fitz.open(pdf_path)
        total = len(doc)
        min_p = max(0, start_page - 1)
        max_p = min(total, end_page)
        pages = []
        for i in range(min_p, max_p):
            t = doc[i].get_text()
            if len(t.strip()) > 30:
                pages.append((i + 1, t))
        return pages


def slice_into_chunks(raw_text: str, target_words: int) -> List[str]:
    words = raw_text.split()
    chunks = []
    for i in range(0, len(words), target_words):
        chunk_words = words[i : i + target_words]
        if len(chunk_words) >= 30:
            chunks.append(" ".join(chunk_words))
    return chunks


def mean_pooling(token_embeddings: np.ndarray, attention_mask: np.ndarray) -> np.ndarray:
    input_mask_expanded = np.expand_dims(attention_mask, axis=-1).astype(np.float32)
    sum_embeddings = np.sum(token_embeddings * input_mask_expanded, axis=1)
    sum_mask = np.clip(input_mask_expanded.sum(axis=1), a_min=1e-9, a_max=None)
    return sum_embeddings / sum_mask


def normalize_l2(vector: np.ndarray) -> np.ndarray:
    norm = np.linalg.norm(vector, axis=-1, keepdims=True)
    norm = np.where(norm == 0, 1e-12, norm)
    return vector / norm


def embed_texts(texts: List[str], tok, session, batch_size: int = 8) -> np.ndarray:
    all_embeddings = []
    for i in range(0, len(texts), batch_size):
        batch = texts[i : i + batch_size]
        encodings = tok.encode_batch(batch)
        max_batch_tokens = max(len(e.ids) for e in encodings)

        input_ids, attention_mask, token_type_ids = [], [], []
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

        outputs = session.run(None, feed_dict)
        raw = outputs[0]
        if len(raw.shape) == 3:
            pooled = mean_pooling(raw, np.array(attention_mask, dtype=np.int64))
        else:
            pooled = raw
        normed = normalize_l2(pooled)
        all_embeddings.append(normed)

    return np.vstack(all_embeddings)


def generate_factual_probes_from_corpus(raw_text: str, max_probes: int = 12) -> List[Dict[str, Any]]:
    """Generates ground-truth test probes dynamically from key sentences in the actual text."""
    # Find informative candidate sentences
    sentences = re.split(r'(?<=[.!?])\s+', raw_text)
    candidates = []
    
    for s in sentences:
        s_clean = s.strip()
        words = s_clean.split()
        if 15 <= len(words) <= 40:
            # Exclude copyright, TOC, page headers
            if not any(x in s_clean.lower() for x in ["copyright", "isbn", "table of contents", "index", "published by"]):
                candidates.append(s_clean)

    # Select spread of candidates across the pages
    if not candidates:
        return []
        
    step = max(1, len(candidates) // max_probes)
    selected_sentences = [candidates[i] for i in range(0, min(len(candidates), max_probes * step), step)][:max_probes]
    
    probes = []
    for sent in selected_sentences:
        # Create natural query from sentence key phrases
        words = [w.strip(".,;:\"'()[]{}") for w in sent.split() if len(w) > 3]
        key_terms = [w for w in words if w.isalpha() and w.lower() not in {"this", "that", "with", "from", "have", "they", "will", "your", "when", "there"}][:5]
        if len(key_terms) >= 3:
            query = "Explain " + " ".join(key_terms[:4]) + " concept"
            probes.append({
                "query": query,
                "ground_truth_snippet": " ".join(words[:12]),
                "key_terms": [k.lower() for k in key_terms]
            })

    return probes


def evaluate_retrieval(chunks: List[str], query_vectors: np.ndarray, chunk_vectors: np.ndarray, probes: List[Dict[str, Any]]) -> Dict[str, Any]:
    hit_1 = 0
    hit_3 = 0
    reciprocal_ranks = []
    signal_noise_gaps = []

    sim_matrix = np.dot(query_vectors, chunk_vectors.T)

    for q_idx, probe in enumerate(probes):
        keywords = probe["key_terms"]
        snippet = probe["ground_truth_snippet"].lower()
        scores = sim_matrix[q_idx]
        ranked_chunk_indices = np.argsort(scores)[::-1]

        chunk_is_relevant = []
        for idx in ranked_chunk_indices:
            text_lower = chunks[idx].lower()
            # Match if ground truth snippet or >=2 key terms match
            matches = sum(1 for kw in keywords if kw in text_lower)
            is_match = (snippet in text_lower) or (matches >= len(keywords) - 1)
            chunk_is_relevant.append(is_match)

        if chunk_is_relevant and chunk_is_relevant[0]:
            hit_1 += 1
        if any(chunk_is_relevant[:3]):
            hit_3 += 1

        first_hit_rank = None
        for r, is_rel in enumerate(chunk_is_relevant):
            if is_rel:
                first_hit_rank = r + 1
                break

        if first_hit_rank is not None:
            reciprocal_ranks.append(1.0 / first_hit_rank)
        else:
            reciprocal_ranks.append(0.0)

        rel_scores = [scores[ranked_chunk_indices[i]] for i, is_rel in enumerate(chunk_is_relevant) if is_rel]
        non_rel_scores = [scores[ranked_chunk_indices[i]] for i, is_rel in enumerate(chunk_is_relevant) if not is_rel]

        if rel_scores and non_rel_scores:
            gap = max(rel_scores) - max(non_rel_scores)
            signal_noise_gaps.append(gap)

    num_queries = len(probes) if probes else 1
    return {
        "hit_at_1": (hit_1 / num_queries) * 100,
        "hit_at_3": (hit_3 / num_queries) * 100,
        "mrr": float(np.mean(reciprocal_ranks)) if reciprocal_ranks else 0.0,
        "cosine_margin": float(np.mean(signal_noise_gaps)) if signal_noise_gaps else 0.0
    }


def run_benchmark(pdf_path: str, start_page: int, end_page: int):
    try:
        from tokenizers import Tokenizer
        import onnxruntime as ort
    except ImportError:
        print("Run with: uv run --with pypdf --with onnxruntime --with tokenizers --with numpy python benchmarks/embeddings/benchmark_retrieval_quality.py")
        sys.exit(1)

    root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
    tokenizer_path = os.path.join(root_dir, "asset", "tokenizer.json")
    model_path = os.path.join(root_dir, "asset", "model_int8.onnx")

    print("=" * 100)
    print(f"RAG RETRIEVAL ACCURACY & QUALITY BENCHMARK (Pages {start_page} to {end_page})")
    print(f"Dataset      : {pdf_path}")
    print(f"Model        : asset/model_int8.onnx (768-d Nomic INT8)")
    print("=" * 100)

    print("\n1. Extracting text from book pages ...", end="", flush=True)
    page_tuples = extract_pdf_pages(pdf_path, start_page=start_page, end_page=end_page)
    raw_text = " ".join([p[1] for p in page_tuples])
    total_words = len(raw_text.split())
    print(f" Extracted {total_words:,} words across {len(page_tuples)} pages.")

    print("2. Generating ground-truth probe questions from actual chapter content ...", end="", flush=True)
    probes = generate_factual_probes_from_corpus(raw_text, max_probes=10)
    print(f" Generated {len(probes)} test probes.")

    if not probes:
        print("Error: Could not generate test probes from page range. Try a larger page range.")
        sys.exit(1)

    print("3. Loading ONNX Embedding Model ...", end="", flush=True)
    tok = Tokenizer.from_file(tokenizer_path)
    tok.enable_truncation(max_length=2048)

    session_opts = ort.SessionOptions()
    session_opts.intra_op_num_threads = os.cpu_count() or 4
    session = ort.InferenceSession(model_path, sess_options=session_opts)
    print(" Loaded successfully.")

    # Embed Queries
    query_texts = [f"search_query: {p['query']}" for p in probes]
    query_vectors = embed_texts(query_texts, tok, session, batch_size=4)

    sweep_sizes = [100, 200, 400, 600, 800, 1000, 1200]
    results = []

    print("\n" + "=" * 100)
    print(f"{'Chunk Size':<12} | {'Chunks':<8} | {'Hit@1 (%)':<11} | {'Hit@3 (%)':<11} | {'MRR':<8} | {'Cosine Margin (Δ)':<18} | {'Assessment'}")
    print("-" * 100)

    for sz in sweep_sizes:
        chunks = slice_into_chunks(raw_text, sz)
        chunk_prefixed = [f"search_document: {c}" for c in chunks]
        chunk_vectors = embed_texts(chunk_prefixed, tok, session, batch_size=4)

        metrics = evaluate_retrieval(chunks, query_vectors, chunk_vectors, probes)
        
        assessment = "🏆 Excellent" if metrics["hit_at_1"] >= 80 and metrics["cosine_margin"] > 0 else (
            "Good" if metrics["hit_at_3"] >= 70 else "Lower Discrimination"
        )
        
        results.append({
            "size": sz,
            "chunks": len(chunks),
            **metrics,
            "assessment": assessment
        })

        print(f"{sz:<12} | {len(chunks):<8} | {metrics['hit_at_1']:>9.1f}% | {metrics['hit_at_3']:>9.1f}% | {metrics['mrr']:>6.3f} | {metrics['cosine_margin']:>16.4f}   | {assessment}")

    print("=" * 100)
    best_mrr = max(results, key=lambda x: (x["hit_at_1"], x["mrr"], x["cosine_margin"]))
    print(f"\n🎯 EMPIRICAL SWEET SPOT FOR THIS TEXTBOOK: {best_mrr['size']} words")
    print(f"  • Hit@1 Accuracy  : {best_mrr['hit_at_1']:.1f}%")
    print(f"  • Hit@3 Accuracy  : {best_mrr['hit_at_3']:.1f}%")
    print(f"  • Mean Rank (MRR) : {best_mrr['mrr']:.3f}")
    print(f"  • Cosine Margin   : +{best_mrr['cosine_margin']:.4f}")


def main():
    root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
    default_pdf = os.path.join(root_dir, "learning go.pdf")

    parser = argparse.ArgumentParser(description="Dynamic RAG Retrieval Quality Benchmark")
    parser.add_argument("--pdf", default=default_pdf, help="Path to textbook PDF")
    parser.add_argument("--start-page", type=int, default=30, help="Start page number (skips preface/TOC)")
    parser.add_argument("--end-page", type=int, default=90, help="End page number")
    args = parser.parse_args()

    run_benchmark(pdf_path=args.pdf, start_page=args.start_page, end_page=args.end_page)


if __name__ == "__main__":
    main()
