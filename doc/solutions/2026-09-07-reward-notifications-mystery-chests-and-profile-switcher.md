# Solution: Two-Tier Reward Notifications, Mystery Chests, and Accessible Profile Switcher

## Overview

This solution documents the completed implementation of the **Two-Tier Reward Notification System**, **Ambient Loot Vault Indicator**, **Streak Freeze Celebration Modal**, and **Accessible Profile Switcher Dropdown** across commits `2913b86`, `6524b74`, and `bd28209`.

### Behavioral Design Rationale
1. **Two-Tier Reward Hybrid Model (Effort Certainty + Variable Ratio Excitement)**:
   - **Tier 1 (Guaranteed Progression)**: Every completed study unit (Reading, Quiz, Flashcard Session, Milestone Exam, Socratic Remedial) guarantees baseline XP and Coins. Effort is always acknowledged, preventing resentment from pure randomized "zero-reward" mechanics.
   - **Tier 2 (Variable Surprise Drop)**: Completed tasks drop a tiered Mystery Chest (`BRONZE`, `SILVER`, `GOLD`, `MYTHIC`) with randomized payouts (XP jackpots, coins, streak freeze shields).
2. **Non-Intrusive Visibility vs Flow Preservation**:
   - Replaced aggressive full-screen modal takeovers with a non-blocking floating **`RewardToast`** containing `[Open Now]` and `[Save to Vault]` options.
   - Studiers in deep focus can keep studying without interruption; the chest is safely persisted to SQLite.
   - An ambient **`🎁 N`** indicator on the sidebar navigation keeps unopened rewards visible as an anticipation trigger for end-of-session celebrations.
3. **Loss-Aversion Celebration**:
   - Added **`StreakFreezeModal`** explaining passive insurance behavior when acquiring streak freeze shields.
4. **User Autonomy**:
   - Added `show_reward_notifications` setting in SQLite `user_settings` and settings UI to allow users to toggle reward popups.
5. **Polished Study Profile Switching**:
   - Modernized the profile switcher into an accessible, keyboard-friendly dropdown menu on the dashboard.

---

## Changes Made

### 1. Database & Settings
- **[internal/db/schema.go](../../internal/db/schema.go)** & **[doc/SCHEMA.md](../../doc/SCHEMA.md)**:
  - Added `show_reward_notifications BOOLEAN DEFAULT 1` column to `user_settings` singleton table.
- **[internal/models/models.go](../../internal/models/models.go)**:
  - Added `ShowRewardNotifications bool` to `UserSettings` struct.
- **[internal/db/store.go](../../internal/db/store.go)**:
  - Updated `GetUserSettings()` and `UpdateUserSettings()` queries to read/write `show_reward_notifications`.
- **[frontend/src/composables/useSettings.js](../../frontend/src/composables/useSettings.js)** & **[frontend/src/components/SettingsStudyBudget.vue](../../frontend/src/components/SettingsStudyBudget.vue)**:
  - Exposed `show_reward_notifications` reactive setting and added a toggle switch in the Study Budget & Preferences UI.

### 2. Backend Reward Routing
- **[internal/app/app_study.go](../../internal/app/app_study.go)**, **[internal/app/app_study_cards.go](../../internal/app/app_study_cards.go)**, **[internal/app/app_study_reading.go](../../internal/app/app_study_reading.go)**:
  - Ensured task completion endpoints (`CompleteReviewSession`, `FinishQuiz`, `AdvanceReadingTask`, `CompleteMilestoneExam`, `FinishSocraticSession`) return `RewardPayload` with earned base XP, coins, and any dropped `PendingLootBox`.

### 3. Frontend Reward Presentation & Audio
- **[frontend/src/components/RewardToast.vue](../../frontend/src/components/RewardToast.vue)**:
  - Floating, non-blocking toast rendering base XP & coin badges, chest tier tag, countdown progress bar, `Open Now`, and `Save to Vault` actions.
  - Automatically respects the user's `show_reward_notifications` preference.
- **[frontend/src/components/StreakFreezeModal.vue](../../frontend/src/components/StreakFreezeModal.vue)**:
  - Celebratory explanation modal displaying streak protection mechanics, passive consumption rules, and balance summary.
- **[frontend/src/components/Sidebar.vue](../../frontend/src/components/Sidebar.vue)**:
  - Tracks `pendingChestsCount` from `getGamificationState()`.
  - Added animated `🎁 N` badge with glowing pulse next to the Rewards link when unopened chests are in the vault.
- **[frontend/src/App.vue](../../frontend/src/App.vue)**:
  - Global listener for `reward-earned` and `streak-freeze-acquired` custom events, managing global `RewardToast`, `MysteryChestModal`, and `StreakFreezeModal` mount states.
- **Task Pages**:
  - **[frontend/src/pages/Flashcards.vue](../../frontend/src/pages/Flashcards.vue)**
  - **[frontend/src/pages/Quiz.vue](../../frontend/src/pages/Quiz.vue)**
  - **[frontend/src/pages/Reader.vue](../../frontend/src/pages/Reader.vue)**
  - **[frontend/src/pages/Socratic.vue](../../frontend/src/pages/Socratic.vue)**
  - **[frontend/src/pages/SocraticRescue.vue](../../frontend/src/pages/SocraticRescue.vue)**
  - Dispatched `window.dispatchEvent(new CustomEvent('reward-earned', { detail: res.rewards }))` on task completion.

### 4. Dashboard Profile Switcher Enhancement
- **[frontend/src/pages/Dashboard.vue](../../frontend/src/pages/Dashboard.vue)**:
  - Replaced basic selector with a styled dropdown menu featuring active indicator checkmarks, deadline badges, click-outside handling, and keyboard accessibility.

---

## Verification

1. **Internal Go Unit Tests**:
   - `go test -short ./internal/...` $\to$ **PASS** (all internal packages compile and pass without errors).
2. **Frontend Build Verification**:
   - `npm run build` (Vite) $\to$ **PASS** (zero compilation or template errors).
3. **Manual Flow Testing**:
   - Completed Flashcard review sessions, quizzes, and reader milestones; verified `RewardToast` appears non-intrusively.
   - Verified `[Save to Vault]` increments the sidebar `🎁 N` badge without blocking screen interaction.
   - Verified opening chests in `/rewards` vault or via `[Open Now]` awards random loot and plays Web Audio fanfare.
