package app

import (
	"ai-tutor/internal/pomodoro/domain"
)

// ── Pomodoro Timer methods (bound to Wails JS) ──────────────────────────────────

func (a *App) PomodoroStartTimer(profileID string, durationSec int) {
	if a.pomoApp != nil {
		a.pomoApp.StartTimer(profileID, durationSec)
	}
}

func (a *App) PomodoroResumeTimer(state domain.SessionState) {
	if a.pomoApp != nil {
		a.pomoApp.ResumeTimer(state)
	}
}

func (a *App) PomodoroPauseTimer() {
	if a.pomoApp != nil {
		a.pomoApp.PauseTimer()
	}
}

func (a *App) PomodoroStopTimer() {
	if a.pomoApp != nil {
		a.pomoApp.StopTimer()
	}
}

func (a *App) PomodoroGetTimerState() map[string]interface{} {
	if a.pomoApp != nil {
		return a.pomoApp.GetTimerState()
	}
	return map[string]interface{}{
		"running":      false,
		"remainingSec": 0,
		"totalSec":     0,
		"profileId":    "",
	}
}

func (a *App) PomodoroCheckResumeSession() *domain.SessionState {
	if a.pomoApp != nil {
		return a.pomoApp.CheckResumeSession()
	}
	return nil
}

// ── Pomodoro Audio methods (bound to Wails JS) ──────────────────────────────────

func (a *App) PomodoroPlayLooping(filePath string) {
	if a.pomoApp != nil {
		a.pomoApp.PlayLooping(filePath)
	}
}

func (a *App) PomodoroPlayShuffleFolder(folder string) {
	if a.pomoApp != nil {
		a.pomoApp.PlayShuffleFolder(folder)
	}
}

func (a *App) PomodoroStopAudio() {
	if a.pomoApp != nil {
		a.pomoApp.StopAudio()
	}
}

func (a *App) PomodoroSetVolume(v int) {
	if a.pomoApp != nil {
		a.pomoApp.SetVolume(v)
	}
}

func (a *App) PomodoroGetAudioState() domain.AudioStatePayload {
	if a.pomoApp != nil {
		return a.pomoApp.GetAudioState()
	}
	return domain.AudioStatePayload{State: domain.AudioIdle}
}

func (a *App) PomodoroPlayChime() {
	if a.pomoApp != nil {
		a.pomoApp.PlayChime()
	}
}

// ── Pomodoro Profile & Settings methods (bound to Wails JS) ────────────────────

func (a *App) PomodoroLoadProfiles() []domain.Profile {
	if a.pomoApp != nil {
		return a.pomoApp.LoadProfiles()
	}
	return nil
}

func (a *App) PomodoroSaveProfile(p domain.Profile) error {
	if a.pomoApp != nil {
		return a.pomoApp.SaveProfile(p)
	}
	return nil
}

func (a *App) PomodoroGetProfileByID(id string) *domain.Profile {
	if a.pomoApp != nil {
		return a.pomoApp.GetProfileByID(id)
	}
	return nil
}

func (a *App) PomodoroDeleteProfile(id string) error {
	if a.pomoApp != nil {
		return a.pomoApp.DeleteProfile(id)
	}
	return nil
}

func (a *App) PomodoroGetSettings() domain.Settings {
	if a.pomoApp != nil {
		return a.pomoApp.GetSettings()
	}
	return domain.DefaultSettings()
}

func (a *App) PomodoroSaveSettings(s domain.Settings) error {
	if a.pomoApp != nil {
		return a.pomoApp.SaveSettings(s)
	}
	return nil
}

func (a *App) PomodoroGetStats() domain.StatsData {
	if a.pomoApp != nil {
		return a.pomoApp.GetStats()
	}
	return domain.StatsData{}
}

func (a *App) PomodoroRecordSessionComplete() domain.StatsData {
	if a.pomoApp != nil {
		return a.pomoApp.RecordSessionComplete()
	}
	return domain.StatsData{}
}

func (a *App) PomodoroPickMusicFile() (string, error) {
	if a.pomoApp != nil {
		return a.pomoApp.PickMusicFile()
	}
	return "", nil
}

func (a *App) PomodoroPickMusicFolder() (string, error) {
	if a.pomoApp != nil {
		return a.pomoApp.PickMusicFolder()
	}
	return "", nil
}
