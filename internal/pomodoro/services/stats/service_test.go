package stats

import (
	"path/filepath"
	"testing"
	"time"
)

func newSvc(t *testing.T) *Service {
	t.Helper()
	// Construct via unexported fields — same package access
	ss := &Service{filePath: filepath.Join(t.TempDir(), "stats.json")}
	ss.load()
	ss.rolloverIfNeededLocked()
	return ss
}

func TestInitZeroStats(t *testing.T) {
	ss := newSvc(t)
	got := ss.GetStats()
	if got.SessionsToday != 0 {
		t.Errorf("SessionsToday: want 0, got %d", got.SessionsToday)
	}
	if got.Streak != 0 {
		t.Errorf("Streak: want 0, got %d", got.Streak)
	}
	if got.Date == "" {
		t.Error("Date must not be empty after init")
	}
}

func TestRecordFirstSession(t *testing.T) {
	ss := newSvc(t)
	got := ss.RecordSessionComplete()
	if got.SessionsToday != 1 {
		t.Errorf("SessionsToday: want 1, got %d", got.SessionsToday)
	}
	if got.Streak != 1 {
		t.Errorf("Streak after first session: want 1, got %d", got.Streak)
	}
}

func TestRecordMultipleSameDay(t *testing.T) {
	ss := newSvc(t)
	ss.RecordSessionComplete()
	got := ss.RecordSessionComplete()
	if got.SessionsToday != 2 {
		t.Errorf("SessionsToday after 2 records: want 2, got %d", got.SessionsToday)
	}
	if got.Streak != 1 {
		t.Errorf("Streak same-day: want 1, got %d", got.Streak)
	}
}

func TestStreakIncrementsOnConsecutiveDay(t *testing.T) {
	ss := newSvc(t)
	ss.RecordSessionComplete()

	// Simulate "next day": set LastActiveDate to yesterday
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	ss.data.LastActiveDate = yesterday
	ss.data.SessionsToday = 0
	ss.data.Date = time.Now().Format("2006-01-02")

	got := ss.RecordSessionComplete()
	if got.Streak != 2 {
		t.Errorf("Consecutive day: want streak 2, got %d", got.Streak)
	}
}

func TestStreakResetsAfterGap(t *testing.T) {
	ss := newSvc(t)
	ss.data.LastActiveDate = time.Now().AddDate(0, 0, -2).Format("2006-01-02")
	ss.data.Streak = 5

	got := ss.RecordSessionComplete()
	if got.Streak != 1 {
		t.Errorf("After gap: want streak 1, got %d", got.Streak)
	}
}

func TestRolloverResetsSessionsToday(t *testing.T) {
	ss := newSvc(t)
	ss.data.Date = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	ss.data.SessionsToday = 5
	ss.data.Streak = 3

	got := ss.GetStats() // triggers rollover
	if got.SessionsToday != 0 {
		t.Errorf("After rollover: want 0, got %d", got.SessionsToday)
	}
	if got.Streak != 3 {
		t.Errorf("Rollover must not change streak; got %d", got.Streak)
	}
	today := time.Now().Format("2006-01-02")
	if got.Date != today {
		t.Errorf("Date after rollover: want %s, got %s", today, got.Date)
	}
}

func TestPersistence(t *testing.T) {
	fp := filepath.Join(t.TempDir(), "stats.json")

	s1 := &Service{filePath: fp}
	s1.load()
	s1.rolloverIfNeededLocked()
	s1.RecordSessionComplete()
	s1.RecordSessionComplete()

	s2 := &Service{filePath: fp}
	s2.load()
	s2.rolloverIfNeededLocked()
	got := s2.GetStats()
	if got.SessionsToday != 2 {
		t.Errorf("Persistence: want 2 sessions, got %d", got.SessionsToday)
	}
}

func TestTenSessionsSameDay(t *testing.T) {
	ss := newSvc(t)
	for i := 0; i < 10; i++ {
		ss.RecordSessionComplete()
	}
	got := ss.GetStats()
	if got.SessionsToday != 10 {
		t.Errorf("Want 10 sessions, got %d", got.SessionsToday)
	}
	if got.Streak != 1 {
		t.Errorf("Same-day streak: want 1, got %d", got.Streak)
	}
}
