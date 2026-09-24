package db

import (
	"database/sql"
	"fmt"
)

// InitSchema creates all tables and indexes with a single clean schema.
// This is a "nuclear" initialization that does NOT preserve existing data.
// All tables are created with CREATE TABLE IF NOT EXISTS and include all
// Sprint 14 requirements: FSRS tables, written questions, and current_page_cursor.
func InitSchema(tx *sql.Tx) error {
	schema := []string{
		// Profiles table
		`CREATE TABLE IF NOT EXISTS study_profiles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			deadline_at INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			classroom_code TEXT DEFAULT '',
			student_username TEXT DEFAULT '',
			cloud_api_token TEXT DEFAULT '',
			pomo_duration_sec INTEGER NOT NULL DEFAULT 1500,
			pomo_break_sec INTEGER NOT NULL DEFAULT 300,
			pomo_music_path TEXT DEFAULT '',
			pomo_shuffle BOOLEAN DEFAULT 0,
			target_session_words INTEGER DEFAULT NULL,
			min_session_words INTEGER DEFAULT NULL,
			theme TEXT DEFAULT '',
			max_flashcards_per_session INTEGER DEFAULT NULL,
			max_active_notebooks INTEGER DEFAULT NULL,
			skip_to_reading_active BOOLEAN DEFAULT NULL,
			default_remedial_strategy TEXT DEFAULT '',
			quiz_question_count INTEGER DEFAULT NULL,
			quiz_passing_score INTEGER DEFAULT NULL,
			tutor_style TEXT DEFAULT ''
		)`,

		// Core tables
		`CREATE TABLE IF NOT EXISTS topics (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'reading',
			start_page INTEGER DEFAULT 0,
			end_page INTEGER DEFAULT 0,
			current_page_cursor INTEGER DEFAULT 0,
			external_help_required BOOLEAN DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS chunks (
			id TEXT PRIMARY KEY,
			topic_id TEXT NOT NULL,
			chunk_text TEXT NOT NULL,
			chunk_hash TEXT DEFAULT '',
			page_num INTEGER DEFAULT 0,
			token_count INTEGER DEFAULT 0,
			importance_score REAL DEFAULT 0,
			weakness_score REAL DEFAULT 0,
			embedding_ref TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (topic_id) REFERENCES topics(id)
		)`,

		`CREATE TABLE IF NOT EXISTS topic_progress (
			topic_id TEXT PRIMARY KEY,
			learned_at TIMESTAMP,
			last_read_at TIMESTAMP,
			mastery_score REAL DEFAULT 0,
			review_enabled INTEGER DEFAULT 0,
			status TEXT DEFAULT 'active'
		)`,

		`CREATE TABLE IF NOT EXISTS quiz_attempts (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			score INTEGER NOT NULL,
			passed INTEGER NOT NULL,
			answers_json TEXT NOT NULL,
			feedback TEXT,
			completed_at INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (task_id) REFERENCES study_queue(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS reread_attempts (
			topic_id TEXT PRIMARY KEY,
			attempt_count INTEGER NOT NULL DEFAULT 0,
			last_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
		)`,

		// Written questions and user answers
		`CREATE TABLE IF NOT EXISTS written_questions (
			id TEXT PRIMARY KEY,
			topic_id TEXT NOT NULL,
			prompt TEXT NOT NULL,
			source_chunk_id TEXT,
			source_heading TEXT,
			source_page_start INTEGER DEFAULT 0,
			source_page_end INTEGER DEFAULT 0,
			llm_model TEXT,
			prompt_version TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
			FOREIGN KEY (source_chunk_id) REFERENCES chunks(id) ON DELETE SET NULL
		)`,

		`CREATE TABLE IF NOT EXISTS written_user_answers (
			id TEXT PRIMARY KEY,
			written_question_id TEXT NOT NULL,
			user_answer TEXT NOT NULL,
			score INTEGER NOT NULL,
			feedback TEXT,
			source_heading TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (written_question_id) REFERENCES written_questions(id) ON DELETE CASCADE
		)`,

		// User settings
		`CREATE TABLE IF NOT EXISTS user_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			max_flashcards_per_session INTEGER NOT NULL DEFAULT 30,
			study_start_time TEXT DEFAULT '17:00',
			study_end_time TEXT DEFAULT '18:00',
			study_slots_json TEXT DEFAULT '[]',
			reminders_enabled BOOLEAN DEFAULT 1,
			show_reward_notifications BOOLEAN DEFAULT 1,
			active_profile_id TEXT,
			skip_to_reading_active BOOLEAN DEFAULT 0,
			cloud_sync_url TEXT DEFAULT '',
			cloud_api_token TEXT DEFAULT '',
			theme TEXT DEFAULT 'dark-gruvbox',
			rag_enabled BOOLEAN DEFAULT 0,
			rag_notebook_chapter BOOLEAN DEFAULT 1,
			rag_entire_notebook BOOLEAN DEFAULT 1,
			rag_queue_study BOOLEAN DEFAULT 1,
			default_remedial_strategy TEXT DEFAULT 'FAST',
			classroom_code TEXT DEFAULT '',
			student_username TEXT DEFAULT '',
			last_synced_at INTEGER DEFAULT 0,
			target_session_words INTEGER NOT NULL DEFAULT 3000,
			min_session_words INTEGER NOT NULL DEFAULT 0,
			max_active_notebooks INTEGER NOT NULL DEFAULT 4,
			quiz_question_count INTEGER NOT NULL DEFAULT 8,
			quiz_passing_score INTEGER NOT NULL DEFAULT 70,
			tutor_style TEXT NOT NULL DEFAULT 'socratic',
			analytics_enabled BOOLEAN DEFAULT 0,
			anonymous_user_id TEXT DEFAULT '',
			llm_prompt_logging BOOLEAN DEFAULT 0,
			extension_settings TEXT DEFAULT '{}',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (active_profile_id) REFERENCES study_profiles(id) ON DELETE SET NULL
		)`,

		`CREATE TABLE IF NOT EXISTS llm_settings (
			tier TEXT PRIMARY KEY CHECK (tier IN ('fast', 'heavy')),
			provider TEXT NOT NULL DEFAULT 'groq',
			base_url TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '',
			timeout_ms INTEGER NOT NULL DEFAULT 30000,
			max_input_tokens INTEGER NOT NULL DEFAULT 4000,
			max_output_tokens INTEGER NOT NULL DEFAULT 2500,
			api_key_source TEXT NOT NULL DEFAULT 'keyring',
			has_api_key BOOLEAN DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Notebooks
		`CREATE TABLE IF NOT EXISTS notebooks (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_type TEXT DEFAULT 'pdf',
			file_hash TEXT DEFAULT '',
			topic_id TEXT,
			priority INTEGER DEFAULT 5,
			status TEXT DEFAULT 'uploaded',
			indexing_status TEXT DEFAULT 'PENDING',
			page_count INTEGER,
			chunk_count INTEGER DEFAULT 0,
			syllabus_draft_json TEXT,
			exam_deadline TEXT,
			uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			profile_id TEXT,
			study_status TEXT DEFAULT 'dormant',
			extraction_engine TEXT DEFAULT 'standard',
			FOREIGN KEY (topic_id) REFERENCES topics(id),
			FOREIGN KEY (profile_id) REFERENCES study_profiles(id) ON DELETE SET NULL
		)`,

		`CREATE TABLE IF NOT EXISTS notebook_topics (
			notebook_id TEXT NOT NULL,
			topic_id TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (notebook_id, topic_id),
			FOREIGN KEY (notebook_id) REFERENCES notebooks(id) ON DELETE CASCADE,
			FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS notebook_chunks (
			id TEXT PRIMARY KEY,
			notebook_id TEXT NOT NULL,
			chunk_id TEXT NOT NULL,
			page_num INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (notebook_id, chunk_id),
			FOREIGN KEY (notebook_id) REFERENCES notebooks(id),
			FOREIGN KEY (chunk_id) REFERENCES chunks(id)
		)`,

		// FSRS tables (Sprint 14)
		`CREATE TABLE IF NOT EXISTS fsrs_cards (
			id TEXT PRIMARY KEY,
			topic_id TEXT NOT NULL,
			source_chunk_id TEXT,
			prompt TEXT NOT NULL,
			answer TEXT NOT NULL,
			state_json TEXT,
			due_at INTEGER,
			suspended BOOLEAN DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
			FOREIGN KEY (source_chunk_id) REFERENCES chunks(id) ON DELETE SET NULL
		)`,

		`CREATE TABLE IF NOT EXISTS fsrs_review_log (
			id TEXT PRIMARY KEY,
			topic_id TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			reference_id TEXT NOT NULL,
			reviewed_at INTEGER NOT NULL,
			rating INTEGER NOT NULL,
			scheduled_days INTEGER NOT NULL,
			state_before_json TEXT NOT NULL,
			state_after_json TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
		)`,

		// Study queue (Sprint 1 foundation)
		`CREATE TABLE IF NOT EXISTS study_queue (
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
			end_page INTEGER,
			current_page INTEGER,
			FOREIGN KEY (notebook_id) REFERENCES notebooks(id),
			FOREIGN KEY (topic_id) REFERENCES topics(id)
		)`,

		`CREATE TABLE IF NOT EXISTS review_task_cards (
			task_id TEXT NOT NULL,
			card_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			PRIMARY KEY (task_id, card_id),
			FOREIGN KEY (task_id) REFERENCES study_queue(id) ON DELETE CASCADE,
			FOREIGN KEY (card_id) REFERENCES fsrs_cards(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS manual_flashcards (
		id TEXT PRIMARY KEY,
		notebook_id TEXT NOT NULL,
		prompt TEXT NOT NULL,
		answer TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (notebook_id) REFERENCES notebooks(id) ON DELETE CASCADE
)`,
		`CREATE TABLE IF NOT EXISTS analytics_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type TEXT NOT NULL,
			file_hash TEXT DEFAULT '',
			page_number INTEGER DEFAULT 0,
			metadata TEXT DEFAULT '',
			synced BOOLEAN DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Gamification tables
		`CREATE TABLE IF NOT EXISTS user_gamification (
			user_id INTEGER PRIMARY KEY CHECK (user_id = 1),
			total_xp INTEGER NOT NULL DEFAULT 0,
			coins INTEGER NOT NULL DEFAULT 0,
			current_title TEXT NOT NULL DEFAULT 'The Apprentice',
			streak_freezes_owned INTEGER NOT NULL DEFAULT 1,
			last_freeze_purchased_at INTEGER NOT NULL DEFAULT 0,
			frozen_dates_json TEXT NOT NULL DEFAULT '[]',
			unlocked_cosmetics_json TEXT NOT NULL DEFAULT '["dark-gruvbox", "light-classic"]',
			stats_json TEXT NOT NULL DEFAULT '{}',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS pending_loot_boxes (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			box_tier TEXT NOT NULL,
			reward_type TEXT NOT NULL,
			reward_amount INTEGER NOT NULL,
			opened BOOLEAN NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	// Execute all table creation statements
	for _, stmt := range schema {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Apply migrations, deduplication, and seed defaults before index creation
	// so that all columns referenced by indexes exist on older upgraded databases.
	if err := RunMigrations(tx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create indexes
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_fsrs_cards_topic_prompt ON fsrs_cards(topic_id, prompt)`,
		`CREATE INDEX IF NOT EXISTS idx_fsrs_review_log_activity_ref_reviewed_at ON fsrs_review_log(activity_type, reference_id, reviewed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_fsrs_review_log_topic_reviewed_at ON fsrs_review_log(topic_id, reviewed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_fsrs_cards_suspended_due_at ON fsrs_cards(suspended, due_at)`,
		`CREATE INDEX IF NOT EXISTS idx_written_questions_topic_created_at ON written_questions(topic_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_topic_page_num ON chunks(topic_id, page_num)`,
		`CREATE INDEX IF NOT EXISTS idx_topics_status_updated_at ON topics(status, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_topics_status_created_at ON topics(status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_study_queue_status_priority_created ON study_queue(status, priority, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_study_queue_notebook_status ON study_queue(notebook_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_review_task_cards_task_status ON review_task_cards(task_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_quiz_attempts_task_completed_at ON quiz_attempts(task_id, completed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_reread_attempts_last_attempt_at ON reread_attempts(last_attempt_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_manual_flashcards_notebook_id ON manual_flashcards(notebook_id)`,
		`CREATE INDEX IF NOT EXISTS idx_analytics_events_synced ON analytics_events(synced)`,
		`CREATE INDEX IF NOT EXISTS idx_pending_loot_boxes_opened ON pending_loot_boxes(opened, created_at DESC)`,
	}

	for _, stmt := range indexes {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
