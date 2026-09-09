package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"ai-tutor/internal/runtime"
)

// Load reads a JSON file at path and unmarshals it into v.
func Load(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Save marshals v as indented JSON and writes it to path.
func Save(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// DataDir returns (and creates) the Pomodoro data directory under Studyloop app dir.
func DataDir() string {
	base, err := runtime.ResolveAppDir()
	if err != nil {
		base, _ = os.UserCacheDir()
		base = filepath.Join(base, "Studyloop")
	}
	dir := filepath.Join(base, "pomodoro")
	_ = os.MkdirAll(dir, 0755)
	return dir
}
