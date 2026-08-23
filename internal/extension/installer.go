package extension

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Uninstall removes the extension directory for the given ID and refreshes discovery.
func (m *Manager) Uninstall(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanID := strings.TrimSpace(id)
	if cleanID == "" || strings.Contains(cleanID, "..") || strings.ContainsAny(cleanID, "/\\") {
		return fmt.Errorf("invalid extension id: %q", id)
	}

	ext, exists := m.extensions[cleanID]
	if !exists {
		return fmt.Errorf("extension %q not found", cleanID)
	}

	targetDir := ext.Dir
	// Safety check: ensure targetDir is within extensionsDir
	rel, err := filepath.Rel(m.extensionsDir, targetDir)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("refusing to remove path outside extensions directory: %s", targetDir)
	}

	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("failed to remove extension directory: %w", err)
	}

	delete(m.extensions, cleanID)
	return nil
}

// InstallZip unpacks a zipped extension into the extensions directory and registers it.
func (m *Manager) InstallZip(zipPath string) (*Extension, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	// 1. Locate and parse manifest.json within the archive
	var manifestData []byte
	var manifestPrefix string

	for _, f := range r.File {
		cleanName := filepath.ToSlash(f.Name)
		if strings.HasSuffix(cleanName, "manifest.json") {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to read manifest from zip: %w", err)
			}
			manifestData, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read manifest content: %w", err)
			}
			manifestPrefix = strings.TrimSuffix(cleanName, "manifest.json")
			break
		}
	}

	if len(manifestData) == 0 {
		return nil, fmt.Errorf("no manifest.json found in extension zip")
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest.json in zip: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	// 2. Ensure base extensions directory exists
	if err := os.MkdirAll(m.extensionsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create extensions directory: %w", err)
	}

	targetExtDir := filepath.Join(m.extensionsDir, manifest.ID)
	_ = os.RemoveAll(targetExtDir) // Clean up any existing installation
	if err := os.MkdirAll(targetExtDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create target extension directory: %w", err)
	}

	// 3. Extract all files under the manifestPrefix into targetExtDir
	for _, f := range r.File {
		cleanName := filepath.ToSlash(f.Name)
		if !strings.HasPrefix(cleanName, manifestPrefix) {
			continue
		}

		relPath := strings.TrimPrefix(cleanName, manifestPrefix)
		if relPath == "" || relPath == "/" {
			continue
		}

		destPath := filepath.Join(targetExtDir, filepath.FromSlash(relPath))
		// Guard against ZipSlip
		if !strings.HasPrefix(destPath, filepath.Clean(targetExtDir)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return nil, fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return nil, fmt.Errorf("failed to create parent directory for %s: %w", destPath, err)
		}

		destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return nil, fmt.Errorf("failed to create destination file %s: %w", destPath, err)
		}

		rc, err := f.Open()
		if err != nil {
			destFile.Close()
			return nil, fmt.Errorf("failed to read zip entry %s: %w", f.Name, err)
		}

		_, copyErr := io.Copy(destFile, rc)
		rc.Close()
		destFile.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("failed to extract file %s: %w", f.Name, copyErr)
		}
	}

	// 4. Register newly installed extension
	ext := &Extension{
		Manifest: manifest,
		Dir:      targetExtDir,
	}

	m.mu.Lock()
	m.extensions[manifest.ID] = ext
	m.mu.Unlock()

	return ext, nil
}
