package db

import (
	"database/sql"
	"fmt"
	"strings"

	"ai-tutor/internal/embeddings"
	"ai-tutor/internal/models"
)

// GetChunksForTopicPageRange retrieves chunks for a topic within a page range.
func (r *Repository) GetChunksForTopicPageRange(topicID string, startPage, endPage int) ([]models.Chunk, error) {
	query := `
		SELECT id, topic_id, chunk_text, importance_score, weakness_score, page_num
		FROM chunks
		WHERE topic_id = ?`

	var args []interface{}
	args = append(args, topicID)

	// Validate that either both bounds are provided or neither is
	if (startPage > 0) != (endPage > 0) {
		return nil, fmt.Errorf("both startPage and endPage must be provided together, or neither")
	}

	if startPage > 0 && endPage > 0 {
		query += " AND page_num BETWEEN ? AND ?"
		args = append(args, startPage, endPage)
	}

	query += " ORDER BY page_num ASC, id ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanChunks(rows)
}

// GetChunksForTopic retrieves all chunks associated with a topic.
func (r *Repository) GetChunksForTopic(topicID string) ([]models.Chunk, error) {
	rows, err := r.db.Query(`
		SELECT id, topic_id, chunk_text, importance_score, weakness_score, page_num
		FROM chunks
		WHERE topic_id = ?
		ORDER BY page_num ASC, id ASC
	`, topicID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanChunks(rows)
}

// GetChunksForNotebook retrieves all chunks associated with a notebook.
func (r *Repository) GetChunksForNotebook(notebookID string) ([]models.Chunk, error) {
	rows, err := r.db.Query(`
		SELECT c.id, c.topic_id, c.chunk_text, c.importance_score, c.weakness_score, nc.page_num
		FROM chunks c
		JOIN notebook_chunks nc ON nc.chunk_id = c.id
		WHERE nc.notebook_id = ?
		ORDER BY nc.page_num ASC, c.id ASC
	`, notebookID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanChunks(rows)
}

// GetChunksForTopics batches chunk loading for multiple topics.
func (r *Repository) GetChunksForTopics(topicIDs []string) (map[string][]models.Chunk, error) {
	if len(topicIDs) == 0 {
		return make(map[string][]models.Chunk), nil
	}

	placeholders := make([]string, len(topicIDs))
	args := make([]interface{}, len(topicIDs))
	for i, id := range topicIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, topic_id, chunk_text, importance_score, weakness_score, page_num
		FROM chunks
		WHERE topic_id IN (%s)
		ORDER BY page_num ASC, id ASC
	`, strings.Join(placeholders, ","))

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	chunksByTopic := make(map[string][]models.Chunk)
	for rows.Next() {
		var chunk models.Chunk
		if err := rows.Scan(&chunk.ID, &chunk.TopicID, &chunk.Text, &chunk.ImportanceScore, &chunk.WeaknessScore, &chunk.PageNum); err != nil {
			return nil, err
		}
		chunksByTopic[chunk.TopicID] = append(chunksByTopic[chunk.TopicID], chunk)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return chunksByTopic, nil
}

// GetTotalChunkTokens returns estimated total tokens for one topic.
// It prefers stored token_count values and falls back to len(chunk_text)/4 when token_count is zero or missing.
func (r *Repository) GetTotalChunkTokens(topicID string) (int, error) {
	return r.getTotalChunkTokens(topicID, 0, 0)
}

// GetTotalChunkTokensForPageRange returns estimated total tokens for one topic/page window.
// It prefers stored token_count values and falls back to len(chunk_text)/4 when token_count is zero or missing.
func (r *Repository) GetTotalChunkTokensForPageRange(topicID string, startPage int, endPage int) (int, error) {
	return r.getTotalChunkTokens(topicID, startPage, endPage)
}

// GetTokensPerPageMap returns a map of page number to total tokens for that page within a page range.
// It prefers stored token_count values and falls back to len(chunk_text)/4 when token_count is zero or missing.
// This uses a single GROUP BY query to avoid N+1 query problems when scanning multiple pages.
func (r *Repository) GetTokensPerPageMap(topicID string, startPage int, endPage int) (map[int]int, error) {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return nil, fmt.Errorf("topic id is required")
	}
	if startPage <= 0 || endPage <= 0 {
		return nil, fmt.Errorf("start page and end page must be positive")
	}
	if startPage > endPage {
		startPage, endPage = endPage, startPage
	}

	query := `
		SELECT page_num, COALESCE(token_count, 0), COALESCE(chunk_text, '')
		FROM chunks
		WHERE topic_id = ?
		  AND page_num BETWEEN ? AND ?
		ORDER BY page_num
	`

	rows, err := r.db.Query(query, topicID, startPage, endPage)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	result := make(map[int]int)
	for rows.Next() {
		var pageNum int
		var tokenCount int
		var chunkText string
		if err := rows.Scan(&pageNum, &tokenCount, &chunkText); err != nil {
			return nil, err
		}

		pageTotal := 0
		if tokenCount > 0 {
			pageTotal = tokenCount
		} else {
			count, err := embeddings.CountTokens(chunkText)
			if err != nil {
				// Fall back to estimation if tokenizer unavailable
				pageTotal = len(chunkText) / 4
				if pageTotal <= 0 && strings.TrimSpace(chunkText) != "" {
					pageTotal = 1
				}
			} else {
				pageTotal = count
			}
		}

		result[pageNum] += pageTotal
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetTopicWordsInRange returns the total words/tokens across chunks in a topic and page range using a single aggregate query.
func (r *Repository) GetTopicWordsInRange(topicID string, startPage, endPage int) (int, error) {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return 0, fmt.Errorf("topic id is required")
	}
	if startPage <= 0 || endPage <= 0 {
		return 0, fmt.Errorf("start page and end page must be positive")
	}
	if startPage > endPage {
		startPage, endPage = endPage, startPage
	}

	query := `
		SELECT COALESCE(SUM(
			CASE
				WHEN token_count > 0 THEN token_count
				WHEN length(chunk_text) > 0 THEN length(chunk_text)/4
				ELSE 0
			END
		), 0)
		FROM chunks
		WHERE topic_id = ?
		  AND page_num BETWEEN ? AND ?
	`
	var totalWords int
	err := r.db.QueryRow(query, topicID, startPage, endPage).Scan(&totalWords)
	if err != nil {
		return 0, err
	}
	return totalWords, nil
}

func (r *Repository) getTotalChunkTokens(topicID string, startPage int, endPage int) (int, error) {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return 0, fmt.Errorf("topic id is required")
	}

	// Validate page bounds
	var filterByPage bool
	if startPage == 0 && endPage == 0 {
		// No page filter - entire topic
		filterByPage = false
	} else if startPage <= 0 || endPage <= 0 {
		// Mixed positive/negative bounds are invalid
		return 0, fmt.Errorf("invalid page bounds: both startPage and endPage must be positive or both must be zero")
	} else {
		// Both bounds are positive - filter by page range
		filterByPage = true
		if startPage > endPage {
			startPage, endPage = endPage, startPage
		}
	}

	query := `
		SELECT COALESCE(token_count, 0), COALESCE(chunk_text, '')
		FROM chunks
		WHERE topic_id = ?
	`
	args := []interface{}{topicID}
	if filterByPage {
		query += ` AND page_num BETWEEN ? AND ?`
		args = append(args, startPage, endPage)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = rows.Close()
	}()

	total := 0
	for rows.Next() {
		var tokenCount int
		var chunkText string
		if err := rows.Scan(&tokenCount, &chunkText); err != nil {
			return 0, err
		}

		if tokenCount > 0 {
			total += tokenCount
			continue
		}

		count, err := embeddings.CountTokens(chunkText)
		if err != nil {
			// Fall back to estimation if tokenizer unavailable
			fallback := len(chunkText) / 4
			if fallback <= 0 && strings.TrimSpace(chunkText) != "" {
				fallback = 1
			}
			total += fallback
		} else {
			total += count
		}
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}

	return total, nil
}

// GetChunkTextsForTopicPageRange returns chunk_text rows ordered by chunk id for one topic/page window.
func (r *Repository) GetChunkTextsForTopicPageRange(topicID string, startPage int, endPage int) ([]string, error) {
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return nil, fmt.Errorf("topic id is required")
	}
	if startPage <= 0 || endPage <= 0 {
		return nil, fmt.Errorf("start page and end page must be positive")
	}
	if startPage > endPage {
		startPage, endPage = endPage, startPage
	}

	rows, err := r.db.Query(`
		SELECT chunk_text
		FROM chunks
		WHERE topic_id = ?
		  AND page_num BETWEEN ? AND ?
		ORDER BY page_num ASC, id ASC
	`, topicID, startPage, endPage)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var chunkTexts []string
	for rows.Next() {
		var chunkText string
		if err := rows.Scan(&chunkText); err != nil {
			return nil, err
		}
		chunkTexts = append(chunkTexts, chunkText)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return chunkTexts, nil
}

// GetTopicHeadingPageRanges returns resolved page bounds per chunk ID for a topic.
func (r *Repository) GetTopicHeadingPageRanges(topicID string) (map[string][2]int, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return nil, fmt.Errorf("topic id is required")
	}

	rows, err := r.db.Query(`
		SELECT id, COALESCE(page_num, 0)
		FROM chunks
		WHERE topic_id = ?
	`, topicID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	result := make(map[string][2]int)
	for rows.Next() {
		var id string
		var pageNum int
		if err := rows.Scan(&id, &pageNum); err != nil {
			return nil, err
		}
		if id == "" {
			continue
		}
		result[id] = [2]int{pageNum, pageNum}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetAllChunks retrieves all chunks in the system.
func (r *Repository) GetAllChunks() ([]models.Chunk, error) {
	rows, err := r.db.Query(`
		SELECT id, topic_id, chunk_text, importance_score, weakness_score, page_num
		FROM chunks
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanChunks(rows)
}

func scanChunks(rows *sql.Rows) ([]models.Chunk, error) {
	var chunks []models.Chunk
	for rows.Next() {
		var chunk models.Chunk
		if err := rows.Scan(&chunk.ID, &chunk.TopicID, &chunk.Text, &chunk.ImportanceScore, &chunk.WeaknessScore, &chunk.PageNum); err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return chunks, nil
}

// GetPageBoundaryCosineSimilarityTx computes the cosine similarity between the last chunk of pageA and the first chunk of pageB for a topic.
// Returns 0.0 (no error) if vectors or sqlite-vec function are unavailable.
func (r *Repository) GetPageBoundaryCosineSimilarityTx(tx *sql.Tx, topicID string, pageA, pageB int) (float64, error) {
	var rowIDA, rowIDB int64
	errA := tx.QueryRow(`
		SELECT rowid FROM chunks 
		WHERE topic_id = ? AND page_num = ? 
		ORDER BY id DESC LIMIT 1
	`, topicID, pageA).Scan(&rowIDA)

	errB := tx.QueryRow(`
		SELECT rowid FROM chunks 
		WHERE topic_id = ? AND page_num = ? 
		ORDER BY id ASC LIMIT 1
	`, topicID, pageB).Scan(&rowIDB)

	if errA != nil || errB != nil {
		return 0.0, nil
	}

	var distance float64
	query := `
		SELECT vec_distance_cosine(cvA.embedding, cvB.embedding)
		FROM chunk_vectors cvA
		JOIN chunk_vectors cvB ON cvB.rowid = ?
		WHERE cvA.rowid = ?
	`
	err := tx.QueryRow(query, rowIDB, rowIDA).Scan(&distance)
	if err != nil {
		return 0.0, nil // Safe fallback on error or missing vec extension
	}

	similarity := 1.0 - distance
	if similarity < 0 {
		similarity = 0
	} else if similarity > 1.0 {
		similarity = 1.0
	}
	return similarity, nil
}

