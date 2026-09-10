package domain

// AudioPlaybackState represents whether audio is idle, playing, or stopped.
type AudioPlaybackState string

const (
	AudioIdle    AudioPlaybackState = "idle"
	AudioPlaying AudioPlaybackState = "playing"
	AudioStopped AudioPlaybackState = "stopped"
)

// AudioStatePayload is emitted via the "audioStateChanged" Wails event.
type AudioStatePayload struct {
	State     AudioPlaybackState `json:"state"`
	TrackName string             `json:"trackName"`
	TrackInfo string             `json:"trackInfo"` // e.g. "Shuffle folder · 12 tracks"
}
