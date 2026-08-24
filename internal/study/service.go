package study

import (
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
	}
}

// selectLLM dynamically routes to heavy provider if context exceeds fast provider limits.
func (s *StudyService) selectLLM(contextText string) (LLMProvider, string) {
	if s.fastLLMProvider != nil {
		limits := s.fastLLMProvider.GetLimits()
		if limits.MaxInputTokens > 0 {
			if tokens, err := embeddings.CountTokens(contextText); err == nil && tokens > limits.MaxInputTokens && s.heavyLLMProvider != nil {
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
