package study

import (
	"fmt"
	"strings"
	"ai-tutor/internal/retrieval"
)

type ReaderRetrievalScope string

const (
	ReaderScopeCurrentPage    ReaderRetrievalScope = "current_page"
	ReaderScopeCurrentChapter ReaderRetrievalScope = "current_chapter"
	ReaderScopeEntireNotebook ReaderRetrievalScope = "entire_notebook"
)

type ReaderAIRequest struct {
	TopicID          string
	NotebookID       string
	Question         string
	Scope            ReaderRetrievalScope
	CurrentPage      int
	ChapterStartPage int
	ChapterEndPage   int
}

func (s *StudyService) AnswerReaderQuestion(req ReaderAIRequest) map[string]interface{} {
	req.TopicID = strings.TrimSpace(req.TopicID)
	req.NotebookID = strings.TrimSpace(req.NotebookID)
	req.Question = strings.TrimSpace(req.Question)

	if req.TopicID == "" {
		return map[string]interface{}{"error": "topic ID is required"}
	}
	if req.Question == "" {
		return map[string]interface{}{"error": "question is required"}
	}
	if s.fastLLMProvider == nil {
		return map[string]interface{}{"error": "FAST_LLM provider not initialized"}
	}
	if s.retrievalEngine == nil {
		return map[string]interface{}{"error": "retrieval engine not initialized"}
	}

	scope := normalizeReaderScope(req.Scope)
	results, err := s.searchReaderScope(req, scope)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if len(results) == 0 {
		return map[string]interface{}{"error": "no relevant content found in the selected reading scope"}
	}

	contextText, citations := buildReaderContext(results)
	scopeLabel := readerScopeLabel(scope)
	prompt := fmt.Sprintf(`You are the Reader sidebar AI: a lightweight, context-aware reading companion.
Use the retrieved reading material below to answer the student's question.
If the answer is not supported by the selected scope, reply exactly: "I couldn’t find a strong answer within the selected scope. Try expanding the retrieval scope."

Rules:
- Keep the response concise and grounded.
- Explain in simple, clear, and easy-to-understand language, avoiding overly academic or complex jargon.
- If the student asks to explain a specific paragraph, passage, or concept from the material, focus on explaining the underlying logic, meaning, and rationale of the passage (e.g., why something is done or how it works) rather than just describing a specific tool or example mentioned within it.
- Prefer 2 short paragraphs or up to 3 bullets.
- Explain clearly, but do not turn this into a full tutoring session.
- Stay anchored to the student's current reading flow and selected scope.

Selected retrieval scope: %s

Retrieved reading material:
%s

Student question: %s

Answer:`, scopeLabel, contextText, req.Question)

	answer, err := s.fastLLMProvider.GenerateAnswer(prompt)
	if err != nil {
		return map[string]interface{}{"error": "reader response failed: " + err.Error()}
	}

	return map[string]interface{}{
		"answer":         answer,
		"cited_sections": citations,
		"scope":          string(scope),
	}
}

func (s *StudyService) searchReaderScope(req ReaderAIRequest, scope ReaderRetrievalScope) ([]retrieval.SearchResult, error) {
	const topK = 5

	switch scope {
	case ReaderScopeCurrentPage:
		if req.CurrentPage <= 0 {
			return nil, fmt.Errorf("current page is required for current-page retrieval")
		}
		return s.retrievalEngine.SemanticSearch(req.TopicID, req.Question, topK, req.CurrentPage, req.CurrentPage)
	case ReaderScopeEntireNotebook:
		if req.NotebookID == "" {
			return nil, fmt.Errorf("notebook ID is required for notebook-wide retrieval")
		}
		return s.retrievalEngine.SemanticSearchNotebook(req.NotebookID, req.TopicID, req.Question, topK)
	case ReaderScopeCurrentChapter:
		startPage := req.ChapterStartPage
		endPage := req.ChapterEndPage
		if startPage <= 0 || endPage <= 0 {
			var err error
			startPage, endPage, err = s.repo.GetTopicPageBounds(req.TopicID)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve chapter bounds: %w", err)
			}
		}
		if startPage > 0 && endPage > 0 {
			return s.retrievalEngine.SemanticSearch(req.TopicID, req.Question, topK, startPage, endPage)
		}
		return s.retrievalEngine.SemanticSearch(req.TopicID, req.Question, topK, 0, 0)
	default:
		return nil, fmt.Errorf("unsupported reader retrieval scope: %s", scope)
	}
}

func normalizeReaderScope(scope ReaderRetrievalScope) ReaderRetrievalScope {
	switch scope {
	case ReaderScopeCurrentPage, ReaderScopeCurrentChapter:
		return scope
	default:
		return ReaderScopeEntireNotebook
	}
}

func readerScopeLabel(scope ReaderRetrievalScope) string {
	switch scope {
	case ReaderScopeCurrentPage:
		return "Current Page"
	case ReaderScopeCurrentChapter:
		return "Current Chapter"
	default:
		return "Entire Notebook"
	}
}

func buildReaderContext(results []retrieval.SearchResult) (string, []string) {
	blocks, citations := buildReaderContextBlocks(results)
	return strings.TrimSpace(strings.Join(blocks, "\n\n")), citations
}

// buildReaderContextBlocks returns the sequence of section blocks (as strings)
// and a parallel list of citation labels. This allows callers to truncate the
// context while keeping citations synchronized to included blocks.
func buildReaderContextBlocks(results []retrieval.SearchResult) ([]string, []string) {
	blocks, citations, _ := buildReaderContextBlocksWithText(results)
	return blocks, citations
}

func buildReaderContextBlocksWithText(results []retrieval.SearchResult) ([]string, []string, []string) {
	blocks := make([]string, 0, len(results))
	citations := make([]string, 0, len(results))
	chunkTexts := make([]string, 0, len(results))

	for _, result := range results {
		text := strings.TrimSpace(result.Text)
		if text == "" {
			continue
		}

		if result.PageNum > 0 {
			blocks = append(blocks, fmt.Sprintf("[Page %d]\n%s", result.PageNum, text))
			citations = append(citations, fmt.Sprintf("Page %d", result.PageNum))
		} else {
			blocks = append(blocks, text)
			citations = append(citations, "")
		}
		chunkTexts = append(chunkTexts, text)
	}

	return blocks, citations, chunkTexts
}
