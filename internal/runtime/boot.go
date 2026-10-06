package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ai-tutor/internal/db"
	"ai-tutor/internal/embeddings"
	"ai-tutor/internal/llm"
	"ai-tutor/internal/notebook"
	"ai-tutor/internal/retrieval"
	"ai-tutor/internal/scheduler"
	"ai-tutor/internal/study"
	"ai-tutor/internal/utils"

	"github.com/joho/godotenv"
)

const (
	// defaultDirPerm defines directory permissions for app data directories.
	defaultDirPerm = 0o755

	// startupBackupDelay is the duration before performing non-blocking startup database backup.
	startupBackupDelay = 3 * time.Second
)

// BootResult holds the initialized states and services for the application.
type BootResult struct {
	mu sync.RWMutex

	// Repo is the SQLite database repository.
	Repo *db.Repository

	// Embedder is the local ONNX embedding engine (nil until RAG/AI is initialized).
	Embedder *embeddings.OnnxEmbedder

	// RetrievalEngine handles hybrid semantic & lexical search across topics and notebooks.
	RetrievalEngine *retrieval.Engine

	// FastLLMProvider handles fast reasoning, card evaluation, and chat responses.
	FastLLMProvider *llm.Provider

	// HeavyLLMProvider handles complex extraction, deep synthesis, and heavy reasoning tasks.
	HeavyLLMProvider *llm.Provider

	// Scheduler drives persistent study queue execution.
	Scheduler scheduler.Service

	// NotebookService manages uploaded notebook files and disk operations.
	NotebookService *notebook.Service

	// StudyService orchestrates study sessions, evaluation, and flashcard generation.
	StudyService *study.StudyService

	// NotebookUploadDir is the absolute directory path where uploaded notebook sources are saved.
	NotebookUploadDir string

	// NotesDir is the absolute directory path where markdown study notes and assets are saved.
	NotesDir string

	// AiReady indicates whether local ONNX embedding and vector search are ready for use.
	AiReady bool

	// AiInitError contains the human-readable error description if AI/RAG initialization failed.
	AiInitError string

	// AiInitErr contains the underlying error if AI/RAG initialization failed.
	AiInitErr error

	// backupCancel cancels any pending delayed startup backup if the app shuts down early.
	backupCancel context.CancelFunc
}

// GetRepo returns the current repository instance in a thread-safe manner.
func (b *BootResult) GetRepo() *db.Repository {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Repo
}

// SetRepo atomically updates the current repository instance.
func (b *BootResult) SetRepo(repo *db.Repository) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Repo = repo
}

// GetEmbedder returns the active ONNX embedder in a thread-safe manner.
func (b *BootResult) GetEmbedder() *embeddings.OnnxEmbedder {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Embedder
}

// IsAIReady reports whether AI/vector capabilities are active and ready.
func (b *BootResult) IsAIReady() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.AiReady
}

// GetAIInitError returns the AI initialization error, if any.
func (b *BootResult) GetAIInitError() error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.AiInitErr
}

// GetAIInitErrorString returns the string representation of any AI initialization error.
func (b *BootResult) GetAIInitErrorString() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.AiInitError
}

// SetAIState updates AI readiness, embedder, and error state under lock.
func (b *BootResult) SetAIState(ready bool, emb *embeddings.OnnxEmbedder, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.AiReady = ready
	b.Embedder = emb
	b.AiInitErr = err
	if err != nil {
		b.AiInitError = err.Error()
	} else {
		b.AiInitError = ""
	}
}

// Close gracefully releases all resources held by BootResult, including cancelling
// pending background backups, closing the ONNX embedder, and closing the database pool.
func (b *BootResult) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.backupCancel != nil {
		b.backupCancel()
		b.backupCancel = nil
	}

	var errs []error
	if b.Embedder != nil {
		if err := b.Embedder.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing embedder: %w", err))
		}
		b.Embedder = nil
	}

	if b.Repo != nil {
		if err := b.Repo.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing repository: %w", err))
		}
		b.Repo = nil
	}

	if len(errs) > 0 {
		return fmt.Errorf("error during BootResult shutdown: %v", errs)
	}
	return nil
}

// Bootstrap runs the entire system initialization sequence: directory resolution,
// database initialization (pre-extension phase), service instantiation, and initial
// lexical index loading.
func Bootstrap(ctx context.Context) (*BootResult, error) {
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}

	res := &BootResult{
		AiReady: false,
	}

	dbPath, err := ResolveDBPath()
	if err != nil {
		res.SetAIState(false, nil, err)
		utils.Errorf("resolving database path: %v", err)
		return nil, err
	}

	// 1. Initialize DB without loading vec0 extension first (so we can query settings safely)
	repo, err := db.Init(dbPath, "")
	if err != nil {
		res.SetAIState(false, nil, err)
		utils.Errorf("initializing database: %v", err)
		return nil, err
	}
	res.SetRepo(repo)
	utils.Infof("Database initialized at %s (extension pre-load phase)", dbPath)

	// Launch non-blocking startup backup after DB successfully initializes
	backupCtx := ctx
	if backupCtx == nil {
		backupCtx = context.Background()
	}
	bCtx, bCancel := context.WithCancel(backupCtx)
	res.backupCancel = bCancel

	go func() {
		select {
		case <-time.After(startupBackupDelay):
			if backupErr := db.BackupDatabase(dbPath); backupErr != nil {
				utils.Warnf("startup database backup warning: %v", backupErr)
			}
		case <-bCtx.Done():
			return
		}
	}()

	// Apply persisted log level immediately upon DB initialization
	if savedLogLevel, lErr := repo.GetLogLevel(); lErr == nil && savedLogLevel != "" {
		_ = utils.SetLogLevel(savedLogLevel)
		utils.Debugf("Active log level initialized to: %s", savedLogLevel)
	}

	// Clean up any interrupted extractions from a crash or sudden app close
	if resetErr := repo.ResetInterruptedNotebookStatuses(); resetErr != nil {
		utils.Warnf("failed to reset interrupted notebook statuses: %v", resetErr)
	}

	// Start cloud sync background worker
	study.StartCloudSyncLoop(repo)

	// Instantiate queue scheduler. Dependencies is empty at bootstrap because the
	// study queue operates directly over the database repository; auxiliary hooks
	// are injected during task execution.
	res.Scheduler = scheduler.New(repo, scheduler.Dependencies{})

	// Initialize retrieval engine with initial lexical chunks
	res.RetrievalEngine = initRetrievalEngine(repo)

	// Configure LLM prompt logging if enabled in database
	if loggingEnabled, err := repo.GetLLMPromptLogging(); err == nil {
		llm.SetPromptLoggingEnabled(loggingEnabled)
	}

	// Initialize fast and heavy LLM providers
	res.FastLLMProvider, res.HeavyLLMProvider = initLLMProviders(repo)

	// Resolve study notes directory
	notesDir, err := ResolveNotesDir()
	if err != nil {
		utils.Warnf("resolving notes directory: %v", err)
	}
	res.NotesDir = notesDir

	// Construct core StudyService orchestrator
	res.StudyService = study.NewStudyService(study.Config{
		Repo:             repo,
		FastLLMProvider:  res.FastLLMProvider,
		HeavyLLMProvider: res.HeavyLLMProvider,
		RetrievalEngine:  res.RetrievalEngine,
		NotesDir:         notesDir,
	})

	// Resolve and initialize notebook service directory
	notebookDir, err := ResolveNotebookDir()
	if err != nil {
		utils.Errorf("resolving notebook directory: %v", err)
		return nil, err
	}
	res.NotebookUploadDir = notebookDir
	res.NotebookService = initNotebookService(notebookDir)

	utils.Infof("App initialized successfully")
	return res, nil
}

// InitializeAI runs the complete RAG/vector initialization sequence: asset verification,
// DLL staging, DB connection reload with sqlite-vec extension, and ONNX embedder stack setup.
// It gracefully degrades if AI assets are unavailable, keeping the plain DB running.
func (b *BootResult) InitializeAI(ctx context.Context) (*embeddings.OnnxEmbedder, error) {
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}

	dbPath, err := ResolveDBPath()
	if err != nil {
		b.SetAIState(false, nil, err)
		return nil, err
	}

	am, err := NewAssetManager(ctx)
	if err != nil {
		initErr := fmt.Errorf("asset manager init failed: %w", err)
		b.SetAIState(false, nil, initErr)
		utils.Warnf("%v", initErr)
		return nil, nil
	}

	if err := am.EnsureAssetsReady(); err != nil {
		initErr := fmt.Errorf("RAG assets not ready: %w", err)
		b.SetAIState(false, nil, initErr)
		utils.Warnf("%v", initErr)
		return nil, nil
	}

	if _, err := am.StageDLLs(); err != nil {
		initErr := fmt.Errorf("failed to stage DLLs: %w", err)
		b.SetAIState(false, nil, initErr)
		utils.Warnf("%v", initErr)
		return nil, nil
	}

	// Close pre-extension repository pool first to prevent Windows file locking conflicts
	prevRepo := b.GetRepo()
	if prevRepo != nil {
		_ = prevRepo.Close()
	}

	newRepo, err := db.Init(dbPath, am.Vec0DllPath())
	if err != nil {
		initErr := fmt.Errorf("failed to reload DB with vector extension: %w", err)
		b.SetAIState(false, nil, initErr)
		utils.Errorf("%v. Falling back to non-vector DB initialization.", initErr)

		fbRepo, fbErr := db.Init(dbPath, "")
		if fbErr != nil {
			return nil, fmt.Errorf("failed to reload DB even without vector extension: %w", fbErr)
		}
		b.SetRepo(fbRepo)
		if b.RetrievalEngine != nil {
			b.RetrievalEngine.SetRepo(fbRepo)
		}
		return nil, nil
	}

	b.SetRepo(newRepo)
	if b.RetrievalEngine != nil {
		b.RetrievalEngine.SetRepo(newRepo)
	}

	if !newRepo.IsVecExtensionLoaded() {
		initErr := errors.New("sqlite-vec extension is missing or failed to load (requires CGO and vec0 binary)")
		b.SetAIState(false, nil, initErr)
		utils.Warnf("%v", initErr)
		return nil, nil
	}

	// Vector DB is live — bring up embedder stack
	return b.initEmbedderStack(am)
}

// initEmbedderStack creates the ONNX embedder, tokenizer, and vector schema table.
func (b *BootResult) initEmbedderStack(am *AssetManager) (*embeddings.OnnxEmbedder, error) {
	emb, err := embeddings.NewOnnxEmbedder(am.ModelPath(), am.TokenizerPath(), am.OnnxRuntimePath())
	if err != nil {
		initErr := fmt.Errorf("failed to load ONNX embedder: %w", err)
		b.SetAIState(false, nil, initErr)
		utils.Warnf("%v", initErr)
		return nil, nil
	}

	if err := embeddings.InitPromptTokenizer(am.TokenizerPath()); err != nil {
		initErr := fmt.Errorf("could not initialize prompt tokenizer: %w", err)
		b.SetAIState(false, nil, initErr)
		utils.Warnf("%v", initErr)
		_ = emb.Close()
		return nil, nil
	}

	currentRepo := b.GetRepo()
	if currentRepo != nil {
		if err := currentRepo.InitWithVectorDimension(emb.GetDimension()); err != nil {
			initErr := fmt.Errorf("could not initialize vector table: %w", err)
			b.SetAIState(false, nil, initErr)
			utils.Warnf("%v", initErr)
			_ = emb.Close()
			return nil, nil
		}

		// Reset any stuck INDEXING status back to PENDING for the indexing queue
		if err := currentRepo.ResetIndexingStatus(); err != nil {
			utils.Warnf("failed to reset notebook indexing statuses: %v", err)
		}
	}

	b.SetAIState(true, emb, nil)
	if b.RetrievalEngine != nil {
		b.RetrievalEngine.SetEmbedder(emb)
	}
	return emb, nil
}

// initRetrievalEngine constructs the hybrid retrieval engine and pre-populates lexical chunks.
func initRetrievalEngine(repo *db.Repository) *retrieval.Engine {
	engine := retrieval.NewEngine(repo, nil)
	if repo == nil {
		return engine
	}

	topicIDs, err := repo.GetAllTopicIDs()
	if err != nil {
		utils.Warnf("could not list topics for lexical fallback: %v", err)
		topicIDs = []string{}
	}

	chunksByTopic, err := repo.GetChunksForTopics(topicIDs)
	if err != nil {
		utils.Warnf("could not batch-load chunks: %v", err)
		return engine
	}

	for _, tid := range topicIDs {
		for _, c := range chunksByTopic[tid] {
			engine.AddChunk(c)
		}
	}
	return engine
}

// initLLMProviders loads provider credentials and settings from DB with fallback to environment.
func initLLMProviders(repo *db.Repository) (*llm.Provider, *llm.Provider) {
	llmSettings, err := repo.GetLLMSettings()
	if err != nil {
		utils.Warnf("failed to load LLM settings: %v. Falling back to environment config.", err)
		llmSettings = nil
	}

	if llmSettings != nil {
		fastKey, err := llm.GetAPIKey("fast")
		if err != nil {
			utils.Warnf("FAST_LLM keyring lookup failed or missing: %v", err)
		}
		heavyKey, err := llm.GetAPIKey("heavy")
		if err != nil {
			utils.Warnf("HEAVY_LLM keyring lookup failed or missing: %v", err)
		}
		if llmSettings.UseSameForHeavy {
			heavyKey = fastKey
		} else if heavyKey == "" && fastKey != "" && strings.EqualFold(llmSettings.Heavy.Provider, llmSettings.Fast.Provider) {
			heavyKey = fastKey
		}
		fast := llm.NewProvider(llm.LoadConfigFromSettingsForPrefix("FAST_LLM", llmSettings.Fast, fastKey))
		heavy := llm.NewProvider(llm.LoadConfigFromSettingsForPrefix("HEAVY_LLM", llmSettings.Heavy, heavyKey))
		return fast, heavy
	}

	fast := llm.NewProvider(llm.LoadConfigFromEnvForPrefix("FAST_LLM"))
	heavy := llm.NewProvider(llm.LoadConfigFromEnvForPrefix("HEAVY_LLM"))
	return fast, heavy
}

// initNotebookService initializes the notebook storage service.
func initNotebookService(notebookDir string) *notebook.Service {
	return notebook.NewService(notebookDir)
}

var loadEnvOnce sync.Once

func loadEnv() {
	loadEnvOnce.Do(func() {
		_ = godotenv.Load()
	})
}

// ResolveAppDir resolves and ensures the base application data directory exists.
func ResolveAppDir() (string, error) {
	loadEnv()

	// Default APP_ENV to "production" if unset to prevent unconfigured fallback paths
	if os.Getenv("APP_ENV") == "" {
		_ = os.Setenv("APP_ENV", "production")
	}

	// Dev: keep data in the repository for convenience.
	if os.Getenv("APP_ENV") == "dev" {
		projectRoot, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to resolve project root: %w", err)
		}
		dir := filepath.Join(projectRoot, "dev_data")
		if err := os.MkdirAll(dir, defaultDirPerm); err != nil {
			return "", fmt.Errorf("failed to create dev_data directory: %w", err)
		}
		return dir, nil
	}

	// Prod/default: use a stable per-user directory and guarantee it exists.
	var dir string
	if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
		dir = filepath.Join(cfgDir, "Studyloop")
	} else if cacheDir, err := os.UserCacheDir(); err == nil && cacheDir != "" {
		dir = filepath.Join(cacheDir, "Studyloop")
	} else if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		dir = filepath.Join(homeDir, ".Studyloop")
	} else {
		return "", fmt.Errorf("failed to resolve application data directory")
	}

	if err := os.MkdirAll(dir, defaultDirPerm); err != nil {
		return "", fmt.Errorf("failed to create app data directory %s: %w", dir, err)
	}
	return dir, nil
}

// ResolveDBPath returns the absolute file path to the SQLite database.
func ResolveDBPath() (string, error) {
	appDir, err := ResolveAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "Studyloop.db"), nil
}

// ResolveSessionPath returns the absolute file path to the session persistence file.
func ResolveSessionPath() (string, error) {
	appDir, err := ResolveAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "session.json"), nil
}

// ResolveNotebookDir returns the directory path for storing uploaded notebooks.
func ResolveNotebookDir() (string, error) {
	appDir, err := ResolveAppDir()
	if err != nil {
		return "", err
	}
	uploadDir := filepath.Join(appDir, "uploads")
	if err := os.MkdirAll(uploadDir, defaultDirPerm); err != nil {
		return "", err
	}
	return uploadDir, nil
}

// ResolveNotesDir returns the directory path for storing study notes.
func ResolveNotesDir() (string, error) {
	if custom := strings.TrimSpace(os.Getenv("STUDYLOOP_NOTES_DIR")); custom != "" {
		if err := os.MkdirAll(custom, defaultDirPerm); err != nil {
			return "", fmt.Errorf("failed to create custom notes directory %s: %w", custom, err)
		}
		return custom, nil
	}
	appDir, err := ResolveAppDir()
	if err != nil {
		return "", err
	}
	notesDir := filepath.Join(appDir, "notes")
	if err := os.MkdirAll(notesDir, defaultDirPerm); err != nil {
		return "", fmt.Errorf("failed to create notes directory %s: %w", notesDir, err)
	}
	return notesDir, nil
}
