# What's New – v1.11.0

## ✨ Features
- **Database Backup & Recovery**  
  Introduced a robust backup system that creates atomic SQLite snapshots using `VACUUM INTO`, compresses them with gzip, and stores them safely. Users can now manually restore the latest backup directly from the **Settings → Database & Recovery** page, with clear status feedback and confirmation dialogs.

- **Enhanced Upgrade Button Tooltip**  
  The “Upgrade” deep‑button now displays a richer tooltip that explains the upgrade process in detail. A fallback message is also shown in the syllabus modal when upgrade information cannot be loaded.

## 🚀 Improvements
- **Flashcard Prompt Quality**  
  Refined the wording and structure of flashcard prompts for a clearer learning experience. Added documentation on backup retention policies to help users understand how long backups are kept.

- **Dashboard Refactor**  
  Extracted `DashboardBanners` and `ProfileSwitcher` into their own components, simplifying the main Dashboard code and improving maintainability.

- **General UI Polish**  
  Minor visual tweaks and accessibility enhancements across the application.

## 🐛 Bug Fixes
*No bug fixes in this release.*

## 🧹 Maintenance
- Bumped project version to **v1.11.0**.
- Updated release‑note templates and documentation for the previous release.

## 📦 Full Changelog
For a complete list of changes, see the commit history between the previous tag and **v1.11.0**.

---

Thanks for using Studyloop!
