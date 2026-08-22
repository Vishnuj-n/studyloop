package extension

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Manager handles discovery, storage, and retrieval of extensions.
type Manager struct {
	extensionsDir string
	mu            sync.RWMutex
	extensions    map[string]*Extension
}

// ResolveExtensionsDir determines the root directory containing extensions.
// It checks explicit overrides, environment variable, project directory, and executable directory.
func ResolveExtensionsDir(customDir string) string {
	if trimmed := strings.TrimSpace(customDir); trimmed != "" {
		return filepath.Clean(trimmed)
	}

	if envDir := strings.TrimSpace(os.Getenv("STUDYLOOP_EXTENSIONS_DIR")); envDir != "" {
		return filepath.Clean(envDir)
	}

	// 1. Check relative ./extensions in working directory
	if info, err := os.Stat("extensions"); err == nil && info.IsDir() {
		absPath, err := filepath.Abs("extensions")
		if err == nil {
			return absPath
		}
		return "extensions"
	}

	// 2. Check adjacent to running executable (for packaged .exe)
	if exePath, err := os.Executable(); err == nil && exePath != "" {
		exeAdjacent := filepath.Join(filepath.Dir(exePath), "extensions")
		if info, err := os.Stat(exeAdjacent); err == nil && info.IsDir() {
			return exeAdjacent
		}
	}

	// Default fallback
	return "extensions"
}

// NewManager creates a new extension Manager for the specified directory or resolved default.
func NewManager(dir ...string) *Manager {
	targetDir := ""
	if len(dir) > 0 && strings.TrimSpace(dir[0]) != "" {
		targetDir = dir[0]
	}
	return &Manager{
		extensionsDir: ResolveExtensionsDir(targetDir),
		extensions:    make(map[string]*Extension),
	}
}

// Dir returns the root extensions directory being managed.
func (m *Manager) Dir() string {
	return m.extensionsDir
}

// Discover scans the extensions directory and registers all valid extensions.
// If the directory does not exist, it gracefully returns an empty slice without error.
func (m *Manager) Discover() ([]*Extension, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.extensions = make(map[string]*Extension)

	info, err := os.Stat(m.extensionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Extension{}, nil
		}
		return nil, fmt.Errorf("failed to access extensions directory %s: %w", m.extensionsDir, err)
	}
	if !info.IsDir() {
		return []*Extension{}, nil
	}

	entries, err := os.ReadDir(m.extensionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read extensions directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		extDir := filepath.Join(m.extensionsDir, entry.Name())
		manifestPath := filepath.Join(extDir, "manifest.json")

		data, err := os.ReadFile(manifestPath)
		if err != nil {
			// Subdirectory without manifest.json is simply ignored
			continue
		}

		var manifest Manifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			// Skip corrupted/invalid JSON manifests
			continue
		}

		if err := manifest.Validate(); err != nil {
			// Skip invalid manifests
			continue
		}

		ext := &Extension{
			Manifest: manifest,
			Dir:      extDir,
		}
		m.extensions[manifest.ID] = ext
	}

	return m.listLocked(), nil
}

// List returns all registered extensions in deterministic order sorted by ID.
func (m *Manager) List() []*Extension {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.listLocked()
}

func (m *Manager) listLocked() []*Extension {
	list := make([]*Extension, 0, len(m.extensions))
	for _, ext := range m.extensions {
		list = append(list, ext)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID() < list[j].ID()
	})
	return list
}

// Get returns an extension by its ID, if registered.
func (m *Manager) Get(id string) (*Extension, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ext, ok := m.extensions[id]
	return ext, ok
}
