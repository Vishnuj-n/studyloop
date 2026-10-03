package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"ai-tutor/internal/db"
)

func TestResolveAppDir_Dev(t *testing.T) {
	origEnv := os.Getenv("APP_ENV")
	defer func() {
		_ = os.Setenv("APP_ENV", origEnv)
	}()

	_ = os.Setenv("APP_ENV", "dev")
	dir, err := ResolveAppDir()
	if err != nil {
		t.Fatalf("ResolveAppDir() failed in dev mode: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() failed: %v", err)
	}

	expected := filepath.Join(wd, "dev_data")
	if dir != expected {
		t.Errorf("ResolveAppDir() = %q, expected %q", dir, expected)
	}
}

func TestResolveAppDir_Prod(t *testing.T) {
	origEnv := os.Getenv("APP_ENV")
	defer func() {
		_ = os.Setenv("APP_ENV", origEnv)
	}()

	_ = os.Setenv("APP_ENV", "production")
	dir, err := ResolveAppDir()
	if err != nil {
		t.Fatalf("ResolveAppDir() failed in production mode: %v", err)
	}
	if dir == "" {
		t.Fatal("ResolveAppDir() returned empty directory in production mode")
	}
}

func TestResolveAppDir_UnsetDefaultsToProd(t *testing.T) {
	origEnv := os.Getenv("APP_ENV")
	defer func() {
		_ = os.Setenv("APP_ENV", origEnv)
	}()

	_ = os.Unsetenv("APP_ENV")
	dir, err := ResolveAppDir()
	if err != nil {
		t.Fatalf("ResolveAppDir() failed when APP_ENV unset: %v", err)
	}
	if dir == "" {
		t.Fatal("ResolveAppDir() returned empty directory")
	}
	if os.Getenv("APP_ENV") != "production" {
		t.Errorf("expected APP_ENV to be set to production, got %q", os.Getenv("APP_ENV"))
	}
}

func TestResolvePaths(t *testing.T) {
	dbPath, err := ResolveDBPath()
	if err != nil {
		t.Fatalf("ResolveDBPath() failed: %v", err)
	}
	if filepath.Base(dbPath) != "Studyloop.db" {
		t.Errorf("ResolveDBPath base = %q, expected Studyloop.db", filepath.Base(dbPath))
	}

	sessionPath, err := ResolveSessionPath()
	if err != nil {
		t.Fatalf("ResolveSessionPath() failed: %v", err)
	}
	if filepath.Base(sessionPath) != "session.json" {
		t.Errorf("ResolveSessionPath base = %q, expected session.json", filepath.Base(sessionPath))
	}

	uploadDir, err := ResolveNotebookDir()
	if err != nil {
		t.Fatalf("ResolveNotebookDir() failed: %v", err)
	}
	if filepath.Base(uploadDir) != "uploads" {
		t.Errorf("ResolveNotebookDir base = %q, expected uploads", filepath.Base(uploadDir))
	}
}

func TestBootResult_ThreadSafetyAndState(t *testing.T) {
	res := &BootResult{}

	// Initial state
	if res.IsAIReady() {
		t.Error("expected IsAIReady to be false initially")
	}
	if res.GetEmbedder() != nil {
		t.Error("expected GetEmbedder to be nil initially")
	}
	if res.GetAIInitError() != nil {
		t.Error("expected GetAIInitError to be nil initially")
	}
	if res.GetAIInitErrorString() != "" {
		t.Errorf("expected empty error string, got %q", res.GetAIInitErrorString())
	}

	// Update AI state with an error
	testErr := errors.New("embedder failure")
	res.SetAIState(false, nil, testErr)

	if res.IsAIReady() {
		t.Error("expected IsAIReady to be false after error")
	}
	if !errors.Is(res.GetAIInitError(), testErr) {
		t.Errorf("expected GetAIInitError to return testErr, got %v", res.GetAIInitError())
	}
	if res.GetAIInitErrorString() != "embedder failure" {
		t.Errorf("expected GetAIInitErrorString 'embedder failure', got %q", res.GetAIInitErrorString())
	}

	// Concurrent read/write test
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				res.SetAIState(true, nil, nil)
			} else {
				res.SetAIState(false, nil, errors.New("concurrent error"))
			}
		}(i)

		go func() {
			defer wg.Done()
			_ = res.IsAIReady()
			_ = res.GetAIInitError()
			_ = res.GetAIInitErrorString()
			_ = res.GetRepo()
		}()
	}
	wg.Wait()
}

func TestBootResult_Close(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	repo, err := db.Init(dbPath, "")
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}

	cancelled := false
	cancelFunc := func() {
		cancelled = true
	}

	res := &BootResult{
		Repo:         repo,
		backupCancel: cancelFunc,
	}

	if err := res.Close(); err != nil {
		t.Fatalf("BootResult.Close() failed: %v", err)
	}

	if !cancelled {
		t.Error("expected backupCancel to be called during Close()")
	}
	if res.Repo != nil {
		t.Error("expected res.Repo to be nil after Close()")
	}
}

func TestBootstrap_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := Bootstrap(ctx)
	if err == nil {
		t.Fatal("expected Bootstrap to fail with cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestBootstrap_Success(t *testing.T) {
	tempDir := t.TempDir()
	origEnv := os.Getenv("APP_ENV")
	defer func() {
		_ = os.Setenv("APP_ENV", origEnv)
	}()

	_ = os.Setenv("APP_ENV", "dev")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := Bootstrap(ctx)
	if err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}
	defer func() {
		_ = res.Close()
	}()

	if res.GetRepo() == nil {
		t.Error("expected res.GetRepo() to be non-nil")
	}
	if res.RetrievalEngine == nil {
		t.Error("expected res.RetrievalEngine to be non-nil")
	}
	if res.FastLLMProvider == nil {
		t.Error("expected res.FastLLMProvider to be non-nil")
	}
	if res.HeavyLLMProvider == nil {
		t.Error("expected res.HeavyLLMProvider to be non-nil")
	}
	if res.Scheduler == nil {
		t.Error("expected res.Scheduler to be non-nil")
	}
	if res.NotebookService == nil {
		t.Error("expected res.NotebookService to be non-nil")
	}
	if res.StudyService == nil {
		t.Error("expected res.StudyService to be non-nil")
	}
	if res.NotebookUploadDir == "" {
		t.Error("expected res.NotebookUploadDir to be non-empty")
	}
	_ = tempDir
}
