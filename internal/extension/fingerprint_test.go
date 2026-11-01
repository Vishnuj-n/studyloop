package extension

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseMajorVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		valid    bool
	}{
		{"1.0.1", 1, true},
		{"v2.4.0", 2, true},
		{"3.0", 3, true},
		{"10.2.1-beta", 10, true},
		{"", 0, false},
		{"alpha", 0, false},
	}

	for _, tt := range tests {
		major, ok := ParseMajorVersion(tt.input)
		if ok != tt.valid {
			t.Errorf("ParseMajorVersion(%q) valid = %v, expected %v", tt.input, ok, tt.valid)
		}
		if ok && major != tt.expected {
			t.Errorf("ParseMajorVersion(%q) = %d, expected %d", tt.input, major, tt.expected)
		}
	}
}

func TestShouldRecreateVenv(t *testing.T) {
	tests := []struct {
		oldVer   string
		newVer   string
		recreate bool
	}{
		{"1.0.0", "1.0.1", false},
		{"1.0.1", "1.2.0", false},
		{"1.0.1", "2.0.0", true},
		{"2.0.0", "1.9.0", true},
		{"v1.0.0", "v1.0.1", false},
		{"v1.0.0", "v2.0.0", true},
		{"1.0.0", "1.0.0", false},
	}

	for _, tt := range tests {
		got := ShouldRecreateVenv(tt.oldVer, tt.newVer)
		if got != tt.recreate {
			t.Errorf("ShouldRecreateVenv(%q, %q) = %v, expected %v", tt.oldVer, tt.newVer, got, tt.recreate)
		}
	}
}

func TestFingerprintLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	extDir := filepath.Join(tempDir, "ext")
	venvDir := filepath.Join(tempDir, "venv")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatalf("failed to create extDir: %v", err)
	}
	if err := os.MkdirAll(venvDir, 0755); err != nil {
		t.Fatalf("failed to create venvDir: %v", err)
	}

	// Create fake python interpreter so GetVenvPython succeeds
	pyPath := GetVenvPython(venvDir)
	if err := os.MkdirAll(filepath.Dir(pyPath), 0755); err != nil {
		t.Fatalf("failed to create pyPath dir: %v", err)
	}
	if err := os.WriteFile(pyPath, []byte("fake-python"), 0755); err != nil {
		t.Fatalf("failed to write fake python: %v", err)
	}

	// Write manifest and requirements
	manifestData := Manifest{
		ID:         "test_ext",
		Name:       "Test Extension",
		Version:    "1.0.0",
		Runtime:    "python",
		Entrypoint: "main.py",
	}
	manifestBytes, _ := json.Marshal(manifestData)
	_ = os.WriteFile(filepath.Join(extDir, "manifest.json"), manifestBytes, 0644)
	_ = os.WriteFile(filepath.Join(extDir, "requirements.txt"), []byte("foo==1.0.0\n"), 0644)

	ext := &Extension{
		Manifest: manifestData,
		Dir:      extDir,
	}

	// 1. Initial state: no fingerprint in venv -> should be stale
	stale, isMajor, _ := IsExtensionVenvStale(ext, venvDir)
	if !stale {
		t.Errorf("expected initial state without fingerprint to be stale")
	}
	if isMajor {
		t.Errorf("initial missing fingerprint should not flag major change")
	}

	// 2. Write fingerprint matching current state
	fp := ComputeFingerprint(ext)
	if fp.ManifestVersion != "1.0.0" {
		t.Errorf("expected manifest version 1.0.0, got %s", fp.ManifestVersion)
	}
	if fp.RequirementsSHA == "" {
		t.Errorf("expected non-empty requirements sha")
	}
	if err := WriteFingerprint(venvDir, fp); err != nil {
		t.Fatalf("WriteFingerprint failed: %v", err)
	}

	// Now it should be valid (not stale)
	stale, _, _ = IsExtensionVenvStale(ext, venvDir)
	if stale {
		t.Errorf("expected venv to be fresh after writing matching fingerprint")
	}

	// 3. Minor version update in manifest (1.0.0 -> 1.1.0)
	ext.Manifest.Version = "1.1.0"
	stale, isMajor, reason := IsExtensionVenvStale(ext, venvDir)
	if !stale {
		t.Errorf("expected venv to be stale after minor version bump")
	}
	if isMajor {
		t.Errorf("minor version bump should not be flagged as major change: %s", reason)
	}

	// 4. Major version update in manifest (1.0.0 -> 2.0.0)
	ext.Manifest.Version = "2.0.0"
	stale, isMajor, _ = IsExtensionVenvStale(ext, venvDir)
	if !stale {
		t.Errorf("expected venv to be stale after major version bump")
	}
	if !isMajor {
		t.Errorf("major version bump 1.0.0 -> 2.0.0 MUST be flagged as major change")
	}

	// 5. Revert version back to 1.0.0, but edit requirements.txt
	ext.Manifest.Version = "1.0.0"
	_ = os.WriteFile(filepath.Join(extDir, "requirements.txt"), []byte("foo==2.0.0\n"), 0644)
	stale, isMajor, _ = IsExtensionVenvStale(ext, venvDir)
	if !stale {
		t.Errorf("expected venv to be stale after modifying requirements.txt")
	}
	if isMajor {
		t.Errorf("modifying requirements.txt should not be flagged as major change")
	}
}
