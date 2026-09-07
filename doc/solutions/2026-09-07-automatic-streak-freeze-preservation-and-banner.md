# Solution: Automatic Streak Freeze Preservation & Dashboard Notice

## Overview
Added automatic streak freeze consumption when a user misses a study day, along with an informative notification banner on the dashboard.

1. **Passive Safety Net**: Streak Freezes are passive insurance shields. If the user misses a day, SQLite automatically consumes 1 freeze, bridges yesterday into the active date set, and preserves their streak count without manual intervention.
2. **Ponytail Banner UI**: Instead of building a complex, floating multi-page toast system, the feature reuses the existing `<StatusBanner>` component directly at the top of `Dashboard.vue`.

---

## Changes Made

### 1. Database & Repository
- **[internal/db/gamification_repo.go](../../internal/db/gamification_repo.go)**:
  - Added `ConsumeStreakFreeze()`: Atomically decrements `streak_freezes_owned` by 1 where `streak_freezes_owned > 0`.

### 2. Backend Streak Engine
- **[internal/app/app_study.go](../../internal/app/app_study.go)**:
  - Updated `getStreakState()`: If today is not completed, yesterday was missed, but the user had an active streak and has at least 1 freeze available:
    - Calls `repo.ConsumeStreakFreeze()`.
    - Bridges yesterday's date into the active set and recalculates the streak.
    - Emits `streak_saved_event` (`streak_length`, `freezes_remaining`) in the IPC payload.
    - Sets `shield_active: true` when a user has a freeze in reserve and has not completed today's study.

### 3. Frontend Notification Banner
- **[frontend/src/pages/Dashboard.vue](../../frontend/src/pages/Dashboard.vue)**:
  - Reuses `<StatusBanner>` to render the notification:
    - **Icon**: `🛡️`
    - **Title**: `Your streak was saved!`
    - **Subtitle**: `We used 1 Streak Freeze to protect your ${streakSavedEvent.streak_length}-day streak yesterday. You have ${streakSavedEvent.freezes_remaining} freeze(s) remaining.`
    - **Action**: Dismiss button.
- **[frontend/src/components/StreakFreezeModal.vue](../../frontend/src/components/StreakFreezeModal.vue)**:
  - Celebratory feedback modal displayed when purchasing a streak freeze on `/rewards`.

---

## Verification
- `go test -v ./internal/app -run TestGetStreakState` -> **PASS**
- `go test -short ./internal/...` -> **PASS**
- `npm run build` (Vite) -> **PASS**
