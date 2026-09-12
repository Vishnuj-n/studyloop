package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupDatabase_NonExistentDB(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentDB := filepath.Join(tempDir, "nonexistent.db")

	err := BackupDatabase(nonExistentDB)
	if err != nil {
		t.Fatalf("expected nil error for non-existent db, got: %v", err)
	}

	bakPath := nonExistentDB + ".bak"
	if _, statErr := os.Stat(bakPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected backup file to not exist, but it was found")
	}
}

func TestBackupDatabase_SuccessAndOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "Studyloop.db")
	bakPath := dbPath + ".bak"

	// 1. Create initial DB file
	initialContent := []byte("sqlite database initial content header and data")
	if err := os.WriteFile(dbPath, initialContent, 0644); err != nil {
		t.Fatalf("failed to write test db: %v", err)
	}

	// 2. Perform backup
	if err := BackupDatabase(dbPath); err != nil {
		t.Fatalf("BackupDatabase failed: %v", err)
	}

	// 3. Verify backup content matches
	bakContent, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("failed to read backup file: %v", err)
	}
	if string(bakContent) != string(initialContent) {
		t.Fatalf("expected backup content %q, got %q", string(initialContent), string(bakContent))
	}

	// 4. Update DB and perform backup again (overwrite test)
	updatedContent := []byte("sqlite database updated content header and data after edits")
	if err := os.WriteFile(dbPath, updatedContent, 0644); err != nil {
		t.Fatalf("failed to update test db: %v", err)
	}

	if err := BackupDatabase(dbPath); err != nil {
		t.Fatalf("BackupDatabase overwrite failed: %v", err)
	}

	newBakContent, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("failed to read overwritten backup file: %v", err)
	}
	if string(newBakContent) != string(updatedContent) {
		t.Fatalf("expected overwritten content %q, got %q", string(updatedContent), string(newBakContent))
	}
}
