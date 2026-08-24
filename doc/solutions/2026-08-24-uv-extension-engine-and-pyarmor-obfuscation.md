# Solution: `uv`-Powered Extension Engine, PyArmor Obfuscation & Pre-Toggle Verification

**Date:** 2026-08-24  
**Author:** Vishnu J Narayanan  
**Status:** Completed & Tested

---

## 1. Problem Statement

1. **Python Dependency & Installation Barrier**:
   Desktop users frequently lack a configured Python environment on their machine or have mismatched versions (e.g. Windows Store stubs, missing pip, no PATH configuration). Requiring manual terminal commands (`winget`, `pip install`, `brew`) creates a high friction barrier for optional extensions (such as YouTube Ingestion and Audio Overview).
2. **Slow Dependency Provisioning & Global Pollution**:
   Standard `pip install` takes 20–45 seconds and risks polluting or breaking system-level Python packages.
3. **Intellectual Property & Script Protection**:
   Python extension scripts (`ingest.py`, `edge_tts_stream.py`) were previously stored in plain text, exposing internal parsing algorithms and heuristics.
4. **First-Time Toggle Verification**:
   The Extensions Hub allowed toggling extensions ON without validating whether dependencies were satisfied, leading to unexpected runtime failures.

---

## 2. Architectural Decisions & Key Choices

### A. Astral's `uv` as the Standalone Extension Engine
- **Why `uv`**: `uv` is a single, self-contained binary written in Rust (~25MB uncompressed, ~7MB in installer) that handles virtual environment creation, Python distribution resolution, and package installation 10–100x faster than standard `pip`.
- **System Python & Package Inheritance**:
  - `uv venv <venvDir> --system-site-packages` automatically reuses compatible system Python (e.g. Python 3.14 / 3.12) and any already-installed packages (`yt-dlp`, `edge-tts`) without re-downloading.
  - If Python is absent, `uv` downloads a standalone portable CPython build into user local storage.
- **Storage Isolation**:
  - Read-only binaries live in `%PROGRAMFILES64%\Studyloop\bin\uv.exe`.
  - User-writable virtual environments live in `%APPDATA%\Studyloop\extensions\<id>\.venv\` (or `./dev_data/extensions/` in dev mode).

### B. PyArmor Obfuscation in Build Pipeline
- `scripts/obfuscate.py` automatically scans `extensions/`, obfuscates Python scripts with **PyArmor** into `build/bin/extensions/`, and preserves manifests and requirements.
- `scripts/build.py` automatically checks for `build/bin/uv.exe` (downloading it via `scripts/install_uv.py` if missing), obfuscates extensions, and executes `wails build -nsis`.

### C. Reusable Verification Popup Modal
- Created `frontend/src/components/ExtensionSetupModal.vue` as a shared component.
- Visualizes the 3-step pipeline:
  1. Python Runtime Environment (`uv venv`)
  2. Package Dependencies (`uv pip install -r requirements.txt`)
  3. Extension Self-Test Probe (`--test`)
- Automatically activates toggle ON upon exit code 0.

---

## 3. Key Changes & File Map

| Component | File Path | Description |
|---|---|---|
| **Build Automation** | `scripts/install_uv.py` | Cross-platform multi-architecture `uv` binary downloader. |
| **Build Automation** | `scripts/obfuscate.py` | PyArmor extension obfuscator with fallback bytecode compilation. |
| **Build Automation** | `scripts/build.py` | Orchestrates `install_uv`, `obfuscate`, and `wails build -nsis`. |
| **Go Extension Engine** | `internal/extension/uv.go` | Locates `uv.exe` and resolves user-writable `.venv` paths. |
| **Go Extension Engine** | `internal/extension/checker.go` | `CheckReadiness`, `SetupExtensionEnv`, and smoke test runner. |
| **Go Extension Engine** | `internal/extension/runner.go` | Executes Python scripts via the extension's dedicated `.venv`. |
| **Wails API** | `internal/app/app_extension.go` | Exposes `CheckExtensionReadiness` and `SetupExtension` bindings. |
| **Extension Scripts** | `extensions/youtube/ingest.py` | Added `--test` argument handling. |
| **Extension Scripts** | `extensions/audio_overview/edge_tts_stream.py` | Added `--test` argument handling. |
| **Frontend UI** | `frontend/src/components/ExtensionSetupModal.vue` | Reusable 3-step verification modal popup. |
| **Frontend UI** | `frontend/src/pages/Extensions.vue` | Integrated pre-toggle readiness checking and setup trigger. |

---

## 4. Verification & Results

1. **Go Unit & Integration Tests**:
   - `go test ./...` passed with exit code 0.
2. **PyArmor Obfuscation Test**:
   - `python scripts/obfuscate.py` successfully obfuscated `youtube` and `audio_overview` extensions into `build/bin/extensions/` with PyArmor runtime.
3. **Frontend Production Build**:
   - `npm --prefix frontend run build` built all Vue components and assets into `dist/` with exit code 0.
