# Solution: Settings Modular Left-Rail Redesign

## Context & Motivation
The previous Settings interface grouped all application configuration panels into a monolithic two-tab vertical card stack. As extensions and additional granular study parameters were introduced, the page suffered from cognitive overload and poor visual scalability.

## Solution Architecture

### 1. Full-Width Master-Detail Sidebar Layout (Linear / VS Code Style)
Restructured `frontend/src/pages/Settings.vue` into a solid, full-height secondary rail and 100% full-width content area:
- **Solid Category Rail**: Fixed dark panel (`260px` width) directly flanking the main app sidebar, with header, active indicators, and cloud sync status.
- **Full-Width Workspace Viewport**: Eliminated artificial centering constraints (`100%` width), seamlessly expanding the configuration panels to fill the available desk space.
- **Categorized Panes**:
  - **Study & Routine**: Session budgets (max flashcards, word targets), daily study schedule range presets, calendar routine export, in-app chime tests, and quiz rescue remediation tracks.
  - **AI & Retrieval**: Fast & Heavy LLM provider selections, endpoints, API keys, and local RAG retrieval configuration.
  - **Profiles & Notebooks**: Study profile creation, active goal switching, deadline management, and notebook-to-profile bindings.
  - **System & Account**: Visual workspace themes, Clerk / School account status, and software update checks.

### 2. Typographic Polish & Clean UI
- Removed all unnecessary emojis across `Settings.vue`, `SettingsStudyBudget.vue`, `SettingsAccount.vue`, and `SettingsUpdate.vue` in favor of clean typography and consistent badge/pill components.
- Responsive breakpoints collapse the rail cleanly on mobile/small viewports.

### 3. Zero Backend Regressions
- Maintained exact bindings with `UserSettings`, `LLMTierSettings`, and all Vue composables (`useSettings`, `useLLM`, `useProfiles`, `useAuth`, `useRAG`).

## Modified Files
- `frontend/src/pages/Settings.vue`
- `frontend/src/components/SettingsStudyBudget.vue`
- `frontend/src/components/SettingsAccount.vue`
- `frontend/src/components/SettingsUpdate.vue`
