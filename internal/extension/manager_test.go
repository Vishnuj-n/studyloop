package extension

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidManifestDiscovery(t *testing.T) {
	tempDir := t.TempDir()

	ext1Dir := filepath.Join(tempDir, "ext-one")
	if err := os.MkdirAll(ext1Dir, 0o755); err != nil {
		t.Fatalf("failed to create ext1 dir: %v", err)
	}
	manifest1 := `{
		"id": "ext-one",
		"name": "Extension One",
		"version": "1.0.0",
		"runtime": "python",
		"entrypoint": "main.py"
	}`
	if err := os.WriteFile(filepath.Join(ext1Dir, "manifest.json"), []byte(manifest1), 0o644); err != nil {
		t.Fatalf("failed to write manifest1: %v", err)
	}

	ext2Dir := filepath.Join(tempDir, "ext-two")
	if err := os.MkdirAll(ext2Dir, 0o755); err != nil {
		t.Fatalf("failed to create ext2 dir: %v", err)
	}
	manifest2 := `{
		"id": "ext-two",
		"name": "Extension Two",
		"version": "2.1.0",
		"runtime": "binary",
		"entrypoint": "bin/run.exe"
	}`
	if err := os.WriteFile(filepath.Join(ext2Dir, "manifest.json"), []byte(manifest2), 0o644); err != nil {
		t.Fatalf("failed to write manifest2: %v", err)
	}

	mgr := NewManager(tempDir)
	list, err := mgr.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 extensions, got %d", len(list))
	}

	if list[0].ID() != "ext-one" || list[1].ID() != "ext-two" {
		t.Errorf("unexpected extension IDs in list: %v, %v", list[0].ID(), list[1].ID())
	}

	ext1, ok := mgr.Get("ext-one")
	if !ok || ext1 == nil {
		t.Fatalf("expected to find ext-one by ID")
	}
	if ext1.Name() != "Extension One" {
		t.Errorf("expected name 'Extension One', got %q", ext1.Name())
	}
	if ext1.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", ext1.Version())
	}
	if ext1.Runtime() != "python" {
		t.Errorf("expected runtime 'python', got %q", ext1.Runtime())
	}
	if ext1.Entrypoint() != "main.py" {
		t.Errorf("expected entrypoint 'main.py', got %q", ext1.Entrypoint())
	}
	expectedEntrypointPath := filepath.Join(ext1Dir, "main.py")
	if ext1.EntrypointPath() != expectedEntrypointPath {
		t.Errorf("expected EntrypointPath %q, got %q", expectedEntrypointPath, ext1.EntrypointPath())
	}
}

func TestMissingExtensionsDir(t *testing.T) {
	nonExistentDir := filepath.Join(t.TempDir(), "does_not_exist")
	mgr := NewManager(nonExistentDir)

	list, err := mgr.Discover()
	if err != nil {
		t.Fatalf("Discover on missing directory returned error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 extensions for missing directory, got %d", len(list))
	}
}

func TestInvalidManifestHandling(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Directory with invalid JSON
	badJSONDir := filepath.Join(tempDir, "bad-json")
	_ = os.MkdirAll(badJSONDir, 0o755)
	_ = os.WriteFile(filepath.Join(badJSONDir, "manifest.json"), []byte(`{ invalid json`), 0o644)

	// 2. Directory missing required ID
	missingIDDir := filepath.Join(tempDir, "missing-id")
	_ = os.MkdirAll(missingIDDir, 0o755)
	_ = os.WriteFile(filepath.Join(missingIDDir, "manifest.json"), []byte(`{"name":"No ID","version":"1.0.0","runtime":"python","entrypoint":"app.py"}`), 0o644)

	// 3. Directory with path traversal in ID
	traversalIDDir := filepath.Join(tempDir, "traversal-id")
	_ = os.MkdirAll(traversalIDDir, 0o755)
	_ = os.WriteFile(filepath.Join(traversalIDDir, "manifest.json"), []byte(`{"id":"../evil","name":"Evil","version":"1.0.0","runtime":"python","entrypoint":"app.py"}`), 0o644)

	// 4. Directory with path traversal in entrypoint
	traversalEntryDir := filepath.Join(tempDir, "traversal-entry")
	_ = os.MkdirAll(traversalEntryDir, 0o755)
	_ = os.WriteFile(filepath.Join(traversalEntryDir, "manifest.json"), []byte(`{"id":"traversal","name":"Traversal","version":"1.0.0","runtime":"python","entrypoint":"../../etc/passwd"}`), 0o644)

	// 5. Valid directory
	validDir := filepath.Join(tempDir, "valid-ext")
	_ = os.MkdirAll(validDir, 0o755)
	_ = os.WriteFile(filepath.Join(validDir, "manifest.json"), []byte(`{"id":"valid-ext","name":"Valid","version":"0.1.0","runtime":"python","entrypoint":"run.py"}`), 0o644)

	// 6. Regular file in root directory (not a subdirectory)
	_ = os.WriteFile(filepath.Join(tempDir, "stray.txt"), []byte("stray"), 0o644)

	mgr := NewManager(tempDir)
	list, err := mgr.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("expected exactly 1 valid extension, got %d", len(list))
	}
	if list[0].ID() != "valid-ext" {
		t.Errorf("expected 'valid-ext', got %q", list[0].ID())
	}
}

func TestManifestValidation(t *testing.T) {
	cases := []struct {
		name      string
		manifest  Manifest
		shouldErr bool
	}{
		{"empty id", Manifest{ID: "", Name: "A", Version: "1.0", Runtime: "python", Entrypoint: "main.py"}, true},
		{"empty name", Manifest{ID: "valid", Name: "", Version: "1.0", Runtime: "python", Entrypoint: "main.py"}, true},
		{"empty version", Manifest{ID: "valid", Name: "A", Version: "", Runtime: "python", Entrypoint: "main.py"}, true},
		{"empty runtime", Manifest{ID: "valid", Name: "A", Version: "1.0", Runtime: "", Entrypoint: "main.py"}, true},
		{"empty entrypoint", Manifest{ID: "valid", Name: "A", Version: "1.0", Runtime: "python", Entrypoint: ""}, true},
		{"slash in id", Manifest{ID: "bad/id", Name: "A", Version: "1.0", Runtime: "python", Entrypoint: "main.py"}, true},
		{"backslash in id", Manifest{ID: "bad\\id", Name: "A", Version: "1.0", Runtime: "python", Entrypoint: "main.py"}, true},
		{"valid manifest", Manifest{ID: "clean-id_1", Name: "Good Extension", Version: "0.1.0", Runtime: "python", Entrypoint: "main.py"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.manifest.Validate()
			if tc.shouldErr && err == nil {
				t.Errorf("expected validation error for case %q, got nil", tc.name)
			}
			if !tc.shouldErr && err != nil {
				t.Errorf("unexpected error for case %q: %v", tc.name, err)
			}
		})
	}
}
