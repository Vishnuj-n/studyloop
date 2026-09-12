package study

import (
	"sync"

	"ai-tutor/internal/db"
	"ai-tutor/internal/embeddings"
	llmpkg "ai-tutor/internal/llm"
	"ai-tutor/internal/retrieval"
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
	repo             *db.Repository
	fastLLMProvider  LLMProvider
	heavyLLMProvider LLMProvider
	retrievalEngine  *retrieval.Engine
	audioCacheMu     sync.RWMutex
	audioScriptCache map[string][]string
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

// selectLLM dynamically routes to heavy provider if context exceeds fast provider limits.
func (s *StudyService) selectLLM(contextText string) (LLMProvider, string) {
	if s.fastLLMProvider != nil {
		fastLimits := s.fastLLMProvider.GetLimits()
		// ponytail: 15% safety buffer accounts for prompt wrapper template overhead
		effectiveLimit := int(float64(fastLimits.MaxInputTokens) * 0.85)
		if effectiveLimit > 0 && s.heavyLLMProvider != nil {
			heavyLimits := s.heavyLLMProvider.GetLimits()
			// Only escalate if heavy tier has higher context limit or distinct model name
			isHeavyLarger := heavyLimits.MaxInputTokens > fastLimits.MaxInputTokens || s.heavyLLMProvider.ModelName() != s.fastLLMProvider.ModelName()
			if tokens, err := embeddings.CountTokens(contextText); err == nil && tokens > effectiveLimit && isHeavyLarger {
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
