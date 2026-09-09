package settings

import (
	"path/filepath"
	"sync"

	"ai-tutor/internal/pomodoro/domain"
	"ai-tutor/internal/pomodoro/infra/storage"
)

// Service persists and exposes user preferences.
type Service struct {
	mu       sync.RWMutex
	current  domain.Settings
	filePath string
}

// New creates a Service that stores settings under dataDir.
func New(dataDir string) *Service {
	ss := &Service{
		filePath: filepath.Join(dataDir, "settings.json"),
		current:  domain.DefaultSettings(),
	}
	ss.load()
	return ss
}

// Get returns the current settings snapshot.
func (ss *Service) Get() domain.Settings {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.current
}

// Save updates the in-memory settings and writes settings.json.
func (ss *Service) Save(s domain.Settings) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.current = s
	return storage.Save(ss.filePath, s)
}

func (ss *Service) load() {
	_ = storage.Load(ss.filePath, &ss.current)
}
