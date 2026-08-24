package extension

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ReadinessStatus describes the environment and verification state of an extension.
type ReadinessStatus struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Runtime     string `json:"runtime"`
	IsReady     bool   `json:"is_ready"`
	NeedsSetup  bool   `json:"needs_setup"`
	VenvPath    string `json:"venv_path,omitempty"`
	PythonPath  string `json:"python_path,omitempty"`
	Message     string `json:"message,omitempty"`
	Error       string `json:"error,omitempty"`
}

// CheckReadiness inspects an extension's runtime, virtual environment, and smoke test status.
func CheckReadiness(ctx context.Context, ext *Extension) ReadinessStatus {
	if ext == nil {
		return ReadinessStatus{
			IsReady: false,
			Error:   "extension is nil",
		}
	}

	status := ReadinessStatus{
		ID:      ext.ID(),
		Name:    ext.Name(),
		Runtime: ext.Runtime(),
	}

	runtimeType := strings.ToLower(strings.TrimSpace(ext.Runtime()))
	// Built-in or internal extensions are always ready
	if runtimeType == "internal" || runtimeType == "builtin" {
		status.IsReady = true
		status.NeedsSetup = false
		status.Message = "Built-in extension ready"
		return status
	}

	// For Python runtime: check virtual environment and smoke test
	if runtimeType == "python" || runtimeType == "py" {
		venvDir := ResolveExtensionVenvDir(ext)
		status.VenvPath = venvDir

		pyPath := GetVenvPython(venvDir)
		status.PythonPath = pyPath

		if pyInfo, err := os.Stat(pyPath); err != nil || pyInfo.IsDir() {
			status.IsReady = false
			status.NeedsSetup = true
			status.Message = "Virtual environment not initialized"
			return status
		}

		// Run fast smoke test
		if err := RunSmokeTest(ctx, ext, pyPath); err != nil {
			status.IsReady = false
			status.NeedsSetup = true
			status.Error = err.Error()
			status.Message = "Environment test failed"
			return status
		}

		status.IsReady = true
		status.NeedsSetup = false
		status.Message = "Extension environment verified & ready"
		return status
	}

	// Binary/executable
	status.IsReady = true
	status.NeedsSetup = false
	return status
}

// RunSmokeTest invokes the extension's entrypoint with the --test flag to ensure runtime readiness.
func RunSmokeTest(ctx context.Context, ext *Extension, pythonPath string) error {
	if ext == nil {
		return fmt.Errorf("cannot test nil extension")
	}

	testCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(testCtx, pythonPath, ext.EntrypointPath(), "--test")
	cmd.Dir = ext.Dir
	AttachAuthEnv(cmd)
	hideConsoleWindow(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(stderr.String())
		if errStr != "" {
			return fmt.Errorf("smoke test failed: %s", errStr)
		}
		outStr := strings.TrimSpace(stdout.String())
		if outStr != "" {
			return fmt.Errorf("smoke test failed: %s", outStr)
		}
		return fmt.Errorf("smoke test failed: %w", err)
	}

	return nil
}

// SetupExtensionEnv automatically provisions the Python virtual environment via uv and installs requirements.
func SetupExtensionEnv(ctx context.Context, ext *Extension, onLog func(line string)) error {
	if ext == nil {
		return fmt.Errorf("cannot setup nil extension")
	}

	uvPath, err := FindUVExecutable()
	if err != nil {
		return fmt.Errorf("uv executable not found: %w. Please ensure uv is bundled in bin/", err)
	}

	venvDir := ResolveExtensionVenvDir(ext)
	parentDir := filepath.Dir(venvDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", parentDir, err)
	}

	logLine := func(msg string) {
		if onLog != nil {
			onLog(msg)
		}
	}

	logLine(fmt.Sprintf("Using uv engine: %s", uvPath))
	logLine(fmt.Sprintf("Configuring isolated environment in: %s", venvDir))

	// 1. Create venv with uv (uv venv <venvDir> --allow-existing --system-site-packages)
	// uv automatically detects system Python and reuses already-installed global packages (e.g. yt-dlp, edge-tts)
	logLine("Step 1/3: Initializing Python virtual environment...")
	venvCmd := exec.CommandContext(ctx, uvPath, "venv", venvDir, "--allow-existing", "--system-site-packages")
	venvCmd.Dir = ext.Dir
	hideConsoleWindow(venvCmd)

	var venvOut, venvErr bytes.Buffer
	venvCmd.Stdout = &venvOut
	venvCmd.Stderr = &venvErr

	if err := venvCmd.Run(); err != nil {
		errMsg := strings.TrimSpace(venvErr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(venvOut.String())
		}
		return fmt.Errorf("failed to create virtual environment with uv: %w (output: %s)", err, errMsg)
	}
	logLine("[OK] Python virtual environment created successfully.")

	// 2. Install requirements if requirements.txt exists
	reqPath := filepath.Join(ext.Dir, "requirements.txt")
	if info, err := os.Stat(reqPath); err == nil && !info.IsDir() {
		logLine("Step 2/3: Installing extension dependencies from requirements.txt...")
		pipCmd := exec.CommandContext(ctx, uvPath, "pip", "install", "--python", venvDir, "-r", reqPath)
		pipCmd.Dir = ext.Dir
		hideConsoleWindow(pipCmd)

		var pipOut, pipErr bytes.Buffer
		pipCmd.Stdout = &pipOut
		pipCmd.Stderr = &pipErr

		if err := pipCmd.Run(); err != nil {
			errMsg := strings.TrimSpace(pipErr.String())
			if errMsg == "" {
				errMsg = strings.TrimSpace(pipOut.String())
			}
			return fmt.Errorf("failed to install dependencies with uv pip: %w (output: %s)", err, errMsg)
		}
		logLine("[OK] Dependencies installed successfully.")
	} else {
		logLine("Step 2/3: No requirements.txt found, skipping dependency install.")
	}

	// 3. Run smoke test
	logLine("Step 3/3: Running extension verification self-test...")
	pyPath := GetVenvPython(venvDir)
	if err := RunSmokeTest(ctx, ext, pyPath); err != nil {
		return fmt.Errorf("extension self-test failed after installation: %w", err)
	}

	logLine("[OK] Extension verified and ready!")
	return nil
}
