# Exam Feasibility & Capacity Routine Pace Architecture

**Date**: 2026-10-22 (Updated 2026-10-24)  
**Status**: Implemented  
**Scope**: Full Stack (`internal/app/notebook_endpoints.go`, `internal/app/app_settings.go`, `frontend/src/components/StudyPaceModal.vue`, `frontend/src/pages/Dashboard.vue`, `frontend/src/components/TelemetryWidget.vue`, `frontend/src/components/ProfileSwitcher.vue`, `frontend/src/pages/ProfileSelect.vue`)

---

## 1. Problem Statement
The previous pacing forecast suffered from several usability and conceptual flaws:
1. **False Precision & Fractional Daily Guidance**: Computed fractional quotas (`1.6 extra sessions/day`, `15.4 days late`) that broke down when users had scheduled slots on specific days or varied routines.
2. **Conflated Schedule Capacity with Historical Velocity**: It did not differentiate between whether a user had enough scheduled windows before the target date versus their actual completion rhythm.
3. **Absence of Actionable Recovery**: When a student fell behind or when target dates passed, the modal lectured them with impossible quotas instead of offering user-directed choices.
4. **Frictionless/Broken Recovery UX**: 
   - Switching to "Self-Paced" previously attempted to send an empty deadline string `""` to the backend, which failed with a parse error because the backend strictly demanded a date string.
   - The "Manage Schedule" button routed into generic routine settings (`/settings?category=study`) rather than profile and notebook assignment management (`/settings?category=profiles`).
   - The inline date-picker ("Extend Exam Target") allowed frictionless date-slipping without cognitive accountability.

---

## 2. Architecture & Design Principles

### Invariants Respected
1. **Zero Queue Tampering**: Pacing is an informational, read-only calculation. It never mutates `study_queue` tasks or auto-schedules sessions behind the scenes.
2. **Session as First-Class Unit**: Workload is measured in whole sessions ($\lceil \text{remaining words} / \text{target words} \rceil$), avoiding noisy word-per-minute or minute estimations.
3. **Scheduled Windows as Opportunities**: Daily slots in `settings.StudySlotsJSON` represent scheduled opportunities to study, not guaranteed 1:1 completions.
4. **Separation of Capacity vs Historical Rhythm**:
   - **Capacity**: Does the student have enough scheduled opportunities before the exam?
   - **Historical Rhythm**: Given recent weekly output (past 7 days), when will they realistically finish?
5. **Clear Self-Paced Semantics (`deadline_at = 0`)**: Profiles with empty deadlines or zero timestamps explicitly denote self-paced studying without deadline pressure.

---

## 3. Implementation Details

### Backend (`internal/app/notebook_endpoints.go` & `internal/app/app_settings.go`)
1. **Remaining Windows**:
   - Parses `settings.StudySlotsJSON` to determine active `slotsPerDay` (defaulting to 1 if empty).
   - $\text{Remaining Windows} = \text{Slots Per Day} \times \text{Days Remaining}$.
2. **Integer Workload & Deficit**:
   - $\text{Remaining Sessions} = \lceil \text{Remaining Words} / \text{Target Words} \rceil$.
   - $\text{Uncovered Sessions} = \max(0, \text{Remaining Sessions} - \text{Remaining Windows})$.
3. **Weekly Run-Rate**:
   - Uses completed reading sessions from the past 7 days (`sessionsPast7Days`) as weekly rhythm.
   - $\text{Weeks to Finish} = \text{Remaining Sessions} / \text{sessionsPast7Days}$.
   - $\text{Projected Finish Date} = \text{Today} + \lceil \text{Weeks to Finish} \times 7 \text{ days} \rceil$.
4. **Feasibility States**:
   - `DEADLINE_PASSED`: When $\text{Days Remaining} \le 0$ and $\text{Remaining Sessions} > 0$.
   - `UNCOVERED_WORKLOAD`: When $\text{Remaining Sessions} > \text{Remaining Windows}$.
   - `AT_RISK`: When workload fits in scheduled windows, but $\text{sessionsPast7Days} < \text{Required Weekly Pace}$.
   - `ON_TRACK`: Workload fits in scheduled opportunities and matches historical weekly rhythm.
   - `NO_DATA`: Fewer than 2 completed reading sessions.
5. **Self-Paced Backend Support**:
   - `UpdateProfile` and `UpdateProfileWithInheritance` now accept `""` or `"0"` for `deadlineStr`, persisting `current.DeadlineAt = 0` instead of failing with a parse error.

### Frontend (`frontend/src/components/StudyPaceModal.vue`)
Structured drawer layout divided into 3 focused sections:
1. **Section A (Capacity Status & Routine Balance)**:
   - Displays Remaining Workload (`X sessions`) and Scheduled Windows (`Y study windows`).
   - Routine Balance card clearly stating plain-language deficit (e.g. `~9 sessions uncovered by current routine` or `Target date has passed with 16 sessions remaining`, or neutral indicator when in self-paced mode).
2. **Section B (Historical Pace Reality Check)**:
   - Displays Recent Rhythm (`~X sessions / week`) and Projected Finish (`~14 Nov 2026`).
   - Disclaims that dates are estimated from recent rhythm rather than guaranteed completion.
3. **Section C (Actionable Recovery Options)**:
   - **Manage Profiles & Notebooks**: Directly routes to `/settings?category=profiles` where target dates, notebook assignments, and profiles are properly configured.
   - **Self-Paced Mode Switch Toggle**: 
     - A clean on/off switch toggle replacing loose buttons and inline date inputs.
     - Toggling ON prompts confirmation via global `ConfirmModal` (`useDialog()`), caches previous deadline in `localStorage`, and updates the backend profile with `""` (`deadline_at = 0`).
     - Toggling OFF allows 1-click reverting/restoring of the previous target deadline with confirmation.

### Supporting UI Updates
- `TelemetryWidget.vue`: Updated pill text and status styling to handle `UNCOVERED_WORKLOAD`, `AT_RISK`, `DEADLINE_PASSED`, and `ON_TRACK`, with fallback display when self-paced (`has_deadline = false`).
- `ProfileSwitcher.vue` & `ProfileSelect.vue`: Updated pace badges for the new feasibility states.
- `Dashboard.vue`: Bound `profile-id` and `@pace-updated` to reload active profile pacing when changes occur.
