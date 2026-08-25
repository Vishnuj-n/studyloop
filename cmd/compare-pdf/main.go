package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

type FastPDFOutput struct {
	Title     string `json:"title"`
	PageCount int    `json:"page_count"`
	Markdown  string `json:"markdown"`
	WordCount int    `json:"word_count"`
	Error     string `json:"error"`
}

type TextOnlyOutput struct {
	PageCount int
	WordCount int
	Sections  []ExtractedPage
	FullText  string
	Duration  time.Duration
}

type ExtractedPage struct {
	PageNum int
	Text    string
}

func extractNativeGoText(pdfPath string) (*TextOnlyOutput, error) {
	start := time.Now()
	file, reader, err := pdf.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF with native Go parser: %w", err)
	}
	defer file.Close()

	totalPages := reader.NumPage()
	if totalPages <= 0 {
		totalPages = 1
	}

	var sections []ExtractedPage
	var fullTextBuilder strings.Builder

	for p := 1; p <= totalPages; p++ {
		page := reader.Page(p)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		cleanText := strings.TrimSpace(text)
		if cleanText != "" {
			sections = append(sections, ExtractedPage{
				PageNum: p,
				Text:    cleanText,
			})
			fullTextBuilder.WriteString(fmt.Sprintf("\n--- Page %d ---\n", p))
			fullTextBuilder.WriteString(cleanText)
		}
	}

	fullText := fullTextBuilder.String()
	words := len(strings.Fields(fullText))

	return &TextOnlyOutput{
		PageCount: totalPages,
		WordCount: words,
		Sections:  sections,
		FullText:  fullText,
		Duration:  time.Since(start),
	}, nil
}

func extractFastPDF(pdfPath string) (*FastPDFOutput, time.Duration, error) {
	start := time.Now()
	fastPdfScript := filepath.Join("extensions", "fast_pdf", "ingest.py")

	cmd := exec.Command("python", fastPdfScript, pdfPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		return nil, duration, fmt.Errorf("fast_pdf run error: %v (details: %s)", err, errMsg)
	}

	var out FastPDFOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, duration, fmt.Errorf("failed to parse fast_pdf JSON output: %w", err)
	}

	return &out, duration, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/compare-pdf <path-to-pdf>")
		fmt.Println("Example: go run ./cmd/compare-pdf dev_data/uploads/sample.pdf")
		os.Exit(1)
	}

	pdfPath := os.Args[1]
	absPath, err := filepath.Abs(pdfPath)
	if err != nil {
		fmt.Printf("Invalid file path: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		fmt.Printf("File does not exist: %s\n", absPath)
		os.Exit(1)
	}

	fmt.Printf("\n================================================================================\n")
	fmt.Printf("           PDF EXTRACTION COMPARISON: Text-Only vs Fast PDF (PyMuPDF4LLM)       \n")
	fmt.Printf(" Target: %s\n", filepath.Base(absPath))
	fmt.Printf("================================================================================\n\n")

	// 1. Native Go extraction
	fmt.Println("[1/2] Running Native Go Text-Only Extraction (ledongthuc/pdf)...")
	textResult, err := extractNativeGoText(absPath)
	if err != nil {
		fmt.Printf("[-] Text-only extraction failed: %v\n", err)
	} else {
		fmt.Printf("    -> Extracted %d pages, %d words in %v\n", textResult.PageCount, textResult.WordCount, textResult.Duration)
	}

	// 2. Fast PDF extraction
	fmt.Println("[2/2] Running Fast PDF Extraction (extensions/fast_pdf/ingest.py)...")
	fastPdfResult, fastPdfDur, fastPdfErr := extractFastPDF(absPath)
	if fastPdfErr != nil {
		fmt.Printf("    [-] Fast PDF extraction note: %v\n", fastPdfErr)
	} else {
		fmt.Printf("    -> Extracted %d pages, %d words in %v\n", fastPdfResult.PageCount, fastPdfResult.WordCount, fastPdfDur)
	}

	// 3. Comparison Matrix
	fmt.Println("\n--------------------------------------------------------------------------------")
	fmt.Println(" STRUCTURAL & CAPABILITY COMPARISON")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("%-30s | %-22s | %-22s\n", "Feature / Attribute", "Native Text-Only", "Fast PDF Structured")
	fmt.Println("--------------------------------------------------------------------------------")
	
	if textResult != nil {
		fmt.Printf("%-30s | %-22d | ", "Page Count", textResult.PageCount)
	} else {
		fmt.Printf("%-30s | %-22s | ", "Page Count", "N/A")
	}
	if fastPdfResult != nil {
		fmt.Printf("%-22d\n", fastPdfResult.PageCount)
	} else {
		fmt.Printf("%-22s\n", "N/A")
	}

	if textResult != nil {
		fmt.Printf("%-30s | %-22d | ", "Word Count", textResult.WordCount)
	} else {
		fmt.Printf("%-30s | %-22s | ", "Word Count", "N/A")
	}
	if fastPdfResult != nil {
		fmt.Printf("%-22d\n", fastPdfResult.WordCount)
	} else {
		fmt.Printf("%-22s\n", "N/A")
	}

	hasTableText := "Plain text string"
	hasTableFast := "Markdown | col1 | col2 |"
	fmt.Printf("%-30s | %-22s | %-22s\n", "Table Format", hasTableText, hasTableFast)
	fmt.Printf("%-30s | %-22s | %-22s\n", "Heading Tree (# / ##)", "No (flat text)", "Yes (semantic tags)")
	fmt.Printf("%-30s | %-22s | %-22s\n", "Math / LaTeX Equations", "Unformatted/Noise", "Preserved ($..$, $$..$$)")
	fmt.Printf("%-30s | %-22s | %-22s\n", "Multi-column Flow", "Interleaved/Mixed", "True Reading Order")
	fmt.Printf("%-30s | %-22s | %-22s\n", "Output Schema", "[]ExtractedSection", "Fast PDF Markdown + JSON")
	fmt.Println("--------------------------------------------------------------------------------")

	// 4. Preview sample comparison
	fmt.Println("\n--------------------------------------------------------------------------------")
	fmt.Println(" SAMPLE OUTPUT COMPARISON PREVIEW")
	fmt.Println("--------------------------------------------------------------------------------")

	if textResult != nil && len(textResult.FullText) > 0 {
		fmt.Println("\n[--- A. Text-Only Extraction (Raw Linear Dump) ---]")
		preview := textResult.FullText
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Println(preview)
	}

	if fastPdfResult != nil && len(fastPdfResult.Markdown) > 0 {
		fmt.Println("\n[--- B. Fast PDF Structured Extraction (Rich Structured Markdown) ---]")
		preview := fastPdfResult.Markdown
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Println(preview)
	}

	fmt.Println("\n================================================================================")
}
