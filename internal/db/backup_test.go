package db

import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackupDatabase_NonExistentDB(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentDB := filepath.Join(tempDir, "nonexistent.db")

	err := BackupDatabase(nonExistentDB)
	if err != nil {
		t.Fatalf("expected nil error for non-existent db, got: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(tempDir, "nonexistent.db.*.bak*"))
	if err != nil || len(matches) > 0 {
		t.Fatalf("expected 0 backup files, got %d", len(matches))
	}
}

func TestBackupDatabase_RingBufferKeep3(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "Studyloop.db")

	// Create 5 iterations of DB content and trigger backup
	for i := 1; i <= 5; i++ {
		content := fmt.Appendf(nil, "sqlite database revision %d", i)
		if err := os.WriteFile(dbPath, content, 0644); err != nil {
			t.Fatalf("failed to write db revision %d: %v", i, err)
		}
		if err := BackupDatabase(dbPath); err != nil {
			t.Fatalf("BackupDatabase iteration %d failed: %v", i, err)
		}
		// Brief sleep to ensure distinct millisecond timestamps
		time.Sleep(5 * time.Millisecond)
	}

	matches, err := filepath.Glob(filepath.Join(tempDir, "Studyloop.db.*.bak*"))
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}

	if len(matches) != 3 {
		t.Fatalf("expected exactly 3 backups preserved, found %d: %v", len(matches), matches)
	}
}

func TestBackupDatabase_ValidSQLiteVacuumAndGzip(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "Studyloop.db")

	// Create a real SQLite database with data
	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open test sqlite db: %v", err)
	}
	_, err = dbConn.Exec("CREATE TABLE test_notes (id INTEGER PRIMARY KEY, title TEXT); INSERT INTO test_notes (title) VALUES ('Physics Chapter 1');")
	_ = dbConn.Close()
	if err != nil {
		t.Fatalf("failed to populate test sqlite db: %v", err)
	}

	if err := BackupDatabase(dbPath); err != nil {
		t.Fatalf("BackupDatabase failed on valid SQLite DB: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(tempDir, "Studyloop.db.*.bak.gz"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("expected 1 .bak.gz file, found %d (err: %v)", len(matches), err)
	}

	backupGzPath := matches[0]
	gzFile, err := os.Open(backupGzPath)
	if err != nil {
		t.Fatalf("failed to open generated .bak.gz file: %v", err)
	}
	defer func() { _ = gzFile.Close() }()

	gzReader, err := gzip.NewReader(gzFile)
	if err != nil {
		t.Fatalf("failed to initialize gzip reader: %v", err)
	}
	defer func() { _ = gzReader.Close() }()

	restoredDBPath := filepath.Join(tempDir, "Restored.db")
	restoredFile, err := os.Create(restoredDBPath)
	if err != nil {
		t.Fatalf("failed to create restored db target file: %v", err)
	}
	if _, err := io.Copy(restoredFile, gzReader); err != nil {
		_ = restoredFile.Close()
		t.Fatalf("failed to decompress .bak.gz content: %v", err)
	}
	_ = restoredFile.Close()

	// Verify restored SQLite DB is valid and queryable
	restoredConn, err := sql.Open("sqlite3", restoredDBPath)
	if err != nil {
		t.Fatalf("failed to open restored sqlite db: %v", err)
	}
	defer func() { _ = restoredConn.Close() }()

	var title string
	if err := restoredConn.QueryRow("SELECT title FROM test_notes WHERE id = 1").Scan(&title); err != nil {
		t.Fatalf("failed to query restored db: %v", err)
	}
	if title != "Physics Chapter 1" {
		t.Fatalf("expected 'Physics Chapter 1', got '%s'", title)
	}
}

func TestRestoreLatestBackup(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "Studyloop.db")

	// Create valid db and backup
	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open test sqlite db: %v", err)
	}
	_, _ = dbConn.Exec("CREATE TABLE test_data (val TEXT); INSERT INTO test_data VALUES ('healthy_data');")
	_ = dbConn.Close()

	if err := BackupDatabase(dbPath); err != nil {
		t.Fatalf("BackupDatabase failed: %v", err)
	}

	// Corrupt dbPath
	if err := os.WriteFile(dbPath, []byte("CORRUPTED DATA STREAM"), 0644); err != nil {
		t.Fatalf("failed to corrupt db: %v", err)
	}

	// Trigger RestoreLatestBackup
	if err := RestoreLatestBackup(dbPath); err != nil {
		t.Fatalf("RestoreLatestBackup failed: %v", err)
	}

	// Verify restored db content
	restoredConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open restored db: %v", err)
	}
	defer func() { _ = restoredConn.Close() }()

	var val string
	if err := restoredConn.QueryRow("SELECT val FROM test_data").Scan(&val); err != nil || val != "healthy_data" {
		t.Fatalf("expected 'healthy_data', got val='%s', err=%v", val, err)
	}
}

