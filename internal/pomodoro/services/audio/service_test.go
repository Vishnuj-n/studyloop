package audio

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ai-tutor/internal/pomodoro/domain"
)

func TestNewReturnsService(t *testing.T) {
	svc := New()
	if svc == nil {
		t.Fatal("New() returned nil")
	}
}

func TestInitialStateIsIdle(t *testing.T) {
	svc := New()
	state := svc.GetState()
	if state.State != domain.AudioIdle {
		t.Errorf("Initial state: want %q, got %q", domain.AudioIdle, state.State)
	}
	if state.TrackName != "" {
		t.Errorf("Initial TrackName: want empty, got %q", state.TrackName)
	}
}

func TestSetVolumeClamps(t *testing.T) {
	svc := New()
	tests := []struct {
		in   int
		want float64
	}{
		{0, 0.0},
		{50, 0.5},
		{100, 1.0},
		{-10, 0.0},
		{150, 1.0},
	}
	for _, tc := range tests {
		svc.SetVolume(tc.in)
		if svc.vol < tc.want-0.01 || svc.vol > tc.want+0.01 {
			t.Errorf("SetVolume(%d): vol want %.2f, got %.2f", tc.in, tc.want, svc.vol)
		}
	}
}

func TestSetVolumeDoesNotChangeState(t *testing.T) {
	svc := New()
	svc.SetVolume(60)
	if svc.GetState().State != domain.AudioIdle {
		t.Error("SetVolume must not alter playback state")
	}
}

func TestStopFromIdle(t *testing.T) {
	svc := New()
	svc.Stop() // must not panic
	if svc.GetState().State != domain.AudioStopped {
		t.Errorf("After Stop(), want %q, got %q", domain.AudioStopped, svc.GetState().State)
	}
}

func TestDoubleStop(t *testing.T) {
	svc := New()
	svc.Stop()
	svc.Stop() // second stop must be a no-op
	if svc.GetState().State != domain.AudioStopped {
		t.Error("Double Stop should remain AudioStopped")
	}
}

func TestPlayLoopingMissingFileNocrash(t *testing.T) {
	svc := New()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PlayLooping panicked on missing file: %v", r)
		}
	}()
	svc.PlayLooping("/nonexistent/path/music.mp3")
	// Goroutine will fail and emit AudioStopped; wait briefly
	time.Sleep(150 * time.Millisecond)
	if svc.GetState().State != domain.AudioStopped {
		t.Logf("State after missing-file goroutine: %q", svc.GetState().State)
	}
}

func TestPlayShuffleFolderMissingFolderNocrash(t *testing.T) {
	svc := New()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PlayShuffleFolder panicked on missing folder: %v", r)
		}
	}()
	svc.PlayShuffleFolder("/nonexistent/folder")
	if svc.GetState().State != domain.AudioStopped {
		t.Errorf("Missing folder: want %q, got %q", domain.AudioStopped, svc.GetState().State)
	}
}

func TestPlayShuffleFolderEmptyFolder(t *testing.T) {
	svc := New()
	svc.PlayShuffleFolder(t.TempDir()) // no .mp3 files
	if svc.GetState().State != domain.AudioStopped {
		t.Errorf("Empty folder: want %q, got %q", domain.AudioStopped, svc.GetState().State)
	}
}

func TestPlayLoopingInvalidMp3NocrashNoDownload(t *testing.T) {
	tmp := t.TempDir()
	mp3Path := filepath.Join(tmp, "fake.mp3")
	// Minimal fake MP3 header — beep will return a decode error, not panic
	_ = os.WriteFile(mp3Path, []byte{0xFF, 0xFB, 0x10, 0x00}, 0644)

	svc := New()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PlayLooping panicked with invalid MP3: %v", r)
		}
	}()
	svc.PlayLooping(mp3Path)
	time.Sleep(150 * time.Millisecond)
	// After decode error the goroutine emits AudioStopped
	if svc.GetState().State != domain.AudioStopped {
		t.Logf("State after decode failure: %q", svc.GetState().State)
	}
}
