package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"ai-tutor/internal/utils"
)

// querier interface allows both *sql.DB and *sql.Tx to be used with database helper functions
type querier interface {
	QueryRow(query string, args ...any) *sql.Row
}

type Repository struct {
	db                 *sql.DB
	embeddingDimension int32
}

const maxRetrievalK = 100 // Maximum k allowed for vector search retrieval

const (
	llmTierFast  = "fast"
	llmTierHeavy = "heavy"
)

// Close releases the active SQLite connection.
func (r *Repository) Close() error {
	if r.db == nil {
		return nil
	}
	err := r.db.Close()
	r.db = nil
	return err
}

// SwapDB swaps the underlying database connection in-place and returns the old connection.
func (r *Repository) SwapDB(newRepo *Repository) *sql.DB {
	oldDB := r.db
	r.db = newRepo.db
	r.embeddingDimension = newRepo.embeddingDimension
	return oldDB
}

// Begin starts a new transaction on the database.
func (r *Repository) Begin() (*sql.Tx, error) {
	return r.db.Begin()
}

// ExecForTest executes a query directly on the underlying database. ONLY for test usage.
func (r *Repository) ExecForTest(query string, args ...interface{}) (sql.Result, error) {
	return r.db.Exec(query, args...)
}

// QueryRowForTest runs a QueryRow directly on the underlying database. ONLY for test usage.
func (r *Repository) QueryRowForTest(query string, args ...interface{}) *sql.Row {
	return r.db.QueryRow(query, args...)
}

// Init initializes the SQLite database and creates tables
// vec0DllPath should be the absolute path to vec0.dll (sqlite-vec extension)
func Init(dbPath, vec0DllPath string) (*Repository, error) {
	utils.RagLogger.Info("db.Init: initializing database pool", "dbPath", dbPath, "vec0DllPath", vec0DllPath)
	driverName := "sqlite3"
	if vec0DllPath != "" {
		if _, err := os.Stat(vec0DllPath); err == nil {
			absPath, err := filepath.Abs(vec0DllPath)
			if err == nil {
				utils.RagLogger.Info("db.Init: vec0 file verified, preparing sqlite3_tutor driver", "absPath", absPath)
				setExtensionPath(absPath)
				driverName = "sqlite3_tutor"
			} else {
				utils.RagLogger.Warn("db.Init: failed to resolve absolute path for vec0 file", "path", vec0DllPath, "error", err)
			}
		} else {
			utils.RagLogger.Warn("db.Init: vec0 file not found at path", "path", vec0DllPath, "error", err)
		}
	}

	utils.RagLogger.Info("db.Init: opening SQL connection pool", "driverName", driverName)
	dbConn, err := sql.Open(driverName, "file:"+dbPath+"?_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		utils.RagLogger.Error("db.Init: failed to open SQL connection pool", "driverName", driverName, "error", err)
		return nil, err
	}
	dbConn.SetMaxOpenConns(1)
	dbConn.SetMaxIdleConns(1)

	utils.RagLogger.Info("db.Init: pinging database connection")
	if err := dbConn.Ping(); err != nil {
		utils.RagLogger.Error("db.Init: database connection ping failed", "error", err)
		if closeErr := dbConn.Close(); closeErr != nil {
			utils.RagLogger.Warn("db.Init: failed to close database connection after ping error", "error", closeErr)
		}
		return nil, err
	}
	utils.RagLogger.Info("db.Init: database connection ping succeeded")

	// Verify extension load if custom driver was used
	if driverName == "sqlite3_tutor" {
		var version string
		utils.RagLogger.Info("db.Init: verifying extension load by running SELECT vec_version()")
		if err := dbConn.QueryRow("SELECT vec_version()").Scan(&version); err != nil {
			utils.RagLogger.Warn("db.Init: vector verification failed, falling back to standard sqlite3", "error", err, "path", vec0DllPath)
			setExtensionPath("")
			_ = dbConn.Close()

			// Fallback to standard sqlite3
			var fbErr error
			utils.RagLogger.Info("db.Init: opening fallback standard sqlite3 connection pool")
			dbConn, fbErr = sql.Open("sqlite3", "file:"+dbPath+"?_foreign_keys=on&_busy_timeout=5000")
			if fbErr != nil {
				utils.RagLogger.Error("db.Init: standard sqlite3 fallback connection pool failed to open", "error", fbErr)
				return nil, fbErr
			}
			dbConn.SetMaxOpenConns(1)
			dbConn.SetMaxIdleConns(1)
		} else {
			utils.RagLogger.Info("db.Init: successfully verified sqlite-vec extension", "path", vec0DllPath, "version", version)
		}
	}

	// Nuclear strategy: Initialize schema with a single transaction
	tx, err := dbConn.Begin()
	if err != nil {
		if closeErr := dbConn.Close(); closeErr != nil {
			log.Printf("Warning: failed to close database connection after begin error: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to begin schema transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := InitSchema(tx); err != nil {
		if closeErr := dbConn.Close(); closeErr != nil {
			log.Printf("Warning: failed to close database connection after schema error: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	if err := tx.Commit(); err != nil {
		if closeErr := dbConn.Close(); closeErr != nil {
			log.Printf("Warning: failed to close database connection after commit error: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to commit schema transaction: %w", err)
	}

	return &Repository{db: dbConn}, nil
}

// IsVecExtensionLoaded checks if the sqlite-vec (vec0) extension is loaded and functional.
func (r *Repository) IsVecExtensionLoaded() bool {
	if r.db == nil {
		return false
	}
	var version string
	err := r.db.QueryRow("SELECT vec_version()").Scan(&version)
	return err == nil
}

// InitWithVectorDimension initializes the database and creates the vec0 virtual table.
// Called after ONNX embedder dimension is discovered.
func (r *Repository) InitWithVectorDimension(embeddingDim int32) error {
	if embeddingDim <= 0 {
		return fmt.Errorf("invalid embedding dimension: %d", embeddingDim)
	}
	r.embeddingDimension = embeddingDim

	// Create vec0 virtual table with the discovered dimension
	return r.createVectorTable()
}
