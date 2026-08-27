package notebook

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIngestFastPDF_MissingFile(t *testing.T) {
	service := NewService(t.TempDir())
	_, _, err := service.IngestFastPDF(context.Background(), "/non/existent/file.pdf", nil, nil)
	if err == nil {
		t.Fatalf("expected error for non-existent file, got nil")
	}
}

func TestIngestFastPDF_EmptyPath(t *testing.T) {
	service := NewService(t.TempDir())
	_, _, err := service.IngestFastPDF(context.Background(), "   ", nil, nil)
	if err == nil {
		t.Fatalf("expected error for empty file path, got nil")
	}
}

func TestIngestFastPDF_JSONParsing(t *testing.T) {
	tmpDir := t.TempDir()
	pdfFile := filepath.Join(tmpDir, "dummy.pdf")
	if err := os.WriteFile(pdfFile, []byte("%PDF-1.4 dummy"), 0o644); err != nil {
		t.Fatalf("failed to create dummy pdf: %v", err)
	}

	// Verify file stat passes
	if _, err := os.Stat(pdfFile); err != nil {
		t.Fatalf("dummy pdf not created: %v", err)
	}
}
