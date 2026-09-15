# Contributing to StudyLoop 🔄

Thank you for your interest in contributing to **StudyLoop**! We are building a local-first active learning and long-term retention desktop app using Go, Wails v2, SQLite, and Vue 3.

Whether you want to fix a bug, polish the UI, add an importer extension, or improve documentation, we welcome your help!

---

## ⚡ 3-Minute Quickstart

### 1. Prerequisites
Make sure you have the following installed:
* **Go 1.22+** (Go 1.26 recommended)
* **Node.js 20+** & `npm`
* **uv** (Fast Python package manager) — [Install uv](https://docs.astral.sh/uv/getting-started/installation/)
* **Wails CLI v2**:
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

### 2. Clone & Sync Assets
```bash
git clone https://github.com/Vishnuj-n/studyloop.git
cd studyloop
```

Download required runtime assets (vector extension libraries & INT8 models):
* **Windows (PowerShell):**
  ```powershell
  ./windows-sync-deps.ps1
  ```
* **macOS / Linux:**
  ```bash
  chmod +x ./sync-deps.sh
  ./sync-deps.sh
  ```

### 3. Run the App
```bash
wails dev -tags sqlite_extension
```
*Note: If you run into CGO issues on Windows during full builds, see the [CGO Toolchain Guide](#-cgo--build-toolchains-guide) below.*

---

## 🧪 Fast Testing & Verification

You can verify and test backend code in under 2 seconds without full CGO compilation:

```bash
# Instant compilation & type-checking without test execution
go test -run=^$ ./internal/...

# Rapid (<2s) unit test verification
go test -short ./internal/...

# Frontend tests
cd frontend
npm test
```

---

## 🛠️ Contribution Tracks (Pick Your Flavor!)

Whether you're into frontend, backend, or developer tooling, here are the primary areas where you can contribute:

### 🎨 Track 1: Vue 3 Frontend & UI/UX
* **Tech:** Vue 3 (Composition API), Vite, Tailwind/CSS design tokens.
* **Tasks:** 
  * Keyboard navigation (`Space` to flip flashcards, `1-4` for FSRS ratings in Flashcards/Quiz).
  * Dark & Light theme polish, responsive reading layouts, and accessibility improvements.
  * Reader component UX & study timer enhancements.
* **Location:** [`frontend/src/`](./frontend/src/).

### 🐹 Track 2: Go Backend & SQLite Engine
* **Tech:** Go, SQLite (`sqlite-vec`), FSRS-4 algorithm.
* **Tasks:** 
  * Spaced repetition scheduling (`internal/study/fsrs.go`) & queue optimizations.
  * Local INT8 ONNX embeddings & vector similarity search pipelines (`internal/rag/`).
  * Ingestion parsers & document chunking algorithms (`internal/rag/chunking.go`).
* **Location:** [`internal/`](./internal/).

### 📖 Track 3: Cross-Platform Builds & Documentation
* **Tech:** Shell scripts, PowerShell, GitHub Actions, Markdown.
* **Tasks:** 
  * Testing and improving build setups for Linux & macOS.
  * Adding unit and integration tests across the Go and Vue test suites.
  * Improving onboarding documentation and developer workflows.

---

## 💡 CGO & Build Toolchains Guide

StudyLoop uses SQLite with local vector search (`sqlite-vec` / `vec0`), which uses CGO for native desktop builds:

* **Windows:**
  * Install MinGW-w64 via Winget: `winget install MSYS2.MSYS2` or Scoop: `scoop install mingw`
  * Or install Visual Studio C++ Build Tools.
* **macOS:**
  * Install Xcode Command Line Tools: `xcode-select --install`
* **Linux:**
  * Install GCC/build tools: `sudo apt install build-essential` (Ubuntu/Debian) or `sudo pacman -S base-devel` (Arch).

---

## 📚 Deep Documentation & Architecture

For deep architectural rules and developer onboarding:
* **[Developer Onboarding & Cloud Sync Guide](doc/DEVELOPER_ONBOARDING.md)**
* **[System Architecture & Invariants](doc/ARCHITECTURE.md)**
* **[Database Schema Reference](doc/SCHEMA.md)**
* **[Queue Rules & Lifecycle Map](doc/AGENT_MAP.md)**

---

## 💬 Community & PR Guidelines

1. **Check existing issues:** Look for issues tagged `good first issue` or `help wanted`.
2. **Open an issue first for major changes:** If you plan to add large features or architectural shifts, open an issue first so we can align on design.
3. **Run tests before submitting:** Ensure `go test -short ./internal/...` and frontend lint/tests pass.
4. **All contributors welcomed:** Every contribution (docs, bug reports, features) gets acknowledged in our README!
