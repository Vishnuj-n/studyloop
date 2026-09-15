# What's New – v1.7.0

## ✨ Features
- **Seamless in‑app auto‑updater**  
  Users can now download and install the latest version directly from the application. The updater streams the installer with real‑time progress, uses a detached PowerShell worker on Windows to avoid file‑lock issues, and falls back gracefully on other platforms. UI components in Settings and the startup modal show download status and let users trigger updates.

- **Anonymous usage telemetry**  
  A lightweight, fire‑and‑forget heartbeat now reports version, platform, and architecture information to help the team understand adoption patterns. All data is anonymized, stored with row‑level security, and automatically purged after 90 days.

- **Contributing guide & updated README**  
  New `CONTRIBUTING.md` and refreshed documentation sections make it easier for newcomers to get involved and understand the project’s workflow.

- **Privacy policy & transparency**  
  Added a comprehensive `PRIVACY.md` that explains the app’s local‑first design and telemetry practices. A one‑time privacy notice modal is shown to new users, and settings now use “Privacy & Data” terminology.

## 🚀 Improvements
- **Privacy modal redesign** – Updated styling with `color-mix` for smoother transparency effects and refined backdrop filters.
- **Onboarding flow revamp** – Switched to a deck‑based step system for smoother transitions and clearer navigation.
- **Settings UI polish** – Updated labels (e.g., “In‑App Session Alerts”) and clarified alert behavior hints.
- **Asset manager performance** – Added SHA‑256 checksum verification to make file copying idempotent, reducing unnecessary disk I/O.
- **Telemetry endpoint resolution** – Simplified logic in the heartbeat module for faster execution.
- **Startup backup resilience** – Backup mechanism now uses a timestamped ring buffer retaining the last three backups, protecting against data loss during repeated crashes.
- **Test suite enhancements** – Expanded frontend mocks for gamification and topic services; cleaned up backend test utilities.

## 🐛 Bug Fixes
*No critical bugs were reported in this release.*

## 🧹 Maintenance
- Clarified the `.gitignore` comment regarding Wails‑generated bindings.
- Minor code clean‑ups and refactoring across UI and backend components.

## 📦 Full Changelog
For a complete list of changes, see the repository’s commit history.

---

Thanks for using Studyloop!
