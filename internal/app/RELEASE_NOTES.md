# What's New – v1.9.0

## ✨ Features
- **Robust Clipboard Export** – A new clipboard utility now falls back to `execCommand` when the modern `navigator.clipboard` API is unavailable (e.g., inside iframes).  
- **YouTube Session Exports** – Exported sessions include human‑readable timecodes (`mm:ss` / `hh:mm:ss`) instead of raw page numbers.  
- **Improved Notebook Title Resolution** – The reader now correctly picks up titles supplied by bundles, fixing edge‑case naming issues.  
- **Timezone‑Aware Scheduling** – Review tasks and due‑card counts respect the user’s local day‑boundary rather than UTC, with a new `clientEndOfDayUnix` helper and optional timezone offsets throughout the scheduling API.  
- **Settings Repository & Vector Storage** – Added a dedicated settings store (active profile, RAG status, etc.) and expanded vector handling with batch embedding updates and a new `vec0`‑based search table.  
- **Cross‑Profile Dashboard Badges** – The profile switcher now shows pacing badges and task counts for each profile, giving a quick overview of progress across multiple study profiles.  
- **AI‑Powered Quiz Diagnostics** – New diagnostic view surfaces why AI‑generated quiz questions failed, helping users understand and improve content quality.  
- **Quiz Performance Analysis UI** – Detailed breakdown of quiz results by topic clusters, score percentages, and failed questions, plus a “Detailed Performance Breakdown” button to navigate to the analysis page.  
- **Audio Overview Range Selector** – Reader now offers a dropdown to select audio playback ranges (full session, current page to end, or custom range).  

## 🚀 Improvements
- **Page Navigation Logic** – Refined `effectiveMinPage` / `effectiveMaxPage` handling for smoother navigation and removed redundant calculations.  
- **Reader Base Hook** – Enhanced `useReaderBase` to better resolve titles from bundled content.  
- **Backend Topic Bundle Handling** – Fixed `GetReaderTopicBundle` to correctly process raw content for YouTube and other file types.  
- **Quiz Cluster & Analysis State** – Adjusted front‑end handling of quiz cluster ranges and analysis state for more reliable results.  

## 🐛 Bug Fixes
- Fixed backend review‑related queries, diagnostics, vector handling, and telemetry issues.  
- Resolved frontend quiz cluster range calculations and pacing errors.  
- Enabled free seeking and a one‑time soft pause in the YouTube player for smoother playback control.  
- Prevented telemetry data from being sent to production endpoints during test runs.  

## 🧹 Maintenance
- Added database migrations for the new settings repository and study profile schema.  
- Refactored profile‑scoped query helpers into `study_queue_queries.go`.  
- Cleaned up unused imports, improved error handling, and updated integration tests across the codebase.  

## 📦 Full Changelog
[View the complete commit history for v1.9.0](#)

---

Thanks for using Studyloop!
