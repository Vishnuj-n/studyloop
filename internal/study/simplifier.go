package study

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
		return tmpl + "\n\nReading Material:\n\"\"\"\n" + content + "\n\"\"\"\n"
	}

	return fmt.Sprintf(`You are an AI Text Simplifier.

Transform the provided reading material into clear, well-structured Markdown notes that are easier to understand while preserving the original meaning and core information.

The user's selected comprehension level is:

%s

Adapt the explanation strictly to this level.

## Core Requirements

- Match tone, sentence complexity, and vocabulary strictly to the user's selected comprehension level above.
- For ELI5 / Very Simple levels: Strictly replace technical jargon, heavy formulas, and abstract terminology with simple, everyday real-world analogies. Do NOT keep un-simplified technical terms.
- For Academic / Simple levels: Retain essential technical vocabulary, precise definitions, and formulas when precision requires it.
- Preserve the author's underlying core meaning, main facts, arguments, and conclusions accurately.
- Do not invent facts, examples, or conclusions, and do not add opinions or meta-commentary.
- Remove unnecessary repetition, fluff, and filler.
- Use clean, natural Markdown structure: headings, subheadings, bullet points, and numbered steps.

## Output

Create a clear, well-structured breakdown of the provided material matched to the selected comprehension level.

Start with a short, simple overview of the main topic.

Then organize the explanation into natural sections based on the material.

End with a short **Key Takeaways** section summarizing the most important ideas.

Reading Material:
"""
%s
"""

Return only the markdown response without meta-commentary.`, styleDirective, content)
}

// SimplifyReadingContent takes dense text content and simplifies it using the fast LLM provider,
// loading the prompt template dynamically from extensions/text_simplifier/prompt.md and injecting the comprehension style.
func (s *StudyService) SimplifyReadingContent(ctx context.Context, content string, level ...string) (string, error) {
// ponytail: select LLM tier dynamically based on reading content length
	llm, _ := s.selectLLM(content)
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

	// Attempt to load dynamic prompt template from extensions directory
	promptTemplate := ""
	promptPath := filepath.Join("extensions", "text_simplifier", "prompt.md")
	if data, err := os.ReadFile(promptPath); err == nil {
		promptTemplate = string(data)
	}

	limits := llm.GetLimits()
	templateText := buildSimplifierPrompt(promptTemplate, "", styleDirective)
	availableBudget, err := CalculateAvailableContextBudget(limits.MaxInputTokens, templateText)
	if err != nil {
		return "", err
	}

	truncatedContent, err := embeddings.TruncateToTokens(content, availableBudget)
	if err != nil {
		return "", fmt.Errorf("failed to budget content tokens: %w", err)
	}

	prompt := buildSimplifierPrompt(promptTemplate, truncatedContent, styleDirective)

	simplified, err := llm.GenerateAnswer(prompt)
	if err != nil {
		utils.Warnf("[SIMPLIFY] LLM simplification error: %v", err)
		return "", fmt.Errorf("failed to simplify content: %w", err)
	}

	return simplified, nil
}

