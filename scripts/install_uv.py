#!/usr/bin/env python3
"""
Downloads and installs the standalone `uv` binary into build/bin.
Supports Windows, macOS, and Linux (x86_64 / arm64).
"""

import os
import platform
import shutil
import sys
import tarfile
import tempfile
import urllib.request
import zipfile
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent
TARGET_DIR = PROJECT_ROOT / "build" / "bin"


def get_download_url() -> tuple[str, str]:
    """Determine the correct uv release URL and archive format for the current platform."""
    system = platform.system().lower()
    machine = platform.machine().lower()

    if system == "windows":
        ext = "zip"
        if machine in ("amd64", "x86_64", "x64"):
            arch = "x86_64-pc-windows-msvc"
        elif machine in ("arm64", "aarch64"):
            arch = "aarch64-pc-windows-msvc"
        elif machine in ("x86", "i686", "i386"):
            arch = "i686-pc-windows-msvc"
        else:
            raise RuntimeError(f"Unsupported Windows architecture: {machine}")
    elif system == "darwin":
        ext = "tar.gz"
        if machine in ("arm64", "aarch64"):
            arch = "aarch64-apple-darwin"
        else:
            arch = "x86_64-apple-darwin"
    elif system == "linux":
        ext = "tar.gz"
        if machine in ("x86_64", "amd64"):
            arch = "x86_64-unknown-linux-gnu"
        elif machine in ("aarch64", "arm64"):
            arch = "aarch64-unknown-linux-gnu"
        elif machine in ("armv7l", "armv7"):
            arch = "armv7-unknown-linux-gnueabihf"
        elif machine in ("i686", "i386"):
            arch = "i686-unknown-linux-gnu"
        else:
            arch = "x86_64-unknown-linux-gnu"
    else:
        raise RuntimeError(f"Unsupported operating system: {system}")

    filename = f"uv-{arch}.{ext}"
    url = f"https://github.com/astral-sh/uv/releases/latest/download/{filename}"
    return url, ext


def install_uv():
    TARGET_DIR.mkdir(parents=True, exist_ok=True)
    is_win = platform.system().lower() == "windows"
    binary_name = "uv.exe" if is_win else "uv"
    uvx_name = "uvx.exe" if is_win else "uvx"

    url, ext = get_download_url()
    print(f"Downloading uv from: {url}")

    with tempfile.TemporaryDirectory() as tmp_dir:
        archive_path = Path(tmp_dir) / f"uv_archive.{ext}"
        
        # Download archive with custom User-Agent
        req = urllib.request.Request(
            url,
            headers={"User-Agent": "Studyloop-UV-Installer/1.0"}
        )
        with urllib.request.urlopen(req) as resp, open(archive_path, "wb") as out_file:
            shutil.copyfileobj(resp, out_file)
        
        print(f"Downloaded archive ({archive_path.stat().st_size / 1024 / 1024:.2f} MB). Extracting...")

        extract_dir = Path(tmp_dir) / "extracted"
        extract_dir.mkdir(parents=True, exist_ok=True)

        if ext == "zip":
            with zipfile.ZipFile(archive_path, "r") as z:
                z.extractall(extract_dir)
        else:
            with tarfile.open(archive_path, "r:*") as t:
                t.extractall(extract_dir)

        # Locate uv binary inside extracted files
        found_uv = None
        found_uvx = None
        for root, _, files in os.walk(extract_dir):
            for file in files:
                if file.lower() == binary_name.lower():
                    found_uv = Path(root) / file
                elif file.lower() == uvx_name.lower():
                    found_uvx = Path(root) / file

        if not found_uv or not found_uv.exists():
            raise FileNotFoundError(f"Could not find '{binary_name}' in downloaded archive.")

        # Copy binary to TARGET_DIR
        dest_uv = TARGET_DIR / binary_name
        shutil.copy2(found_uv, dest_uv)
        if not is_win:
            dest_uv.chmod(0o755)
        print(f"Installed '{binary_name}' -> {dest_uv}")

        if found_uvx and found_uvx.exists():
            dest_uvx = TARGET_DIR / uvx_name
            shutil.copy2(found_uvx, dest_uvx)
            if not is_win:
                dest_uvx.chmod(0o755)
            print(f"Installed '{uvx_name}' -> {dest_uvx}")

    # Verify installation
    try:
        import subprocess
        result = subprocess.run([str(dest_uv), "--version"], capture_output=True, text=True, check=True)
        print(f"\nVerification successful: {result.stdout.strip()}")
    except Exception as e:
        print(f"Warning: Could not run test on downloaded binary: {e}")


if __name__ == "__main__":
    try:
        install_uv()
    except Exception as err:
        print(f"Installation failed: {err}", file=sys.stderr)
        sys.exit(1)
