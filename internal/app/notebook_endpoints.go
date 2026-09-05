package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-tutor/internal/db"
	"ai-tutor/internal/extension"
	"ai-tutor/internal/models"
	"ai-tutor/internal/notebook"
	"ai-tutor/internal/utils"

	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const ingestionEventName = "ingestion-progress"

type ingestionProgressPayload struct {
	NotebookID   string `json:"notebook_id"`
	TopicID      string `json:"topic_id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	Phase        string `json:"phase"`
	Processed    int    `json:"processed"`
	Total        int    `json:"total"`
	IndexedCount int    `json:"indexed_count"`
	FailedCount  int    `json:"failed_count"`
	Percent      int    `json:"percent"`
}

// UploadNotebook handles file upload and creates notebook record
func (a *App) UploadNotebook(fileData []byte, fileName string) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	if a.notebookService == nil {
		return map[string]interface{}{
			"error": "notebook service not initialized",
		}
	}

	uploadResult, err := a.notebookService.SaveUploadedFile(fileData, fileName)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	return a.finalizeNotebookUpload(uploadResult)
}

// SelectAndUploadDeepStructuredPDF opens a native OS file picker and initiates background deep structured PDF extraction.
// This is zero-copy and instant on desktop without browser DOM file array serialization.
func (a *App) SelectAndUploadDeepStructuredPDF(isPro bool) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	if a.notebookService == nil {
		return map[string]interface{}{"error": "notebook service not initialized"}
	}

	var ext *extension.Extension
	if a.extManager != nil {
		ext, _ = a.extManager.Get("deep_pdf")
		if ext != nil && extension.GetEffectiveTier(ext) == "pro" && !isPro {
			return map[string]interface{}{
				"error":        "Deep Structured PDF Ingestion is a Pro feature. Please upgrade your plan to unlock.",
				"requires_pro": true,
			}
		}
	}

	selectedPath, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select PDF for Deep Structured Analysis",
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "PDF Files (*.pdf)",
				Pattern:     "*.pdf",
			},
		},
	})
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("failed to open file dialog: %v", err)}
	}

	selectedPath = strings.TrimSpace(selectedPath)
	if selectedPath == "" {
		// User canceled the file dialog
		return map[string]interface{}{"canceled": true}
	}

	uploadResult, err := a.notebookService.SaveUploadedFileFromPath(selectedPath)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return a.finalizeDeepStructuredPDFUpload(uploadResult, ext)
}

func (a *App) finalizeDeepStructuredPDFUpload(uploadResult *notebook.UploadResult, ext *extension.Extension) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}

	fileHash, hashErr := utils.FileSHA256(uploadResult.FilePath)
	if hashErr != nil {
		_ = a.notebookService.DeleteFile(uploadResult.FilePath)
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to compute file hash: %v", hashErr),
		}
	}

	profileID := a.resolveExplicitActiveProfileID()
	title := strings.TrimSpace(uploadResult.FileName)
	if title == "" {
		title = filepath.Base(uploadResult.FilePath)
	}

	// Check for duplicate document in the current profile (or global)
	existingNb, findErr := repo.FindNotebookByFileHash(fileHash, profileID)
	if findErr != nil {
		utils.Warnf("finalizeDeepStructuredPDFUpload: error checking duplicate file_hash: %v", findErr)
	} else if existingNb != nil {
		_ = a.notebookService.DeleteFile(uploadResult.FilePath)
		return map[string]interface{}{
			"id":            existingNb.ID,
			"file_name":     existingNb.Title,
			"file_type":     existingNb.FileType,
			"page_count":    existingNb.PageCount,
			"chunk_count":   existingNb.ChunkCount,
			"status":        existingNb.Status,
			"duplicate":     true,
			"existing_id":   existingNb.ID,
			"message":       fmt.Sprintf("Document already exists as '%s'", existingNb.Title),
		}
	}

	// Register notebook immediately in SQLite
	err := repo.CreateNotebook(uploadResult.ID, title, uploadResult.FilePath, uploadResult.FileType, "", fileHash, 0, profileID)
	if err != nil {
		_ = a.notebookService.DeleteFile(uploadResult.FilePath)
		return map[string]interface{}{
			"error": err.Error(),
		}
	}
	_ = repo.UpdateNotebookStatus(uploadResult.ID, "uploaded")

	// Run deep extraction asynchronously in background — zero main thread blocking
	a.runDeepPDFExtraction(uploadResult.ID, uploadResult.FilePath, title, ext, "uploaded", "dormant")

	return map[string]interface{}{
		"id":            uploadResult.ID,
		"file_name":     title,
		"file_type":     uploadResult.FileType,
		"size":          uploadResult.Size,
		"page_count":    0,
		"word_count":    0,
		"chunk_count":   0,
		"indexed_count": 0,
		"failed_count":  0,
		"status":        "processing",
	}
}

func (a *App) runDeepPDFExtraction(nbID, filePath, fileName string, extObj *extension.Extension, prevStatus, prevStudyStatus string) {
	repo := a.getRepo()
	if repo == nil {
		return
	}

	_ = repo.UpdateNotebookStatus(nbID, "processing")
	emitIngestionProgress(a, ingestionProgressPayload{
		NotebookID: nbID,
		Status:     "processing",
		Message:    "Starting Deep Structured extraction...",
		Phase:      "extraction",
		Percent:    10,
	})

	go func() {
		// ponytail: no artificial timeout ceiling; background PDF processing runs until done
		ctx := context.Background()

		onProgress := func(processed, total, percent int, message string) {
			utils.Infof("[DEEP_PDF] %s (%s): %s (%d%%)", fileName, nbID, message, percent)
			emitIngestionProgress(a, ingestionProgressPayload{
				NotebookID: nbID,
				Status:     "processing",
				Message:    message,
				Phase:      "extraction",
				Processed:  processed,
				Total:      total,
				Percent:    percent,
			})
		}

		doc, result, extErr := a.notebookService.IngestDeepPDFWithProgress(ctx, filePath, a.extRunner, extObj, onProgress)
		if extErr != nil {
			utils.Warnf("[DEEP_PDF] Extraction failed for %s (%s): %v", fileName, nbID, extErr)
			fallbackStatus := "failed"
			if nb, err := repo.GetNotebookByID(nbID); err == nil && nb != nil && nb.ChunkCount > 0 {
				fallbackStatus = prevStatus
			}
			_ = repo.UpdateNotebookStatus(nbID, fallbackStatus)
			_ = repo.UpdateNotebookStudyStatus(nbID, prevStudyStatus)
			emitIngestionProgress(a, ingestionProgressPayload{
				NotebookID: nbID,
				Status:     fallbackStatus,
				Message:    fmt.Sprintf("Extraction failed: %v", extErr),
			})
			return
		}

		if err := repo.UpdateNotebookPageCount(nbID, doc.PageCount); err != nil {
			utils.Warnf("[DEEP_PDF] Failed to update page count for %s (%s): %v", fileName, nbID, err)
		}

		chaptersDraft := notebook.ExtractSyllabusChaptersFromMarkdown(result.Markdown, doc.PageCount)
		if len(chaptersDraft) == 0 {
			chaptersDraft = []models.SyllabusChapterDraft{
				{
					Title:     fileName,
					StartPage: 1,
					EndPage:   doc.PageCount,
				},
			}
		}

		if err := persistSyllabusDraft(repo, nbID, doc.PageCount, chaptersDraft, false); err != nil {
			utils.Warnf("[DEEP_PDF] Failed to persist syllabus draft for %s (%s): %v", fileName, nbID, err)
			_ = repo.UpdateNotebookStatus(nbID, prevStatus)
			_ = repo.UpdateNotebookStudyStatus(nbID, prevStudyStatus)
			emitIngestionProgress(a, ingestionProgressPayload{
				NotebookID: nbID,
				Status:     prevStatus,
				Message:    fmt.Sprintf("Failed to save syllabus draft: %v", err),
			})
			return
		}

		if err := repo.UpdateNotebookExtractionEngine(nbID, "deep_structured"); err != nil {
			utils.Warnf("[DEEP_PDF] Failed to update extraction engine for %s (%s): %v", fileName, nbID, err)
		}

		emitIngestionProgress(a, ingestionProgressPayload{
			NotebookID: nbID,
			Status:     "draft_ready",
			Message:    "Syllabus draft ready",
		})
	}()
}

// UploadYouTubeNotebook handles YouTube video URL ingestion, extracting metadata and chapters via the YouTube extension.
func (a *App) UploadYouTubeNotebook(videoURL string, isPro bool) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	if a.notebookService == nil {
		return map[string]interface{}{
			"error": "notebook service not initialized",
		}
	}

	cleanURL := strings.TrimSpace(videoURL)
	if cleanURL == "" {
		return map[string]interface{}{"error": "YouTube URL or video ID is required"}
	}

	var ext *extension.Extension
	if a.extManager != nil {
		ext, _ = a.extManager.Get("youtube")
		if ext != nil && extension.GetEffectiveTier(ext) == "pro" && !isPro {
			return map[string]interface{}{
				"error":        "YouTube Ingestion is a Pro feature. Please upgrade your plan to unlock.",
				"requires_pro": true,
			}
		}
	}

	var downloadQuality string
	var autoDownload bool
	if extJSON, err := repo.GetExtensionConfig(); err == nil && extJSON != "" && extJSON != "{}" {
		var extCfg map[string]map[string]interface{}
		if err := json.Unmarshal([]byte(extJSON), &extCfg); err == nil {
			if ytCfg, ok := extCfg["youtube"]; ok {
				if ad, ok := ytCfg["auto_download"].(bool); ok {
					autoDownload = ad
				}
				if dq, ok := ytCfg["download_quality"].(string); ok {
					downloadQuality = dq
				}
			}
		}
	}
	utils.Infof("[YOUTUBE_INGEST] Settings resolved -> auto_download: %v, download_quality: %q", autoDownload, downloadQuality)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	doc, result, err := a.notebookService.IngestYouTubeVideo(ctx, cleanURL, a.extRunner, ext)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to ingest YouTube video: %v", err),
		}
	}

	notebookID := uuid.New().String()
	jsonFileName := fmt.Sprintf("%s.json", notebookID)
	jsonFilePath := filepath.Join(a.notebookUploadDir, jsonFileName)

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to serialize video metadata: %v", err),
		}
	}

	if err := os.WriteFile(jsonFilePath, jsonBytes, 0o644); err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to save video metadata: %v", err),
		}
	}

	fileHash := utils.MD5Hex(result.VideoID)
	profileID := a.resolveExplicitActiveProfileID()
	title := strings.TrimSpace(result.Title)
	if title == "" {
		title = "YouTube Video"
	}

	err = repo.CreateNotebook(notebookID, title, jsonFilePath, "youtube", "", fileHash, len(result.Chapters), profileID)
	if err != nil {
		_ = os.Remove(jsonFilePath)
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	_ = repo.UpdateNotebookStatus(notebookID, "uploaded")

	// Pre-create syllabus draft from chapters (1-based page numbers)
	chaptersDraft := make([]models.SyllabusChapterDraft, 0, len(result.Chapters))
	for i, ch := range result.Chapters {
		pageNum := i + 1
		chaptersDraft = append(chaptersDraft, models.SyllabusChapterDraft{
			Title:     ch.Title,
			StartPage: pageNum,
			EndPage:   pageNum,
		})
	}
	if err := persistSyllabusDraft(repo, notebookID, len(result.Chapters), chaptersDraft, false); err != nil {
		_ = os.Remove(jsonFilePath)
		_ = repo.DeleteNotebook(notebookID)
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to save syllabus draft: %v", err),
		}
	}

	// Trigger non-blocking background video download if enabled
	if autoDownload {
		videoDir := filepath.Join(a.notebookUploadDir, "videos")
		videoFilePath := filepath.Join(videoDir, fmt.Sprintf("%s.mp4", notebookID))
		if downloadQuality == "" {
			downloadQuality = "720p"
		}
		go func(vURL, outPath, qual string) {
			dlCtx, dlCancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer dlCancel()
			utils.Infof("[YOUTUBE_CACHE] Starting background video download for %s (%s)...", notebookID, vURL)
			if err := a.notebookService.DownloadYouTubeVideo(dlCtx, vURL, outPath, qual, a.extRunner, ext); err != nil {
				utils.Warnf("[YOUTUBE_CACHE] Background video download failed for %s: %v", notebookID, err)
			} else {
				utils.Infof("[YOUTUBE_CACHE] Video successfully cached at %s", outPath)
			}
		}(cleanURL, videoFilePath, downloadQuality)
	}

	return map[string]interface{}{
		"id":            notebookID,
		"file_name":     title,
		"file_type":     "youtube",
		"page_count":    len(result.Chapters),
		"word_count":    doc.WordCount,
		"chunk_count":   0,
		"indexed_count": 0,
		"failed_count":  0,
		"status":        "uploaded",
		"video_id":      result.VideoID,
		"uploader":      result.Uploader,
		"duration":      result.DurationSeconds,
	}
}

func (a *App) finalizeNotebookUpload(uploadResult *notebook.UploadResult) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	if uploadResult == nil {
		return map[string]interface{}{
			"error": "upload failed",
		}
	}

	// Extract lightweight metadata for page count and size details during initial upload.
	meta, err := a.notebookService.ExtractMetadata(uploadResult.FilePath, uploadResult.FileType)
	if err != nil {
		_ = a.notebookService.DeleteFile(uploadResult.FilePath)
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	// Compute file hash for cloud sync identification.
	fileHash, hashErr := utils.FileSHA256(uploadResult.FilePath)
	if hashErr != nil {
		_ = a.notebookService.DeleteFile(uploadResult.FilePath)
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to compute file hash: %v", hashErr),
		}
	}

	profileID := a.resolveExplicitActiveProfileID()
	// Check for duplicate document in the current profile (or global)
	existingNb, findErr := repo.FindNotebookByFileHash(fileHash, profileID)
	if findErr != nil {
		utils.Warnf("finalizeNotebookUpload: error checking duplicate file_hash: %v", findErr)
	} else if existingNb != nil {
		_ = a.notebookService.DeleteFile(uploadResult.FilePath)
		return map[string]interface{}{
			"id":            existingNb.ID,
			"file_name":     existingNb.Title,
			"file_type":     existingNb.FileType,
			"page_count":    existingNb.PageCount,
			"chunk_count":   existingNb.ChunkCount,
			"status":        existingNb.Status,
			"duplicate":     true,
			"existing_id":   existingNb.ID,
			"message":       fmt.Sprintf("Document already exists as '%s'", existingNb.Title),
		}
	}

	// Create notebook record as unlinked; Sprint 11 uses a draft/confirm ingestion flow.
	err = repo.CreateNotebook(uploadResult.ID, uploadResult.FileName, uploadResult.FilePath, uploadResult.FileType, "", fileHash, meta.PageCount, profileID)
	if err != nil {
		_ = a.notebookService.DeleteFile(uploadResult.FilePath)
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	status := "uploaded"
	_ = repo.UpdateNotebookStatus(uploadResult.ID, status)

	return map[string]interface{}{
		"id":            uploadResult.ID,
		"file_name":     uploadResult.FileName,
		"file_type":     uploadResult.FileType,
		"size":          uploadResult.Size,
		"page_count":    meta.PageCount,
		"word_count":    meta.WordCount,
		"chunk_count":   0,
		"indexed_count": 0,
		"failed_count":  0,
		"status":        status,
	}
}

// resolveExplicitActiveProfileID returns the active profile ID from user settings
// only when an explicit active profile has been set — no fallback to oldest profile.
func (a *App) resolveExplicitActiveProfileID() string {
	repo := a.getRepo()
	if repo == nil {
		return ""
	}
	s, err := repo.GetUserSettings()
	if err == nil && s != nil && s.ActiveProfileID != "" {
		return s.ActiveProfileID
	}
	return ""
}

// DraftNotebookSyllabus creates editable chapter ranges for HITL verification.
func (a *App) getNotebookAndRepo(notebookID string) (*db.Repository, *models.Notebook, map[string]interface{}) {
	repo := a.getRepo()
	if repo == nil {
		return nil, nil, map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	notebookID = strings.TrimSpace(notebookID)
	if notebookID == "" {
		return nil, nil, map[string]interface{}{"error": "notebook id is required"}
	}
	if a.notebookService == nil {
		return nil, nil, map[string]interface{}{"error": "notebook service not initialized"}
	}

	nb, err := repo.GetNotebookByID(notebookID)
	if err != nil {
		return nil, nil, map[string]interface{}{"error": err.Error()}
	}
	if nb == nil {
		return nil, nil, map[string]interface{}{"error": "notebook not found"}
	}
	return repo, nb, nil
}

func persistSyllabusDraft(repo *db.Repository, notebookID string, pageCount int, chapters []models.SyllabusChapterDraft, fallbackUsed bool) error {
	draftJSON, err := json.Marshal(models.SyllabusDraft{PageCount: pageCount, Chapters: chapters, FallbackUsed: fallbackUsed})
	if err != nil {
		return fmt.Errorf("failed to marshal draft: %w", err)
	}
	if err := repo.UpdateNotebookSyllabusDraft(notebookID, string(draftJSON)); err != nil {
		return fmt.Errorf("failed to persist syllabus draft: %w", err)
	}
	if err := repo.UpdateNotebookStatus(notebookID, "draft_ready"); err != nil {
		return fmt.Errorf("failed to update status to draft_ready: %w", err)
	}
	return nil
}

// DraftNotebookSyllabus creates editable chapter ranges for HITL verification.
// Uses bookmark extraction only (no LLM) for fast default response.
// If regenerate=true, runs full extraction+LLM (same as AICleanupNotebookSyllabus).
// If regenerate=false and a draft exists in DB, returns the persisted draft.
func (a *App) DraftNotebookSyllabus(notebookID string, regenerate bool) map[string]interface{} {
	repo, nb, errResp := a.getNotebookAndRepo(notebookID)
	if errResp != nil {
		return errResp
	}

	var segments interface{}
	if strings.EqualFold(strings.TrimSpace(nb.FileType), "youtube") {
		if segList, err := a.notebookService.GetYouTubeSegmentTimestamps(nb.FilePath); err == nil {
			segments = segList
		}
	}

	// Try to load persisted draft if not regenerating
	if !regenerate {
		draftJSON, err := repo.GetNotebookSyllabusDraft(notebookID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		if draftJSON != "" {
			var persistedDraft models.SyllabusDraft
			if err := json.Unmarshal([]byte(draftJSON), &persistedDraft); err == nil {
				resp := map[string]interface{}{
					"notebook_id":   notebookID,
					"page_count":    persistedDraft.PageCount,
					"chapters":      persistedDraft.Chapters,
					"status":        "draft_ready",
					"fallback_used": persistedDraft.FallbackUsed,
				}
				if segments != nil {
					resp["segments"] = segments
				}
				return resp
			}
		}
	}

	// Extract lightweight document sample for page count
	doc, err := a.notebookService.ExtractDocumentSample(nb.FilePath, nb.FileType, 30)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	if doc.PageCount <= 0 {
		doc.PageCount = 1
	}

	if err := repo.UpdateNotebookStatus(notebookID, "analyzing"); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("failed to update notebook status: %v", err)}
	}

	if !regenerate {
		var chapters []models.SyllabusChapterDraft
		fallbackUsed := false

		if res, err := a.notebookService.DraftSyllabusChapters(nb.FileType, nb.FilePath, doc, nil); err == nil && len(res.Chapters) > 0 {
			chapters = res.Chapters
			fallbackUsed = res.FallbackUsed
		}

		if len(chapters) == 0 {
			fallbackUsed = true
			title := strings.TrimSpace(nb.Title)
			if title == "" {
				title = "General"
			}
			chapters = []models.SyllabusChapterDraft{{Title: title, StartPage: 1, EndPage: doc.PageCount}}
		}

		if err := persistSyllabusDraft(repo, notebookID, doc.PageCount, chapters, fallbackUsed); err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		resp := map[string]interface{}{
			"notebook_id":   notebookID,
			"page_count":    doc.PageCount,
			"chapters":      chapters,
			"status":        "draft_ready",
			"fallback_used": fallbackUsed,
		}
		if segments != nil {
			resp["segments"] = segments
		}
		return resp
	}

	// regenerate=true: full extraction + LLM (used by AI Clean Up)
	// Stop and return error if LLM is unavailable or draft generation fails.
	if a.heavyLLMProvider == nil {
		return map[string]interface{}{"error": "heavy LLM provider is not available for AI cleanup"}
	}

	result, llmErr := a.notebookService.DraftSyllabusChapters(nb.FileType, nb.FilePath, doc, a.heavyLLMProvider)
	if llmErr != nil {
		return map[string]interface{}{"error": fmt.Sprintf("AI extraction failed: %v", llmErr)}
	}
	if len(result.Chapters) == 0 {
		return map[string]interface{}{"error": "AI extraction returned no chapters"}
	}

	if err := persistSyllabusDraft(repo, notebookID, doc.PageCount, result.Chapters, result.FallbackUsed); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	resp := map[string]interface{}{
		"notebook_id":   notebookID,
		"page_count":    doc.PageCount,
		"chapters":      result.Chapters,
		"status":        "draft_ready",
		"fallback_used": result.FallbackUsed,
	}
	if segments != nil {
		resp["segments"] = segments
	}
	return resp
}

// AICleanupNotebookSyllabus re-runs chapter extraction with LLM to improve bookmark-based drafts.
// Explicit user action — not called automatically.
func (a *App) AICleanupNotebookSyllabus(notebookID string) map[string]interface{} {
	return a.DraftNotebookSyllabus(notebookID, true)
}

// ConfirmNotebookSyllabus commits notebook ingestion from user-confirmed chapter bounds.
func (a *App) ConfirmNotebookSyllabus(notebookID string, chapters []models.SyllabusChapterDraft) map[string]interface{} {
	repo, nb, errResp := a.getNotebookAndRepo(notebookID)
	if errResp != nil {
		return errResp
	}

	notebookID = strings.TrimSpace(notebookID)

	// Extract document only when a full re-ingest is necessary. We'll try to detect
	// whether a metadata-only or topic-metadata-only update is sufficient.
	normalized := notebook.NormalizeSyllabusChapters(chapters, nb.PageCount)
	if len(normalized) == 0 {
		return map[string]interface{}{"error": "at least one valid chapter is required"}
	}

	// Attempt to fetch existing topics/bounds for this notebook to decide path
	existingTopics, etErr := repo.GetNotebookTopicsWithBounds(notebookID)
	existingTopicIDs := make(map[string]struct{}, len(existingTopics))
	for _, et := range existingTopics {
		existingTopicIDs[et.TopicID] = struct{}{}
	}
	if etErr != nil {
		// Log but continue with conservative full re-ingest flow
		utils.Warnf("ConfirmNotebookSyllabus: unable to load existing topics for %s: %v", notebookID, etErr)
	}

	// If notebook already chunked and we have existing topic info, compare bounds/titles
	if nb.Status == "chunked" && len(existingTopics) > 0 {
		boundsChanged := false
		titlesChanged := false

		if len(existingTopics) != len(normalized) {
			boundsChanged = true
		} else {
			for i := range normalized {
				if existingTopics[i].StartPage != normalized[i].StartPage || existingTopics[i].EndPage != normalized[i].EndPage {
					boundsChanged = true
					break
				}
				if strings.TrimSpace(existingTopics[i].Title) != strings.TrimSpace(normalized[i].Title) {
					titlesChanged = true
				}
			}
		}

		if !boundsChanged && !titlesChanged {
			// Nothing changed (no chapter or title changes) — treat as metadata_only/no-op
			utils.Infof("ConfirmNotebookSyllabus: metadata_only (no chapter/title changes) for %s", notebookID)
			a.reconcileConfirmedNotebookTask(repo, notebookID, nb.ProfileID, nb.StudyStatus)
			return map[string]interface{}{
				"success":     true,
				"status":      nb.Status,
				"notebook_id": notebookID,
				"mode":        "metadata_only",
			}
		}

		if !boundsChanged && titlesChanged {
			// Only titles changed — update topic titles in-place and preserve chunks/vectors
			utils.Infof("ConfirmNotebookSyllabus: topic_metadata_only for %s — updating topic titles only", notebookID)

			topicItems := make([]db.TopicBatchItem, 0, len(existingTopics))
			topicIDs := make([]string, 0, len(existingTopics))
			for i, et := range existingTopics {
				topicItems = append(topicItems, db.TopicBatchItem{TopicID: et.TopicID, Title: normalized[i].Title})
				topicIDs = append(topicIDs, et.TopicID)
			}

			if err := repo.EnsureTopicsBatch(topicItems); err != nil {
				_ = repo.UpdateNotebookStatus(notebookID, "failed")
				return map[string]interface{}{"error": "failed to update topics: " + err.Error()}
			}

			if len(topicIDs) > 0 {
				_ = repo.UpdateNotebookTopic(notebookID, topicIDs[0])
			}

			a.reconcileConfirmedNotebookTask(repo, notebookID, nb.ProfileID, nb.StudyStatus)

			// Return without running extraction/ingestion or embedding updates
			return map[string]interface{}{
				"success":     true,
				"status":      nb.Status,
				"notebook_id": notebookID,
				"mode":        "topic_metadata_only",
				"topic_ids":   topicIDs,
			}
		}
		// If boundsChanged==true fall through to full re-ingest
	}

	// Full re-ingest path (extract document and rebuild chunks)
	doc, err := a.notebookService.ExtractDocument(nb.FilePath, nb.FileType)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	// Re-normalize with real page count from document
	normalized = notebook.NormalizeSyllabusChapters(chapters, doc.PageCount)
	if len(normalized) == 0 {
		return map[string]interface{}{"error": "at least one valid chapter is required"}
	}

	// Collect all topics and bounds for batch processing
	topicItems := make([]db.TopicBatchItem, 0, len(normalized))
	boundsItems := make([]db.TopicPageBoundsBatchItem, 0, len(normalized))
	topicIDs := make([]string, 0, len(normalized))

	for i, ch := range normalized {
		chTitle := strings.TrimSpace(ch.Title)
		if chTitle == "" {
			chTitle = fmt.Sprintf("Chapter %d", i+1)
		}
		// Sanitize topic ID: lowercase, replace non-alphanumerics with hyphens, collapse duplicates
		sanitized := strings.ToLower(chTitle)
		// Replace any character not in [a-z0-9] with hyphen
		var result []rune
		for _, r := range sanitized {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				result = append(result, r)
			} else {
				result = append(result, '-')
			}
		}
		sanitized = string(result)
		// Collapse duplicate hyphens
		for strings.Contains(sanitized, "--") {
			sanitized = strings.ReplaceAll(sanitized, "--", "-")
		}
		// Trim leading/trailing hyphens
		sanitized = strings.Trim(sanitized, "-")
		// Fallback if empty
		if sanitized == "" {
			sanitized = "topic"
		}
		// Limit length
		if len(sanitized) > 20 {
			sanitized = sanitized[:20]
		}
		topicID := fmt.Sprintf("nb-%s-ch-%02d-%s", notebookID, i+1, sanitized)
		topicIDs = append(topicIDs, topicID)

		topicItems = append(topicItems, db.TopicBatchItem{
			TopicID: topicID,
			Title:   chTitle,
		})

		boundsItems = append(boundsItems, db.TopicPageBoundsBatchItem{
			TopicID:   topicID,
			StartPage: ch.StartPage,
			EndPage:   ch.EndPage,
		})
	}

	// Batch create/update topics
	if err := repo.EnsureTopicsBatch(topicItems); err != nil {
		_ = repo.UpdateNotebookStatus(notebookID, "failed")
		return map[string]interface{}{"error": "failed to create topics: " + err.Error()}
	}

	// Batch update page bounds
	if err := repo.UpdateTopicPageBoundsBatch(boundsItems); err != nil {
		_ = repo.UpdateNotebookStatus(notebookID, "failed")
		// Cleanup only topics provably created in this request; skip cleanup if existing-topic lookup failed.
		if etErr == nil {
			var toDelete []string
			for _, item := range topicItems {
				if _, existed := existingTopicIDs[item.TopicID]; !existed {
					toDelete = append(toDelete, item.TopicID)
				}
			}
			_ = repo.DeleteTopics(toDelete)
		}
		return map[string]interface{}{"error": "failed to persist topic bounds: " + err.Error()}
	}

	if len(topicIDs) > 0 {
		_ = repo.UpdateNotebookTopic(notebookID, topicIDs[0])
	}

	// Track which topic IDs were newly created for cleanup
	var newlyCreatedTopicIDs []string
	if etErr == nil {
		for _, item := range topicItems {
			if _, existed := existingTopicIDs[item.TopicID]; !existed {
				newlyCreatedTopicIDs = append(newlyCreatedTopicIDs, item.TopicID)
			}
		}
	}

	groups, allChunks := notebook.BuildTopicGroupsFromChapters(notebookID, doc, topicIDs, normalized)
	if len(groups) == 0 || len(allChunks) == 0 {
		_ = repo.UpdateNotebookStatus(notebookID, "failed")
		// Cleanup: delete only newly created topic rows to avoid orphaned records
		_ = repo.DeleteTopics(newlyCreatedTopicIDs)
		return map[string]interface{}{"error": "confirmed chapters produced no chunks"}
	}

	utils.Infof("ConfirmNotebookSyllabus: full_reingest for %s — creating %d chunks", notebookID, len(allChunks))

	emitIngestionProgress(a, ingestionProgressPayload{
		NotebookID: notebookID,
		Status:     "chunking",
		Message:    fmt.Sprintf("Creating %d chunks for confirmed chapters", len(allChunks)),
		Phase:      "chunking",
		Processed:  0,
		Total:      len(allChunks),
		Percent:    20,
	})

	if err := repo.IngestNotebookContentByTopic(notebookID, groups); err != nil {
		_ = repo.UpdateNotebookStatus(notebookID, "failed")
		// Cleanup: delete only newly created topic rows to avoid orphaned records
		_ = repo.DeleteTopics(newlyCreatedTopicIDs)
		emitIngestionProgress(a, ingestionProgressPayload{
			NotebookID: notebookID,
			Status:     "failed",
			Message:    "Chunk ingestion failed",
			Phase:      "chunking",
			Processed:  0,
			Total:      len(allChunks),
			Percent:    100,
		})
		return map[string]interface{}{"error": "chunk ingestion failed: " + err.Error()}
	}

	// Link new topics to notebook in database
	if err := repo.LinkNotebookTopics(notebookID, topicIDs); err != nil {
		_ = repo.UpdateNotebookStatus(notebookID, "failed")
		// Cleanup: delete newly created topic rows (cascades to chunks, cards, etc.) to avoid orphaned records
		_ = repo.DeleteTopics(newlyCreatedTopicIDs)
		return map[string]interface{}{"error": "failed to link notebook topics: " + err.Error()}
	}

	// Delete old orphaned topics that are no longer part of the new syllabus
	if etErr == nil {
		newTopicIDsMap := make(map[string]bool)
		for _, tid := range topicIDs {
			newTopicIDsMap[tid] = true
		}
		var orphanedTopicIDs []string
		for _, et := range existingTopics {
			if !newTopicIDsMap[et.TopicID] {
				orphanedTopicIDs = append(orphanedTopicIDs, et.TopicID)
			}
		}
		_ = repo.DeleteTopics(orphanedTopicIDs)
	}

	status := "chunked"
	emitIngestionProgress(a, ingestionProgressPayload{
		NotebookID: notebookID,
		Status:     status,
		Message:    "Chunking complete",
		Phase:      "complete",
		Processed:  len(allChunks),
		Total:      len(allChunks),
		Percent:    100,
	})

	_ = repo.UpdateNotebookStatus(notebookID, status)

	a.reconcileConfirmedNotebookTask(repo, notebookID, nb.ProfileID, nb.StudyStatus)

	ragEnabled, err := repo.GetRAGEnabled()
	if err == nil && ragEnabled && a.indexQueue != nil {
		a.indexQueue.Enqueue(notebookID)
	}

	return map[string]interface{}{
		"success":     true,
		"status":      status,
		"notebook_id": notebookID,
		"mode":        "full_reingest",
		"topic_ids":   topicIDs,
		"chunk_count": len(allChunks),
	}
}

func (a *App) reconcileConfirmedNotebookTask(repo *db.Repository, notebookID, profileID, currentStudyStatus string) {
	isActivated := currentStudyStatus == "active"
	// Auto-activate the notebook if the active profile currently has less than 4 active notebooks
	if currentStudyStatus == "dormant" || currentStudyStatus == "" {
		activeCount, err := repo.CountActiveNotebooksForActiveProfile(profileID)
		if err != nil {
			utils.Warnf("[INGESTION] failed to count active notebooks for profile %s: %v", profileID, err)
		} else if activeCount < 4 {
			if err := repo.UpdateNotebookStudyStatus(notebookID, "active"); err != nil {
				utils.Warnf("[INGESTION] failed to auto-activate notebook %s: %v", notebookID, err)
			} else {
				isActivated = true
			}
		}
	}

	// Seed initial READING task into study_queue if active
	if isActivated {
		settings, err := repo.GetUserSettings()
		targetWords := 1500
		if err != nil {
			utils.Warnf("[INGESTION] failed to load user settings for notebook %s, using default: %v", notebookID, err)
		} else if settings != nil && settings.TargetSessionWords > 0 {
			targetWords = settings.TargetSessionWords
		}

		if err := repo.EnsurePendingReadingTaskForNotebook(notebookID, targetWords); err != nil {
			utils.Warnf("[INGESTION] failed to ensure initial reading task for %s: %v", notebookID, err)
		}
	}
}

// GetNotebooks retrieves all notebooks, optionally filtered by topic and profile.
// When profileID is empty, returns all notebooks (backward compatible).
// When profileID is set, returns only notebooks belonging to that profile or unassigned notebooks.
func (a *App) GetNotebooks(topicID, profileID string) []map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return []map[string]interface{}{{"error": errDatabaseNotInitialized}}
	}
	notebooks, err := repo.GetNotebooks(topicID, profileID)
	if err != nil {
		return []map[string]interface{}{
			{"error": err.Error()},
		}
	}

	var result []map[string]interface{}
	for _, nb := range notebooks {
		result = append(result, map[string]interface{}{
			"id":              nb.ID,
			"title":           nb.Title,
			"file_type":       nb.FileType,
			"topic_id":        nb.TopicID,
			"status":          nb.Status,
			"indexing_status": nb.IndexingStatus,
			"page_count":      nb.PageCount,
			"chunk_count":     nb.ChunkCount,
			"priority":        nb.Priority,
			"exam_deadline":   nb.ExamDeadline,
			"uploaded_at":     nb.UploadedAt,
			"profile_id":      nb.ProfileID,
			"study_status":    nb.StudyStatus,
			"start_page":      nb.StartPage,
			"end_page":        nb.EndPage,
		})
	}

	return result
}

// GetNotebookTopicTree returns notebook-scoped topic options for hierarchical selectors.
func (a *App) GetNotebookTopicTree() ([]models.NotebookTopicTreeNode, error) {
	repo := a.getRepo()
	if repo == nil {
		return nil, errors.New(errDatabaseNotInitialized)
	}
	profileID := a.resolveExplicitActiveProfileID()
	tree, err := repo.GetNotebookTopicTree(profileID)
	if err != nil {
		return nil, err
	}

	return tree, nil
}

func emitIngestionProgress(a *App, payload ingestionProgressPayload) {
	if a == nil || a.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(a.ctx, ingestionEventName, payload)
}

// UpdateNotebookTitle updates notebook metadata for user edits before re-ingestion.
func (a *App) UpdateNotebookTitle(notebookID string, title string) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	notebookID = strings.TrimSpace(notebookID)
	title = strings.TrimSpace(title)
	if notebookID == "" {
		return map[string]interface{}{"error": "notebook id is required"}
	}
	if title == "" {
		return map[string]interface{}{"error": "title is required"}
	}

	if err := repo.UpdateNotebookTitle(notebookID, title); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{"success": true}
}

// UpdateNotebookPriority updates the notebook priority level (1-10).
func (a *App) UpdateNotebookPriority(notebookID string, priority int) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	notebookID = strings.TrimSpace(notebookID)
	if notebookID == "" {
		return map[string]interface{}{"error": "notebook id is required"}
	}
	// Clamp priority to valid range 1-10
	if priority < 1 {
		priority = 1
	}
	if priority > 10 {
		priority = 10
	}

	if err := repo.UpdateNotebookPriority(notebookID, priority); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{"success": true}
}

// DeleteNotebook removes a notebook and its associated file
func (a *App) DeleteNotebook(notebookID string) map[string]interface{} {
	notebookID = strings.TrimSpace(notebookID)
	repo, nb, errResp := a.getNotebookAndRepo(notebookID)
	if errResp != nil {
		return errResp
	}

	// 1. Delete associated physical file from disk first
	if nb.FilePath != "" {
		if err := a.notebookService.DeleteFile(nb.FilePath); err != nil && !os.IsNotExist(err) {
			return map[string]interface{}{
				"error": fmt.Sprintf("failed to delete notebook file %s: %v", nb.FilePath, err),
			}
		}
	}

	// 2. Delete database record and all associated chunks/topics/tasks
	if err := repo.DeleteNotebook(notebookID); err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	return map[string]interface{}{
		"success": true,
	}
}

// GetProfileDailyPace calculates and returns the daily study pace to meet the profile deadline.
func (a *App) GetProfileDailyPace(profileID string) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return map[string]interface{}{"error": "profile id is required"}
	}

	p, err := repo.GetProfileByID(profileID)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if p == nil {
		return map[string]interface{}{"error": "profile not found"}
	}

	remainingWords, err := repo.GetProfileRemainingWords(profileID)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	if p.DeadlineAt <= 0 {
		return map[string]interface{}{
			"has_deadline":     false,
			"deadline":         "",
			"daily_pace":       0,
			"remaining_words":  remainingWords,
			"days_remaining":   0,
			"sessions_per_day": 0,
			"pace_label":       "",
		}
	}

	deadlineTime := time.Unix(p.DeadlineAt, 0)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	deadlineDate := time.Date(deadlineTime.Year(), deadlineTime.Month(), deadlineTime.Day(), 0, 0, 0, 0, now.Location())

	duration := deadlineDate.Sub(today)
	daysRemaining := int(math.Round(duration.Hours() / 24))

	var dailyPace int
	if daysRemaining > 0 {
		dailyPace = int(math.Ceil(float64(remainingWords) / float64(daysRemaining)))
	} else {
		dailyPace = remainingWords
	}

	settings, err := repo.GetUserSettings()
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("failed to load user settings: %v", err)}
	}
	if settings == nil || settings.TargetSessionWords <= 0 {
		return map[string]interface{}{"error": "invalid target_session_words in user settings"}
	}
	targetWords := settings.TargetSessionWords

	sessionsPerDay := 0.0
	if dailyPace > 0 {
		sessionsPerDay = float64(dailyPace) / float64(targetWords)
	}

	n := int(math.Ceil(sessionsPerDay))
	paceLabel := ""
	if n > 0 {
		if n == 1 {
			paceLabel = "On track — 1 session/day"
		} else if n <= 2 {
			paceLabel = "Moderate pace"
		} else if n <= 4 {
			paceLabel = "Tight schedule"
		} else {
			paceLabel = "Consider adding more books or extending deadline"
		}
	}

	return map[string]interface{}{
		"has_deadline":     true,
		"deadline":         deadlineTime.Format(dateFormatYYYYMMDD),
		"daily_pace":       dailyPace,
		"remaining_words":  remainingWords,
		"days_remaining":   daysRemaining,
		"sessions_per_day": sessionsPerDay,
		"pace_label":       paceLabel,
	}
}

// UpgradeNotebookToDeepPDF re-extracts an existing PDF using the deep_pdf (PyMuPDF) engine.
func (a *App) UpgradeNotebookToDeepPDF(notebookID string) map[string]interface{} {
	_, nb, errResp := a.getNotebookAndRepo(notebookID)
	if errResp != nil {
		return errResp
	}

	if !strings.EqualFold(nb.FileType, "pdf") {
		return map[string]interface{}{"error": "only PDF documents can be upgraded with Deep PDF"}
	}

	var ext *extension.Extension
	if a.extManager != nil {
		ext, _ = a.extManager.Get("deep_pdf")
	}

	repo := a.getRepo()
	if repo != nil {
		if err := repo.UpdateNotebookStudyStatus(nb.ID, "dormant"); err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("failed to update notebook study status: %v", err)}
		}
	}

	// Delegate to unified extraction worker
	a.runDeepPDFExtraction(nb.ID, nb.FilePath, nb.Title, ext, nb.Status, nb.StudyStatus)

	return map[string]interface{}{
		"success":     true,
		"notebook_id": nb.ID,
		"status":      "processing",
	}
}

