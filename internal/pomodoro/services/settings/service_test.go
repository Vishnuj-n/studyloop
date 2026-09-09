package settings

import (
	"path/filepath"
	"testing"

	"ai-tutor/internal/pomodoro/domain"
)

func newSvc(t *testing.T) *Service {
	t.Helper()
	return &Service{
		filePath: filepath.Join(t.TempDir(), "settings.json"),
		current:  domain.DefaultSettings(),
	}
}

func TestDefaultValues(t *testing.T) {
	svc := newSvc(t)
	got := svc.Get()

	if got.DefaultVolume != 70 {
		t.Errorf("DefaultVolume: want 70, got %d", got.DefaultVolume)
	}
	if !got.AutoStartAudio {
		t.Error("AutoStartAudio should default to true")
	}
	if !got.NotifyOnComplete {
		t.Error("NotifyOnComplete should default to true")
	}
	if got.AutoStartNextTimer {
		t.Error("AutoStartNextTimer should default to false")
	}
}

func TestSaveAndGet(t *testing.T) {
	svc := newSvc(t)
	modified := domain.Settings{
		DefaultVolume:      85,
		AutoStartAudio:     false,
		NotifyOnComplete:   false,
		AutoStartNextTimer: true,
	}
	if err := svc.Save(modified); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	got := svc.Get()
	if got.DefaultVolume != 85 {
		t.Errorf("DefaultVolume: want 85, got %d", got.DefaultVolume)
	}
	if got.AutoStartAudio {
		t.Error("AutoStartAudio should be false")
	}
	if !got.AutoStartNextTimer {
		t.Error("AutoStartNextTimer should be true")
	}
}

func TestPersistenceAcrossInstances(t *testing.T) {
	fp := filepath.Join(t.TempDir(), "settings.json")

	s1 := &Service{filePath: fp, current: domain.DefaultSettings()}
	s1.Save(domain.Settings{DefaultVolume: 90, AutoStartAudio: false,
		NotifyOnComplete: true, AutoStartNextTimer: false})

	s2 := &Service{filePath: fp, current: domain.DefaultSettings()}
	s2.load()
	got := s2.Get()

	if got.DefaultVolume != 90 {
		t.Errorf("Persisted DefaultVolume: want 90, got %d", got.DefaultVolume)
	}
}

func TestSaveVolumeBoundaries(t *testing.T) {
	svc := newSvc(t)
	for _, v := range []int{0, 50, 100} {
		if err := svc.Save(domain.Settings{DefaultVolume: v}); err != nil {
			t.Errorf("Save volume %d: unexpected error %v", v, err)
		}
	}
}
