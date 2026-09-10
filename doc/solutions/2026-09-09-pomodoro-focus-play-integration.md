# Solution: FocusPlay Pomodoro & Ambient Audio Integration

## Overview
Integrated the native **FocusPlay** Pomodoro timer, ambient audio streaming engine (`gopxl/beep`), and profile management into **Studyloop** on the `feature/pomodoro-focus-mode` branch. The integration provides a lightweight, distraction-free study timer with local music streaming directly in the desktop app.

---

## Changes Made

### 1. Backend Subfolder Import & Wails Bridge
- **[internal/pomodoro/](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/pomodoro)**:
  - Direct, modular import of FocusPlay backend services:
    - `timer`: Goroutine countdown ticker with non-blocking event broadcasting (`timerTicked`, `timerCompleted`).
    - `audio`: Pure Go MP3 decoder and speaker output via `github.com/gopxl/beep` with logarithmic gain volume scaling.
    - `persistence`: Auto-save session checkpoints in app storage (`%APPDATA%/Studyloop/pomodoro` or `dev_data/pomodoro`).
    - `stats`: Daily session counter and streak tracking.
- **[internal/pomodoro/app/app.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/pomodoro/app/app.go)**:
  - Enhanced `PickMusicFile()` to automatically copy selected MP3 tracks into the local app data directory (`pomodoro/audio/`), ensuring persistent playback even if original user files are moved or deleted.
- **[internal/app/app_pomodoro.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/app/app_pomodoro.go)**:
  - Bound typed Wails methods (`PomodoroStartTimer`, `PomodoroPauseTimer`, `PomodoroResumeTimer`, `PomodoroStopTimer`, `PomodoroPlayLooping`, `PomodoroPlayShuffleFolder`, `PomodoroStopAudio`, `PomodoroSetVolume`, `PomodoroGetSettings`, `PomodoroSaveSettings`, `PomodoroPickMusicFile`, `PomodoroPickMusicFolder`).
- **[internal/app/app.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/app/app.go)**:
  - Initialized `pomoApp` on startup and guaranteed clean shutdown of audio streams and timers.

### 2. Frontend Sidebar Widget & Settings Pane
- **[frontend/src/services/pomodoroApi.js](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/services/pomodoroApi.js)**:
  - Typed JS service for invoking Pomodoro backend methods.
- **[frontend/src/components/PomodoroWidget.vue](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/components/PomodoroWidget.vue)**:
  - Native Studyloop styled sidebar widget with remaining time (`MM:SS`), dynamic progress bar, Work / Break status badge, Play/Pause/Stop/Skip actions, live track info, and audio mute toggle.
- **[frontend/src/components/Sidebar.vue](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/components/Sidebar.vue)**:
  - Embedded `PomodoroWidget` above bottom navigation.
- **[frontend/src/components/SettingsStudyBudget.vue](file:///c:/Users/vishn/PROJECT/ai-tutor/frontend/src/components/SettingsStudyBudget.vue)**:
  - Added **"Pomodoro & Focus Audio"** section under *Study Budget & Routine*:
    - Global show/hide toggle for sidebar widget.
    - Work and break duration minute inputs.
    - Single MP3 file picker (Free) vs. Folder Lo-Fi Shuffling (Pro).
    - Dynamic volume slider (0–100%).

### 3. Free vs. Pro Feature Entitlement
- **Free Tier**: Single Pomodoro timer (configurable work/break minutes), background ticking, completion chimes, and single MP3 looping.
- **Pro Tier**: Lo-Fi Folder Shuffling across full study music directories, guarded by both client-side modal trigger (`openBilling()`) and authoritative backend session checks (`a.IsProUser()`) on `PomodoroPickMusicFolder` and `PomodoroPlayShuffleFolder`.

---

## Verification Results

### Automated Tests
1. **Whole-Repository Short Test Suite**:
   ```pwsh
   go test -short ./internal/...
   ```
   *Output*: All packages passed (`internal/app`, `internal/pomodoro/...`, `internal/study`, etc.).

2. **Frontend Production Build**:
   ```pwsh
   npm run build
   ```
   *Output*: Built without errors or warnings.
