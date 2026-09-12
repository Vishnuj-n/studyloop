package db

import (
	"fmt"
	"io"
	"os"

	"ai-tutor/internal/utils"
)

// BackupDatabase creates or overwrites Studyloop.db.bak from the current Studyloop.db file.
func BackupDatabase(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// First run before DB creation: nothing to back up yet
		return nil
	}

	backupPath := dbPath + ".bak"
	srcFile, err := os.Open(dbPath)
	if err != nil {
		return fmt.Errorf("backup: open source db: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("backup: create destination bak file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("backup: copy content: %w", err)
	}

	utils.Warnf("[BACKUP] Successfully created database backup: %s", backupPath)
	return nil
}
