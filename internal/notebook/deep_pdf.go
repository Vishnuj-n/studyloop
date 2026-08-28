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

// DeepPDFIngestResult holds the output from extensions/deep_pdf/ingest.py (PyMuPDF4LLM).
type DeepPDFIngestResult struct {
	Title     string `json:"title"`
	PageCount int    `json:"page_count"`
	Markdown  string `json:"markdown"`
	WordCount int    `json:"word_count"`
	Error     string `json:"error,omitempty"`
}

// DeepPDFProgress represents a progress event from the deep_pdf Python worker.
type DeepPDFProgress struct {
	Type      string `json:"type"`
	Processed int    `json:"processed"`
	Total     int    `json:"total"`
	Percent   int    `json:"percent"`
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
}

// IngestDeepPDF runs the deep_pdf Pro extension to rapidly convert a PDF into structured Markdown.
func (s *Service) IngestDeepPDF(ctx context.Context, filePath string, runner *extension.Runner, ext *extension.Extension) (*ExtractedDocument, *DeepPDFIngestResult, error) {
	return s.IngestDeepPDFWithProgress(ctx, filePath, runner, ext, nil)
}

// IngestDeepPDFWithProgress runs the deep_pdf Pro extension with live progress callbacks.
func (s *Service) IngestDeepPDFWithProgress(ctx context.Context, filePath string, runner *extension.Runner, ext *extension.Extension, onProgress func(processed, total, percent int, message string)) (*ExtractedDocument, *DeepPDFIngestResult, error) {
	cleanPath := strings.TrimSpace(filePath)
	if cleanPath == "" {
		return nil, nil, fmt.Errorf("pdf file path cannot be empty")
	}

	if _, err := os.Stat(cleanPath); err != nil {
		return nil, nil, fmt.Errorf("pdf file not found: %w", err)
	}

	if ext == nil {
		extDir := extension.ResolveExtensionsDir("")
		ext = &extension.Extension{
			Manifest: extension.Manifest{
				ID:         "deep_pdf",
				Name:       "Deep Structured PDF Parser",
				Runtime:    "python",
				Entrypoint: "ingest.py",
				Tier:       "pro",
			},
			Dir: filepath.Join(extDir, "deep_pdf"),
		}
	}

	if runner == nil {
		runner = extension.NewRunner()
	}

	pythonPath, err := extension.FindExtensionPython(ext)
	if err != nil {
		return nil, nil, fmt.Errorf("python runtime is required for deep_pdf ingestion: %w", err)
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
	}

	var res DeepPDFIngestResult
	var lastErr string

	onLine := func(line string) error {
		line = strings.TrimSpace(line)
		if line == "" {
			return nil
		}

		var prog DeepPDFProgress
		if pErr := json.Unmarshal([]byte(line), &prog); pErr == nil {
			if prog.Type == "progress" {
				if onProgress != nil {
					onProgress(prog.Processed, prog.Total, prog.Percent, prog.Message)
				}
				return nil
			}
			if prog.Error != "" {
				lastErr = prog.Error
			}
		}

		var r DeepPDFIngestResult
		if rErr := json.Unmarshal([]byte(line), &r); rErr == nil && (r.Markdown != "" || r.Error != "") {
			res = r
			if r.Error != "" {
				lastErr = r.Error
			}
		}
		return nil
	}

	err = runner.RunStreamWithInput(ctx, ext.Dir, pythonPath, nil, onLine, ext.EntrypointPath(), cleanPath)
	if err != nil && strings.TrimSpace(res.Markdown) == "" {
		if lastErr != "" {
			return nil, nil, fmt.Errorf("%s", lastErr)
		}
		return nil, nil, fmt.Errorf("deep_pdf extension execution failed: %w", err)
	}

	if res.Error != "" {
		return nil, nil, fmt.Errorf("%s", res.Error)
	}

	if strings.TrimSpace(res.Markdown) == "" {
		return nil, nil, fmt.Errorf("deep_pdf extracted no readable content")
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
		Title:      filepath.Base(cleanPath),
		PageCount:  pageCount,
		WordCount:  wordCount,
		IsMarkdown: true,
		Sections:   docSections,
	}

	return doc, &res, nil
}
