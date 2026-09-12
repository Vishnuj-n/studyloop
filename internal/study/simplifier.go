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
	case "very_simple", "eli10":
		return "Very Simple:\nUse everyday language and short sentences. Explain necessary technical terms simply and use intuitive analogies when helpful."
	case "academic":
		return "Academic:\nUse precise academic and technical language. Preserve terminology, definitions, distinctions, and technical details. Simplify only unnecessarily complicated wording."
	case "summary", "bullet":
		return "Summary:\nFocus on the essential ideas, arguments, definitions, and conclusions. Be substantially shorter than the source while preserving the information needed to understand the topic."
	default: // "simple", "eli15", or default
		return "Simple:\nUse clear, straightforward language. Explain difficult terminology, but retain the important technical vocabulary and detail."
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

Transform the provided reading material into clear, well-structured Markdown notes that are easier to understand while preserving the original meaning and important information.

The user's selected comprehension level is:

%s

Adapt the explanation strictly to this level.

## Core Requirements

- Preserve the author's meaning, important facts, definitions, arguments, formulas, examples, and necessary technical details.
- Simplify wording and sentence structure according to the selected comprehension level.
- Explain difficult terminology when necessary for understanding.
- Do not remove important information simply because it is difficult.
- Do not invent facts, examples, arguments, or conclusions.
- Do not add opinions, motivation, or unrelated information.
- Do not turn the material into a lecture or a conversation.
- Remove unnecessary repetition and filler.
- Keep technical terminology when precision requires it.
- Use natural Markdown structure: headings, subheadings, paragraphs, bullets, numbered steps, and equations where appropriate.
- Use tables only when they genuinely improve understanding.
- Follow the logical flow of the source material.
- Use analogies or brief clarifications when they genuinely make a difficult concept easier to understand.

## Output

Create a concise but sufficiently detailed explanation of the provided material.

Start with a short overview of the main topic.

Then organize the explanation into natural sections based on the material. Do not force the content into a fixed template.

End with a short **Key Takeaways** section containing the most important ideas.

The result should feel like a clearer and better-organized version of the original reading, not a separate interpretation or lesson about it.

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

