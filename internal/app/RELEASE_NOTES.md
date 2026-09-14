# What's New in **v1.5.0**

Studyloop 1.5.0 brings a richer gamified experience, smarter study tools, and a polished UI. Highlights include a new loot‑box reward system, expanded theme options, real‑time streak protection, and a one‑time release‑notes modal to keep you informed.

## ✨ Features

- **Gamification upgrades**
  - Real‑time rewards updates via a `gamification‑updated` event, keeping the Dashboard and Rewards pages in sync.
  - Robust streak‑protection loop that automatically applies streak freezes across missed days.
  - New loot‑box system with tiered rewards (Bronze, Silver, Gold, Mythic) and expanded progression models (XP elixirs, luck charms, user wagers).
  - Rebalanced rank progression with numerical levels and sub‑tiers for clearer advancement.
  - Achievement reconciliation logic to ensure earned milestones are correctly recorded.

- **Study workflow enhancements**
  - Expanded task tracking now includes all completed study‑queue tasks.
  - Optimized notebook ingestion with explicit chunk handling and safer deletions.
  - Partial quiz‑recovery parsing and assessment updates for smoother resume after interruptions.

- **Assessment UI improvements**
  - Refined Quiz component with a new `isCorrect` helper, retry mechanisms, and automatic state reset on task changes.
  - Written Assessment now features better button states, dynamic shortcut hints, and enhanced accessibility (ARIA live regions).

- **Socratic tutoring**
  - Distinct instruction sets for “rescue” (remedial) and general tutoring modes.
  - Context‑block length validation and smarter prompt construction for more reliable answers.

- **User interface & experience**
  - Three brand‑new visual themes: **Dark Academia**, **Neon Cyberpunk**, and **Zen Minimalist**.
  - Updated gamification sidebar with level chips and XP text.
  - SVG‑based icons for crisp scaling across devices.
  - Top‑bar GitHub button, polished star call‑to‑action, and custom scrollbars.
  - Dismissable GitHub‑star banner and toast notifications to celebrate community support.
  - One‑time release‑notes modal that appears after each version upgrade.

- **LLM output**
  - Free‑flow generation mode with removed token caps for more natural responses.

## 🚀 Improvements

- Reading session initialization now logs structured error details and provides consistent navigation state.
- PDF viewer performance boosted by stabilizing container width, reducing canvas re‑renders.
- ReaderChat component gains ARIA attributes for better screen‑reader support.
- Transition‑task handling in the queue includes stronger error handling and score management.
- Settings profile switching now includes graceful error handling and faster data loading.
- Dashboard GitHub repository link refactored for reliability.
- StatusBanner enhancements for clearer GitHub star prompts.
- Groq link text clarified and UI refined.
- Global CSS variables updated to support the new themes.
- Miscellaneous UI polish: custom scrollbars, improved typography on Notebook cards, and streamlined key‑down handling on the Socratic composer.

## 🐛 Bug Fixes

- Removed redundant CSS selectors in `RewardsShopModal` to prevent styling conflicts.
- Fixed page navigation calculations to correctly reflect the current reading position even when topic bundles fail to load.

## 🧹 Maintenance

- Cleaned up unused component logic, dead code, and obsolete imports across the frontend.
- Added a skip‑release option and enhanced commit‑history retrieval for smoother release automation.
- Integrated integration tests for notebook repository fixes and chunk deletion.
- Updated release scripts to automatically generate and embed `RELEASE_NOTES.md`.
- Removed unused onboarding CSS styles and lock‑overlay assets.

## 📦 Full Changelog

[View all changes between v1.4.0 and v1.5.0](https://github.com/your-repo/Studyloop/compare/v1.4.0...v1.5.0)

---

Thanks for using **Studyloop**!
