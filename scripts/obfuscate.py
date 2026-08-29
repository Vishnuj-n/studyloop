#!/usr/bin/env python3
"""
Stage and compile Python extensions for production release.
Preserves manifest.json and requirements.txt while injecting authorization secrets
and compiling portable bytecode (.pyc) for fast, reliable execution across Python runtimes.
"""

import argparse
import compileall
import json
import os
import shutil
import subprocess
import sys
import tempfile
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

try:
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    if hasattr(sys.stderr, "reconfigure"):
        sys.stderr.reconfigure(encoding="utf-8", errors="replace")
except Exception:
    pass

PROJECT_ROOT = Path(__file__).resolve().parent.parent
EXTENSIONS_SRC = PROJECT_ROOT / "extensions"
DEFAULT_OUTPUT_DIR = PROJECT_ROOT / "build" / "bin" / "extensions"


def _process_single_extension(item: Path, output_dir: Path, use_pyarmor: bool = False, secret_key: str = ""):
    ext_id = item.name
    dest_ext_dir = output_dir / ext_id
    manifest_file = item / "manifest.json"
    if manifest_file.exists():
        try:
            m_data = json.loads(manifest_file.read_text(encoding="utf-8"))
            if m_data.get("dev_only") is True:
                if dest_ext_dir.exists():
                    shutil.rmtree(dest_ext_dir, ignore_errors=True)
                return f"  [SKIP] Skipping dev-only extension from release build: {ext_id}"
        except Exception as e:
            if dest_ext_dir.exists():
                shutil.rmtree(dest_ext_dir, ignore_errors=True)
            return f"  [ERROR] Failed to parse manifest for {ext_id} ({e}); skipping extension to ensure security."

    # Clean existing destination directory for a fresh build
    if dest_ext_dir.exists():
        shutil.rmtree(dest_ext_dir, ignore_errors=True)
    dest_ext_dir.mkdir(parents=True, exist_ok=True)

    # Copy non-python files (manifest.json, requirements.txt, etc.)
    for file in item.iterdir():
        if file.is_file() and file.suffix != ".py":
            shutil.copy2(file, dest_ext_dir / file.name)

    py_files = list(item.glob("*.py"))
    if not py_files:
        return f"Processed asset-only extension: {ext_id}"

    # Stage Python files with injected build secret
    with tempfile.TemporaryDirectory() as staging_dir_str:
        staging_dir = Path(staging_dir_str)
        staged_py_names = []
        for py_file in py_files:
            content = py_file.read_text(encoding="utf-8")
            if secret_key:
                content = content.replace('__STUDYLOOP_SECRET__ = "DEV_UNRESTRICTED_TOKEN"', f'__STUDYLOOP_SECRET__ = "{secret_key}"')
            staged_py = staging_dir / py_file.name
            staged_py.write_text(content, encoding="utf-8")
            staged_py_names.append(staged_py.name)

        if use_pyarmor and (shutil.which("pyarmor") or subprocess.run([sys.executable, "-m", "pyarmor.cli", "--version"], capture_output=True).returncode == 0):
            try:
                py_args = staged_py_names
                if shutil.which("pyarmor"):
                    cmd = ["pyarmor", "gen", "-O", str(dest_ext_dir), *py_args]
                else:
                    cmd = [sys.executable, "-m", "pyarmor.cli", "gen", "-O", str(dest_ext_dir), *py_args]

                subprocess.run(cmd, check=True, cwd=str(staging_dir), capture_output=True, text=True)
                return f"  [OK] PyArmor obfuscation successful for {ext_id}"
            except Exception as e:
                print(f"  Warning: PyArmor failed for {ext_id} ({e}), falling back to portable copy/compile.")

        # Portable standard distribution: Copy staged .py and compile bytecode
        for py_file_name in staged_py_names:
            shutil.copy2(staging_dir / py_file_name, dest_ext_dir / py_file_name)

        try:
            compileall.compile_dir(str(dest_ext_dir), force=True, quiet=1)
            return f"  [OK] Staged and bytecode compiled for {ext_id}"
        except Exception as e:
            return f"  [OK] Staged python scripts for {ext_id} (compile notice: {e})"


def stage_extensions(output_dir: Path, use_pyarmor: bool = False, secret_key: str = ""):
    """Stage and compile all extensions into the target output directory concurrently."""
    output_dir.mkdir(parents=True, exist_ok=True)

    ext_dirs = [item for item in EXTENSIONS_SRC.iterdir() if item.is_dir() and not item.name.startswith(".")]
    if not ext_dirs:
        print("No extension directories found to stage.")
        return

    print(f"Staging {len(ext_dirs)} extension(s) in parallel (Secret injected: {'yes' if secret_key else 'no'}, PyArmor: {'yes' if use_pyarmor else 'no'})...")
    with ThreadPoolExecutor(max_workers=min(len(ext_dirs), 4)) as executor:
        future_to_ext = {
            executor.submit(_process_single_extension, item, output_dir, use_pyarmor, secret_key): item.name
            for item in ext_dirs
        }
        errors = []
        for future in as_completed(future_to_ext):
            ext_name = future_to_ext[future]
            try:
                res = future.result()
                print(res)
            except Exception as e:
                print(f"  [ERROR] Failed to process extension {ext_name}: {e}")
                errors.append((ext_name, e))

        if errors:
            print(f"\n[ERROR] {len(errors)} extension(s) failed during staging.")
            sys.exit(1)


def main():
    parser = argparse.ArgumentParser(description="Stage and compile StudyLoop extensions.")
    parser.add_argument("output_dir", nargs="?", default=str(DEFAULT_OUTPUT_DIR), help="Target output directory")
    parser.add_argument("--secret", default=os.environ.get("EXTENSION_SECRET_KEY", ""), help="Build-time authorization secret")
    parser.add_argument("--pyarmor", action="store_true", help="Enable PyArmor obfuscation (optional; defaults to portable bytecode)")
    args = parser.parse_args()

    out_dir = Path(args.output_dir).resolve()

    print(f"Staging extensions from {EXTENSIONS_SRC} -> {out_dir}")
    stage_extensions(out_dir, use_pyarmor=args.pyarmor, secret_key=args.secret.strip())
    print("\nExtension processing finished successfully.")


if __name__ == "__main__":
    main()

