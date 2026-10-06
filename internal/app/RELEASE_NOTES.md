## ✨ Features

- **Pet Companion Revamp** – The new fox pet “Rusty” replaces the dragon, and additional unlockable pets and skins are now available. Access the Pet Sanctuary, view and toggle your current pet and skin from the Rewards overview, and enjoy fresh pet visuals with the new Pet Avatar component.  
- **Bi‑directional Study Loop** – Seamlessly move between Flashcards, Study Notes, and the Reader. New “View Notes” button on Flashcards, contextual back buttons in Notes and Reader, and preserved navigation context for a smoother study flow.  
- **Structured Study Notes** – Study notes are now stored as Markdown files with YAML front‑matter, supporting page‑range generation, persistent storage, and easy offline access. A dedicated Notes page view, API client, and milestone triggers enhance note management.  
- **Rate‑Limit Banner & Settings** – A visible banner informs you when LLM requests are throttled. New settings let you choose between “Standard” and “Paced” rate‑limit strategies.  
- **Tabbed Sidebar for Chat & Notes** – The reader sidebar now has tabs to switch between the AI chat assistant and the Chapter Study Note view, replacing the old modal drawer.  
- **Overflow Menu in Reader** – Quick actions (copy, audio, simplify, add note) are now available from an overflow menu in the Reader.  
- **Enhanced Icons & Toasts** – New SVG icons (message square, send, chevron, sidebar) and a unified BaseIcon component improve visual consistency, including toast notifications.  

## 🚀 Improvements

- **UI Consistency** – Refreshed border styles, box‑shadows, and component styling across RateLimitBanner, ReaderChat, Notes, and other UI elements.  
- **Compression Feedback** – Real‑time compression status badge with batch progress, plus performance‑focused batching and callbacks for faster note generation.  
- **Pacing Precision** – More accurate daily session calculations and pacing logic for smoother study sessions.  
- **Notebook Filtering** – Topics now show only those with existing notes or active progress, decluttering the notebook view.  
- **Flashcards Enhancements** – Added button to view associated notes directly from Flashcards.  
- **Navigation Context** – Reader now retains `fromOrigin` and `taskId` when returning to notes, ensuring continuity.  
- **Pet UI** – Updated layout and styling for pet selection, skin display, and sanctuary navigation.  
- **Settings UI** – New controls for rate‑limit strategy and other preferences.  

## 🐛 Bug Fixes & Issue Resolutions

- Fixed missing or broken toast notifications in chat components.  
- Resolved navigation bugs that lost context when moving between Reader, Notes, and Flashcards.  
- Corrected unused CSS and style inconsistencies.  
- Patched issues with pet skin unlocking and display.  
- Fixed per‑session range indexing for study notes.  
- Addressed errors in the compression pipeline and status tracking.  
- Fixed overflow‑menu actions and related UI glitches.  
- Removed legacy note‑drawer remnants and backdrop issues.  
- Fixed pagination and data retrieval for review‑session cards.  
- Resolved various minor UI glitches across the app.  

---  
*Thanks for using Studyloop!*
