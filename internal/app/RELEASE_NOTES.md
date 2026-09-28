## ✨ Features
- **Profile‑scoped Settings** – Settings can now be defined per study profile. When you create a new profile its settings are automatically cloned from the active one, and global‑scope badges make it clear which settings apply app‑wide.  
- **“Complete Here” Split Session** – A new **Complete Here** button lets you finish a reading task at any page, automatically generating a page‑bounded quiz and preserving queue continuity.  
- **Flashcard Deck Manager** – Manage all your flashcards in one place with a dedicated deck manager UI. View FSRS retention metrics, suspend or delete cards (individually or by notebook), and see deck statistics across notebooks and topics.  
- **Gamification Tweaks** – Streak‑freeze inventory now has a hard cap, a 7‑day usage limit, and a cost of 150 coins, making streak management clearer and more balanced.  

## 🚀 Improvements
- **Secure Session Handling** – Session verification now uses HMAC signatures and includes graceful grace‑period checks for smoother, safer logins.  
- **Profile Switcher UI** – Updated with design‑system tokens for a cleaner, more consistent look.  
- **Theme Stability** – Active theme is now persisted to both user settings and `localStorage`; the app will automatically roll back to the previous theme if persistence fails.  
- **Reading Flow Safeguards** – Reading tasks can only be marked complete when they are truly active, preventing accidental completions.  
- **Flashcard Overview Links** – Flashcard deck overview now correctly links to notebook and topic pages via a unified query.  

## 🐛 Bug Fixes & Issue Resolutions
- Fixed race conditions and clobbering when switching profiles; active profile ID is now persisted independently.  
- Resolved issues where global settings overwrote profile‑specific settings.  
- Ensured profile‑scoped settings are saved correctly and re‑loaded on profile switch.  
- “Complete Here” button now always appears in the completion dropdown and reliably enables when splitting is possible.  
- Flashcard deck manager now surfaces mutation errors and disables review hotkeys while the modal is open.  
- Prompt‑logging preference is now retained across app restarts.  
- Theme persistence errors no longer break the UI; a fallback theme is applied automatically.  
- Pace calculations now correctly prioritize profile‑specific word targets before falling back to user defaults.  
- Reading task completion now requires an ACTIVE status, protecting against premature completions.  
- Session persistence path unified to avoid inconsistencies.  

---  
*Thanks for using Studyloop!*
