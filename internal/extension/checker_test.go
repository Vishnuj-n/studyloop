package extension

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveExtensionVenvDirAbsolute(t *testing.T) {
	tempDir := t.TempDir()
	ext := &Extension{
		Manifest: Manifest{
			ID:         "test_compressor",
			Name:       "Test Compressor",
			Version:    "1.0.0",
			Runtime:    "python",
			Entrypoint: "compress.py",
		},
		Dir: tempDir,
	}

	venvDir := ResolveExtensionVenvDir(ext)
	if !filepath.IsAbs(venvDir) {
		t.Errorf("expected venvDir to be absolute path, got: %s", venvDir)
	}

	pyPath := GetVenvPython(venvDir)
	if !filepath.IsAbs(pyPath) {
		t.Errorf("expected pyPath to be absolute path, got: %s", pyPath)
	}
}

func TestRunSmokeTestRelativePath(t *testing.T) {
	tempDir := t.TempDir()
	pyScript := filepath.Join(tempDir, "smoke_test.py")
	scriptContent := `import sys, json
if len(sys.argv) > 1 and sys.argv[1] == "--test":
    sys.stdout.write(json.dumps({"status": "ok"}) + "\n")
    sys.exit(0)
sys.exit(1)
`
	if err := os.WriteFile(pyScript, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("failed to write test python script: %v", err)
	}

	ext := &Extension{
		Manifest: Manifest{
			ID:         "test_smoke",
			Name:       "Test Smoke",
			Version:    "1.0.0",
			Runtime:    "python",
			Entrypoint: "smoke_test.py",
		},
		Dir: tempDir,
	}

	pyExe, err := FindPythonExecutable()
	if err != nil {
		t.Skip("skipping smoke test execution, python not installed on PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RunSmokeTest(ctx, ext, pyExe); err != nil {
		t.Fatalf("RunSmokeTest failed: %v", err)
	}
}
