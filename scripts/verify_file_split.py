#!/usr/bin/env python3
"""
verify_file_split.py

Verifies that splitting a Go source file into multiple files does not drop,
duplicate, or modify any top-level declaration (functions, methods, types,
constants, variables).

Usage:
  # Compare original file to one or more split files:
  python scripts/verify_file_split.py --original internal/db/study_queue_repo.go --splits internal/db/study_queue_repo.go internal/db/study_queue_queries.go internal/db/study_queue_mutations.go

  # Compare a git revision of the original file against local split files:
  python scripts/verify_file_split.py --git-ref HEAD~1:internal/db/study_queue_repo.go --splits internal/db/study_queue_*.go

  # Strict exact character comparison (excluding package/import statements):
  python scripts/verify_file_split.py --original orig.go --splits split1.go split2.go --strict
"""

import argparse
import difflib
import glob
import os
import re
import subprocess
import sys
from typing import Dict, List, Optional, Tuple


def extract_top_level_declarations(code: str) -> Dict[str, str]:
    """
    Parses Go code into top-level declarations:
    - package declaration
    - imports
    - types
    - const/var blocks
    - functions and methods
    """
    declarations = {}
    lines = code.splitlines(keepends=True)
    n = len(lines)
    i = 0

    decl_pattern = re.compile(
        r"^(func\s*(?:\([^)]+\)\s*)?(\w+)|type\s+(\w+)|const\s+(?:\(|(\w+))|var\s+(?:\(|(\w+)))"
    )

    while i < n:
        line = lines[i]

        # Ignore comments and blank lines outside decls
        trimmed = line.strip()
        if not trimmed or trimmed.startswith("//") or trimmed.startswith("/*"):
            # Check if it's a doc comment before a top-level decl
            doc_lines = []
            while i < n and (lines[i].strip().startswith("//") or lines[i].strip().startswith("/*") or not lines[i].strip()):
                doc_lines.append(lines[i])
                i += 1
            if i >= n:
                break
            line = lines[i]
            match = decl_pattern.match(line)
            if not match:
                # Floating comments or package/import
                if line.startswith("package ") or line.startswith("import "):
                    # Handled separately
                    pass
                else:
                    i += 1
                continue
            # Attach doc comments to following declaration
            prefix = "".join(doc_lines)
        else:
            doc_lines = []
            prefix = ""
            match = decl_pattern.match(line)

        if line.startswith("package ") or line.startswith("import"):
            # Skip package and import statements in declaration equality
            if line.startswith("import ("):
                while i < n and not lines[i].strip().startswith(")"):
                    i += 1
            i += 1
            continue

        if match:
            start_i = i
            # Determine declaration key
            func_match = re.match(r"^func\s*(\([^)]+\)\s*)?(\w+)", line)
            type_match = re.match(r"^type\s+(\w+)", line)
            const_match = re.match(r"^const\s*(?:\(\s*)?(\w*)", line)
            var_match = re.match(r"^var\s*(?:\(\s*)?(\w*)", line)

            if func_match:
                recv = func_match.group(1) or ""
                recv = re.sub(r"\s+", " ", recv.strip())
                name = func_match.group(2)
                key = f"func {recv} {name}".strip()
            elif type_match:
                key = f"type {type_match.group(1)}"
            elif const_match and const_match.group(1):
                key = f"const {const_match.group(1)}"
            elif var_match and var_match.group(1):
                key = f"var {var_match.group(1)}"
            else:
                key = f"decl_line_{start_i + 1}_{line.strip()[:30]}"

            # Parse balanced braces or block parenthesis
            brace_count = 0
            paren_count = 0
            decl_lines = [prefix] if prefix else []

            in_raw_string = False
            in_block_comment = False

            while i < n:
                cur_line = lines[i]
                decl_lines.append(cur_line)
                
                in_string = False
                in_rune = False
                in_line_comment = False
                escape = False
                
                idx = 0
                while idx < len(cur_line):
                    char = cur_line[idx]
                    
                    if in_block_comment:
                        if char == '*' and idx + 1 < len(cur_line) and cur_line[idx + 1] == '/':
                            in_block_comment = False
                            idx += 2
                            continue
                    elif in_raw_string:
                        if char == '`':
                            in_raw_string = False
                    elif in_string:
                        if escape:
                            escape = False
                        elif char == '\\':
                            escape = True
                        elif char == '"':
                            in_string = False
                    elif in_rune:
                        if escape:
                            escape = False
                        elif char == '\\':
                            escape = True
                        elif char == "'":
                            in_rune = False
                    elif in_line_comment:
                        break
                    else:
                        if char == '/' and idx + 1 < len(cur_line) and cur_line[idx + 1] == '/':
                            in_line_comment = True
                            idx += 2
                            continue
                        elif char == '/' and idx + 1 < len(cur_line) and cur_line[idx + 1] == '*':
                            in_block_comment = True
                            idx += 2
                            continue
                        elif char == '"':
                            in_string = True
                            escape = False
                        elif char == '`':
                            in_raw_string = True
                        elif char == "'":
                            in_rune = True
                            escape = False
                        elif char == '{':
                            brace_count += 1
                        elif char == '}':
                            brace_count -= 1
                        elif char == '(':
                            paren_count += 1
                        elif char == ')':
                            paren_count -= 1
                    
                    idx += 1

                i += 1
                if brace_count == 0 and paren_count == 0 and not in_raw_string and not in_block_comment and i > start_i:
                    break

            content = "".join(decl_lines).strip()
            
            # Avoid overwriting identical keys
            original_key = key
            counter = 1
            while key in declarations:
                key = f"{original_key}#{counter}"
                counter += 1

            declarations[key] = content
        else:
            i += 1

    return declarations


def normalize_code(code: str) -> str:
    """Normalizes whitespace and line endings for comparison."""
    code = code.replace("\r\n", "\n")
    # Collapse multiple blank lines
    code = re.sub(r"\n\s*\n", "\n\n", code)
    return code.strip()


def get_file_content_from_git(git_ref: str) -> str:
    """Extract file content from git ref (e.g. 'HEAD~1:internal/db/study_queue_repo.go')."""
    cmd = ["git", "show", git_ref]
    res = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, encoding="utf-8")
    if res.returncode != 0:
        raise RuntimeError(f"Git command failed ({' '.join(cmd)}): {res.stderr}")
    return res.stdout


def verify_split(original_code: str, split_codes: List[Tuple[str, str]], strict: bool = False) -> bool:
    orig_decls = extract_top_level_declarations(original_code)
    
    split_decls = {}
    decl_sources = {}
    duplicate_count = 0
    errors_found = False

    for filename, code in split_codes:
        file_decls = extract_top_level_declarations(code)
        for k, v in file_decls.items():
            if k in split_decls:
                print(f"[!] DUPLICATE DECLARATION: '{k}' found in both '{decl_sources[k]}' and '{filename}'")
                duplicate_count += 1
                errors_found = True
            else:
                split_decls[k] = v
                decl_sources[k] = filename

    orig_keys = set(orig_decls.keys())
    split_keys = set(split_decls.keys())

    missing_in_splits = orig_keys - split_keys
    added_in_splits = split_keys - orig_keys
    common_keys = orig_keys & split_keys

    print("=" * 70)
    print(f"VERIFICATION SUMMARY: {len(orig_keys)} original declarations -> {len(split_keys)} split declarations")
    print("=" * 70)

    if missing_in_splits:
        errors_found = True
        print(f"\n[-] MISSING ({len(missing_in_splits)} declarations dropped in split):")
        for k in sorted(missing_in_splits):
            print(f"    - {k}")

    if added_in_splits:
        print(f"\n[+] ADDED/EXTRA ({len(added_in_splits)} new declarations in split):")
        for k in sorted(added_in_splits):
            print(f"    + {k} (in {decl_sources[k]})")

    modified_count = 0
    for k in sorted(common_keys):
        v1 = orig_decls[k]
        v2 = split_decls[k]

        c1 = v1 if strict else normalize_code(v1)
        c2 = v2 if strict else normalize_code(v2)

        if c1 != c2:
            errors_found = True
            modified_count += 1
            print(f"\n[!] MODIFIED DECLARATION: {k} (found in {decl_sources[k]})")
            diff = difflib.unified_diff(
                c1.splitlines(keepends=True),
                c2.splitlines(keepends=True),
                fromfile="original",
                tofile=decl_sources[k],
                n=3,
            )
            print("".join(diff))

    if not errors_found and not added_in_splits:
        print("\n[SUCCESS] All declarations matched exactly! 0 changes, 0 dropped, 0 duplicates.")
        return True
    elif not errors_found and added_in_splits:
        print("\n[SUCCESS with additions] All original declarations exist without changes. New helper declarations were added.")
        return True
    else:
        print(f"\n[FAILED] Detected {len(missing_in_splits)} missing, {modified_count} modified, and {duplicate_count} duplicate declarations.")
        return False


def main():
    parser = argparse.ArgumentParser(description="Verify that splitting a Go source file does not change any content.")
    parser.add_argument("--original", "-o", help="Path to original unsplit Go file")
    parser.add_argument("--git-ref", "-g", help="Git revision of original file (e.g. HEAD:internal/db/study_queue_repo.go)")
    parser.add_argument("--splits", "-s", nargs="+", required=True, help="Path(s) or glob patterns of split files")
    parser.add_argument("--strict", action="store_true", help="Perform strict whitespace & comment matching")

    args = parser.parse_args()

    # Load original content
    if args.git_ref:
        print(f"Reading original file from git ref: {args.git_ref}")
        orig_code = get_file_content_from_git(args.git_ref)
    elif args.original:
        if not os.path.exists(args.original):
            print(f"Error: Original file '{args.original}' does not exist.")
            sys.exit(1)
        with open(args.original, "r", encoding="utf-8") as f:
            orig_code = f.read()
    else:
        print("Error: Either --original or --git-ref must be specified.")
        sys.exit(1)

    # Resolve split files
    split_paths = []
    for pattern in args.splits:
        matches = glob.glob(pattern)
        if matches:
            split_paths.extend(matches)
        else:
            split_paths.append(pattern)

    split_paths = list(dict.fromkeys(split_paths))  # deduplicate preserving order

    split_codes = []
    for path in split_paths:
        if not os.path.exists(path):
            print(f"Error: Split file '{path}' does not exist.")
            sys.exit(1)
        with open(path, "r", encoding="utf-8") as f:
            split_codes.append((path, f.read()))

    success = verify_split(orig_code, split_codes, strict=args.strict)
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
