#!/usr/bin/env python3
"""
AI-Powered Version Bump & Release Automation.

Workflow:
1. Verify repository is on 'main' branch (or --force).
2. Get commit history since the latest git tag.
3. Read current version from internal/app/VERSION.
4. Prompt AI to analyze commits and recommend patch (e.g., 1.3.12) vs minor/major (e.g., 1.4.0) in JSON.
5. Update internal/app/VERSION (and root VERSION if present).
6. Commit & push updated VERSION file to remote branch.
7. Run scripts/build.py.
8. Run scripts/release.py.

CLI Options:
    --dry-run   Perform check and AI analysis without modifying files or building/releasing.
    --force     Bypass main branch check.
    --revert [TAG]  Revert a release: deletes GitHub release, remote tag, local tag, and resets VERSION to previous tag.

Usage:
    python scripts/smart_release.py
    python scripts/smart_release.py --dry-run
    python scripts/smart_release.py --revert
    python scripts/smart_release.py --revert v1.3.12
"""

import argparse
import json
import os
import re
import subprocess
import sys
from pathlib import Path

# Ensure UTF-8 encoding for standard output/error streams on Windows
if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")
if hasattr(sys.stderr, "reconfigure"):
    sys.stderr.reconfigure(encoding="utf-8")

PROJECT_ROOT = Path(__file__).resolve().parent.parent
os.chdir(PROJECT_ROOT)

from openai import OpenAI

DEFAULT_MODEL = "openai/gpt-oss-120b"


def load_env_file(filepath=".env"):
    """Load key-value pairs from .env file into os.environ if present."""
    env_path = PROJECT_ROOT / filepath
    if not env_path.exists():
        return
    with open(env_path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, val = line.split("=", 1)
            key = key.strip()
            val = val.strip().strip("'\"")
            if key and key not in os.environ:
                os.environ[key] = val


load_env_file()


def run_cmd(cmd, check=True, capture=False):
    """Run a shell command and return CompletedProcess."""
    result = subprocess.run(
        cmd,
        check=check,
        stdout=subprocess.PIPE if capture else None,
        stderr=subprocess.PIPE if capture else None,
        text=True,
        encoding="utf-8",
        errors="replace",
    )
    return result


def get_current_branch():
    """Return the active git branch name."""
    try:
        res = run_cmd(["git", "rev-parse", "--abbrev-ref", "HEAD"], capture=True)
        return res.stdout.strip()
    except subprocess.CalledProcessError:
        return ""


def get_latest_tag():
    """Return latest git tag or None."""
    try:
        return run_cmd(
            ["git", "describe", "--tags", "--abbrev=0"],
            capture=True,
        ).stdout.strip()
    except subprocess.CalledProcessError:
        return None


def get_all_tags_descending():
    """Return all git tags ordered by version descending."""
    try:
        raw = run_cmd(["git", "tag", "-l", "--sort=-v:refname"], capture=True).stdout.strip()
        return [t.strip() for t in raw.splitlines() if t.strip()]
    except subprocess.CalledProcessError:
        return []


def get_commit_history(previous_tag=None):
    """Return detailed commit history since previous tag."""
    cmd = [
        "git",
        "log",
        "--no-merges",
        "--pretty=format:Commit: %h%nAuthor: %an%nSubject: %s%nBody:%n%b%n---",
    ]

    if previous_tag:
        cmd.append(f"{previous_tag}..HEAD")

    return run_cmd(cmd, capture=True).stdout.strip()


def get_current_version():
    """Read current version string from VERSION file."""
    version_file = PROJECT_ROOT / "internal" / "app" / "VERSION"
    if not version_file.exists():
        version_file = PROJECT_ROOT / "VERSION"

    if not version_file.exists():
        raise FileNotFoundError("Neither internal/app/VERSION nor VERSION file exists.")

    return version_file.read_text(encoding="utf-8").strip()


def update_version_files(new_version):
    """Update internal/app/VERSION (and root VERSION if present)."""
    curr_ver = get_current_version()
    has_v = curr_ver.startswith("v")

    clean_ver = new_version.lstrip("v")
    formatted_ver = f"v{clean_ver}" if has_v else clean_ver

    version_file = PROJECT_ROOT / "internal" / "app" / "VERSION"
    if version_file.exists():
        version_file.write_text(formatted_ver + "\n", encoding="utf-8")
        print(f"Updated {version_file.relative_to(PROJECT_ROOT)} -> {formatted_ver}")

    root_file = PROJECT_ROOT / "VERSION"
    if root_file.exists():
        root_file.write_text(formatted_ver + "\n", encoding="utf-8")
        print(f"Updated {root_file.relative_to(PROJECT_ROOT)} -> {formatted_ver}")

    return formatted_ver


def parse_json_response(raw_text):
    """Extract and parse JSON object from LLM output."""
    raw_text = raw_text.strip()
    if raw_text.startswith("```"):
        raw_text = re.sub(r"^```(?:json)?\n?", "", raw_text, flags=re.IGNORECASE)
        raw_text = re.sub(r"\n?```$", "", raw_text)

    start_idx = raw_text.find("{")
    end_idx = raw_text.rfind("}")
    if start_idx != -1 and end_idx != -1:
        raw_text = raw_text[start_idx : end_idx + 1]

    return json.loads(raw_text)


def ask_ai_version_bump(current_version, commit_history):
    """Use AI to analyze git history and recommend the next version in JSON format."""
    api_key = os.getenv("FAST_LLM_API_KEY") or os.getenv("HEAVY_LLM_API_KEY")
    base_url = (
        os.getenv("FAST_LLM_BASE_URL")
        or os.getenv("HEAVY_LLM_BASE_URL")
        or "https://api.openai.com/v1"
    )

    if base_url:
        base_url = base_url.rstrip("/")
        if base_url.endswith("/openai"):
            base_url = base_url[:-7] + "/openai/v1"

    if not api_key:
        raise RuntimeError("Neither FAST_LLM_API_KEY nor HEAVY_LLM_API_KEY environment variable is set.")

    model = (
        os.getenv("FAST_LLM_MODEL")
        or os.getenv("HEAVY_LLM_MODEL")
        or DEFAULT_MODEL
    )

    client = OpenAI(
        api_key=api_key,
        base_url=base_url,
    )

    prompt = f"""
You are a release management expert for software projects using Semantic Versioning (SemVer).

Current Version: {current_version}

Commit Log since last tag/release:
{commit_history}

Instructions:
1. Analyze the changes in the commit log.
2. Determine whether the next release should be a patch bump (e.g., 1.3.11 -> 1.3.12), a minor bump (e.g., 1.3.11 -> 1.4.0), or a major bump (e.g., 1.3.11 -> 2.0.0).
3. Return ONLY a valid JSON object matching this structure:
{{
  "current_version": "{current_version}",
  "recommended_version": "<NEW_VERSION_STRING_WITHOUT_V>",
  "bump_type": "patch|minor|major",
  "reason": "<Short sentence explaining why this bump was chosen>"
}}

Do NOT include extra commentary or Markdown wrappers outside the JSON.
"""

    response = client.chat.completions.create(
        model=model,
        temperature=0.2,
        messages=[
            {
                "role": "system",
                "content": "You analyze git commits and determine Semantic Version bumps. Respond only with JSON.",
            },
            {"role": "user", "content": prompt},
        ],
    )

    content = response.choices[0].message.content
    return parse_json_response(content)


def revert_release(target_tag=None, dry_run=False):
    """Revert a release: delete GitHub release, delete remote & local tags, and revert VERSION file."""
    tags = get_all_tags_descending()

    if not tags:
        print("Error: No git tags found in repository.", file=sys.stderr)
        sys.exit(1)

    if not target_tag:
        target_tag = tags[0]
    elif not target_tag.startswith("v"):
        target_tag = "v" + target_tag

    if target_tag not in tags:
        print(f"Error: Tag '{target_tag}' does not exist in local git tags.", file=sys.stderr)
        sys.exit(1)

    tag_idx = tags.index(target_tag)
    prev_tag = tags[tag_idx + 1] if tag_idx + 1 < len(tags) else None

    print(f"\n=== Reverting Release: {target_tag} ===")
    if prev_tag:
        print(f"Target fallback version (previous tag): {prev_tag}")
    else:
        print("Warning: No earlier tag found to fall back to.")

    if dry_run:
        print(f"\n[DRY RUN] Actions that would be taken for {target_tag}:")
        print(f" 1. Delete GitHub release: gh release delete {target_tag} -y --cleanup-tag")
        print(f" 2. Delete remote tag: git push origin :refs/tags/{target_tag}")
        print(f" 3. Delete local tag: git tag -d {target_tag}")
        if prev_tag:
            print(f" 4. Update VERSION file: internal/app/VERSION -> {prev_tag}")
        return

    # 1. Delete GitHub Release
    print(f"Deleting GitHub release for {target_tag}...")
    run_cmd(["gh", "release", "delete", target_tag, "-y", "--cleanup-tag"], check=False)

    # 2. Delete Remote Tag
    print(f"Deleting remote tag {target_tag} from origin...")
    run_cmd(["git", "push", "origin", f":refs/tags/{target_tag}"], check=False)

    # 3. Delete Local Tag
    print(f"Deleting local tag {target_tag}...")
    run_cmd(["git", "tag", "-d", target_tag], check=False)

    # 4. Revert VERSION file
    if prev_tag:
        update_version_files(prev_tag)

    print(f"\nSuccessfully reverted release {target_tag}!")


def main():
    parser = argparse.ArgumentParser(
        description="Smart AI-driven version bump and release workflow."
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Bypass branch check (allow running on non-main branches)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Perform dry run without modifying files, tags, or executing release commands",
    )
    parser.add_argument(
        "--revert",
        nargs="?",
        const="LATEST",
        default=None,
        metavar="TAG",
        help="Revert release: deletes GitHub release, remote & local tag, and resets VERSION to previous tag",
    )

    args = parser.parse_args()

    # Handle Revert flow
    if args.revert is not None:
        target_tag = None if args.revert == "LATEST" else args.revert
        revert_release(target_tag, dry_run=args.dry_run)
        return

    # 1. Branch Check
    branch = get_current_branch()
    print(f"Current branch: {branch}")
    if branch != "main" and not args.force:
        print("Error: Smart release script must be run on 'main' branch. Use --force to override.", file=sys.stderr)
        sys.exit(1)

    # 2. Previous Tag & Commits
    latest_tag = get_latest_tag()
    print(f"Latest release tag: {latest_tag or 'None'}")

    commit_history = get_commit_history(latest_tag)
    if not commit_history:
        print("No new commits found since last tag. Nothing to release.")
        sys.exit(0)

    # 3. Read Current Version
    curr_version = get_current_version()
    print(f"Current VERSION file content: {curr_version}")

    # 4. Ask AI for Version Bump
    print("\nQuerying AI model for semantic version bump recommendation...")
    try:
        ai_recommendation = ask_ai_version_bump(curr_version, commit_history)
    except Exception as e:
        print(f"Error querying AI model: {e}", file=sys.stderr)
        sys.exit(1)

    print("\n--- AI Version Decision ---")
    print(json.dumps(ai_recommendation, indent=2))

    rec_version = ai_recommendation.get("recommended_version")
    bump_type = ai_recommendation.get("bump_type")
    reason = ai_recommendation.get("reason")

    if not rec_version:
        print("Error: AI did not return a valid recommended_version.", file=sys.stderr)
        sys.exit(1)

    print(f"\nDecision: Bump [{bump_type.upper()}] to {rec_version} ({reason})")

    if args.dry_run:
        print("\n[DRY RUN] Skipping version file updates, git commit/push, build.py, and release.py.")
        return

    # 5. Update Version File & Commit to Remote
    formatted_new_ver = update_version_files(rec_version)

    print(f"\nCommitting version update ({formatted_new_ver}) and pushing to remote branch '{branch}'...")
    run_cmd(["git", "add", "internal/app/VERSION"], check=False)
    root_file = PROJECT_ROOT / "VERSION"
    if root_file.exists():
        run_cmd(["git", "add", "VERSION"], check=False)
    run_cmd(["git", "commit", "-m", f"chore(release): bump version to {formatted_new_ver}"], check=False)
    run_cmd(["git", "push", "origin", branch], check=False)

    # 6. Run build.py
    print("\n=== Step 1/2: Running scripts/build.py ===")
    run_cmd([sys.executable, str(PROJECT_ROOT / "scripts" / "build.py")], check=True)

    # 7. Run release.py
    print("\n=== Step 2/2: Running scripts/release.py ===")
    run_cmd([sys.executable, str(PROJECT_ROOT / "scripts" / "release.py"), formatted_new_ver], check=True)

    print(f"\nSmart release process completed successfully for {formatted_new_ver}!")


if __name__ == "__main__":
    main()
