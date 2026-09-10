package notebook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ai-tutor/internal/extension"
)

var deepPDFPageMarker = regexp.MustCompile(`(?m)^<!--\s*page:\s*(\d+)\s*-->\s*$`)

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
		// ponytail: no artificial timeout ceiling
		ctx = context.Background()
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

	sections := splitDeepPDFMarkdown(res.Markdown)
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
				PageNum: sec.PageNum,
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

func splitDeepPDFMarkdown(content string) []ExtractedSection {
	matches := deepPDFPageMarker.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		fallback := splitMarkdownByHeadings(content)
		sections := make([]ExtractedSection, 0, len(fallback))
		for i, sec := range fallback {
			sections = append(sections, ExtractedSection{Heading: sec.Heading, Text: stripMarkdownHeadings(sec.Text), PageNum: i + 1})
		}
		return sections
	}
	sections := make([]ExtractedSection, 0, len(matches))
	for i, match := range matches {
		end := len(content)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		page := 0
		_, _ = fmt.Sscanf(content[match[2]:match[3]], "%d", &page)
		sections = append(sections, ExtractedSection{Text: stripMarkdownHeadings(strings.TrimSpace(content[match[1]:end])), PageNum: page})
	}
	return sections
}

func stripMarkdownHeadings(content string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
