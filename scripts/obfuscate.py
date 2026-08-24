#!/usr/bin/env python3
"""
Obfuscate Python extensions using PyArmor for production release.
Preserves manifest.json and requirements.txt while protecting Python entrypoint code.
"""

import os
import shutil
import subprocess
import sys
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


def ensure_pyarmor():
    """Ensure pyarmor CLI is available, attempting user pip install if missing."""
    if shutil.which("pyarmor") is not None:
        return True
    
    print("PyArmor not found in PATH. Attempting to install via pip...")
    try:
        subprocess.run([sys.executable, "-m", "pip", "install", "--quiet", "pyarmor"], check=True)
        return shutil.which("pyarmor") is not None or subprocess.run([sys.executable, "-m", "pyarmor.cli", "--version"], capture_output=True).returncode == 0
    except Exception as e:
        print(f"Notice: Could not auto-install PyArmor ({e}). Will use bytecode compilation fallback.")
        return False


from concurrent.futures import ThreadPoolExecutor, as_completed

def _process_single_extension(item: Path, output_dir: Path, has_pyarmor: bool):
    ext_id = item.name
    dest_ext_dir = output_dir / ext_id
    dest_ext_dir.mkdir(parents=True, exist_ok=True)

    # Copy non-python files (manifest.json, requirements.txt, etc.)
    for file in item.iterdir():
        if file.is_file() and file.suffix != ".py":
            shutil.copy2(file, dest_ext_dir / file.name)

    py_files = list(item.glob("*.py"))
    if not py_files:
        return f"Processed asset-only extension: {ext_id}"

    if has_pyarmor:
        try:
            py_args = [str(f.name) for f in py_files]
            if shutil.which("pyarmor"):
                cmd = ["pyarmor", "gen", "-O", str(dest_ext_dir), *py_args]
            else:
                cmd = [sys.executable, "-m", "pyarmor.cli", "gen", "-O", str(dest_ext_dir), *py_args]

            subprocess.run(cmd, check=True, cwd=str(item), capture_output=True, text=True)
            return f"  [OK] PyArmor obfuscation successful for {ext_id}"
        except Exception as e:
            print(f"  Warning: PyArmor failed for {ext_id} ({e}), falling back to standard copy/compile.")

    # Fallback: Copy .py and compile to .pyc
    for py_file in py_files:
        dest_py = dest_ext_dir / py_file.name
        shutil.copy2(py_file, dest_py)

    try:
        import compileall
        compileall.compile_dir(str(dest_ext_dir), force=True, quiet=1)
        return f"  [OK] Bytecode compilation completed for {ext_id}"
    except Exception as e:
        return f"  Notice: Bytecode compilation skipped for {ext_id} ({e})"


def obfuscate_extensions(output_dir: Path):
    """Obfuscate all extensions into the target output directory concurrently."""
    output_dir.mkdir(parents=True, exist_ok=True)
    has_pyarmor = ensure_pyarmor()

    ext_dirs = [item for item in EXTENSIONS_SRC.iterdir() if item.is_dir() and not item.name.startswith(".")]
    if not ext_dirs:
        print("No extension directories found to obfuscate.")
        return

    print(f"Obfuscating {len(ext_dirs)} extension(s) in parallel...")
    with ThreadPoolExecutor(max_workers=min(len(ext_dirs), 4)) as executor:
        future_to_ext = {
            executor.submit(_process_single_extension, item, output_dir, has_pyarmor): item.name
            for item in ext_dirs
        }
        for future in as_completed(future_to_ext):
            ext_name = future_to_ext[future]
            try:
                res = future.result()
                print(res)
            except Exception as e:
                print(f"  [ERROR] Failed to process extension {ext_name}: {e}")


def main():
    out_dir = DEFAULT_OUTPUT_DIR
    if len(sys.argv) > 1 and sys.argv[1].strip():
        out_dir = Path(sys.argv[1]).resolve()

    print(f"Obfuscating extensions from {EXTENSIONS_SRC} -> {out_dir}")
    obfuscate_extensions(out_dir)
    print("\nExtension processing finished successfully.")


if __name__ == "__main__":
    main()
