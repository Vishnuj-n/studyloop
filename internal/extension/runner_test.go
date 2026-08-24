package extension

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunnerExecution(t *testing.T) {
	runner := NewRunner()
	ext := &Extension{
		Manifest: Manifest{
			ID:         "test-ext",
			Name:       "Test Extension",
			Version:    "0.1.0",
			Runtime:    "system",
			Entrypoint: "test",
		},
		Dir: t.TempDir(),
	}

	ctx := context.Background()
	var out []byte
	var err error

	if runtime.GOOS == "windows" {
		out, err = runner.Run(ctx, ext, "cmd.exe", "/c", "echo Hello Extension")
	} else {
		out, err = runner.Run(ctx, ext, "echo", "Hello Extension")
	}

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !strings.Contains(string(out), "Hello Extension") {
		t.Errorf("expected output to contain 'Hello Extension', got %q", string(out))
	}
}

func TestRunnerContextCancellation(t *testing.T) {
	runner := NewRunner()
	ext := &Extension{
		Manifest: Manifest{
			ID:         "test-timeout",
			Name:       "Timeout Test",
			Version:    "0.1.0",
			Runtime:    "system",
			Entrypoint: "sleep",
		},
		Dir: t.TempDir(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var err error
	if runtime.GOOS == "windows" {
		// ping localhost for 5 seconds
		_, err = runner.Run(ctx, ext, "ping", "-n", "5", "127.0.0.1")
	} else {
		_, err = runner.Run(ctx, ext, "sleep", "5")
	}

	if err == nil {
		t.Fatalf("expected command to be canceled, but succeeded")
	}
	if !strings.Contains(err.Error(), "canceled") && !strings.Contains(err.Error(), "context deadline exceeded") {
		// In some environments, os/exec returns process killed or deadline exceeded
		t.Logf("Process failed on timeout as expected: %v", err)
	}
}

func TestRunnerNilExtension(t *testing.T) {
	runner := NewRunner()
	_, err := runner.Run(context.Background(), nil, "echo")
	if err == nil {
		t.Fatalf("expected error when running nil extension")
	}
}

func TestFindPythonExecutable(t *testing.T) {
	pyPath, err := FindPythonExecutable()
	if err != nil {
		t.Logf("python not found in path: %v", err)
		return
	}
	if pyPath == "" {
		t.Errorf("expected non-empty python path")
	}
}

func TestRunStreamWithInput(t *testing.T) {
	pyPath, err := FindPythonExecutable()
	if err != nil {
		t.Skip("skipping python stream test, python not installed")
	}

	runner := NewRunner()
	var lines []string
	input := []byte("hello stream\nworld stream\n")

	err = runner.RunStreamWithInput(context.Background(), "", pyPath, input, func(line string) error {
		lines = append(lines, line)
		return nil
	}, "-c", "import sys; [print(line.strip().upper()) for line in sys.stdin]")

	if err != nil {
		t.Fatalf("RunStreamWithInput failed: %v", err)
	}

	if len(lines) < 2 {
		t.Fatalf("expected 2 lines, got: %v", lines)
	}

	if lines[0] != "HELLO STREAM" || lines[1] != "WORLD STREAM" {
		t.Errorf("unexpected output lines: %v", lines)
	}
}

