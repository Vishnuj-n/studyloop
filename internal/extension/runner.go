package extension

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Runner manages execution of external processes for extensions.
type Runner struct{}

// NewRunner creates a new Runner instance.
func NewRunner() *Runner {
	return &Runner{}
}

// Run executes a command with the specified executable and arguments within the extension directory.
// It supports context cancellation and captures standard output and error streams.
func (r *Runner) Run(ctx context.Context, ext *Extension, executable string, args ...string) ([]byte, error) {
	if ext == nil {
		return nil, fmt.Errorf("cannot run nil extension")
	}

	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = ext.Dir
	hideConsoleWindow(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("extension execution canceled: %w", ctx.Err())
		}
		errMsg := stderr.String()
		if errMsg != "" {
			return stdout.Bytes(), fmt.Errorf("extension failed: %w (stderr: %s)", err, errMsg)
		}
		return stdout.Bytes(), fmt.Errorf("extension failed: %w", err)
	}

	return stdout.Bytes(), nil
}
