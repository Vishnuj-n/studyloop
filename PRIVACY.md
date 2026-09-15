# Privacy Policy — Studyloop

Studyloop is an open-source, local-first study workspace designed with privacy as a foundational guarantee.

---

## 1. Local-First & 100% Offline Capable
- **Your Notes & PDFs**: All study documents, textbooks, markdown files, and YouTube transcripts are stored locally in your app data directory (`SQLite` database and `uploads/` directory).
- **Your AI Conversations**: Socratic rescue sessions, reading queries, and quiz generations are sent strictly to the AI provider endpoint that you explicitly configure (e.g., Google AI Studio, Groq, OpenRouter, OpenAI, or local Ollama).
- **Your API Keys**: Stored securely in your operating system's native credential manager (Windows Credential Manager / macOS Keychain / Linux Secret Service) — never in plain text SQLite.

---

## 2. Minimal Anonymous Heartbeat (Tier 1)
To understand if people are using the app and which operating systems to maintain, Studyloop sends a tiny, non-intrusive launch ping when the app starts:

- **What is collected**:
  - `installation_id`: A randomly generated UUID created once on first run.
  - `event`: `"app_started"`
  - `app_version`: App release version (e.g. `1.6.0`).
  - `platform`: Operating system (`windows`, `linux`, `darwin`).
  - `architecture`: CPU architecture (`amd64`, `arm64`).
  - `created_at`: Timestamp.

- **What is NEVER collected**:
  - ❌ No files, textbook titles, or PDF content
  - ❌ No notes, study highlights, or reading data
  - ❌ No user prompts or AI responses
  - ❌ No personal names, email addresses, or account credentials
  - ❌ No IP address tracking or geolocations

---

## 3. Study Insights Telemetry (Tier 2 - Opt-In Only)
You can optionally choose to share high-level study metrics (like quiz completion rates or spaced repetition intervals) during onboarding or in **Settings → Community & Feedback**. 

- Default is **OFF**.
- If disabled, zero feature analytics events are recorded or transmitted.

---

## 4. Open-Source Verification
Studyloop is open-source software. You can inspect the entire telemetry and sync implementation in:
- `internal/telemetry/heartbeat.go`
- `internal/study/sync.go`
- `cloud-dashboard/supabase/telemetry_schema.sql`

If you have questions, please open an issue on [GitHub](https://github.com/Vishnuj-n/studyloop/issues).
