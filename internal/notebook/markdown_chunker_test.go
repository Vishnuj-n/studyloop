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
}

func TestBuildBreadcrumbText(t *testing.T) {
	res := BuildBreadcrumbText("Memory Management", "Stack allocation happens quickly.")
	expected := "[Memory Management]\nStack allocation happens quickly."
	if res != expected {
		t.Errorf("expected %q, got %q", expected, res)
	}

	// Should not double-prepend if already a heading
	res2 := BuildBreadcrumbText("Memory Management", "# Memory Management\nStack allocation")
	if !strings.HasPrefix(res2, "# Memory Management") {
		t.Errorf("expected heading to be preserved, got %q", res2)
	}
}
