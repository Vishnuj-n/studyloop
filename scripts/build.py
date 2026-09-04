#!/usr/bin/env python3
"""
Build a Wails application with an NSIS installer.

Usage:
    python scripts/build.py
    python scripts/build.py -clean
    python scripts/build.py -debug
"""

import os
import shutil
import subprocess
import sys
import secrets
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent

# ponytail: read .env directly, no third-party dotenv dependency needed
def load_env(path=PROJECT_ROOT / ".env"):
    if not path.exists():
        return {}
    return dict(
        line.strip().split("=", 1)
        for line in path.read_text(encoding="utf-8").splitlines()
        if "=" in line and not line.strip().startswith("#")
    )

_ENV = load_env()

# Production Sync & Research Analytics Credentials (loaded directly from .env)
PRODUCTION_SYNC_URL = _ENV.get("CLOUD_SYNC_URL", "")
PRODUCTION_ANON_KEY = _ENV.get("SUPABASE_ANON_KEY", "")
PRODUCTION_RESEARCH_URL = _ENV.get("RESEARCH_ANALYTICS_URL", "")
PRODUCTION_RESEARCH_ANON_KEY = _ENV.get("SUPABASE_PUBLISHABLE_KEY_LOG", _ENV.get("RESEARCH_ANALYTICS_ANON_KEY", ""))

# Extension Authorization Key (Injected into Go binary and PyArmor-obfuscated Python extensions)
EXTENSION_SECRET_KEY = os.environ.get("EXTENSION_SECRET_KEY", "").strip()
if not EXTENSION_SECRET_KEY:
    EXTENSION_SECRET_KEY = secrets.token_hex(32)


def main():
    os.chdir(PROJECT_ROOT)

    if shutil.which("wails") is None:
        print("Error: Wails CLI not found in PATH.")
        sys.exit(1)

    from concurrent.futures import ThreadPoolExecutor

    def ensure_uv():
        uv_bin = PROJECT_ROOT / "build" / "bin" / ("uv.exe" if sys.platform == "win32" else "uv")
        if not uv_bin.exists():
            print("uv binary not found in build/bin. Running scripts/install_uv.py...")
            subprocess.run([sys.executable, str(PROJECT_ROOT / "scripts" / "install_uv.py")], check=True)

    def run_obfuscation():
        obfuscate_script = PROJECT_ROOT / "scripts" / "obfuscate.py"
        if obfuscate_script.exists():
            print("Staging & Obfuscating extensions in parallel (with secure key injection)...")
            subprocess.run([sys.executable, str(obfuscate_script), "--secret", EXTENSION_SECRET_KEY], check=True)

    print("\n--- Running Pre-Build Preparation (Parallel) ---")
    with ThreadPoolExecutor(max_workers=2) as executor:
        f_uv = executor.submit(ensure_uv)
        f_obf = executor.submit(run_obfuscation)
        f_uv.result()
        f_obf.result()
    print("--- Pre-Build Preparation Complete ---\n")

    ldflags = (
        f"-X ai-tutor/internal/study.DefaultProductionSyncURL={PRODUCTION_SYNC_URL} "
        f"-X ai-tutor/internal/study.DefaultProductionAnonKey={PRODUCTION_ANON_KEY} "
        f"-X ai-tutor/internal/study.DefaultResearchAnalyticsURL={PRODUCTION_RESEARCH_URL} "
        f"-X ai-tutor/internal/study.DefaultResearchAnalyticsAnonKey={PRODUCTION_RESEARCH_ANON_KEY} "
        f"-X ai-tutor/internal/extension.ExtensionAuthKey={EXTENSION_SECRET_KEY}"
    )

    cmd = ["wails", "build", "-nsis", "-ldflags", ldflags, *sys.argv[1:]]

    sanitized_cmd = [
        arg.replace(f"ExtensionAuthKey={EXTENSION_SECRET_KEY}", "ExtensionAuthKey=***") if EXTENSION_SECRET_KEY else arg
        for arg in cmd
    ]
    print("\nExecuting:", " ".join(sanitized_cmd))

    try:
        subprocess.run(cmd, check=True)
        print("\nBuild completed successfully.")
    except subprocess.CalledProcessError as e:
        print(f"\nBuild failed with exit code {e.returncode}.")
        sys.exit(e.returncode)


if __name__ == "__main__":
    main()
