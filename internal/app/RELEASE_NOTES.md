## ✨ Features

- **Flashcards**
  - Added a **Copy** button to quickly duplicate flashcard content.  
  - Improved accessibility for screen‑reader users.

- **Dynamic Prompt Compression**
  - Background compression service (LLMLingua‑2) automatically trims oversized document chunks, reducing token usage and speeding up AI responses.  
  - New settings let you control compression mode and target token retention.  
  - UI now shows compression statistics and token‑saving badges in the Reader view.

- **Extension Setup Experience**
  - Real‑time **ExtensionSetupToast** and **ExtensionSetupModal** give clear feedback while extensions install.  
  - Progress badges and status indicators keep you informed of each installation step.

- **Toast System Revamp**
  - Legacy toast notifications replaced with a modern **useToast** composable.  
  - Updated styling includes a subtle backdrop filter for better readability.

- **Smooth Quiz Interaction**
  - After submitting a quiz or scoring an answer, the page now smoothly scrolls to the top, keeping the flow natural.

- **Authentication Enhancements**
  - More reliable session handling and synchronization across the app, reducing unexpected log‑outs.  
  - Improved session verification and observability for a smoother login experience.

- **Compression Badge Component**
  - A new UI badge lets you toggle detailed information about active compression settings.

## 🚀 Improvements

- **Search Performance**
  - Lexical search now caches TF vectors and reduces lock contention, delivering faster results.

- **UI Consistency**
  - Refreshed styling for setup badges, buttons, and toast components for a cleaner look across the app.

- **Environment Configuration**
  - Build script now respects the `ClerkPublishableKey` environment variable, simplifying deployments.

- **Logging**
  - Dynamic log‑level configuration and persistence make troubleshooting easier without affecting the UI.

- **Study Queue Queries**
  - Refactored queries improve readability and maintainability, indirectly enhancing reliability.

## 🐛 Bug Fixes & Issue Resolutions

- Fixed a crash when the notebook drafting toast was shown in an unexpected syllabus state.  
- Resolved a panic that could occur during quiz transaction rollbacks.  
- Ensured proper closure of resources in various backend functions, preventing leaks.  
- Added error handling for attempts to delete files outside the designated upload directory.  
- Corrected the privacy policy URL in Settings.  
- Improved log‑level restoration to avoid hidden errors.  
- Prevented session rollback panics and enhanced error handling in logging.  
- Updated markdown rendering to normalize LaTeX delimiters for better KaTeX compatibility.  

*Thanks for using Studyloop!*
