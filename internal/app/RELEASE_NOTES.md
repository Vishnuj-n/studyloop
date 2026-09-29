## ✨ Features
- **Flashcards** – A refreshed card library overview makes it easier to browse and manage your decks. Deleting FSRS‑based cards now includes an extra safety guard to prevent accidental loss.  
- **Rewards Hub** – Completely redesigned with a clear roadmap, new achievement badges, and celebratory milestones to keep you motivated.  
- **PDF Reader** – Clickable internal links are now supported, and a history back button lets you navigate PDF documents just like a web browser.

## 🚀 Improvements
- **User Interface polish**
  - Added a subtle outline border to the flashcard session overlay for better visual separation.
  - Cleaned up redundant `.currency-icon` CSS and resolved design lint violations, bringing the UI fully in line with the latest `DESIGN.md` guidelines.  
- **Tokenizer safety** – Diagnostic prompt budgeting and built‑in safeguards help keep LLM interactions within token limits, reducing unexpected truncation.  

## 🐛 Bug Fixes & Issue Resolutions
- **Authentication** – The app now waits for the backend bridge to be ready before syncing Clerk sessions, preventing login hiccups.  
- **Settings** – Auto‑save is now guarded against premature triggers during the initial load, ensuring your preferences are stored correctly.  
- **Reading flow** – Force‑seeding of reading tasks works correctly when you continue after a quiz, eliminating stalls.  

---  
*Thanks for using Studyloop!*
