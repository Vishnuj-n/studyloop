package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ChunkRecord struct {
	ID        string
	DocID     string
	ChunkIdx  int
	Content   string
	PageNum   int
	CreatedAt string
}

func generateSyntheticChunks(count int) []ChunkRecord {
	records := make([]ChunkRecord, count)
	now := time.Now().UTC().Format(time.RFC3339)
	for i := 0; i < count; i++ {
		records[i] = ChunkRecord{
			ID:        fmt.Sprintf("chunk-%06d-%d", i, rand.Intn(1000000)),
			DocID:     "doc-sample-test",
			ChunkIdx:  i,
			Content:   fmt.Sprintf("This is synthetic chunk #%d with representative token density for retrieval benchmarking. It describes concurrent go routines and SQLite WAL properties.", i),
			PageNum:   (i / 5) + 1,
			CreatedAt: now,
		}
	}
	return records
}

func setupTempDB(walMode bool) (*sql.DB, string, error) {
	tempDir, err := os.MkdirTemp("", "sqlite_bench_*")
	if err != nil {
		return nil, "", err
	}
	dbPath := filepath.Join(tempDir, "bench.db")
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return nil, "", err
	}

	if walMode {
		_, _ = db.Exec("PRAGMA journal_mode=WAL;")
		_, _ = db.Exec("PRAGMA synchronous=NORMAL;")
	}

	schema := `
	CREATE TABLE chunks (
		id TEXT PRIMARY KEY,
		doc_id TEXT NOT NULL,
		chunk_index INTEGER NOT NULL,
		content TEXT NOT NULL,
		page_number INTEGER NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE INDEX idx_chunks_doc ON chunks(doc_id, chunk_index);
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, "", err
	}

	return db, tempDir, nil
}

// Strategy 1: Individual auto-commit inserts (slowest)
func benchmarkIndividualAutoCommit(db *sql.DB, records []ChunkRecord) (time.Duration, error) {
	t0 := time.Now()
	for _, r := range records {
		_, err := db.Exec(
			"INSERT INTO chunks (id, doc_id, chunk_index, content, page_number, created_at) VALUES (?, ?, ?, ?, ?, ?)",
			r.ID, r.DocID, r.ChunkIdx, r.Content, r.PageNum, r.CreatedAt,
		)
		if err != nil {
			return 0, err
		}
	}
	return time.Since(t0), nil
}

// Strategy 2: Single explicit transaction with prepared statement
func benchmarkSingleTxPrepared(db *sql.DB, records []ChunkRecord) (time.Duration, error) {
	t0 := time.Now()
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	stmt, err := tx.Prepare("INSERT INTO chunks (id, doc_id, chunk_index, content, page_number, created_at) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	defer stmt.Close()

	for _, r := range records {
		if _, err := stmt.Exec(r.ID, r.DocID, r.ChunkIdx, r.Content, r.PageNum, r.CreatedAt); err != nil {
			_ = tx.Rollback()
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return time.Since(t0), nil
}

// Strategy 3: Chunked transactions (Batch size = 500)
func benchmarkChunkedTx(db *sql.DB, records []ChunkRecord, batchSize int) (time.Duration, error) {
	t0 := time.Now()
	total := len(records)
	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}
		slice := records[i:end]

		tx, err := db.Begin()
		if err != nil {
			return 0, err
		}
		stmt, err := tx.Prepare("INSERT INTO chunks (id, doc_id, chunk_index, content, page_number, created_at) VALUES (?, ?, ?, ?, ?, ?)")
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		for _, r := range slice {
			if _, err := stmt.Exec(r.ID, r.DocID, r.ChunkIdx, r.Content, r.PageNum, r.CreatedAt); err != nil {
				stmt.Close()
				_ = tx.Rollback()
				return 0, err
			}
		}
		stmt.Close()
		if err := tx.Commit(); err != nil {
			return 0, err
		}
	}
	return time.Since(t0), nil
}

// Strategy 4: Multi-row VALUES (?, ?, ?), (?, ?, ?) batches
func benchmarkMultiRowValues(db *sql.DB, records []ChunkRecord, batchSize int) (time.Duration, error) {
	t0 := time.Now()
	total := len(records)
	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}
		slice := records[i:end]

		var valuePlaceholders []string
		var args []any
		for _, r := range slice {
			valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?)")
			args = append(args, r.ID, r.DocID, r.ChunkIdx, r.Content, r.PageNum, r.CreatedAt)
		}

		query := fmt.Sprintf(
			"INSERT INTO chunks (id, doc_id, chunk_index, content, page_number, created_at) VALUES %s",
			strings.Join(valuePlaceholders, ","),
		)

		tx, err := db.Begin()
		if err != nil {
			return 0, err
		}
		if _, err := tx.Exec(query, args...); err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		if err := tx.Commit(); err != nil {
			return 0, err
		}
	}
	return time.Since(t0), nil
}

func main() {
	const totalRecords = 5000
	const autoCommitSample = 500 // Limit autocommit sample to prevent minutes of disk sync

	fmt.Println(strings.Repeat("=", 85))
	fmt.Println("SQLITE BULK WRITE & TRANSACTION STRATEGY BENCHMARK")
	fmt.Printf("Dataset Size : %d chunk records\n", totalRecords)
	fmt.Println(strings.Repeat("=", 85))

	records := generateSyntheticChunks(totalRecords)

	type BenchResult struct {
		Name     string
		Duration time.Duration
		Rate     float64
		Speedup  string
	}
	var results []BenchResult
	var baselineDur time.Duration

	// Test 1: Auto-commit (extrapolated from sample)
	{
		db, tempDir, err := setupTempDB(false)
		if err != nil {
			fmt.Printf("setupTempDB error: %v\n", err)
			return
		}
		dur, err := benchmarkIndividualAutoCommit(db, records[:autoCommitSample])
		db.Close()
		os.RemoveAll(tempDir)
		if err != nil {
			fmt.Printf("Test 1 error: %v\n", err)
			return
		}

		extrapolatedDur := dur * (totalRecords / autoCommitSample)
		baselineDur = extrapolatedDur
		rate := float64(totalRecords) / extrapolatedDur.Seconds()
		results = append(results, BenchResult{
			Name:     fmt.Sprintf("Auto-Commit (1 statement/tx, Extrapolated %d rows)", totalRecords),
			Duration: extrapolatedDur,
			Rate:     rate,
			Speedup:  "1.00x",
		})
		fmt.Printf("[1/5] Auto-Commit (1 row/tx)         : %v (%.1f rows/s, 1.00x)\n", extrapolatedDur.Round(time.Millisecond), rate)
	}

	// Test 2: Single Tx with Prepared Statement (Delete Mode)
	{
		db, tempDir, err := setupTempDB(false)
		if err != nil {
			fmt.Printf("setupTempDB error: %v\n", err)
			return
		}
		dur, err := benchmarkSingleTxPrepared(db, records)
		db.Close()
		os.RemoveAll(tempDir)
		if err != nil {
			fmt.Printf("Test 2 error: %v\n", err)
			return
		}

		rate := float64(totalRecords) / dur.Seconds()
		speedup := float64(baselineDur) / float64(dur)
		results = append(results, BenchResult{
			Name:     "Single Transaction + Prepared Stmt (DELETE mode)",
			Duration: dur,
			Rate:     rate,
			Speedup:  fmt.Sprintf("%.1fx", speedup),
		})
		fmt.Printf("[2/5] Single Tx + Prepared (DELETE)  : %v (%.1f rows/s, %.1fx)\n", dur.Round(time.Millisecond), rate, speedup)
	}

	// Test 3: Single Tx with Prepared Statement + WAL Mode
	{
		db, tempDir, err := setupTempDB(true)
		if err != nil {
			fmt.Printf("setupTempDB error: %v\n", err)
			return
		}
		dur, err := benchmarkSingleTxPrepared(db, records)
		db.Close()
		os.RemoveAll(tempDir)
		if err != nil {
			fmt.Printf("Test 3 error: %v\n", err)
			return
		}

		rate := float64(totalRecords) / dur.Seconds()
		speedup := float64(baselineDur) / float64(dur)
		results = append(results, BenchResult{
			Name:     "Single Transaction + Prepared Stmt (WAL mode)",
			Duration: dur,
			Rate:     rate,
			Speedup:  fmt.Sprintf("%.1fx", speedup),
		})
		fmt.Printf("[3/5] Single Tx + Prepared (WAL)     : %v (%.1f rows/s, %.1fx)\n", dur.Round(time.Millisecond), rate, speedup)
	}

	// Test 4: Chunked Transactions (WAL mode, batch=500)
	{
		db, tempDir, err := setupTempDB(true)
		if err != nil {
			fmt.Printf("setupTempDB error: %v\n", err)
			return
		}
		dur, err := benchmarkChunkedTx(db, records, 500)
		db.Close()
		os.RemoveAll(tempDir)
		if err != nil {
			fmt.Printf("Test 4 error: %v\n", err)
			return
		}

		rate := float64(totalRecords) / dur.Seconds()
		speedup := float64(baselineDur) / float64(dur)
		results = append(results, BenchResult{
			Name:     "Chunked Transactions (Batch=500, WAL mode)",
			Duration: dur,
			Rate:     rate,
			Speedup:  fmt.Sprintf("%.1fx", speedup),
		})
		fmt.Printf("[4/5] Chunked Tx (Batch=500, WAL)   : %v (%.1f rows/s, %.1fx)\n", dur.Round(time.Millisecond), rate, speedup)
	}

	// Test 5: Multi-row VALUES batches (WAL mode, batch=250)
	{
		db, tempDir, err := setupTempDB(true)
		if err != nil {
			fmt.Printf("setupTempDB error: %v\n", err)
			return
		}
		dur, err := benchmarkMultiRowValues(db, records, 250)
		db.Close()
		os.RemoveAll(tempDir)
		if err != nil {
			fmt.Printf("Test 5 error: %v\n", err)
			return
		}

		rate := float64(totalRecords) / dur.Seconds()
		speedup := float64(baselineDur) / float64(dur)
		results = append(results, BenchResult{
			Name:     "Multi-Row VALUES Batches (Batch=250, WAL mode)",
			Duration: dur,
			Rate:     rate,
			Speedup:  fmt.Sprintf("%.1fx", speedup),
		})
		fmt.Printf("[5/5] Multi-Row VALUES (Batch=250)  : %v (%.1f rows/s, %.1fx)\n", dur.Round(time.Millisecond), rate, speedup)
	}

	// Summary Table
	fmt.Println("\n" + strings.Repeat("=", 85))
	fmt.Println("BENCHMARK RESULTS SUMMARY")
	fmt.Println(strings.Repeat("=", 85))
	fmt.Printf("%-52s | %-10s | %-10s | %-8s\n", "Strategy / Mode", "Time", "Throughput", "Speedup")
	fmt.Println(strings.Repeat("-", 85))

	bestDur := results[0].Duration
	for _, r := range results {
		if r.Duration < bestDur {
			bestDur = r.Duration
		}
	}

	for _, r := range results {
		winner := ""
		if r.Duration == bestDur {
			winner = " 🏆"
		}
		fmt.Printf("%-52s | %9v | %8.1f r/s | %8s%s\n", r.Name, r.Duration.Round(time.Millisecond), r.Rate, r.Speedup, winner)
	}
	fmt.Println(strings.Repeat("=", 85))
}
