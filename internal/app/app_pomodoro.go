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

// ── Pomodoro Profile & Settings methods (bound to Wails JS) ────────────────────

func (a *App) PomodoroLoadProfiles() []domain.Profile {
	repo := a.getRepo()
	if repo == nil {
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

	settings, _ := repo.GetUserSettings()
	activeID := ""
	if settings != nil {
		activeID = settings.ActiveProfileID
	}

	profiles, err := repo.GetProfiles()
	if err != nil || len(profiles) == 0 {
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

	res := make([]domain.Profile, 0, len(profiles))
	for _, p := range profiles {
		dur := p.PomoDurationSec
		if dur <= 0 {
			dur = 25 * 60
		}
		brk := p.PomoBreakSec
		if brk < 0 {
			brk = 5 * 60
		}
		isDef := p.ID == activeID
		res = append(res, domain.Profile{
			ID:               p.ID,
			Name:             p.Name,
			DurationSec:      dur,
			BreakDurationSec: brk,
			MusicPath:        p.PomoMusicPath,
			Shuffle:          p.PomoShuffle,
			IsDefault:        isDef,
		})
	}
	// If no profile matched activeID, mark the first one as default
	if len(res) > 0 {
		hasDefault := false
		for _, r := range res {
			if r.IsDefault {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			res[0].IsDefault = true
		}
	}
	return res
}

func (a *App) PomodoroSaveProfile(p domain.Profile) error {
	repo := a.getRepo()
	if repo == nil {
		return fmt.Errorf("database not initialized")
	}
	if p.ID == "" || p.ID == "default" {
		// Fallback to active profile if ID is 'default' or empty
		settings, err := repo.GetUserSettings()
		if err == nil && settings != nil && settings.ActiveProfileID != "" {
			p.ID = settings.ActiveProfileID
		} else {
			profiles, err := repo.GetProfiles()
			if err == nil && len(profiles) > 0 {
				p.ID = profiles[0].ID
			}
		}
	}
	if p.ID == "" || p.ID == "default" {
		return nil
	}
	return repo.UpdateProfilePomoSettings(p.ID, p.DurationSec, p.BreakDurationSec, p.MusicPath, p.Shuffle)
}

func (a *App) PomodoroGetProfileByID(id string) *domain.Profile {
	repo := a.getRepo()
	if repo == nil {
		p := domain.Profile{
			ID:               "default",
			Name:             "Standard Focus",
			DurationSec:      25 * 60,
			BreakDurationSec: 5 * 60,
			IsDefault:        true,
		}
		return &p
	}
	prof, err := repo.GetProfileByID(id)
	if err != nil || prof == nil {
		return nil
	}
	dur := prof.PomoDurationSec
	if dur <= 0 {
		dur = 25 * 60
	}
	brk := prof.PomoBreakSec
	if brk < 0 {
		brk = 5 * 60
	}
	p := domain.Profile{
		ID:               prof.ID,
		Name:             prof.Name,
		DurationSec:      dur,
		BreakDurationSec: brk,
		MusicPath:        prof.PomoMusicPath,
		Shuffle:          prof.PomoShuffle,
	}
	return &p
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

func copyPomoFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = in.Close() // read-only file descriptor
	}()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(out, in)
	return err
}

