# Solution Document — Fast Track Default Remediation Strategy

We have made the **Fast Track** (direct Socratic AI tutor on first failure) the default Quiz Failure Rescue strategy instead of the **Classic Track**. We also added a selection interface to Step 1 of the onboarding wizard so new users can customize this setting immediately.

## Proposed Changes

### Database & Settings
- **[`internal/db/schema.go`](../internal/db/schema.go)**: Changed default column value of `default_remedial_strategy` to `'FAST'` in the `user_settings` table definition and migration statements.
- **[`internal/db/store.go`](../internal/db/store.go)**: Changed default fallback strategy to `'FAST'` in database getters when no settings record exists.

### Tests
- **[`internal/app/remedial_strategy_test.go`](../internal/app/remedial_strategy_test.go)**: Updated `TestDefaultIsClassic` to `TestDefaultIsFast` and verified that the default strategy directly routes to Socratic remedial tasks.
- **[`internal/app/quiz_flashcard_test.go`](../internal/app/quiz_flashcard_test.go)**: Added `newClassicTestApp` test helper to easily setup and isolate tests that specifically verify traditional reread behavior without repeating SQL updates.

### Frontend UI
- **[`frontend/src/composables/useSettings.js`](../frontend/src/composables/useSettings.js)**: Configured initial reactive settings and fallback defaults for `default_remedial_strategy` to `'FAST'`.
- **[`frontend/src/pages/Dashboard.vue`](../frontend/src/pages/Dashboard.vue)**: Updated default settings state to `'FAST'`.
- **[`frontend/src/pages/Dashboard.spec.js`](../frontend/src/pages/Dashboard.spec.js)**: Updated mock settings default value to `'FAST'`.
- **[`frontend/src/pages/Onboarding.vue`](../frontend/src/pages/Onboarding.vue)**: Added a "Quiz Failure Rescue" strategy selector in Step 1 of the onboarding flow, default-initializing its ref to `'FAST'` and passing the chosen option in the initial settings save payload.
