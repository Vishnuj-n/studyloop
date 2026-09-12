# Startup Database Backup (`Studyloop.db.bak`)

## Problem

Unexpected system crashes, power loss, or bugs during app operations could potentially corrupt the primary SQLite database (`Studyloop.db`), leading to loss of study progress, flashcards, and user settings. 

## Solution Architecture

We implemented a lightweight, non-blocking startup database backup mechanism (`db.BackupDatabase`) that runs during app bootstrap before any SQLite connections or vec0 extensions are initialized.

### 1. Fail-Safe Startup Backup Function
- Created `internal/db/backup.go` providing `db.BackupDatabase(dbPath string) error`.
- If `Studyloop.db` does not exist (e.g. first app launch), it gracefully returns `nil` without failing.
- Uses standard stream copying (`io.Copy`) to duplicate `Studyloop.db` to `Studyloop.db.bak`.
- Overwrites any pre-existing `Studyloop.db.bak` file cleanly on each boot.

### 2. Zero-Lag Boot Integration
- In `internal/runtime/boot.go`, `db.BackupDatabase(dbPath)` is invoked immediately after `ResolveDBPath()` and prior to `db.Init(dbPath, "")`.
- If an error occurs during backup (e.g. permission issues or disk full), it logs a warning via `utils.Warnf` and continues launching the app normally without blocking startup.

---

## File Changes Map

| Module / Path | Description |
|---|---|
| `internal/db/backup.go` | Implemented `BackupDatabase(dbPath)` using standard file stream duplication |
| `internal/db/backup_test.go` | Unit tests for non-existent DB handling, initial creation, and overwrite handling |
| `internal/runtime/boot.go` | Trigger `db.BackupDatabase` in `runtime.Bootstrap` prior to `db.Init` |

---

## Data Flow Diagram

```mermaid
sequenceDiagram
    participant Boot as runtime.Bootstrap (boot.go)
    participant Backup as db.BackupDatabase (backup.go)
    participant Disk as File System
    participant DB as db.Init (schema.go)

    Boot->>Disk: ResolveDBPath()
    Boot->>Backup: BackupDatabase(dbPath)
    alt DB file exists
        Backup->>Disk: Open Studyloop.db
        Backup->>Disk: Create/Overwrite Studyloop.db.bak
        Backup->>Disk: io.Copy contents
        Backup-->>Boot: Success (logged via Warnf/Infof)
    else First boot (DB missing)
        Backup-->>Boot: Return nil
    end
    Boot->>DB: db.Init(dbPath, "")
```

---

## Verification

- **Unit Tests**:
  - `go test -v ./internal/db -run TestBackupDatabase` verified non-existent file handling and content duplication/overwriting.
  - `go test -short ./internal/...` verified repo-wide compilation and test passing.
