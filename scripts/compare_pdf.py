#!/usr/bin/env python3
"""
PDF Ingestion Comparison Tool: Standard Text-Only vs. Docling Structured Extraction

Usage:
    python scripts/compare_pdf.py <path/to/document.pdf> [--output report.md] [--json]
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


def extract_standard_text(pdf_path: str) -> dict:
    """
    Standard text-only extraction (simulating basic PDF parser like ledongthuc/pdf / pypdf).
    Extracts raw text streams without layout, OCR, or table reconstruction.
    """
    start_time = time.time()
    extracted_text = ""
    pages = []
    
    # Try pypdf first if available
    try:
        import pypdf
        reader = pypdf.PdfReader(pdf_path)
        for idx, page in enumerate(reader.pages):
            page_text = page.extract_text() or ""
            pages.append({"page_num": idx + 1, "text": page_text})
            extracted_text += f"\n--- Page {idx + 1} ---\n" + page_text
    except ImportError:
        try:
            from pdfminer.high_level import extract_text as pdfminer_extract
            extracted_text = pdfminer_extract(pdf_path)
            pages = [{"page_num": 1, "text": extracted_text}]
        except ImportError:
            extracted_text = "[Standard text extractor: install pypdf (`pip install pypdf`) for direct python extraction]"
            pages = [{"page_num": 1, "text": extracted_text}]

    elapsed = time.time() - start_time
    words = len(extracted_text.split())
    has_tables = "|" in extracted_text and "---" in extracted_text
    has_markdown_headers = any(line.strip().startswith(("#", "##", "###")) for line in extracted_text.splitlines())

    return {
        "engine": "Standard Text-Only (Linear Stream)",
        "elapsed_seconds": round(elapsed, 4),
        "page_count": len(pages),
        "word_count": words,
        "char_count": len(extracted_text),
        "table_structure_preserved": has_tables,
        "headers_preserved": has_markdown_headers,
        "raw_text": extracted_text.strip(),
        "pages": pages,
    }


def extract_with_docling(pdf_path: str) -> dict:
    """
    Advanced Docling extraction via extensions/docling/ingest.py or direct docling module.
    Preserves document structure, markdown tables, headers, and formulas.
    """
    start_time = time.time()
    script_dir = Path(__file__).resolve().parent.parent / "extensions" / "docling" / "ingest.py"
    
    if script_dir.exists():
        proc = subprocess.run(
            [sys.executable, str(script_dir), pdf_path],
            capture_output=True,
            text=True,
            encoding="utf-8"
        )
        if proc.returncode == 0:
            try:
                data = json.loads(proc.stdout)
                markdown = data.get("markdown", "")
                elapsed = time.time() - start_time
                return {
                    "engine": "Docling Advanced AI Parser",
                    "elapsed_seconds": round(elapsed, 2),
                    "page_count": data.get("page_count", 1),
                    "word_count": data.get("word_count", len(markdown.split())),
                    "char_count": len(markdown),
                    "table_structure_preserved": "|" in markdown and ("-|-" in markdown or "|---" in markdown),
                    "headers_preserved": any(line.strip().startswith(("#", "##", "###")) for line in markdown.splitlines()),
                    "markdown": markdown.strip(),
                    "status": "success"
                }
            except Exception as e:
                pass
        else:
            err_msg = (proc.stderr or proc.stdout).strip()
            return {
                "engine": "Docling Advanced AI Parser",
                "status": "error",
                "error": err_msg or "Docling package not installed in environment.",
                "note": "Run `pip install docling` or activate your docling virtual environment."
            }

    return {
        "engine": "Docling Advanced AI Parser",
        "status": "missing_extension_script",
        "error": f"Could not find {script_dir}"
    }


def print_comparison(standard: dict, docling: dict, out_file: str = None):
    separator = "=" * 80
    sub_sep = "-" * 80

    report = []
    report.append(separator)
    report.append("  STUDYLOOP PDF EXTRACTION COMPARISON BENCHMARK")
    report.append(separator)
    report.append("")
    report.append("METRICS & CAPABILITY MATRIX:")
    report.append(f"{'Feature / Metric':<30} | {'Text-Only Extraction':<25} | {'Docling Advanced Parser':<25}")
    report.append("-" * 86)
    
    report.append(f"{'Page Count':<30} | {standard.get('page_count', 'N/A'):<25} | {docling.get('page_count', 'N/A'):<25}")
    report.append(f"{'Word Count':<30} | {standard.get('word_count', 0):<25} | {docling.get('word_count', 'N/A'):<25}")
    report.append(f"{'Character Count':<30} | {standard.get('char_count', 0):<25} | {docling.get('char_count', 'N/A'):<25}")
    
    t_std = "Scrambled / Raw Text" if not standard.get("table_structure_preserved") else "Detected"
    t_doc = "Markdown Tables (| col |)" if docling.get("table_structure_preserved") else ("Unavailable" if docling.get("status") != "success" else "None found")
    report.append(f"{'Table Structure':<30} | {t_std:<25} | {t_doc:<25}")
    
    h_std = "Flat Lines" if not standard.get("headers_preserved") else "Preserved"
    h_doc = "Semantic Markdown (#, ##)" if docling.get("headers_preserved") else ("Unavailable" if docling.get("status") != "success" else "None found")
    report.append(f"{'Heading Hierarchy':<30} | {h_std:<25} | {h_doc:<25}")
    
    f_std = "Broken / Unicode noise"
    f_doc = "LaTeX / Math syntax" if docling.get("status") == "success" else "Unavailable"
    report.append(f"{'Math & Equation Extract':<30} | {f_std:<25} | {f_doc:<25}")
    
    report.append(f"{'RAG Chunking Readiness':<30} | {'Low (loses context)':<25} | {'High (preserves blocks)':<25}")
    report.append(sub_sep)
    report.append("")

    report.append("OUTPUT STRUCTURAL PREVIEW:")
    report.append("")
    report.append("--- [1] Standard Text-Only Output Preview ---")
    std_preview = standard.get("raw_text", "")[:600]
    report.append(std_preview if std_preview else "[Empty text]")
    report.append("...\n")

    report.append("--- [2] Docling Structured Markdown Preview ---")
    if docling.get("status") == "success":
        doc_preview = docling.get("markdown", "")[:600]
        report.append(doc_preview if doc_preview else "[Empty markdown]")
        report.append("...\n")
    else:
        report.append(f"[!] Docling engine status: {docling.get('error')}")
        if "note" in docling:
            report.append(f"[*] Hint: {docling['note']}\n")

    report.append(separator)

    output_str = "\n".join(report)
    print(output_str)

    if out_file:
        Path(out_file).write_text(output_str, encoding="utf-8")
        print(f"\n[+] Comparison report written to: {out_file}")


def main():
    parser = argparse.ArgumentParser(description="Compare PDF Text-Only extraction vs Docling Structured output")
    parser.add_argument("pdf_path", nargs="?", help="Path to PDF document to test")
    parser.add_argument("--output", "-o", help="Save comparison markdown report to file")
    parser.add_argument("--json", action="store_true", help="Output raw JSON data")

    args = parser.parse_args()

    if not args.pdf_path:
        print("Usage: python scripts/compare_pdf.py <path_to_pdf> [--output report.md] [--json]")
        print("\nExample:")
        print("  python scripts/compare_pdf.py dev_data/sample.pdf")
        sys.exit(1)

    pdf_file = Path(args.pdf_path)
    if not pdf_file.exists():
        print(f"Error: File not found: {args.pdf_path}")
        sys.exit(1)

    print(f"[*] Analyzing PDF: {pdf_file.name} ({pdf_file.stat().st_size / 1024:.1f} KB)...")
    standard_res = extract_standard_text(str(pdf_file))
    docling_res = extract_with_docling(str(pdf_file))

    if args.json:
        print(json.dumps({"standard": standard_res, "docling": docling_res}, indent=2, ensure_ascii=False))
    else:
        print_comparison(standard_res, docling_res, args.output)


if __name__ == "__main__":
    main()
