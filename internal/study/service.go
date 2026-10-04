package study

import (
	"fmt"
	"sync"
	"time"

	"ai-tutor/internal/db"
	"ai-tutor/internal/embeddings"
	llmpkg "ai-tutor/internal/llm"
	"ai-tutor/internal/retrieval"
	"ai-tutor/internal/utils"
)

// LLMProvider is the minimal interface both LLM tiers satisfy.
type LLMProvider interface {
	GenerateAnswer(prompt string) (string, error)
	ModelName() string
	GetLimits() llmpkg.ModelLimits
}

// pacedProvider decorates an LLMProvider to monitor invocation times.
type pacedProvider struct {
	LLMProvider
	onCalled func()
}

func (p *pacedProvider) GenerateAnswer(prompt string) (string, error) {
	if p.onCalled != nil {
		p.onCalled()
	}
	return p.LLMProvider.GenerateAnswer(prompt)
}

// Config wires all dependencies into StudyService via constructor injection.
type Config struct {
	Repo             *db.Repository
	FastLLMProvider  LLMProvider
	HeavyLLMProvider LLMProvider
	RetrievalEngine  *retrieval.Engine
	NotesDir         string
}

// StudyService owns all study-mode generation and scoring logic.
type StudyService struct {
	repo                 *db.Repository
	fastLLMProvider      LLMProvider
	heavyLLMProvider     LLMProvider
	retrievalEngine      *retrieval.Engine
	notesDir             string
	audioCacheMu         sync.RWMutex
	audioScriptCache     map[string][]string
	rateLimitMu          sync.RWMutex
	fastRateLimitedUntil time.Time
	lastFastCallTime     time.Time
	inFlightCompressMu   sync.Mutex
	inFlightCompress     map[string]bool
	inFlightNotesMu      sync.Mutex
	inFlightNotes        map[string]bool
}

// NewStudyService constructs a StudyService from injected dependencies.
func NewStudyService(cfg Config) *StudyService {
	if cfg.Repo == nil {
		panic("study service: repository is required")
	}
	s := &StudyService{
		repo:             cfg.Repo,
		heavyLLMProvider: cfg.HeavyLLMProvider,
		retrievalEngine:  cfg.RetrievalEngine,
		notesDir:         cfg.NotesDir,
		audioScriptCache: make(map[string][]string),
		inFlightCompress: make(map[string]bool),
		inFlightNotes:    make(map[string]bool),
	}
	if cfg.FastLLMProvider != nil {
		s.fastLLMProvider = &pacedProvider{
			LLMProvider: cfg.FastLLMProvider,
			onCalled:    s.markFastCalled,
		}
	}
	return s
}

func (s *StudyService) markFastCalled() {
	s.rateLimitMu.Lock()
	defer s.rateLimitMu.Unlock()
	s.lastFastCallTime = time.Now()
}

func (s *StudyService) timeSinceLastFastCall() time.Duration {
	s.rateLimitMu.RLock()
	defer s.rateLimitMu.RUnlock()
	if s.lastFastCallTime.IsZero() {
		return time.Hour // large duration if never called
	}
	return time.Since(s.lastFastCallTime)
}

func (s *StudyService) isPacingEnabled() bool {
	if s.repo == nil {
		return false
	}
	settings, err := s.repo.GetUserSettings()
	if err != nil || settings == nil {
		return false
	}
	return settings.RateLimitStrategy == "PACED"
}

// MarkFastRateLimited sets a temporary cooldown duration for the fast LLM tier.
func (s *StudyService) MarkFastRateLimited(cooldownDuration time.Duration) {
	s.rateLimitMu.Lock()
	defer s.rateLimitMu.Unlock()
	s.fastRateLimitedUntil = time.Now().Add(cooldownDuration)
	utils.Warnf("[STUDY_SERVICE] Fast LLM tier marked rate-limited until %s (cooldown=%s)", s.fastRateLimitedUntil.Format(time.RFC3339), cooldownDuration)
}

// IsFastRateLimited returns whether the fast LLM provider is currently in cooldown.
func (s *StudyService) IsFastRateLimited() bool {
	s.rateLimitMu.RLock()
	defer s.rateLimitMu.RUnlock()
	return time.Now().Before(s.fastRateLimitedUntil)
}

// FormatLLMError checks if an error is a rate limit error (HTTP 429), triggers fast tier cooldown if appropriate,
// and returns a user-transparent error message.
func (s *StudyService) FormatLLMError(err error, tier string) error {
	if err == nil {
		return nil
	}
	if llmpkg.IsRateLimitError(err) {
		if tier == "fast" || tier == "" {
			s.MarkFastRateLimited(45 * time.Second)
		}
		if s.heavyLLMProvider != nil && (s.fastLLMProvider == nil || s.heavyLLMProvider.ModelName() != s.fastLLMProvider.ModelName()) {
			return fmt.Errorf("Fast provider rate-limited (status 429). Please try again — next attempt will use Heavy provider (%s).", s.heavyLLMProvider.ModelName())
		}
		return fmt.Errorf("Fast provider rate-limited (status 429: TPM limit reached). Please try again in a few seconds, or configure a Heavy provider (e.g. Gemini AI Studio) in Settings.")
	}
	return err
}

// selectLLM dynamically routes to heavy provider if context exceeds fast provider input limits,
// or if fast provider is currently in rate-limit cooldown, or if PACED smoothing strategy applies.
func (s *StudyService) selectLLM(contextText string) (LLMProvider, string) {
	s.rateLimitMu.Lock()
	defer s.rateLimitMu.Unlock()

	// 1. 429 Cooldown check (always applies)
	if time.Now().Before(s.fastRateLimitedUntil) && s.heavyLLMProvider != nil {
		return s.heavyLLMProvider, "heavy"
	}

	// 2. PACED strategy check (if enabled by user setting)
	if s.isPacingEnabled() && s.heavyLLMProvider != nil {
		if s.fastLLMProvider == nil || s.heavyLLMProvider.ModelName() != s.fastLLMProvider.ModelName() {
			timeSince := time.Hour
			if !s.lastFastCallTime.IsZero() {
				timeSince = time.Since(s.lastFastCallTime)
			}
			if timeSince < 60*time.Second {
				utils.Infof("[STUDY_SERVICE] Paced smoothing active: fast tier called %v ago (<60s). Escalating to heavy tier.", timeSince.Round(time.Millisecond))
				return s.heavyLLMProvider, "heavy"
			}
		}
	}

	// 3. Normal token limit & fallback logic
	if s.fastLLMProvider != nil {
		fastLimits := s.fastLLMProvider.GetLimits()
		// ponytail: 15% safety buffer accounts for prompt wrapper template overhead
		effectiveInputLimit := int(float64(fastLimits.MaxInputTokens) * 0.85)

		inputExceeded := false
		if effectiveInputLimit > 0 {
			if tokens, err := embeddings.CountTokens(contextText); err == nil && tokens > effectiveInputLimit {
				inputExceeded = true
			}
		}

		if inputExceeded && s.heavyLLMProvider != nil {
			heavyLimits := s.heavyLLMProvider.GetLimits()
			// Only escalate if heavy tier has larger input limit or distinct model name
			isHeavyLarger := heavyLimits.MaxInputTokens > fastLimits.MaxInputTokens || s.heavyLLMProvider.ModelName() != s.fastLLMProvider.ModelName()
			if isHeavyLarger {
				return s.heavyLLMProvider, "heavy"
			}
		}
		s.lastFastCallTime = time.Now()
		return s.fastLLMProvider, "fast"
	}
	if s.heavyLLMProvider != nil {
		return s.heavyLLMProvider, "heavy"
	}
	return nil, "none"
}
