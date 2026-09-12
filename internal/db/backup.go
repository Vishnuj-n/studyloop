package db

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"ai-tutor/internal/utils"
)

// BackupDatabase creates or overwrites Studyloop.db.bak from the current Studyloop.db file.
func BackupDatabase(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// First run before DB creation: nothing to back up yet
		return nil
	}

	backupPath := dbPath + ".bak"
	destDir := filepath.Dir(backupPath)

	srcFile, err := os.Open(dbPath)
	if err != nil {
		return fmt.Errorf("backup: open source db: %w", err)
	}
	defer srcFile.Close()

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
		return fmt.Errorf("backup: atomic replace backup file: %w", err)
	}

	success = true
	utils.Warnf("[BACKUP] Successfully created database backup: %s", backupPath)
	return nil
}
