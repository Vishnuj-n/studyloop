## 🚀 Improvements
- **Reliable profile switching** – Changing your active profile now updates instantly with clear loading states and error handling, preventing accidental loss of settings.  
- **Profile‑scoped settings persistence** – Your study preferences (e.g., study prompts, notification settings) are saved per profile, so they follow you wherever you switch.  
- **Streamlined study prompts** – Prompts are now presented more consistently, making the learning flow smoother.  
- **More stable session handling** – Session data is persisted using a unified path, reducing login hiccups after page reloads or browser restarts.  
- **Prompt‑logging preference retained** – The option to log LLM prompts is now stored in SQLite, so your choice persists across app restarts.

## 🐛 Bug Fixes & Issue Resolutions
- Fixed race conditions that could overwrite profile‑specific settings during profile switches.  
- Resolved intermittent authentication/session persistence glitches.  
- Fixed the issue where prompt‑logging settings were lost after closing the app.

---
*Thanks for using Studyloop!*
