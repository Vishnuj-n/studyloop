# API Dependency Analysis: False Positives & Status Reference
<!-- Reference document to avoid rescanning and false reporting -->

## 1. Executive Summary

In `doc/api_dependency_report.md`, **Section 3 flagged 66 internal Go functions and methods as "unreachable / dead code"**. 

**Almost all of them are FALSE POSITIVES.**
The Go codebase compiles cleanly (`go vet ./internal/...` passed with 0 warnings), all unit tests pass (`go test -short ./internal/...`), and these functions are actively called inside the Go backend.

### Why Did the Generator Report Them as Unreachable?
The scanner assumed all `App` methods were located solely in `internal/app/app.go`. Because the Go methods are distributed across multiple files (`notebook_endpoints.go`, `app_study.go`, `app_settings.go`, `app_extension.go`, etc.), any internal helpers invoked from those files were flagged as having no caller.

---

## 2. False Positives (Live, Active Go Code)

These functions were falsely marked as unreachable, but are **actively called in production Go code**:

| Function / Method | Defined In | Actively Called By | Status |
| :--- | :--- | :--- | :--- |
| `DraftSyllabusChapters` | `internal/notebook/syllabus.go` | `App.DraftNotebookSyllabus` (`notebook_endpoints.go`) | **Active** |
| `finalizeNotebookUpload` | `internal/app/notebook_endpoints.go` | `App.UploadNotebook` | **Active** |
| `finalizeDeepStructuredPDFUpload` | `internal/app/notebook_endpoints.go` | `App.SelectAndUploadDeepStructuredPDF` | **Active** |
| `persistSyllabusDraft` | `internal/app/notebook_endpoints.go` | `DraftNotebookSyllabus`, `AICleanupNotebookSyllabus` | **Active** |
| `requireRepo` | `internal/app/app_study.go` | Called 24+ times across study, cards, and reading endpoints | **Active** |
| `calculateDailyStudyMinutes` | `internal/app/app_study.go` | `App.GetTodayPlan` | **Active** |
| `calculateFlashcardBudgets` | `internal/app/app_study.go` | `App.GetTodayPlan` | **Active** |
| `aggregateQueueTasks` | `internal/app/app_study.go` | `App.GetTodayPlan` | **Active** |
| `buildReviewTaskForPlan` | `internal/app/app_study.go` | `App.GetTodayPlan` | **Active** |
| `computeCurrentStreak` / `computeLongestStreak` | `internal/app/app_study.go` | `App.GetDashboardOverview`, `App.GetGamificationState` | **Active** |
| `reconcileConfirmedNotebookTask` | `internal/app/notebook_endpoints.go` | `ConfirmNotebookSyllabus`, `UpdateNotebookStudyStatus` | **Active** |
| `runDeepPDFExtraction` | `internal/app/notebook_endpoints.go` | `SelectAndUploadDeepStructuredPDF`, `UpgradeNotebookToDeepPDF` | **Active** |
| `IngestDeepPDFWithProgress` | `internal/notebook/deep_pdf.go` | `runDeepPDFExtraction` | **Active** |
| `ExtractFullPDFCPUBookmarkNodes` | `internal/notebook/pdfcpu.go` | `DraftSyllabusChapters` | **Active** |
| `ParsePDFCPUBookmarkDraftFromJSON` | `internal/notebook/pdfcpu.go` | `DraftSyllabusChapters` | **Active** |
| `runPDFCPUBookmarksExport` | `internal/notebook/pdfcpu.go` | `DraftSyllabusChapters` | **Active** |
| `TransitionTask` | `internal/study/queue_transition.go` | Called 12+ times across study card & reading endpoints | **Active** |
| `awardCompletionRewards` | `internal/study/queue_transition.go` | `TransitionTask` | **Active** |
| `RollTaskRewards` | `internal/study/rewards.go` | `awardCompletionRewards` | **Active** |
| `generateLootBox` | `internal/study/rewards.go` | `awardCompletionRewards` | **Active** |
| `CreateReviewSession` | `internal/db/review_session_repo.go` | `App.GetReviewSession` | **Active** |
| `ONNX Embedder Internals` (`embedBatchInternal`, `buildInputValues`, etc.) | `internal/embeddings/onnx.go` | `OnnxEmbedder.Embed`, `OnnxEmbedder.EmbedBatch` | **Active** |
| `Extension Setup Helpers` (`CheckReadiness`, `SetupExtensionEnv`, `GetEffectiveTier`) | `internal/extension/` | `App.CheckExtensionReadiness`, `App.SetupExtension` | **Active** |
| `Keyring Store Helpers` (`SaveAPIKey`, `DeleteAPIKey`, `MarkLLMKeyStored`) | `internal/llm/keyring.go`, `internal/db/store.go` | `App.SaveLLMAPIKey`, `App.DeleteLLMAPIKey` | **Active** |

---

## 3. Truly Unused / Pending Features & Cleaned Stubs

1. **Pomodoro Subsystem (`App.Pomodoro*`)**:
   - Cleaned up 6 empty stub methods from backend and frontend: `PomodoroGetStats`, `PomodoroRecordSessionComplete`, `PomodoroCheckResumeSession`, `PomodoroDeleteProfile`, `PomodoroGetSettings`, `PomodoroSaveSettings`.
   - The functional Timer (`Start`, `Pause`, `Resume`, `Stop`, `GetTimerState`), Audio Engine (`PlayLooping`, `PlayShuffleFolder`, `StopAudio`, `SetVolume`, `PlayChime`, `PickMusicFile`, `PickMusicFolder`), and Settings/Profile Persistence (`LoadProfiles`, `SaveProfile`, `GetProfileByID`) remain intact and active.
2. **Internal Authorization & Session Endpoints**:
   - `App.IsProUser` is actively used as an internal backend gate for pro extensions and lo-fi folder shuffling.
   - `App.GetUserSession` and `App.SetSession` are legacy/test-only helpers.
3. **Test Fixtures**:
   - `SeedDemoDataForTests` in `internal/db/testhelper.go`.
