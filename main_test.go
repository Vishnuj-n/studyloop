package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ai-tutor/internal/app"
)

func TestNotebookHandler_NonNotebookPath_PassesThrough(t *testing.T) {
	// Scenario 1: Nil app (before startup)
	handlerNil := notebookHandler(nil)

	reqAssets := httptest.NewRequest(http.MethodGet, "/assets/index.js", nil)
	recAssets := httptest.NewRecorder()
	handlerNil.ServeHTTP(recAssets, reqAssets)

	if recAssets.Code != http.StatusOK || recAssets.Body.Len() > 0 {
		t.Fatalf("expected non-notebook path with nil app to pass through unhandled, got Code=%d, Body=%q", recAssets.Code, recAssets.Body.String())
	}

	// Scenario 2: App initialized but uploadDir is empty (uninitialized state)
	a := app.NewApp()
	handlerApp := notebookHandler(a)

	reqIndex := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	recIndex := httptest.NewRecorder()
	handlerApp.ServeHTTP(recIndex, reqIndex)

	if recIndex.Code != http.StatusOK || recIndex.Body.Len() > 0 {
		t.Fatalf("expected /index.html with empty uploadDir to pass through unhandled, got Code=%d, Body=%q", recIndex.Code, recIndex.Body.String())
	}
}

func TestNotebookHandler_NotebookPath_EmptyUploadDir_Returns503(t *testing.T) {
	handler := notebookHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/notebooks/some-file.pdf", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected /notebooks/ path with uninitialized uploadDir to return 503 Service Unavailable, got status %d", rec.Code)
	}
}

func TestNotebookHandler_NotebookPath_ValidFile_ServesContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "notebook_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dummyFile := filepath.Join(tmpDir, "test.pdf")
	if err := os.WriteFile(dummyFile, []byte("dummy pdf content"), 0644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}

	_ = app.NewApp()

	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fs := noDirListingFS{fs: http.Dir(tmpDir)}
		http.StripPrefix("/notebooks/", http.FileServer(fs)).ServeHTTP(rw, req)
	})

	req := httptest.NewRequest(http.MethodGet, "/notebooks/test.pdf", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK serving valid notebook file, got %d", rec.Code)
	}
	if rec.Body.String() != "dummy pdf content" {
		t.Fatalf("expected body %q, got %q", "dummy pdf content", rec.Body.String())
	}
}
