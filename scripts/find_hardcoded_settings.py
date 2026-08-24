#!/usr/bin/env python3
import os
import re
import sys
import argparse
from path_utils import validate_path

# Default directories to scan (relative to project root)
DEFAULT_DIRS = ["internal", "cmd", "frontend/src"]
DEFAULT_FILES = ["main.go"]

# Default file extensions to scan
DEFAULT_EXTENSIONS = {".go", ".js", ".ts", ".vue"}

# Excluded patterns (e.g. test files)
EXCLUDE_FILE_PATTERNS = [
    r"_test\.go$",
    r"\.test\.js$",
    r"\.spec\.js$",
    r"\.test\.ts$",
    r"\.spec\.ts$",
]

# Keywords indicating a setting-like variable (case-insensitive)
SETTING_KEYWORDS = [
    "word", "limit", "max", "min", "timeout", "interval", "threshold",
    "duration", "delay", "capacity", "size", "count", "url", "host",
    "port", "retries", "expiry", "target", "prompt", "model", "api",
    "pace", "endpoint", "token", "secret", "key", "theme", "enabled",
    "active", "rate", "coefficient", "factor", "weight", "bias"
]

# Build keyword regex pattern component
KEYWORD_PATTERN = "|".join(
    f"{kw[0].upper()}{kw[1:]}|{kw[0].lower()}{kw[1:]}" for kw in SETTING_KEYWORDS
)

# Regex Patterns for scanning
# 1. Go assignment patterns: targetWords := 5000, MaxTimeout = 10 * time.Second, etc.
GO_PATTERN = re.compile(
    r"\b([a-zA-Z_][a-zA-Z0-9_]*(?:" + KEYWORD_PATTERN + r")[a-zA-Z0-9_]*)\s*(:=|=)\s*"
    r"("
    r"\d+(?:\.\d+)?"                          # Number (integer/float)
    r'|"[^"]*"'                                # Double quoted string
    r"|'[^']*'"                                # Single quoted string
    r"|true|false"                             # Boolean
    r"|(?:time\.(?:Second|Minute|Hour|Millisecond|Microsecond|Nanosecond)(?:\s*\*\s*\d+)?)" # Duration suffix
    r"|(?:\d+\s*\*\s*time\.(?:Second|Minute|Hour|Millisecond|Microsecond|Nanosecond))"      # Duration prefix
    r")\b"
)

# 2. JS/TS/Vue variable declarations and object properties
JS_PATTERN = re.compile(
    r"\b(?:const|let|var)\s+([a-zA-Z_][a-zA-Z0-9_]*(?:" + KEYWORD_PATTERN + r")[a-zA-Z0-9_]*)\s*=\s*"
    r"("
    r"\d+(?:\.\d+)?"                          # Number
    r'|"[^"]*"'                                # Double quoted string
    r"|'[^']*'"                                # Single quoted string
    r"|`[^`]*`"                                # Template string
    r"|true|false"                             # Boolean
    r")\b"
)

JS_PROP_PATTERN = re.compile(
    r"\b([a-zA-Z_][a-zA-Z0-9_]*(?:" + KEYWORD_PATTERN + r")[a-zA-Z0-9_]*)\s*:\s*"
    r"("
    r"\d+(?:\.\d+)?"                          # Number
    r'|"[^"]*"'                                # Double quoted string
    r"|'[^']*'"                                # Single quoted string
    r"|`[^`]*`"                                # Template string
    r"|true|false"                             # Boolean
    r")\b"
)

def should_exclude_file(filename):
    for pattern in EXCLUDE_FILE_PATTERNS:
        if re.search(pattern, filename):
            return True
    return False

def is_noise_zero(val_str):
    clean = val_str.strip().strip("'\"`")
    return clean in ("0", "0.0", "false", '""', "''", "``", "nil", "null", "undefined")

def scan_file(filepath, include_zeros=False, target_values=None):
    results = []
    ext = os.path.splitext(filepath)[1]
    
    try:
        with open(filepath, "r", encoding="utf-8", errors="ignore") as f:
            for line_num, line in enumerate(f, 1):
                clean_line = line.strip()
                # Skip comments
                if clean_line.startswith("//") or clean_line.startswith("#") or clean_line.startswith("/*") or clean_line.startswith("*"):
                    continue
                
                matches = []
                if ext == ".go":
                    for match in GO_PATTERN.finditer(line):
                        matches.append((match.group(1), match.group(3)))
                elif ext in (".js", ".ts", ".vue"):
                    for match in JS_PATTERN.finditer(line):
                        matches.append((match.group(1), match.group(2)))
                    for match in JS_PROP_PATTERN.finditer(line):
                        # Avoid duplicates if matching both (unlikely but safe)
                        item = (match.group(1), match.group(2))
                        if item not in matches:
                            matches.append(item)
                
                for var_name, value in matches:
                    # Double-check: ensure we didn't just match helper function names or keywords as variables
                    if var_name.lower() in ("true", "false", "nil", "null", "undefined"):
                        continue
                    
                    # Filter out noise zeros if not requested
                    if not include_zeros and is_noise_zero(value):
                        continue
                    
                    # Filter by target values if requested (e.g. 8000, 3000, 4000)
                    if target_values:
                        val_clean = value.strip().strip("'\"`")
                        if not any(tv == val_clean or tv in val_clean for tv in target_values):
                            continue

                    results.append({
                        "line": line_num,
                        "variable": var_name,
                        "value": value,
                        "content": clean_line
                    })
    except Exception as e:
        print(f"Error reading {filepath}: {e}", file=sys.stderr)
        
    return results

def main():
    parser = argparse.ArgumentParser(description="Find hardcoded settings-like variables in the codebase.")
    parser.add_argument("--dir", action="append", help="Directories to scan (can specify multiple)")
    parser.add_argument("--file", action="append", help="Specific files to scan (can specify multiple)")
    parser.add_argument("--output", help="Write findings to a Markdown file instead of stdout")
    parser.add_argument("--value", "-v", action="append", help="Target value(s) to search for (e.g. 8000, 3000, 4000). Can be comma-separated or repeated.")
    parser.add_argument("--include-zeros", action="store_true", help="Include 0/false/empty zero-initializers in the report")
    
    args = parser.parse_args()
    
    target_values = []
    if args.value:
        for val_arg in args.value:
            for v in val_arg.split(","):
                v_clean = v.strip()
                if v_clean:
                    target_values.append(v_clean)
    
    # Get project root (parent directory of this script's directory)
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(script_dir)
    
    scan_dirs = args.dir if args.dir else DEFAULT_DIRS
    scan_files = args.file if args.file else DEFAULT_FILES
    
    all_findings = {}
    
    # Scan individual files
    for f in scan_files:
        full_path = validate_path(os.path.join(project_root, f), project_root)
        if os.path.isfile(full_path) and not should_exclude_file(f):
            results = scan_file(full_path, include_zeros=args.include_zeros, target_values=target_values)
            if results:
                all_findings[f] = results
                
    # Scan directories
    for d in scan_dirs:
        dir_path = validate_path(os.path.join(project_root, d), project_root)
        if not os.path.isdir(dir_path):
            continue
            
        for root, _, files in os.walk(dir_path):
            for file in files:
                ext = os.path.splitext(file)[1]
                if ext in DEFAULT_EXTENSIONS:
                    rel_path = os.path.relpath(os.path.join(root, file), project_root)
                    if should_exclude_file(file):
                        continue
                    results = scan_file(os.path.join(root, file), include_zeros=args.include_zeros, target_values=target_values)
                    if results:
                        all_findings[rel_path] = results

    # Generate output
    output_lines = []
    output_lines.append("# Hardcoded Settings Scanner Report")
    if target_values:
        output_lines.append(f"**Filter**: Filtered for specific target values: `{', '.join(target_values)}`\n")
    else:
        output_lines.append("This report lists variables, constants, and properties matching setting-like keywords assigned to hardcoded values.\n")
    
    total_files = len(all_findings)
    total_instances = sum(len(instances) for instances in all_findings.values())
    
    output_lines.append(f"**Summary**: Found **{total_instances}** instances across **{total_files}** files.\n")
    
    for filepath, findings in sorted(all_findings.items()):
        output_lines.append(f"### `{filepath}`")
        output_lines.append("| Line | Variable | Hardcoded Value | Context |")
        output_lines.append("| --- | --- | --- | --- |")
        for f in findings:
            # Escape pipes in content for markdown tables
            safe_content = f["content"].replace("|", "\\|")
            output_lines.append(f"| {f['line']} | `{f['variable']}` | `{f['value']}` | `{safe_content}` |")
        output_lines.append("")
        
    report = "\n".join(output_lines)
    
    if args.output:
        out_path = validate_path(os.path.join(project_root, args.output), project_root)
        with open(out_path, "w", encoding="utf-8") as out_f:
            out_f.write(report)
        print(f"Report written to {out_path}")
    else:
        print(report)

if __name__ == "__main__":
    main()
