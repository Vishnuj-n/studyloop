package db

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"ai-tutor/internal/utils"
)

const maxBackups = 3

// BackupDatabase creates a timestamped backup of Studyloop.db and keeps the last 3 copies.
func BackupDatabase(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// First run before DB creation: nothing to back up yet
		return nil
	}

	destDir := filepath.Dir(dbPath)
	baseName := filepath.Base(dbPath)
	// ponytail: simple timestamped name + lexical prune keeps the last 3 without extra metadata
	timestamp := time.Now().Format("20060102-150405.000")
	backupPath := filepath.Join(destDir, fmt.Sprintf("%s.%s.bak", baseName, timestamp))

	srcFile, err := os.Open(dbPath)
	if err != nil {
		return fmt.Errorf("backup: open source db: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	tmpFile, err := os.CreateTemp(destDir, "studyloop-backup-*.tmp")
	if err != nil {
		return fmt.Errorf("backup: create temp backup file: %w", err)
	}
	tmpName := tmpFile.Name()
	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := io.Copy(tmpFile, srcFile); err != nil {
		return fmt.Errorf("backup: copy content: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("backup: sync temp backup file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("backup: close temp backup file: %w", err)
	}
	if err := os.Rename(tmpName, backupPath); err != nil {
		return fmt.Errorf("backup: atomic place backup file: %w", err)
	}

	success = true
	utils.Warnf("[BACKUP] Successfully created database backup: %s", backupPath)

	pruneOldBackups(destDir, baseName, maxBackups)
	return nil
}

func pruneOldBackups(dir, baseName string, keep int) {
	pattern := filepath.Join(dir, baseName+".*.bak")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) <= keep {
		return
	}

	sort.Strings(matches)
	for _, old := range matches[:len(matches)-keep] {
		_ = os.Remove(old)
	}
}
