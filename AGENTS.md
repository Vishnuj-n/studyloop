# AI Tutor — Agent Instructions

## Project & Stack
- **Architecture**: Persistent Guided Study Queue desktop app using Go (Wails) backend and Vue 3 frontend.
- **Source of Truth**: SQLite database (`dev_data/Studyloop.db` in development mode, uploads in `dev_data/uploads/`).

---

## Core Architecture Invariants (NEVER Violate)
1. **Deterministic Queue Progression**: The `study_queue` drives task order (Priority hierarchy: Task type > Notebook priority > Task priority > FIFO).
2. **Thin Frontend**: UI holds only ephemeral state. All business logic, state mutations, and validation live in the Go backend.
3. **Explicit State Mutations**: All task lifecycle changes (`PENDING → ACTIVE → COMPLETED / FAILED / SKIPPED`) are direct SQLite writes with clear audit trails.
4. **No Hidden Orchestration**: No background daemons, auto-inserters, event buses, or autonomous schedulers mutating the queue.
5. **RAG & FSRS are Data Sources**: FSRS computes intervals and inserts `FLASHCARD_REVIEW` tasks; RAG retrieves context. Neither controls workflow flow.

---

## Development Stage Rules
- **No Hardcoded Fallbacks (Fail Fast)**: Always use user-configured or database-persisted settings dynamically. Do not hardcode limits, credentials, model parameters, or configuration values. If a required setting, dependency, or configuration is missing or invalid, fail fast and return an explicit error ("if it breaks, let it break") rather than silently falling back to magic hardcoded defaults.
- **No Backward Compatibility Required**: Active development. Do not add legacy database table adapters, column fallback checks, or migration repair loops. Keep schemas clean and lean.
- **Code Integrity**: Windows builds use `extension_nocgo.go` (no CGO). `go test ./internal/...` must pass.

---

## Document Reference (Single Source of Truth)
- `doc/ARCHITECTURE.md` - System architecture, invariants, queue rules, and extension runtime
- `doc/APP_FLOW.md` - App flow and user interactions
- `doc/SCHEMA.md` - Database schema matching `internal/db/schema.go`
- `doc/DEVELOPER_ONBOARDING.md` - Local setup, uv dependencies, cloud dashboard handover
- `doc/PROJECT_STRUCTURE.md` - Source file organization
- `doc/DATA_API.md` - API endpoints and contracts
- `doc/AGENT_MAP.md` - Module responsibilities
- `doc/RAG.md` - Retrieval pipeline

**Rules for Documentation:**
**DO:** Update SSoT docs when code changes. Keep `SCHEMA.md` matching `schema.go`.
**DON'T:** Let docs drift. Duplicate info across multiple `.md` files. Create new monolithic documents.
