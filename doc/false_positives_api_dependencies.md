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

| `BuildTopicGroupsFromChapters` | `internal/notebook/ingestion.go` | `App.UploadNotebook`, `App.ConfirmNotebookSyllabus` | **Active** |
| `CompressTopicChunks` / `CompressTopicChunksAsync` | `internal/study/compression_service.go` | `App.GetTopicCompressionStats`, `App.GetTopicSectionsContent` | **Active** |
| `DownloadYouTubeVideo` / `IngestYouTubeVideo` / `GetYouTubeSegmentTimestamps` | `internal/notebook/youtube.go` | `App.UploadYouTubeNotebook` | **Active** |
| `ExtractSyllabusChaptersFromMarkdown` / `SplitMarkdownIntoChunks` | `internal/notebook/markdown_chunker.go` | `App.UploadNotebook`, `DraftNotebookSyllabus` | **Active** |
| `InstallZip` / `Uninstall` | `internal/extension/installer.go` | `ExtensionManager` / `App.SetupExtension` | **Active** |
| `RunSmokeTest` | `internal/extension/checker.go` | `CheckReadiness` / `SetupExtensionEnv` | **Active** |
| `bookmarkNodesToDraft` / `parseBookmarkNode` / `walkBookmarkNode` / `findPDFCPUExecutable` | `internal/notebook/pdfcpu.go` | `ExtractFullPDFCPUBookmarkNodes`, `runPDFCPUBookmarksExport` | **Active** |
| `splitDeepPDFMarkdown` / `stripMarkdownHeadings` | `internal/notebook/deep_pdf.go` | `IngestDeepPDFWithProgress` | **Active** |
| `appendFailedQuestionsSection` / `buildSocraticRemedialPrompt` / `checkAndInsertMilestoneExam` / `insertMilestoneForAttempts` / `resolveRetryTopicAndBounds` | `internal/app/app_study_cards.go` | `App.CompleteMilestoneExam`, `App.CompleteSocraticRescue`, `App.RetryFlashcardGeneration` | **Active** |
| `clientEndOfDayUnix` / `mapTaskError` / `queueTaskToScheduledTask` | `internal/app/app_study.go` | `App.GetTodayPlan`, `App.ActivateTask` | **Active** |
| `computeSessionSignature` / `getMachineID` / `getNormalizedMachineID` / `getSessionFilePath` / `resolveClerkPublishableKey` | `internal/app/app_auth.go` | `App.LoginStudent`, `App.RestoreSession`, `App.ClearSession` | **Active** |
| `envHasLLMAPIKey` / `normalizeLLMTierForApp` / `sameLLMSettingsForUI` | `internal/app/app_settings.go` | `App.GetLLMSettings`, `App.UpdateLLMSettings` | **Active** |
| `copyPomoFile` / `scanMP3s` / `shuffleStrings` / `linearToLog` / `playFile` / `playChimeAsync` / `emitState` | `internal/app/app_pomodoro.go`, `internal/pomodoro/services/audio/service.go` | `App.Pomodoro*` endpoints and background audio player loop | **Active** |

---

## 3. Pending Features, Architectural Gates & Cleaned Functions

1. **Frontend Bindings (`appApi.js`) and Extension Readiness**:
   - `checkExtensionReadiness` / `App.CheckExtensionReadiness`: Evaluated for redundancy against `SetupExtension`. While `SetupExtension` provisions virtual environments with `uv` and runs self-tests, `CheckExtensionReadiness` is a read-only fast pre-flight probe (`CheckReadiness` -> `RunSmokeTest --test`) used to verify that an existing Python virtual environment and its dependencies are intact and executable without reinstalling or mutating packages.
2. **Removed Developer Bypass Helpers**:
   - `DevUnlockNotebookCertificate` / `devUnlockNotebookCertificate`: Removed from `Repository` (`internal/db/notebook_certificate.go`), `App` (`internal/app/notebook_endpoints.go`), and `frontend/src/services/appApi.js`.
3. **Pomodoro Subsystem (`App.Pomodoro*`)**:
   - Cleaned up 6 empty stub methods previously: `PomodoroGetStats`, `PomodoroRecordSessionComplete`, `PomodoroCheckResumeSession`, `PomodoroDeleteProfile`, `PomodoroGetSettings`, `PomodoroSaveSettings`.
   - The functional Timer (`Start`, `Pause`, `Resume`, `Stop`, `GetTimerState`), Audio Engine (`PlayLooping`, `PlayShuffleFolder`, `StopAudio`, `SetVolume`, `PlayChime`, `PickMusicFile`, `PickMusicFolder`), and Settings/Profile Persistence (`LoadProfiles`, `SaveProfile`, `GetProfileByID`) remain intact and active.
4. **Internal Authorization & Session Endpoints**:
   - `App.IsProUser` is actively used as an internal backend gate for pro extensions and lo-fi folder shuffling.
   - `App.GetUserSession` and `App.SetSession` are legacy/test helpers.
5. **Test Fixtures**:
   - `SeedDemoDataForTests` in `internal/db/testhelper.go`.
---

## 4. Knip Static Analysis & Frontend Dependency Audit

This section documents the audit of exports and dependencies flagged by Knip (`npx knip`) in the frontend, detailing what was removed as genuine dead code versus what is an intentional false positive / architectural necessity:

### A. Evaluated False Positives (Retained or Scoped)

| Item | File / Location | Classification | Reason / Reality |
| :--- | :--- | :--- | :--- |
| `puppeteer` | `frontend/package.json` | **Root Tooling Dependency** | Flagged as unused in frontend by Knip because it is not imported in `frontend/src`. However, it is actively required by the root script `scripts/generate_slides.mjs` (`import puppeteer from 'puppeteer'`). |
| `activeSetup` | `frontend/src/composables/useExtensions.js` | **Composable Reactive State** | Flagged by Knip when unexported directly. It is actively consumed throughout `ExtensionSetupModal.vue`, `ExtensionSetupToast.vue`, and `Extensions.vue` via the `useExtensions()` composable. |
| `setupModalState` | `frontend/src/composables/useExtensions.js` | **Composable Reactive State** | Flagged by Knip when unexported directly. It is actively consumed by `App.vue` and modal components via `useExtensions()`. |
| `checkExtensionReadiness` | `frontend/src/services/appApi.js` | **Documented Architectural Probe** | Matches backend method `App.CheckExtensionReadiness`. Retained as a fast pre-flight probe (`CheckReadiness` -> `RunSmokeTest --test`) for virtual environments (see Section 3.1). |
| `DEFAULT_EXTENSION_CONFIG` | `frontend/src/composables/useExtensions.js` | **Internal Configuration** | Retained internally in `useExtensions.js` to seed and merge extension configurations. Scoped to file-private (`export` keyword removed). |
| `isRateLimitError` | `frontend/src/composables/useToast.js` | **Internal Helper** | Retained internally in `useToast.js` to intercept 429 errors. Scoped to file-private (`export` keyword removed). |
| `initAudioContext` | `frontend/src/services/calendarService.js` | **Internal Helper** | Retained internally in `calendarService.js` for audio cue playback. Scoped to file-private (`export` keyword removed). |
| `NOTEBOOK_IMPORTERS` | `frontend/src/services/importerRegistry.js` | **Internal Registry** | Retained internally in `importerRegistry.js` and queried by `getActiveNotebookImporters()`. Scoped to file-private (`export` keyword removed). |
| `resolveNoteAssetSrc` / `parseImageSize` | `frontend/src/services/markdown.js` | **Internal Markdown Parser Helpers** | Retained internally in `markdown.js` for custom image and asset token rendering. Scoped to file-private (`export` keyword removed). |

### B. Pruned Obsolete / Dead Code

| Item | Location | Action Taken |
| :--- | :--- | :--- |
| `@clerk/vue` | `frontend/package.json` | **Uninstalled**. Authentication in StudyLoop is architected as an out-of-band desktop loopback flow in the Go backend (`internal/app/app_auth.go` -> `clerkAuth.js` via Wails IPC). The `@clerk/vue` component package was unused runtime overhead and has been safely removed. |
| `submitQuizAttemptWithReflection` | `frontend/src/services/appApi.js` | **Removed**. Redundant duplicate of `submitQuizAttempt(taskID, answers, reflectionText)`, which already proxies to backend `SubmitQuizAttemptWithReflection`. |
| `GetTaskContext` | `frontend/src/services/appApi.js` | **Removed**. Redundant PascalCase alias of `getTaskContext`. Standardized consumer (`QuizAnalysis.vue`) to `getTaskContext`. |
| `formatDaysRemaining` | `frontend/src/utils/dateFormat.js` | **Removed**. Superseded by local short-format pace presentation (`formatDaysRemainingShort` in `TelemetryWidget.vue`). |

---

## 5. Dead Code Audit: Go Backend (Unreachable Warnings & Test Seams)

This section documents internal Go functions and methods flagged by static reachability analyzers as "unreachable" from `main()`, clarifying their architectural role, test coverage, and whether they are active, test seams, lifecycle hooks, or dormant features:

| Function / Method | Defined In | Classification | Architectural Role & Reality |
| :--- | :--- | :--- | :--- |
| `WithExtractPDFFunc` | `internal/notebook/upload.go:47` | **Intentional Test Seam** | Functional option (`Option`) allowing tests to mock the PDF extraction engine without touching disk or reading real PDF files. Actively called in `internal/notebook/upload_test.go` and `internal/app/notebook_upload_test.go`. Removing this breaks unit test compilation. |
| `BootResult.GetRepo` | `internal/runtime/boot.go:81` | **Thread-Safe Accessor / Test Utility** | Thread-safe accessor (`RWMutex`) for the active SQLite repository. Actively exercised and verified in concurrency tests in `internal/runtime/boot_test.go`. |
| `BootResult.GetEmbedder` | `internal/runtime/boot.go:95` | **Thread-Safe Accessor / Test Utility** | Thread-safe accessor (`RWMutex`) for the ONNX embedder instance. Actively tested in `internal/runtime/boot_test.go`. |
| `BootResult.IsAIReady` | `internal/runtime/boot.go:102` | **Thread-Safe Accessor / State Query** | Thread-safe status query (`RWMutex`) returning whether AI/vector capabilities are active. Verified in concurrency tests in `internal/runtime/boot_test.go`. |
| `BootResult.GetAIInitError` | `internal/runtime/boot.go:109` | **Error Inspection Accessor** | Returns the raw underlying `error` object for error unwrapping / `errors.Is(...)` checks. Tested in `internal/runtime/boot_test.go`. Not a duplicate of `GetAIInitErrorString`. |
| `BootResult.GetAIInitErrorString` | `internal/runtime/boot.go:116` | **IPC / String Error Accessor** | Returns the string representation of any AI initialization error, designed for serialized frontend boundary reporting. Tested in `internal/runtime/boot_test.go`. Not a duplicate of `GetAIInitError`. |
| `BootResult.Close` | `internal/runtime/boot.go:138` | **Lifecycle Teardown Hook** | Gracefully releases runtime resources by cancelling delayed startup backups, closing the ONNX embedder session, and shutting down the database connection pool. Tested in `internal/runtime/boot_test.go`. Standard Go hygiene for graceful teardown and integration test cleanup. |
| `BootResult.InitializeAI` | `internal/runtime/boot.go:278` | **Dormant / Dynamic RAG Initializer** | Entry point for on-demand RAG initialization: verifies assets, stages DLLs, reloads SQLite with `sqlite-vec`, and brings up the ONNX embedder stack. Built for on-demand or post-bootstrap RAG enablement. |
| `BootResult.initEmbedderStack` | `internal/runtime/boot.go:351` | **Internal Initializer Helper** | Helper method called directly by `BootResult.InitializeAI` (line 347) to instantiate the ONNX embedder, load prompt tokenizers, and initialize vector table schemas. Flagged solely as a consequence of `InitializeAI`. |

