# Solution: Exam Feasibility Pacing Engine, Multi-Slot Study Routine, and Quiet Telemetry

## Overview
Implemented an exam feasibility pacing system and multi-slot daily study routine on branch `feature/exam-pacing-multislot-schedule`. The solution replaces static word-per-minute estimates with workload-based session forecasting (~3,000 words/session), enables morning/evening multi-window study routines with in-app boundary notifications and calendar export (.ics, Google, Outlook), and introduces a quiet, click-to-expand telemetry drawer on the main dashboard to keep the interface calm and uncrowded.

---

## Architectural Principles & Invariants Preserved
1. **Queue Remains the Single Source of Truth**: The study routine defines *when* the user prefers to study (notifications, alarms, calendar events). It does **not** autonomously inject or mutate items in `study_queue`. The SQLite deterministic queue drives *what* to study.
2. **Workload-Based Feasibility (Not Arbitrary WPM)**: Pacing is calculated from completed reading tasks over the past 7 days (`GetProfileCompletedReadingStatsPastNDays`) against remaining words across active notebooks, converted to discrete session workloads (~3,000 words per session).
3. **Strict Backward Compatibility**: Retained `study_start_time` and `study_end_time` in DB schema, models, and UI fallback when `study_slots_json` is empty or legacy single-window schedules are loaded. Added schema migration to `alterStatements` to automatically update existing user databases on startup.
4. **Quiet, Non-Intrusive UI**: Rather than loud dashboard cards, feasibility telemetry is rendered as a clean interactive pill in the top header action strip. Detailed metrics, multi-window slot editing, and calendar sync open on demand in a sliding drawer popover.

---

## Changes Made

### 1. Database Schema & Migration
- **`internal/db/schema.go`**:
  - Added `study_slots_json TEXT DEFAULT '[]'` column definition to `user_settings` table.
  - Added entry to `alterStatements`:
    ```go
    {"user_settings", "study_slots_json", "ALTER TABLE user_settings ADD COLUMN study_slots_json TEXT DEFAULT '[]'"}
    ```
    Guarantees automatic, non-destructive migration on existing user databases via `columnExists`.
- **`internal/models/models.go`**:
  - Added `StudySlotsJSON string` field with json tag `study_slots_json` to `UserSettings` struct.
- **`internal/db/store.go`**:
  - Updated `GetUserSettings` and `UpdateUserSettings` queries to persist and scan `study_slots_json` while preserving all parameters and legacy defaults.
- **`internal/db/study_queue_queries.go`**:
  - Implemented `GetProfileCompletedReadingStatsPastNDays(profileID, days)` to query actual completed reading tasks in the past 7 days (`tasks_completed`, `words_read`, `distinct_days_active`).

### 2. Backend Feasibility Engine
- **`internal/app/notebook_endpoints.go`**:
  - Refactored `GetProfileDailyPace` to return comprehensive feasibility and session-based velocity:
    - `remaining_sessions`: Total remaining words divided by `target_session_words` (default 3,000).
    - `current_daily_sessions`: Actual 7-day completed reading tasks / 7.
    - `required_daily_sessions`: Remaining sessions / days left until deadline.
    - `feasibility_status`: Evaluated as `NO_DATA` (less than 2 completed sessions in past 7 days), `AHEAD`, `ON_TRACK`, or `BEHIND`.
    - `days_gap`: Buffer days before exam (positive) or days late (negative).
    - `projected_finish`: Projected completion date formatted as `YYYY-MM-DD`.
    - `study_slots_json`: User's configured daily study slots.

### 3. Frontend Routine & Telemetry Components
- **`frontend/src/components/TelemetryWidget.vue`**:
  - Refactored into a quiet, clickable telemetry badge showing live pace status (e.g., *"On track · Finish Oct 24"* or *"Estimating pace..."*). Emits `@open-pace-modal`.
- **`frontend/src/components/StudyPaceModal.vue`**:
  - Created a quiet drawer popover providing:
    - Feasibility banner and metrics grid (Remaining sessions, Current velocity, Target pace, Projected finish with buffer).
    - Multi-window daily schedule editor (Morning, Evening, etc.) with presets: *Morning + Evening*, *College Split*, *Power Hour*.
    - Calendar sync actions (`.ics`, Google Calendar, Outlook Web).
- **`frontend/src/pages/Dashboard.vue`**:
  - Connected `TelemetryWidget` `@open-pace-modal` to open `StudyPaceModal`.
  - Added `handleSaveStudySlots` to update user settings in backend SQLite and trigger in-app scheduler sync.

### 4. Multi-Slot Calendar Export & In-App Reminders
- **`frontend/src/services/calendarService.js`**:
  - Enhanced `generateRoutineICS` and `downloadRoutineICS` to accept multiple slots and output RFC 5545 compliant `.ics` files with repeating `RRULE:FREQ=DAILY` blocks and audio alarms for each slot.
- **`frontend/src/App.vue`**:
  - Updated `getNextEventTimeout` and `syncScheduler` to parse `study_slots_json` and schedule chimes/banners for the next upcoming slot start/end across all active study windows.

---

## Verification Results

### 1. Backend Compilation and Unit Tests
```pwsh
go test -short ./internal/db/... ./internal/app/...
```
**Output**:
- `ai-tutor/internal/db`: `ok` (all database queries and migrations passed)
- `ai-tutor/internal/app`: `ok` (all endpoint handlers passed)

### 2. Frontend Production Build
```pwsh
npm run build
```
**Output**:
- Built in 23.99s with `dist/index.html` and assets cleanly emitted without errors.
