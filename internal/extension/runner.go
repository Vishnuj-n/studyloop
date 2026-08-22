package extension

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
)

// Runner manages execution of external processes for extensions.
type Runner struct{}

// NewRunner creates a new Runner instance.
func NewRunner() *Runner {
	return &Runner{}
}

// FindPythonExecutable attempts to locate a Python interpreter on PATH.
func FindPythonExecutable() (string, error) {
	for _, name := range []string{"python", "python3", "py"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no python executable found in PATH")
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

// RunStreamWithInput runs an executable, streams `input` to its stdin, and calls `onLine` for every line printed to stdout.
func (r *Runner) RunStreamWithInput(ctx context.Context, dir string, executable string, input []byte, onLine func(line string) error, args ...string) error {
	cmd := exec.CommandContext(ctx, executable, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	hideConsoleWindow(cmd)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to open stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		stdinPipe.Close()
		return fmt.Errorf("failed to open stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		stdinPipe.Close()
		stdoutPipe.Close()
		return fmt.Errorf("failed to start command: %w", err)
	}

	go func() {
		defer stdinPipe.Close()
		if len(input) > 0 {
			_, _ = stdinPipe.Write(input)
		}
	}()

	scanner := bufio.NewScanner(stdoutPipe)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var lineErr error
	for scanner.Scan() {
		line := scanner.Text()
		if onLine != nil && line != "" {
			if err := onLine(line); err != nil {
				lineErr = err
				break
			}
		}
	}

	scanErr := scanner.Err()

	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return fmt.Errorf("execution canceled: %w", ctx.Err())
	}
	if lineErr != nil {
		return lineErr
	}
	if scanErr != nil && scanErr != io.EOF {
		return fmt.Errorf("scanner error: %w", scanErr)
	}
	if waitErr != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return fmt.Errorf("command failed: %w (stderr: %s)", waitErr, errMsg)
		}
		return fmt.Errorf("command failed: %w", waitErr)
	}

	return nil
}
