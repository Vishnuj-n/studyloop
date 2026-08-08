package db

import (
	"ai-tutor/internal/models"
)

const (
	FallbackWordsPerPage = 500
	MaxPageScanLimit     = 100
	ClampWindowPages     = 4
)

// ResolvePageWindow resolves the start and end page window for a reading topic cursor based on token budget.
func ResolvePageWindow(
	topic models.ReadingTopicCursor,
	tokenBudget int,
	queryTokensPerPageMap func(topicID string, startPage int, endPage int) (map[int]int, error),
) (int, int, bool, map[int]int) {

	if topic.EndPage <= 0 {
		return 0, 0, false, nil
	}

	if tokenBudget <= 0 {
		return 0, 0, false, nil
	}

	startPage := topic.CurrentPageCursor
	if startPage < 1 {
		startPage = max(1, topic.StartPage)
	}
	if topic.StartPage > 0 && startPage < topic.StartPage {
		startPage = topic.StartPage
	}

	if startPage > topic.EndPage {
		return 0, 0, false, nil
	}

	endPage := startPage
	accumulatedWords := 0

	tokenMap := make(map[int]int)
	if queryTokensPerPageMap != nil {
		if fetchedMap, err := queryTokensPerPageMap(topic.ID, startPage, topic.EndPage); err == nil && fetchedMap != nil {
			tokenMap = fetchedMap
		}
	}

	for page := startPage; page <= topic.EndPage; page++ {
		if page-startPage >= MaxPageScanLimit {
			break
		}

		pageWords, ok := tokenMap[page]
		if !ok || pageWords <= 0 {
			pageWords = FallbackWordsPerPage
		}

		if accumulatedWords+pageWords > tokenBudget && accumulatedWords > 0 {
			break
		}

		accumulatedWords += pageWords
		endPage = page
	}

	if topic.EndPage-endPage <= ClampWindowPages {
		endPage = topic.EndPage
	}

	if endPage < startPage {
		return 0, 0, false, nil
	}

	return startPage, endPage, true, tokenMap
}
