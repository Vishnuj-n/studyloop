#!/usr/bin/env python3
"""
Studyloop Emoji Scanner & Linter
Scans frontend source files for raw emojis and categorizes them:
1. ACTION NEEDED (Static UI icons, banners, timestamps, buttons that should use <BaseIcon />)
2. KEPT / EXEMPT (Gamified rewards, loot chests, avatars, animations, streak freeze)
"""

import os
import re
import sys
from pathlib import Path

# Ensure UTF-8 output on Windows console
if sys.platform == "win32":
    sys.stdout.reconfigure(encoding="utf-8")

RED = "\033[91m"
GREEN = "\033[92m"
YELLOW = "\033[93m"
CYAN = "\033[96m"
BOLD = "\033[1m"
DIM = "\033[2m"
RESET = "\033[0m"

ROOT_DIR = Path(__file__).resolve().parent.parent / "frontend" / "src"

EXEMPT_FILES = {
    "Rewards.vue",
    "RewardsShopModal.vue",
    "MysteryChestModal.vue",
    "StreakFreezeModal.vue",
    "RewardToast.vue",
    "RewardToast.spec.js",
    "GamificationIcon.vue",
    "calendarService.js",
    "calendarService.spec.js",
}

EMOJI_REGEX = re.compile(
    r"[\U0001F000-\U0001FAFF"
    r"\U00002700-\U000027BF"
    r"\U00002600-\U000026FF"
    r"\U00002300-\U000023FF]"
)

def scan_file(file_path: Path):
    rel_path = file_path.relative_to(ROOT_DIR.parent.parent)
    is_exempt = file_path.name in EXEMPT_FILES
    
    try:
        lines = file_path.read_text(encoding="utf-8", errors="ignore").splitlines()
    except Exception as e:
        print(f"{RED}Error reading {rel_path}: {e}{RESET}")
        return []

    findings = []
    for line_idx, line in enumerate(lines, 1):
        if "/* design-lint-ignore */" in line or "// design-lint-ignore" in line:
            continue

        matches = EMOJI_REGEX.findall(line)
        if matches:
            unique_emojis = list(dict.fromkeys(matches))
            findings.append({
                "file": str(rel_path),
                "filename": file_path.name,
                "line": line_idx,
                "emojis": unique_emojis,
                "code": line.strip(),
                "is_exempt": is_exempt,
            })

    return findings

def main():
    print(f"\n{BOLD}{CYAN}=================================================={RESET}")
    print(f"{BOLD}{CYAN}  Studyloop Emoji Scanner & Linter{RESET}")
    print(f"{BOLD}{CYAN}=================================================={RESET}\n")

    if not ROOT_DIR.exists():
        print(f"{RED}Error: Frontend directory not found at {ROOT_DIR}{RESET}")
        sys.exit(1)

    scanned_count = 0
    all_findings = []

    for root, _, files in os.walk(ROOT_DIR):
        for file in files:
            if file.endswith((".vue", ".js", ".ts", ".css")):
                scanned_count += 1
                results = scan_file(Path(root) / file)
                all_findings.extend(results)

    show_all = "--all" in sys.argv
    action_only = "--action-only" in sys.argv
    gamification_only = "--gamification-only" in sys.argv

    action_needed = [f for f in all_findings if not f["is_exempt"]]
    exempt_gamification = [f for f in all_findings if f["is_exempt"]]

    if action_needed and not gamification_only:
        print(f"{BOLD}{RED}[ACTION NEEDED] Static UI Emojis to replace with <BaseIcon /> ({len(action_needed)} instances):{RESET}\n")
        for item in action_needed:
            emojis_str = " ".join(item["emojis"])
            print(f"  {YELLOW}{item['file']}:{item['line']}{RESET}")
            print(f"    Emoji : {BOLD}{emojis_str}{RESET}")
            print(f"    Code  : {DIM}`{item['code'][:100]}`{RESET}\n")
    elif not gamification_only:
        print(f"{GREEN}[SUCCESS] No unauthorized static UI emojis found in components!{RESET}\n")

    if not action_only:
        print(f"{BOLD}{CYAN}[INFO] Intentional Gamification / Animated Emojis ({len(exempt_gamification)} instances):{RESET}")
        for item in exempt_gamification:
            emojis_str = " ".join(item["emojis"])
            print(f"  {DIM}{item['file']}:{item['line']} -> {emojis_str} ({item['code'][:60]}){RESET}")

    print(f"\n{BOLD}{CYAN}=================================================={RESET}")
    print(f"Files scanned: {scanned_count}")
    print(f"Status: {RED if action_needed else GREEN}{len(action_needed)} Action Needed{RESET}, {CYAN}{len(exempt_gamification)} Gamification Kept{RESET}")
    print(f"{BOLD}{CYAN}=================================================={RESET}\n")

    return 1 if action_needed else 0

if __name__ == "__main__":
    if "--help" in sys.argv or "-h" in sys.argv:
        print("Usage: python scripts/find_emojis.py [OPTIONS]")
        print("Options:")
        print("  --action-only       Only show emojis needing replacement")
        print("  --gamification-only Only show exempt gamification emojis")
        print("  --all               Show both categories in full detail")
        print("  --help, -h          Show this help message")
        sys.exit(0)
    sys.exit(main())
