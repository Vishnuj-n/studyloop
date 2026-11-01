package extension

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Manifest represents the metadata defined in an extension's manifest.json.
type Manifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Runtime     string `json:"runtime"`
	Entrypoint  string `json:"entrypoint"`
	Tier         string `json:"tier,omitempty"`         // "free" (default) or "pro"
	Description  string `json:"description,omitempty"`  // Brief summary
	Category     string `json:"category,omitempty"`     // "reader", "study", "utility", etc.
	DownloadSize string `json:"download_size,omitempty"` // E.g. "~400 MB"
	SetupNotice  string `json:"setup_notice,omitempty"`  // Notice banner in UI setup
}

// Validate checks that required fields are present and safe.
func (m *Manifest) Validate() error {
	if strings.TrimSpace(m.ID) == "" {
		return fmt.Errorf("extension manifest missing required field: id")
	}
	if strings.Contains(m.ID, "..") || strings.ContainsAny(m.ID, "/\\") {
		return fmt.Errorf("invalid extension id: %q", m.ID)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("extension manifest missing required field: name")
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("extension manifest missing required field: version")
	}
	if strings.TrimSpace(m.Runtime) == "" {
		return fmt.Errorf("extension manifest missing required field: runtime")
	}
	if strings.TrimSpace(m.Entrypoint) == "" {
		return fmt.Errorf("extension manifest missing required field: entrypoint")
	}
	if strings.Contains(m.Entrypoint, "..") {
		return fmt.Errorf("invalid entrypoint path traversal: %q", m.Entrypoint)
	}
	if strings.TrimSpace(m.Tier) == "" {
		m.Tier = "free"
	}
	m.Tier = strings.ToLower(strings.TrimSpace(m.Tier))
	if m.Tier != "free" && m.Tier != "pro" {
		m.Tier = "free"
	}
	return nil
}

// Extension encapsulates an extension's manifest and directory location.
type Extension struct {
	Manifest Manifest
	Dir      string
}

// ID returns the extension identifier.
func (e *Extension) ID() string {
	return e.Manifest.ID
}

// Name returns the extension human-readable name.
func (e *Extension) Name() string {
	return e.Manifest.Name
}

// Version returns the extension version.
func (e *Extension) Version() string {
	return e.Manifest.Version
}

// Runtime returns the extension runtime (e.g. "python").
func (e *Extension) Runtime() string {
	return e.Manifest.Runtime
}

// Entrypoint returns the relative entrypoint filename.
func (e *Extension) Entrypoint() string {
	return e.Manifest.Entrypoint
}

// EntrypointPath returns the absolute path to the extension's entrypoint.
func (e *Extension) EntrypointPath() string {
	return filepath.Join(e.Dir, e.Manifest.Entrypoint)
}

// EnsureManifestLoaded reads and populates Manifest from ext.Dir/manifest.json if Version or ID is not set.
func (e *Extension) EnsureManifestLoaded() {
	if e == nil || e.Dir == "" {
		return
	}
	if e.Manifest.Version != "" && e.Manifest.ID != "" && e.Manifest.Runtime != "" {
		return
	}
	manifestPath := filepath.Join(e.Dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err == nil {
		var m Manifest
		if jsonErr := json.Unmarshal(data, &m); jsonErr == nil {
			if e.Manifest.ID == "" {
				e.Manifest.ID = m.ID
			}
			if e.Manifest.Name == "" {
				e.Manifest.Name = m.Name
			}
			if e.Manifest.Version == "" {
				e.Manifest.Version = m.Version
			}
			if e.Manifest.Runtime == "" {
				e.Manifest.Runtime = m.Runtime
			}
			if e.Manifest.Entrypoint == "" {
				e.Manifest.Entrypoint = m.Entrypoint
			}
			if e.Manifest.Tier == "" {
				e.Manifest.Tier = m.Tier
			}
			if e.Manifest.Description == "" {
				e.Manifest.Description = m.Description
			}
			if e.Manifest.Category == "" {
				e.Manifest.Category = m.Category
			}
		}
	}
}
