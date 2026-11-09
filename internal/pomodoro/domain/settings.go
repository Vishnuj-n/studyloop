package domain

// Settings holds global app preferences persisted to settings.json.
type Settings struct {
	Enabled             bool   `json:"enabled"`
	DefaultVolume       int    `json:"defaultVolume"` // 0-100
	AutoStartAudio      bool   `json:"autoStartAudio"`
	NotifyOnComplete    bool   `json:"notifyOnComplete"`
	AutoStartNextTimer  bool   `json:"autoStartNextTimer"`
	PlaySoundOnComplete bool   `json:"playSoundOnComplete"`
	Theme               string `json:"theme"` // "dark" | "ocean" | "forest" | "minimal-black"
}


