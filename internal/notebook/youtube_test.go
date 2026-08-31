package notebook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"ai-tutor/internal/extension"
)

// Helper to create an extension with an executable mock script
func createMockYouTubeExt(t *testing.T, outputJSON string, exitCode int) (*extension.Extension, string) {
	t.Helper()
	dir := t.TempDir()
	var scriptName string
	var scriptContent string

	// Create a python script that outputs JSON or exits with code
	scriptName = "ingest.py"
	if exitCode != 0 {
		scriptContent = fmt.Sprintf("import sys\nsys.stderr.write(%q)\nsys.exit(%d)\n", outputJSON, exitCode)
	} else {
		scriptContent = fmt.Sprintf("import sys\nsys.stdout.write(%q)\nsys.exit(0)\n", outputJSON)
	}

	scriptPath := filepath.Join(dir, scriptName)
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}

	ext := &extension.Extension{
		Manifest: extension.Manifest{
			ID:         "youtube",
			Name:       "YouTube Ingestion",
			Runtime:    "python",
			Entrypoint: scriptName,
		},
		Dir: dir,
	}

	pyExe, err := extension.FindPythonExecutable()
	if err != nil {
		t.Skip("skipping YouTube extension test: python executable not found")
	}

	return ext, pyExe
}

func TestIngestYouTubeVideo_EmptyURL(t *testing.T) {
	svc := NewService(t.TempDir())
	doc, res, err := svc.IngestYouTubeVideo(context.Background(), "", nil, nil, "", "")
	if err == nil {
		t.Fatalf("expected error for empty video URL, got nil")
	}
	if doc != nil || res != nil {
		t.Fatalf("expected nil results on error")
	}
}

func TestIngestYouTubeVideo_Success(t *testing.T) {
	mockResult := YouTubeIngestResult{
		Title:   "Test Video",
		VideoID: "abc12345",
	}
	mockResult.Chapters = []struct {
		ChapterIndex int    `json:"chapter_index"`
		Title        string `json:"title"`
		StartSeconds int    `json:"start_seconds"`
		EndSeconds   int    `json:"end_seconds"`
		Transcript   string `json:"transcript"`
	}{
		{
			Title:        "Introduction",
			StartSeconds: 0,
			EndSeconds:   60,
			Transcript:   "Hello and welcome to this video.",
			ChapterIndex: 0,
		},
		{
			Title:        "Main Concept",
			StartSeconds: 60,
			EndSeconds:   180,
			Transcript:   "Here is the core lesson content with more words to count.",
			ChapterIndex: 1,
		},
	}

	rawJSON, err := json.Marshal(mockResult)
	if err != nil {
		t.Fatalf("failed to marshal mock result: %v", err)
	}

	ext, _ := createMockYouTubeExt(t, string(rawJSON), 0)
	svc := NewService(t.TempDir())
	runner := extension.NewRunner()

	doc, res, err := svc.IngestYouTubeVideo(context.Background(), "https://youtube.com/watch?v=abc12345", runner, ext, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res == nil || res.Title != "Test Video" {
		t.Fatalf("expected result title 'Test Video', got %v", res)
	}

	if doc == nil || doc.PageCount != 2 {
		t.Fatalf("expected 2 pages in doc, got %v", doc)
	}

	if len(doc.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(doc.Sections))
	}

	// Verify 1-based page numbers
	if doc.Sections[0].PageNum != 1 {
		t.Errorf("expected section 0 PageNum to be 1, got %d", doc.Sections[0].PageNum)
	}
	if doc.Sections[1].PageNum != 2 {
		t.Errorf("expected section 1 PageNum to be 2, got %d", doc.Sections[1].PageNum)
	}

	if doc.WordCount <= 0 {
		t.Errorf("expected positive total word count, got %d", doc.WordCount)
	}
}

func TestIngestYouTubeVideo_ExtensionErrorPayload(t *testing.T) {
	mockResult := YouTubeIngestResult{
		Error: "Transcript is disabled for this video",
	}
	rawJSON, _ := json.Marshal(mockResult)

	ext, _ := createMockYouTubeExt(t, string(rawJSON), 0)
	svc := NewService(t.TempDir())
	runner := extension.NewRunner()

	_, _, err := svc.IngestYouTubeVideo(context.Background(), "https://youtube.com/watch?v=abc", runner, ext, "", "")
	if err == nil || err.Error() != "Transcript is disabled for this video" {
		t.Fatalf("expected extension error payload propagated, got: %v", err)
	}
}

func TestIngestYouTubeVideo_MalformedJSON(t *testing.T) {
	ext, _ := createMockYouTubeExt(t, "invalid json output", 0)
	svc := NewService(t.TempDir())
	runner := extension.NewRunner()

	_, _, err := svc.IngestYouTubeVideo(context.Background(), "https://youtube.com/watch?v=abc", runner, ext, "", "")
	if err == nil {
		t.Fatalf("expected error for malformed json, got nil")
	}
}

func TestIngestYouTubeVideo_EmptyChapters(t *testing.T) {
	mockResult := YouTubeIngestResult{
		Title:    "No Chapters",
		Chapters: nil,
	}
	rawJSON, _ := json.Marshal(mockResult)

	ext, _ := createMockYouTubeExt(t, string(rawJSON), 0)
	svc := NewService(t.TempDir())
	runner := extension.NewRunner()

	_, _, err := svc.IngestYouTubeVideo(context.Background(), "https://youtube.com/watch?v=abc", runner, ext, "", "")
	if err == nil {
		t.Fatalf("expected error for empty chapters, got nil")
	}
}
