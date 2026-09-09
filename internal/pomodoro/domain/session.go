package domain

// SessionState is persisted to state.json so a session survives restarts.
// SavedAt is a Unix timestamp (int64) to avoid Wails binding issues with time.Time.
type SessionState struct {
	ProfileID    string `json:"profileId"`
	TotalSec     int    `json:"totalSec"`
	RemainingSec int    `json:"remainingSec"`
	SavedAt      int64  `json:"savedAt"`
}

// StatsData holds daily session counts and a running streak, persisted to stats.json.
type StatsData struct {
	Date           string `json:"date"` // today as "YYYY-MM-DD"
	SessionsToday  int    `json:"sessionsToday"`
	LastActiveDate string `json:"lastActiveDate"` // last day a session completed
	Streak         int    `json:"streak"`
}
