package domain

// Profile defines a named timer configuration.
type Profile struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	DurationSec      int    `json:"durationSec"`      // work session length in seconds
	MusicPath        string `json:"musicPath"`        // work music: file or folder
	Shuffle          bool   `json:"shuffle"`          // true = shuffle work music folder
	BreakDurationSec int    `json:"breakDurationSec"` // break length (0 = no break)
	BreakMusicPath   string `json:"breakMusicPath"`   // break music: file or folder (empty = silent)
	BreakShuffle     bool   `json:"breakShuffle"`     // true = shuffle break music folder
	IsDefault        bool   `json:"isDefault"`        // selected automatically on startup
}
