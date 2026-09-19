package db

import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"ai-tutor/internal/utils"
)

const maxBackups = 3

// BackupDatabase creates a compressed timestamped backup of Studyloop.db (using VACUUM INTO when available)
// and retains the last 3 copies.
func BackupDatabase(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// First run before DB creation: nothing to back up yet
		return nil
	}

	destDir := filepath.Dir(dbPath)
	baseName := filepath.Base(dbPath)
	timestamp := time.Now().Format("20060102-150405.000")
	backupPath := filepath.Join(destDir, fmt.Sprintf("%s.%s.bak.gz", baseName, timestamp))

	tmpGzFile, err := os.CreateTemp(destDir, "studyloop-backup-*.tmp.gz")
	if err != nil {
		return fmt.Errorf("backup: create temp gzip file: %w", err)
	}
	tmpGzName := tmpGzFile.Name()
	success := false
	defer func() {
		if !success {
			_ = tmpGzFile.Close()
			_ = os.Remove(tmpGzName)
		}
	}()

	// 1. Attempt atomic online snapshot using SQLite's VACUUM INTO
	vacuumTmp, err := createVacuumSnapshot(dbPath, destDir)
	var srcReader io.ReadCloser

	if err == nil {
		defer func() { _ = os.Remove(vacuumTmp) }()
		f, openErr := os.Open(vacuumTmp)
		if openErr == nil {
			srcReader = f
		}
	}

	// Fallback to raw dbPath reading if VACUUM INTO failed or could not open snapshot
	if srcReader == nil {
		rawFile, openErr := os.Open(dbPath)
		if openErr != nil {
			return fmt.Errorf("backup: open source db: %w", openErr)
		}
		srcReader = rawFile
	}
	defer func() { _ = srcReader.Close() }()

	// 2. Compress snapshot into final gzip output
	gzWriter := gzip.NewWriter(tmpGzFile)
	if _, err := io.Copy(gzWriter, srcReader); err != nil {
		_ = gzWriter.Close()
		return fmt.Errorf("backup: compress content: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return fmt.Errorf("backup: finish gzip stream: %w", err)
	}
	if err := tmpGzFile.Sync(); err != nil {
		return fmt.Errorf("backup: sync temp backup file: %w", err)
	}
	if err := tmpGzFile.Close(); err != nil {
		return fmt.Errorf("backup: close temp backup file: %w", err)
	}

	// 3. Atomic rename into final backupPath (.bak.gz)
	if err := os.Rename(tmpGzName, backupPath); err != nil {
		return fmt.Errorf("backup: atomic place backup file: %w", err)
	}

	success = true
	utils.Warnf("[BACKUP] Successfully created database backup: %s", backupPath)

	pruneOldBackups(destDir, baseName, maxBackups)
	return nil
}

func createVacuumSnapshot(dbPath, destDir string) (string, error) {
	vacFile, err := os.CreateTemp(destDir, "studyloop-vacuum-*.tmp")
	if err != nil {
		return "", err
	}
	vacPath := vacFile.Name()
	_ = vacFile.Close()
	// VACUUM INTO expects the destination file to NOT exist beforehand
	_ = os.Remove(vacPath)

	dbConn, err := sql.Open("sqlite3", "file:"+dbPath+"?_busy_timeout=5000")
	if err != nil {
		return "", err
	}
	defer func() { _ = dbConn.Close() }()

	if _, err := dbConn.Exec("VACUUM INTO ?", vacPath); err != nil {
		_ = os.Remove(vacPath)
		return "", err
	}

	return vacPath, nil
}

func pruneOldBackups(dir, baseName string, keep int) {
	patternBak := filepath.Join(dir, baseName+".*.bak")
	patternGz := filepath.Join(dir, baseName+".*.bak.gz")

	matchesBak, _ := filepath.Glob(patternBak)
	matchesGz, _ := filepath.Glob(patternGz)

	var matches []string
	matches = append(matches, matchesBak...)
	matches = append(matches, matchesGz...)

	if len(matches) <= keep {
		return
	}

	sort.Strings(matches)
	for _, old := range matches[:len(matches)-keep] {
		_ = os.Remove(old)
	}
}

// RestoreLatestBackup finds the newest .bak or .bak.gz file for dbPath and restores it.
func RestoreLatestBackup(dbPath string) error {
	dir := filepath.Dir(dbPath)
	baseName := filepath.Base(dbPath)
	patternBak := filepath.Join(dir, baseName+".*.bak")
	patternGz := filepath.Join(dir, baseName+".*.bak.gz")

	matchesBak, _ := filepath.Glob(patternBak)
	matchesGz, _ := filepath.Glob(patternGz)
	var matches []string
	matches = append(matches, matchesBak...)
	matches = append(matches, matchesGz...)

	if len(matches) == 0 {
		return fmt.Errorf("restore: no backup files found in %s", dir)
	}
	sort.Strings(matches)
	latest := matches[len(matches)-1]

	if _, err := os.Stat(dbPath); err == nil {
		_ = os.Rename(dbPath, fmt.Sprintf("%s.corrupted.%s", dbPath, time.Now().Format("20060102-150405")))
	}
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")

	src, err := os.Open(latest)
	if err != nil {
		return fmt.Errorf("restore: open backup: %w", err)
	}
	defer func() { _ = src.Close() }()

	var r io.Reader = src
	if filepath.Ext(latest) == ".gz" {
		gz, err := gzip.NewReader(src)
		if err != nil {
			return fmt.Errorf("restore: open gzip: %w", err)
		}
		defer func() { _ = gz.Close() }()
		r = gz
	}

	dst, err := os.Create(dbPath)
	if err != nil {
		return fmt.Errorf("restore: create destination: %w", err)
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, r); err != nil {
		return fmt.Errorf("restore: copy content: %w", err)
	}

	utils.Warnf("[RESTORE] Successfully restored database from backup: %s", latest)
	return nil
}

