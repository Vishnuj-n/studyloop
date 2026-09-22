package db

import (
	"ai-tutor/internal/models"
	"database/sql"
	"fmt"
	"strings"
	"testing"
)

// assertCountEquals asserts that a query returns exactly the expected count
func assertCountEquals(t *testing.T, query string, arg interface{}, want int) {
	t.Helper()

	if testRepo == nil || testRepo.db == nil {
		t.Fatalf("nil db connection")
	}

	var got int
	if err := testRepo.db.QueryRow(query, arg).Scan(&got); err != nil {
		t.Fatalf("query failed (%s): %v", sanitizeWhitespace(query), err)
	}
	if got != want {
		t.Fatalf("unexpected count for query (%s): got=%d want=%d", sanitizeWhitespace(query), got, want)
	}
}

// contains checks if a target string exists in a slice of strings
func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
}

// sanitizeWhitespace normalizes whitespace in a string for consistent error messages
func sanitizeWhitespace(input string) string {
	return strings.Join(strings.Fields(input), " ")
}

// seedTestNotebookWithTopic sets up a standard active, chunked notebook linked to a topic with page bounds
func seedTestNotebookWithTopic(t *testing.T, notebookID, topicID, profileID string, startPage, endPage int) {
	t.Helper()
	if profileID != "" {
		_ = testRepo.CreateProfile(models.StudyProfile{ID: profileID, Name: profileID + " Name"})
	}
	if err := testRepo.EnsureTopic(topicID, topicID+" Title"); err != nil {
		t.Fatalf("EnsureTopic failed: %v", err)
	}
	if startPage > 0 && endPage >= startPage {
		if err := testRepo.UpdateTopicPageBounds(topicID, startPage, endPage); err != nil {
			t.Fatalf("UpdateTopicPageBounds failed: %v", err)
		}
	}
	pageCount := endPage
	if pageCount <= 0 {
		pageCount = 10
	}
	if err := testRepo.CreateNotebook(notebookID, notebookID+" Title", "/tmp/"+notebookID+".pdf", "pdf", topicID, "", pageCount, profileID); err != nil {
		t.Fatalf("CreateNotebook failed: %v", err)
	}
	if err := testRepo.LinkNotebookTopics(notebookID, []string{topicID}); err != nil {
		t.Fatalf("LinkNotebookTopics failed: %v", err)
	}
	if err := testRepo.UpdateNotebookStatus(notebookID, "chunked"); err != nil {
		t.Fatalf("UpdateNotebookStatus failed: %v", err)
	}
	if err := testRepo.UpdateNotebookStudyStatus(notebookID, "active"); err != nil {
		t.Fatalf("UpdateNotebookStudyStatus failed: %v", err)
	}
}

// seedQuizTaskAndAttempt creates a quiz task and its associated quiz attempt record
func seedQuizTaskAndAttempt(t *testing.T, notebookID, topicID, taskID, attemptID string, score int, passed bool, completedAt int64) {
	t.Helper()
	if err := testRepo.InsertStudyTask(models.StudyQueueTask{
		ID:          taskID,
		NotebookID:  notebookID,
		TopicID:     topicID,
		TaskType:    models.StudyTaskTypeQuiz,
		Status:      models.StudyTaskStatusCompleted,
		PayloadJSON: `{"questions":[{"id":"q1","prompt":"P","options":["A","B"],"correct_answer":"A"}],"passing_score":70}`,
	}); err != nil {
		t.Fatalf("InsertStudyTask quiz failed: %v", err)
	}
	if err := testRepo.withTx(func(tx *sql.Tx) error {
		return testRepo.SaveQuizAttemptTx(tx, models.QuizAttemptRecord{
			ID:          attemptID,
			TaskID:      taskID,
			Score:       score,
			Passed:      passed,
			AnswersJSON: `[{"question_id":"q1","selected":"A"}]`,
			Feedback:    "",
			CompletedAt: completedAt,
		})
	}); err != nil {
		t.Fatalf("SaveQuizAttemptTx failed: %v", err)
	}
}

// seedTopicChunks creates chunk rows and links them to notebook_chunks
func seedTopicChunks(t *testing.T, notebookID, topicID string, startPage, endPage, wordsPerPage int) {
	t.Helper()
	for p := startPage; p <= endPage; p++ {
		cID := fmt.Sprintf("chunk-%s-p%d", topicID, p)
		_, err := testRepo.db.Exec(`
			INSERT INTO chunks (id, topic_id, chunk_text, page_num, token_count)
			VALUES (?, ?, ?, ?, ?)
		`, cID, topicID, fmt.Sprintf("Text content for topic %s on page %d with many words", topicID, p), p, wordsPerPage)
		if err != nil {
			t.Fatalf("insert chunk failed: %v", err)
		}
		_, err = testRepo.db.Exec(`
			INSERT INTO notebook_chunks (id, notebook_id, chunk_id, page_num)
			VALUES (?, ?, ?, ?)
		`, cID+"-nc", notebookID, cID, p)
		if err != nil {
			t.Fatalf("insert notebook_chunk failed: %v", err)
		}
	}
}


