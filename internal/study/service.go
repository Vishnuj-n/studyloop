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

// Config wires all dependencies into StudyService via constructor injection.
type Config struct {
	Repo             *db.Repository
	FastLLMProvider  LLMProvider
	HeavyLLMProvider LLMProvider
	RetrievalEngine  *retrieval.Engine
}

// StudyService owns all study-mode generation and scoring logic.
type StudyService struct {
	repo                 *db.Repository
	fastLLMProvider      LLMProvider
	heavyLLMProvider     LLMProvider
	retrievalEngine      *retrieval.Engine
	audioCacheMu         sync.RWMutex
	audioScriptCache     map[string][]string
	rateLimitMu          sync.RWMutex
	fastRateLimitedUntil time.Time
}

// NewStudyService constructs a StudyService from injected dependencies.
func NewStudyService(cfg Config) *StudyService {
	if cfg.Repo == nil {
		panic("study service: repository is required")
	}
	return &StudyService{
		repo:             cfg.Repo,
		fastLLMProvider:  cfg.FastLLMProvider,
		heavyLLMProvider: cfg.HeavyLLMProvider,
		retrievalEngine:  cfg.RetrievalEngine,
		audioScriptCache: make(map[string][]string),
	}
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

// selectLLM dynamically routes to heavy provider if context exceeds fast provider limits or output budget,
// or if fast provider is currently in rate-limit cooldown.
func (s *StudyService) selectLLM(contextText string, requiredOutputBudget int) (LLMProvider, string) {
	if s.IsFastRateLimited() && s.heavyLLMProvider != nil {
		return s.heavyLLMProvider, "heavy"
	}
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

		outputExceeded := requiredOutputBudget > 0 && fastLimits.MaxOutputTokens < requiredOutputBudget

		if (inputExceeded || outputExceeded) && s.heavyLLMProvider != nil {
			heavyLimits := s.heavyLLMProvider.GetLimits()
			// Only escalate if heavy tier can satisfy budget or has distinct model name
			isHeavyLarger := heavyLimits.MaxInputTokens > fastLimits.MaxInputTokens || heavyLimits.MaxOutputTokens > fastLimits.MaxOutputTokens || s.heavyLLMProvider.ModelName() != s.fastLLMProvider.ModelName()
			if isHeavyLarger {
				return s.heavyLLMProvider, "heavy"
			}
		}
		return s.fastLLMProvider, "fast"
	}
	if s.heavyLLMProvider != nil {
		return s.heavyLLMProvider, "heavy"
	}
	return nil, "none"
}
