package notebook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-tutor/internal/extension"
)

// FastPDFIngestResult holds the output from extensions/fast_pdf/ingest.py (PyMuPDF4LLM).
type FastPDFIngestResult struct {
	Title     string `json:"title"`
	PageCount int    `json:"page_count"`
	Markdown  string `json:"markdown"`
	WordCount int    `json:"word_count"`
	Error     string `json:"error,omitempty"`
}

// IngestFastPDF runs the fast_pdf Pro extension to rapidly convert a PDF into structured Markdown.
func (s *Service) IngestFastPDF(ctx context.Context, filePath string, runner *extension.Runner, ext *extension.Extension) (*ExtractedDocument, *FastPDFIngestResult, error) {
	cleanPath := strings.TrimSpace(filePath)
	if cleanPath == "" {
		return nil, nil, fmt.Errorf("pdf file path cannot be empty")
	}

	if _, err := os.Stat(cleanPath); err != nil {
		return nil, nil, fmt.Errorf("pdf file not found: %w", err)
	}

	if runner == nil {
		runner = extension.NewRunner()
	}

	entrypoint := "ingest.py"
	if ext != nil && ext.Entrypoint() != "" {
		entrypoint = ext.Entrypoint()
	} else {
		extDir := extension.ResolveExtensionsDir("")
		ext = &extension.Extension{
			Manifest: extension.Manifest{
				ID:         "fast_pdf",
				Name:       "Deep Structured PDF Parser",
				Runtime:    "python",
				Entrypoint: "ingest.py",
				Tier:       "pro",
			},
			Dir: filepath.Join(extDir, "fast_pdf"),
		}
	}

	pythonPath, err := extension.FindExtensionPython(ext)
	if err != nil {
		return nil, nil, fmt.Errorf("python runtime is required for fast_pdf ingestion: %w", err)
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
	}

	outputBytes, err := runner.Run(ctx, ext, pythonPath, entrypoint, cleanPath)
	if err != nil && len(outputBytes) == 0 {
		return nil, nil, fmt.Errorf("fast_pdf extension execution failed: %w", err)
	}

	// Extract json string (in case there's any stdout prelude)
	var res FastPDFIngestResult
	trimmedOutput := strings.TrimSpace(string(outputBytes))
	lastOpenBrace := strings.LastIndex(trimmedOutput, "{")
	if lastOpenBrace >= 0 {
		trimmedOutput = trimmedOutput[lastOpenBrace:]
	}

	if err := json.Unmarshal([]byte(trimmedOutput), &res); err != nil {
		return nil, nil, fmt.Errorf("invalid fast_pdf JSON output: %w (raw: %s)", err, string(outputBytes))
	}

	if res.Error != "" {
		return nil, nil, fmt.Errorf("%s", res.Error)
	}

	if strings.TrimSpace(res.Markdown) == "" {
		return nil, nil, fmt.Errorf("fast_pdf extracted no readable content")
	}

	sections := splitMarkdownByHeadings(res.Markdown)
	docSections := make([]ExtractedSection, 0, len(sections))

	if len(sections) == 0 {
		docSections = append(docSections, ExtractedSection{
			Heading: "Document",
			Text:    res.Markdown,
			PageNum: 1,
		})
	} else {
		for i, sec := range sections {
			docSections = append(docSections, ExtractedSection{
				Heading: sec.Heading,
				Text:    sec.Text,
				PageNum: i + 1,
			})
		}
	}

	pageCount := res.PageCount
	if pageCount <= 0 {
		pageCount = len(docSections)
	}

	wordCount := res.WordCount
	if wordCount <= 0 {
		wordCount = len(strings.Fields(res.Markdown))
	}

	doc := &ExtractedDocument{
		Title:     filepath.Base(cleanPath),
		PageCount: pageCount,
		WordCount: wordCount,
		Sections:  docSections,
	}

	return doc, &res, nil
}
