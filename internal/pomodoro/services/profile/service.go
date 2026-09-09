package profile

import (
	"path/filepath"
	"sync"

	"ai-tutor/internal/pomodoro/domain"
	"ai-tutor/internal/pomodoro/infra/storage"
)

// Service handles loading and saving profiles from AppData/profiles.json.
// JSON is only touched here — all other services use in-memory domain.Profile values.
type Service struct {
	mu       sync.RWMutex
	profiles []domain.Profile
	filePath string
}

// New creates a Service that stores profiles under dataDir.
func New(dataDir string) *Service {
	return &Service{
		filePath: filepath.Join(dataDir, "profiles.json"),
	}
}

// Load reads profiles.json and caches the result. Returns defaults on first run.
func (s *Service) Load() []domain.Profile {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := storage.Load(s.filePath, &s.profiles); err != nil || len(s.profiles) == 0 {
		s.profiles = defaultProfiles()
		_ = s.saveUnlocked()
	}
	return s.profiles
}

// Save upserts a profile in the cache and writes profiles.json.
// If p.IsDefault is true, all other profiles are cleared of their IsDefault flag first.
func (s *Service) Save(p domain.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.IsDefault {
		for i := range s.profiles {
			if s.profiles[i].ID != p.ID {
				s.profiles[i].IsDefault = false
			}
		}
	}
	for i, existing := range s.profiles {
		if existing.ID == p.ID {
			s.profiles[i] = p
			return s.saveUnlocked()
		}
	}
	s.profiles = append(s.profiles, p)
	return s.saveUnlocked()
}

// GetByID returns a profile from the in-memory cache only.
func (s *Service) GetByID(id string) *domain.Profile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.profiles {
		if p.ID == id {
			cp := p
			return &cp
		}
	}
	return nil
}

// Delete removes a profile by ID and persists the change.
func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	filtered := s.profiles[:0]
	for _, p := range s.profiles {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}
	s.profiles = filtered
	return s.saveUnlocked()
}

func (s *Service) saveUnlocked() error {
	return storage.Save(s.filePath, s.profiles)
}

func defaultProfiles() []domain.Profile {
	return []domain.Profile{
		{ID: "deep-work", Name: "Deep Work — 90 min", DurationSec: 90 * 60},
		{ID: "pomodoro", Name: "Pomodoro — 25 min", DurationSec: 25 * 60, BreakDurationSec: 5 * 60, IsDefault: true},
		{ID: "short-break", Name: "Short Break — 5 min", DurationSec: 5 * 60},
	}
}
