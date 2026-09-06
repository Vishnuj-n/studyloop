package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-tutor/internal/extension"
	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"

	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type ankiExtensionOutput struct {
	DeckName   string `json:"deck_name"`
	TotalCards int    `json:"total_cards"`
	Cards      []struct {
		Prompt string `json:"prompt"`
		Answer string `json:"answer"`
	} `json:"cards"`
	Error string `json:"error,omitempty"`
}

// SelectAnkiFile opens a native desktop file picker for .apkg / .colpkg files.
func (a *App) SelectAnkiFile() map[string]interface{} {
	selectedPath, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select Anki Deck Package",
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "Anki Packages (*.apkg, *.colpkg)",
				Pattern:     "*.apkg;*.colpkg",
			},
		},
	})
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("failed to open file dialog: %v", err)}
	}

	selectedPath = strings.TrimSpace(selectedPath)
	if selectedPath == "" {
		return map[string]interface{}{"canceled": true}
	}

	return map[string]interface{}{
		"file_path": selectedPath,
		"file_name": filepath.Base(selectedPath),
	}
}

// ImportAnkiDeck imports an Anki deck (.apkg / .colpkg) either as a standalone notebook or into an existing notebook/topic.
func (a *App) ImportAnkiDeck(filePath string, targetNotebookID string, targetTopicID string) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	if a.extManager == nil || a.extRunner == nil {
		return map[string]interface{}{"error": "extension runtime not initialized"}
	}

	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return map[string]interface{}{"error": "file path is required"}
	}
	if _, err := os.Stat(filePath); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("file not found: %s", filePath)}
	}

	ext, ok := a.extManager.Get("anki_importer")
	if !ok {
		return map[string]interface{}{"error": "anki_importer extension not found. Please enable it in Settings."}
	}

	venvDir := extension.ResolveExtensionVenvDir(ext)
	pyExe := extension.GetVenvPython(venvDir)
	if info, sErr := os.Stat(pyExe); sErr != nil || info.IsDir() {
		return map[string]interface{}{"error": "Anki Importer environment not initialized. Please click Setup in Settings."}
	}

	profileID := a.resolveExplicitActiveProfileID()
	targetNotebookID = strings.TrimSpace(targetNotebookID)
	targetTopicID = strings.TrimSpace(targetTopicID)

	var finalNotebookID string
	var isStandalone bool
	if targetNotebookID != "" {
		finalNotebookID = targetNotebookID
	} else {
		finalNotebookID = uuid.NewString()
		isStandalone = true
	}

	// Prepare media directory under uploadDir/media/{finalNotebookID}
	// ponytail: serve via Wails AssetServer at /notebooks/media/{finalNotebookID}/
	mediaDir := filepath.Join(a.GetNotebookUploadDir(), "media", finalNotebookID)
	_ = os.MkdirAll(mediaDir, 0755)
	mediaPrefix := fmt.Sprintf("/notebooks/media/%s/", finalNotebookID)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	args := []string{
		ext.EntrypointPath(),
		"--file", filePath,
		"--media-dir", mediaDir,
		"--media-prefix", mediaPrefix,
	}
	output, err := a.extRunner.Run(ctx, ext, pyExe, args...)
	if err != nil {
		return map[string]interface{}{
			"error":  fmt.Sprintf("Anki parser error: %v", err),
			"output": string(output),
		}
	}

	var parsed ankiExtensionOutput
	if uErr := json.Unmarshal(output, &parsed); uErr != nil {
		return map[string]interface{}{
			"error":  fmt.Sprintf("Failed to parse Anki importer JSON output: %v", uErr),
			"output": string(output),
		}
	}

	if parsed.Error != "" {
		return map[string]interface{}{"error": parsed.Error}
	}

	if len(parsed.Cards) == 0 {
		return map[string]interface{}{"error": "No valid flashcards found in Anki package."}
	}

	var finalTopicID string

	if !isStandalone {
		// Importing into existing notebook
		if targetTopicID != "" {
			finalTopicID = targetTopicID
		} else {
			finalTopicID = fmt.Sprintf("topic-anki-%s", uuid.NewString()[:8])
			topicTitle := parsed.DeckName
			if topicTitle == "" {
				topicTitle = "Anki Flashcards"
			}
			if err := repo.EnsureTopic(finalTopicID, topicTitle); err != nil {
				return map[string]interface{}{"error": fmt.Sprintf("Failed to create topic: %v", err)}
			}
			_ = repo.LinkNotebookTopics(finalNotebookID, []string{finalTopicID})
		}
	} else {
		// Standalone Notebook Creation (page_count = 0, no reading tasks)
		deckTitle := parsed.DeckName
		if strings.TrimSpace(deckTitle) == "" {
			deckTitle = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		}
		if deckTitle == "" {
			deckTitle = "Anki Deck"
		}

		finalTopicID = fmt.Sprintf("topic-%s", finalNotebookID[:8])
		if err := repo.EnsureTopicWithStatus(finalTopicID, deckTitle, "completed"); err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Failed to create topic: %v", err)}
		}

		fileHash, _ := utils.FileSHA256(filePath)
		if err := repo.CreateNotebook(finalNotebookID, deckTitle, filePath, "anki", finalTopicID, fileHash, 0, profileID); err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Failed to create notebook: %v", err)}
		}
		_ = repo.UpdateNotebookStatus(finalNotebookID, "ready")
		_ = repo.LinkNotebookTopics(finalNotebookID, []string{finalTopicID})

		// Auto-activate in the active lane if the profile has less than 4 active notebooks
		// ponytail: matches textbook ingestion behavior
		if activeCount, err := repo.CountActiveNotebooksForActiveProfile(profileID); err == nil && activeCount < 4 {
			_ = repo.UpdateNotebookStudyStatus(finalNotebookID, "active")
		}
	}

	// Prepare cards and initial FSRS states
	now := time.Now().Unix()
	cardsToInsert := make([]models.Flashcard, 0, len(parsed.Cards))
	statesToInsert := make(map[string]models.FlashcardState, len(parsed.Cards))

	for _, c := range parsed.Cards {
		cardID := uuid.NewString()
		cardsToInsert = append(cardsToInsert, models.Flashcard{
			ID:        cardID,
			TopicID:   finalTopicID,
			Prompt:    c.Prompt,
			Answer:    c.Answer,
			DueAt:     now, // immediately available for review
			Suspended: false,
		})
		statesToInsert[cardID] = models.FlashcardState{
			Stability:     2.0,
			Difficulty:    5.0,
			ElapsedDays:   0,
			ScheduledDays: 0,
			Reps:          0,
			Lapses:        0,
			StateCode:     2, // Review state
		}
	}

	createdCards, _, err := repo.GetOrCreateFlashcardsForTopic(finalTopicID, cardsToInsert, statesToInsert)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Failed to persist flashcards: %v", err)}
	}

	return map[string]interface{}{
		"success":      true,
		"notebook_id":  finalNotebookID,
		"topic_id":     finalTopicID,
		"deck_name":    parsed.DeckName,
		"cards_count":  len(createdCards),
		"message":      fmt.Sprintf("Successfully imported %d flashcards from Anki deck %q", len(createdCards), parsed.DeckName),
	}
}
