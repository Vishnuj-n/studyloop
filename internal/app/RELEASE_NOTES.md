# What's New – v1.4.1

### ✨ Features
- **Smarter chapter matching** – Overlapping page ranges are now considered when matching study chapters, improving syllabus accuracy.
- **Study queue visibility** – The notebook syllabus modal now shows a "Status" column with clear queued/not-queued indicators.
- **Persistent page tracking** – Each entry in the study queue records its own `current_page`, allowing independent progress for multiple reading tasks.
- **Asset manifest** – Added a manifest file and optimized file-copy logic to avoid redundant operations during asset handling.
- **LLM output budgeting** – The system now respects explicit output-token limits, automatically routing requests based on context window limits.
