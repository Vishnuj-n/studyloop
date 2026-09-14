# v1.5.0 – Release Notes  

## What's New  

### ✨ Features
- **Gamification overhaul**
  - Real‑time rewards updates via a new `gamification‑updated` event, keeping the Dashboard and Rewards pages in sync.  
  - Introduced a loot‑box system with tiered rewards (Bronze → Mythic) and new progression models (XP elixirs, luck charms, wager support).  
  - Rebalanced rank progression with numeric levels and sub‑tiers, plus achievement reconciliation logic.  
- **Assessment & Socratic enhancements**
  - Refined Quiz and Written Assessment components with better state handling, retry mechanisms, and accessibility (ARIA live regions, improved inputs).  
  - Added distinct instruction sets for Socratic “rescue” mode vs. general tutoring, plus context‑length validation for prompts.  
- **Study workflow upgrades**
  - Expanded task tracking to include all completed study‑queue tasks.  
  - Added partial‑quiz recovery parsing and assessment updates.  
  - Optimized notebook ingestion with explicit chunk handling and safer deletions.  
- **User Interface & Theming**
  - New premium themes: Dark Academia, Neon Cyberpunk, Zen Minimalist, with updated global CSS variables.  
  - Replaced static icons with scalable SVGs; added custom scrollbars and a top‑bar GitHub button with a dismissable star CTA.  
  - Enhanced StatusBanner for GitHub star call‑out and introduced a one‑time Release Notes modal that appears on first launch of a new version.  
  - Updated Groq link text and added a GitHub star toast for community engagement.  
- **LLM output**
  - Enabled free‑flow generation and removed hard caps on maximum output tokens.  

### 🚀 Improvements
- **Reading session & logging**
  - More robust session activation with detailed structured logs and consistent payloads.  
  - Adjusted navigation logic to correctly reflect page bounds and loading states.  
- **PDF viewer & chat**
  - Stabilized PDF container width to cut down costly canvas re‑renders; refined page‑jump handling.  
  - Added ARIA attributes to ReaderChat for better screen‑reader support.  
- **Error handling & reliability**
  - Strengthened streak protection with an auto‑protection loop that can span multiple missed days.  
  - Added validation for time parsing in daily study‑minute calculations and warning logs for failed review session creation.  
  - Improved queue transition tasks with clearer error handling and score management.  
- **Settings & Dashboard**
  - Safer profile switching with enhanced error handling and smoother data loading.  
  - Refactored GitHub repo link logic and improved dashboard state management.  

### 🐛 Bug Fixes
- Removed redundant CSS selectors in `RewardsShopModal`.  
- Fixed page navigation edge cases when topic bundles fail to load.  
- Cleaned up unused state variables, imports, and dead code across multiple UI components.  

### 🧹 Maintenance
- Comprehensive frontend cleanup: eliminated dead code, unused functions, and stale CSS.  
- Refactored numerous components for clearer logic and reduced bundle size.  
- Added integration tests for notebook repository fixes and chunk deletion.  
- Updated test expectations to match new streak‑completion logic.  

## 📦 Full Changelog
[View the complete commit history for v1.5.0](#)  

---  

Thanks for using Studyloop!
