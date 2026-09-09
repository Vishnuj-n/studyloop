package persistence

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"ai-tutor/internal/pomodoro/domain"
	"ai-tutor/internal/pomodoro/infra/storage"
)

// Service handles reading and writing state.json.
// JSON is only touched here — all other services use in-memory domain.SessionState values.
type Service struct {
	mu       sync.Mutex
	filePath string
}

// New creates a Service that stores session state under dataDir.
func New(dataDir string) *Service {
	return &Service{
		filePath: filepath.Join(dataDir, "state.json"),
	}
}

// Load reads state.json. Returns nil if the file is absent or stale (>24 h).
func (s *Service) Load() *domain.SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()

	var state domain.SessionState
	if err := storage.Load(s.filePath, &state); err != nil {
		return nil
	}
	if state.SavedAt == 0 || time.Now().Unix()-state.SavedAt > 86400 {
		_ = os.Remove(s.filePath)
		return nil
	}
	return &state
}

// Save writes state.json with the current Unix timestamp.
func (s *Service) Save(state domain.SessionState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state.SavedAt = time.Now().Unix()
	return storage.Save(s.filePath, state)
}

// Clear deletes state.json (called on session completion or manual stop).
func (s *Service) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = os.Remove(s.filePath)
}
