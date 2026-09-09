package stats

import (
	"path/filepath"
	"sync"
	"time"

	"ai-tutor/internal/pomodoro/domain"
	"ai-tutor/internal/pomodoro/infra/storage"
)

// Service tracks completed sessions per day and a running streak.
type Service struct {
	mu       sync.Mutex
	data     domain.StatsData
	filePath string
}

// New creates and initialises a Service.
func New(dataDir string) *Service {
	ss := &Service{
		filePath: filepath.Join(dataDir, "stats.json"),
	}
	ss.load()
	ss.rolloverIfNeededLocked()
	return ss
}

// GetStats returns the current stats snapshot (triggers day rollover if needed).
func (ss *Service) GetStats() domain.StatsData {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.rolloverIfNeededLocked()
	return ss.data
}

// RecordSessionComplete increments today's count and updates the streak.
func (ss *Service) RecordSessionComplete() domain.StatsData {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	today := todayStr()
	ss.rolloverIfNeededLocked()

	ss.data.SessionsToday++

	switch ss.data.LastActiveDate {
	case "":
		ss.data.Streak = 1
	case today:
		// Already recorded today — streak unchanged
	default:
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		if ss.data.LastActiveDate == yesterday {
			ss.data.Streak++
		} else {
			ss.data.Streak = 1
		}
	}
	ss.data.LastActiveDate = today
	ss.save()
	return ss.data
}

// ── internal ──────────────────────────────────────────────────────────────────

func todayStr() string {
	return time.Now().Format("2006-01-02")
}

// rolloverIfNeededLocked resets SessionsToday when the calendar day changes.
// It also resets the streak if more than one day has passed since last activity.
// Must be called with ss.mu held.
func (ss *Service) rolloverIfNeededLocked() {
	today := todayStr()
	if ss.data.Date != today {
		ss.data.Date = today
		ss.data.SessionsToday = 0

		// Check if streak should be reset (more than one day since last activity)
		if ss.data.LastActiveDate != "" {
			yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
			if ss.data.LastActiveDate != yesterday {
				ss.data.Streak = 0
			}
		}

		ss.save()
	}
}

func (ss *Service) load() {
	if err := storage.Load(ss.filePath, &ss.data); err != nil {
		ss.data = domain.StatsData{Date: todayStr()}
	}
}

func (ss *Service) save() {
	_ = storage.Save(ss.filePath, ss.data)
}
