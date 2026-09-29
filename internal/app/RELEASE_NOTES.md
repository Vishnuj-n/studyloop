## ✨ Features
- **Profile‑scoped Settings** – Each study profile now has its own word budget, queue rules, quiz settings, and theme. Switching profiles preserves its unique settings, with clear global‑scope badges. New profiles automatically clone the settings of the currently active profile.  
- **Flashcard Deck Manager** – A brand‑new UI for viewing and managing flashcards across notebooks and topics. See FSRS retention metrics, suspend or bulk‑suspend cards, delete cards, and get an overview of deck statistics.  
- **Dynamic Reading Session Splitting** – The “Complete Here” button lets you finish a reading task at any page while keeping quizzes and the queue intact.  
- **Gamification Enhancements** – Streak‑freeze inventory now has a cap, a 7‑day usage rate limit, and a cost of 150 coins.  
- **Developer Tool – Revert Reading Sessions** – A “Revert” action in the Reading Logs panel lets developers reset completed reading tasks back to an active state for testing or correction.

## 🚀 Improvements
- Settings UI now shows loading states and better error handling when switching profiles.  
- Theme persistence is more reliable: the active theme is saved to user settings and `localStorage`, with automatic rollback on persistence errors.  
- Profile‑switcher dropdown styled using design‑system tokens for a cleaner look.  
- Global extensions badges are now normalized across the application.  
- Session management hardened with HMAC verification and grace‑period checks.  
- Prompt‑logging setting for the LLM now persists across app restarts.  
- Flashcard deck overview links are unified for consistent navigation.  
- “Complete Here” option is always displayed in the reader, even for multi‑page tasks.

## 🐛 Bug Fixes & Issue Resolutions
- Fixed race conditions that could overwrite active‑profile settings when updating global settings.  
- Prevented accidental clobbering of profile‑specific settings during profile switches.  
- Added a dedicated mutation to persist the active profile ID independently, eliminating race‑related bugs.  
- Enforced ACTIVE task status to protect reading completion logic.  
- Resolved flashcard deck manager mutation errors and blocked review hotkeys while the modal is open.  
- Corrected pace calculations to prioritize profile‑specific target session words before falling back to user defaults.  
- Fixed theme rollback on persistence errors and standardized badge display.  
- Unified session persistence across authentication flows.  
- Persisted LLM prompt‑logging settings in SQLite across restarts.  
- Unified flashcard deck overview notebook‑topic links via a union query.  
- Various minor UI and backend stability fixes.

---  
*Thanks for using Studyloop!*
