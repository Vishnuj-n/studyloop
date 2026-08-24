package app

import (
	"strings"
	"testing"

	"ai-tutor/internal/extension"
)

func TestListExtensions_Uninitialized(t *testing.T) {
	a := &App{}
	_, err := a.ListExtensions()
	if err == nil {
		t.Fatal("expected error when extManager is uninitialized, got nil")
	}
}

func TestListExtensions_WithInitError(t *testing.T) {
	a := &App{
		extInitError: "directory permission denied",
	}
	_, err := a.ListExtensions()
	if err == nil {
		t.Fatal("expected error when extInitError is set, got nil")
	}
	if !strings.Contains(err.Error(), "directory permission denied") {
		t.Fatalf("expected error to contain init error message, got %v", err)
	}
}

func TestListExtensions_WithManager(t *testing.T) {
	tempDir := t.TempDir()
	mgr := extension.NewManager(tempDir)
	a := &App{
		extManager: mgr,
	}

	exts, err := a.ListExtensions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exts) != 0 {
		t.Fatalf("expected 0 extensions, got %d", len(exts))
	}
}
