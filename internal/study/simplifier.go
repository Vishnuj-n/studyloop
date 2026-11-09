package study

import (
	"context"
	"fmt"
	"strings"

	"ai-tutor/internal/embeddings"
	"ai-tutor/internal/utils"
)

// getSimplifierLevelDirective returns the prompt instruction for a given style level.
func getSimplifierLevelDirective(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "eli5":
		return "ELI5 (Explain Like I'm 5):\nUse extremely simple everyday language, clear fun analogies, and short sentences. Strictly avoid complex technical jargon. Translate every technical term into simple real-world concepts a 5-year-old can easily understand."
	case "very_simple", "eli10":
		return "Very Simple:\nUse everyday language and short sentences. Replace complex jargon with simple everyday concepts and intuitive analogies wherever possible."
	case "academic":
		return "Academic:\nUse precise academic and technical language. Preserve terminology, definitions, distinctions, and technical details. Simplify only unnecessarily complicated wording."
	case "summary", "bullet":
		return "Summary:\nFocus on the essential ideas, arguments, definitions, and conclusions. Be substantially shorter than the source while preserving the information needed to understand the topic."
	default: // "simple", "eli15", or default
		return "Simple:\nUse clear, straightforward language. Explain difficult terminology simply, while retaining essential core vocabulary."
	}
}

// buildSimplifierPrompt formats the prompt given a template, material content, and comprehension style.
// Preserved for backward compatibility in tests.
func buildSimplifierPrompt(promptTemplate, content, styleDirective string) string {
	if styleDirective == "" {
		styleDirective = getSimplifierLevelDirective("simple")
	}

	if promptTemplate != "" {
		tmpl := strings.ReplaceAll(promptTemplate, "{{COMPREHENSION_LEVEL}}", styleDirective)
		tmpl = strings.ReplaceAll(tmpl, "{{style}}", styleDirective)
		if strings.Contains(tmpl, "{{content}}") {
			return strings.ReplaceAll(tmpl, "{{content}}", content)
		}
		if strings.Contains(tmpl, "{CONTENT}") {
			tmpl = strings.ReplaceAll(tmpl, "{CONTENT}", content)
		}
		if strings.Contains(tmpl, "{DIFFICULTY}") {
			tmpl = strings.ReplaceAll(tmpl, "{DIFFICULTY}", styleDirective)
		}
		return tmpl + "\n\nReading Material:\n\"\"\"\n" + content + "\n\"\"\"\n"
	}

	return renderPlaceholders(DefaultTemplates[PromptCategorySimplify], map[string]string{
		"{CONTENT}":    content,
		"{DIFFICULTY}": styleDirective,
	})
}

// SimplifyReadingContent takes dense text content and simplifies it using PromptService and the preferred language.
func (s *StudyService) SimplifyReadingContent(ctx context.Context, content string, level ...string) (string, error) {
	// ponytail: select LLM tier dynamically based on reading content length
	llm, tier := s.selectLLM(content)
	if llm == nil {
		return "", fmt.Errorf("AI provider not available. Please check your API key in Settings")
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("content cannot be empty")
	}

	styleLevel := "eli15"
	if len(level) > 0 && strings.TrimSpace(level[0]) != "" {
		styleLevel = strings.TrimSpace(level[0])
	}
	styleDirective := getSimplifierLevelDirective(styleLevel)

	// Build template prompt to compute context budget
	limits := llm.GetLimits()
	var templateText string
	var err error
	if s.promptService != nil {
		templateText, err = s.promptService.BuildPrompt(PromptCategorySimplify, map[string]string{
			"{CONTENT}":    "",
			"{DIFFICULTY}": styleDirective,
		})
	}
	if err != nil || templateText == "" {
		templateText = buildSimplifierPrompt("", "", styleDirective)
	}

	availableBudget, err := CalculateAvailableContextBudget(limits.MaxInputTokens, templateText)
	if err != nil {
		return "", err
	}

	truncatedContent, err := embeddings.TruncateToTokens(content, availableBudget)
	if err != nil {
		return "", fmt.Errorf("failed to budget content tokens: %w", err)
	}

	var prompt string
	if s.promptService != nil {
		prompt, err = s.promptService.BuildPrompt(PromptCategorySimplify, map[string]string{
			"{CONTENT}":    truncatedContent,
			"{DIFFICULTY}": styleDirective,
		})
	}
	if err != nil || prompt == "" {
		prompt = buildSimplifierPrompt("", truncatedContent, styleDirective)
	}

	simplified, err := llm.GenerateAnswer(prompt)
	if err != nil {
		utils.Warnf("[SIMPLIFY] LLM simplification error: %v", err)
		return "", s.FormatLLMError(err, tier)
	}

	return simplified, nil
}
