#!/usr/bin/env python3
"""
Find and classify all toast notification systems across the frontend.
Identifies standard global toasts vs page-specific custom inline toasts.
"""

import os
import re
from pathlib import Path

FRONTEND_SRC = Path(__file__).resolve().parent.parent / "frontend" / "src"

def scan_toasters():
    toast_patterns = {
        "global_composables": [],
        "custom_inline_toasters": [],
        "specialized_toasts": [],
        "standard_toast_usages": [],
    }

    if not FRONTEND_SRC.exists():
        print(f"Error: Directory {FRONTEND_SRC} not found.")
        return

    vue_files = list(FRONTEND_SRC.rglob("*.vue")) + list(FRONTEND_SRC.rglob("*.js"))

    for filepath in sorted(vue_files):
        rel_path = filepath.relative_to(FRONTEND_SRC.parent)
        content = filepath.read_text(encoding="utf-8", errors="ignore")
        lines = content.splitlines()

        # Check if it's the global useToast composable or AppToast component
        if filepath.name == "useToast.js":
            toast_patterns["global_composables"].append({
                "file": str(rel_path),
                "type": "Global Composable (Standard)",
                "details": "Central reactive toast store providing showNotice() and showError()."
            })
            continue

        if filepath.name == "AppToast.vue":
            toast_patterns["global_composables"].append({
                "file": str(rel_path),
                "type": "Global Component (Standard)",
                "details": "Standard global toast UI mounted in App.vue."
            })
            continue

        # Check for specialized toast components
        if "toast" in filepath.name.lower() and filepath.name not in ["AppToast.vue"] and not filepath.name.endswith(".spec.js"):
            toast_patterns["specialized_toasts"].append({
                "file": str(rel_path),
                "name": filepath.name,
                "details": "Dedicated toast component for a specific domain/workflow."
            })

        # Check for local inline toast state definitions (e.g. const showActionToast = ref(...) or const toast = ref(...))
        local_toast_refs = []
        for idx, line in enumerate(lines, 1):
            if re.search(r"const\s+(?:show\w*Toast|\w*toast\w*)\s*=\s*ref\(", line, re.IGNORECASE) and filepath.name != "useToast.js":
                local_toast_refs.append((idx, line.strip()))
            elif re.search(r"function\s+showToast\s*\(", line):
                local_toast_refs.append((idx, line.strip()))

        # Check for inline toast template classes
        toast_template_classes = []
        for idx, line in enumerate(lines, 1):
            matches = re.findall(r'class="([^"]*toast[^"]*)"', line, re.IGNORECASE)
            for m in matches:
                toast_template_classes.append((idx, m))

        if local_toast_refs or (toast_template_classes and filepath.name not in ["AppToast.vue", "RewardToast.vue", "ExtensionSetupToast.vue"]):
            # Check if it also imports useToast
            uses_global = "useToast" in content
            toast_patterns["custom_inline_toasters"].append({
                "file": str(rel_path),
                "local_refs": local_toast_refs,
                "classes": toast_template_classes,
                "uses_global": uses_global
            })

        # Check for useToast calls
        if "useToast" in content and filepath.name not in ["useToast.js", "AppToast.vue"]:
            usages = []
            for idx, line in enumerate(lines, 1):
                if re.search(r"(?:showNotice|showError)\s*\(", line):
                    usages.append((idx, line.strip()))
            if usages:
                toast_patterns["standard_toast_usages"].append({
                    "file": str(rel_path),
                    "usages": usages
                })

    return toast_patterns

def print_report(patterns):
    print("=" * 80)
    print(" " * 20 + "STUDYLOOP FRONTEND TOAST AUDIT REPORT")
    print("=" * 80)
    print("\n1. GLOBAL / STANDARD TOAST SYSTEM")
    print("-" * 80)
    for item in patterns["global_composables"]:
        print(f"  • {item['file']} ({item['type']})")
        print(f"    {item['details']}")

    print("\n2. CUSTOM / INLINE TOAST IMPLEMENTATIONS (Page-Specific / Legacy)")
    print("-" * 80)
    for item in patterns["custom_inline_toasters"]:
        print(f"\n  [!] {item['file']}")
        if item['uses_global']:
            print("      Note: Also imports useToast()")
        if item['local_refs']:
            print("      Local definitions / functions:")
            for line_no, code in item['local_refs']:
                print(f"        Line {line_no:4d}: {code}")
        if item['classes']:
            print("      Custom template elements / classes:")
            for line_no, cls in item['classes']:
                print(f"        Line {line_no:4d}: class=\"{cls}\"")

    print("\n3. SPECIALIZED STANDALONE TOAST COMPONENTS")
    print("-" * 80)
    for item in patterns["specialized_toasts"]:
        print(f"  • {item['file']}")
        print(f"    {item['details']}")

    print("\n4. STANDARD GLOBAL TOAST USAGES (useToast)")
    print("-" * 80)
    for item in patterns["standard_toast_usages"]:
        print(f"  • {item['file']} ({len(item['usages'])} calls)")

    print("\n" + "=" * 80)

if __name__ == "__main__":
    results = scan_toasters()
    if results:
        print_report(results)
