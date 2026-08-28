# API & Function Dependency Report
This report lists unused API endpoints, dead Go backend code, and reachability stats.

## 1. Frontend API (`appApi.js`) Usages
Lists functions defined in `appApi.js` and their usage count in the frontend.
| JS Function | Calls Wails Method | Usage Count | Status |
| --- | --- | --- | --- |
| `GetTaskContext` | `GetTaskContext` | 2 | Active |
| `activateTask` | `ActivateTask` | 4 | Active |
| `aiCleanupNotebookSyllabus` | `AICleanupNotebookSyllabus` | 1 | Active |
| `askReaderAI` | `AskReaderAI` | 1 | Active |
| `askSocratic` | `AskSocratic` | 1 | Active |
| `assignNotebookToProfile` | `AssignNotebookToProfile` | 1 | Active |
| `cancelExtensionSetup` | `CancelExtensionSetup` | 1 | Active |
| `checkExtensionReadiness` | `CheckExtensionReadiness` | 1 | Active |
| `checkForUpdates` | `CheckForUpdates` | 2 | Active |
| `completeMilestoneExam` | `CompleteMilestoneExam` | 1 | Active |
| `completeReading` | `CompleteReading` | 1 | Active |
| `completeReviewSession` | `CompleteReviewSession` | 1 | Active |
| `completeSocraticRescue` | `CompleteSocraticRescue` | 2 | Active |
| `confirmNotebookSyllabus` | `ConfirmNotebookSyllabus` | 1 | Active |
| `createProfile` | `CreateProfile` | 2 | Active |
| `deleteLLMAPIKey` | `DeleteLLMAPIKey` | 1 | Active |
| `deleteNotebook` | `DeleteNotebook` | 1 | Active |
| `deleteProfile` | `DeleteProfile` | 1 | Active |
| `devForceFlashcardGenerate` | `DevForceFlashcardGenerate` | 1 | Active |
| `devForceSocraticRescue` | `DevForceSocraticRescue` | 1 | Active |
| `draftNotebookSyllabus` | `DraftNotebookSyllabus` | 1 | Active |
| `forceDueFlashcardsNow` | `ForceDueFlashcardsNow` | 1 | Active |
| `generateComprehensiveExam` | `GenerateComprehensiveExam` | 1 | Active |
| `generateFlashcardsForQuizTask` | `GenerateFlashcardsForQuizTask` | 1 | Active |
| `generateManualFlashcards` | `GenerateManualFlashcards` | 1 | Active |
| `generateQuizForPageRange` | `GenerateQuizForPageRange` | 1 | Active |
| `getAppEnv` | `GetAppEnv` | 2 | Active |
| `getAvailableTopics` | `GetAvailableTopics` | 2 | Active |
| `getCloudConfig` | `GetCloudConfig` | 1 | Active |
| `getDashboardOverview` | `GetDashboardOverview` | 1 | Active |
| `getFlashcardDueTimeline` | `GetFlashcardDueTimeline` | 1 | Active |
| `getLLMProviderPreset` | `GetLLMProviderPreset` | 2 | Active |
| `getLLMSettings` | `GetLLMSettings` | 1 | Active |
| `getNotebookTopicTree` | `GetNotebookTopicTree` | 1 | Active |
| `getNotebooks` | `GetNotebooks` | 7 | Active |
| `getProfileDailyPace` | `GetProfileDailyPace` | 1 | Active |
| `getProfiles` | `GetProfiles` | 1 | Active |
| `getReaderTopicBundle` | `GetReaderTopicBundle` | 1 | Active |
| `getReviewSession` | `GetReviewSession` | 1 | Active |
| `getTask` | `GetTask` | 1 | Active |
| `getTodayPlan` | `GetTodayPlan` | 1 | Active |
| `getTopicSectionsContent` | `GetTopicSectionsContent` | 3 | Active |
| `getUserSettings` | `GetUserSettings` | 8 | Active |
| `initializeRAG` | `InitializeRAG` | 2 | Active |
| `initializeReadingSession` | `InitializeReadingSession` | 1 | Active |
| `isOnboarded` | `IsOnboarded` | 1 | Active |
| `listExtensions` | `ListExtensions` | 2 | Active |
| `logFrontendEvent` | `LogFrontendEvent` | 4 | Active |
| `loginStudent` | `LoginStudent` | 1 | Active |
| `logoutStudent` | `LogoutStudent` | 1 | Active |
| `openRepoURL` | `OpenRepoURL` | 2 | Active |
| `openURLInBrowser` | `OpenURLInBrowser` | 1 | Active |
| `recordCardReview` | `RecordCardReview` | 1 | Active |
| `retryFlashcardGeneration` | `RetryFlashcardGeneration` | 2 | Active |
| `runExtension` | `RunExtension` | 1 | Active |
| `saveLLMAPIKey` | `SaveLLMAPIKey` | 2 | Active |
| `scoreShortAnswer` | `ScoreShortAnswer` | 1 | Active |
| `setupExtension` | `SetupExtension` | 1 | Active |
| `signUpStudent` | `SignUpStudent` | 1 | Active |
| `simplifyReadingContent` | `SimplifyReadingContent` | 1 | Active |
| `startBrowserAuth` | `StartBrowserAuth` | 1 | Active |
| `startTopicAudioOverview` | `StartTopicAudioOverview` | 1 | Active |
| `stopTopicAudioOverview` | `StopTopicAudioOverview` | 1 | Active |
| `submitQuizAttempt` | `SubmitQuizAttempt` | 1 | Active |
| `suspendFlashcard` | `SuspendFlashcard` | 1 | Active |
| `testLLMConnection` | `TestLLMConnection` | 1 | Active |
| `trackAnalyticsEvent` | `TrackAnalyticsEvent` | 2 | Active |
| `triggerCloudSync` | `TriggerCloudSync` | 2 | Active |
| `updateLLMSettings` | `UpdateLLMSettings` | 2 | Active |
| `updateNotebookPriority` | `UpdateNotebookPriority` | 1 | Active |
| `updateNotebookStudyStatus` | `UpdateNotebookStudyStatus` | 1 | Active |
| `updateNotebookTitle` | `UpdateNotebookTitle` | 1 | Active |
| `updateProfile` | `UpdateProfile` | 1 | Active |
| `updateUserSettings` | `UpdateUserSettings` | 4 | Active |
| `uploadFastPDFNotebook` | `UploadFastPDFNotebook` | 1 | Active |
| `uploadNotebook` | `UploadNotebook` | 1 | Active |
| `uploadYouTubeNotebook` | `UploadYouTubeNotebook` | 1 | Active |

## 2. Go Wails `App` API Endpoints
Lists exported API endpoints defined on `App` in the backend and whether they are invoked by `appApi.js` (excluding standard Wails lifecycle methods).
| Wails Method | Reachable from Frontend or main.go |
| --- | --- |
| `AICleanupNotebookSyllabus` | Yes |
| `ActivateTask` | Yes |
| `AskReaderAI` | Yes |
| `AskSocratic` | Yes |
| `AssignNotebookToProfile` | Yes |
| `CancelExtensionSetup` | Yes |
| `CheckExtensionReadiness` | Yes |
| `CheckForUpdates` | Yes |
| `CompleteMilestoneExam` | Yes |
| `CompleteReading` | Yes |
| `CompleteReviewSession` | Yes |
| `CompleteSocraticRescue` | Yes |
| `ConfirmNotebookSyllabus` | Yes |
| `CreateProfile` | Yes |
| `DeleteLLMAPIKey` | Yes |
| `DeleteNotebook` | Yes |
| `DeleteProfile` | Yes |
| `DevForceFlashcardGenerate` | Yes |
| `DevForceSocraticRescue` | Yes |
| `DraftNotebookSyllabus` | Yes |
| `ForceDueFlashcardsNow` | Yes |
| `GenerateComprehensiveExam` | Yes |
| `GenerateFlashcardsForQuizTask` | Yes |
| `GenerateManualFlashcards` | Yes |
| `GenerateQuizForPageRange` | Yes |
| `GetAppEnv` | Yes |
| `GetAvailableTopics` | Yes |
| `GetCloudConfig` | Yes |
| `GetCtx` | Yes |
| `GetDashboardOverview` | Yes |
| `GetFlashcardDueTimeline` | Yes |
| `GetLLMProviderPreset` | Yes |
| `GetLLMSettings` | Yes |
| `GetNotebookTopicTree` | Yes |
| `GetNotebookUploadDir` | Yes |
| `GetNotebooks` | Yes |
| `GetProfileDailyPace` | Yes |
| `GetProfiles` | Yes |
| `GetReaderTopicBundle` | Yes |
| `GetReviewSession` | Yes |
| `GetTask` | Yes |
| `GetTaskContext` | Yes |
| `GetTodayPlan` | Yes |
| `GetTopicSectionsContent` | Yes |
| `GetUserSettings` | Yes |
| `InitializeRAG` | Yes |
| `InitializeReadingSession` | Yes |
| `IsOnboarded` | Yes |
| `ListExtensions` | Yes |
| `LogFrontendEvent` | Yes |
| `LoginStudent` | Yes |
| `LogoutStudent` | Yes |
| `OpenRepoURL` | Yes |
| `OpenURLInBrowser` | Yes |
| `RecordCardReview` | Yes |
| `RetryFlashcardGeneration` | Yes |
| `RunExtension` | Yes |
| `SaveLLMAPIKey` | Yes |
| `ScoreShortAnswer` | Yes |
| `SetupExtension` | Yes |
| `SignUpStudent` | Yes |
| `SimplifyReadingContent` | Yes |
| `StartBrowserAuth` | Yes |
| `StartTopicAudioOverview` | Yes |
| `StopTopicAudioOverview` | Yes |
| `SubmitQuizAttempt` | Yes |
| `SuspendFlashcard` | Yes |
| `TestLLMConnection` | Yes |
| `TrackAnalyticsEvent` | Yes |
| `TriggerCloudSync` | Yes |
| `UpdateLLMSettings` | Yes |
| `UpdateNotebookPriority` | Yes |
| `UpdateNotebookStudyStatus` | Yes |
| `UpdateNotebookTitle` | Yes |
| `UpdateProfile` | Yes |
| `UpdateUserSettings` | Yes |
| `UploadFastPDFNotebook` | Yes |
| `UploadNotebook` | Yes |
| `UploadYouTubeNotebook` | Yes |

## 3. Go Internal Reachability Analysis
Lists internal Go functions/methods and unexported `App` helpers that are **not reachable** from any active frontend API or lifecycle entry point (excluding tests).
| Function/Method | File | Receiver | Type |
| --- | --- | --- | --- |
| `BuildBreadcrumbText` | `internal\notebook\markdown_chunker.go:277` | None | `function` |
| `BuildTopicGroupsFromChapters` | `internal\notebook\ingestion.go:22` | None | `function` |
| `CheckReadiness` | `internal\extension\checker.go:32` | None | `function` |
| `CreateReviewSession` | `internal\db\review_session_repo.go:172` | `Repository` | `method` |
| `DeleteAPIKey` | `internal\llm\keyring.go:37` | None | `function` |
| `DraftSyllabusChapters` | `internal\notebook\syllabus.go:33` | `Service` | `method` |
| `ExtractFullPDFCPUBookmarkNodes` | `internal\notebook\pdfcpu.go:73` | None | `function` |
| `ExtractSyllabusChaptersFromMarkdown` | `internal\notebook\markdown_chunker.go:196` | None | `function` |
| `GetEffectiveTier` | `internal\extension\tiers.go:17` | None | `function` |
| `GetRereadAttemptCount` | `internal\db\reread_attempts_repo.go:10` | `Repository` | `method` |
| `IngestFastPDF` | `internal\notebook\fast_pdf.go:35` | `Service` | `method` |
| `IngestFastPDFWithProgress` | `internal\notebook\fast_pdf.go:40` | `Service` | `method` |
| `IngestYouTubeVideo` | `internal\notebook\youtube.go:32` | `Service` | `method` |
| `InstallZip` | `internal\extension\installer.go:45` | `Manager` | `method` |
| `MakeAllFlashcardsDueNow` | `internal\db\flashcard_repo.go:503` | `Repository` | `method` |
| `MarkLLMKeyStored` | `internal\db\store.go:597` | `Repository` | `method` |
| `NormalizeSyllabusChapters` | `internal\notebook\syllabus.go:199` | None | `function` |
| `ParsePDFCPUBookmarkDraftFromJSON` | `internal\notebook\pdfcpu.go:85` | None | `function` |
| `RunSmokeTest` | `internal\extension\checker.go:92` | None | `function` |
| `SaveAPIKey` | `internal\llm\keyring.go:21` | None | `function` |
| `SeedDemoDataForTests` | `internal\db\testhelper.go:7` | `Repository` | `method` |
| `SetupExtensionEnv` | `internal\extension\checker.go:125` | None | `function` |
| `SplitMarkdownIntoChunks` | `internal\notebook\markdown_chunker.go:31` | None | `function` |
| `TransitionTask` | `internal\study\queue_transition.go:46` | `StudyService` | `method` |
| `Uninstall` | `internal\extension\installer.go:15` | `Manager` | `method` |
| `activateReadingSessionTask` | `internal/app/app.go` | `App` | `AppHelper` |
| `aggregateQueueTasks` | `internal\app\app_study.go:52` | None | `function` |
| `appendFailedQuestionsSection` | `internal\app\app_study_cards.go:137` | None | `function` |
| `bookmarkNodesToDraft` | `internal\notebook\pdfcpu.go:43` | None | `function` |
| `buildInputValues` | `internal\embeddings\onnx.go:274` | `OnnxEmbedder` | `method` |
| `buildPageSample` | `internal\notebook\syllabus.go:281` | None | `function` |
| `buildReviewTaskForPlan` | `internal\app\app_study.go:281` | None | `function` |
| `buildSocraticRemedialPrompt` | `internal\app\app_study_cards.go:105` | None | `function` |
| `buildTokenArrays` | `internal\embeddings\onnx.go:223` | None | `function` |
| `calculateDailyStudyMinutes` | `internal\app\app_study.go:20` | None | `function` |
| `calculateFlashcardBudgets` | `internal\app\app_study.go:39` | None | `function` |
| `calculateStreak` | `internal\app\app_study.go:86` | None | `function` |
| `chapterIndexForPage` | `internal\notebook\ingestion.go:91` | None | `function` |
| `checkAndInsertMilestoneExam` | `internal\app\app_study_cards.go:238` | None | `function` |
| `computeCurrentStreak` | `internal\app\app_study.go:142` | None | `function` |
| `computeLongestStreak` | `internal\app\app_study.go:112` | None | `function` |
| `destroyValues` | `internal\embeddings\onnx.go:576` | None | `function` |
| `embedInternal` | `internal\embeddings\onnx.go:180` | `OnnxEmbedder` | `method` |
| `emitIngestionProgress` | `internal\app\notebook_endpoints.go:867` | None | `function` |
| `envHasLLMAPIKey` | `internal\app\app_settings.go:311` | None | `function` |
| `extractEmbedding` | `internal\embeddings\onnx.go:322` | None | `function` |
| `extractIONames` | `internal\embeddings\onnx.go:536` | None | `function` |
| `extractPDFCPUBookmarkDraft` | `internal\notebook\pdfcpu.go:18` | None | `function` |
| `finalizeFastPDFUpload` | `internal/app/app.go` | `App` | `AppHelper` |
| `finalizeNotebookUpload` | `internal/app/app.go` | `App` | `AppHelper` |
| `findPDFCPUExecutable` | `internal\notebook\pdfcpu.go:229` | None | `function` |
| `firstInt` | `internal\notebook\pdfcpu.go:291` | None | `function` |
| `firstN` | `internal\notebook\syllabus.go:316` | None | `function` |
| `firstString` | `internal\notebook\pdfcpu.go:276` | None | `function` |
| `getAppVersion` | `internal\app\app_update.go:17` | None | `function` |
| `getNotebookAndRepo` | `internal/app/app.go` | `App` | `AppHelper` |
| `getStreakState` | `internal/app/app.go` | `App` | `AppHelper` |
| `inferMaxSeqLen` | `internal\embeddings\onnx.go:564` | None | `function` |
| `isMarkdownSection` | `internal\notebook\ingestion.go:14` | None | `function` |
| `isTableRow` | `internal\notebook\markdown_chunker.go:191` | None | `function` |
| `mapTaskError` | `internal\app\app_study.go:165` | None | `function` |
| `maxPage` | `internal\notebook\syllabus.go:308` | None | `function` |
| `meanPool2D` | `internal\embeddings\onnx.go:451` | None | `function` |
| `meanPool2DFloat64` | `internal\embeddings\onnx.go:495` | None | `function` |
| `meanPool3D` | `internal\embeddings\onnx.go:429` | None | `function` |
| `meanPool3DFloat64` | `internal\embeddings\onnx.go:473` | None | `function` |
| `normalizeL2` | `internal\embeddings\onnx.go:517` | None | `function` |
| `normalizeLLMTierForApp` | `internal\app\app_settings.go:301` | None | `function` |
| `parseBookmarkNode` | `internal\notebook\pdfcpu.go:33` | None | `function` |
| `parseMarkdownBlocks` | `internal\notebook\markdown_chunker.go:97` | None | `function` |
| `parseSyllabusDraft` | `internal\notebook\syllabus.go:180` | None | `function` |
| `persistSyllabusDraft` | `internal\app\notebook_endpoints.go:391` | None | `function` |
| `pickInputSource` | `internal\embeddings\onnx.go:544` | None | `function` |
| `poolFloat32Tensor` | `internal\embeddings\onnx.go:350` | None | `function` |
| `poolFloat64Tensor` | `internal\embeddings\onnx.go:387` | None | `function` |
| `queueTaskToScheduledTask` | `internal\app\app_study.go:312` | None | `function` |
| `reconcileConfirmedNotebookTask` | `internal/app/app.go` | `App` | `AppHelper` |
| `reloadLLMProviders` | `internal/app/app.go` | `App` | `AppHelper` |
| `requireRepo` | `internal\app\app_study.go:183` | None | `function` |
| `resolveExplicitActiveProfileID` | `internal/app/app.go` | `App` | `AppHelper` |
| `resolveRetryTopicAndBounds` | `internal\app\app_study_cards.go:553` | None | `function` |
| `resolveRuntimeLibraryPath` | `internal\embeddings\onnx.go:584` | None | `function` |
| `runPDFCPUBookmarksExport` | `internal\notebook\pdfcpu.go:127` | None | `function` |
| `sameLLMSettingsForUI` | `internal\app\app_settings.go:325` | None | `function` |
| `tensorFromInputData` | `internal\embeddings\onnx.go:299` | None | `function` |
| `truncateToCharBoundary` | `internal\notebook\syllabus.go:324` | None | `function` |
| `validatePDFCPUInputFilePath` | `internal\notebook\pdfcpu.go:189` | None | `function` |
| `walkBookmarkNode` | `internal\notebook\pdfcpu.go:54` | None | `function` |

## 4. Mermaid Flowchart (Active Core API)
```mermaid
flowchart TD
    subgraph Frontend
        F_GetTaskContext[GetTaskContext]
        F_activateTask[activateTask]
        F_aiCleanupNotebookSyllabus[aiCleanupNotebookSyllabus]
        F_askReaderAI[askReaderAI]
        F_askSocratic[askSocratic]
        F_assignNotebookToProfile[assignNotebookToProfile]
        F_cancelExtensionSetup[cancelExtensionSetup]
        F_checkExtensionReadiness[checkExtensionReadiness]
        F_checkForUpdates[checkForUpdates]
        F_completeMilestoneExam[completeMilestoneExam]
        F_completeReading[completeReading]
        F_completeReviewSession[completeReviewSession]
        F_completeSocraticRescue[completeSocraticRescue]
        F_confirmNotebookSyllabus[confirmNotebookSyllabus]
        F_createProfile[createProfile]
    end
    subgraph Go_Wails_Bridge
        G_AICleanupNotebookSyllabus[App.AICleanupNotebookSyllabus]
        G_ActivateTask[App.ActivateTask]
        G_AskReaderAI[App.AskReaderAI]
        G_AskSocratic[App.AskSocratic]
        G_AssignNotebookToProfile[App.AssignNotebookToProfile]
        G_CancelExtensionSetup[App.CancelExtensionSetup]
        G_CheckExtensionReadiness[App.CheckExtensionReadiness]
        G_CheckForUpdates[App.CheckForUpdates]
        G_CompleteMilestoneExam[App.CompleteMilestoneExam]
        G_CompleteReading[App.CompleteReading]
        G_CompleteReviewSession[App.CompleteReviewSession]
        G_CompleteSocraticRescue[App.CompleteSocraticRescue]
        G_ConfirmNotebookSyllabus[App.ConfirmNotebookSyllabus]
        G_CreateProfile[App.CreateProfile]
        G_DeleteLLMAPIKey[App.DeleteLLMAPIKey]
    end
        F_GetTaskContext --> G_GetTaskContext
        F_activateTask --> G_ActivateTask
        F_aiCleanupNotebookSyllabus --> G_AICleanupNotebookSyllabus
        F_askReaderAI --> G_AskReaderAI
        F_askSocratic --> G_AskSocratic
        F_assignNotebookToProfile --> G_AssignNotebookToProfile
        F_cancelExtensionSetup --> G_CancelExtensionSetup
        F_checkExtensionReadiness --> G_CheckExtensionReadiness
        F_checkForUpdates --> G_CheckForUpdates
        F_completeMilestoneExam --> G_CompleteMilestoneExam
        F_completeReading --> G_CompleteReading
        F_completeReviewSession --> G_CompleteReviewSession
        F_completeSocraticRescue --> G_CompleteSocraticRescue
        F_confirmNotebookSyllabus --> G_ConfirmNotebookSyllabus
        F_createProfile --> G_CreateProfile
```

*Note: The Mermaid flowchart is capped to the first 15 active endpoints for visual clarity.*