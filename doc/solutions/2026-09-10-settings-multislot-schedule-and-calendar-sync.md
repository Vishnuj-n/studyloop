# Solution: Multi-Schedule Routine in Settings & Dedicated Calendar Sync

## Overview
Consolidated daily study schedule configuration and calendar synchronization into **Settings > Study & Routine**, allowing users to add, name, configure, and delete multiple study windows easily. Cleaned up redundant calendar export buttons from the dashboard pace drawer, directing routine and notification management to Settings.

---

## Changes Made

### 1. Settings Study Routine Multi-Schedule (rontend/src/components/TimeRangeInput.vue)
- **Add & Remove Schedule Slots**: Added an + Add Schedule button allowing multiple daily study sessions (e.g., Morning Session, Evening Deep Work).
- **Per-Slot Customization**:
  - Customizable session names with #1, #2 index badges.
  - Start and end time inputs with SVG clock icons.
  - Per-slot duration chips and quick presets (30 min, 1 hour, 1.5 hours, 2 hours, 3 hours).
  - Delete button for each slot (safeguards the last remaining slot from being deleted).
- **Cumulative Duration**: Automatically calculates and displays the total scheduled study time in the header badge.
- **Bi-directional Sync**: Directly synchronizes with study_slots_json as well as legacy study_start_time / study_end_time for full backward compatibility.

### 2. Multi-Slot Calendar & ICS Sync (rontend/src/components/SettingsStudyBudget.vue)
- **Download All Sessions (.ics)**: Single-click download generates an RFC 5545 compliant .ics calendar file with individual BEGIN:VEVENT blocks for every configured study window, daily recurrence (RRULE:FREQ=DAILY), unique UIDs, and 10-minute warning alarms.
- **Per-Slot Web Calendar Links**: Displays individual + Google Calendar and + Outlook Web buttons for each configured study window, allowing 1-click web calendar event creation per session.

### 3. Cleaned Dashboard Pace Drawer (rontend/src/components/StudyPaceModal.vue)
- Removed redundant calendar export buttons (Download .ics, Google Calendar, Outlook Web) from the Dashboard's study pace drawer.
- Provided a clear navigation button: **"⚙ Manage Schedule & Calendar Sync in Settings"**, guiding users to the canonical configuration page.

### 4. Robust RFC 5545 Calendar Generator (rontend/src/services/calendarService.js)
- Updated generateRoutineICS() to accept arrays of slots, JSON strings, or fallback strings.
- Added overnight schedule rollover support (where end time is earlier than start time, rolling DTEND to next day).
- Added unique randomized UIDs to guarantee calendar apps do not merge separate study windows into one.

---

## Verification
- **Automated Tests**:
  - go test -short ./internal/... passed with 0 errors.
  - 
pm test passed 47/47 tests across all 13 test suites.
  - Added dedicated test cases in src/services/calendarService.spec.js for multi-slot .ics generation, JSON string inputs, and overnight schedules.
