package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ai-tutor/internal/pomodoro/domain"
)

func TestSaveAndLoad(t *testing.T) {
	svc := &Service{filePath: filepath.Join(t.TempDir(), "state.json")}

	orig := domain.SessionState{ProfileID: "pomodoro", TotalSec: 1500, RemainingSec: 1200}
	if err := svc.Save(orig); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got := svc.Load()
	if got == nil {
		t.Fatal("Load returned nil for fresh session")
	}
	if got.ProfileID != "pomodoro" {
		t.Errorf("ProfileID: want 'pomodoro', got %q", got.ProfileID)
	}
	if got.RemainingSec != 1200 {
		t.Errorf("RemainingSec: want 1200, got %d", got.RemainingSec)
	}
}

func TestLoadMissingFileReturnsNil(t *testing.T) {
	svc := &Service{filePath: filepath.Join(t.TempDir(), "no-such.json")}
	if svc.Load() != nil {
		t.Error("Expected nil when file absent")
	}
}

func TestLoadStaleSessionReturnsNil(t *testing.T) {
	svc := &Service{filePath: filepath.Join(t.TempDir(), "state.json")}

	// Write stale data directly — Save() would overwrite SavedAt with time.Now()
	stale := domain.SessionState{
		ProfileID:    "pomodoro",
		TotalSec:     1500,
		RemainingSec: 1200,
		SavedAt:      time.Now().Unix() - 86401, // 24 h + 1 s
	}
	data, _ := json.MarshalIndent(stale, "", "  ")
	os.WriteFile(svc.filePath, data, 0644)

	if svc.Load() != nil {
		t.Error("Expected nil for stale session")
	}
	if _, err := os.Stat(svc.filePath); !os.IsNotExist(err) {
		t.Error("Stale state.json was not deleted")
	}
}

func TestLoadFreshSessionReturns(t *testing.T) {
	svc := &Service{filePath: filepath.Join(t.TempDir(), "state.json")}

	fresh := domain.SessionState{
		ProfileID:    "pomodoro",
		TotalSec:     1500,
		RemainingSec: 900,
		SavedAt:      time.Now().Unix() - 3600, // 1 h old — still fresh
	}
	svc.Save(fresh)

	got := svc.Load()
	if got == nil {
		t.Fatal("Expected session to load; got nil")
	}
	if got.RemainingSec != 900 {
		t.Errorf("RemainingSec: want 900, got %d", got.RemainingSec)
	}
}

func TestClearDeletesFile(t *testing.T) {
	svc := &Service{filePath: filepath.Join(t.TempDir(), "state.json")}

	svc.Save(domain.SessionState{ProfileID: "x", TotalSec: 10, RemainingSec: 5})
	svc.Clear()

	if _, err := os.Stat(svc.filePath); !os.IsNotExist(err) {
		t.Error("state.json not deleted after Clear")
	}
}
