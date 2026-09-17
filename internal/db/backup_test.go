package db

import (
	"fmt"
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

	matches, err := filepath.Glob(filepath.Join(tempDir, "nonexistent.db.*.bak"))
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

	matches, err := filepath.Glob(filepath.Join(tempDir, "Studyloop.db.*.bak"))
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}

	if len(matches) != 3 {
		t.Fatalf("expected exactly 3 backups preserved, found %d: %v", len(matches), matches)
	}
}

