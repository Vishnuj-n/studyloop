# Solution: Enforce Invariant Boundaries on Analytics Fallback and Profile Resolution

## Context
An architectural audit identified potential invariant leaks and silent fallbacks:
1. `syncAnalyticsFallback` ran when `CloudSyncURL` was unconfigured, risking calls to unconfigured endpoints if research tokens/URLs were empty or missing.
2. `GetUserSettings` performed hidden database writes (`UPDATE user_settings SET active_profile_id = ...`) during read operations to resolve a default profile when `active_profile_id` was empty or missing.

## Changes Made

### 1. Telemetry & Analytics Fallback Clean Skipping
- **[`internal/study/sync.go`](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/study/sync.go)**:
  - Added strict check in `syncAnalyticsFallback` to ensure that if `researchURL == ""` or `researchToken == ""`, the method immediately returns `nil` (no-op) without attempting network calls or throwing errors.
  - In production builds, `scripts/build.py` provides build-time `-ldflags` injection, while in development mode, dynamic resolution checks `.env` variables cleanly.

### 2. Read Purity & Profile Isolation
- **[`internal/db/store.go`](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/db/store.go)**:
  - Removed silent `UPDATE user_settings SET active_profile_id = ...` operations from `GetUserSettings()`.
  - When `ActiveProfileID` does not exist in `study_profiles`, it simply resets the in-memory field to `""` rather than mutating SQLite records during a getter query.
  - Setting an active profile remains an explicit mutation via `SetActiveProfileID`.

## Verification
- Ran full short test suite:
  ```bash
  go test -short ./internal/...
  ```
- All packages passed with 0 errors.
