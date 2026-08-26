package notebook

import (
	"strings"

	"ai-tutor/internal/embeddings"
	"ai-tutor/internal/models"
)

// MarkdownBlockType categorizes a block of markdown text.
type MarkdownBlockType int

const (
	BlockParagraph MarkdownBlockType = iota
	BlockHeading
	BlockCode
	BlockTable
	BlockList
)

// MarkdownBlock represents a parsed block with its type and content.
type MarkdownBlock struct {
	Type    MarkdownBlockType
	Content string
	Heading string
}

// SplitMarkdownIntoChunks splits markdown text into semantically cohesive chunks.
// It respects heading hierarchy, fenced code blocks (```), and markdown tables (| ... |)
// so that structural elements are never severed across chunk boundaries.
func SplitMarkdownIntoChunks(markdownText string, targetWords int) []string {
	if targetWords <= 0 {
		targetWords = DefaultChunkTargetWords
	}

	blocks := parseMarkdownBlocks(markdownText)
	if len(blocks) == 0 {
		return nil
	}

	chunks := make([]string, 0)
	var currentChunk strings.Builder
	currentWordCount := 0

	flushCurrent := func() {
		str := strings.TrimSpace(currentChunk.String())
		if str != "" {
			chunks = append(chunks, str)
		}
		currentChunk.Reset()
		currentWordCount = 0
	}

	for _, block := range blocks {
		blockText := strings.TrimSpace(block.Content)
		if blockText == "" {
			continue
		}

		blockWords := len(strings.Fields(blockText))

		// If this is an H1/H2 heading and we already have accumulated content, flush to start a new section
		if block.Type == BlockHeading && currentWordCount >= chunkLowerBoundWords {
			flushCurrent()
		}

		// If adding this block exceeds the upper bound, flush first (unless current is empty)
		if currentWordCount > 0 && (currentWordCount+blockWords > chunkUpperBoundWords) {
			flushCurrent()
		}

		// If a single block (e.g. huge table or long paragraph) exceeds upper bound on its own
		if blockWords > chunkUpperBoundWords && currentWordCount == 0 {
			// For large code blocks or tables, keep them intact if possible; otherwise fall back to paragraph slicing
			if block.Type == BlockCode || block.Type == BlockTable {
				chunks = append(chunks, blockText)
				continue
			}
			// Slicing long text block
			subChunks := SplitPageIntoChunks(blockText, targetWords)
			chunks = append(chunks, subChunks...)
			continue
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(blockText)
		currentWordCount += blockWords
	}

	flushCurrent()
	return chunks
}

// parseMarkdownBlocks parses lines into coherent blocks (code, table, heading, paragraph).
func parseMarkdownBlocks(text string) []MarkdownBlock {
	lines := strings.Split(text, "\n")
	blocks := make([]MarkdownBlock, 0)

	var currentLines []string
	currentType := BlockParagraph
	currentHeading := ""
	inCodeFence := false

	flushBlock := func() {
		if len(currentLines) == 0 {
			return
		}
		content := strings.Join(currentLines, "\n")
		if strings.TrimSpace(content) != "" {
			blocks = append(blocks, MarkdownBlock{
				Type:    currentType,
				Content: content,
				Heading: currentHeading,
			})
		}
		currentLines = nil
		currentType = BlockParagraph
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for code fence start/end
		if strings.HasPrefix(trimmed, "```") {
			if inCodeFence {
				// Closing code fence
				currentLines = append(currentLines, line)
				inCodeFence = false
				flushBlock()
				continue
			} else {
				// Opening code fence
				flushBlock()
				inCodeFence = true
				currentType = BlockCode
				currentLines = append(currentLines, line)
				continue
			}
		}

		if inCodeFence {
			currentLines = append(currentLines, line)
			continue
		}

		// Check for Markdown Headings (#, ##, ###)
		if strings.HasPrefix(trimmed, "#") {
			flushBlock()
			currentHeading = strings.TrimLeft(trimmed, "# ")
			blocks = append(blocks, MarkdownBlock{
				Type:    BlockHeading,
				Content: trimmed,
				Heading: currentHeading,
			})
			continue
		}

		// Check for Table rows (| ... |)
		if isTableRow(trimmed) {
			if currentType != BlockTable {
				flushBlock()
				currentType = BlockTable
			}
			currentLines = append(currentLines, line)
			continue
		}

		// Empty line separates paragraphs
		if trimmed == "" {
			if currentType == BlockTable {
				flushBlock()
			} else if len(currentLines) > 0 {
				flushBlock()
			}
			continue
		}

		// Regular text lines
		if currentType == BlockTable {
			flushBlock()
		}
		currentLines = append(currentLines, line)
	}

	flushBlock()
	return blocks
}

func isTableRow(line string) bool {
	return strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") && len(line) >= 2
}

// ExtractSyllabusChaptersFromMarkdown parses headings from markdown into SyllabusChapterDraft items.
func ExtractSyllabusChaptersFromMarkdown(markdownText string, totalPages int) []models.SyllabusChapterDraft {
	lines := strings.Split(markdownText, "\n")
	headings := make([]string, 0)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "##") {
			h := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			if h != "" {
				headings = append(headings, h)
			}
		}
	}

	// If no H1 headings found, try H2 headings
	if len(headings) == 0 {
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "###") {
				h := strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
				if h != "" {
					headings = append(headings, h)
				}
			}
		}
	}

	if len(headings) == 0 {
		return nil
	}

	if totalPages <= 0 {
		totalPages = len(headings)
	}

	chapters := make([]models.SyllabusChapterDraft, 0, len(headings))
	if len(headings) >= totalPages {
		// ponytail: 1 page per chapter clamped to totalPages
		for i, h := range headings {
			p := i + 1
			if p > totalPages {
				p = totalPages
			}
			chapters = append(chapters, models.SyllabusChapterDraft{
				Title:     h,
				StartPage: p,
				EndPage:   p,
			})
		}
		return chapters
	}

	pagesPerChapter := totalPages / len(headings)
	if pagesPerChapter < 1 {
		pagesPerChapter = 1
	}

	for i, h := range headings {
		startP := i*pagesPerChapter + 1
		endP := (i + 1) * pagesPerChapter
		if i == len(headings)-1 || endP > totalPages {
			endP = totalPages
		}
		if startP > totalPages {
			startP = totalPages
		}
		if endP < startP {
			endP = startP
		}

		chapters = append(chapters, models.SyllabusChapterDraft{
			Title:     h,
			StartPage: startP,
			EndPage:   endP,
		})
	}

	return chapters
}

// BuildBreadcrumbText prepends section heading metadata to chunk text for vector similarity embeddings.
func BuildBreadcrumbText(heading string, chunkText string) string {
	heading = strings.TrimSpace(heading)
	chunkText = embeddings.NormalizeWhitespace(chunkText)
	if heading == "" {
		return chunkText
	}
	if strings.HasPrefix(chunkText, "#") {
		return chunkText
	}
	return "[" + heading + "]\n" + chunkText
}
