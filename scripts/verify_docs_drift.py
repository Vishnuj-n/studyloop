#!/usr/bin/env python3
"""
Documentation Drift Verification Tool

Statically checks whether project documentation in doc/ matches the actual codebase:
1. Schema Sync: internal/db/schema.go <-> doc/SCHEMA.md
   - Verifies all SQLite tables and columns in schema.go exist in doc/SCHEMA.md.
   - Verifies no obsolete tables/columns are documented.
2. Package Structure Sync: internal/ <-> doc/PROJECT_STRUCTURE.md
   - Verifies all internal packages are registered in doc/PROJECT_STRUCTURE.md.
3. Auto-Generated Doc Freshness: scripts/analyze_api_dependency_graph.py <-> doc/api_dependency_report.md
   - Checks if doc/api_dependency_report.md is up-to-date with current API routes.

Usage:
    python scripts/verify_docs_drift.py [options]

Options:
    --fix         Automatically regenerate auto-generated docs (e.g. api_dependency_report.md)
    --verbose     Show detailed parsing and matching information
"""

import os
import re
import sys
import subprocess
import argparse
from typing import Dict, Set, List, Tuple

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


def get_repo_root() -> str:
    """Finds repository root relative to this script."""
    script_dir = os.path.dirname(os.path.abspath(__file__))
    return os.path.dirname(script_dir)


def parse_schema_go(schema_path: str) -> Dict[str, Set[str]]:
    """
    Parses internal/db/schema.go and extracts:
    { table_name: set(column_names) }
    Uses balanced parenthesis matching to properly handle nested constraints like CHECK (...).
    """
    if not os.path.exists(schema_path):
        raise FileNotFoundError(f"Schema file not found at {schema_path}")

    with open(schema_path, "r", encoding="utf-8", errors="ignore") as f:
        content = f.read()

    pattern = re.compile(
        r"CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_]+)\s*\(",
        re.IGNORECASE
    )

    tables: Dict[str, Set[str]] = {}

    for match in pattern.finditer(content):
        table_name = match.group(1).strip()
        start_idx = match.end()
        depth = 1
        i = start_idx
        while i < len(content) and depth > 0:
            if content[i] == '(':
                depth += 1
            elif content[i] == ')':
                depth -= 1
            i += 1

        if depth != 0:
            continue

        body = content[start_idx:i-1]
        columns: Set[str] = set()

        for raw_line in body.splitlines():
            line = raw_line.strip().rstrip(",")
            if not line:
                continue

            upper = line.upper()
            # Skip table-level constraints and keys
            if upper.startswith(("PRIMARY KEY", "FOREIGN KEY", "UNIQUE", "CHECK", "CONSTRAINT")):
                continue

            tokens = line.split()
            if tokens:
                col_name = tokens[0].strip("`\"'[]")
                if col_name and not col_name.upper().startswith(("PRIMARY", "FOREIGN", "UNIQUE", "CHECK", "CONSTRAINT")):
                    columns.add(col_name)

        tables[table_name] = columns

    return tables


def parse_schema_md(schema_doc_path: str) -> Dict[str, Set[str]]:
    """
    Parses doc/SCHEMA.md and extracts documented tables and their columns.
    Returns { table_name: set(column_names) }
    """
    if not os.path.exists(schema_doc_path):
        raise FileNotFoundError(f"Schema doc not found at {schema_doc_path}")

    with open(schema_doc_path, "r", encoding="utf-8", errors="ignore") as f:
        content = f.read()

    tables: Dict[str, Set[str]] = {}

    # Split by markdown headers
    sections = re.split(r"(^#{1,4}\s+.*$)", content, flags=re.MULTILINE)

    for i in range(1, len(sections), 2):
        header_line = sections[i].strip()
        body = sections[i + 1] if i + 1 < len(sections) else ""

        table_match = re.match(r"^###\s+`([a-zA-Z0-9_]+)`", header_line)
        if not table_match:
            continue

        table_name = table_match.group(1).strip()
        columns: Set[str] = set()

        for line in body.splitlines():
            line = line.strip()
            if not (line.startswith("|") and line.endswith("|")):
                continue

            parts = [p.strip() for p in line.split("|")[1:-1]]
            if not parts:
                continue

            raw_field = parts[0]
            # Skip header row or separator row
            if re.search(r"^[-:\s]+$", raw_field) or raw_field.lower() in ("field", "column"):
                continue

            # Extract identifier enclosed in backticks or bare word
            field_match = re.search(r"`([a-zA-Z0-9_]+)`", raw_field)
            if field_match:
                columns.add(field_match.group(1))
            else:
                cleaned = raw_field.strip("`\"' *")
                if cleaned and re.match(r"^[a-zA-Z0-9_]+$", cleaned) and cleaned.lower() not in ("field", "column"):
                    columns.add(cleaned)

        tables[table_name] = columns

    return tables


def check_schema_sync(repo_root: str, verbose: bool = False) -> Tuple[bool, List[str]]:
    """Checks for drift between internal/db/schema.go and doc/SCHEMA.md."""
    schema_go_path = os.path.join(repo_root, "internal", "db", "schema.go")
    schema_md_path = os.path.join(repo_root, "doc", "SCHEMA.md")

    code_tables = parse_schema_go(schema_go_path)
    doc_tables = parse_schema_md(schema_md_path)

    issues: List[str] = []

    # Check for missing tables in doc
    missing_tables = set(code_tables.keys()) - set(doc_tables.keys())
    for t in sorted(missing_tables):
        issues.append(f"Table `{t}` defined in schema.go is MISSING from doc/SCHEMA.md")

    # Check for extra undocumented tables in doc (obsolete)
    extra_tables = set(doc_tables.keys()) - set(code_tables.keys())
    for t in sorted(extra_tables):
        issues.append(f"Table `{t}` documented in doc/SCHEMA.md DOES NOT EXIST in schema.go")

    # Check columns for matching tables
    for t in sorted(set(code_tables.keys()) & set(doc_tables.keys())):
        code_cols = code_tables[t]
        doc_cols = doc_tables[t]

        missing_cols = code_cols - doc_cols
        for col in sorted(missing_cols):
            issues.append(f"Table `{t}`: column `{col}` is in schema.go but MISSING from doc/SCHEMA.md")

        extra_cols = doc_cols - code_cols
        for col in sorted(extra_cols):
            issues.append(f"Table `{t}`: column `{col}` in doc/SCHEMA.md DOES NOT EXIST in schema.go")

    if verbose:
        print(f"  [DEBUG] Parsed {len(code_tables)} tables from schema.go: {', '.join(sorted(code_tables.keys()))}")
        print(f"  [DEBUG] Parsed {len(doc_tables)} tables from doc/SCHEMA.md: {', '.join(sorted(doc_tables.keys()))}")

    return len(issues) == 0, issues


def check_package_structure(repo_root: str, verbose: bool = False) -> Tuple[bool, List[str]]:
    """Checks if all internal/ packages are registered in doc/PROJECT_STRUCTURE.md."""
    internal_dir = os.path.join(repo_root, "internal")
    doc_path = os.path.join(repo_root, "doc", "PROJECT_STRUCTURE.md")

    if not os.path.exists(internal_dir) or not os.path.exists(doc_path):
        return True, []

    with open(doc_path, "r", encoding="utf-8", errors="ignore") as f:
        doc_content = f.read()

    issues: List[str] = []
    ignored_subdirs = {"__pycache__", ".git", "testdata"}

    for item in sorted(os.listdir(internal_dir)):
        item_path = os.path.join(internal_dir, item)
        if os.path.isdir(item_path) and item not in ignored_subdirs:
            # Package name should appear in doc/PROJECT_STRUCTURE.md (e.g. `item/` or `internal/item`)
            pattern = rf"(?:internal/)?{re.escape(item)}/"
            if not re.search(pattern, doc_content):
                issues.append(f"Internal package `internal/{item}` is NOT registered in doc/PROJECT_STRUCTURE.md")

    return len(issues) == 0, issues


def check_api_dependency_freshness(repo_root: str, fix: bool = False, verbose: bool = False) -> Tuple[bool, List[str]]:
    """Checks if doc/api_dependency_report.md is fresh."""
    script_path = os.path.join(repo_root, "scripts", "analyze_api_dependency_graph.py")
    doc_path = os.path.join(repo_root, "doc", "api_dependency_report.md")

    if not os.path.exists(script_path):
        return True, []

    issues: List[str] = []

    if fix:
        cmd = [sys.executable, script_path, "--output", "doc/api_dependency_report.md"]
        res = subprocess.run(cmd, cwd=repo_root, capture_output=True, text=True)
        if res.returncode != 0:
            issues.append(f"Failed to regenerate api_dependency_report.md: {res.stderr}")
            return False, issues
        return True, []

    if not os.path.exists(doc_path):
        issues.append("doc/api_dependency_report.md is missing. Run with --fix or python scripts/analyze_api_dependency_graph.py --output doc/api_dependency_report.md")
        return False, issues

    # Generate in-memory and compare
    cmd = [sys.executable, script_path]
    res = subprocess.run(cmd, cwd=repo_root, capture_output=True, text=True, encoding="utf-8")
    if res.returncode != 0:
        issues.append(f"Failed running analyze_api_dependency_graph.py: {res.stderr}")
        return False, issues

    with open(doc_path, "r", encoding="utf-8", errors="ignore") as f:
        existing_content = f.read()

    # Normalize newlines
    existing_normalized = existing_content.replace("\r\n", "\n").strip()
    generated_normalized = res.stdout.replace("\r\n", "\n").strip()

    if existing_normalized != generated_normalized:
        issues.append("doc/api_dependency_report.md is OUT OF DATE with current codebase. Run with --fix to update it.")

    return len(issues) == 0, issues


def main():
    parser = argparse.ArgumentParser(description="Verify documentation synchronization against codebase.")
    parser.add_argument("--fix", action="store_true", help="Auto-fix regenerable documentation files")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose diagnostic output")
    args = parser.parse_args()

    repo_root = get_repo_root()

    print(f"\n{BOLD}{CYAN}=== AI Tutor Documentation Drift Verification ==={RESET}\n")

    all_passed = True
    total_issues = 0

    # 1. Schema Sync Check
    print(f"{BOLD}[1/3] Checking Database Schema Sync (schema.go <-> doc/SCHEMA.md)...{RESET}")
    schema_ok, schema_issues = check_schema_sync(repo_root, verbose=args.verbose)
    if schema_ok:
        print(f"  {GREEN}✔ Database schema documentation is completely in sync.{RESET}")
    else:
        all_passed = False
        total_issues += len(schema_issues)
        print(f"  {RED}✘ Found {len(schema_issues)} schema drift issue(s):{RESET}")
        for issue in schema_issues:
            print(f"    {YELLOW}• {issue}{RESET}")

    # 2. Package Structure Check
    print(f"\n{BOLD}[2/3] Checking Internal Package Structure (internal/ <-> doc/PROJECT_STRUCTURE.md)...{RESET}")
    pkg_ok, pkg_issues = check_package_structure(repo_root, verbose=args.verbose)
    if pkg_ok:
        print(f"  {GREEN}✔ Internal package directory structure is completely documented.{RESET}")
    else:
        all_passed = False
        total_issues += len(pkg_issues)
        print(f"  {RED}✘ Found {len(pkg_issues)} package structure drift issue(s):{RESET}")
        for issue in pkg_issues:
            print(f"    {YELLOW}• {issue}{RESET}")

    # 3. API Dependency Report Check
    print(f"\n{BOLD}[3/3] Checking API Dependency Report Freshness (doc/api_dependency_report.md)...{RESET}")
    api_ok, api_issues = check_api_dependency_freshness(repo_root, fix=args.fix, verbose=args.verbose)
    if api_ok:
        if args.fix:
            print(f"  {GREEN}✔ API dependency report verified and updated.{RESET}")
        else:
            print(f"  {GREEN}✔ API dependency report is completely up-to-date.{RESET}")
    else:
        all_passed = False
        total_issues += len(api_issues)
        print(f"  {RED}✘ API dependency report drift detected:{RESET}")
        for issue in api_issues:
            print(f"    {YELLOW}• {issue}{RESET}")

    print("\n" + "=" * 50)
    if all_passed:
        print(f"{GREEN}{BOLD}✨ ALL DOCUMENTATION CHECKS PASSED — NO DRIFT DETECTED! ✨{RESET}\n")
        sys.exit(0)
    else:
        print(f"{RED}{BOLD}💥 DOCUMENTATION DRIFT DETECTED: {total_issues} total issue(s) found.{RESET}")
        print(f"{DIM}Tip: Update docs in doc/ or run 'python scripts/verify_docs_drift.py --fix' for auto-generated files.{RESET}\n")
        sys.exit(1)


if __name__ == "__main__":
    main()
