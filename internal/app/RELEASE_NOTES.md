## ✨ Features

- **Mastery Certificates**  
  - Earn a downloadable certificate when you complete a notebook (100% progress).  
  - View the certificate in a new modal, with customizable recipient name and a polished design that respects your chosen theme.  
  - Developers can instantly unlock certificates from the Settings → Developer panel for testing.

- **Export & Copy Certificates**  
  - Export certificates as images or copy them directly to the clipboard with a single click.  
  - Visual feedback lets you know when the copy succeeds or if an error occurs.

- **Viva Exam Generation**  
  - After finishing a quiz, you can now generate a written “viva” exam that re‑uses the same questions and concepts, providing a seamless follow‑up assessment.

- **Inline Flashcard Editing**  
  - Edit flashcard questions and answers directly inside the deck manager modal. Changes are saved instantly, making it easy to keep your study material up‑to‑date.

- **Predicted Review Intervals**  
  - While reviewing flashcards, each rating button now shows the estimated next review time (e.g., “1 d”, “3 mo”), helping you understand the impact of your rating choices.

## 🚀 Improvements

- **Flashcard UI Refresh**  
  - Updated layout and styling for a cleaner, more readable flashcard experience.  
  - Added a top bar with file‑type indicators and quick‑action buttons on notebook cards.

- **Certificate UI & Accessibility**  
  - Refactored colors to use theme tokens and improved contrast for better accessibility.  
  - Added error handling and alerts for clipboard copy failures.  
  - Simplified navigation between quiz and assessment pages, preserving the `flashcardsPending` flag only when needed.

- **Consistent Styling**  
  - Unified border styles using the `outline-variant` token across the app.  
  - Introduced a design‑rule check to prevent low‑contrast pastel text.

- **Developer Experience**  
  - Replaced native confirmation dialogs with a custom dialog component for a consistent look and feel.

## 🐛 Bug Fixes & Issue Resolutions

- Fixed clipboard copy errors on the certificate modal, now showing a clear alert and auto‑reset timer when copying fails.  
- Resolved navigation glitches that could cause redundant flashcard generation when moving from quizzes to written assessments.  
- Corrected quiz task ID handling to ensure proper redirection to the examiner view.  

---

*Thanks for using Studyloop!*
