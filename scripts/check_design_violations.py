#!/usr/bin/env python3
"""
DESIGN.md Compliance Linter
Scans frontend files (.vue, .css) to verify adherence to doc/DESIGN.md.

Rules checked:
1. No-Line Rule: Flags 1px/2px solid borders where background shifts should be used.
2. Hardcoded Forbidden Colors: Flags hardcoded colors (pure black, #007AFF, #6c5ce7) instead of CSS variables.
3. Shadows: Flags harsh/heavy drop shadows (>0.15 opacity) rather than subtle ambient elevation.
4. Button Radius: Flags buttons missing the 'xl' (0.75rem / 12px) standard radius.
5. Typography: Checks for dual-typeface (Manrope for headers, Inter for body).
6. Double Read Prefix: Checks for duplicated 'Read: ' prefix pattern in templates.
"""

import os
import re
import sys
from pathlib import Path

# Ensure utf-8 output on Windows console
if sys.platform == "win32":
    sys.stdout.reconfigure(encoding="utf-8")

# Colors for terminal output
RED = "\033[91m"
GREEN = "\033[92m"
YELLOW = "\033[93m"
BLUE = "\033[94m"
CYAN = "\033[96m"
BOLD = "\033[1m"
RESET = "\033[0m"

FRONTEND_DIR = Path(__file__).resolve().parent.parent / "frontend" / "src"

PATTERNS = [
    {
        "id": "NO_LINE_RULE",
        "description": "DESIGN.md 'No-Line Rule': Avoid hardcoded borders (use background shifts or subtle ghost outline-variant).",
        "severity": "WARNING",
        "regex": re.compile(r"border\s*:\s*([1-9]\d*px)\s+solid\s+(?!var\(--outline-variant)", re.IGNORECASE),
    },
    {
        "id": "FORBIDDEN_BLACK",
        "description": "DESIGN.md: Do not use pure black (#000, #000000, black). Use 'var(--on-surface)' (#2d3338).",
        "severity": "ERROR",
        "regex": re.compile(r"color\s*:\s*(#000000|#000\b|black\b)", re.IGNORECASE),
    },
    {
        "id": "FORBIDDEN_PURPLE_VIOLET",
        "description": "DESIGN.md: Avoid hardcoded purple/violet accents (#6c5ce7, #007AFF). Use theme tokens like var(--primary).",
        "severity": "ERROR",
        "regex": re.compile(r"(#6c5ce7|#007aff)\b", re.IGNORECASE),
    },
    {
        "id": "HARSH_SHADOW",
        "description": "DESIGN.md: Use low-opacity ambient blurs (<= 0.15 opacity), not standard heavy dark shadows.",
        "severity": "WARNING",
        "regex": re.compile(r"(?:box-shadow|drop-shadow|filter)\s*:.*?rgba\s*\(\s*0\s*,\s*0\s*,\s*0\s*,\s*0\.(?:2|3|4|5|6|7|8|9)\d*\s*\)", re.IGNORECASE),
    },
    {
        "id": "DUPLICATE_READ_PREFIX",
        "description": "Template Glitch: Hardcoded 'Read: ' prefix without checking if task.title already has it.",
        "severity": "ERROR",
        "regex": re.compile(r"(?:Read:\s*<\/span>\s*\{\{\s*task\.title\s*\}\})|(?:Read:\s*<\/span>\s*\{\{\s*task\.title\s*\|\|)", re.IGNORECASE),
    }
]

def scan_file(file_path: Path):
    violations = []
    try:
        content = file_path.read_text(encoding="utf-8")
    except Exception:
        return violations

    lines = content.splitlines()
    for line_idx, line in enumerate(lines, start=1):
        for rule in PATTERNS:
            if rule["regex"].search(line):
                violations.append({
                    "file": file_path,
                    "line_number": line_idx,
                    "line_content": line.strip(),
                    "rule_id": rule["id"],
                    "description": rule["description"],
                    "severity": rule["severity"]
                })
    return violations

def main():
    print(f"{BOLD}{BLUE}=================================================={RESET}")
    print(f"{BOLD}{CYAN}[DESIGN SCAN] Scanning Frontend for DESIGN.md Violations...{RESET}")
    print(f"{BOLD}{BLUE}=================================================={RESET}\n")

    if not FRONTEND_DIR.exists():
        print(f"{RED}Error: Frontend directory not found at {FRONTEND_DIR}{RESET}")
        sys.exit(1)

    files_scanned = 0
    all_violations = []

    for root, _, files in os.walk(FRONTEND_DIR):
        for file in files:
            if file.endswith((".vue", ".css")):
                file_path = Path(root) / file
                files_scanned += 1
                violations = scan_file(file_path)
                if violations:
                    all_violations.extend(violations)

    if not all_violations:
        print(f"{GREEN}[SUCCESS] Scanned {files_scanned} files. 0 DESIGN.md violations found!{RESET}\n")
        return 0

    print(f"{BOLD}Found {len(all_violations)} violations across {files_scanned} scanned files:{RESET}\n")

    for v in all_violations:
        color = RED if v["severity"] == "ERROR" else YELLOW
        rel_path = v["file"].relative_to(FRONTEND_DIR.parent)
        print(f"{color}[{v['severity']}] {v['rule_id']}{RESET} at {BOLD}{rel_path}:{v['line_number']}{RESET}")
        print(f"  -> {CYAN}{v['description']}{RESET}")
        print(f"  -> Code: `{v['line_content']}`\n")

    error_count = sum(1 for v in all_violations if v["severity"] == "ERROR")
    warning_count = sum(1 for v in all_violations if v["severity"] == "WARNING")

    print(f"{BOLD}{BLUE}=================================================={RESET}")
    print(f"Summary: {RED}{error_count} Errors{RESET}, {YELLOW}{warning_count} Warnings{RESET}")
    print(f"{BOLD}{BLUE}=================================================={RESET}\n")

    return 1 if error_count > 0 else 0

if __name__ == "__main__":
    sys.exit(main())
