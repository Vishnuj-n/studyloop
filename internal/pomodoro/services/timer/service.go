package timer

import (
	"context"
	"sync"
	"time"

	"ai-tutor/internal/pomodoro/domain"
	"ai-tutor/internal/pomodoro/infra/events"
	"ai-tutor/internal/pomodoro/services/persistence"
)

// Service manages the countdown timer and emits Wails events via an Emitter.
type Service struct {
	mu          sync.Mutex
	persistence *persistence.Service
	emitter     events.Emitter

	totalSec  int
	remainSec int
	profileID string
	running   bool
	cancel    context.CancelFunc
}

// New creates a Service. Call SetEmitter after the Wails context is available.
func New(ps *persistence.Service) *Service {
	return &Service{
		persistence: ps,
		emitter:     events.Noop{},
	}
}

// SetEmitter replaces the emitter (called from App.startup with the live Wails emitter).
func (s *Service) SetEmitter(e events.Emitter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emitter = e
}

// Start begins a new countdown for durationSec seconds.
func (s *Service) Start(profileID string, durationSec int) {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.profileID = profileID
	s.totalSec = durationSec
	s.remainSec = durationSec
	s.running = true
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.mu.Unlock()

	go s.run(ctx)
}

// Resume restarts the timer from a previously saved state.
func (s *Service) Resume(state domain.SessionState) {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.profileID = state.ProfileID
	s.totalSec = state.TotalSec
	s.remainSec = state.RemainingSec
	s.running = true
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.mu.Unlock()

	go s.run(ctx)
}

// Pause stops the tick loop while preserving remaining time.
func (s *Service) Pause() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.running = false
}

// Stop halts the timer and clears persisted state.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.running = false
	s.remainSec = s.totalSec
	s.persistence.Clear()
}

// GetState returns a current snapshot safe to send to the frontend.
func (s *Service) GetState() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]interface{}{
		"running":      s.running,
		"remainingSec": s.remainSec,
		"totalSec":     s.totalSec,
		"profileId":    s.profileID,
	}
}

// ── internal ─────────────────────────────────────────────────────────────────

func (s *Service) run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	autosave := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	defer autosave.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-autosave.C:
			s.mu.Lock()
			_ = s.persistence.Save(domain.SessionState{
				ProfileID:    s.profileID,
				TotalSec:     s.totalSec,
				RemainingSec: s.remainSec,
			})
			s.mu.Unlock()

		case <-ticker.C:
			s.mu.Lock()
			if s.remainSec > 0 {
				s.remainSec--
				remaining := s.remainSec
				profileID := s.profileID
				s.mu.Unlock()
				s.emitter.Emit("timerTicked", map[string]interface{}{
					"remainingSec": remaining,
					"profileId":    profileID,
				})
			} else {
				s.running = false
				profileID := s.profileID
				s.mu.Unlock()
				s.persistence.Clear()
				s.emitter.Emit("timerCompleted", map[string]interface{}{
					"profileId": profileID,
				})
				return
			}
		}
	}
}
