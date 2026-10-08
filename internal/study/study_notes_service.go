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

const briefingNoteSystemPrompt = `You are an executive intelligence analyst and academic synthesizer.
Create a briefing document that synthesizes the main themes and ideas from the reference material.

Requirements:
- Length & Density: 300–450 words total (balanced mid-size briefing).
- Tone: Objective, incisive, authoritative, and analytical. Strictly formal; avoid colloquialisms, slang, and conversational filler.
- Style: DO NOT use emojis. DO NOT include conversational openings, greetings, preamble (e.g., "Sure, here is your summary"), or meta-commentary.
- Formatting: Clean markdown with clear section headers, bold technical terms on first mention, and structured bullet points.

Structure:
### Executive Summary
A concise 2–3 sentence distillation presenting the core thesis and most critical takeaways upfront.

### Key Themes & Evidence
Organize into 2–3 logical thematic sections with subheadings. For each theme:
- Provide a rigorous, focused examination of the main concepts, mechanisms, and findings.
- Include 2–3 structured bullet points with concrete evidence, governing rules, or formulas.

### Strategic Takeaways & Boundary Conditions
1–2 concise paragraphs or bullet points highlighting key boundary conditions, limitations, or practical implications.
`

func getStudyNoteSystemPrompt(detailLevel string) string {
	switch strings.ToLower(strings.TrimSpace(detailLevel)) {
	case "briefing":
		return briefingNoteSystemPrompt
	case "exec_summary", "concise":
		return `You are an expert academic summarizer.
Generate a fast, high-density executive summary note from the provided material.

Requirements:
- Length: 150–200 words total.
- Tone: Rigorous, clear, conceptual, and direct. Strictly formal; zero conversational filler.
- Style: DO NOT use emojis. DO NOT include greetings, preamble, meta-explanations, or conversational text.
- Formatting: Clean markdown with bold technical terms and strictly capped bullet points.

Structure:
### Executive Summary
A direct 2–3 sentence distillation of what this topic is, why it works, and its core significance.

### Key Takeaways
- **[Core Principle 1]**: 1–2 line precise definition or governing rule.
- **[Mechanism / Formula 2]**: 1–2 line process, mathematical formulation, or relationship.
- **[Application / Boundary 3]**: 1–2 line practical implication, distinction, or key caveat.
(Strictly limit to 3–4 high-yield bullets maximum).
`
	case "concept_card":
		return `You are a high-yield study assistant.
Extract the fundamental conceptual core from the reference text into a tight, focused reference card.

Requirements:
- Length: 180–220 words total.
- Tone: Academic, rigorous, direct, and precise. No informal phrasing or conversational filler.
- Style: DO NOT use emojis. DO NOT include greetings, preamble, meta-explanations, or conversational text.
- Formatting: Clean Markdown. Bold key technical terms on first mention. Maximum 4 bullet points total across the note.

Structure:
### Core Definition
1–2 precise sentences defining the concept, its scope, and why it matters.

### Core Mechanism & Rules
- **Governing Law / Mechanism**: 1–2 lines explaining how it operates or equations involved.
- **Key Distinctions**: 1–2 lines clarifying differences from related concepts.
- **Application Context**: 1–2 lines on where and how it is applied.

### Key Takeaway
1 practical rule of thumb, memory anchor, or critical boundary condition.
`
	case "cheatsheet", "bullet":
		return `You are a technical cheatsheet generator.
Convert the provided material into an ultra-concise reference cheatsheet.

Requirements:
- Format: Strictly bullet points, definitions, laws, and equations. Zero narrative prose or conversational transitions.
- Length: 120–180 words total (maximum 5–6 high-density bullet points).
- Tone: Dense, factual, objective, and direct. Zero fluff.
- Style: DO NOT use emojis. DO NOT include greetings, preamble, meta-explanations, or conversational text.

Structure:
### Quick Reference
- **Definition**: 1-line exact definition of the core topic.
- **Core Rules & Equations**: 2–3 concise bullet points covering formulas, laws, or syntax.
- **Pitfalls & Edge Cases**: 1–2 bullet points highlighting common traps, boundary conditions, or exceptions.
`
	case "feynman":
		return `You are a first-principles study synthesis tutor.
Synthesize the reference text into an intuitive, highly lucid explanation using first principles and concrete reasoning.

Requirements:
- Length: 180–260 words total.
- Tone: Objective, clear, plain-spoken yet technically rigorous. Avoid childish slang or conversational chatter.
- Style: DO NOT use emojis. DO NOT include greetings, conversational filler, or meta-commentary.
- Formatting: Clean markdown with bold technical terms.

Structure:
### Intuition & Analogy
2–3 sentences grounding the abstract idea in a clear, intuitive mental model or physical analogy.

### First-Principles Mechanics
- **[Governing Principle]**: 1–2 lines on what fundamental law or logic governs this system.
- **[Operational Flow / Formula]**: 1–2 lines on how it is calculated, executed, or applied.
- **[Common Misconception]**: 1–2 lines on why people get this wrong and what the actual reality is.

### Bottom Line
1 sentence summarizing the core insight in plain terms.
`
	case "detailed":
		return `You are a comprehensive academic study notes assistant.
Generate an in-depth, structured study note covering the full breadth and nuance of the provided reference material.

Requirements:
- Length: 500–800 words total.
- Coverage: Thorough and exhaustive. Capture all key arguments, subtopics, derivations, mechanisms, and examples.
- Tone: Academic, rigorous, objective, and structured.
- Style: DO NOT use emojis. DO NOT include greetings, preamble, meta-explanations, or conversational text.
- Formatting: Clean markdown with structured headers, bold terminology, and clear bullet points.

Structure:
### 1. Overview & Conceptual Foundations
Detailed definition of core concepts, historical/theoretical context, and motivations.

### 2. Deep-Dive Mechanisms & Principles
In-depth breakdown of concepts, processes, mathematical formulations, and critical distinctions.

### 3. Concrete Applications & Examples
Specific applications, scenarios, or case studies illustrated in the text.

### 4. Nuances, Exceptions & Edge Cases
Boundary conditions, common misconceptions, and critical caveats.

### 5. Executive Synthesis
2–3 sentence high-yield takeaway summarizing the core conclusion.
`
	default:
		return briefingNoteSystemPrompt
	}
}

// NotesBaseDir returns the base directory where markdown notes are stored on disk for this service instance.
func (s *StudyService) NotesBaseDir() string {
	if s != nil && strings.TrimSpace(s.notesDir) != "" {
		return s.notesDir
	}
	return NotesBaseDir()
}

// NotesBaseDir returns the base directory where markdown notes are stored on disk.
func NotesBaseDir() string {
	if custom := strings.TrimSpace(os.Getenv("STUDYLOOP_NOTES_DIR")); custom != "" {
		return custom
	}
	// Dev: keep data in the repository for convenience.
	if os.Getenv("APP_ENV") == "dev" {
		projectRoot, err := os.Getwd()
		if err == nil && projectRoot != "" {
			return filepath.Join(projectRoot, "dev_data", "notes")
		}
		return filepath.Join("dev_data", "notes")
	}

	// Prod/default: use a stable per-user directory matching app data location.
	if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
		return filepath.Join(cfgDir, "Studyloop", "notes")
	} else if cacheDir, err := os.UserCacheDir(); err == nil && cacheDir != "" {
		return filepath.Join(cacheDir, "Studyloop", "notes")
	} else if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		return filepath.Join(homeDir, ".Studyloop", "notes")
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
				case "order":
					if ord, err := strconv.Atoi(val); err == nil {
						note.Order = ord
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
	if note.Order > 0 {
		fmt.Fprintf(&sb, "order: %d\n", note.Order)
	}
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

// ExtractUserNotesSection extracts user-authored notes (e.g. "### My Notes", "## Personal Notes", or custom manual notes)
// from an existing study note markdown so they can be preserved across AI regenerations.
func ExtractUserNotesSection(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}

	lines := strings.Split(trimmed, "\n")
	myNotesHeaderIndex := -1

	// Check for a heading matching "My Notes", "Personal Notes", "User Notes", "Custom Notes"
	for i, line := range lines {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "#") {
			headerText := strings.ToLower(strings.TrimLeft(l, "# "))
			if strings.HasPrefix(headerText, "my notes") ||
				strings.HasPrefix(headerText, "personal notes") ||
				strings.HasPrefix(headerText, "user notes") ||
				strings.HasPrefix(headerText, "custom notes") {
				myNotesHeaderIndex = i
				break
			}
		}
	}

	if myNotesHeaderIndex != -1 {
		return strings.TrimSpace(strings.Join(lines[myNotesHeaderIndex:], "\n"))
	}

	// If no explicit "My Notes" header was found: check if this was a manually created note (not AI generated)
	lower := strings.ToLower(trimmed)
	hasAITemplate := strings.Contains(lower, "core concept") ||
		strings.Contains(lower, "key mechanisms") ||
		strings.Contains(lower, "structured study note") ||
		strings.Contains(lower, "summary of pages")

	if !hasAITemplate {
		// The entire note was written by the user! Preserve it wrapped in "### My Notes"
		return fmt.Sprintf("### My Notes\n%s", trimmed)
	}

	return ""
}

// GetTopicNoteFolder returns the directory path and assets subfolder for a given topic.
func (s *StudyService) GetTopicNoteFolder(topicID, notebookID string) (folderPath, assetsPath string) {
	_, nbTitle, tID, tTitle := s.resolveNotebookAndTopicTitles(topicID, notebookID)
	cleanNb := SanitizePathSegment(nbTitle)
	cleanTopic := SanitizePathSegment(tTitle)

	folderName := cleanTopic
	if tID != "" {
		folderName = fmt.Sprintf("%s_%s", cleanTopic, SanitizePathSegment(tID))
	}

	folderPath = filepath.Join(s.NotesBaseDir(), cleanNb, folderName)
	legacyPath := filepath.Join(s.NotesBaseDir(), cleanNb, cleanTopic)

	// If legacy folder exists and new suffixed folder doesn't exist, fallback to legacy
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		if _, legErr := os.Stat(legacyPath); legErr == nil {
			folderPath = legacyPath
		}
	}

	assetsPath = filepath.Join(folderPath, "assets")
	return folderPath, assetsPath
}

// GetTopicRelativeFolder returns the relative folder path from NotesBaseDir for a topic (e.g. "Notebook/Topic_id").
func (s *StudyService) GetTopicRelativeFolder(topicID, notebookID string) string {
	folderPath, _ := s.GetTopicNoteFolder(topicID, notebookID)
	baseDir := s.NotesBaseDir()
	if baseDir == "" || folderPath == "" {
		return ""
	}
	rel, err := filepath.Rel(baseDir, folderPath)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
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
	if note.RelativeFolderPath == "" {
		note.RelativeFolderPath = s.GetTopicRelativeFolder(note.TopicID, note.NotebookID)
	}

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
	if topicID == "" && notebookID == "" {
		return nil, fmt.Errorf("topic ID or notebook ID is required")
	}

	nbID, nbTitle, tID, tTitle := s.resolveNotebookAndTopicTitles(topicID, notebookID)

	// Fetch candidate chunks (scoped to session page range if provided with fallback ladder)
	var chunks []models.Chunk
	var err error
	if startPage > 0 && endPage >= startPage && tID != "" {
		chunks, err = s.repo.GetChunksForTopicPageRange(tID, startPage, endPage)
	} else if tID != "" {
		chunks, err = s.repo.GetChunksForTopic(tID)
	}
	if err != nil {
		utils.Warnf("[STUDY_NOTES] fetch chunks for topic %s err: %v", topicID, err)
	}
	if len(chunks) == 0 && tID != "" {
		chunks, _ = s.repo.GetChunksForTopic(tID)
	}
	if len(chunks) == 0 && nbID != "" && startPage > 0 && endPage >= startPage {
		chunks, _ = s.repo.GetChunksForNotebookPageRange(nbID, startPage, endPage)
	}
	if len(chunks) == 0 && nbID != "" {
		chunks, _ = s.repo.GetChunksForNotebook(nbID)
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

	// User settings for model tier and detail level
	detailLevel := "concise"
	modelTierPref := "fast"
	if s.repo != nil {
		if uSettings, sErr := s.repo.GetUserSettings(); sErr == nil && uSettings != nil {
			if strings.TrimSpace(uSettings.NotesDetailLevel) != "" {
				detailLevel = uSettings.NotesDetailLevel
			}
			if strings.TrimSpace(uSettings.NotesModelTier) != "" {
				modelTierPref = uSettings.NotesModelTier
			}
		}
	}

	// Select LLM provider respecting modelTierPref
	var provider LLMProvider
	var tier string
	if strings.EqualFold(modelTierPref, "heavy") && s.heavyLLMProvider != nil {
		provider = s.heavyLLMProvider
		tier = "heavy"
	} else {
		provider, tier = s.selectLLM("")
	}
	if provider == nil {
		if s.fastLLMProvider != nil {
			provider = s.fastLLMProvider
			tier = "fast"
		} else if s.heavyLLMProvider != nil {
			provider = s.heavyLLMProvider
			tier = "heavy"
		}
	}
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

	systemPrompt := getStudyNoteSystemPrompt(detailLevel)
	prompt := fmt.Sprintf("%s\n\nTopic Title: %s%s\n\nReference Material:\n%s\n\nStructured Study Note:", systemPrompt, tTitle, sectionContext, referenceText)

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

	// Check for existing note to preserve user notes / ### My Notes sections and timestamps
	existingNote, _ := s.GetTopicStudyNoteForRange(tID, startPage, endPage)
	var userNotes string
	createdAt := ""
	var lastReviewedAt int64
	if existingNote != nil {
		createdAt = existingNote.CreatedAt
		lastReviewedAt = existingNote.LastReviewedAt
		userNotes = ExtractUserNotesSection(existingNote.Content)
	}

	finalContent := cleanedContent
	if strings.TrimSpace(userNotes) != "" {
		finalContent = strings.TrimSpace(cleanedContent) + "\n\n" + strings.TrimSpace(userNotes)
	}

	if createdAt == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339)
	}

	note := models.TopicStudyNote{
		ID:                 fmt.Sprintf("note-%s-%d-%d", utils.MD5Hex(tID), startPage, endPage),
		TopicID:            tID,
		TopicTitle:         tTitle,
		NotebookID:         nbID,
		NotebookTitle:      nbTitle,
		StartPage:          startPage,
		EndPage:            endPage,
		RelativeFolderPath: s.GetTopicRelativeFolder(tID, nbID),
		Content:            finalContent,
		LastReviewedAt:     lastReviewedAt,
		CreatedAt:          createdAt,
		UpdatedAt:          time.Now().UTC().Format(time.RFC3339),
	}

	filePath, saveErr := s.SaveNoteToDisk(note)
	if saveErr != nil {
		return nil, fmt.Errorf("failed to save study note to disk: %w", saveErr)
	}
	note.FilePath = filePath

	utils.Infof("[STUDY_NOTES] successfully generated study note for topic %s%s at %s (tier=%s, detail=%s)", topicID, sectionContext, filePath, tier, detailLevel)
	return &note, nil
}

// GenerateTopicStudyNoteAsync triggers asynchronous study note generation in the background.
func (s *StudyService) GenerateTopicStudyNoteAsync(ctx context.Context, topicID, notebookID string, startPage, endPage int) {
	topicID = strings.TrimSpace(topicID)
	notebookID = strings.TrimSpace(notebookID)
	if topicID == "" && notebookID == "" {
		return
	}

	key := fmt.Sprintf("%s:%s:%d:%d", topicID, notebookID, startPage, endPage)
	s.inFlightNotesMu.Lock()
	if s.inFlightNotes == nil {
		s.inFlightNotes = make(map[string]bool)
	}
	if s.inFlightNotes[key] {
		s.inFlightNotesMu.Unlock()
		return
	}
	s.inFlightNotes[key] = true
	s.inFlightNotesMu.Unlock()

	go func() {
		defer func() {
			s.inFlightNotesMu.Lock()
			delete(s.inFlightNotes, key)
			s.inFlightNotesMu.Unlock()

			if r := recover(); r != nil {
				utils.Errorf("[STUDY_NOTES] panic in GenerateTopicStudyNoteAsync: %v", r)
			}
		}()

		bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		_ = bgCtx

		if _, err := s.GenerateTopicStudyNoteForRange(topicID, notebookID, startPage, endPage); err != nil {
			utils.Warnf("[STUDY_NOTES] async note generation failed for topic %s notebook %s (pages %d-%d): %v", topicID, notebookID, startPage, endPage, err)
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
		ID:                 fmt.Sprintf("note-%s-%d-%d", utils.MD5Hex(tID), startPage, endPage),
		TopicID:            tID,
		TopicTitle:         tTitle,
		NotebookID:         nbID,
		NotebookTitle:      nbTitle,
		StartPage:          startPage,
		EndPage:            endPage,
		FilePath:           filePath,
		RelativeFolderPath: s.GetTopicRelativeFolder(tID, nbID),
	}

	note := ParseMarkdownNote(string(data), fallback)
	if note.TopicID != "" && tID != "" && note.TopicID != tID {
		return nil, nil
	}
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
	if note.RelativeFolderPath == "" {
		note.RelativeFolderPath = s.GetTopicRelativeFolder(note.TopicID, note.NotebookID)
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
				ID:                 fmt.Sprintf("note-%s-%d-%d", utils.MD5Hex(tID), startP, endP),
				TopicID:            tID,
				TopicTitle:         tTitle,
				NotebookID:         nbID,
				NotebookTitle:      nbTitle,
				StartPage:          startP,
				EndPage:            endP,
				FilePath:           filePath,
				RelativeFolderPath: s.GetTopicRelativeFolder(tID, nbID),
			}

			note := ParseMarkdownNote(string(data), fallback)
			if note.TopicID != "" && tID != "" && note.TopicID != tID {
				continue
			}
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
			if note.RelativeFolderPath == "" {
				note.RelativeFolderPath = s.GetTopicRelativeFolder(note.TopicID, note.NotebookID)
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
						ID:                 fmt.Sprintf("note-%s-%d-%d", utils.MD5Hex(tID), sess.StartPage, sess.EndPage),
						TopicID:            tID,
						TopicTitle:         tTitle,
						NotebookID:         nbID,
						NotebookTitle:      nbTitle,
						StartPage:          sess.StartPage,
						EndPage:            sess.EndPage,
						RelativeFolderPath: s.GetTopicRelativeFolder(tID, nbID),
						Content:            "",
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
		// 1. If explicit order is set on both, compare order
		if notes[i].Order != 0 && notes[j].Order != 0 {
			if notes[i].Order != notes[j].Order {
				return notes[i].Order < notes[j].Order
			}
		} else if notes[i].Order != 0 {
			return true
		} else if notes[j].Order != 0 {
			return false
		}

		// 2. (0,0) / chapter_summary always first if no explicit order
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

// ReorderTopicStudyNotes updates the order frontmatter of a list of note slots in a topic.
func (s *StudyService) ReorderTopicStudyNotes(topicID string, slots []models.NoteSlotRange) error {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return fmt.Errorf("topic ID is required")
	}
	nbID, nbTitle, tID, tTitle := s.resolveNotebookAndTopicTitles(topicID, "")

	for idx, slot := range slots {
		newOrder := idx + 1
		existingNote, err := s.GetTopicStudyNoteForRange(topicID, slot.StartPage, slot.EndPage)
		if err != nil || existingNote == nil {
			continue
		}
		existingNote.Order = newOrder
		if existingNote.NotebookID == "" {
			existingNote.NotebookID = nbID
		}
		if existingNote.NotebookTitle == "" {
			existingNote.NotebookTitle = nbTitle
		}
		if existingNote.TopicID == "" {
			existingNote.TopicID = tID
		}
		if existingNote.TopicTitle == "" {
			existingNote.TopicTitle = tTitle
		}
		if _, saveErr := s.SaveNoteToDisk(*existingNote); saveErr != nil {
			utils.Warnf("[STUDY_NOTES] Failed saving reordered note %d-%d: %v", slot.StartPage, slot.EndPage, saveErr)
		}
	}
	return nil
}

// UpdateTopicStudyNote updates the markdown text of a specific (topic, page range) note directly on disk.
func (s *StudyService) UpdateTopicStudyNote(topicID string, startPage, endPage int, content string) error {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return fmt.Errorf("topic ID is required")
	}

	nbID, nbTitle, tID, tTitle := s.resolveNotebookAndTopicTitles(topicID, "")

	// Check if file already exists to preserve created_at, last_reviewed_at & order
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
		note.Order = existingNote.Order
	}

	_, err := s.SaveNoteToDisk(note)
	return err
}

// GetNotesByNotebook scans markdown files on disk for all topics in a notebook (or all notebooks).
func (s *StudyService) GetNotesByNotebook(notebookID string) ([]models.TopicStudyNote, error) {
	notebookID = strings.TrimSpace(notebookID)
	baseDir := s.NotesBaseDir()

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
		if baseDir != "" {
			if rel, err := filepath.Rel(baseDir, filepath.Dir(path)); err == nil {
				note.RelativeFolderPath = filepath.ToSlash(rel)
			}
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
		return nil
	}

	existing.LastReviewedAt = time.Now().Unix()
	_, saveErr := s.SaveNoteToDisk(*existing)
	return saveErr
}

// OpenNotesFolder opens the local directory containing markdown study notes in the OS file explorer.
func (s *StudyService) OpenNotesFolder(notebookID, topicID string) (string, error) {
	folderPath := s.NotesBaseDir()
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

// SaveNoteImage saves raw base64 or binary image data into the topic note's assets directory.
func (s *StudyService) SaveNoteImage(topicID, notebookID, fileName string, fileData []byte) (string, string, error) {
	if len(fileData) == 0 {
		return "", "", fmt.Errorf("file data is empty")
	}

	_, assetsPath := s.GetTopicNoteFolder(topicID, notebookID)
	if err := os.MkdirAll(assetsPath, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create assets folder: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" || (ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" && ext != ".gif" && ext != ".svg") {
		ext = ".png"
	}

	cleanBase := strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
	if cleanBase == "" || cleanBase == "." {
		cleanBase = "pasted_image"
	}
	cleanBase = SanitizePathSegment(cleanBase)

	uniqueName := fmt.Sprintf("%s_%d%s", cleanBase, time.Now().UnixMilli(), ext)
	targetPath := filepath.Join(assetsPath, uniqueName)

	if err := os.WriteFile(targetPath, fileData, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write image file: %w", err)
	}

	// Return relative path suitable for markdown image embedding
	relPath := fmt.Sprintf("assets/%s", uniqueName)
	relFolder := s.GetTopicRelativeFolder(topicID, notebookID)
	return relPath, relFolder, nil
}

