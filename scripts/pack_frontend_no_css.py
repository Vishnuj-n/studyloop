#!/usr/bin/env python3
"""
Packs frontend source files for LLM context without CSS.
Strips:
  1. All `<style>...</style>` blocks inside `.vue` files.
  2. All external .css / .scss files.
  3. Binary files (fonts, images).
  4. Test files (*.spec.ts, *.test.ts, *.spec.js, *.test.js).
Produces a clean XML format similar to repomix.
"""

import os
import re
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parent.parent
FRONTEND_SRC = ROOT_DIR / "frontend" / "src"
OUTPUT_FILE = ROOT_DIR / "repomix-frontend-no-css.xml"

STYLE_BLOCK_RE = re.compile(r"<style\b[^>]*>.*?</style>", re.DOTALL | re.IGNORECASE)

EXCLUDED_EXTENSIONS = {
    ".css", ".scss", ".sass", ".less",
    ".svg", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico",
    ".woff", ".woff2", ".ttf", ".eot", ".otf"
}

def should_process(file_path: Path) -> bool:
    name = file_path.name.lower()
    if any(name.endswith(suffix) for suffix in [".spec.ts", ".test.ts", ".spec.js", ".test.js"]):
        return False
    if file_path.suffix.lower() in EXCLUDED_EXTENSIONS:
        return False
    return True

def clean_vue_content(content: str) -> str:
    cleaned = STYLE_BLOCK_RE.sub("<!-- <style> omitted for brevity -->", content)
    # clean extra blank lines
    lines = [l for l in cleaned.splitlines() if l.strip()]
    return "\n".join(lines)

def clean_code_content(content: str) -> str:
    lines = [l for l in content.splitlines() if l.strip()]
    return "\n".join(lines)

def main():
    files_processed = 0
    total_chars = 0

    print("Gathering frontend files without CSS...")
    with open(OUTPUT_FILE, "w", encoding="utf-8") as out:
        out.write("This file is a packed representation of the frontend application source code without CSS styling.\n")
        out.write("<repository>\n")

        for file_path in sorted(FRONTEND_SRC.rglob("*")):
            if not file_path.is_file():
                continue
            if not should_process(file_path):
                continue

            rel_path = file_path.relative_to(ROOT_DIR).as_posix()
            try:
                content = file_path.read_text(encoding="utf-8", errors="replace")
            except Exception as e:
                print(f"Error reading {rel_path}: {e}")
                continue

            if file_path.suffix.lower() == ".vue":
                content = clean_vue_content(content)
            else:
                content = clean_code_content(content)

            out.write(f'  <file path="{rel_path}">\n')
            out.write(f'<![CDATA[\n{content}\n]]>\n')
            out.write("  </file>\n")

            files_processed += 1
            total_chars += len(content)

        out.write("</repository>\n")

    print(f"Done! Packaged {files_processed} files into {OUTPUT_FILE.name}")
    print(f"Total size: {OUTPUT_FILE.stat().st_size:,} bytes (~{int(total_chars / 4):,} estimated tokens)")

if __name__ == "__main__":
    main()
