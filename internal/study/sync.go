package study

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-tutor/internal/db"
	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"

	pdfreader "github.com/ledongthuc/pdf"
)

// Production Fallbacks (Can be overridden at build-time using -ldflags)
// Example: go build -ldflags "-X ai-tutor/internal/study.DefaultProductionSyncURL=https://... -X ai-tutor/internal/study.DefaultProductionAnonKey=..."
var (
	DefaultProductionSyncURL        = ""
	DefaultProductionCloudServerURL = "http://localhost:8080"
	DefaultProductionAnonKey        = ""
	DefaultResearchAnalyticsURL     = ""
	DefaultResearchAnalyticsAnonKey = ""
)

// ResolveCloudServerURL returns the effective Cloud Server URL (e.g. Render / Fly.io / localhost).
// Resolution order: CLOUD_SERVER_URL / VITE_API_URL env var → DefaultProductionCloudServerURL.
func ResolveCloudServerURL() string {
	for _, envKey := range []string{"CLOUD_SERVER_URL", "VITE_API_URL"} {
		if env := os.Getenv(envKey); env != "" {
			return strings.TrimSuffix(env, "/")
		}
	}
	return DefaultProductionCloudServerURL
}

// ResolveCloudSyncURL returns the effective sync URL.
// Resolution order: stored SQLite value → CLOUD_SYNC_URL / SUPABASE_URL / VITE_SUPABASE_URL env var → DefaultProductionSyncURL.
func ResolveCloudSyncURL(storedURL string) string {
	if storedURL != "" {
		return storedURL
	}
	if env := os.Getenv("CLOUD_SYNC_URL"); env != "" {
		return env
	}
	for _, envKey := range []string{"SUPABASE_URL", "VITE_SUPABASE_URL"} {
		if env := os.Getenv(envKey); env != "" {
			if !strings.Contains(env, "/rest/v1/rpc/") {
				return fmt.Sprintf("%s/rest/v1/rpc/handle_cloud_sync", strings.TrimSuffix(env, "/"))
			}
			return env
		}
	}
	return DefaultProductionSyncURL
}

// ResolveAnonKey returns the project Supabase anon/publishable API key from environment variables.
func ResolveAnonKey() string {
	for _, envKey := range []string{"CLOUD_API_TOKEN", "SUPABASE_ANON_KEY", "SUPABASE_PUBLISHABLE_KEY", "VITE_SUPABASE_ANON_KEY"} {
		if env := os.Getenv(envKey); env != "" {
			return env
		}
	}
	return DefaultProductionAnonKey
}

// ResolveCloudAPIToken returns the effective user session token / API token.
// Resolution order: stored SQLite value → ResolveAnonKey().
func ResolveCloudAPIToken(storedToken string) string {
	if storedToken != "" {
		return storedToken
	}
	return ResolveAnonKey()
}

// NotebookSyncRecord is the minimal notebook identity the server needs.
// filepath.Base strips the local path — only the filename crosses the wire.
type NotebookSyncRecord struct {
	FileHash             string `json:"file_hash"`
	Filename             string `json:"filename"`
	Title                string `json:"title"`
	StudyStatus          string `json:"study_status"`
	ExternalHelpRequired bool   `json:"external_help_required"` // Red Alert indicator
}

type SyncPayload struct {
	UserToken     string                `json:"p_user_token"`
	ClassroomCode string                `json:"p_classroom_code"`
	Notebooks     []NotebookSyncRecord  `json:"p_notebooks"`
	Logs          []models.SyncLogEntry `json:"p_logs"`
}

type SyncResponse struct {
	NewNotebooks []AssignedNotebook `json:"new_notebooks"`
}

type AssignedNotebook struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	DownloadURL string `json:"download_url"`
	StartPage   *int   `json:"start_page"`
	EndPage     *int   `json:"end_page"`
}

func StartCloudSyncLoop(repo *db.Repository) {
	ticker := time.NewTicker(15 * time.Minute)
	go func() {
		utils.Warnf("[SYNC] Background cloud sync worker started.")
		for range ticker.C {
			if err := TriggerCloudSync(repo); err != nil {
				utils.Warnf("[SYNC] Periodic sync warning: %v", err)
			}
		}
	}()
}

func TriggerCloudSync(repo *db.Repository) error {
	settings, err := repo.GetUserSettings()
	if err != nil {
		return err
	}

	syncURL := ResolveCloudSyncURL(settings.CloudSyncURL)
	userToken := ResolveCloudAPIToken(settings.CloudAPIToken)
	anonKey := ResolveAnonKey()

	if syncURL == "" {
		if syncErr := repo.ResolveFlashcardGenerateTasksForTopic(""); syncErr != nil {
			utils.Warnf("[SYNC] failed to resolve FLASHCARD_GENERATE tasks: %v", syncErr)
		}
		if settings.AnalyticsEnabled {
			if fbErr := syncAnalyticsFallback(repo); fbErr != nil {
				utils.Warnf("[SYNC] fallback analytics upload failed: %v", fbErr)
			}
		}
		return nil // Cloud sync not configured
	}

	utils.Warnf("[SYNC] Running cloud sync to: %s", syncURL)

	// Build slim notebook records — filename only, no local paths or internal IDs
	notebooks, err := repo.GetNotebooks("", "")
	if err != nil {
		return fmt.Errorf("failed to fetch notebooks: %w", err)
	}
	notebookRecords := make([]NotebookSyncRecord, 0, len(notebooks))
	for _, nb := range notebooks {
		if nb.FileHash == "" {
			utils.Warnf("[SYNC] skipping notebook with empty FileHash: title=%q, path=%q", nb.Title, nb.FilePath)
			continue
		}
		notebookRecords = append(notebookRecords, NotebookSyncRecord{
			FileHash:             nb.FileHash,
			Filename:             filepath.Base(nb.FilePath),
			Title:                nb.Title,
			StudyStatus:          nb.StudyStatus,
			ExternalHelpRequired: nb.ExternalHelpRequired,
		})
	}

	// Delta: only logs newer than the last successful sync
	logs, err := repo.GetReviewLogsSinceWithFileInfo(settings.LastSyncedAt)
	if err != nil {
		utils.Warnf("[SYNC] failed to fetch delta review logs: %v", err)
		return err
	}
	utils.Warnf("[SYNC] delta logs to send: %d (since %d)", len(logs), settings.LastSyncedAt)

	payload := SyncPayload{
		UserToken:     userToken,
		ClassroomCode: settings.ClassroomCode,
		Notebooks:     notebookRecords,
		Logs:          logs,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal sync payload: %w", err)
	}

	headers := make(map[string]string)
	if anonKey != "" {
		headers["apikey"] = anonKey
	}
	if userToken != "" && strings.Count(userToken, ".") == 2 {
		headers["Authorization"] = "Bearer " + userToken
	} else if anonKey != "" && strings.Count(anonKey, ".") == 2 {
		headers["Authorization"] = "Bearer " + anonKey
	}

	var syncResp SyncResponse
	lastErr := postJSONWithRetry(syncURL, jsonBytes, headers, 3, &syncResp)
	if lastErr == nil {
		// Handle assigned notebooks from teacher
		if len(syncResp.NewNotebooks) > 0 {
			utils.Warnf("[SYNC] Found %d new teacher assignments", len(syncResp.NewNotebooks))
			for _, assigned := range syncResp.NewNotebooks {
				if err := downloadAndRegisterNotebook(repo, assigned); err != nil {
					utils.Warnf("[SYNC] Failed to download assigned notebook %s: %v", assigned.Title, err)
				}
			}
		}

		// Advance the delta cursor so next sync only sends new events
		maxReviewedAt := settings.LastSyncedAt
		for _, entry := range logs {
			if entry.ReviewedAt > maxReviewedAt {
				maxReviewedAt = entry.ReviewedAt
			}
		}
		if setErr := repo.SetLastSyncedAt(maxReviewedAt); setErr != nil {
			utils.Warnf("[SYNC] failed to persist last_synced_at: %v", setErr)
		}



		// Sync completed successfully. Clear any pending FLASHCARD_GENERATE tasks.
		if syncErr := repo.ResolveFlashcardGenerateTasksForTopic(""); syncErr != nil {
			utils.Warnf("[SYNC] failed to resolve FLASHCARD_GENERATE tasks: %v", syncErr)
		}
	}

	if lastErr != nil {
		utils.Warnf("[SYNC] Cloud sync failed after %d attempts: %v", 3, lastErr)
		// Insert FLASHCARD_GENERATE task if not already pending/active and a valid notebook exists
		if len(notebooks) > 0 {
			notebookID := notebooks[0].ID
			if syncErr := repo.EnsurePendingFlashcardGenerateTask(notebookID, "", 0, 0, "Cloud Sync Recovery"); syncErr != nil {
				utils.Warnf("[SYNC] failed to insert FLASHCARD_GENERATE task: %v", syncErr)
			}
		}
		return lastErr
	}

	utils.Warnf("[SYNC] Cloud sync completed successfully.")
	return nil
}

func syncAnalyticsFallback(repo *db.Repository) error {
	events, ids, err := repo.GetUnsyncedAnalyticsEvents()
	if err != nil {
		return fmt.Errorf("failed to fetch unsynced analytics for fallback: %w", err)
	}
	if len(events) == 0 {
		return nil
	}

	researchURL := os.Getenv("RESEARCH_ANALYTICS_URL")
	if researchURL == "" {
		researchURL = DefaultResearchAnalyticsURL
	}
	researchToken := os.Getenv("RESEARCH_ANALYTICS_ANON_KEY")
	if researchToken == "" {
		researchToken = DefaultResearchAnalyticsAnonKey
	}

	jsonBytes, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal fallback analytics: %w", err)
	}

	headers := map[string]string{
		"apikey":        researchToken,
		"Authorization": "Bearer " + researchToken,
	}

	if err := postJSONWithRetry(researchURL, jsonBytes, headers, 3, nil); err != nil {
		return err
	}

	if err := repo.MarkAnalyticsSynced(ids); err != nil {
		utils.Warnf("[SYNC-ANALYTICS] failed to mark fallback events synced: %v", err)
	}
	utils.Warnf("[SYNC-ANALYTICS] Fallback analytics sync of %d events succeeded.", len(events))
	return nil
}

func postJSONWithRetry(url string, jsonBytes []byte, headers map[string]string, attempts int, decodeTarget interface{}) error {
	client := &http.Client{Timeout: 10 * time.Second}
	var lastErr error

	for i := 0; i < attempts; i++ {
		if i > 0 {
			utils.Warnf("[SYNC-RETRY] Attempt %d/%d due to: %v", i+1, attempts, lastErr)
			time.Sleep(1 * time.Second)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
		if err != nil {
			lastErr = fmt.Errorf("failed to create http request: %w", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("network error: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			bodyBytes, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(bodyBytes))
			continue
		}

		if decodeTarget != nil {
			decodeErr := json.NewDecoder(resp.Body).Decode(decodeTarget)
			_ = resp.Body.Close()
			if decodeErr != nil {
				lastErr = fmt.Errorf("failed to decode response: %w", decodeErr)
				continue
			}
		} else {
			_ = resp.Body.Close()
		}

		return nil
	}
	return lastErr
}

func resolveNotebookUploadDir() (string, error) {
	if os.Getenv("APP_ENV") == "dev" {
		projectRoot, err := os.Getwd()
		if err == nil {
			dir := filepath.Join(projectRoot, "dev_data", "uploads")
			if mkErr := os.MkdirAll(dir, 0o755); mkErr == nil {
				return dir, nil
			}
		}
	}

	var dir string
	if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
		dir = filepath.Join(cfgDir, "Studyloop", "uploads")
	} else if cacheDir, err := os.UserCacheDir(); err == nil && cacheDir != "" {
		dir = filepath.Join(cacheDir, "Studyloop", "uploads")
	} else if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		dir = filepath.Join(homeDir, ".Studyloop", "uploads")
	} else {
		return "", fmt.Errorf("failed to resolve application upload directory")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create app upload directory %s: %w", dir, err)
	}
	return dir, nil
}

func downloadAndRegisterNotebook(repo *db.Repository, nb AssignedNotebook) error {
	utils.Warnf("[SYNC] Processing assigned notebook: title=%q, id=%q, url=%q", nb.Title, nb.ID, nb.DownloadURL)

	uploadDir, err := resolveNotebookUploadDir()
	if err != nil {
		utils.Warnf("[SYNC] Failed to resolve notebook upload directory: %v", err)
		return err
	}
	canonicalPath := filepath.Join(uploadDir, nb.ID+".pdf")

	// 1. Check if already registered by ID
	if existing, _ := repo.GetNotebookByID(nb.ID); existing != nil {
		utils.Warnf("[SYNC] Assignment %q (%s) already exists in database", nb.Title, nb.ID)
		if settings, sErr := repo.GetUserSettings(); sErr == nil && settings.ActiveProfileID != "" && existing.ProfileID != settings.ActiveProfileID {
			if assignErr := repo.AssignNotebookToProfile(nb.ID, settings.ActiveProfileID); assignErr == nil {
				utils.Warnf("[SYNC] Associated existing notebook %s with active profile %s", nb.ID, settings.ActiveProfileID)
			}
		}

		// Verify file exists on disk at stored path; if missing or in wrong path, repair it
		if _, statErr := os.Stat(existing.FilePath); statErr == nil {
			return nil
		}

		// If file exists at canonical path, update DB file_path
		if _, statErr := os.Stat(canonicalPath); statErr == nil {
			utils.Warnf("[SYNC] Updating notebook %s file path to canonical upload dir: %s", nb.ID, canonicalPath)
			_ = repo.UpdateNotebookFilePath(nb.ID, canonicalPath)
			return nil
		}

		// If file is missing altogether, fall through to download to canonical path and update DB file_path
		utils.Warnf("[SYNC] File missing at %s, re-downloading to %s", existing.FilePath, canonicalPath)
		if downloadErr := downloadPDFToFile(nb.DownloadURL, canonicalPath); downloadErr != nil {
			return downloadErr
		}
		_ = repo.UpdateNotebookFilePath(nb.ID, canonicalPath)
		return nil
	}

	// 2. Check if notebook with matching title already exists locally (e.g. uploaded manually by user)
	if allNotebooks, fetchErr := repo.GetNotebooks("", ""); fetchErr == nil {
		for _, existing := range allNotebooks {
			if strings.EqualFold(strings.TrimSpace(existing.Title), strings.TrimSpace(nb.Title)) {
				utils.Warnf("[SYNC] Assignment %q matches existing local notebook %q (%s), linking to active profile", nb.Title, existing.Title, existing.ID)
				if settings, sErr := repo.GetUserSettings(); sErr == nil && settings.ActiveProfileID != "" {
					if assignErr := repo.AssignNotebookToProfile(existing.ID, settings.ActiveProfileID); assignErr == nil {
						utils.Warnf("[SYNC] Linked local notebook %s to active profile %s", existing.ID, settings.ActiveProfileID)
					}
				}
				return nil
			}
		}
	}

	// 3. Download from remote URL to canonical upload directory
	utils.Warnf("[SYNC] Downloading PDF from URL to canonical path %s: %s", canonicalPath, nb.DownloadURL)
	if downloadErr := downloadPDFToFile(nb.DownloadURL, canonicalPath); downloadErr != nil {
		return downloadErr
	}

	// 4. Register in SQLite
	fileHash, hashErr := utils.FileSHA256(canonicalPath)
	if hashErr != nil {
		_ = os.Remove(canonicalPath)
		return fmt.Errorf("failed to compute file hash: %w", hashErr)
	}

	pageCount := 0
	if f, r, pErr := pdfreader.Open(canonicalPath); pErr == nil {
		pageCount = r.NumPage()
		_ = f.Close()
	}

	err = repo.CreateNotebook(nb.ID, nb.Title, canonicalPath, "pdf", "", fileHash, pageCount)
	if err != nil {
		_ = os.Remove(canonicalPath)
		return fmt.Errorf("failed to insert notebook into database: %w", err)
	}

	// Assign to active profile if configured
	if settings, sErr := repo.GetUserSettings(); sErr == nil && settings.ActiveProfileID != "" {
		if assignErr := repo.AssignNotebookToProfile(nb.ID, settings.ActiveProfileID); assignErr != nil {
			utils.Warnf("[SYNC] Warning: failed to assign downloaded notebook to profile %s: %v", settings.ActiveProfileID, assignErr)
		} else {
			utils.Warnf("[SYNC] Assigned notebook %s to active profile %s", nb.ID, settings.ActiveProfileID)
		}
	}

	utils.Warnf("[SYNC] Successfully registered newly assigned notebook: %s (ID: %s, Hash: %s)", nb.Title, nb.ID, fileHash)
	return nil
}

func downloadPDFToFile(downloadURL string, localPath string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create GET request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP GET failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download server returned status %d", resp.StatusCode)
	}

	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local PDF file: %w", err)
	}
	defer func() { _ = out.Close() }()

	const maxDownloadBytes = 100 << 20 // 100 MiB
	if resp.ContentLength > maxDownloadBytes {
		_ = out.Close()
		_ = os.Remove(localPath)
		return fmt.Errorf("download rejected: Content-Length %d exceeds 100 MiB limit", resp.ContentLength)
	}
	limitedBody := &io.LimitedReader{R: resp.Body, N: maxDownloadBytes + 1}
	written, err := io.Copy(out, limitedBody)
	if err != nil {
		_ = out.Close()
		_ = os.Remove(localPath)
		return fmt.Errorf("failed to write PDF data to file: %w", err)
	}
	_ = out.Close()
	if limitedBody.N <= 0 {
		_ = os.Remove(localPath)
		return fmt.Errorf("download aborted: response exceeded 100 MiB limit")
	}

	utils.Warnf("[SYNC] Downloaded %d bytes to %s", written, localPath)
	return nil
}

