package timer

import (
	"testing"
	"time"

	"ai-tutor/internal/pomodoro/domain"
	"ai-tutor/internal/pomodoro/services/persistence"
)

func newTestTimer(t *testing.T) *Service {
	t.Helper()
	// persistence.New accepts any dataDir — use a temp dir so tests are isolated
	ps := persistence.New(t.TempDir())
	return New(ps) // emitter defaults to events.Noop
}

func TestTimerStart(t *testing.T) {
	svc := newTestTimer(t)
	svc.Start("profile-1", 60)

	state := svc.GetState()
	if !state["running"].(bool) {
		t.Error("Timer should be running after Start")
	}
	if state["totalSec"].(int) != 60 {
		t.Errorf("totalSec: want 60, got %v", state["totalSec"])
	}
	if state["remainingSec"].(int) != 60 {
		t.Errorf("remainingSec: want 60, got %v", state["remainingSec"])
	}
	if state["profileId"].(string) != "profile-1" {
		t.Errorf("profileId: want 'profile-1', got %v", state["profileId"])
	}
}

func TestTimerPause(t *testing.T) {
	svc := newTestTimer(t)
	svc.Start("p", 60)
	time.Sleep(100 * time.Millisecond)

	svc.Pause()
	state := svc.GetState()
	if state["running"].(bool) {
		t.Error("Timer should not be running after Pause")
	}

	before := state["remainingSec"].(int)
	time.Sleep(100 * time.Millisecond)
	after := svc.GetState()["remainingSec"].(int)
	if after != before {
		t.Error("remainingSec must not decrease while paused")
	}
}

func TestTimerStop(t *testing.T) {
	svc := newTestTimer(t)
	svc.Start("p", 60)
	svc.Stop()

	state := svc.GetState()
	if state["running"].(bool) {
		t.Error("Timer should not be running after Stop")
	}
	if state["remainingSec"].(int) != 60 {
		t.Errorf("remainingSec after Stop: want 60, got %v", state["remainingSec"])
	}
}

func TestTimerResume(t *testing.T) {
	svc := newTestTimer(t)

	ss := domain.SessionState{ProfileID: "pomo", TotalSec: 1500, RemainingSec: 1200}
	svc.Resume(ss)

	got := svc.GetState()
	if !got["running"].(bool) {
		t.Error("Timer should be running after Resume")
	}
	if got["totalSec"].(int) != 1500 {
		t.Errorf("totalSec: want 1500, got %v", got["totalSec"])
	}
	if got["remainingSec"].(int) != 1200 {
		t.Errorf("remainingSec: want 1200, got %v", got["remainingSec"])
	}
}

func TestTimerSecondStartOverridesFirst(t *testing.T) {
	svc := newTestTimer(t)
	svc.Start("p1", 60)
	svc.Start("p2", 120)

	state := svc.GetState()
	if state["profileId"].(string) != "p2" {
		t.Errorf("profileId: want 'p2', got %v", state["profileId"])
	}
	if state["totalSec"].(int) != 120 {
		t.Errorf("totalSec: want 120, got %v", state["totalSec"])
	}
}
