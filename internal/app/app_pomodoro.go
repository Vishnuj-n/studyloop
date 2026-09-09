package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"ai-tutor/internal/pomodoro/domain"
	"ai-tutor/internal/runtime"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ── Pomodoro Timer methods (bound to Wails JS) ──────────────────────────────────

func (a *App) PomodoroStartTimer(profileID string, durationSec int) {
	if a.pomoTimer != nil {
		a.pomoTimer.Start(profileID, durationSec)
	}
}

func (a *App) PomodoroResumeTimer(state domain.SessionState) {
	if a.pomoTimer != nil {
		a.pomoTimer.Resume(state)
	}
}

func (a *App) PomodoroPauseTimer() {
	if a.pomoTimer != nil {
		a.pomoTimer.Pause()
	}
}

func (a *App) PomodoroStopTimer() {
	if a.pomoTimer != nil {
		a.pomoTimer.Stop()
	}
}

func (a *App) PomodoroGetTimerState() map[string]interface{} {
	if a.pomoTimer != nil {
		return a.pomoTimer.GetState()
	}
	return map[string]interface{}{
		"running":      false,
		"remainingSec": 0,
		"totalSec":     0,
		"profileId":    "",
	}
}

func (a *App) PomodoroCheckResumeSession() *domain.SessionState {
	return nil
}

// ── Pomodoro Audio methods (bound to Wails JS) ──────────────────────────────────

func (a *App) PomodoroPlayLooping(filePath string) {
	if a.pomoAudio != nil {
		a.pomoAudio.PlayLooping(filePath)
	}
}

func (a *App) PomodoroPlayShuffleFolder(folder string) {
	if !a.IsProUser() {
		return
	}
	if a.pomoAudio != nil {
		a.pomoAudio.PlayShuffleFolder(folder)
	}
}

func (a *App) PomodoroStopAudio() {
	if a.pomoAudio != nil {
		a.pomoAudio.Stop()
	}
}

func (a *App) PomodoroSetVolume(v int) {
	if a.pomoAudio != nil {
		a.pomoAudio.SetVolume(v)
	}
}

func (a *App) PomodoroGetAudioState() domain.AudioStatePayload {
	if a.pomoAudio != nil {
		return a.pomoAudio.GetState()
	}
	return domain.AudioStatePayload{State: domain.AudioIdle}
}

func (a *App) PomodoroPlayChime() {
	if a.pomoAudio != nil {
		a.pomoAudio.PlayChime()
	}
}

// ── Pomodoro Profile & Settings methods (bound to Wails JS) ────────────────────

func (a *App) PomodoroLoadProfiles() []domain.Profile {
	return []domain.Profile{
		{
			ID:               "default",
			Name:             "Standard Focus",
			DurationSec:      25 * 60,
			BreakDurationSec: 5 * 60,
			IsDefault:        true,
		},
	}
}

func (a *App) PomodoroSaveProfile(p domain.Profile) error {
	return nil
}

func (a *App) PomodoroGetProfileByID(id string) *domain.Profile {
	p := domain.Profile{
		ID:               "default",
		Name:             "Standard Focus",
		DurationSec:      25 * 60,
		BreakDurationSec: 5 * 60,
		IsDefault:        true,
	}
	return &p
}

func (a *App) PomodoroDeleteProfile(id string) error {
	return nil
}

func (a *App) PomodoroGetSettings() domain.Settings {
	return domain.DefaultSettings()
}

func (a *App) PomodoroSaveSettings(s domain.Settings) error {
	return nil
}

func (a *App) PomodoroGetStats() domain.StatsData {
	return domain.StatsData{}
}

func (a *App) PomodoroRecordSessionComplete() domain.StatsData {
	return domain.StatsData{}
}

func (a *App) PomodoroPickMusicFile() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("app context unavailable")
	}
	srcPath, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:   "Select MP3 file",
		Filters: []wailsruntime.FileFilter{{DisplayName: "MP3 Audio (*.mp3)", Pattern: "*.mp3"}},
	})
	if err != nil || srcPath == "" {
		return srcPath, err
	}

	appDir, err := runtime.ResolveAppDir()
	if err != nil {
		return srcPath, nil
	}
	audioDir := filepath.Join(appDir, "audio")
	_ = os.MkdirAll(audioDir, 0755)

	destPath := filepath.Join(audioDir, filepath.Base(srcPath))
	if err := copyPomoFile(srcPath, destPath); err != nil {
		return srcPath, nil // fallback to original path
	}
	return destPath, nil
}

func (a *App) PomodoroPickMusicFolder() (string, error) {
	if !a.IsProUser() {
		return "", fmt.Errorf("folder shuffle requires a Pro subscription")
	}
	if a.ctx == nil {
		return "", fmt.Errorf("app context unavailable")
	}
	path, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select music folder",
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

func copyPomoFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

