# What's New – v1.10.0

## ✨ Features
- **Real‑time update progress** – The app now emits an `update:progress` event, showing download statistics (MB downloaded vs total) during updates.  
- **Clear conversation button** – Users can instantly wipe the current chat history with a single click.  
- **Default retrieval scope** – The reader now automatically searches within the current chapter, and quick‑action chips are available without emojis.  
- **Enhanced clipboard support** – Improved copying of session content and richer clipboard interactions.  
- **Tutor style shortcuts** – Direct, detailed, and Socratic tutoring modes can be toggled from the header for faster workflow.  
- **Dynamic prompt logging** – LLM prompt logs are now resolved automatically for both development and production environments.  

## 🚀 Improvements
- **Chat component refactor** – Switched to composable functions for state management, making the codebase cleaner and easier to maintain.  
- **BaseIcon adoption** – Replaced raw emojis across the UI with the new `BaseIcon` SVG component, delivering a consistent visual language and fixing several prop mismatches (e.g., `custom-class`).  
- **Icon registry expansion** – Added a comprehensive set of SVG icons (grid, flame, alert‑triangle, gift, cloud, etc.) and unit‑tested the `BaseIcon` component.  
- **Responsive layout tweaks** – Streamlined viewport and flexbox calculations for the reader and overall UI, ensuring a stable height on all devices.  
- **Navigation enhancements** – Integrated `useRouter` in Socratic views for smoother page transitions.  

## 🐛 Bug Fixes
- Fixed brittle viewport‑height calculations that caused layout glitches on certain screens.  
- Resolved chat flexbox height issues, improving responsiveness on mobile and desktop.  

## 🧹 Maintenance
- Updated the README with richer descriptions and a new retention‑loop diagram.  
- Added a design‑linter rule (`EMOJI_HAS_SVG`) and de‑duplicated smell detectors.  
- Introduced linting and testing scripts for the new `BaseIcon` implementation.  

## 📦 Full Changelog
[View all changes](https://github.com/your-repo/Studyloop/compare/v1.9.0...v1.10.0)

Thanks for using Studyloop!
