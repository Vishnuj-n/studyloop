package extension

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// FindUVExecutable locates the uv executable in standard search paths.
func FindUVExecutable() (string, error) {
	binaryName := "uv"
	if runtime.GOOS == "windows" {
		binaryName = "uv.exe"
	}

	// 1. Check relative paths in project/runtime root
	searchDirs := []string{
		"build/bin",
		"bin",
		"../build/bin",
		"../bin",
	}

	for _, dir := range searchDirs {
		candidate := filepath.Join(dir, binaryName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			if abs, err := filepath.Abs(candidate); err == nil {
				return abs, nil
			}
			return candidate, nil
		}
	}

	// 2. Check adjacent to current running executable (e.g. packaged Studyloop.exe)
	if exePath, err := os.Executable(); err == nil && exePath != "" {
		exeDir := filepath.Dir(exePath)
		candidate := filepath.Join(exeDir, "bin", binaryName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		candidateDirect := filepath.Join(exeDir, binaryName)
		if info, err := os.Stat(candidateDirect); err == nil && !info.IsDir() {
			return candidateDirect, nil
		}
	}

	// 3. Check AppData / Local runtime directory
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			candidate := filepath.Join(appData, "Studyloop", "bin", binaryName)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			candidate := filepath.Join(localAppData, "Studyloop", "bin", binaryName)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	} else {
		homeDir, err := os.UserHomeDir()
		if err == nil && homeDir != "" {
			candidate := filepath.Join(homeDir, ".local", "share", "studyloop", "bin", binaryName)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}

	// 4. Check system PATH
	if p, err := exec.LookPath(binaryName); err == nil {
		return p, nil
	}

	return "", fmt.Errorf("uv executable (%s) not found in application bin or PATH", binaryName)
}

// GetVenvPython returns the path to the Python interpreter inside a virtual environment.
func GetVenvPython(venvDir string) string {
	if strings.TrimSpace(venvDir) == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(venvDir, "bin", "python3")
}

// ResolveExtensionVenvDir determines where the isolated virtual environment for an extension should live.
func ResolveExtensionVenvDir(ext *Extension) string {
	if ext == nil {
		return ""
	}

	// In dev mode or when extension directory is writable, default to <ext.Dir>/.venv
	localVenv := filepath.Join(ext.Dir, ".venv")
	
	// Test if ext.Dir is writable
	testFile := filepath.Join(ext.Dir, ".write_test")
	if err := os.WriteFile(testFile, []byte("ok"), 0644); err == nil {
		_ = os.Remove(testFile)
		return localVenv
	}

	// If ext.Dir is read-only (e.g. Program Files), use user AppData/extensions/<id>/.venv
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "Studyloop", "extensions", ext.ID(), ".venv")
		}
	}

	homeDir, err := os.UserHomeDir()
	if err == nil && homeDir != "" {
		return filepath.Join(homeDir, ".local", "share", "studyloop", "extensions", ext.ID(), ".venv")
	}

	return localVenv
}
