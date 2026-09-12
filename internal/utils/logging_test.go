package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRotateLogFile(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")
	oldPath := logPath + ".old"

	// Case 1: File does not exist -> should not rotate or fail
	if err := rotateLogFile(logPath, 100); err != nil {
		t.Fatalf("unexpected error on non-existent log file: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected .old file to not exist when log does not exist")
	}

	// Case 2: File is smaller than maxSize -> should not rotate
	if err := os.WriteFile(logPath, []byte("short content"), 0644); err != nil {
		t.Fatalf("failed to write test log file: %v", err)
	}
	if err := rotateLogFile(logPath, 100); err != nil {
		t.Fatalf("unexpected error on small log file: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected .old file to not exist when log size < maxSize")
	}

	// Case 3: File exceeds maxSize -> should rotate to .old
	largeData := make([]byte, 150)
	if err := os.WriteFile(logPath, largeData, 0644); err != nil {
		t.Fatalf("failed to write large log file: %v", err)
	}
	if err := rotateLogFile(logPath, 100); err != nil {
		t.Fatalf("unexpected error on large log file: %v", err)
	}

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("expected original log file to be moved away after rotation")
	}
	info, err := os.Stat(oldPath)
	if err != nil {
		t.Fatalf("expected .old file to exist after rotation: %v", err)
	}
	if info.Size() != 150 {
		t.Errorf("expected .old file size 150, got %d", info.Size())
	}
}

func TestInitMultiFileLoggerRotation(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create logs dir: %v", err)
	}

	// Create a dummy queue.log that exceeds maxLogSizeBytes
	queueLogPath := filepath.Join(logDir, "queue.log")
	largeData := make([]byte, maxLogSizeBytes+1024)
	if err := os.WriteFile(queueLogPath, largeData, 0644); err != nil {
		t.Fatalf("failed to write dummy log: %v", err)
	}

	// Initialize multi file logger
	if err := InitMultiFileLogger(tempDir); err != nil {
		t.Fatalf("InitMultiFileLogger failed: %v", err)
	}
	defer CloseMultiFileLogger()

	// Verify queue.log.old exists with largeData size
	oldPath := queueLogPath + ".old"
	info, err := os.Stat(oldPath)
	if err != nil {
		t.Fatalf("expected queue.log.old to exist after InitMultiFileLogger: %v", err)
	}
	if info.Size() != int64(len(largeData)) {
		t.Errorf("expected rotated file size %d, got %d", len(largeData), info.Size())
	}

	// Verify fresh queue.log was created and opened
	if _, err := os.Stat(queueLogPath); err != nil {
		t.Errorf("expected fresh queue.log to exist: %v", err)
	}
}
