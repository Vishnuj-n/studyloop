package app

import (
	"encoding/json"
	"strings"

	"ai-tutor/internal/models"
)

// AnalyzeQuizFailure generates an LLM-based root cause analysis for missed quiz questions.
func (a *App) AnalyzeQuizFailure(notebookID, topicID string, startPage, endPage int, failedQuestionsJSON string, includeBookText bool) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}

	a.aiMutex.Lock()
	if !a.aiReady {
		reason := a.aiInitError
		a.aiMutex.Unlock()
		if reason == "" {
			reason = errLocalAIRuntimeNotReady
		}
		return map[string]interface{}{"error": "AI diagnostic unavailable: " + reason}
	}
	if a.studyService == nil {
		a.aiMutex.Unlock()
		return map[string]interface{}{"error": errStudyServiceNotInitialized}
	}
	svc := a.studyService
	a.aiMutex.Unlock()

	var failedQuestions []models.FailedQuestionDetail
	if strings.TrimSpace(failedQuestionsJSON) != "" {
		if err := json.Unmarshal([]byte(failedQuestionsJSON), &failedQuestions); err != nil {
			return map[string]interface{}{"error": "failed to parse failed questions JSON: " + err.Error()}
		}
	}

	bookContent := ""
	if includeBookText && notebookID != "" && startPage > 0 && endPage >= startPage {
		chunks, err := repo.GetChunksForNotebookPageRange(notebookID, startPage, endPage)
		if err == nil && len(chunks) > 0 {
			var sb strings.Builder
			for _, c := range chunks {
				text := strings.TrimSpace(c.Text)
				if text != "" {
					sb.WriteString(text)
					sb.WriteString("\n\n")
				}
			}
			bookContent = sb.String()
		}
	}

	diagnostic, err := svc.AnalyzeQuizFailure(bookContent, failedQuestions)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"result": diagnostic,
	}
}
