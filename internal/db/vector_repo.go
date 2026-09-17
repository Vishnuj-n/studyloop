package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"ai-tutor/internal/utils"
)

// ChunkVectorBatchItem contains one vector persistence request.
type ChunkVectorBatchItem struct {
	ChunkID      string
	Vector       []float32
	EmbeddingRef string
}

// UpsertChunkVector stores or updates a chunk embedding vector.
func (r *Repository) UpsertChunkVector(chunkID string, vector []float32) error {
	chunkID = strings.TrimSpace(chunkID)
	if chunkID == "" {
		return fmt.Errorf("chunk id is required")
	}
	if len(vector) == 0 {
		return fmt.Errorf("vector is required")
	}

	if len(vector) != int(r.embeddingDimension) {
		return fmt.Errorf("vector dimension mismatch: got %d, expected %d", len(vector), r.embeddingDimension)
	}

	vectorJSON, err := r.vectorToJSON(vector)
	if err != nil {
		return fmt.Errorf("failed to encode vector: %w", err)
	}

	rowID, err := r.lookupChunkRowID(chunkID)
	if err != nil {
		return fmt.Errorf("failed to resolve chunk rowid for %s: %w", chunkID, err)
	}

	var exists int
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM chunk_vectors WHERE rowid = ?
	`, rowID).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if exists > 0 {
		_, err = r.db.Exec(`
			UPDATE chunk_vectors SET embedding = ? WHERE rowid = ?
		`, vectorJSON, rowID)
		return err
	}

	_, err = r.db.Exec(`
		INSERT INTO chunk_vectors (rowid, embedding) VALUES (?, ?)
	`, rowID, vectorJSON)
	return err
}

// UpsertChunkVectorsBatch stores vectors and embedding refs in a single transaction.
func (r *Repository) UpsertChunkVectorsBatch(items []ChunkVectorBatchItem) error {
	if len(items) == 0 {
		return nil
	}

	return r.withTx(func(tx *sql.Tx) error {
		// Prepare statements to prevent re-compilation in the loop
		stmtGetRowID, err := tx.Prepare(`SELECT rowid FROM chunks WHERE id = ?`)
		if err != nil {
			return fmt.Errorf("failed to prepare stmtGetRowID: %w", err)
		}
		defer func() {
			_ = stmtGetRowID.Close()
		}()

		stmtCheckExists, err := tx.Prepare(`SELECT COUNT(*) FROM chunk_vectors WHERE rowid = ?`)
		if err != nil {
			return fmt.Errorf("failed to prepare stmtCheckExists: %w", err)
		}
		defer func() {
			_ = stmtCheckExists.Close()
		}()

		stmtUpdateVector, err := tx.Prepare(`UPDATE chunk_vectors SET embedding = ? WHERE rowid = ?`)
		if err != nil {
			return fmt.Errorf("failed to prepare stmtUpdateVector: %w", err)
		}
		defer func() {
			_ = stmtUpdateVector.Close()
		}()

		stmtInsertVector, err := tx.Prepare(`INSERT INTO chunk_vectors (rowid, embedding) VALUES (?, ?)`)
		if err != nil {
			return fmt.Errorf("failed to prepare stmtInsertVector: %w", err)
		}
		defer func() {
			_ = stmtInsertVector.Close()
		}()

		stmtUpdateRef, err := tx.Prepare(`UPDATE chunks SET embedding_ref = ? WHERE id = ?`)
		if err != nil {
			return fmt.Errorf("failed to prepare stmtUpdateRef: %w", err)
		}
		defer func() {
			_ = stmtUpdateRef.Close()
		}()

		for _, item := range items {
			item.ChunkID = strings.TrimSpace(item.ChunkID)
			if item.ChunkID == "" {
				return fmt.Errorf("chunk id is required for each batch item")
			}
			if len(item.Vector) == 0 {
				return fmt.Errorf("vector is required for each batch item")
			}

			if len(item.Vector) != int(r.embeddingDimension) {
				return fmt.Errorf("vector dimension mismatch for chunk %s: got %d, expected %d", item.ChunkID, len(item.Vector), r.embeddingDimension)
			}

			vectorJSON, encodeErr := r.vectorToJSON(item.Vector)
			if encodeErr != nil {
				return fmt.Errorf("failed to encode vector for chunk %s: %w", item.ChunkID, encodeErr)
			}

			var rowID int64
			if scanErr := stmtGetRowID.QueryRow(item.ChunkID).Scan(&rowID); scanErr != nil {
				return fmt.Errorf("failed to resolve chunk rowid for %s: %w", item.ChunkID, scanErr)
			}

			var exists int
			countErr := stmtCheckExists.QueryRow(rowID).Scan(&exists)
			if countErr != nil && countErr != sql.ErrNoRows {
				return countErr
			}

			if exists > 0 {
				if _, execErr := stmtUpdateVector.Exec(vectorJSON, rowID); execErr != nil {
					return execErr
				}
			} else {
				if _, execErr := stmtInsertVector.Exec(rowID, vectorJSON); execErr != nil {
					return execErr
				}
			}

			if item.EmbeddingRef != "" {
				if _, execErr := stmtUpdateRef.Exec(item.EmbeddingRef, item.ChunkID); execErr != nil {
					return execErr
				}
			}
		}
		return nil
	})
}

// SearchVectorsForTopic finds the top-k most similar vectors for a topic-scoped query.
// When startPage and endPage are positive, search is context-locked to that page window.
func (r *Repository) SearchVectorsForTopic(topicID string, queryVector []float32, k int, startPage int, endPage int) ([]string, error) {
	topicID = strings.TrimSpace(topicID)
	filterByPage := startPage > 0 && endPage > 0
	if filterByPage && startPage > endPage {
		startPage, endPage = endPage, startPage
	}

	rowidQuery := `
		SELECT rowid, id
		FROM chunks
		WHERE topic_id = ?
	`
	rowidArgs := []interface{}{topicID}
	if filterByPage {
		rowidQuery += " AND page_num BETWEEN ? AND ?"
		rowidArgs = append(rowidArgs, startPage, endPage)
	}

	return r.searchVectors(
		"SearchVectorsForTopic",
		"topicID",
		topicID,
		"topic",
		queryVector,
		k,
		rowidQuery,
		rowidArgs,
		"startPage", startPage, "endPage", endPage, "filterByPage", filterByPage,
	)
}

// SearchVectorsForNotebook finds the top-k most similar vectors for a notebook-scoped query.
func (r *Repository) SearchVectorsForNotebook(notebookID string, queryVector []float32, k int) ([]string, error) {
	notebookID = strings.TrimSpace(notebookID)
	rowidQuery := `
		SELECT DISTINCT c.rowid, c.id
		FROM notebook_chunks nc
		JOIN chunks c ON c.id = nc.chunk_id
		WHERE nc.notebook_id = ?
	`
	return r.searchVectors(
		"SearchVectorsForNotebook",
		"notebookID",
		notebookID,
		"notebook",
		queryVector,
		k,
		rowidQuery,
		[]interface{}{notebookID},
	)
}

func (r *Repository) searchVectors(
	funcName string,
	scopeKey string,
	scopeID string,
	scopeTag string,
	queryVector []float32,
	k int,
	prefilterSQL string,
	prefilterArgs []interface{},
	extraLogKv ...interface{},
) ([]string, error) {
	reqLog := []interface{}{"vector_repo: " + funcName + " requested", scopeKey, scopeID, "k", k, "embeddingDimension", r.embeddingDimension, "queryVectorLen", len(queryVector)}
	reqLog = append(reqLog, extraLogKv...)
	utils.RagLogger.Info(reqLog[0].(string), reqLog[1:]...)

	if scopeID == "" {
		return nil, fmt.Errorf("%s id is required", scopeTag)
	}
	if len(queryVector) == 0 {
		return nil, fmt.Errorf("query vector is required")
	}
	if k <= 0 || k > maxRetrievalK {
		return nil, fmt.Errorf("k must be between 1 and %d", maxRetrievalK)
	}

	if r.embeddingDimension <= 0 {
		utils.RagLogger.Warn("vector_repo: "+funcName+" skipped, embedding dimension not initialized", scopeKey, scopeID)
		return []string{}, nil
	}

	if len(queryVector) != int(r.embeddingDimension) {
		utils.RagLogger.Error("vector_repo: "+funcName+" dimension mismatch", "got", len(queryVector), "expected", r.embeddingDimension)
		return nil, fmt.Errorf("query vector dimension mismatch: got %d, expected %d", len(queryVector), r.embeddingDimension)
	}

	queryVectorJSON, err := r.vectorToJSON(queryVector)
	if err != nil {
		return nil, fmt.Errorf("failed to encode query vector: %w", err)
	}

	rowRows, err := r.db.Query(prefilterSQL, prefilterArgs...)
	if err != nil {
		return nil, fmt.Errorf("chunk prefilter failed: %w", err)
	}
	defer func() {
		_ = rowRows.Close()
	}()

	allowedChunkByRowID := make(map[int64]string)
	allowedRowIDs := make([]int64, 0)
	for rowRows.Next() {
		var rowID int64
		var chunkID string
		if scanErr := rowRows.Scan(&rowID, &chunkID); scanErr != nil {
			return nil, scanErr
		}
		allowedChunkByRowID[rowID] = chunkID
		allowedRowIDs = append(allowedRowIDs, rowID)
	}
	if err := rowRows.Err(); err != nil {
		return nil, err
	}
	if len(allowedRowIDs) == 0 {
		noChunkLog := []interface{}{"vector_repo: " + funcName + ": no chunks found matching filter", scopeKey, scopeID}
		noChunkLog = append(noChunkLog, extraLogKv...)
		utils.RagLogger.Info(noChunkLog[0].(string), noChunkLog[1:]...)
		return []string{}, nil
	}

	allowedRowIDsJSON, err := json.Marshal(allowedRowIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to encode allowed row ids: %w", err)
	}

	vectorArgs := []interface{}{string(allowedRowIDsJSON), queryVectorJSON, k}
	vectorSQL := `
		SELECT rowid
		FROM chunk_vectors
		WHERE rowid IN (SELECT CAST(value AS INTEGER) FROM json_each(?))
		ORDER BY vec_distance_cosine(embedding, ?) ASC
		LIMIT ?
	`

	utils.RagLogger.Info("vector_repo: executing "+funcName+" vector query", scopeKey, scopeID, "allowedRowIDsCount", len(allowedRowIDs))
	rows, err := r.db.Query(vectorSQL, vectorArgs...)
	if err != nil {
		if isVectorUnavailableError(err) {
			utils.RagLogger.Warn("vector search unavailable, using lexical fallback", "scope", scopeTag, scopeKey, scopeID, "error", err)
			return []string{}, nil
		}
		utils.RagLogger.Error("vector_repo: "+funcName+" query execution failed", scopeKey, scopeID, "error", err)
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	chunkIDs := make([]string, 0, k)
	for rows.Next() {
		var rowID int64
		if err := rows.Scan(&rowID); err != nil {
			return nil, err
		}
		if chunkID, ok := allowedChunkByRowID[rowID]; ok {
			chunkIDs = append(chunkIDs, chunkID)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	utils.RagLogger.Info("vector_repo: "+funcName+" completed successfully", scopeKey, scopeID, "resultsCount", len(chunkIDs))
	return chunkIDs, nil
}

func (r *Repository) vectorToJSON(vector []float32) (string, error) {
	if len(vector) == 0 {
		return "[]", nil
	}

	values := make([]float64, len(vector))
	for i, value := range vector {
		values[i] = float64(value)
	}

	encoded, err := json.Marshal(values)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}

func (r *Repository) lookupChunkRowID(chunkID string) (int64, error) {
	var rowID int64
	if err := r.db.QueryRow(`
		SELECT rowid FROM chunks WHERE id = ?
	`, chunkID).Scan(&rowID); err != nil {
		return 0, err
	}

	return rowID, nil
}

func isVectorUnavailableError(err error) bool {
	if err == nil {
		return false
	}

	errText := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errText, "no such module: vec0"):
		return true
	case strings.Contains(errText, "no such table: chunk_vectors"):
		return true
	case strings.Contains(errText, "no such function: distance"):
		return true
	case strings.Contains(errText, "no such function: vec_distance_cosine"):
		return true
	default:
		return false
	}
}

// createVectorTable creates the vec0 virtual table with the discovered embedding dimension.
func (r *Repository) createVectorTable() error {
	if r.embeddingDimension <= 0 {
		return fmt.Errorf("embedding dimension not initialized")
	}

	var existingSQL string
	err := r.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='chunk_vectors'`).Scan(&existingSQL)
	if err == nil && existingSQL != "" {
		expectedCol := fmt.Sprintf("float[%d]", r.embeddingDimension)
		if !strings.Contains(existingSQL, expectedCol) {
			utils.Warnf("chunk_vectors table dimension mismatch, rebuilding table for dimension %d", r.embeddingDimension)
			if _, dropErr := r.db.Exec(`DROP TABLE IF EXISTS chunk_vectors`); dropErr != nil {
				return fmt.Errorf("failed to drop existing chunk_vectors table: %w", dropErr)
			}
			// Reset embedding refs so stored vectors are reindexed
			if _, resetErr := r.db.Exec(`UPDATE chunks SET embedding_ref = NULL`); resetErr != nil {
				utils.Warnf("failed to reset chunk embedding_refs: %v", resetErr)
			}
		}
	}

	// Create vec0 virtual table for vector search
	schema := fmt.Sprintf(`
		CREATE VIRTUAL TABLE IF NOT EXISTS chunk_vectors USING vec0(
			embedding float[%d]
		);
	`, r.embeddingDimension)

	_, err = r.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create vec0 table: %w", err)
	}

	utils.Infof("Created vec0 virtual table with embedding dimension %d", r.embeddingDimension)
	return nil
}

// UpdateChunkEmbedding updates the embedding_ref (hash) for a chunk to track changes.
func (r *Repository) UpdateChunkEmbedding(chunkID string, hash string) error {
	res, err := r.db.Exec(`
		UPDATE chunks SET embedding_ref = ? WHERE id = ?
	`, hash, chunkID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no rows inserted for chunk_id %s", chunkID)
	}
	return nil
}

// ChunkEmbeddingBatchItem represents a chunk embedding update to be processed in batch
type ChunkEmbeddingBatchItem struct {
	ChunkID string
	Hash    string
}

// UpdateChunkEmbeddingsBatch updates embedding metadata for multiple chunks in a single transaction
func (r *Repository) UpdateChunkEmbeddingsBatch(items []ChunkEmbeddingBatchItem) error {
	if len(items) == 0 {
		return nil
	}

	return r.withTx(func(tx *sql.Tx) error {
		stmt, err := tx.Prepare(`
			UPDATE chunks SET embedding_ref = ? WHERE id = ?
		`)
		if err != nil {
			return err
		}
		defer func() {
			_ = stmt.Close()
		}()

		for _, item := range items {
			if item.ChunkID == "" {
				return fmt.Errorf("chunk id is required for all batch items")
			}

			res, err := stmt.Exec(item.Hash, item.ChunkID)
			if err != nil {
				return err
			}
			rowsAffected, err := res.RowsAffected()
			if err != nil {
				return err
			}
			if rowsAffected == 0 {
				return fmt.Errorf("no rows inserted for chunk_id %s", item.ChunkID)
			}
		}
		return nil
	})
}

// GetChunkEmbeddingRefsForTopic returns embedding_ref values for all chunks in a topic.
func (r *Repository) GetChunkEmbeddingRefsForTopic(topicID string) (map[string]string, error) {
	rows, err := r.db.Query(`
		SELECT id, COALESCE(embedding_ref, '')
		FROM chunks
		WHERE topic_id = ?
	`, topicID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			utils.Warnf("failed to close chunk embedding refs rows: %v", closeErr)
		}
	}()

	refs := make(map[string]string)
	for rows.Next() {
		var chunkID string
		var hash string
		if err := rows.Scan(&chunkID, &hash); err != nil {
			return nil, err
		}
		refs[chunkID] = hash
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return refs, nil
}
