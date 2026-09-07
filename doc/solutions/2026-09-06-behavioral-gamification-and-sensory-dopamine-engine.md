# Solution: Behavioral Gamification & Sensory Dopamine Engine

## Overview
Added a neuroscience-backed behavioral gamification and sensory dopamine engine to StudyLoop following the **Ponytail** minimal architecture principle:
1. **Deterministic Backend Authority**: SQLite persists XP, coins, narrative rank titles, and unopened mystery chests. No background daemons, auto-schedulers, or hidden event loops.
2. **Zero External Assets**: All audio feedback (chimes, clicks, thuds, fanfare) is procedurally generated at runtime via the browser's native Web Audio API (`AudioContext`). Zero `.mp3` or `.wav` files and zero external 3D/canvas libraries.
3. **Loss-Aversion & Zeigarnik Mechanics**: Prevents user study churn through purchaseable Streak Freeze shields (50 coins) and dynamic open-loop tension cards on the dashboard.

---

## Changes Made

### 1. Database & Schema
- **[internal/db/schema.go](../../internal/db/schema.go)**:
  - Added `user_gamification` singleton table (`user_id = 1`) tracking `total_xp`, `coins`, `current_title`, and `streak_freezes_owned`.
  - Added `pending_loot_boxes` table with `idx_pending_loot_boxes_opened` for unclaimed mystery chests.
  - Auto-seeded row `1` with title `'The Apprentice'` and `1` starting streak freeze.
- **[internal/db/gamification_repo.go](../../internal/db/gamification_repo.go)**:
  - `GetGamificationProfile()`: Computes dynamic rank progress boundaries.
  - `AddXPAndCoins(xp, coins)`: Atomically adds XP/coins and detects title milestone level-ups.
  - `CreatePendingLootBox()`, `GetUnopenedLootBoxes()`, `ClaimLootBox(id)`: Idempotent loot box claims.
  - `BuyStreakFreeze(cost)`: Deducts 50 coins to purchase a streak freeze shield.
  - `GetZeigarnikOpenLoops(activeProfileID)`: Evaluates notebooks and topics between 35% and 99% progress to build completion tension.
- **[doc/SCHEMA.md](../../doc/SCHEMA.md)**: Updated database schema Single Source of Truth.

### 2. Backend Rewards & Queue Transition Integration
- **[internal/study/rewards.go](../../internal/study/rewards.go)**:
  - Definitive drop tables for reading sessions, quizzes, flashcards, milestone exams, and socratic rescues.
  - Variable-tier chests: `BRONZE`, `SILVER`, `GOLD`, `MYTHIC` with probabilistic bonus XP, coins, and rare Streak Freezes.
- **[internal/study/queue_transition.go](../../internal/study/queue_transition.go)**:
  - Added `awardCompletionRewards(...)` and attached `Rewards: *models.RewardPayload` to `TransitionResult` on all task completions.
- **[internal/app/app_study.go](../../internal/app/app_study.go)**:
  - Exposed Wails bridge endpoints: `GetGamificationState()`, `ClaimLootBox(id)`, `BuyStreakFreeze()`, `GetZeigarnikOpenLoops()`.

### 3. Frontend & Procedural Audio
- **[frontend/src/utils/audioJuice.js](../../frontend/src/utils/audioJuice.js)**:
  - Procedural sound synthesizer via native Web Audio API:
    - `playCorrectChime()`: Ascending major triad (`C5 -> E5 -> G5`).
    - `playIncorrectThud()`: Low damped sine thud (`130Hz -> 50Hz`).
    - `playCardRatingTick(rating)`: Frequency-scaled clicks for flashcard ratings.
    - `playChestRattle()`: Rapid square-wave wooden clicks.
    - `playChestOpenFanfare()`: Arpeggiated triumphant fanfare (`C5 -> E5 -> G5 -> C6`).
- **[frontend/src/components/MysteryChestModal.vue](../../frontend/src/components/MysteryChestModal.vue)**:
  - Tactile bounce & rattle reward modal with lid pop animation and fanfare trigger.
- **[frontend/src/components/ZeigarnikOpenLoopCard.vue](../../frontend/src/components/ZeigarnikOpenLoopCard.vue)**:
  - SVG progress ring displaying completion percentage and quick "Resume →" action.
- **[frontend/src/pages/Rewards.vue](../../frontend/src/pages/Rewards.vue)**:
  - Dedicated `/rewards` view with rank XP progress bar, coin balance, streak freeze shop, and unopened chest vault.
- **[frontend/src/components/Sidebar.vue](../../frontend/src/components/Sidebar.vue)**:
  - Added Rewards navigation link and compact rank title + mini XP progress pill.
- **[frontend/src/components/StreakCalendar.vue](../../frontend/src/components/StreakCalendar.vue)**:
  - Added loss-aversion evening warning pulse (past 20:00 without study) and freeze shield visual indicator.
- **[frontend/src/pages/Flashcards.vue](../../frontend/src/pages/Flashcards.vue)** & **[frontend/src/pages/Quiz.vue](../../frontend/src/pages/Quiz.vue)**:
  - Wired audio feedback and mounted `MysteryChestModal` on session completion.

---

## Verification
- `go test -v ./internal/db -run TestGamificationRepo` $\to$ **PASS**
- `go test -v ./internal/study -run TestRollTaskRewards` $\to$ **PASS**
- `go test -short ./internal/...` $\to$ **PASS** (all internal Go packages compile and pass)
- `npm run build` (Vite) $\to$ **PASS** (zero compilation or import errors)
