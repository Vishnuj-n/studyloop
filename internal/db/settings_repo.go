package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"ai-tutor/internal/models"
	"ai-tutor/internal/utils"

	"github.com/google/uuid"
)

// GetActiveProfileID retrieves the active profile ID from settings.
func (r *Repository) GetActiveProfileID() (string, error) {
	var activeProfileID sql.NullString
	err := r.db.QueryRow(`SELECT COALESCE(active_profile_id, '') FROM user_settings WHERE id = 1`).Scan(&activeProfileID)
	if err != nil {
		return "", err
	}
	return activeProfileID.String, nil
}

// GetDefaultProfileID retrieves the oldest profile ID.
func (r *Repository) GetDefaultProfileID() (string, error) {
	var id string
	err := r.db.QueryRow(`
		SELECT id FROM study_profiles ORDER BY created_at ASC LIMIT 1
	`).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}

// GetRAGEnabled returns the status of RAG flag.
func (r *Repository) GetRAGEnabled() (bool, error) {
	var enabled bool
	err := r.db.QueryRow(`
		SELECT COALESCE(rag_enabled, 0)
		FROM user_settings
		WHERE id = 1
	`).Scan(&enabled)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return enabled, err
}

// GetUserSettings returns the full settings config.
func (r *Repository) GetUserSettings() (*models.UserSettings, error) {
	var s models.UserSettings
	var activeProfileID sql.NullString
	err := r.db.QueryRow(`
		SELECT max_flashcards_per_session, COALESCE(study_start_time, '17:00'), COALESCE(study_end_time, '18:00'), COALESCE(study_slots_json, '[]'), COALESCE(reminders_enabled, 1), COALESCE(show_reward_notifications, 1), COALESCE(active_profile_id, ''), skip_to_reading_active, COALESCE(cloud_sync_url, ''), COALESCE(cloud_api_token, ''), COALESCE(theme, 'dark-gruvbox'), COALESCE(rag_enabled, 0), COALESCE(rag_notebook_chapter, 1), COALESCE(rag_entire_notebook, 1), COALESCE(rag_queue_study, 1), COALESCE(default_remedial_strategy, 'FAST'), COALESCE(classroom_code, ''), COALESCE(student_username, ''), COALESCE(last_synced_at, 0), COALESCE(analytics_enabled, 0), COALESCE(anonymous_user_id, ''), COALESCE(target_session_words, 3000), COALESCE(min_session_words, 0), COALESCE(max_active_notebooks, 4), COALESCE(quiz_question_count, 8), COALESCE(quiz_passing_score, 70), COALESCE(tutor_style, 'socratic'), COALESCE(llm_prompt_logging, 0)
		FROM user_settings
		WHERE id = 1
	`).Scan(&s.MaxFlashcardsPerSession, &s.StudyStartTime, &s.StudyEndTime, &s.StudySlotsJSON, &s.RemindersEnabled, &s.ShowRewardNotifications, &activeProfileID, &s.SkipToReadingActive, &s.CloudSyncURL, &s.CloudAPIToken, &s.Theme, &s.RAGEnabled, &s.RAGNotebookChapter, &s.RAGEntireNotebook, &s.RAGQueueStudy, &s.DefaultRemedialStrategy, &s.ClassroomCode, &s.StudentUsername, &s.LastSyncedAt, &s.AnalyticsEnabled, &s.AnonymousUserID, &s.TargetSessionWords, &s.MinSessionWords, &s.MaxActiveNotebooks, &s.QuizQuestionCount, &s.QuizPassingScore, &s.TutorStyle, &s.LLMPromptLogging)
	if err == sql.ErrNoRows {
		s = models.UserSettings{
			MaxFlashcardsPerSession: 30,
			StudyStartTime:          "17:00",
			StudyEndTime:            "18:00",
			StudySlotsJSON:          "[]",
			RemindersEnabled:        true,
			ShowRewardNotifications: true,
			Theme:                   "dark-gruvbox",
			RAGEnabled:              false,
			RAGNotebookChapter:      true,
			RAGEntireNotebook:       true,
			RAGQueueStudy:           true,
			DefaultRemedialStrategy: "FAST",
			AnalyticsEnabled:        false,
			AnonymousUserID:         "",
			TargetSessionWords:      3000,
			MinSessionWords:         0,
			MaxActiveNotebooks:      4,
			QuizQuestionCount:       8,
			QuizPassingScore:        70,
			TutorStyle:              "socratic",
		}
	} else if err != nil {
		return nil, err
	} else {
		if activeProfileID.Valid {
			s.ActiveProfileID = activeProfileID.String
		}
	}
	if s.StudySlotsJSON == "" {
		s.StudySlotsJSON = "[]"
	}
	if s.TargetSessionWords <= 0 {
		s.TargetSessionWords = 3000
		if _, updateErr := r.db.Exec(`UPDATE user_settings SET target_session_words = ? WHERE id = 1`, s.TargetSessionWords); updateErr != nil {
			utils.Warnf("failed to persist default target_session_words: %v", updateErr)
		}
	}
	if s.MaxActiveNotebooks < 0 {
		s.MaxActiveNotebooks = 4
	}
	if s.QuizQuestionCount <= 0 {
		quizCount := 8
		s.QuizQuestionCount = quizCount
	}
	if s.QuizPassingScore <= 0 {
		s.QuizPassingScore = 70
	}
	if s.TutorStyle == "" {
		s.TutorStyle = "socratic"
	}

	// Auto-generate anonymous_user_id if not set yet
	if s.AnonymousUserID == "" {
		newUUID := uuid.New().String()
		_, updateErr := r.db.Exec(`UPDATE user_settings SET anonymous_user_id = ? WHERE id = 1`, newUUID)
		if updateErr != nil {
			utils.Warnf("failed to generate and save anonymous_user_id: %v", updateErr)
		} else {
			s.AnonymousUserID = newUUID
		}
	}

	// Verify if active profile exists
	if s.ActiveProfileID != "" {
		var exists bool
		if err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM study_profiles WHERE id = ?)`, s.ActiveProfileID).Scan(&exists); err != nil {
			return nil, fmt.Errorf("failed to check active profile existence: %w", err)
		}
		if !exists {
			s.ActiveProfileID = ""
		}
	}

	// Read active profile's settings & cloud credentials if available
	if s.ActiveProfileID != "" {
		var profClassroom, profUser, profToken, profTheme, profStrategy, profTutorStyle sql.NullString
		var profTargetWords, profMinWords, profMaxFlashcards, profMaxActive sql.NullInt64
		var profQuizCount, profQuizScore sql.NullInt64
		var profSkipToReading sql.NullBool

		err := r.db.QueryRow(`
			SELECT COALESCE(classroom_code, ''), COALESCE(student_username, ''), COALESCE(cloud_api_token, ''),
			       theme, default_remedial_strategy, tutor_style,
			       target_session_words, min_session_words, max_flashcards_per_session, max_active_notebooks,
			       quiz_question_count, quiz_passing_score, skip_to_reading_active
			FROM study_profiles
			WHERE id = ?
		`, s.ActiveProfileID).Scan(
			&profClassroom, &profUser, &profToken,
			&profTheme, &profStrategy, &profTutorStyle,
			&profTargetWords, &profMinWords, &profMaxFlashcards, &profMaxActive,
			&profQuizCount, &profQuizScore, &profSkipToReading,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to query active profile credentials and settings: %w", err)
		}
		s.ClassroomCode = profClassroom.String
		s.StudentUsername = profUser.String
		s.CloudAPIToken = profToken.String
		if profTheme.Valid && strings.TrimSpace(profTheme.String) != "" {
			s.Theme = profTheme.String
		}
		if profStrategy.Valid && strings.TrimSpace(profStrategy.String) != "" {
			s.DefaultRemedialStrategy = profStrategy.String
		}
		if profTutorStyle.Valid && strings.TrimSpace(profTutorStyle.String) != "" {
			s.TutorStyle = profTutorStyle.String
		}
		if profTargetWords.Valid && profTargetWords.Int64 > 0 {
			s.TargetSessionWords = int(profTargetWords.Int64)
		}
		if profMinWords.Valid && profMinWords.Int64 >= 0 {
			s.MinSessionWords = int(profMinWords.Int64)
		}
		if profMaxFlashcards.Valid && profMaxFlashcards.Int64 > 0 {
			s.MaxFlashcardsPerSession = int(profMaxFlashcards.Int64)
		}
		if profMaxActive.Valid && profMaxActive.Int64 >= 0 {
			s.MaxActiveNotebooks = int(profMaxActive.Int64)
		}
		if profQuizCount.Valid && profQuizCount.Int64 > 0 {
			s.QuizQuestionCount = int(profQuizCount.Int64)
		}
		if profQuizScore.Valid && profQuizScore.Int64 > 0 {
			s.QuizPassingScore = int(profQuizScore.Int64)
		}
		if profSkipToReading.Valid {
			s.SkipToReadingActive = profSkipToReading.Bool
		}
	}

	return &s, nil
}

// UpdateUserSettings updates the user settings.
func (r *Repository) UpdateUserSettings(s models.UserSettings) error {
	var activeProfileID interface{} = nil
	if s.ActiveProfileID != "" {
		activeProfileID = s.ActiveProfileID
	}
	theme := s.Theme
	if theme == "" {
		theme = "dark-gruvbox"
	}
	strategy := s.DefaultRemedialStrategy
	if strategy == "" {
		strategy = "FAST"
	}
	targetWords := s.TargetSessionWords
	if targetWords <= 0 {
		targetWords = 3000
	}
	minWords := s.MinSessionWords
	if minWords < 0 {
		minWords = 0
	}
	maxActive := s.MaxActiveNotebooks
	if maxActive < 0 {
		maxActive = 4
	}
	quizCount := s.QuizQuestionCount
	if quizCount <= 0 {
		quizCount = 8
	}
	passingScore := s.QuizPassingScore
	if passingScore <= 0 {
		passingScore = 70
	}
	tutorStyle := s.TutorStyle
	if tutorStyle == "" {
		tutorStyle = "socratic"
	}
	studySlots := strings.TrimSpace(s.StudySlotsJSON)
	if studySlots == "" {
		studySlots = "[]"
	}
	_, err := r.db.Exec(`
		INSERT INTO user_settings (id, max_flashcards_per_session, study_start_time, study_end_time, study_slots_json, reminders_enabled, show_reward_notifications, active_profile_id, skip_to_reading_active, cloud_sync_url, cloud_api_token, theme, rag_enabled, rag_notebook_chapter, rag_entire_notebook, rag_queue_study, default_remedial_strategy, classroom_code, student_username, analytics_enabled, anonymous_user_id, target_session_words, min_session_words, max_active_notebooks, quiz_question_count, quiz_passing_score, tutor_style, llm_prompt_logging)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			max_flashcards_per_session = excluded.max_flashcards_per_session,
			study_start_time = excluded.study_start_time,
			study_end_time = excluded.study_end_time,
			study_slots_json = excluded.study_slots_json,
			reminders_enabled = excluded.reminders_enabled,
			show_reward_notifications = excluded.show_reward_notifications,
			active_profile_id = excluded.active_profile_id,
			skip_to_reading_active = excluded.skip_to_reading_active,
			cloud_sync_url = excluded.cloud_sync_url,
			cloud_api_token = excluded.cloud_api_token,
			theme = excluded.theme,
			rag_enabled = excluded.rag_enabled,
			rag_notebook_chapter = excluded.rag_notebook_chapter,
			rag_entire_notebook = excluded.rag_entire_notebook,
			rag_queue_study = excluded.rag_queue_study,
			default_remedial_strategy = excluded.default_remedial_strategy,
			classroom_code = excluded.classroom_code,
			student_username = CASE WHEN excluded.student_username != '' THEN excluded.student_username ELSE user_settings.student_username END,
			analytics_enabled = excluded.analytics_enabled,
			anonymous_user_id = CASE WHEN excluded.anonymous_user_id != '' THEN excluded.anonymous_user_id ELSE user_settings.anonymous_user_id END,
			target_session_words = excluded.target_session_words,
			min_session_words = excluded.min_session_words,
			max_active_notebooks = excluded.max_active_notebooks,
			quiz_question_count = excluded.quiz_question_count,
			quiz_passing_score = excluded.quiz_passing_score,
			tutor_style = excluded.tutor_style,
			llm_prompt_logging = excluded.llm_prompt_logging,
			updated_at = CURRENT_TIMESTAMP
	`, s.MaxFlashcardsPerSession, s.StudyStartTime, s.StudyEndTime, studySlots, s.RemindersEnabled, s.ShowRewardNotifications, activeProfileID, s.SkipToReadingActive, s.CloudSyncURL, s.CloudAPIToken, theme, s.RAGEnabled, s.RAGNotebookChapter, s.RAGEntireNotebook, s.RAGQueueStudy, strategy, s.ClassroomCode, s.StudentUsername, s.AnalyticsEnabled, s.AnonymousUserID, targetWords, minWords, maxActive, quizCount, passingScore, tutorStyle, s.LLMPromptLogging)
	if err != nil {
		return err
	}

	return nil
}

// SetLastSyncedAt updates the last_synced_at timestamp after a successful cloud sync.
// ponytail: dedicated single-column UPDATE — keeps sync state out of the full UpdateUserSettings call.
func (r *Repository) SetLastSyncedAt(ts int64) error {
	_, err := r.db.Exec(`
		UPDATE user_settings SET last_synced_at = ? WHERE id = 1
	`, ts)
	return err
}

func (r *Repository) GetLLMSettings() (*models.LLMSettings, error) {
	rows, err := r.db.Query(`
		SELECT tier, provider, base_url, model, timeout_ms, max_input_tokens, max_output_tokens, api_key_source, COALESCE(has_api_key, 0)
		FROM llm_settings
		WHERE tier IN ('fast', 'heavy')
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			log.Printf("warning: failed to close LLM settings rows: %v", closeErr)
		}
	}()

	settings := defaultLLMSettings()
	seenFast := false
	seenHeavy := false
	for rows.Next() {
		var tier models.LLMTierSettings
		if err := rows.Scan(&tier.Tier, &tier.Provider, &tier.BaseURL, &tier.Model, &tier.TimeoutMs, &tier.MaxInputTokens, &tier.MaxOutputTokens, &tier.APIKeySource, &tier.HasAPIKey); err != nil {
			return nil, err
		}
		tier = normalizeLLMTierSettings(tier)
		switch tier.Tier {
		case llmTierFast:
			settings.Fast = tier
			seenFast = true
		case llmTierHeavy:
			settings.Heavy = tier
			seenHeavy = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !seenFast {
		settings.Fast = defaultLLMTier(llmTierFast)
	}
	if !seenHeavy {
		settings.Heavy = defaultLLMTier(llmTierHeavy)
	}
	settings.UseSameForHeavy = sameLLMConfig(settings.Fast, settings.Heavy)
	return &settings, nil
}

func (r *Repository) UpdateLLMSettings(settings models.LLMSettings) error {
	fast := normalizeLLMTierSettings(settings.Fast)
	fast.Tier = llmTierFast
	heavy := normalizeLLMTierSettings(settings.Heavy)
	heavy.Tier = llmTierHeavy
	if settings.UseSameForHeavy {
		heavy.Provider = fast.Provider
		heavy.BaseURL = fast.BaseURL
		heavy.Model = fast.Model
		heavy.TimeoutMs = fast.TimeoutMs
		heavy.MaxInputTokens = fast.MaxInputTokens
		heavy.MaxOutputTokens = fast.MaxOutputTokens
		heavy.APIKeySource = fast.APIKeySource
		heavy.HasAPIKey = fast.HasAPIKey
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, tier := range []models.LLMTierSettings{fast, heavy} {
		if _, err := tx.Exec(`
			INSERT INTO llm_settings (tier, provider, base_url, model, timeout_ms, max_input_tokens, max_output_tokens, api_key_source, has_api_key)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(tier) DO UPDATE SET
				provider = excluded.provider,
				base_url = excluded.base_url,
				model = excluded.model,
				timeout_ms = excluded.timeout_ms,
				max_input_tokens = excluded.max_input_tokens,
				max_output_tokens = excluded.max_output_tokens,
				api_key_source = excluded.api_key_source,
				has_api_key = excluded.has_api_key,
				updated_at = CURRENT_TIMESTAMP
		`, tier.Tier, tier.Provider, tier.BaseURL, tier.Model, tier.TimeoutMs, tier.MaxInputTokens, tier.MaxOutputTokens, tier.APIKeySource, tier.HasAPIKey); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) MarkLLMKeyStored(tier string, stored bool) error {
	tier = normalizeLLMTier(tier)
	if tier == "" {
		return fmt.Errorf("llm tier is required")
	}
	_, err := r.db.Exec(`
		UPDATE llm_settings
		SET has_api_key = ?, api_key_source = 'keyring', updated_at = CURRENT_TIMESTAMP
		WHERE tier = ?
	`, stored, tier)
	return err
}

func defaultLLMSettings() models.LLMSettings {
	return models.LLMSettings{
		UseSameForHeavy: true,
		Fast:            defaultLLMTier(llmTierFast),
		Heavy:           defaultLLMTier(llmTierHeavy),
	}
}

func defaultLLMTier(tier string) models.LLMTierSettings {
	timeout := 60000
	if tier == llmTierHeavy {
		timeout = 90000
	}
	return models.LLMTierSettings{
		Tier:            tier,
		Provider:        "groq",
		BaseURL:         "https://api.groq.com/openai/v1",
		Model:           "openai/gpt-oss-120b",
		TimeoutMs:       timeout,
		MaxInputTokens:  4000,
		MaxOutputTokens: 1000,
		APIKeySource:    "keyring",
		HasAPIKey:       false,
	}
}

func normalizeLLMTierSettings(tier models.LLMTierSettings) models.LLMTierSettings {
	tier.Tier = normalizeLLMTier(tier.Tier)
	if tier.Tier == "" {
		tier.Tier = llmTierFast
	}
	tier.Provider = strings.TrimSpace(strings.ToLower(tier.Provider))
	if tier.Provider == "" {
		tier.Provider = "custom"
	}
	tier.BaseURL = strings.TrimSpace(tier.BaseURL)
	if tier.BaseURL == "" {
		tier.BaseURL = defaultBaseURLForProvider(tier.Provider)
	}
	tier.Model = strings.TrimSpace(tier.Model)
	if tier.Model == "" {
		tier.Model = defaultModelForProvider(tier.Provider)
	}
	if tier.TimeoutMs <= 0 {
		tier.TimeoutMs = 30000
	}
	if tier.MaxInputTokens <= 0 {
		tier.MaxInputTokens = 4000
	}
	if tier.MaxOutputTokens <= 0 {
		tier.MaxOutputTokens = 1000
	}
	tier.APIKeySource = strings.TrimSpace(strings.ToLower(tier.APIKeySource))
	if tier.APIKeySource == "" {
		tier.APIKeySource = "keyring"
	}
	return tier
}

func normalizeLLMTier(tier string) string {
	tier = strings.TrimSpace(strings.ToLower(tier))
	switch tier {
	case llmTierFast, llmTierHeavy:
		return tier
	default:
		return ""
	}
}

func defaultBaseURLForProvider(provider string) string {
	switch provider {
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta/openai"
	case "groq":
		return "https://api.groq.com/openai/v1"
	case "openai":
		return "https://api.openai.com/v1"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	default:
		return ""
	}
}

func defaultModelForProvider(provider string) string {
	switch provider {
	case "gemini":
		return "gemini-flash-lite-latest"
	case "groq":
		return "openai/gpt-oss-120b"
	case "openai":
		return "gpt-4.1-mini"
	case "openrouter":
		return "openai/gpt-4.1-mini"
	default:
		return ""
	}
}

func sameLLMConfig(a, b models.LLMTierSettings) bool {
	return strings.EqualFold(a.Provider, b.Provider) &&
		strings.TrimSpace(a.BaseURL) == strings.TrimSpace(b.BaseURL) &&
		strings.TrimSpace(a.Model) == strings.TrimSpace(b.Model) &&
		a.TimeoutMs == b.TimeoutMs &&
		a.MaxInputTokens == b.MaxInputTokens &&
		a.MaxOutputTokens == b.MaxOutputTokens &&
		strings.EqualFold(a.APIKeySource, b.APIKeySource) &&
		a.HasAPIKey == b.HasAPIKey
}

// GetProfiles retrieves all study profiles.
func (r *Repository) GetProfiles() ([]models.StudyProfile, error) {
	rows, err := r.db.Query(`
		SELECT id, name, deadline_at, created_at, COALESCE(classroom_code, ''), COALESCE(student_username, ''), COALESCE(cloud_api_token, ''),
		       COALESCE(pomo_duration_sec, 1500), COALESCE(pomo_break_sec, 300), COALESCE(pomo_music_path, ''), COALESCE(pomo_shuffle, 0),
		       target_session_words, min_session_words, COALESCE(theme, ''), max_flashcards_per_session, max_active_notebooks,
		       skip_to_reading_active, COALESCE(default_remedial_strategy, ''), quiz_question_count, quiz_passing_score, COALESCE(tutor_style, '')
		FROM study_profiles
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			log.Printf("warning: failed to close profiles rows: %v", closeErr)
		}
	}()

	profiles := make([]models.StudyProfile, 0)
	for rows.Next() {
		var p models.StudyProfile
		var targetWords, minWords, maxCards, maxActive, quizCount, quizScore sql.NullInt64
		var skipRead sql.NullBool
		if err := rows.Scan(
			&p.ID, &p.Name, &p.DeadlineAt, &p.CreatedAt, &p.ClassroomCode, &p.StudentUsername, &p.CloudAPIToken,
			&p.PomoDurationSec, &p.PomoBreakSec, &p.PomoMusicPath, &p.PomoShuffle,
			&targetWords, &minWords, &p.Theme, &maxCards, &maxActive,
			&skipRead, &p.DefaultRemedialStrategy, &quizCount, &quizScore, &p.TutorStyle,
		); err != nil {
			return nil, err
		}
		if targetWords.Valid {
			v := int(targetWords.Int64)
			p.TargetSessionWords = &v
		}
		if minWords.Valid {
			v := int(minWords.Int64)
			p.MinSessionWords = &v
		}
		if maxCards.Valid {
			v := int(maxCards.Int64)
			p.MaxFlashcardsPerSession = &v
		}
		if maxActive.Valid {
			v := int(maxActive.Int64)
			p.MaxActiveNotebooks = &v
		}
		if skipRead.Valid {
			v := skipRead.Bool
			p.SkipToReadingActive = &v
		}
		if quizCount.Valid {
			v := int(quizCount.Int64)
			p.QuizQuestionCount = &v
		}
		if quizScore.Valid {
			v := int(quizScore.Int64)
			p.QuizPassingScore = &v
		}
		profiles = append(profiles, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return profiles, nil
}

// GetProfileByID retrieves a specific profile by ID.
func (r *Repository) GetProfileByID(id string) (*models.StudyProfile, error) {
	var p models.StudyProfile
	var targetWords, minWords, maxCards, maxActive, quizCount, quizScore sql.NullInt64
	var skipRead sql.NullBool
	err := r.db.QueryRow(`
		SELECT id, name, deadline_at, created_at, COALESCE(classroom_code, ''), COALESCE(student_username, ''), COALESCE(cloud_api_token, ''),
		       COALESCE(pomo_duration_sec, 1500), COALESCE(pomo_break_sec, 300), COALESCE(pomo_music_path, ''), COALESCE(pomo_shuffle, 0),
		       target_session_words, min_session_words, COALESCE(theme, ''), max_flashcards_per_session, max_active_notebooks,
		       skip_to_reading_active, COALESCE(default_remedial_strategy, ''), quiz_question_count, quiz_passing_score, COALESCE(tutor_style, '')
		FROM study_profiles
		WHERE id = ?
	`, id).Scan(
		&p.ID, &p.Name, &p.DeadlineAt, &p.CreatedAt, &p.ClassroomCode, &p.StudentUsername, &p.CloudAPIToken,
		&p.PomoDurationSec, &p.PomoBreakSec, &p.PomoMusicPath, &p.PomoShuffle,
		&targetWords, &minWords, &p.Theme, &maxCards, &maxActive,
		&skipRead, &p.DefaultRemedialStrategy, &quizCount, &quizScore, &p.TutorStyle,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if targetWords.Valid {
		v := int(targetWords.Int64)
		p.TargetSessionWords = &v
	}
	if minWords.Valid {
		v := int(minWords.Int64)
		p.MinSessionWords = &v
	}
	if maxCards.Valid {
		v := int(maxCards.Int64)
		p.MaxFlashcardsPerSession = &v
	}
	if maxActive.Valid {
		v := int(maxActive.Int64)
		p.MaxActiveNotebooks = &v
	}
	if skipRead.Valid {
		v := skipRead.Bool
		p.SkipToReadingActive = &v
	}
	if quizCount.Valid {
		v := int(quizCount.Int64)
		p.QuizQuestionCount = &v
	}
	if quizScore.Valid {
		v := int(quizScore.Int64)
		p.QuizPassingScore = &v
	}
	return &p, nil
}

// CreateProfile creates a new study profile.
func (r *Repository) CreateProfile(p models.StudyProfile) error {
	dur := p.PomoDurationSec
	if dur <= 0 {
		dur = 1500
	}
	brk := p.PomoBreakSec
	if brk < 0 {
		brk = 300
	}
	_, err := r.db.Exec(`
		INSERT INTO study_profiles (
			id, name, deadline_at, classroom_code, student_username, cloud_api_token,
			pomo_duration_sec, pomo_break_sec, pomo_music_path, pomo_shuffle,
			target_session_words, min_session_words, theme, max_flashcards_per_session,
			max_active_notebooks, skip_to_reading_active, default_remedial_strategy,
			quiz_question_count, quiz_passing_score, tutor_style
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Name, p.DeadlineAt, p.ClassroomCode, p.StudentUsername, p.CloudAPIToken,
		dur, brk, p.PomoMusicPath, p.PomoShuffle,
		p.TargetSessionWords, p.MinSessionWords, p.Theme, p.MaxFlashcardsPerSession,
		p.MaxActiveNotebooks, p.SkipToReadingActive, p.DefaultRemedialStrategy,
		p.QuizQuestionCount, p.QuizPassingScore, p.TutorStyle)
	return err
}

// UpdateProfile updates an existing profile metadata and settings.
func (r *Repository) UpdateProfile(p models.StudyProfile) error {
	_, err := r.db.Exec(`
		UPDATE study_profiles
		SET name = ?,
		    deadline_at = ?,
		    target_session_words = ?,
		    min_session_words = ?,
		    theme = ?,
		    max_flashcards_per_session = ?,
		    max_active_notebooks = ?,
		    skip_to_reading_active = ?,
		    default_remedial_strategy = ?,
		    quiz_question_count = ?,
		    quiz_passing_score = ?,
		    tutor_style = ?
		WHERE id = ?
	`, p.Name, p.DeadlineAt, p.TargetSessionWords, p.MinSessionWords, p.Theme, p.MaxFlashcardsPerSession,
		p.MaxActiveNotebooks, p.SkipToReadingActive, p.DefaultRemedialStrategy,
		p.QuizQuestionCount, p.QuizPassingScore, p.TutorStyle, p.ID)
	return err
}

// UpdateProfilePomoSettings updates the Pomodoro focus and audio settings for a profile.
func (r *Repository) UpdateProfilePomoSettings(profileID string, durationSec, breakSec int, musicPath string, shuffle bool) error {
	if durationSec <= 0 {
		durationSec = 1500
	}
	if breakSec < 0 {
		breakSec = 300
	}
	_, err := r.db.Exec(`
		UPDATE study_profiles
		SET pomo_duration_sec = ?, pomo_break_sec = ?, pomo_music_path = ?, pomo_shuffle = ?
		WHERE id = ?
	`, durationSec, breakSec, musicPath, shuffle, profileID)
	return err
}

// UpdateProfileCloudCredentials updates the cloud credentials for a specific profile.
func (r *Repository) UpdateProfileCloudCredentials(profileID, classroomCode, studentUsername, cloudAPIToken string) error {
	_, err := r.db.Exec(`
		UPDATE study_profiles
		SET classroom_code = ?, student_username = ?, cloud_api_token = ?
		WHERE id = ?
	`, classroomCode, studentUsername, cloudAPIToken, profileID)
	return err
}

// DeleteProfile deletes a profile atomically.
func (r *Repository) DeleteProfile(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("DeleteProfile: failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`UPDATE notebooks SET profile_id = NULL WHERE profile_id = ?`, id); err != nil {
		return fmt.Errorf("DeleteProfile: failed to unlink notebooks: %w", err)
	}
	if _, err := tx.Exec(`UPDATE user_settings SET active_profile_id = NULL WHERE active_profile_id = ?`, id); err != nil {
		return fmt.Errorf("DeleteProfile: failed to unlink user_settings: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM study_profiles WHERE id = ?`, id); err != nil {
		return fmt.Errorf("DeleteProfile: failed to delete profile: %w", err)
	}
	return tx.Commit()
}

func (r *Repository) GetRemedialStrategy() (string, error) {
	var strategy string
	err := r.db.QueryRow(
		`SELECT COALESCE(default_remedial_strategy, 'FAST') FROM user_settings WHERE id = 1`,
	).Scan(&strategy)
	if err == sql.ErrNoRows {
		return "FAST", nil
	}
	if err != nil {
		return "", err
	}
	if strategy == "" {
		return "FAST", nil
	}
	return strategy, nil
}

// GetExtensionConfig retrieves the serialized JSON configuration for extensions from user_settings.
func (r *Repository) GetExtensionConfig() (string, error) {
	var config string
	err := r.db.QueryRow(`SELECT COALESCE(extension_settings, '{}') FROM user_settings WHERE id = 1`).Scan(&config)
	if err == sql.ErrNoRows {
		return "{}", nil
	}
	if err != nil {
		return "{}", err
	}
	if strings.TrimSpace(config) == "" {
		return "{}", nil
	}
	return config, nil
}

// SaveExtensionConfig persists the serialized JSON configuration for extensions to user_settings.
func (r *Repository) SaveExtensionConfig(configJSON string) error {
	trimmed := strings.TrimSpace(configJSON)
	if trimmed == "" {
		trimmed = "{}"
	}
	_, err := r.db.Exec(`UPDATE user_settings SET extension_settings = ? WHERE id = 1`, trimmed)
	return err
}

// GetLLMPromptLogging returns the persisted LLM prompt logging boolean from user_settings.
func (r *Repository) GetLLMPromptLogging() (bool, error) {
	var enabled bool
	err := r.db.QueryRow(`SELECT COALESCE(llm_prompt_logging, 0) FROM user_settings WHERE id = 1`).Scan(&enabled)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return enabled, err
}

// SetLLMPromptLogging persists the LLM prompt logging toggle state into user_settings.
func (r *Repository) SetLLMPromptLogging(enabled bool) error {
	_, err := r.db.Exec(`UPDATE user_settings SET llm_prompt_logging = ? WHERE id = 1`, enabled)
	return err
}

