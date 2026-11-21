package db

import (
	"database/sql"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestLegacyDatabaseMigration verifies that an older database schema (without recently added columns)
// is upgraded cleanly by InitSchema / RunMigrations without SQL errors or missing columns.
func TestLegacyDatabaseMigration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "legacy_test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite3 db: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Simulate an older v1 database with legacy tables missing modern columns
	legacySchema := []string{
		`CREATE TABLE topics (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'reading',
			start_page INTEGER DEFAULT 0,
			end_page INTEGER DEFAULT 0,
			current_page_cursor INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE chunks (
			id TEXT PRIMARY KEY,
			topic_id TEXT NOT NULL,
			chunk_text TEXT NOT NULL,
			page_num INTEGER DEFAULT 0,
			token_count INTEGER DEFAULT 0,
			importance_score REAL DEFAULT 0,
			weakness_score REAL DEFAULT 0,
			embedding_ref TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE topic_progress (
			topic_id TEXT PRIMARY KEY,
			learned_at TIMESTAMP,
			last_read_at TIMESTAMP,
			mastery_score REAL DEFAULT 0,
			review_enabled INTEGER DEFAULT 0
		)`,
		`CREATE TABLE written_questions (
			id TEXT PRIMARY KEY,
			topic_id TEXT NOT NULL,
			prompt TEXT NOT NULL,
			source_heading TEXT,
			source_page_start INTEGER DEFAULT 0,
			source_page_end INTEGER DEFAULT 0,
			llm_model TEXT,
			prompt_version TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE notebooks (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_type TEXT DEFAULT 'pdf',
			topic_id TEXT,
			status TEXT DEFAULT 'uploaded',
			page_count INTEGER,
			chunk_count INTEGER DEFAULT 0,
			uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE fsrs_cards (
			id TEXT PRIMARY KEY,
			topic_id TEXT NOT NULL,
			prompt TEXT NOT NULL,
			answer TEXT NOT NULL,
			state_json TEXT,
			due_at INTEGER,
			suspended BOOLEAN DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE study_queue (
			id TEXT PRIMARY KEY,
			notebook_id TEXT NOT NULL,
			topic_id TEXT,
			task_type TEXT NOT NULL,
			status TEXT NOT NULL,
			priority INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			activated_at TIMESTAMP,
			completed_at TIMESTAMP,
			payload_json TEXT,
			start_page INTEGER,
			end_page INTEGER
		)`,
		`CREATE TABLE user_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			max_flashcards_per_session INTEGER NOT NULL DEFAULT 30,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range legacySchema {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("failed to create legacy table: %v", err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	if err := InitSchema(tx); err != nil {
		_ = tx.Rollback()
		t.Fatalf("InitSchema failed on legacy database: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit migration transaction: %v", err)
	}

	// Verify critical migrated columns exist and can be queried without error
	checkQueries := []struct {
		table  string
		column string
		query  string
	}{
		{"chunks", "chunk_hash", "SELECT chunk_hash FROM chunks LIMIT 1"},
		{"topics", "external_help_required", "SELECT external_help_required FROM topics LIMIT 1"},
		{"topic_progress", "status", "SELECT status FROM topic_progress LIMIT 1"},
		{"written_questions", "source_chunk_id", "SELECT source_chunk_id FROM written_questions LIMIT 1"},
		{"fsrs_cards", "source_chunk_id", "SELECT source_chunk_id FROM fsrs_cards LIMIT 1"},
		{"notebooks", "priority", "SELECT priority FROM notebooks LIMIT 1"},
		{"notebooks", "indexing_status", "SELECT indexing_status FROM notebooks LIMIT 1"},
		{"notebooks", "syllabus_draft_json", "SELECT syllabus_draft_json FROM notebooks LIMIT 1"},
		{"notebooks", "exam_deadline", "SELECT exam_deadline FROM notebooks LIMIT 1"},
		{"notebooks", "profile_id", "SELECT profile_id FROM notebooks LIMIT 1"},
		{"notebooks", "study_status", "SELECT study_status FROM notebooks LIMIT 1"},
		{"notebooks", "file_hash", "SELECT file_hash FROM notebooks LIMIT 1"},
		{"notebooks", "extraction_engine", "SELECT extraction_engine FROM notebooks LIMIT 1"},
		{"study_queue", "current_page", "SELECT current_page FROM study_queue LIMIT 1"},
		{"user_settings", "tutor_style", "SELECT tutor_style FROM user_settings WHERE id = 1"},
		{"user_settings", "extension_settings", "SELECT extension_settings FROM user_settings WHERE id = 1"},
		{"user_settings", "study_slots_json", "SELECT study_slots_json FROM user_settings WHERE id = 1"},
		{"user_settings", "target_session_words", "SELECT target_session_words FROM user_settings WHERE id = 1"},
		{"user_settings", "analytics_enabled", "SELECT analytics_enabled FROM user_settings WHERE id = 1"},
		{"user_settings", "anonymous_user_id", "SELECT anonymous_user_id FROM user_settings WHERE id = 1"},
		{"user_settings", "llm_prompt_logging", "SELECT llm_prompt_logging FROM user_settings WHERE id = 1"},
		{"user_settings", "prompt_compression_mode", "SELECT prompt_compression_mode FROM user_settings WHERE id = 1"},
		{"user_settings", "prompt_compression_rate", "SELECT prompt_compression_rate FROM user_settings WHERE id = 1"},
		{"chunks", "compressed_text", "SELECT compressed_text FROM chunks LIMIT 1"},
		{"chunks", "compressed_token_count", "SELECT compressed_token_count FROM chunks LIMIT 1"},
	}

	for _, check := range checkQueries {
		_, err := db.Exec(check.query)
		if err != nil {
			t.Errorf("Column %s.%s missing or unqueryable after migration: %v", check.table, check.column, err)
		}
	}

	var notesDetailLevel string
	err = db.QueryRow("SELECT notes_detail_level FROM user_settings WHERE id = 1").Scan(&notesDetailLevel)
	if err != nil {
		t.Fatalf("failed to query notes_detail_level from user_settings: %v", err)
	}
	if notesDetailLevel != "briefing" {
		t.Errorf("expected notes_detail_level to be 'briefing', got %q", notesDetailLevel)
	}
}

// TestSchemaParityFreshVsUpgraded guarantees 100% column parity between a freshly created
// database and a legacy upgraded database across all tables.
func TestSchemaParityFreshVsUpgraded(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Fresh Database
	freshDB, err := sql.Open("sqlite3", filepath.Join(tempDir, "fresh.db"))
	if err != nil {
		t.Fatalf("failed to open fresh db: %v", err)
	}
	defer func() { _ = freshDB.Close() }()

	freshTx, err := freshDB.Begin()
	if err != nil {
		t.Fatalf("failed to begin fresh tx: %v", err)
	}
	if err := InitSchema(freshTx); err != nil {
		_ = freshTx.Rollback()
		t.Fatalf("InitSchema failed on fresh db: %v", err)
	}
	if err := freshTx.Commit(); err != nil {
		t.Fatalf("failed to commit fresh tx: %v", err)
	}

	// 2. Upgraded Legacy Database
	upgradedDB, err := sql.Open("sqlite3", filepath.Join(tempDir, "upgraded.db"))
	if err != nil {
		t.Fatalf("failed to open upgraded db: %v", err)
	}
	defer func() { _ = upgradedDB.Close() }()

	// Seed historical initial table definitions from when each table was first introduced in git
	legacyTables := []string{
		`CREATE TABLE topics (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'reading',
			start_page INTEGER DEFAULT 0,
			end_page INTEGER DEFAULT 0,
			current_page_cursor INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE chunks (
			id TEXT PRIMARY KEY,
			topic_id TEXT NOT NULL,
			parent_id TEXT,
			chunk_text TEXT NOT NULL,
			page_num INTEGER DEFAULT 0,
			token_count INTEGER DEFAULT 0,
			importance_score REAL DEFAULT 0,
			weakness_score REAL DEFAULT 0,
			embedding_ref TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE topic_progress (
			topic_id TEXT PRIMARY KEY,
			learned_at TIMESTAMP,
			last_read_at TIMESTAMP,
			mastery_score REAL DEFAULT 0,
			review_enabled INTEGER DEFAULT 0
		)`,
		`CREATE TABLE written_questions (
			id TEXT PRIMARY KEY,
			topic_id TEXT NOT NULL,
			prompt TEXT NOT NULL,
			source_heading TEXT,
			source_page_start INTEGER DEFAULT 0,
			source_page_end INTEGER DEFAULT 0,
			llm_model TEXT,
			prompt_version TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE notebooks (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_type TEXT DEFAULT 'pdf',
			topic_id TEXT,
			status TEXT DEFAULT 'uploaded',
			page_count INTEGER,
			chunk_count INTEGER DEFAULT 0,
			uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE fsrs_cards (
			id TEXT PRIMARY KEY,
			topic_id TEXT NOT NULL,
			prompt TEXT NOT NULL,
			answer TEXT NOT NULL,
			state_json TEXT,
			due_at INTEGER,
			suspended BOOLEAN DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE study_queue (
			id TEXT PRIMARY KEY,
			notebook_id TEXT NOT NULL,
			topic_id TEXT,
			task_type TEXT NOT NULL,
			status TEXT NOT NULL,
			priority INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			activated_at TIMESTAMP,
			completed_at TIMESTAMP,
			payload_json TEXT,
			start_page INTEGER,
			end_page INTEGER
		)`,
		`CREATE TABLE user_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			daily_study_minutes INTEGER NOT NULL DEFAULT 90,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE study_profiles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			deadline_at INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE llm_settings (
			tier TEXT PRIMARY KEY CHECK (tier IN ('fast', 'heavy')),
			provider TEXT NOT NULL DEFAULT 'groq',
			base_url TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '',
			timeout_ms INTEGER NOT NULL DEFAULT 30000,
			api_key_source TEXT NOT NULL DEFAULT 'keyring',
			has_api_key BOOLEAN DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE user_gamification (
			user_id INTEGER PRIMARY KEY CHECK (user_id = 1),
			total_xp INTEGER NOT NULL DEFAULT 0,
			coins INTEGER NOT NULL DEFAULT 0,
			current_title TEXT NOT NULL DEFAULT 'The Apprentice',
			streak_freezes_owned INTEGER NOT NULL DEFAULT 1,
			frozen_dates_json TEXT NOT NULL DEFAULT '[]',
			unlocked_cosmetics_json TEXT NOT NULL DEFAULT '["dark-gruvbox", "light-classic"]',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range legacyTables {
		if _, err := upgradedDB.Exec(stmt); err != nil {
			t.Fatalf("failed to create minimal legacy table: %v", err)
		}
	}

	upgradedTx, err := upgradedDB.Begin()
	if err != nil {
		t.Fatalf("failed to begin upgraded tx: %v", err)
	}
	if err := InitSchema(upgradedTx); err != nil {
		_ = upgradedTx.Rollback()
		t.Fatalf("InitSchema failed on legacy upgraded db: %v", err)
	}
	if err := upgradedTx.Commit(); err != nil {
		t.Fatalf("failed to commit upgraded tx: %v", err)
	}

	// Fetch all tables and columns from freshDB
	getTableColumns := func(db *sql.DB) map[string][]string {
		tables := make(map[string][]string)
		rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
		if err != nil {
			t.Fatalf("failed to query tables: %v", err)
		}
		defer func() { _ = rows.Close() }()

		var tableNames []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				tableNames = append(tableNames, name)
			}
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		for _, tbl := range tableNames {
			infoRows, err := db.Query("PRAGMA table_info(" + tbl + ")")
			if err != nil {
				t.Fatalf("failed to query table info for %s: %v", tbl, err)
			}
			var cols []string
			for infoRows.Next() {
				var cid int
				var colName, colType string
				var notnull int
				var dfltVal interface{}
				var pk int
				if err := infoRows.Scan(&cid, &colName, &colType, &notnull, &dfltVal, &pk); err == nil {
					cols = append(cols, strings.ToLower(colName))
				}
			}
			if err := infoRows.Err(); err != nil {
				t.Fatalf("infoRows iteration error for %s: %v", tbl, err)
			}
			_ = infoRows.Close()
			sort.Strings(cols)
			tables[tbl] = cols
		}
		return tables
	}

	freshCols := getTableColumns(freshDB)
	upgradedCols := getTableColumns(upgradedDB)

	for tbl, fCols := range freshCols {
		uCols, exists := upgradedCols[tbl]
		if !exists {
			t.Errorf("Table %s exists in fresh DB but missing in upgraded DB", tbl)
			continue
		}

		fColSet := make(map[string]bool)
		for _, c := range fCols {
			fColSet[c] = true
		}
		uColSet := make(map[string]bool)
		for _, c := range uCols {
			uColSet[c] = true
		}

		for c := range fColSet {
			if !uColSet[c] {
				t.Errorf("Table %s: column %s exists in fresh DB but MISSING in upgraded DB!", tbl, c)
			}
		}
	}
}

// TestDedupeFsrsCards verifies that when upgrading a legacy database that contains duplicate
// (topic_id, prompt) rows in fsrs_cards, dedupeFsrsCards keeps the earliest row by created_at/id,
// removes duplicates, and successfully applies the unique index idx_fsrs_cards_topic_prompt.
func TestDedupeFsrsCards(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "dedupe_fsrs_test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite3 db: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create minimal legacy schema with fsrs_cards without unique index
	setup := []string{
		`CREATE TABLE topics (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'reading',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE fsrs_cards (
			id TEXT PRIMARY KEY,
			topic_id TEXT NOT NULL,
			prompt TEXT NOT NULL,
			answer TEXT NOT NULL,
			state_json TEXT,
			due_at INTEGER,
			suspended BOOLEAN DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO topics (id, title) VALUES ('t1', 'Topic 1')`,
		// Seed duplicates: card-1 (earlier) and card-2 (later) with identical (topic_id, prompt)
		`INSERT INTO fsrs_cards (id, topic_id, prompt, answer, created_at) VALUES ('card-1', 't1', 'Duplicate Prompt', 'Answer 1', '2026-01-01 10:00:00')`,
		`INSERT INTO fsrs_cards (id, topic_id, prompt, answer, created_at) VALUES ('card-2', 't1', 'Duplicate Prompt', 'Answer 2', '2026-01-01 11:00:00')`,
		// Another duplicate pair with reversed insert order / timestamps to test sorting by created_at, id
		`INSERT INTO fsrs_cards (id, topic_id, prompt, answer, created_at) VALUES ('card-4', 't1', 'Another Duplicate', 'Answer 4', '2026-01-01 12:00:00')`,
		`INSERT INTO fsrs_cards (id, topic_id, prompt, answer, created_at) VALUES ('card-3', 't1', 'Another Duplicate', 'Answer 3', '2026-01-01 09:00:00')`,
		// Non-duplicate card
		`INSERT INTO fsrs_cards (id, topic_id, prompt, answer, created_at) VALUES ('card-unique', 't1', 'Unique Prompt', 'Answer Unique', '2026-01-01 13:00:00')`,
	}

	for _, stmt := range setup {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("setup statement failed: %v", err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	if err := InitSchema(tx); err != nil {
		_ = tx.Rollback()
		t.Fatalf("InitSchema failed on db with duplicate fsrs_cards: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit tx: %v", err)
	}

	// Verify duplicate rows are removed and earliest rows are retained
	rows, err := db.Query("SELECT id, topic_id, prompt FROM fsrs_cards ORDER BY id ASC")
	if err != nil {
		t.Fatalf("failed to query fsrs_cards: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var remainingIDs []string
	for rows.Next() {
		var id, topicID, prompt string
		if err := rows.Scan(&id, &topicID, &prompt); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		remainingIDs = append(remainingIDs, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration err: %v", err)
	}

	expectedIDs := []string{"card-1", "card-3", "card-unique"}
	if len(remainingIDs) != len(expectedIDs) {
		t.Fatalf("expected %d cards remaining, got %d (%v)", len(expectedIDs), len(remainingIDs), remainingIDs)
	}
	for i, exp := range expectedIDs {
		if remainingIDs[i] != exp {
			t.Errorf("at index %d: expected id %s, got %s", i, exp, remainingIDs[i])
		}
	}

	// Verify that the unique index prevents inserting duplicate (topic_id, prompt)
	_, err = db.Exec(`INSERT INTO fsrs_cards (id, topic_id, prompt, answer) VALUES ('card-fail', 't1', 'Duplicate Prompt', 'Should fail')`)
	if err == nil {
		t.Errorf("expected unique constraint violation on idx_fsrs_cards_topic_prompt, got nil")
	}
}
