#!/usr/bin/env python3
"""
AI Slop & Overly Defensive Code Detector

Scans Go, TypeScript, JavaScript, Vue, and Python files to detect:
1. Overly defensive code (redundant nil/null checks, paranoid wrappers, swallowing catches, over-chaining)
2. AI slop (stating-the-obvious comments, LLM conversational remnants, parrot docstrings, over-abstracted wrappers)

Usage:
    python scripts/find_code_smells.py [options]
    python scripts/find_code_smells.py --category defensive
    python scripts/find_code_smells.py --category slop
    python scripts/find_code_smells.py --dir internal --json
"""

import os
import re
import sys
import json
import argparse
from typing import List, Dict, Any, Optional

# Ensure UTF-8 output on Windows consoles
if hasattr(sys.stdout, "reconfigure"):
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass

# ANSI Colors
CYAN = "\033[96m"
YELLOW = "\033[93m"
GREEN = "\033[92m"
RED = "\033[91m"
MAGENTA = "\033[95m"
BOLD = "\033[1m"
DIM = "\033[2m"
RESET = "\033[0m"

# Default directories and files to scan
DEFAULT_DIRS = ["internal", "frontend/src", "cmd", "scripts"]
DEFAULT_EXTENSIONS = {".go", ".ts", ".js", ".vue", ".py"}

EXCLUDE_DIRS = {
    "node_modules", "dist", "bin", "build", "assets", "asset",
    ".git", "__pycache__", "wailsjs", "dev_data", "tmp"
}

EXCLUDE_FILE_PATTERNS = [
    r"_test\.go$",
    r"\.test\.(js|ts)$",
    r"\.spec\.(js|ts)$",
    r"\.d\.ts$",
    r"wailsjs",
]


class SmellRule:
    def __init__(self, rule_id: str, name: str, category: str, severity: str, description: str, hint: str):
        self.rule_id = rule_id
        self.name = name
        self.category = category  # 'defensive' or 'slop'
        self.severity = severity  # 'warning', 'info'
        self.description = description
        self.hint = hint


# --------------------------------------------------------------------------------------
# Rules Definitions & Detection Engines
# --------------------------------------------------------------------------------------

# AI Comment Patterns (Stating the obvious)
OBVIOUS_COMMENT_PATTERNS = [
    (re.compile(r"^\s*//\s*(?:Package|Module)?\s*imports?\s*$", re.IGNORECASE), "Trivial 'imports' header comment"),
    (re.compile(r"^\s*//\s*(?:Initialize|Constructor|Init)\s+(?:component|service|struct|variables?)?\s*$", re.IGNORECASE), "Trivial 'initialize' header comment"),
    (re.compile(r"^\s*//\s*(?:Helper|Utility)\s+functions?\s*$", re.IGNORECASE), "Trivial 'helper functions' comment"),
    (re.compile(r"^\s*//\s*(?:Handle|Check for|Check)\s+errors?\s*$", re.IGNORECASE), "Trivial 'handle error' comment"),
    (re.compile(r"^\s*//\s*Check if (?:err|error|response|user|data) is (?:nil|null|defined|valid|empty)\s*$", re.IGNORECASE), "Parroting condition in comment"),
    (re.compile(r"^\s*//\s*Return (?:nil|null|true|false|response|result|data|error|err)(?: if .+| on .+)?\s*$", re.IGNORECASE), "Parroting return statement"),
    (re.compile(r"^\s*//\s*Set loading to (?:true|false)\s*$", re.IGNORECASE), "Stating obvious UI state assignment"),
    (re.compile(r"^\s*//\s*(?:Define|Declare)\s+(?:constants?|variables?|structs?|interfaces?|types?)\s*$", re.IGNORECASE), "Trivial declaration comment"),
    (re.compile(r"^\s*//\s*End of (?:function|class|file|component|struct)\s*$", re.IGNORECASE), "Unnecessary end-of-block marker comment"),
]

# AI Conversational remnants
AI_CONVERSATIONAL_PATTERNS = [
    (re.compile(r"//\s*(?:Here is the|Here's the|Note that in a production|In a production environment|In real implementation)", re.IGNORECASE), "LLM conversational boilerplate in comment"),
    (re.compile(r"//\s*(?:As requested|As per requirement|This ensures that|This is to prevent|Ensure that)\b", re.IGNORECASE), "LLM reasoning explanation in code comment"),
    (re.compile(r"//\s*(?:Step\s*\d+\s*:|Phase\s*\d+\s*:)", re.IGNORECASE), "LLM step-by-step tutorial comment in production code"),
    (re.compile(r"//\s*(?:In the future, (?:you can|we can)|TODO:\s*implement actual logic)", re.IGNORECASE), "LLM speculative future placeholder"),
]

# Parrot Docstrings (e.g. // FetchUser fetches the user)
PARROT_DOC_GO = re.compile(r"^\s*//\s*([A-Z]\w+)\s+(?:is|will|gets|sets|handles|fetches|creates|returns|checks)\s+(?:the\s+)?([a-z]\w*)\.?\s*$", re.IGNORECASE)

# Over-abstracted trivial one-liner wrappers
TRIVIAL_WRAPPER_JS = re.compile(
    r"^\s*(?:export\s+)?(?:const|function)\s+(is(?:Not)?(?:Null|Undefined|Empty|Zero|True|False|Nil|Blank|String|Number|Object))\b.*=\s*(?:\([^\)]*\)|[a-zA-Z_]+)\s*=>\s*.*(?:\=\=\=|\!\=\=|typeof)",
    re.IGNORECASE
)

# Go Defensive Overkill:
# 1. len(slice) > 0 check immediately before for ... range
GO_LEN_BEFORE_RANGE = re.compile(r"if\s+len\(\s*([a-zA-Z0-9_\.]+)\s*\)\s*>\s*0\s*\{\s*for\s+(?:_\s*,\s*)?[a-zA-Z0-9_]+\s*:=\s*range\s+\1\b")
# 2. if m != nil { delete(m, k) }
GO_NIL_BEFORE_DELETE = re.compile(r"if\s+([a-zA-Z0-9_\.]+)\s*!=\s*nil\s*\{\s*delete\(\s*\1\s*,")
# 3. if s != nil { s = append(s, ...) } or if s == nil { s = []... }
GO_NIL_BEFORE_APPEND = re.compile(r"if\s+([a-zA-Z0-9_\.]+)\s*==\s*nil\s*\{\s*\1\s*=\s*(?:make\(\s*\[\]|\[\])")

# JS / TS Defensive Overkill:
# 1. if (arr && Array.isArray(arr) && arr.length > 0)
JS_PARANOID_ARRAY_GUARD = re.compile(r"if\s*\(\s*([a-zA-Z0-9_\.]+)\s*&&\s*Array\.isArray\(\s*\1\s*\)\s*&&\s*\1\.length")
# 2. 4+ optional chains in a row: a?.b?.c?.d?.e
JS_EXCESSIVE_OPTIONAL_CHAIN = re.compile(r"\b[a-zA-Z0-9_]+(?:\?\.[a-zA-Z0-9_]+){4,}\b")
# 3. Chained fallbacks: a || b || c || d
JS_TRIPLE_OR_FALLBACK = re.compile(r"\b([a-zA-Z0-9_\.]+\s*(?:\|\||\?\?)\s*){3,}")
# 4. Redundant boolean conversions: Boolean(!!x) or !!Boolean(x) or Boolean(x === true)
JS_REDUNDANT_BOOLEAN = re.compile(r"(?:Boolean\(\s*!!|!!\s*Boolean\(|Boolean\(\s*[a-zA-Z0-9_\.]+\s*===?\s*(?:true|false)\s*\))")
# 5. Redundant String/Number double conversions: Number(n || 0) || 0 or String(String(x))
JS_REDUNDANT_COERCION = re.compile(r"(?:String\(\s*String\(|Number\(\s*Number\(|Number\(\s*[a-zA-Z0-9_\.]+\s*\|\|\s*0\s*\)\s*\|\|\s*0)")
# 6. Swallowing catch blocks: catch (e) {} or catch (err) { /* ignore */ }
JS_SWALLOWING_CATCH = re.compile(r"catch\s*\([^\)]*\)\s*\{\s*(?://[^\n]*)?\s*\}")
# 7. Paranoid typeof stack: typeof x !== 'undefined' && x !== null && x !== ''
JS_PARANOID_TYPEOF = re.compile(r"typeof\s+([a-zA-Z0-9_\.]+)\s*!==?\s*['\"]undefined['\"]\s*&&\s*\1\s*!==?\s*null")
# 8. Nested ternary operator check (requires ? ... : ... ? ... : ...)
NESTED_TERNARIES = re.compile(r"[^?:\n]+\?[^?:\n]+:[^?:\n]+\?[^?:\n]+:")


def scan_file_contents(filepath: str, content: str) -> List[Dict[str, Any]]:
    findings = []
    lines = content.splitlines()
    ext = os.path.splitext(filepath)[1].lower()

    # Multi-line / raw string analysis
    # 1. Check Go len() before range
    if ext == ".go":
        for m in GO_LEN_BEFORE_RANGE.finditer(content):
            start_pos = m.start()
            line_no = content.count("\n", 0, start_pos) + 1
            matched_text = m.group(0).strip().replace("\n", " ")
            findings.append({
                "file": filepath,
                "line": line_no,
                "code": matched_text[:80],
                "rule_id": "go_len_before_range",
                "category": "defensive",
                "severity": "warning",
                "message": f"Redundant 'len({m.group(1)}) > 0' check before 'range'. In Go, range on nil/empty slice is already a no-op.",
                "hint": f"Remove the enclosing 'if len({m.group(1)}) > 0' condition."
            })

        for m in GO_NIL_BEFORE_DELETE.finditer(content):
            line_no = content.count("\n", 0, m.start()) + 1
            findings.append({
                "file": filepath,
                "line": line_no,
                "code": m.group(0).strip(),
                "rule_id": "go_nil_before_delete",
                "category": "defensive",
                "severity": "warning",
                "message": f"Redundant nil check before delete(). In Go, delete() is a safe no-op on nil maps.",
                "hint": f"Directly call delete({m.group(1)}, ...) without the if condition."
            })

    # Line-by-line checks
    for idx, line in enumerate(lines):
        line_no = idx + 1
        stripped = line.strip()

        # Skip empty lines
        if not stripped:
            continue

        is_comment = (
            stripped.startswith("//") or
            stripped.startswith("#") or
            stripped.startswith("/*") or
            stripped.startswith("*")
        )

        # Rule: AI Obvious Comments
        if is_comment:
            for pattern, desc in OBVIOUS_COMMENT_PATTERNS:
                if pattern.search(line):
                    findings.append({
                        "file": filepath,
                        "line": line_no,
                        "code": stripped,
                        "rule_id": "ai_obvious_comment",
                        "category": "slop",
                        "severity": "info",
                        "message": f"AI Slop: {desc}.",
                        "hint": "Remove comments that merely restate self-explanatory code."
                    })
                    break

            for pattern, desc in AI_CONVERSATIONAL_PATTERNS:
                if pattern.search(line):
                    findings.append({
                        "file": filepath,
                        "line": line_no,
                        "code": stripped,
                        "rule_id": "ai_conversational_leftover",
                        "category": "slop",
                        "severity": "warning",
                        "message": f"AI Slop: {desc}.",
                        "hint": "Clean up AI conversational artifacts or meta-explanations."
                    })
                    break

            # Go Parrot Docstring
            if ext == ".go":
                m_doc = PARROT_DOC_GO.match(stripped)
                if m_doc:
                    name1, name2 = m_doc.group(1).lower(), m_doc.group(2).lower()
                    if name1 == name2 or name1.startswith(name2) or name2.startswith(name1):
                        findings.append({
                            "file": filepath,
                            "line": line_no,
                            "code": stripped,
                            "rule_id": "ai_parrot_docstring",
                            "category": "slop",
                            "severity": "info",
                            "message": f"Parrot docstring: comments function '{m_doc.group(1)}' with identical words without adding information.",
                            "hint": "Provide meaningful behavioral context or omit obvious doc comments."
                        })
            continue

        # Code-only checks below (ignore comments)

        # Rule: Trivial wrapper in JS/TS
        if ext in {".js", ".ts", ".vue"}:
            if TRIVIAL_WRAPPER_JS.search(stripped):
                findings.append({
                    "file": filepath,
                    "line": line_no,
                    "code": stripped,
                    "rule_id": "ai_trivial_wrapper",
                    "category": "slop",
                    "severity": "warning",
                    "message": "Over-abstracted trivial wrapper function for standard language operator.",
                    "hint": "Use native operators directly instead of wrapping them in micro-utilities."
                })

            # Rule: Paranoid Array Guard
            if JS_PARANOID_ARRAY_GUARD.search(stripped):
                findings.append({
                    "file": filepath,
                    "line": line_no,
                    "code": stripped,
                    "rule_id": "js_paranoid_array_guard",
                    "category": "defensive",
                    "severity": "warning",
                    "message": "Overly defensive array check (`arr && Array.isArray(arr) && arr.length`).",
                    "hint": "Ensure array initialization upstream or simplify to `arr?.length` / `Array.isArray(arr)`."
                })

            # Rule: Excessive Optional Chaining
            m_opt = JS_EXCESSIVE_OPTIONAL_CHAIN.search(stripped)
            if m_opt:
                findings.append({
                    "file": filepath,
                    "line": line_no,
                    "code": m_opt.group(0),
                    "rule_id": "js_excessive_optional_chaining",
                    "category": "defensive",
                    "severity": "info",
                    "message": f"Deep optional chaining overkill: '{m_opt.group(0)}' (5+ chain levels).",
                    "hint": "Destructure expected models or handle nullability at API boundaries."
                })

            # Rule: Triple Fallback Chain
            m_fall = JS_TRIPLE_OR_FALLBACK.search(stripped)
            if m_fall:
                findings.append({
                    "file": filepath,
                    "line": line_no,
                    "code": stripped,
                    "rule_id": "js_triple_fallback_chain",
                    "category": "defensive",
                    "severity": "info",
                    "message": "Overly defensive multi-level fallback chain (3+ fallback operators).",
                    "hint": "Consolidate default value handling at ingestion or use a single fallback."
                })

            # Rule: Redundant Boolean / Type coercion
            if JS_REDUNDANT_BOOLEAN.search(stripped):
                findings.append({
                    "file": filepath,
                    "line": line_no,
                    "code": stripped,
                    "rule_id": "js_redundant_boolean",
                    "category": "defensive",
                    "severity": "warning",
                    "message": "Redundant double boolean conversion (e.g. `Boolean(!!x)` or `Boolean(x === true)`).",
                    "hint": "Simplify to `!!x` or `Boolean(x)`."
                })

            if JS_REDUNDANT_COERCION.search(stripped):
                findings.append({
                    "file": filepath,
                    "line": line_no,
                    "code": stripped,
                    "rule_id": "js_redundant_coercion",
                    "category": "defensive",
                    "severity": "warning",
                    "message": "Redundant fallback inside/outside type coercion (e.g. `String(x || '')` or `Number(x || 0) || 0`).",
                    "hint": "Simplify to `String(x || '')` or `Number(x) || 0`."
                })

            # Rule: Swallowing catch
            if JS_SWALLOWING_CATCH.search(stripped):
                findings.append({
                    "file": filepath,
                    "line": line_no,
                    "code": stripped,
                    "rule_id": "empty_swallowing_catch",
                    "category": "defensive",
                    "severity": "warning",
                    "message": "Silent swallowing catch block with no error handling or logging.",
                    "hint": "Handle the error explicitly or at least log/rethrow it."
                })

            # Rule: Paranoid typeof stack
            if JS_PARANOID_TYPEOF.search(stripped):
                findings.append({
                    "file": filepath,
                    "line": line_no,
                    "code": stripped,
                    "rule_id": "js_paranoid_typeof",
                    "category": "defensive",
                    "severity": "warning",
                    "message": "Paranoid typeof and null checks (`typeof x !== 'undefined' && x !== null`).",
                    "hint": "In modern JS/TS, use `x != null` (handles both null & undefined) or optional chaining."
                })

        # Rule: Nested ternary hell (code only)
        if NESTED_TERNARIES.search(stripped):
            findings.append({
                "file": filepath,
                "line": line_no,
                "code": stripped[:90] + "..." if len(stripped) > 90 else stripped,
                "rule_id": "ai_nested_ternaries",
                "category": "slop",
                "severity": "info",
                "message": "Deeply nested ternary expression (3+ conditionals chained on one line).",
                "hint": "Refactor into standard if/else statements or a dictionary/lookup map."
            })

    return findings


def scan_project(root_dir: str, target_dirs: List[str], target_files: List[str],
                 categories: Optional[List[str]] = None) -> List[Dict[str, Any]]:
    all_findings = []
    scanned_files_count = 0

    files_to_scan = []

    # Collect explicit files
    for tf in target_files:
        full_path = os.path.join(root_dir, tf)
        if os.path.isfile(full_path):
            files_to_scan.append(full_path)

    # Collect directory files
    for td in target_dirs:
        full_dir = os.path.join(root_dir, td)
        if not os.path.exists(full_dir):
            continue

        for dirpath, dirnames, filenames in os.walk(full_dir):
            # Exclude unwanted directories
            dirnames[:] = [d for d in dirnames if d not in EXCLUDE_DIRS and not d.startswith(".")]

            for filename in filenames:
                ext = os.path.splitext(filename)[1].lower()
                if ext not in DEFAULT_EXTENSIONS:
                    continue

                rel_path = os.path.relpath(os.path.join(dirpath, filename), root_dir).replace("\\", "/")

                # Exclude pattern check
                if any(re.search(pat, rel_path) for pat in EXCLUDE_FILE_PATTERNS):
                    continue

                files_to_scan.append(os.path.join(dirpath, filename))

    for filepath in set(files_to_scan):
        try:
            with open(filepath, "r", encoding="utf-8", errors="ignore") as f:
                content = f.read()
            scanned_files_count += 1
            rel_path = os.path.relpath(filepath, root_dir).replace("\\", "/")
            findings = scan_file_contents(rel_path, content)

            if categories:
                findings = [f for f in findings if f["category"] in categories]

            all_findings.extend(findings)
        except Exception as e:
            print(f"{RED}Error reading {filepath}: {e}{RESET}", file=sys.stderr)

    return all_findings, scanned_files_count


def format_report(findings: List[Dict[str, Any]], scanned_count: int, root_dir: str, show_hints: bool = True):
    print(f"\n{BOLD}{CYAN}=== AI Slop & Defensive Code Smells Report ==={RESET}")
    print(f"Scanned {BOLD}{scanned_count}{RESET} files | Found {BOLD}{len(findings)}{RESET} potential issues\n")

    if not findings:
        print(f"{GREEN}✓ No excessive defensive code or AI slop patterns detected! Clean codebase.{RESET}\n")
        return

    # Group by category and rule
    slop_findings = [f for f in findings if f["category"] == "slop"]
    defensive_findings = [f for f in findings if f["category"] == "defensive"]

    def print_section(title: str, items: List[Dict[str, Any]], color: str):
        if not items:
            return
        print(f"{BOLD}{color}▶ {title} ({len(items)} issues){RESET}")
        print("─" * 70)

        # Group by file
        by_file: Dict[str, List[Dict[str, Any]]] = {}
        for item in items:
            by_file.setdefault(item["file"], []).append(item)

        for file_path, file_items in by_file.items():
            abs_link = f"file:///{os.path.abspath(os.path.join(root_dir, file_path)).replace('\\', '/')}"
            print(f"\n{BOLD}{file_path}{RESET}  {DIM}({abs_link}){RESET}")

            for f in file_items:
                sev_color = YELLOW if f["severity"] == "warning" else CYAN
                sev_tag = f"[{f['severity'].upper()}]"
                print(f"  Line {BOLD}{f['line']:<4}{RESET} {sev_color}{sev_tag:<9}{RESET} {f['message']}")
                print(f"  {DIM}Code:{RESET}  {f['code']}")
                if show_hints and f.get("hint"):
                    print(f"  {DIM}Fix :{RESET}  {GREEN}{f['hint']}{RESET}")
        print()

    print_section("AI Slop & Boilerplate", slop_findings, MAGENTA)
    print_section("Overly Defensive Code", defensive_findings, YELLOW)

    print(f"{BOLD}Summary:{RESET}")
    print(f"  • AI Slop / Verbose Comments:  {BOLD}{len(slop_findings)}{RESET}")
    print(f"  • Defensive Overkill Patterns: {BOLD}{len(defensive_findings)}{RESET}")
    print()


def main():
    parser = argparse.ArgumentParser(
        description="Detect overly defensive code and AI slop in Go, JS, TS, Vue, and Python files."
    )
    parser.add_argument(
        "--dir", "-d",
        nargs="+",
        default=DEFAULT_DIRS,
        help=f"Directories to scan (default: {' '.join(DEFAULT_DIRS)})"
    )
    parser.add_argument(
        "--file", "-f",
        nargs="+",
        default=[],
        help="Specific files to scan"
    )
    parser.add_argument(
        "--category", "-c",
        choices=["all", "defensive", "slop"],
        default="all",
        help="Category to filter by (default: all)"
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output results as JSON"
    )
    parser.add_argument(
        "--no-hints",
        action="store_true",
        help="Do not display fix hints"
    )

    args = parser.parse_args()

    # Determine project root (parent directory of scripts/ or current working directory)
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(script_dir) if os.path.basename(script_dir) == "scripts" else os.getcwd()

    categories = None
    if args.category != "all":
        categories = [args.category]

    findings, scanned_count = scan_project(
        root_dir=project_root,
        target_dirs=args.dir,
        target_files=args.file,
        categories=categories
    )

    if args.json:
        print(json.dumps({
            "scanned_files": scanned_count,
            "total_issues": len(findings),
            "findings": findings
        }, indent=2))
    else:
        format_report(findings, scanned_count, project_root, show_hints=not args.no_hints)

    # Exit code: 0 if no warnings, 1 if warnings found
    sys.exit(0)


if __name__ == "__main__":
    main()
