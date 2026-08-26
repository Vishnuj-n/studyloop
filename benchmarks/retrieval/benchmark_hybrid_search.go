package main

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	VectorDim  = 384
	CorpusSize = 5000
	RRF_K      = 60
)

type VectorChunk struct {
	ID        string
	Content   string
	Embedding []float32
}

func generateRandomVector() []float32 {
	v := make([]float32, VectorDim)
	var norm float64
	for i := 0; i < VectorDim; i++ {
		val := rand.Float32()*2 - 1
		v[i] = val
		norm += float64(val * val)
	}
	sqrtNorm := float32(math.Sqrt(norm))
	for i := 0; i < VectorDim; i++ {
		v[i] /= sqrtNorm
	}
	return v
}

func cosineSimilarity(a, b []float32) float32 {
	var dot float32
	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
	}
	return dot
}

func setupRetrievalDB(chunks []VectorChunk) (*sql.DB, string, error) {
	tempDir, err := os.MkdirTemp("", "retrieval_bench_*")
	if err != nil {
		return nil, "", err
	}
	dbPath := filepath.Join(tempDir, "retrieval.db")
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return nil, "", err
	}

	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL;")

	schema := `
	CREATE TABLE chunks (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, "", err
	}

	tx, _ := db.Begin()
	stmt, _ := tx.Prepare("INSERT INTO chunks (id, content) VALUES (?, ?)")
	for _, c := range chunks {
		_, _ = stmt.Exec(c.ID, c.Content)
	}
	stmt.Close()
	_ = tx.Commit()

	return db, tempDir, nil
}

// 1. SQLite LIKE %term% scan
func benchmarkSQLLikeSearch(db *sql.DB, queryTerms []string, topK int) time.Duration {
	t0 := time.Now()
	for _, term := range queryTerms {
		pattern := "%" + term + "%"
		rows, err := db.Query("SELECT id FROM chunks WHERE content LIKE ? LIMIT ?", pattern, topK)
		if err != nil {
			continue
		}
		for rows.Next() {
			var id string
			_ = rows.Scan(&id)
		}
		rows.Close()
	}
	return time.Since(t0)
}

// 2. In-Memory Inverted Index (BM25 / Lexical token map)
type InvertedIndex map[string][]string

func buildInvertedIndex(chunks []VectorChunk) InvertedIndex {
	idx := make(InvertedIndex)
	for _, c := range chunks {
		words := strings.Fields(strings.ToLower(c.Content))
		for _, w := range words {
			w = strings.Trim(w, ".,#")
			idx[w] = append(idx[w], c.ID)
		}
	}
	return idx
}

func benchmarkInvertedIndexSearch(idx InvertedIndex, queryTerms []string, topK int) time.Duration {
	t0 := time.Now()
	for _, term := range queryTerms {
		matches := idx[strings.ToLower(term)]
		if len(matches) > topK {
			matches = matches[:topK]
		}
		_ = matches
	}
	return time.Since(t0)
}

// 3. Dense Vector Cosine Scan
func benchmarkVectorScan(chunks []VectorChunk, queryVecs [][]float32, topK int) time.Duration {
	t0 := time.Now()
	for _, qv := range queryVecs {
		type scoredChunk struct {
			id    string
			score float32
		}
		scores := make([]scoredChunk, len(chunks))
		for i, c := range chunks {
			scores[i] = scoredChunk{id: c.ID, score: cosineSimilarity(qv, c.Embedding)}
		}
		sort.Slice(scores, func(i, j int) bool {
			return scores[i].score > scores[j].score
		})
		_ = scores[:topK]
	}
	return time.Since(t0)
}

// 4. Hybrid Reciprocal Rank Fusion (Inverted Index + Vector Cosine + RRF)
func benchmarkHybridRRF(idx InvertedIndex, chunks []VectorChunk, queryTerms []string, queryVecs [][]float32, topK int) time.Duration {
	t0 := time.Now()
	for i, term := range queryTerms {
		qv := queryVecs[i]

		// Step A: Inverted index lexical lookup
		lexMatches := idx[strings.ToLower(term)]
		if len(lexMatches) > 50 {
			lexMatches = lexMatches[:50]
		}

		// Step B: Dense vector score
		type scoredChunk struct {
			id    string
			score float32
		}
		scores := make([]scoredChunk, len(chunks))
		for j, c := range chunks {
			scores[j] = scoredChunk{id: c.ID, score: cosineSimilarity(qv, c.Embedding)}
		}
		sort.Slice(scores, func(a, b int) bool {
			return scores[a].score > scores[b].score
		})

		// Step C: Reciprocal Rank Fusion (RRF)
		rrfScores := make(map[string]float64)
		for rank, m := range lexMatches {
			rrfScores[m] += 1.0 / float64(RRF_K+rank+1)
		}
		for rank, sc := range scores[:50] {
			rrfScores[sc.id] += 1.0 / float64(RRF_K+rank+1)
		}

		type finalRanked struct {
			id    string
			score float64
		}
		finalList := make([]finalRanked, 0, len(rrfScores))
		for id, sc := range rrfScores {
			finalList = append(finalList, finalRanked{id: id, score: sc})
		}
		sort.Slice(finalList, func(a, b int) bool {
			return finalList[a].score > finalList[b].score
		})
		if len(finalList) > topK {
			finalList = finalList[:topK]
		}
	}
	return time.Since(t0)
}

func main() {
	fmt.Println(strings.Repeat("=", 85))
	fmt.Println("RETRIEVAL LATENCY BENCHMARK: LEXICAL vs VECTOR vs HYBRID RRF")
	fmt.Printf("Corpus Size : %d chunks (384-dim dense vectors)\n", CorpusSize)
	fmt.Printf("Query Count : 100 queries evaluated\n")
	fmt.Println(strings.Repeat("=", 85))

	vocabulary := []string{"goroutine", "channels", "memory", "sqlite", "fsrs", "retrieval", "compiler", "garbage"}
	chunks := make([]VectorChunk, CorpusSize)
	for i := 0; i < CorpusSize; i++ {
		term1 := vocabulary[rand.Intn(len(vocabulary))]
		term2 := vocabulary[rand.Intn(len(vocabulary))]
		chunks[i] = VectorChunk{
			ID:        fmt.Sprintf("chunk-%05d", i),
			Content:   fmt.Sprintf("Document text chunk #%d discussing %s concepts alongside %s architecture patterns.", i, term1, term2),
			Embedding: generateRandomVector(),
		}
	}

	db, tempDir, err := setupRetrievalDB(chunks)
	if err != nil {
		fmt.Printf("Error setting up DB: %v\n", err)
		return
	}
	defer func() {
		db.Close()
		os.RemoveAll(tempDir)
	}()

	invIdx := buildInvertedIndex(chunks)

	const numQueries = 100
	queryTerms := make([]string, numQueries)
	queryVecs := make([][]float32, numQueries)
	for i := 0; i < numQueries; i++ {
		queryTerms[i] = vocabulary[rand.Intn(len(vocabulary))]
		queryVecs[i] = generateRandomVector()
	}

	// 1. SQLite LIKE
	durLike := benchmarkSQLLikeSearch(db, queryTerms, 10)
	avgLike := float64(durLike.Microseconds()) / float64(numQueries) / 1000.0

	// 2. Inverted Index
	durInv := benchmarkInvertedIndexSearch(invIdx, queryTerms, 10)
	avgInv := float64(durInv.Microseconds()) / float64(numQueries) / 1000.0

	// 3. Vector Cosine Scan
	durVec := benchmarkVectorScan(chunks, queryVecs, 10)
	avgVec := float64(durVec.Microseconds()) / float64(numQueries) / 1000.0

	// 4. Hybrid RRF
	durHybrid := benchmarkHybridRRF(invIdx, chunks, queryTerms, queryVecs, 10)
	avgHybrid := float64(durHybrid.Microseconds()) / float64(numQueries) / 1000.0

	fmt.Println("\n" + strings.Repeat("=", 85))
	fmt.Println("RETRIEVAL SEARCH LATENCY RESULTS (Top-10 Results across 100 queries)")
	fmt.Println(strings.Repeat("=", 85))
	fmt.Printf("%-38s | %-14s | %-14s | %-12s\n", "Search Strategy", "Total (100 Qs)", "Avg Latency/Q", "Throughput")
	fmt.Println(strings.Repeat("-", 85))
	fmt.Printf("%-38s | %12v | %10.3f ms | %10.1f q/s\n", "SQLite LIKE %term% Table Scan", durLike.Round(time.Millisecond), avgLike, float64(numQueries)/durLike.Seconds())
	fmt.Printf("%-38s | %12v | %10.3f ms | %10.1f q/s\n", "In-Memory Inverted Index (Lexical)", durInv.Round(time.Microsecond), avgInv, float64(numQueries)/durInv.Seconds())
	fmt.Printf("%-38s | %12v | %10.3f ms | %10.1f q/s\n", "Brute-Force Vector Cosine Scan (384d)", durVec.Round(time.Millisecond), avgVec, float64(numQueries)/durVec.Seconds())
	fmt.Printf("%-38s | %12v | %10.3f ms | %10.1f q/s\n", "Hybrid RRF (Lexical + Vector + RRF)", durHybrid.Round(time.Millisecond), avgHybrid, float64(numQueries)/durHybrid.Seconds())
	fmt.Println(strings.Repeat("=", 85))
}
