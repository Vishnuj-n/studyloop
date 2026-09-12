package utils

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var (
	queueLogWriter *rotatingWriter
	ragLogWriter   *rotatingWriter
	appLogWriter   *rotatingWriter

	// QueueLogger writes structured queue lifecycle events to queue.log.
	QueueLogger *slog.Logger
	// RagLogger writes structured RAG/boot events to rag_engine.log.
	RagLogger *slog.Logger

	logMutex sync.Mutex
)

func init() {
	// Default loggers write to io.Discard until InitMultiFileLogger is called to avoid leaking console lines on Windows.
	QueueLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	RagLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

const maxLogSizeBytes int64 = 5 * 1024 * 1024 // 5 MB per log file

func rotateLogFile(logPath string, maxSize int64) error {
	info, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat log file %s: %w", logPath, err)
	}
	if info.Size() < maxSize {
		return nil
	}
	oldPath := logPath + ".old"
	if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove old log file %s: %w", oldPath, err)
	}
	if err := os.Rename(logPath, oldPath); err != nil {
		return fmt.Errorf("rename log file %s to %s: %w", logPath, oldPath, err)
	}
	return nil
}

type rotatingWriter struct {
	mu       sync.Mutex
	logPath  string
	maxSize  int64
	file     *os.File
	currSize int64
}

func newRotatingWriter(logPath string, maxSize int64) (*rotatingWriter, error) {
	if err := rotateLogFile(logPath, maxSize); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file %s: %w", logPath, err)
	}
	var currSize int64
	if info, err := f.Stat(); err == nil {
		currSize = info.Size()
	}
	return &rotatingWriter{
		logPath:  logPath,
		maxSize:  maxSize,
		file:     f,
		currSize: currSize,
	}, nil
}

func (w *rotatingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil && w.currSize+int64(len(p)) > w.maxSize {
		_ = w.file.Sync()
		_ = w.file.Close()
		_ = rotateLogFile(w.logPath, w.maxSize)
		f, err := os.OpenFile(w.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			w.file = f
			w.currSize = 0
		}
	}
	if w.file == nil {
		return 0, fmt.Errorf("log file closed")
	}
	n, err = w.file.Write(p)
	w.currSize += int64(n)
	return n, err
}

func (w *rotatingWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Sync()
	}
	return nil
}

func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		_ = w.file.Sync()
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}

// InitMultiFileLogger creates the logs subdirectory under appDataDir and
// redirects QueueLogger, RagLogger, and the default slog logger to their
// respective files.
func InitMultiFileLogger(appDataDir string) error {
	logMutex.Lock()
	defer logMutex.Unlock()

	logDir := filepath.Join(appDataDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Close existing files if any are open (for safety in multi-call or testing environments).
	if queueLogWriter != nil {
		_ = queueLogWriter.Close()
		queueLogWriter = nil
	}
	if ragLogWriter != nil {
		_ = ragLogWriter.Close()
		ragLogWriter = nil
	}
	if appLogWriter != nil {
		_ = appLogWriter.Close()
		appLogWriter = nil
	}

	// Install io.Discard fallback loggers immediately after closing existing file handles.
	QueueLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	RagLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, nil)))

	queuePath := filepath.Join(logDir, "queue.log")
	ragPath := filepath.Join(logDir, "rag_engine.log")
	appPath := filepath.Join(logDir, "app.log")

	var err error
	queueLogWriter, err = newRotatingWriter(queuePath, maxLogSizeBytes)
	if err != nil {
		return fmt.Errorf("failed to open queue log file: %w", err)
	}

	ragLogWriter, err = newRotatingWriter(ragPath, maxLogSizeBytes)
	if err != nil {
		_ = queueLogWriter.Close()
		queueLogWriter = nil
		return fmt.Errorf("failed to open rag engine log file: %w", err)
	}

	appLogWriter, err = newRotatingWriter(appPath, maxLogSizeBytes)
	if err != nil {
		_ = queueLogWriter.Close()
		_ = ragLogWriter.Close()
		queueLogWriter = nil
		ragLogWriter = nil
		return fmt.Errorf("failed to open app log file: %w", err)
	}

	QueueLogger = slog.New(slog.NewJSONHandler(queueLogWriter, nil))
	RagLogger = slog.New(slog.NewJSONHandler(ragLogWriter, nil))
	slog.SetDefault(slog.New(slog.NewJSONHandler(appLogWriter, nil)))

	return nil
}

// CloseMultiFileLogger flushes and closes all active file handles.
func CloseMultiFileLogger() {
	logMutex.Lock()
	defer logMutex.Unlock()

	if queueLogWriter != nil {
		_ = queueLogWriter.Close()
		queueLogWriter = nil
	}
	if ragLogWriter != nil {
		_ = ragLogWriter.Close()
		ragLogWriter = nil
	}
	if appLogWriter != nil {
		_ = appLogWriter.Close()
		appLogWriter = nil
	}

	// Revert to io.Discard fallback loggers on close to avoid leaking console streams or nil dereference.
	QueueLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	RagLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

// ---------- Global Level Helpers ----------

func Debugf(format string, args ...any) {
	slog.Debug(fmt.Sprintf(format, args...))
}

func Infof(format string, args ...any) {
	slog.Info(fmt.Sprintf(format, args...))
}

func Warnf(format string, args ...any) {
	slog.Warn(fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...any) {
	slog.Error(fmt.Sprintf(format, args...))
}

// ---------- Queue Lifecycle Logging ----------

// LogQueueTransition logs structured queue transition events.
func LogQueueTransition(taskID, taskType, oldStatus, newStatus, reason string) {
	if taskID == "" {
		taskID = "unknown"
	}
	if taskType == "" {
		taskType = "unknown"
	}
	if oldStatus == "" {
		oldStatus = "none"
	}
	if newStatus == "" {
		newStatus = "unknown"
	}
	if reason == "" {
		reason = "transition"
	}
	QueueLogger.Info("queue_transition",
		"task", taskID, "type", taskType,
		"from", oldStatus, "to", newStatus, "reason", reason)
}

// LogQueueTaskCreated logs when a new task is inserted into the queue.
func LogQueueTaskCreated(taskID, taskType, notebookID, topicID string) {
	if taskID == "" {
		taskID = "unknown"
	}
	if taskType == "" {
		taskType = "unknown"
	}
	QueueLogger.Info("task_inserted",
		"task", taskID, "type", taskType,
		"notebook", notebookID, "topic", topicID)
}

// LogQuizResult logs quiz completion with pass/fail outcome.
func LogQuizResult(taskID string, score int, passed bool, rereadTaskID string) {
	if taskID == "" {
		taskID = "unknown"
	}
	outcome := "failed"
	if passed {
		outcome = "passed"
	}
	QueueLogger.Info("quiz_completed",
		"task", taskID, "type", "QUIZ",
		"score", score, "outcome", outcome, "reread", rereadTaskID)
}

// LogRereadInsertion logs when a reread task is generated after quiz failure.
func LogRereadInsertion(taskID, topicID, attemptCount, maxAttempts string) {
	if taskID == "" {
		taskID = "unknown"
	}
	QueueLogger.Info("reread_inserted",
		"task", taskID, "type", "REREAD",
		"topic", topicID, "attempt", attemptCount, "max", maxAttempts)
}

// LogReviewSession logs review session lifecycle events.
func LogReviewSession(taskID, notebookID, cardCount, event string) {
	if taskID == "" {
		taskID = "unknown"
	}
	QueueLogger.Info(event,
		"task", taskID, "type", "FLASHCARD_REVIEW",
		"notebook", notebookID, "cards", cardCount)
}

// LogSchedulerDecision logs adaptive reading window decisions.
func LogSchedulerDecision(topicID string, startPage, endPage int, tokenBudget, reason string) {
	if topicID == "" {
		topicID = "unknown"
	}
	QueueLogger.Info("scheduler_decision",
		"topic", topicID, "startPage", startPage, "endPage", endPage,
		"tokenBudget", tokenBudget, "reason", reason)
}

// LogReadingEstimate logs word count and duration estimates for reading tasks.
func LogReadingEstimate(taskID, topicID string, startPage, endPage, wordCount, estimateMinutes int, source string) {
	if taskID == "" {
		taskID = "unknown"
	}
	QueueLogger.Info("reading_estimate",
		"task", taskID, "topic", topicID,
		"pages", fmt.Sprintf("%d-%d", startPage, endPage),
		"word_count", wordCount, "estimate_minutes", estimateMinutes, "source", source)
}


