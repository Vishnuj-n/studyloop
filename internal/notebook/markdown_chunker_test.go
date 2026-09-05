package notebook

import (
	"strings"
	"testing"
)

func TestSplitMarkdownIntoChunks_PreservesTablesAndCode(t *testing.T) {
	mdContent := `# Chapter 1: Introduction to Go

Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.

| Type | Size (bytes) | Description |
|---|---|---|
| int8 | 1 | Signed 8-bit integer |
| int16 | 2 | Signed 16-bit integer |
| int32 | 4 | Signed 32-bit integer |
| int64 | 8 | Signed 64-bit integer |

Here is a simple Hello World program in Go:

` + "```go" + `
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
` + "```" + `

## Next Section: Concurrency

Concurrency is the composition of independently executing computations. Go provides goroutines and channels to facilitate concurrent execution safely.
`

	chunks := SplitMarkdownIntoChunks(mdContent, 50)
	if len(chunks) == 0 {
		t.Fatalf("expected chunks, got none")
	}

	// Verify table is intact in at least one chunk
	foundTable := false
	foundCode := false
	for _, chunk := range chunks {
		if strings.Contains(chunk, "| Type | Size") && strings.Contains(chunk, "| int64 | 8 |") {
			foundTable = true
		}
		if strings.Contains(chunk, "```go") && strings.Contains(chunk, "fmt.Println") {
			foundCode = true
		}
	}

	if !foundTable {
		t.Errorf("expected table to be kept intact within a chunk, chunks: %+v", chunks)
	}
	if !foundCode {
		t.Errorf("expected code block to be kept intact within a chunk, chunks: %+v", chunks)
	}
}

func TestExtractSyllabusChaptersFromMarkdown(t *testing.T) {
	md := `# Getting Started
Some overview text here.

# Architecture & Design
Details about system architecture.

# Concurrency Patterns
Goroutines, channels, and sync primitives.
`
	chapters := ExtractSyllabusChaptersFromMarkdown(md, 30)
	if len(chapters) != 3 {
		t.Fatalf("expected 3 chapters, got %d", len(chapters))
	}

	if chapters[0].Title != "Getting Started" {
		t.Errorf("unexpected chapter 0 title: %s", chapters[0].Title)
	}
	if chapters[1].Title != "Architecture & Design" {
		t.Errorf("unexpected chapter 1 title: %s", chapters[1].Title)
	}
	if chapters[2].Title != "Concurrency Patterns" {
		t.Errorf("unexpected chapter 2 title: %s", chapters[2].Title)
	}

	if chapters[0].StartPage != 1 || chapters[2].EndPage != 30 {
		t.Errorf("unexpected page ranges: %+v", chapters)
	}

	// Regression test: 3 headings across 2 pages
	chapters2 := ExtractSyllabusChaptersFromMarkdown(md, 2)
	if len(chapters2) != 3 {
		t.Fatalf("expected 3 chapters, got %d", len(chapters2))
	}
	if chapters2[0].StartPage != 1 || chapters2[0].EndPage != 1 {
		t.Errorf("expected chapter 0 to be on page 1, got %d-%d", chapters2[0].StartPage, chapters2[0].EndPage)
	}
	if chapters2[1].StartPage != 2 || chapters2[1].EndPage != 2 {
		t.Errorf("expected chapter 1 to be on page 2, got %d-%d", chapters2[1].StartPage, chapters2[1].EndPage)
	}
	if chapters2[2].StartPage != 2 || chapters2[2].EndPage != 2 {
		t.Errorf("expected chapter 2 to be clamped to page 2, got %d-%d", chapters2[2].StartPage, chapters2[2].EndPage)
	}
}

func TestSplitMarkdownIntoChunks_SpacedTableSeparators(t *testing.T) {
	spacedTableContent := `Overview of data structures in the program:

| Data Type | Memory (Bytes) | Alignment |
| --- | --- | --- |
| bool | 1 | 1 |
| int32 | 4 | 4 |
| float64 | 8 | 8 |

Summary follows after the table with detailed explanations.`

	chunks := SplitMarkdownIntoChunks(spacedTableContent, 30)
	if len(chunks) == 0 {
		t.Fatalf("expected chunks from spaced table markdown content, got none")
	}

	foundTable := false
	for _, c := range chunks {
		if strings.Contains(c, "| Data Type | Memory") && strings.Contains(c, "| float64 | 8 |") {
			foundTable = true
			break
		}
	}
	if !foundTable {
		t.Errorf("expected spaced table to be preserved intact in a chunk, chunks: %+v", chunks)
	}
}
