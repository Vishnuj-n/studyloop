package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// alterStatements defines all column additions needed to migrate existing databases forward.
// Every newly added column to an existing table MUST have an entry here for backward compatibility.
var alterStatements = []struct {
	Table  string
	Column string
	SQL    string
}{
	// user_settings
	{"user_settings", "max_flashcards_per_session", "ALTER TABLE user_settings ADD COLUMN max_flashcards_per_session INTEGER NOT NULL DEFAULT 30"},
	{"user_settings", "study_start_time", "ALTER TABLE user_settings ADD COLUMN study_start_time TEXT DEFAULT '17:00'"},
	{"user_settings", "study_end_time", "ALTER TABLE user_settings ADD COLUMN study_end_time TEXT DEFAULT '18:00'"},
	{"user_settings", "study_slots_json", "ALTER TABLE user_settings ADD COLUMN study_slots_json TEXT DEFAULT '[]'"},
	{"user_settings", "reminders_enabled", "ALTER TABLE user_settings ADD COLUMN reminders_enabled BOOLEAN DEFAULT 1"},
	{"user_settings", "show_reward_notifications", "ALTER TABLE user_settings ADD COLUMN show_reward_notifications BOOLEAN DEFAULT 1"},
	{"user_settings", "active_profile_id", "ALTER TABLE user_settings ADD COLUMN active_profile_id TEXT"},
	{"user_settings", "skip_to_reading_active", "ALTER TABLE user_settings ADD COLUMN skip_to_reading_active BOOLEAN DEFAULT 0"},
	{"user_settings", "cloud_sync_url", "ALTER TABLE user_settings ADD COLUMN cloud_sync_url TEXT DEFAULT ''"},
	{"user_settings", "cloud_api_token", "ALTER TABLE user_settings ADD COLUMN cloud_api_token TEXT DEFAULT ''"},
	{"user_settings", "theme", "ALTER TABLE user_settings ADD COLUMN theme TEXT DEFAULT 'dark-gruvbox'"},
	{"user_settings", "rag_enabled", "ALTER TABLE user_settings ADD COLUMN rag_enabled BOOLEAN DEFAULT 0"},
	{"user_settings", "rag_notebook_chapter", "ALTER TABLE user_settings ADD COLUMN rag_notebook_chapter BOOLEAN DEFAULT 1"},
	{"user_settings", "rag_entire_notebook", "ALTER TABLE user_settings ADD COLUMN rag_entire_notebook BOOLEAN DEFAULT 1"},
	{"user_settings", "rag_queue_study", "ALTER TABLE user_settings ADD COLUMN rag_queue_study BOOLEAN DEFAULT 1"},
	{"user_settings", "default_remedial_strategy", "ALTER TABLE user_settings ADD COLUMN default_remedial_strategy TEXT DEFAULT 'FAST'"},
	{"user_settings", "classroom_code", "ALTER TABLE user_settings ADD COLUMN classroom_code TEXT DEFAULT ''"},
	{"user_settings", "student_username", "ALTER TABLE user_settings ADD COLUMN student_username TEXT DEFAULT ''"},
	{"user_settings", "last_synced_at", "ALTER TABLE user_settings ADD COLUMN last_synced_at INTEGER DEFAULT 0"},
	{"user_settings", "target_session_words", "ALTER TABLE user_settings ADD COLUMN target_session_words INTEGER NOT NULL DEFAULT 3000"},
	{"user_settings", "min_session_words", "ALTER TABLE user_settings ADD COLUMN min_session_words INTEGER NOT NULL DEFAULT 0"},
	{"user_settings", "max_active_notebooks", "ALTER TABLE user_settings ADD COLUMN max_active_notebooks INTEGER NOT NULL DEFAULT 4"},
	{"user_settings", "quiz_question_count", "ALTER TABLE user_settings ADD COLUMN quiz_question_count INTEGER NOT NULL DEFAULT 8"},
	{"user_settings", "quiz_passing_score", "ALTER TABLE user_settings ADD COLUMN quiz_passing_score INTEGER NOT NULL DEFAULT 70"},
	{"user_settings", "tutor_style", "ALTER TABLE user_settings ADD COLUMN tutor_style TEXT NOT NULL DEFAULT 'socratic'"},
	{"user_settings", "analytics_enabled", "ALTER TABLE user_settings ADD COLUMN analytics_enabled BOOLEAN DEFAULT 0"},
	{"user_settings", "anonymous_user_id", "ALTER TABLE user_settings ADD COLUMN anonymous_user_id TEXT DEFAULT ''"},
	{"user_settings", "extension_settings", "ALTER TABLE user_settings ADD COLUMN extension_settings TEXT DEFAULT '{}'"},

	// study_profiles
	{"study_profiles", "classroom_code", "ALTER TABLE study_profiles ADD COLUMN classroom_code TEXT DEFAULT ''"},
	{"study_profiles", "student_username", "ALTER TABLE study_profiles ADD COLUMN student_username TEXT DEFAULT ''"},
	{"study_profiles", "cloud_api_token", "ALTER TABLE study_profiles ADD COLUMN cloud_api_token TEXT DEFAULT ''"},
	{"study_profiles", "pomo_duration_sec", "ALTER TABLE study_profiles ADD COLUMN pomo_duration_sec INTEGER NOT NULL DEFAULT 1500"},
	{"study_profiles", "pomo_break_sec", "ALTER TABLE study_profiles ADD COLUMN pomo_break_sec INTEGER NOT NULL DEFAULT 300"},
	{"study_profiles", "pomo_music_path", "ALTER TABLE study_profiles ADD COLUMN pomo_music_path TEXT DEFAULT ''"},
	{"study_profiles", "pomo_shuffle", "ALTER TABLE study_profiles ADD COLUMN pomo_shuffle BOOLEAN DEFAULT 0"},
	{"study_profiles", "target_session_words", "ALTER TABLE study_profiles ADD COLUMN target_session_words INTEGER DEFAULT NULL"},
	{"study_profiles", "min_session_words", "ALTER TABLE study_profiles ADD COLUMN min_session_words INTEGER DEFAULT NULL"},
	{"study_profiles", "theme", "ALTER TABLE study_profiles ADD COLUMN theme TEXT DEFAULT ''"},
	{"study_profiles", "max_flashcards_per_session", "ALTER TABLE study_profiles ADD COLUMN max_flashcards_per_session INTEGER DEFAULT NULL"},
	{"study_profiles", "max_active_notebooks", "ALTER TABLE study_profiles ADD COLUMN max_active_notebooks INTEGER DEFAULT NULL"},
	{"study_profiles", "skip_to_reading_active", "ALTER TABLE study_profiles ADD COLUMN skip_to_reading_active BOOLEAN DEFAULT NULL"},
	{"study_profiles", "default_remedial_strategy", "ALTER TABLE study_profiles ADD COLUMN default_remedial_strategy TEXT DEFAULT ''"},
	{"study_profiles", "quiz_question_count", "ALTER TABLE study_profiles ADD COLUMN quiz_question_count INTEGER DEFAULT NULL"},
	{"study_profiles", "quiz_passing_score", "ALTER TABLE study_profiles ADD COLUMN quiz_passing_score INTEGER DEFAULT NULL"},
	{"study_profiles", "tutor_style", "ALTER TABLE study_profiles ADD COLUMN tutor_style TEXT DEFAULT ''"},

	// topics
	{"topics", "external_help_required", "ALTER TABLE topics ADD COLUMN external_help_required BOOLEAN DEFAULT 0"},

	// topic_progress
	{"topic_progress", "status", "ALTER TABLE topic_progress ADD COLUMN status TEXT DEFAULT 'active'"},

	// notebooks
	{"notebooks", "file_hash", "ALTER TABLE notebooks ADD COLUMN file_hash TEXT DEFAULT ''"},
	{"notebooks", "priority", "ALTER TABLE notebooks ADD COLUMN priority INTEGER DEFAULT 5"},
	{"notebooks", "indexing_status", "ALTER TABLE notebooks ADD COLUMN indexing_status TEXT DEFAULT 'PENDING'"},
	{"notebooks", "syllabus_draft_json", "ALTER TABLE notebooks ADD COLUMN syllabus_draft_json TEXT"},
	{"notebooks", "exam_deadline", "ALTER TABLE notebooks ADD COLUMN exam_deadline TEXT"},
	{"notebooks", "profile_id", "ALTER TABLE notebooks ADD COLUMN profile_id TEXT"},
	{"notebooks", "study_status", "ALTER TABLE notebooks ADD COLUMN study_status TEXT DEFAULT 'dormant'"},
	{"notebooks", "extraction_engine", "ALTER TABLE notebooks ADD COLUMN extraction_engine TEXT DEFAULT 'standard'"},

	// chunks
	{"chunks", "chunk_hash", "ALTER TABLE chunks ADD COLUMN chunk_hash TEXT DEFAULT ''"},

	// written_questions & fsrs_cards
	{"written_questions", "source_chunk_id", "ALTER TABLE written_questions ADD COLUMN source_chunk_id TEXT"},
	{"fsrs_cards", "source_chunk_id", "ALTER TABLE fsrs_cards ADD COLUMN source_chunk_id TEXT"},

	// llm_settings
	{"llm_settings", "max_input_tokens", "ALTER TABLE llm_settings ADD COLUMN max_input_tokens INTEGER NOT NULL DEFAULT 4000"},
	{"llm_settings", "max_output_tokens", "ALTER TABLE llm_settings ADD COLUMN max_output_tokens INTEGER NOT NULL DEFAULT 2500"},

	// study_queue
	{"study_queue", "current_page", "ALTER TABLE study_queue ADD COLUMN current_page INTEGER"},

	// gamification
	{"user_gamification", "stats_json", "ALTER TABLE user_gamification ADD COLUMN stats_json TEXT NOT NULL DEFAULT '{}'"},
	{"user_gamification", "last_freeze_purchased_at", "ALTER TABLE user_gamification ADD COLUMN last_freeze_purchased_at INTEGER NOT NULL DEFAULT 0"},
}

// RunMigrations applies idempotent column additions, table deduping, and data backfills.
func RunMigrations(tx *sql.Tx) error {
	// 1. Deduplicate notebook_chunks for older databases
	if err := dedupeNotebookChunks(tx); err != nil {
		return err
	}

	// 2. Apply alter statements for missing columns
	for _, alter := range alterStatements {
		exists, err := columnExists(tx, alter.Table, alter.Column)
		if err != nil {
			return fmt.Errorf("failed to check column %s in table %s: %w", alter.Column, alter.Table, err)
		}
		if !exists {
			if _, err := tx.Exec(alter.SQL); err != nil {
				return fmt.Errorf("failed to execute alter statement for %s.%s: %w", alter.Table, alter.Column, err)
			}
		}
	}

	// 3. Seed default rows and backfill settings
	if err := seedDefaults(tx); err != nil {
		return err
	}

	return nil
}

func dedupeNotebookChunks(tx *sql.Tx) error {
	var count int
	err := tx.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='notebook_chunks'`).Scan(&count)
	if err != nil || count == 0 {
		return nil
	}

	rows, err := tx.Query(`
		SELECT id, notebook_id, chunk_id
		FROM notebook_chunks
		ORDER BY notebook_id, chunk_id, created_at ASC
	`)
	if err != nil {
		return fmt.Errorf("failed to query notebook_chunks for dedupe: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	seen := make(map[string]bool)
	var idsToDelete []string
	for rows.Next() {
		var id, nb, cid string
		if err := rows.Scan(&id, &nb, &cid); err != nil {
			return fmt.Errorf("failed to scan notebook_chunks dedupe row id=%s notebook_id=%s chunk_id=%s: %w", id, nb, cid, err)
		}
		key := nb + "::" + cid
		if seen[key] {
			idsToDelete = append(idsToDelete, id)
			continue
		}
		seen[key] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed while iterating notebook_chunks dedupe rows: %w", err)
	}

	for _, id := range idsToDelete {
		if _, err := tx.Exec(`DELETE FROM notebook_chunks WHERE id = ?`, id); err != nil {
			return fmt.Errorf("failed to delete duplicate notebook_chunks row id=%s: %w", id, err)
		}
	}

	if _, err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_notebook_chunk_unique ON notebook_chunks(notebook_id, chunk_id)`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
			return fmt.Errorf("failed to create unique index on notebook_chunks: %w", err)
		}
	}
	return nil
}

func seedDefaults(tx *sql.Tx) error {
	// Default user settings
	if _, err := tx.Exec(`
		INSERT INTO user_settings (id, max_flashcards_per_session, study_start_time, study_end_time, reminders_enabled)
		VALUES (1, 30, '17:00', '18:00', 1)
		ON CONFLICT(id) DO NOTHING
	`); err != nil {
		return fmt.Errorf("failed to initialize user settings: %w", err)
	}

	// Default LLM settings
	if _, err := tx.Exec(`
		INSERT INTO llm_settings (tier, provider, base_url, model, timeout_ms, max_input_tokens, max_output_tokens, api_key_source, has_api_key)
		VALUES
			('fast', 'groq', 'https://api.groq.com/openai/v1', 'openai/gpt-oss-120b', 60000, 4000, 2500, 'keyring', 0),
			('heavy', 'groq', 'https://api.groq.com/openai/v1', 'openai/gpt-oss-120b', 90000, 4000, 2500, 'keyring', 0)
		ON CONFLICT(tier) DO NOTHING
	`); err != nil {
		return fmt.Errorf("failed to initialize llm settings: %w", err)
	}

	// Upgrade legacy llm_settings max_output_tokens from 1000 to 2500
	if _, err := tx.Exec(`
		UPDATE llm_settings
		SET max_output_tokens = 2500
		WHERE max_output_tokens = 1000
	`); err != nil {
		return fmt.Errorf("failed to update legacy max_output_tokens: %w", err)
	}

	// Upgrade any previously seeded groq base_url that missed /v1
	if _, err := tx.Exec(`
		UPDATE llm_settings
		SET base_url = 'https://api.groq.com/openai/v1'
		WHERE provider = 'groq' AND (base_url = 'https://api.groq.com/openai' OR base_url = 'https://api.groq.com/openai/')
	`); err != nil {
		return fmt.Errorf("failed to update legacy groq base_url: %w", err)
	}

	// Default gamification state
	if _, err := tx.Exec(`
		INSERT INTO user_gamification (user_id, total_xp, coins, current_title, streak_freezes_owned, unlocked_cosmetics_json, stats_json)
		VALUES (1, 0, 0, 'The Apprentice', 1, '["dark-gruvbox", "light-classic"]', '{}')
		ON CONFLICT(user_id) DO NOTHING
	`); err != nil {
		return fmt.Errorf("failed to initialize user gamification: %w", err)
	}

	// Backfill missing starter cosmetics into existing user profiles
	var currentUnlockedJSON string
	err := tx.QueryRow(`SELECT unlocked_cosmetics_json FROM user_gamification WHERE user_id = 1`).Scan(&currentUnlockedJSON)
	if err == nil && currentUnlockedJSON != "" {
		var unlocked []string
		if jsonErr := json.Unmarshal([]byte(currentUnlockedJSON), &unlocked); jsonErr == nil {
			hasGruvbox := false
			hasClassic := false
			for _, id := range unlocked {
				if id == "dark-gruvbox" {
					hasGruvbox = true
				}
				if id == "light-classic" {
					hasClassic = true
				}
			}
			changed := false
			if !hasGruvbox {
				unlocked = append(unlocked, "dark-gruvbox")
				changed = true
			}
			if !hasClassic {
				unlocked = append(unlocked, "light-classic")
				changed = true
			}
			if changed {
				newBytes, _ := json.Marshal(unlocked)
				if _, err := tx.Exec(`UPDATE user_gamification SET unlocked_cosmetics_json = ? WHERE user_id = 1`, string(newBytes)); err != nil {
					return fmt.Errorf("failed to backfill default cosmetics: %w", err)
				}
			}
		}
	}

	return nil
}

func columnExists(tx *sql.Tx, table, column string) (bool, error) {
	rows, err := tx.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var cid int
		var name, typeStr string
		var notnull int
		var dfltValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &typeStr, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, column) {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}
