package study

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"ai-tutor/internal/embeddings"
	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"
)

const studyNoteSystemPrompt = `You are a high-yield academic study notes assistant.
Generate a structured, concise study note designed for rapid conceptual review and high retention from the provided reference material.

Requirements:
- Length & Density: Scale length dynamically to match the density of the source material. Be as concise as possible while ensuring zero loss of core concepts. No padding, filler, or fluff.
- Tone: Rigorous, clear, conceptual, direct.
- Style: DO NOT use emojis. DO NOT include greetings, preamble, meta-explanations, or conversational text.
- Formatting: Clean markdown with bold technical terms and structured bullet points.

Structure:
- **Core Concept & Motivation**: 1–2 sentences defining the central idea and why it matters.
- **Key Mechanisms & Principles**: High-yield bullet points breaking down essential mechanics, relationships, and distinctions. Include relevant formulas, notation, or algorithmic steps if present.
- **Critical Nuances / Edge Cases**: Key trade-offs, boundary conditions, or common pitfalls (if applicable to the content).
- **Operational Takeaway**: 1 concluding rule of thumb, heuristic, or high-yield synthesis.
`

// NotesBaseDir returns the base directory where markdown notes are stored on disk.
func NotesBaseDir() string {
	if custom := strings.TrimSpace(os.Getenv("STUDYLOOP_NOTES_DIR")); custom != "" {
		return custom
	}
	return filepath.Join("dev_data", "notes")
}

// SanitizePathSegment cleans invalid OS filesystem characters for folder/file names.
func SanitizePathSegment(name string) string {
	invalid := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|", "\n", "\r", "\t"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	result = strings.TrimSpace(result)
	result = strings.Trim(result, ".")
	if result == "" {
		return "Untitled"
	}
	return result
}

// ParseMarkdownNote extracts YAML frontmatter (if present) and body content from a raw markdown string.
func ParseMarkdownNote(raw string, fallback models.TopicStudyNote) models.TopicStudyNote {
	note := fallback
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "---") {
		rest := strings.TrimPrefix(trimmed, "---")
		endIdx := strings.Index(rest, "\n---")
		if endIdx != -1 {
			frontmatter := rest[:endIdx]
			body := strings.TrimSpace(rest[endIdx+4:])
			note.Content = body

			lines := strings.Split(frontmatter, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, ":", 2)
				if len(parts) != 2 {
					continue
				}
				key := strings.ToLower(strings.TrimSpace(parts[0]))
				val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

				switch key {
				case "id":
					note.ID = val
				case "topic_id":
					note.TopicID = val
				case "topic_title":
					note.TopicTitle = utils.CleanTopicTitle(val)
				case "notebook_id":
					note.NotebookID = val
				case "notebook_title":
					note.NotebookTitle = val
				case "start_page":
					if p, err := strconv.Atoi(val); err == nil {
						note.StartPage = p
					}
				case "end_page":
					if p, err := strconv.Atoi(val); err == nil {
						note.EndPage = p
					}
				case "last_reviewed_at":
					if t, err := strconv.ParseInt(val, 10, 64); err == nil {
						note.LastReviewedAt = t
					}
				case "created_at":
					note.CreatedAt = val
				case "updated_at":
					note.UpdatedAt = val
				}
			}
			return note
		}
	}

	note.Content = raw
	return note
}

// FormatMarkdownNote serializes a TopicStudyNote with standard YAML frontmatter and markdown body.
func FormatMarkdownNote(note models.TopicStudyNote) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	if note.ID != "" {
		fmt.Fprintf(&sb, "id: %q\n", note.ID)
	}
	if note.NotebookID != "" {
		fmt.Fprintf(&sb, "notebook_id: %q\n", note.NotebookID)
	}
	if note.NotebookTitle != "" {
		fmt.Fprintf(&sb, "notebook_title: %q\n", note.NotebookTitle)
	}
	if note.TopicID != "" {
		fmt.Fprintf(&sb, "topic_id: %q\n", note.TopicID)
	}
	if note.TopicTitle != "" {
		fmt.Fprintf(&sb, "topic_title: %q\n", utils.CleanTopicTitle(note.TopicTitle))
	}
	fmt.Fprintf(&sb, "start_page: %d\n", note.StartPage)
	fmt.Fprintf(&sb, "end_page: %d\n", note.EndPage)
	fmt.Fprintf(&sb, "last_reviewed_at: %d\n", note.LastReviewedAt)
	if note.CreatedAt != "" {
		fmt.Fprintf(&sb, "created_at: %q\n", note.CreatedAt)
	}
	if note.UpdatedAt != "" {
		fmt.Fprintf(&sb, "updated_at: %q\n", note.UpdatedAt)
	} else {
		fmt.Fprintf(&sb, "updated_at: %q\n", time.Now().UTC().Format(time.RFC3339))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(strings.TrimSpace(note.Content))
	sb.WriteString("\n")
	return sb.String()
}

func (s *StudyService) resolveNotebookAndTopicTitles(topicID, notebookID string) (nbID, nbTitle, tID, tTitle string) {
	tID = strings.TrimSpace(topicID)
	nbID = strings.TrimSpace(notebookID)
	tTitle = utils.CleanTopicTitle(tID)
	nbTitle = "General"

	if tID != "" && s.repo != nil {
		if t, err := s.repo.GetTopic(tID); err == nil && t != nil && t.Title != "" {
			tTitle = utils.CleanTopicTitle(t.Title)
		}
		if nbID == "" {
			if nid, err := s.repo.GetNotebookIDForTopic(tID); err == nil && nid != "" {
				nbID = nid
			}
		}
	}

	if nbID != "" && s.repo != nil {
		if title, err := s.repo.GetNotebookTitle(nbID); err == nil && title != "" {
			nbTitle = title
		}
	}

	return nbID, nbTitle, tID, tTitle
}

// GetTopicNoteFolder returns the directory path and assets subfolder for a given topic.
func (s *StudyService) GetTopicNoteFolder(topicID, notebookID string) (folderPath, assetsPath string) {
	_, nbTitle, _, tTitle := s.resolveNotebookAndTopicTitles(topicID, notebookID)
	cleanNb := SanitizePathSegment(nbTitle)
	cleanTopic := SanitizePathSegment(tTitle)
	folderPath = filepath.Join(NotesBaseDir(), cleanNb, cleanTopic)
	assetsPath = filepath.Join(folderPath, "assets")
	return folderPath, assetsPath
}

// GetNoteFilePath computes the standard disk file path for a (topic, startPage, endPage) note.
func (s *StudyService) GetNoteFilePath(topicID, notebookID string, startPage, endPage int) string {
	folderPath, _ := s.GetTopicNoteFolder(topicID, notebookID)
	var fileName string
	if startPage == 0 && endPage == 0 {
		fileName = "chapter_summary.md"
	} else {
		fileName = fmt.Sprintf("pages_%d_%d.md", startPage, endPage)
	}
	return filepath.Join(folderPath, fileName)
}

// SaveNoteToDisk writes a TopicStudyNote to disk with YAML frontmatter, creating the parent folder and assets folder.
func (s *StudyService) SaveNoteToDisk(note models.TopicStudyNote) (string, error) {
	folderPath, assetsPath := s.GetTopicNoteFolder(note.TopicID, note.NotebookID)
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create note folder: %w", err)
	}
	if err := os.MkdirAll(assetsPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create assets folder: %w", err)
	}

	var fileName string
	if note.StartPage == 0 && note.EndPage == 0 {
		fileName = "chapter_summary.md"
	} else {
		fileName = fmt.Sprintf("pages_%d_%d.md", note.StartPage, note.EndPage)
	}
	filePath := filepath.Join(folderPath, fileName)

	if note.CreatedAt == "" {
		note.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	note.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	note.FilePath = filePath

	formatted := FormatMarkdownNote(note)
	if err := os.WriteFile(filePath, []byte(formatted), 0644); err != nil {
		return "", fmt.Errorf("failed to write markdown note to disk: %w", err)
	}
	return filePath, nil
}

// GenerateTopicStudyNote creates or regenerates a structured study note for a whole topic (start=0, end=0 slot).
func (s *StudyService) GenerateTopicStudyNote(topicID, notebookID string) (*models.TopicStudyNote, error) {
	return s.GenerateTopicStudyNoteForRange(topicID, notebookID, 0, 0)
}

// GenerateTopicStudyNoteForRange creates or replaces the study note for a specific reading session page range.
// It writes the markdown file directly to dev_data/notes/<Notebook>/<Chapter>/pages_<start>_<end>.md.
func (s *StudyService) GenerateTopicStudyNoteForRange(topicID, notebookID string, startPage, endPage int) (*models.TopicStudyNote, error) {
	topicID = strings.TrimSpace(topicID)
	notebookID = strings.TrimSpace(notebookID)
	if topicID == "" {
		return nil, fmt.Errorf("topic ID is required")
	}

	nbID, nbTitle, tID, tTitle := s.resolveNotebookAndTopicTitles(topicID, notebookID)

	// Fetch candidate chunks (scoped to session page range if provided)
	var chunks []models.Chunk
	var err error
	if startPage > 0 && endPage >= startPage {
		chunks, err = s.repo.GetChunksForTopicPageRange(tID, startPage, endPage)
	} else {
		chunks, err = s.repo.GetChunksForTopic(tID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chunks for topic %s: %w", topicID, err)
	}
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no content chunks available for topic %s", topicID)
	}

	// Sort chunks: top importance score first, then page number
	sortedChunks := make([]models.Chunk, len(chunks))
	copy(sortedChunks, chunks)
	sort.SliceStable(sortedChunks, func(i, j int) bool {
		if sortedChunks[i].ImportanceScore != sortedChunks[j].ImportanceScore {
			return sortedChunks[i].ImportanceScore > sortedChunks[j].ImportanceScore
		}
		return sortedChunks[i].PageNum < sortedChunks[j].PageNum
	})

	// Select LLM provider
	provider, tier := s.selectLLM("")
	if provider == nil {
		return nil, fmt.Errorf("no LLM provider configured")
	}

	limits := provider.GetLimits()
	maxInput := limits.MaxInputTokens
	if maxInput <= 0 {
		maxInput = 4000
	}

	// Assemble excerpt within budget (~70% of max input tokens)
	var contentBuilder strings.Builder
	currentTokens := 0
	tokenBudget := int(float64(maxInput) * 0.70)
	if tokenBudget < 500 {
		tokenBudget = 500
	}

	for _, c := range sortedChunks {
		chunkContent := c.CompressedText
		if strings.TrimSpace(chunkContent) == "" {
			chunkContent = c.Text
		}
		chunkContent = strings.TrimSpace(chunkContent)
		if chunkContent == "" {
			continue
		}

		estTokens, _ := embeddings.CountTokens(chunkContent)
		if currentTokens+estTokens > tokenBudget && currentTokens > 0 {
			break
		}

		if contentBuilder.Len() > 0 {
			contentBuilder.WriteString("\n\n")
		}
		if c.PageNum > 0 {
			fmt.Fprintf(&contentBuilder, "[Page %d]\n%s", c.PageNum, chunkContent)
		} else {
			contentBuilder.WriteString(chunkContent)
		}
		currentTokens += estTokens
	}

	referenceText := contentBuilder.String()
	if strings.TrimSpace(referenceText) == "" {
		return nil, fmt.Errorf("no substantive text content found for topic %s", topicID)
	}

	sectionContext := ""
	if startPage > 0 && endPage >= startPage {
		sectionContext = fmt.Sprintf(" (Pages %d–%d)", startPage, endPage)
	}

	prompt := fmt.Sprintf("%s\n\nTopic Title: %s%s\n\nReference Material:\n%s\n\nStructured Study Note:", studyNoteSystemPrompt, tTitle, sectionContext, referenceText)

	answer, genErr := provider.GenerateAnswer(prompt)
	if genErr != nil {
		return nil, s.FormatLLMError(genErr, tier)
	}

	cleanedContent := strings.TrimSpace(answer)
	if strings.HasPrefix(cleanedContent, "```markdown") {
		cleanedContent = strings.TrimPrefix(cleanedContent, "```markdown")
		cleanedContent = strings.TrimSuffix(cleanedContent, "```")
		cleanedContent = strings.TrimSpace(cleanedContent)
	} else if strings.HasPrefix(cleanedContent, "```") && strings.HasSuffix(cleanedContent, "```") {
		cleanedContent = strings.TrimPrefix(cleanedContent, "```")
		cleanedContent = strings.TrimSuffix(cleanedContent, "```")
		cleanedContent = strings.TrimSpace(cleanedContent)
	}

	note := models.TopicStudyNote{
		ID:             fmt.Sprintf("note-%s-%d-%d", utils.MD5Hex(tID), startPage, endPage),
		TopicID:        tID,
		TopicTitle:     tTitle,
		NotebookID:     nbID,
		NotebookTitle:  nbTitle,
		StartPage:      startPage,
		EndPage:        endPage,
		Content:        cleanedContent,
		LastReviewedAt: 0,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	filePath, saveErr := s.SaveNoteToDisk(note)
	if saveErr != nil {
		return nil, fmt.Errorf("failed to save study note to disk: %w", saveErr)
	}
	note.FilePath = filePath

	utils.Infof("[STUDY_NOTES] successfully generated study note for topic %s%s at %s (tier=%s)", topicID, sectionContext, filePath, tier)
	return &note, nil
}

// GenerateTopicStudyNoteAsync triggers asynchronous study note generation in the background.
func (s *StudyService) GenerateTopicStudyNoteAsync(ctx context.Context, topicID, notebookID string, startPage, endPage int) {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Errorf("[STUDY_NOTES] panic in GenerateTopicStudyNoteAsync: %v", r)
			}
		}()

		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		select {
		case <-bgCtx.Done():
			return
		default:
		}

		if _, err := s.GenerateTopicStudyNoteForRange(topicID, notebookID, startPage, endPage); err != nil {
			utils.Warnf("[STUDY_NOTES] async note generation failed for topic %s (pages %d-%d): %v", topicID, startPage, endPage, err)
		}
	}()
}

// GetTopicStudyNote retrieves the whole-chapter (0,0) study note for a given topic ID from disk.
func (s *StudyService) GetTopicStudyNote(topicID string) (*models.TopicStudyNote, error) {
	return s.GetTopicStudyNoteForRange(topicID, 0, 0)
}

// GetTopicStudyNoteForRange retrieves the study note for a specific (topic, startPage, endPage) slot directly from disk.
func (s *StudyService) GetTopicStudyNoteForRange(topicID string, startPage, endPage int) (*models.TopicStudyNote, error) {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return nil, fmt.Errorf("topic ID is required")
	}

	nbID, nbTitle, tID, tTitle := s.resolveNotebookAndTopicTitles(topicID, "")
	filePath := s.GetNoteFilePath(topicID, nbID, startPage, endPage)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	fallback := models.TopicStudyNote{
		ID:            fmt.Sprintf("note-%s-%d-%d", utils.MD5Hex(tID), startPage, endPage),
		TopicID:       tID,
		TopicTitle:    tTitle,
		NotebookID:    nbID,
		NotebookTitle: nbTitle,
		StartPage:     startPage,
		EndPage:       endPage,
		FilePath:      filePath,
	}

	note := ParseMarkdownNote(string(data), fallback)
	note.FilePath = filePath
	if note.TopicID == "" {
		note.TopicID = tID
	}
	if note.TopicTitle == "" {
		note.TopicTitle = tTitle
	} else {
		note.TopicTitle = utils.CleanTopicTitle(note.TopicTitle)
	}
	if note.NotebookID == "" {
		note.NotebookID = nbID
	}
	if note.NotebookTitle == "" {
		note.NotebookTitle = nbTitle
	}
	return &note, nil
}

// GetTopicStudyNoteSlots returns all note cards for a topic by scanning its folder on disk
// and merging with completed/active reading sessions from study_queue.
func (s *StudyService) GetTopicStudyNoteSlots(topicID string) ([]models.TopicStudyNote, error) {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return nil, fmt.Errorf("topic ID is required")
	}

	nbID, nbTitle, tID, tTitle := s.resolveNotebookAndTopicTitles(topicID, "")
	folderPath, _ := s.GetTopicNoteFolder(topicID, nbID)

	slotsMap := make(map[string]models.TopicStudyNote)

	// 1. Scan directory on disk for all existing .md note files
	if entries, err := os.ReadDir(folderPath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
				continue
			}

			filePath := filepath.Join(folderPath, entry.Name())
			data, readErr := os.ReadFile(filePath)
			if readErr != nil {
				continue
			}

			// Parse start and end page from file name if named pages_<start>_<end>.md
			startP, endP := 0, 0
			nameNoExt := strings.TrimSuffix(entry.Name(), ".md")
			if strings.HasPrefix(nameNoExt, "pages_") {
				parts := strings.Split(strings.TrimPrefix(nameNoExt, "pages_"), "_")
				if len(parts) == 2 {
					sP, _ := strconv.Atoi(parts[0])
					eP, _ := strconv.Atoi(parts[1])
					startP = sP
					endP = eP
				}
			}

			fallback := models.TopicStudyNote{
				ID:            fmt.Sprintf("note-%s-%d-%d", utils.MD5Hex(tID), startP, endP),
				TopicID:       tID,
				TopicTitle:    tTitle,
				NotebookID:    nbID,
				NotebookTitle: nbTitle,
				StartPage:     startP,
				EndPage:       endP,
				FilePath:      filePath,
			}

			note := ParseMarkdownNote(string(data), fallback)
			note.FilePath = filePath
			if note.TopicID == "" {
				note.TopicID = tID
			}
			if note.TopicTitle == "" {
				note.TopicTitle = tTitle
			} else {
				note.TopicTitle = utils.CleanTopicTitle(note.TopicTitle)
			}
			if note.NotebookID == "" {
				note.NotebookID = nbID
			}
			if note.NotebookTitle == "" {
				note.NotebookTitle = nbTitle
			}

			key := fmt.Sprintf("%d_%d", note.StartPage, note.EndPage)
			if note.StartPage == 0 && note.EndPage == 0 && nameNoExt != "chapter_summary" {
				key = nameNoExt
			}
			slotsMap[key] = note
		}
	}

	// 2. Discover completed and active reading session ranges from study_queue (to offer empty cards with "Generate" button)
	if s.repo != nil {
		if sessions, err := s.repo.GetCompletedReadingSessionsForTopic(topicID); err == nil {
			for _, sess := range sessions {
				key := fmt.Sprintf("%d_%d", sess.StartPage, sess.EndPage)
				if _, exists := slotsMap[key]; !exists {
					slotsMap[key] = models.TopicStudyNote{
						ID:            fmt.Sprintf("note-%s-%d-%d", utils.MD5Hex(tID), sess.StartPage, sess.EndPage),
						TopicID:       tID,
						TopicTitle:    tTitle,
						NotebookID:    nbID,
						NotebookTitle: nbTitle,
						StartPage:     sess.StartPage,
						EndPage:       sess.EndPage,
						Content:       "",
					}
				}
			}
		}
	}

	// 3. Flatten and sort slots
	notes := make([]models.TopicStudyNote, 0, len(slotsMap))
	for _, n := range slotsMap {
		notes = append(notes, n)
	}

	sort.Slice(notes, func(i, j int) bool {
		// (0,0) / chapter_summary always first
		if notes[i].StartPage == 0 && notes[i].EndPage == 0 && (notes[j].StartPage != 0 || notes[j].EndPage != 0) {
			return true
		}
		if notes[j].StartPage == 0 && notes[j].EndPage == 0 && (notes[i].StartPage != 0 || notes[i].EndPage != 0) {
			return false
		}
		if notes[i].StartPage != notes[j].StartPage {
			return notes[i].StartPage < notes[j].StartPage
		}
		return notes[i].EndPage < notes[j].EndPage
	})

	return notes, nil
}

// UpdateTopicStudyNote updates the markdown text of a specific (topic, page range) note directly on disk.
func (s *StudyService) UpdateTopicStudyNote(topicID string, startPage, endPage int, content string) error {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return fmt.Errorf("topic ID is required")
	}

	nbID, nbTitle, tID, tTitle := s.resolveNotebookAndTopicTitles(topicID, "")

	// Check if file already exists to preserve created_at & last_reviewed_at
	existingNote, _ := s.GetTopicStudyNoteForRange(topicID, startPage, endPage)
	note := models.TopicStudyNote{
		ID:            fmt.Sprintf("note-%s-%d-%d", utils.MD5Hex(tID), startPage, endPage),
		TopicID:       tID,
		TopicTitle:    tTitle,
		NotebookID:    nbID,
		NotebookTitle: nbTitle,
		StartPage:     startPage,
		EndPage:       endPage,
		Content:       content,
	}

	if existingNote != nil {
		note.ID = existingNote.ID
		note.CreatedAt = existingNote.CreatedAt
		note.LastReviewedAt = existingNote.LastReviewedAt
	}

	_, err := s.SaveNoteToDisk(note)
	return err
}

// GetNotesByNotebook scans markdown files on disk for all topics in a notebook (or all notebooks).
func (s *StudyService) GetNotesByNotebook(notebookID string) ([]models.TopicStudyNote, error) {
	notebookID = strings.TrimSpace(notebookID)
	baseDir := NotesBaseDir()

	var searchDir string
	if notebookID != "" && s.repo != nil {
		if title, err := s.repo.GetNotebookTitle(notebookID); err == nil && title != "" {
			searchDir = filepath.Join(baseDir, SanitizePathSegment(title))
		}
	}
	if searchDir == "" {
		searchDir = baseDir
	}

	if _, err := os.Stat(searchDir); os.IsNotExist(err) {
		return []models.TopicStudyNote{}, nil
	}

	var notes []models.TopicStudyNote
	err := filepath.WalkDir(searchDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.EqualFold(d.Name(), "assets") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		fallback := models.TopicStudyNote{
			FilePath: path,
		}
		note := ParseMarkdownNote(string(data), fallback)
		note.FilePath = path
		if note.TopicTitle != "" {
			note.TopicTitle = utils.CleanTopicTitle(note.TopicTitle)
		}
		if strings.TrimSpace(note.Content) != "" {
			notes = append(notes, note)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(notes, func(i, j int) bool {
		if notes[i].NotebookTitle != notes[j].NotebookTitle {
			return notes[i].NotebookTitle < notes[j].NotebookTitle
		}
		if notes[i].TopicTitle != notes[j].TopicTitle {
			return notes[i].TopicTitle < notes[j].TopicTitle
		}
		return notes[i].StartPage < notes[j].StartPage
	})

	return notes, nil
}

// MarkTopicReviewed updates the last_reviewed_at timestamp in the markdown note's YAML frontmatter on disk.
func (s *StudyService) MarkTopicReviewed(topicID string, startPage, endPage int) error {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return fmt.Errorf("topic ID is required")
	}

	existing, err := s.GetTopicStudyNoteForRange(topicID, startPage, endPage)
	if err != nil {
		return err
	}

	if existing == nil {
		nbID, nbTitle, tID, tTitle := s.resolveNotebookAndTopicTitles(topicID, "")
		existing = &models.TopicStudyNote{
			ID:            fmt.Sprintf("note-%s-%d-%d", utils.MD5Hex(tID), startPage, endPage),
			TopicID:       tID,
			TopicTitle:    tTitle,
			NotebookID:    nbID,
			NotebookTitle: nbTitle,
			StartPage:     startPage,
			EndPage:       endPage,
			Content:       "",
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		}
	}

	existing.LastReviewedAt = time.Now().Unix()
	_, saveErr := s.SaveNoteToDisk(*existing)
	return saveErr
}

// OpenNotesFolder opens the local directory containing markdown study notes in the OS file explorer.
func (s *StudyService) OpenNotesFolder(notebookID, topicID string) (string, error) {
	folderPath := NotesBaseDir()
	nbID, nbTitle, tID, tTitle := s.resolveNotebookAndTopicTitles(topicID, notebookID)
	_ = nbID
	_ = tID

	if strings.TrimSpace(notebookID) != "" || strings.TrimSpace(topicID) != "" {
		cleanNb := SanitizePathSegment(nbTitle)
		if strings.TrimSpace(topicID) != "" {
			cleanTopic := SanitizePathSegment(tTitle)
			folderPath = filepath.Join(folderPath, cleanNb, cleanTopic)
		} else {
			folderPath = filepath.Join(folderPath, cleanNb)
		}
	}

	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		absPath = folderPath
	}

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create notes folder: %w", err)
	}

	// Also ensure assets folder exists if chapter level
	if strings.TrimSpace(topicID) != "" {
		_ = os.MkdirAll(filepath.Join(absPath, "assets"), 0755)
	}

	var cmd *exec.Cmd
	switch stdruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", absPath)
	case "darwin":
		cmd = exec.Command("open", absPath)
	default:
		cmd = exec.Command("xdg-open", absPath)
	}

	if err := cmd.Start(); err != nil {
		return absPath, fmt.Errorf("failed to launch file explorer: %w", err)
	}

	return absPath, nil
}
