# Solution: Profile-Scoped Settings and Minimalist Global Scoping Architecture

## Context & Motivation
Previously, study parameters (target session words, quiz question count, passing thresholds, remedial strategies, flashcard review limits, active notebook concurrency, and visual themes) were stored solely as global configuration in `user_settings`.

However, distinct study subjects require vastly different study rhythms:
- Reading dense, formula-heavy textbooks (Physics, Law) requires smaller word budgets (~1,200 words) and higher quiz thresholds compared to survey reading (History, Literature).
- Students preparing for different exams (e.g., UPSC vs. Medical Boards) need isolated review volumes, focus intervals, and themes without having their settings globally collide.

## Solution Architecture

### 1. The Zero-Bloat "Inherit by Default" Overlay Pattern
Rather than creating new abstraction layers, complex multi-tenant databases, or event buses, we leveraged the existing SQLite overlay architecture:
- **`study_profiles` Table Extension**: Added 10 nullable columns (`target_session_words`, `min_session_words`, `theme`, `max_flashcards_per_session`, `max_active_notebooks`, `skip_to_reading_active`, `default_remedial_strategy`, `quiz_question_count`, `quiz_passing_score`, `tutor_style`).
- **Dynamic Overlay in `GetUserSettings()`**: When `active_profile_id != ""`, the backend reads the active profile row in a single `<0.1ms` indexed query and overlays non-null/non-empty values on top of the base settings struct.
- **Zero Downstream Refactoring**: Queue schedulers, quiz generators, and FSRS engines continue calling `repo.GetUserSettings()`. They seamlessly receive the effective active configuration without knowing or caring whether it came from the profile row or global table.

### 2. Auto-Cloning on Profile Creation
- In `internal/app/app_settings.go`, `CreateProfile` automatically copies all study parameters from the currently active profile.
- Users only need to enter a Name and Deadline; all preferred study rhythms, quiz rules, and themes carry over with zero friction.

### 3. Minimalist Frontend UI Scoping (Global Only)
- Since the vast majority of settings are now profile-scoped, adding badges to every panel created visual clutter.
- **Only Global panels** are marked with a standardized SVG globe badge: `<span class="global-badge"><BaseIcon name="globe" size="12" /> Global</span>`.
- Marked sections:
  1. AI Providers & Keyring API Keys
  2. Daily Real-World Study Schedule Windows & Calendar Sync
  3. Extensions & Developer Mode
- All other panels (Theme, Reading Word Budget, Quiz Parameters, Flashcard Review Limits, Remediation Style) naturally belong to the active profile.
- Switching active profiles triggers an instant reload of effective settings, updating the UI theme and parameters without a page reload.

## Modified Files
- `internal/db/schema.go`
- `internal/db/migrations.go`
- `internal/models/models.go`
- `internal/db/settings_repo.go`
- `internal/app/app_settings.go`
- `internal/app/quiz_flashcard_test.go`
- `frontend/src/assets/icons/index.js`
- `frontend/src/style.css`
- `frontend/src/components/SettingsAIProvider.vue`
- `frontend/src/components/SettingsDeveloperPanel.vue`
- `frontend/src/components/SettingsExtensions.vue`
- `frontend/src/components/SettingsProfileModal.vue`
- `frontend/src/components/TimeRangeInput.vue`
- `frontend/src/pages/Settings.vue`
- `doc/SCHEMA.md`
