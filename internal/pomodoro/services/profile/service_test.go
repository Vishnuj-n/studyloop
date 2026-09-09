package profile

import (
	"os"
	"path/filepath"
	"testing"

	"ai-tutor/internal/pomodoro/domain"
)

func TestLoadReturnsDefaults(t *testing.T) {
	svc := &Service{filePath: filepath.Join(t.TempDir(), "profiles.json")}

	profiles := svc.Load()
	if len(profiles) != 3 {
		t.Errorf("Expected 3 default profiles, got %d", len(profiles))
	}
	if profiles[0].ID != "deep-work" {
		t.Errorf("Expected first ID 'deep-work', got %q", profiles[0].ID)
	}
	if profiles[0].DurationSec != 90*60 {
		t.Errorf("Expected 5400 sec, got %d", profiles[0].DurationSec)
	}

	// profiles.json must be created on first load
	_, err := os.Stat(svc.filePath)
	if err != nil {
		t.Errorf("profiles.json not created: %v", err)
	}
}

func TestSaveNewProfile(t *testing.T) {
	svc := &Service{filePath: filepath.Join(t.TempDir(), "profiles.json")}
	svc.Load()

	p := domain.Profile{ID: "custom", Name: "Custom", DurationSec: 45 * 60}
	if err := svc.Save(p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got := svc.GetByID("custom")
	if got == nil {
		t.Fatal("Profile not found after save")
	}
	if got.Name != "Custom" {
		t.Errorf("Expected name 'Custom', got %q", got.Name)
	}
}

func TestSaveUpdatesExisting(t *testing.T) {
	svc := &Service{filePath: filepath.Join(t.TempDir(), "profiles.json")}
	svc.Load()

	updated := domain.Profile{ID: "pomodoro", Name: "Pomodoro 35", DurationSec: 35 * 60}
	svc.Save(updated)

	got := svc.GetByID("pomodoro")
	if got.DurationSec != 35*60 {
		t.Errorf("Expected 2100, got %d", got.DurationSec)
	}
}

func TestDeleteRemovesProfile(t *testing.T) {
	svc := &Service{filePath: filepath.Join(t.TempDir(), "profiles.json")}
	svc.Load()
	before := len(svc.profiles)

	if err := svc.Delete("pomodoro"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if len(svc.profiles) != before-1 {
		t.Errorf("Expected %d profiles, got %d", before-1, len(svc.profiles))
	}
	if svc.GetByID("pomodoro") != nil {
		t.Error("Deleted profile still reachable")
	}
}

func TestGetByIDMissingReturnsNil(t *testing.T) {
	svc := &Service{filePath: filepath.Join(t.TempDir(), "profiles.json")}
	svc.Load()

	if svc.GetByID("nonexistent") != nil {
		t.Error("Expected nil for nonexistent profile")
	}
}

func TestPersistenceAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")

	s1 := &Service{filePath: path}
	s1.Load()
	s1.Save(domain.Profile{ID: "test-persist", Name: "Persist", DurationSec: 120})

	s2 := &Service{filePath: path}
	profiles := s2.Load()

	found := false
	for _, p := range profiles {
		if p.ID == "test-persist" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Saved profile not found in new instance")
	}
}
