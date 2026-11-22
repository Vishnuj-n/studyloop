## ✨ Features
- **Custom Prompt Studio Variants** – You can now choose from multiple prompt‑studio variants when configuring notes, giving you more flexibility in how notes are generated and displayed.

## 🚀 Improvements
- **Modal Design Refresh** – Modal windows now follow the updated design tokens, with aligned borders and improved contrast for a cleaner, more accessible look.  
- **Calendar Reliability** – Calendar entries now generate unique IDs even in environments without `crypto.getRandomValues`, preventing duplicate or missing events.  
- **Notebook Management** – Deleting a notebook no longer removes shared topics, and deck overviews are deduplicated for clearer organization.  
- **Certificate Statistics** – Mature cards are now counted correctly, giving you more accurate progress metrics.  
- **Study Queue Stability** – The queue advances in a forward‑only manner, automatically prunes orphaned tasks, and surfaces similarity‑related errors, resulting in smoother study sessions.

## 🐛 Bug Fixes & Issue Resolutions
- Fixed a crash when restarting a quiz caused by undefined `notebookID` or `selectedTopicID`.  
- Resolved migration issues that affected note detail levels, FSRS card duplication, and rank title calculations.  
- Corrected cursor handling and task reconciliation in the study queue, eliminating stale or orphaned tasks.  
- Addressed security scan alerts to keep the platform safe and compliant.  

---  
*Thanks for using Studyloop!*
