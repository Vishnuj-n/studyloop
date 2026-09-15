# What's New – v1.8.0

## ✨ Features
- **Quiz Settings & Study Slots**  
  Users can now customize quiz behavior and allocate dedicated study slots directly from the settings page, giving finer control over their learning sessions.

## 🚀 Improvements
- **Reading Task Coordination**  
  The system now treats pending `FLASHCARD_GENERATE` tasks as blockers for new reading tasks. This prevents the creation of redundant reading jobs while flashcard generation is in progress, ensuring a cleaner task queue and more efficient resource usage.

- **Review Task Scheduling & Priority**  
  - Refactored the study‑plan generation logic to correctly aggregate existing flashcard review tasks, eliminating duplicate reviews.  
  - Updated SQL ordering to include `study_queue.priority`, delivering more accurate task prioritization.  
  - Fixed Dashboard filtering so the review hero component no longer shows duplicate entries.  
  - Enhanced the database inspection script to handle various file encodings reliably.

## 🐛 Bug Fixes
- Resolved an issue where duplicate review tasks could appear in the queue, leading to inflated card/minute counts.

## 🧹 Maintenance
- Version bump to **v1.8.0**.  
- Updated release notes for the previous **v1.7.0** release.

## 📦 Full Changelog
[View the full commit history on GitHub](#)

---

Thanks for using Studyloop!
