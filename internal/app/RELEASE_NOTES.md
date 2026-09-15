# What's New – v1.6.0

## ✨ Features
- **Continuous Reading & Deferred Quiz**  
  - Readers can now finish a session without being forced into an immediate quiz. Selecting **“Complete & Defer Quiz”** queues the generated quiz as a pending task, keeping the reading flow uninterrupted.  
  - Backend now auto‑seeds and fetches the next reading task for a notebook, preserving momentum across sessions.

- **Expanded Study‑Queue Task Types**  
  - The queue now recognises `QUIZ`, `MILESTONE_EXAM`, and `SOCRATIC_REMEDIAL` tasks when checking for pending reading work, preventing duplicate task creation.  
  - Added `scripts/inspect_db.py` for quick database inspection.

- **Developer Mode & Diagnostics**  
  - New panel in Settings provides:  
    - Real‑time LLM prompt logging toggle.  
    - Quick shortcuts to open the app’s data and logs folders.  
    - Reading Task History diagnostics and a modular DB split option.

- **LLM Provider Key Synchronisation & Validation**  
  - Heavy‑tier API key automatically mirrors the fast‑tier key when the providers match or when “use same for heavy” is enabled.  
  - Basic validation now warns about mismatched key prefixes (e.g., Groq vs. Gemini).

- **Reading Bounds by Word Budget**  
  - Reading sessions respect the user‑defined `TargetSessionWords` setting.  
  - Word‑count‑based calculations cap session length with a 30 % buffer, ensuring sessions stay within the desired size.

- **Reader UI Enhancements**  
  - Added a split‑button for **“Complete”** vs. **“Complete & Defer Quiz”**.  
  - Updated default PDF zoom to 100 % for a more natural view.

## 🚀 Improvements
- **Session Completion Flow** – Streamlined UI consistency across the reader and rewards shop.  
- **Chunk Payload Limits** – Enforced word‑count caps on chunk responses, reducing data transfer and aligning with user session preferences.  
- **Dashboard Telemetry** – Daily reading session stats now include completed `READING` tasks for better insight.

## 🐛 Bug Fixes
- Fixed missing `gamification-updated` event after item purchases, allowing UI components to react correctly.  
- Preserved sub‑session reading bounds and enforced the target session word‑budget ceiling.  
- Corrected PDF default zoom from 70 % to 100 %.  
- Adjusted release‑notes modal layout to prevent scrollbar shifts.  
- Updated UI to dispatch proper events and maintain layout stability.

## 🧹 Maintenance
- Version bump to **v1.6.0**.  
- Updated release notes for the previous v1.5.0 release.  

## 📦 Full Changelog
[View all changes on GitHub](#)

---

Thanks for using Studyloop!
