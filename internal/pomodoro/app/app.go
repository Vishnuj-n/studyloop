package app

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"ai-tutor/internal/pomodoro/domain"
	"ai-tutor/internal/pomodoro/infra/events"
	"ai-tutor/internal/pomodoro/infra/storage"
	"ai-tutor/internal/pomodoro/services/audio"
	"ai-tutor/internal/pomodoro/services/persistence"
	"ai-tutor/internal/pomodoro/services/profile"
	"ai-tutor/internal/pomodoro/services/settings"
	"ai-tutor/internal/pomodoro/services/stats"
	"ai-tutor/internal/pomodoro/services/timer"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the root Wails application struct.
// It owns all services and exposes bound methods to the JS frontend.
type App struct {
	ctx         context.Context
	profiles    *profile.Service
	persistence *persistence.Service
	timer       *timer.Service
	audio       *audio.Service
	settings    *settings.Service
	stats       *stats.Service
}

// New creates and wires up all services.
func New(chimeData []byte) *App {
	dir := storage.DataDir()
	ps := persistence.New(dir)
	audioSvc := audio.New()
	audioSvc.SetChimeData(chimeData)
	return &App{
		profiles:    profile.New(dir),
		persistence: ps,
		timer:       timer.New(ps),
		audio:       audioSvc,
		settings:    settings.New(dir),
		stats:       stats.New(dir),
	}
}

// Startup is called by Wails after the window is ready.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	e := events.NewWailsEmitter(ctx)
	a.timer.SetEmitter(e)
	a.audio.SetEmitter(e)
	a.profiles.Load()
}

// DomReady is called after the front-end DOM has been loaded.
func (a *App) DomReady(ctx context.Context) {
	runtime.WindowCenter(ctx)
}

// ── Profile methods (bound to JS) ───────────────────────────────────────────

func (a *App) LoadProfiles() []domain.Profile {
	return a.profiles.Load()
}

func (a *App) SaveProfile(p domain.Profile) error {
	return a.profiles.Save(p)
}

func (a *App) GetProfileByID(id string) *domain.Profile {
	return a.profiles.GetByID(id)
}

func (a *App) DeleteProfile(id string) error {
	return a.profiles.Delete(id)
}

// ── File / folder pickers (bound to JS) ─────────────────────────────────────

func (a *App) PickMusicFile() (string, error) {
	srcPath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Select MP3 file",
		Filters: []runtime.FileFilter{{DisplayName: "MP3 Audio (*.mp3)", Pattern: "*.mp3"}},
	})
	if err != nil || srcPath == "" {
		return srcPath, err
	}

	// ponytail: copy picked file to app data directory so it remains valid across moving files
	audioDir := filepath.Join(storage.DataDir(), "audio")
	_ = os.MkdirAll(audioDir, 0755)

	destPath := filepath.Join(audioDir, filepath.Base(srcPath))
	if err := copyFile(srcPath, destPath); err != nil {
		return srcPath, nil // fallback to original path if copy fails
	}
	return destPath, nil
}

func copyFile(src, dst string) error {
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

func (a *App) PickMusicFolder() (string, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select music folder",
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

// ── Session persistence (bound to JS) ───────────────────────────────────────

func (a *App) CheckResumeSession() *domain.SessionState {
	return a.persistence.Load()
}

// ── Timer methods (bound to JS) ─────────────────────────────────────────────

func (a *App) StartTimer(profileID string, durationSec int) {
	a.timer.Start(profileID, durationSec)
}

func (a *App) ResumeTimer(state domain.SessionState) {
	a.timer.Resume(state)
}

func (a *App) PauseTimer() {
	a.timer.Pause()
}

func (a *App) StopTimer() {
	a.timer.Stop()
}

func (a *App) GetTimerState() map[string]interface{} {
	return a.timer.GetState()
}

// ── Audio methods (bound to JS) ─────────────────────────────────────────────

func (a *App) PlayLooping(filePath string) {
	a.audio.PlayLooping(filePath)
}

func (a *App) PlayShuffleFolder(folder string) {
	a.audio.PlayShuffleFolder(folder)
}

func (a *App) StopAudio() {
	a.audio.Stop()
}

func (a *App) SetVolume(v int) {
	a.audio.SetVolume(v)
}

func (a *App) GetAudioState() domain.AudioStatePayload {
	return a.audio.GetState()
}

// ── Stats methods (bound to JS) ─────────────────────────────────────────────

func (a *App) GetStats() domain.StatsData {
	return a.stats.GetStats()
}

func (a *App) RecordSessionComplete() domain.StatsData {
	return a.stats.RecordSessionComplete()
}

// ── Settings methods (bound to JS) ──────────────────────────────────────────

func (a *App) GetSettings() domain.Settings {
	return a.settings.Get()
}

func (a *App) SaveSettings(s domain.Settings) error {
	return a.settings.Save(s)
}

// ── Audio chime methods (bound to JS) ───────────────────────────────────────

func (a *App) PlayChime() {
	a.audio.PlayChime()
}
