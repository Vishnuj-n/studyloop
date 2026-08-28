package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"ai-tutor/internal/embeddings"
	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"
)

func (r *Repository) queryChunks(filter string, args ...interface{}) ([]models.Chunk, error) {
	query := `
		SELECT id, topic_id, chunk_text, importance_score, weakness_score, page_num
		FROM chunks ` + filter + `
		ORDER BY page_num ASC, id ASC`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	return scanChunks(rows)
}

// GetChunksForTopicPageRange retrieves chunks for a topic within a page range.
func (r *Repository) GetChunksForTopicPageRange(topicID string, startPage, endPage int) ([]models.Chunk, error) {
	// Validate that either both bounds are provided or neither is
	if (startPage > 0) != (endPage > 0) {
		return nil, fmt.Errorf("both startPage and endPage must be provided together, or neither")
	}

	if startPage > 0 && endPage > 0 {
		return r.queryChunks("WHERE topic_id = ? AND page_num BETWEEN ? AND ?", topicID, startPage, endPage)
	}
	return r.queryChunks("WHERE topic_id = ?", topicID)
}

// GetChunksForTopic retrieves all chunks associated with a topic.
func (r *Repository) GetChunksForTopic(topicID string) ([]models.Chunk, error) {
	return r.queryChunks("WHERE topic_id = ?", topicID)
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

// GetReaderTopicBundle returns notebook metadata plus ordered sections with resolved page numbers.
// If notebookID is provided, section page mapping is scoped to that notebook.
func (r *Repository) GetReaderTopicBundle(topicID string, notebookID string) (*models.ReaderTopicBundle, error) {
	topicID = strings.TrimSpace(topicID)
	selectedNotebookID := strings.TrimSpace(notebookID)
	if topicID == "" {
		return nil, fmt.Errorf("topic ID is required")
	}

	bundle := &models.ReaderTopicBundle{
		TopicID:  topicID,
		Sections: []models.ReaderSection{},
	}

	var startPage int
	var endPage int
	if err := r.db.QueryRow(`
		SELECT title, COALESCE(start_page, 0), COALESCE(end_page, 0)
		FROM topics WHERE id = ?
	`, topicID).Scan(&bundle.TopicTitle, &startPage, &endPage); err != nil {
		return nil, err
	}
	// ponytail: ensure human-readable title for reader UI and session exports
	bundle.TopicTitle = utils.CleanTopicTitle(bundle.TopicTitle)
	bundle.TopicStartPage = startPage
	bundle.TopicEndPage = endPage

	var notebookIDRow sql.NullString
	var notebookTitle sql.NullString
	var filePath sql.NullString
	var fileType sql.NullString
	var pageCount sql.NullInt64
	var fileHashRow sql.NullString

	var err error
	if selectedNotebookID != "" {
		err = r.db.QueryRow(`
			SELECT id, title, file_path, file_type, COALESCE(page_count, 0), COALESCE(file_hash, '')
			FROM notebooks n
			WHERE n.id = ?
			  AND (
				n.topic_id = ?
				OR EXISTS (
					SELECT 1
					FROM notebook_chunks nc
					JOIN chunks c ON c.id = nc.chunk_id
					WHERE nc.notebook_id = n.id AND c.topic_id = ?
				)
			  )
			LIMIT 1
		`, selectedNotebookID, topicID, topicID).Scan(&notebookIDRow, &notebookTitle, &filePath, &fileType, &pageCount, &fileHashRow)
		if err == sql.ErrNoRows {
			// Fallback: If notebookID is explicitly provided (e.g. from task mode), look up notebook directly by ID.
			err = r.db.QueryRow(`
				SELECT id, title, file_path, file_type, COALESCE(page_count, 0), COALESCE(file_hash, '')
				FROM notebooks n
				WHERE n.id = ?
				LIMIT 1
			`, selectedNotebookID).Scan(&notebookIDRow, &notebookTitle, &filePath, &fileType, &pageCount, &fileHashRow)
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("selected notebook does not exist")
			}
		}
	} else {
		err = r.db.QueryRow(`
			SELECT id, title, file_path, file_type, page_count, file_hash
			FROM (
				SELECT
					n.id,
					n.title,
					n.file_path,
					n.file_type,
					COALESCE(n.page_count, 0) AS page_count,
					COALESCE(n.file_hash, '') AS file_hash,
					n.uploaded_at,
					0 AS rank
				FROM notebooks n
				WHERE n.topic_id = ?

				UNION

				SELECT
					n.id,
					n.title,
					n.file_path,
					n.file_type,
					COALESCE(n.page_count, 0) AS page_count,
					COALESCE(n.file_hash, '') AS file_hash,
					n.uploaded_at,
					1 AS rank
				FROM notebooks n
				JOIN notebook_chunks nc ON nc.notebook_id = n.id
				JOIN chunks c ON c.id = nc.chunk_id
				WHERE c.topic_id = ?
			)
			ORDER BY rank ASC, uploaded_at DESC, id ASC
			LIMIT 1
		`, topicID, topicID).Scan(&notebookIDRow, &notebookTitle, &filePath, &fileType, &pageCount, &fileHashRow)
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if notebookIDRow.Valid {
		bundle.NotebookID = notebookIDRow.String
	}
	if notebookTitle.Valid {
		bundle.NotebookTitle = notebookTitle.String
	}
	if fileHashRow.Valid {
		bundle.NotebookFileHash = fileHashRow.String
	}
	if filePath.Valid {
		// Convert filesystem path to URL path for the file server
		filename := filepath.Base(filePath.String)
		bundle.NotebookURL = "/notebooks/" + url.PathEscape(filename)
	}
	if fileType.Valid {
		bundle.FileType = fileType.String
	}
	if pageCount.Valid {
		bundle.PageCount = int(pageCount.Int64)
	}

	fetchSections := func(extraWhere string, args ...interface{}) ([]models.ReaderSection, error) {
		query := `
			SELECT
				c.id,
				'Page ' || CAST(COALESCE(c.page_num, 0) AS TEXT),
				c.chunk_text,
				COALESCE(c.page_num, 0),
				COALESCE(c.page_num, 0) AS page_num
			FROM chunks c ` + extraWhere + `
			ORDER BY c.page_num ASC, c.id ASC`
		if bundle.NotebookID != "" {
			query = `
				SELECT
					c.id,
					'Page ' || CAST(COALESCE(nc.page_num, 0) AS TEXT),
					c.chunk_text,
					COALESCE(nc.page_num, 0),
					COALESCE(nc.page_num, 0) AS page_num
				FROM chunks c
				JOIN notebook_chunks nc ON nc.chunk_id = c.id AND nc.notebook_id = ?
				` + extraWhere + `
				ORDER BY nc.page_num ASC, c.id ASC`
			args = append([]interface{}{bundle.NotebookID}, args...)
		}

		rows, err := r.db.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer func() { _ = rows.Close() }()

		var result []models.ReaderSection
		for rows.Next() {
			var section models.ReaderSection
			if err := rows.Scan(
				&section.ID,
				&section.Heading,
				&section.Content,
				&section.Order,
				&section.PageNum,
			); err != nil {
				return nil, err
			}
			result = append(result, section)
		}
		return result, rows.Err()
	}

	if bundle.NotebookID != "" {
		if sec, err := fetchSections("WHERE c.topic_id = ?", topicID); err == nil && len(sec) > 0 {
			bundle.Sections = sec
		} else if bundle.TopicStartPage > 0 && bundle.TopicEndPage >= bundle.TopicStartPage {
			// Fallback 1: Page range match if no sections match topic_id exactly
			if sec, err := fetchSections("WHERE nc.page_num >= ? AND nc.page_num <= ?", bundle.TopicStartPage, bundle.TopicEndPage); err == nil && len(sec) > 0 {
				bundle.Sections = sec
			}
		}
		// Fallback 2: All chunks in notebook if still 0 sections
		if len(bundle.Sections) == 0 {
			if sec, err := fetchSections(""); err == nil {
				bundle.Sections = sec
			}
		}
	} else {
		if sec, err := fetchSections("WHERE c.topic_id = ?", topicID); err == nil {
			bundle.Sections = sec
		}
	}

	// For text, markdown, and youtube files, load raw content directly from disk for instant rendering
	if !strings.EqualFold(bundle.FileType, "pdf") && filePath.Valid && filePath.String != "" {
		if contentBytes, err := os.ReadFile(filePath.String); err == nil {
			bundle.RawContent = string(contentBytes)
			if strings.EqualFold(bundle.FileType, "youtube") {
				var meta struct {
					VideoID  string `json:"video_id"`
					Chapters []struct {
						Idx   int    `json:"chapter_index"`
						Start int    `json:"start_seconds"`
						End   int    `json:"end_seconds"`
						Text  string `json:"transcript"`
					} `json:"chapters"`
				}
				if json.Unmarshal(contentBytes, &meta) == nil && meta.VideoID != "" {
					target := bundle.TopicStartPage
					if target <= 0 {
						target = 1
					}
					var ch struct {
						Idx   int    `json:"chapter_index"`
						Start int    `json:"start_seconds"`
						End   int    `json:"end_seconds"`
						Text  string `json:"transcript"`
					}
					for _, c := range meta.Chapters {
						if c.Idx == target || (target == 1 && c.Idx == 0) {
							ch = c
							break
						}
					}
					if ch.Idx == 0 && len(meta.Chapters) > 0 {
						ch = meta.Chapters[0]
					}
					bundle.NotebookURL = fmt.Sprintf("https://www.youtube-nocookie.com/embed/%s?enablejsapi=1&start=%d&end=%d", meta.VideoID, ch.Start, ch.End)
					if ch.Text != "" {
						bundle.RawContent = ch.Text
					}
				}
			}
		}
	}

	return bundle, nil
}

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

	var startPage int
	if topic.CurrentPageCursor <= 0 || topic.CurrentPageCursor < topic.StartPage {
		startPage = max(1, topic.StartPage)
	} else {
		startPage = topic.CurrentPageCursor + 1
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

