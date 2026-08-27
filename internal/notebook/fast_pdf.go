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

// FastPDFProgress represents a progress event from the fast_pdf Python worker.
type FastPDFProgress struct {
	Type      string `json:"type"`
	Processed int    `json:"processed"`
	Total     int    `json:"total"`
	Percent   int    `json:"percent"`
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
}

// IngestFastPDF runs the fast_pdf Pro extension to rapidly convert a PDF into structured Markdown.
func (s *Service) IngestFastPDF(ctx context.Context, filePath string, runner *extension.Runner, ext *extension.Extension) (*ExtractedDocument, *FastPDFIngestResult, error) {
	return s.IngestFastPDFWithProgress(ctx, filePath, runner, ext, nil)
}

// IngestFastPDFWithProgress runs the fast_pdf Pro extension with live progress callbacks.
func (s *Service) IngestFastPDFWithProgress(ctx context.Context, filePath string, runner *extension.Runner, ext *extension.Extension, onProgress func(processed, total, percent int, message string)) (*ExtractedDocument, *FastPDFIngestResult, error) {
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
				ID:         "fast_pdf",
				Name:       "Deep Structured PDF Parser",
				Runtime:    "python",
				Entrypoint: "ingest.py",
				Tier:       "pro",
			},
			Dir: filepath.Join(extDir, "fast_pdf"),
		}
	}

	if runner == nil {
		runner = extension.NewRunner()
	}

	pythonPath, err := extension.FindExtensionPython(ext)
	if err != nil {
		return nil, nil, fmt.Errorf("python runtime is required for fast_pdf ingestion: %w", err)
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
	}

	var res FastPDFIngestResult
	var lastErr string

	onLine := func(line string) error {
		line = strings.TrimSpace(line)
		if line == "" {
			return nil
		}

		var prog FastPDFProgress
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

		var r FastPDFIngestResult
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
		return nil, nil, fmt.Errorf("fast_pdf extension execution failed: %w", err)
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
