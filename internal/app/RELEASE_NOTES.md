# What's New in **v1.12.0**

## ✨ Features

- **Secure Session Management** – Added HMAC verification and grace‑period checks to authentication sessions, making user log‑ins more robust against tampering.  
- **Flashcard Deck Manager** – Introduced a full‑screen deck management UI with FSRS‑based retention metrics (New, Learning, Young, Mature). Users can now view deck statistics, toggle card suspension, bulk‑suspend notebooks, and delete cards. Backend APIs and database queries were added to support these operations.  
- **Dynamic Reading Session Splitting** – New “Complete Here” button lets readers finish a task at any page, automatically creating a split session with a page‑bounded quiz and preserving queue continuity. The option is always available in the completion dropdown and works reliably across multi‑page tasks.  
- **Profile‑Scoped Settings** – Settings can now be scoped to individual study profiles. When creating a new profile, settings are cloned from the active one, and global‑scope badges clearly indicate which settings apply app‑wide.  
- **Gamification Enhancements** – Streak‑freeze items now respect an inventory cap, a 7‑day usage limit, and a cost of 150 coins, preventing abuse while keeping the reward system engaging.  

## 🚀 Improvements

- Cleaned up the SVG icon registry by removing unused globe and chevron icons.  
- Refactored the Dashboard component to eliminate dead imports.  
- Streamlined the Notebook Syllabus modal by deleting obsolete style rules.  
- Consolidated test setup boilerplate with shared seed helpers, reducing duplication.  
- Updated the Profile Switcher UI to use design‑system tokens for a consistent look and feel.  
- Added documentation on the architecture of dynamic reading splits and profile‑scoped settings.  

## 🐛 Bug Fixes

- **Theme Persistence** – Implemented a rollback mechanism for theme changes that fail to persist, and ensured the active theme is saved to both user settings and `localStorage`.  
- **Flashcard Errors** – surfaced mutation errors in the deck manager and disabled review hotkeys while the modal is open to prevent accidental actions.  
- **Reading Completion** – Guarded task completion by requiring the task to be in an ACTIVE state, preventing premature finishes.  
- **Settings Isolation** – Prevented global settings updates from unintentionally overwriting active profile settings.  
- **Pace Calculations** – Now checks a profile’s target session word count before falling back to user defaults.  
- **Database Queries** – Unified flashcard deck overview links across notebooks and topics using a UNION query for accurate navigation.  
- **Rewards Theme Sync** – Fixed persistence of the active theme within the rewards system.  
- **Complete Here Button** – Ensured the button always appears in the completion dropdown and that the `canSplitHere` logic reliably enables it during multi‑page tasks.  

## 🧹 Maintenance

- Version bump to **v1.12.0**.  
- Refactored test scaffolding and removed unused code across several components.  
- Updated `RELEASE_NOTES` and `SCHEMA.md` to reflect new features and database changes.  

## 📦 Full Changelog

[View all changes between v1.11.1 and v1.12.0](https://github.com/your-repo/Studyloop/compare/v1.11.1...v1.12.0)

---

Thanks for using Studyloop!
