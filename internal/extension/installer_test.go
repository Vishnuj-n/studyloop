package extension

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func createTestZip(t *testing.T, files map[string]string) string {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip file entry: %v", err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write content: %v", err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	tmpZip := filepath.Join(t.TempDir(), "test_ext.zip")
	if err := os.WriteFile(tmpZip, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("failed to write zip file: %v", err)
	}
	return tmpZip
}

func TestInstallAndUninstallZip(t *testing.T) {
	tempBase := t.TempDir()
	mgr := NewManager(tempBase)

	manifestJSON := `{
		"id": "my-plugin",
		"name": "My Plugin",
		"version": "1.0.0",
		"runtime": "python",
		"entrypoint": "main.py",
		"tier": "pro",
		"category": "study"
	}`

	zipPath := createTestZip(t, map[string]string{
		"my-plugin/manifest.json": manifestJSON,
		"my-plugin/main.py":       "print('hello world')",
	})

	// 1. Install
	ext, err := mgr.InstallZip(zipPath)
	if err != nil {
		t.Fatalf("InstallZip failed: %v", err)
	}
	if ext.ID() != "my-plugin" {
		t.Errorf("expected ID my-plugin, got %s", ext.ID())
	}
	if ext.Manifest.Tier != "pro" {
		t.Errorf("expected Tier pro, got %s", ext.Manifest.Tier)
	}

	// 2. Discover / Get
	fetched, ok := mgr.Get("my-plugin")
	if !ok || fetched == nil {
		t.Fatalf("expected extension to be registered")
	}

	// 3. Uninstall
	if err := mgr.Uninstall("my-plugin"); err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	if _, ok := mgr.Get("my-plugin"); ok {
		t.Errorf("expected extension to be removed from manager")
	}
}
