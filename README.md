# 🔄 StudyLoop

### **You shouldn't look at your notes 6 months from now and ask "What was that?"**

Most AI study tools give you the illusion of learning. You upload a 300-page textbook to a NotebookLM clone, chat with a bot, generate a neat summary, and feel productive. But passive consumption doesn't create memory. Six months later, when exams hit or you need the knowledge in production, it's completely gone.

**StudyLoop is not an open-source NotebookLM or a PDF chat sandbox.**  
It is a local-first **active learning and long-term retention engine** designed so you actually remember what you study months and years down the line.

---

## 🔁 The Knowledge Intake & Retention Loop

StudyLoop turns passive source material (textbooks, lecture videos, markdown notes) into a deterministic daily queue powered by cognitive science:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      MULTI-SOURCE KNOWLEDGE INTAKE                      │
│                                                                         │
│   📄 Textbooks & PDFs        📝 Markdown Notes       🎥 Video Lectures  │
│   (Standard & Deep OCR)        (Native .md notes)       (YouTube + yt-dlp)│
└────────────────────────────────────┬────────────────────────────────────┘
                                     │ Auto-Chunk & Seed Queue
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        THE ACTIVE RETENTION LOOP                        │
│                                                                         │
│   [ 1. LEARN / READ ] ──▶ Focus-locked reading & timestamped video      │
│            │                                                            │
│            ▼                                                            │
│   [ 2. VERIFY ]       ──▶ AI Checkpoint Quizzes & Socratic Rescue       │
│            │              (2-strike rule prevents false mastery)        │
│            ▼                                                            │
│   [ 3. RETAIN ]       ──▶ FSRS Spaced Repetition Flashcards             │
│                           (Targeted, long-term memory encoding)         │
└─────────────────────────────────────────────────────────────────────────┘
```

> **The Core Difference:** NotebookLM is built for *referencing* documents. StudyLoop is built for *encoding knowledge into your long-term memory*.

---

## ⚡ Why StudyLoop?

| Dimension | NotebookLM / PDF Chatbots | Flashcard Apps (Anki) | 🔄 **StudyLoop** |
| :--- | :--- | :--- | :--- |
| **Primary Goal** | Passive Q&A / Summaries | Rote flashcard review | **End-to-End Long-Term Retention** |
| **Supported Sources** | PDFs / Docs | Manual card creation | **PDFs (OCR), Markdown (`.md`), YouTube Lectures** |
| **Learning Workflow** | Open-ended conversational sandbox | Flashcard queue only | **Deterministic Loop (Read ➔ Quiz ➔ Socratic Rescue ➔ FSRS)** |
| **Failure Intervention** | None (gives you answers) | Manual card reset | **2-Strike Socratic Rescue (Forces you to understand)** |
| **Data Privacy** | Cloud-hosted / Vendor AI | Local sync / plugins | **100% Local-First (SQLite + ONNX embeddings + OS Keyring)** |

---

## 🌟 Core Pillars & Features

### 1. Multi-Source Knowledge Ingestion
- **📄 Textbooks & PDFs:** Standard layout extraction plus Deep Structured OCR (`deep_pdf`) with heading preservation.
- **📝 Markdown Notes (`.md`):** Ingest raw markdown documentation, obsidian notes, or course notes with table and code-block integrity.
- **🎥 YouTube Video Lectures:** Paste video URLs to extract timestamped chapters, synchronized transcripts, and offline cached playback via `yt-dlp`.

### 2. Deterministic Study Queue
- **Zero Decision Fatigue:** The SQLite-backed `study_queue` serves your next highest-priority task automatically.
- **Multi-Notebook Priorities:** Balance multiple subjects with configurable notebook weights (1–10).
- **Starvation Protection:** Ensures new reading tasks never get indefinitely buried beneath heavy review loads.

### 3. Mastery Verification & Socratic Rescue
- **Checkpoint Quizzes:** Dynamic AI-generated quizzes immediately following reading sessions.
- **2-Strike Concept Rescue:** If you fail a quiz twice, the queue blocks progression and launches a Socratic intervention to fix the conceptual flaw before letting you advance.
- **Examiner Assessments:** Open-ended written mastery assessments evaluated with rigorous criteria.

### 4. Spaced Repetition (FSRS Engine)
- **Modern Spaced Repetition:** Powered by the state-of-the-art **FSRS-4** algorithm (Free Spaced Repetition Scheduler), outperforming legacy SM-2.
- **Automatic Deck Generation:** Flashcards generated directly from your ingested chapters and reading blocks.
- **Integrated Explanations:** In-card AI deep-dives for any flashcard concept you struggle with.

### 5. Local-First, Private & Fast
- **Local Data & Embeddings:** SQLite database (`Studyloop.db`) with local ONNX INT8 vector embeddings (`sqlite-vec`).
- **Dual-Tier LLM Architecture:** Route quick quizzes through ultra-fast models (e.g. Groq / Llama 3) and deep Socratic evaluations through reasoning models (OpenAI / OpenRouter).
- **Secure Key Storage:** API credentials stored securely in your operating system's native keyring.

---

## 📡 Local-First and Offline Behavior

**Works 100% Offline:**
- Reading textbook & markdown chunks
- Video playback (cached YouTube lectures)
- FSRS flashcard reviews & scheduling
- Queue progression, notes, and local vector searches

**Requires Internet:**
- AI Socratic tutor & Ask-AI queries
- Dynamic quiz generation & Examiner grading

**Failure rule:**

- If AI is unavailable, show clear error and do not simulate output

## Tech Stack

- Go 1.26
- Wails v2
- Vue 3 (multi-page, hash-based routing)
- SQLite + sqlite-vec (vec0 extension)
- ONNX Runtime (local INT8 embedding model)
- go-fsrs/v4 (FSRS spaced repetition algorithm)
- OpenAI-compatible LLM API (dual-tier: Fast + Heavy)
- go-keyring (OS keyring for API key storage)

## Quick Start

### Prerequisites

- Go 1.26+
- Node.js 20+
- Wails CLI
- CGO-capable compiler toolchain (macOS/Linux) or pure Go build (Windows via `db/extension_nocgo.go`)
- Local RAG assets in `asset/`:
	- `tokenizer.json`
	- `model_int8.onnx`
	- `onnxruntime.dll` (Windows) / `libonnxruntime.dylib` (macOS) / `libonnxruntime.so` (Linux)
	- `vec0.dll` (Windows) / `vec0.dylib` (macOS) / `vec0.so` (Linux)

Run dependency checks and asset download:

```bash
# macOS / Linux
./scripts/sync-deps.sh

# Windows (pure Go, no CGO required)
./scripts/windows-sync-deps.ps1
```

### Development

```bash
wails doctor
wails dev -tags sqlite_extension
```

### Build

```bash
wails build -tags sqlite_extension
```

## Local RAG Troubleshooting

- `Ask AI unavailable` on startup:
	- Run `./scripts/sync-deps.sh` (or `./scripts/windows-sync-deps.ps1` on Windows)
	- Confirm all required files exist under `asset/`
- `no such module: vec0`:
	- Ensure build includes `-tags sqlite_extension`
	- Ensure platform-specific `vec0` library exists in `asset/`
- ONNX runtime load failure:
	- Ensure platform-specific ONNX runtime library exists in `asset/`
	- Rebuild with `CGO_ENABLED=1` (macOS / Linux only; Windows uses pure Go `extension_nocgo.go`)
- Build fails due to missing C compiler:
	- On macOS/Linux: Install Xcode Command Line Tools or build-essential
	- On Windows: Windows builds do not require a C compiler or CGO; ensure build uses `db/extension_nocgo.go`

## Documentation

- Developer Onboarding & Handover: [doc/DEVELOPER_ONBOARDING.md](doc/DEVELOPER_ONBOARDING.md)
- System design & Invariants: [doc/ARCHITECTURE.md](doc/ARCHITECTURE.md)
- App flow and user interactions: [doc/APP_FLOW.md](doc/APP_FLOW.md)
- Database schema: [doc/SCHEMA.md](doc/SCHEMA.md)
- Source structure: [doc/PROJECT_STRUCTURE.md](doc/PROJECT_STRUCTURE.md)
- API contracts: [doc/DATA_API.md](doc/DATA_API.md)
- Module responsibilities: [doc/AGENT_MAP.md](doc/AGENT_MAP.md)
- Retrieval pipeline & RAG: [doc/RAG.md](doc/RAG.md)

## Open-Source Credits & Acknowledgments

- **yt-dlp**: Video lecture metadata and transcript extraction tooling is powered by the open-source [yt-dlp](https://github.com/yt-dlp/yt-dlp) project.
- **sqlite-vec**: Fast local embedding vector storage by [sqlite-vec](https://github.com/asg017/sqlite-vec).
- **FSRS**: Modern spaced repetition scheduling via [go-fsrs](https://github.com/open-spaced-repetition/go-fsrs).

## Constraints

- Keep the system simple and implementation-ready
- Avoid unnecessary abstraction and premature optimization
- Do not use LangChain, agent orchestration, or chatbot-style memory