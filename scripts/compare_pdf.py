#!/usr/bin/env python3
"""
PDF Ingestion 2-Way Benchmark: Standard Linear Text vs. Fast PDF (PyMuPDF4LLM)

Usage:
    python scripts/compare_pdf.py "learning go.pdf" --start-page 25 --end-page 30 --output dev_data/comparison.md
"""

import os
import sys
import json
import argparse
import subprocess
import time
from pathlib import Path

# Ensure UTF-8 output encoding
try:
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")
    if hasattr(sys.stderr, "reconfigure"):
        sys.stderr.reconfigure(encoding="utf-8")
except Exception:
    pass


def extract_standard_text(pdf_path: str, start_page: int = 0, end_page: int = 0) -> dict:
    """
    Standard linear text extraction (simulates Go ledongthuc/pdf baseline).
    """
    start_time = time.time()
    extracted_text = ""
    page_count = 0

    # Use pypdf for standard linear text stream extraction
    try:
        import pypdf
        reader = pypdf.PdfReader(pdf_path)
        total = len(reader.pages)
        min_p = max(0, start_page - 1) if start_page > 0 else 0
        max_p = min(total, end_page) if end_page > 0 else total
        page_count = max_p - min_p
        for idx in range(min_p, max_p):
            page_text = reader.pages[idx].extract_text() or ""
            extracted_text += f"\n--- Page {idx + 1} ---\n" + page_text
    except ImportError:
        extracted_text = "[Standard text extractor: pypdf not found]"

    elapsed = time.time() - start_time
    clean_text = extracted_text.strip()
    words = len(clean_text.split())
    has_tables = "|" in clean_text and "---" in clean_text
    has_headers = any(line.strip().startswith(("#", "##", "###")) for line in clean_text.splitlines())
    has_code_blocks = "```" in clean_text

    return {
        "engine": "pypdf / Standard Linear Text",
        "elapsed_seconds": round(elapsed, 4),
        "page_count": page_count,
        "word_count": words,
        "char_count": len(clean_text),
        "table_structure": "❌ Broken / Interleaved" if not has_tables else "✅ Detected",
        "headers": "❌ Flat / Uppercase" if not has_headers else "✅ Semantic (#)",
        "code_blocks": "❌ Unformatted lines" if not has_code_blocks else "✅ Preserved (```)",
        "raw_text": clean_text,
        "status": "success",
    }


def extract_fast_pdf(pdf_path: str, start_page: int = 0, end_page: int = 0) -> dict:
    """
    Fast structured extraction using extensions/fast_pdf (PyMuPDF4LLM).
    """
    start_time = time.time()
    ext_dir = Path(__file__).resolve().parent.parent / "extensions" / "fast_pdf"
    script = ext_dir / "ingest.py"
    
    cmd = ["uv", "run", "--directory", str(ext_dir), "python", str(script), str(Path(pdf_path).resolve())]
    if start_page > 0:
        cmd.extend(["--start-page", str(start_page)])
    if end_page > 0:
        cmd.extend(["--end-page", str(end_page)])

    try:
        proc = subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8")
        elapsed = time.time() - start_time
        if proc.returncode == 0:
            lines = [l for l in proc.stdout.splitlines() if l.strip().startswith("{")]
            if lines:
                data = json.loads(lines[-1])
                md = data.get("markdown", "").strip()
                has_tables = "|" in md and ("-|-" in md or "|---" in md)
                has_headers = any(l.strip().startswith(("#", "##", "###")) for l in md.splitlines())
                has_code = "```" in md
                return {
                    "engine": "Fast PDF (PyMuPDF4LLM)",
                    "elapsed_seconds": round(elapsed, 3),
                    "page_count": data.get("page_count", end_page - start_page + 1 if end_page else 1),
                    "word_count": data.get("word_count", len(md.split())),
                    "char_count": len(md),
                    "table_structure": "✅ Markdown Tables" if has_tables else "None found",
                    "headers": "✅ Semantic (#, ##)" if has_headers else "None found",
                    "code_blocks": "✅ Fenced Code (```)" if has_code else "None found",
                    "raw_text": md,
                    "status": "success",
                }
        return {
            "engine": "Fast PDF (PyMuPDF4LLM)",
            "elapsed_seconds": round(elapsed, 3),
            "status": "error",
            "error": proc.stderr or proc.stdout,
        }
    except Exception as e:
        return {"engine": "Fast PDF (PyMuPDF4LLM)", "status": "error", "error": str(e)}


def build_markdown_report(results: list, file_name: str, pages_label: str) -> str:
    md = []
    md.append(f"# PDF Extraction 2-Way Benchmark Report")
    md.append(f"**Target Document**: `{file_name}` | **Scope**: {pages_label}\n")
    md.append("## 1. Metric Comparison Matrix\n")
    
    headers = ["Metric / Capability", "Go / Linear Stream", "Fast PDF (PyMuPDF4LLM)"]
    md.append("| " + " | ".join(headers) + " |")
    md.append("| " + " | ".join(["---"] * len(headers)) + " |")
    
    r_std, r_fast = results[0], results[1]
    
    md.append(f"| **Execution Time** | `{r_std.get('elapsed_seconds', 'N/A')}s` | `{r_fast.get('elapsed_seconds', 'N/A')}s` |")
    md.append(f"| **Pages Processed** | {r_std.get('page_count', 'N/A')} | {r_fast.get('page_count', 'N/A')} |")
    md.append(f"| **Word Count** | {r_std.get('word_count', 'N/A')} | {r_fast.get('word_count', 'N/A')} |")
    md.append(f"| **Character Count** | {r_std.get('char_count', 'N/A')} | {r_fast.get('char_count', 'N/A')} |")
    md.append(f"| **Heading Hierarchy** | {r_std.get('headers', 'N/A')} | {r_fast.get('headers', 'N/A')} |")
    md.append(f"| **Table Reconstruction** | {r_std.get('table_structure', 'N/A')} | {r_fast.get('table_structure', 'N/A')} |")
    md.append(f"| **Code Indentation & Fences** | {r_std.get('code_blocks', 'N/A')} | {r_fast.get('code_blocks', 'N/A')} |")
    md.append("")
    md.append("## 2. Output Snippet Comparison\n")

    for res in results:
        md.append(f"### Engine: {res['engine']}")
        if res.get("status") == "success":
            snippet = res.get("raw_text", "")[:800].strip()
            md.append("```markdown")
            md.append(snippet if snippet else "[Empty output]")
            md.append("```\n")
        else:
            md.append(f"> ⚠️ **Error / Unavailable**: {res.get('error', 'Unknown')}\n")

    return "\n".join(md)


def main():
    parser = argparse.ArgumentParser(description="2-Way Benchmark: pypdf Standard vs Fast PDF (PyMuPDF4LLM)")
    parser.add_argument("pdf_path", nargs="?", default="learning go.pdf", help="Path to PDF")
    parser.add_argument("--start-page", type=int, default=25, help="Start page number (1-indexed)")
    parser.add_argument("--end-page", type=int, default=30, help="End page number (1-indexed)")
    parser.add_argument("--output", "-o", default="dev_data/comparison.md", help="Save markdown report to file")

    args = parser.parse_args()
    pdf_file = Path(args.pdf_path)
    if not pdf_file.exists():
        print(f"Error: File not found: {args.pdf_path}")
        sys.exit(1)

    pages_label = f"Pages {args.start_page} to {args.end_page}" if args.start_page and args.end_page else "Full Document"
    print(f"[*] Running 2-way PDF benchmark on '{pdf_file.name}' ({pages_label})...")

    print(" [1/2] Running pypdf / Standard Linear Extraction...")
    res_std = extract_standard_text(str(pdf_file), args.start_page, args.end_page)

    print(" [2/2] Running Fast PDF (PyMuPDF4LLM)...")
    res_fast = extract_fast_pdf(str(pdf_file), args.start_page, args.end_page)

    results = [res_std, res_fast]
    report_md = build_markdown_report(results, pdf_file.name, pages_label)

    print("\n" + report_md + "\n")

    if args.output:
        out_path = Path(args.output)
        out_path.parent.mkdir(parents=True, exist_ok=True)
        out_path.write_text(report_md, encoding="utf-8")
        print(f"[+] Markdown report saved to: {out_path.resolve()}")


if __name__ == "__main__":
    main()

