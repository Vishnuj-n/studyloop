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
	case "eli10":
		return "Explain using simple 5th-grade vocabulary, short sentences, and crystal-clear everyday analogies."
	case "bullet":
		return "Provide an ultra-concise executive summary with high-impact bullet points and key takeaways."
	case "academic":
		return "Maintain rigorous academic precision, definitions, formulas, and domain terminology intact while clarifying logical flow."
	default: // "eli15" or fallback
		return "Explain for a high school student using intuitive real-world analogies without losing core technical concepts or accuracy."
	}
}

// buildSimplifierPrompt formats the prompt given a template, material content, and comprehension style.
func buildSimplifierPrompt(promptTemplate, content, styleDirective string) string {
	if styleDirective == "" {
		styleDirective = getSimplifierLevelDirective("eli15")
	}

	if promptTemplate != "" {
		tmpl := strings.ReplaceAll(promptTemplate, "{{style}}", styleDirective)
		if strings.Contains(tmpl, "{{content}}") {
			return strings.ReplaceAll(tmpl, "{{content}}", content)
		}
		return tmpl + "\n\nReading Material:\n\"\"\"\n" + content + "\n\"\"\"\n"
	}

	return fmt.Sprintf(`You are an expert tutor and text clarifier.
Your goal is to rewrite and simplify the following reading material so that it is intuitive, crystal clear, and easy to understand, WITHOUT LOSING any technical accuracy, formulas, key definitions, or essential details.

Audience & Comprehension Style:
%s

Format your output in clean Markdown with:
1. **TL;DR Overview**: 1-2 sentence core intuition.
2. **Key Concepts Explained Simply**: Clear, intuitive explanations using real-world analogies where helpful.
3. **Step-by-Step Breakdown**: Detailed, structured explanation with all core facts intact.
4. **Quick Summary / Key Takeaways**: Bullet points of what to remember.

Reading Material:
"""
%s
"""

Return only the markdown response without meta-commentary.`, styleDirective, content)
}

// SimplifyReadingContent takes dense text content and simplifies it using the fast LLM provider,
// loading the prompt template dynamically from extensions/text_simplifier/prompt.md and injecting the comprehension style.
func (s *StudyService) SimplifyReadingContent(ctx context.Context, content string, level ...string) (string, error) {
	if s.fastLLMProvider == nil {
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

	limits := s.fastLLMProvider.GetLimits()
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

	simplified, err := s.fastLLMProvider.GenerateAnswer(prompt)
	if err != nil {
		utils.Warnf("[SIMPLIFY] LLM simplification error: %v", err)
		return "", fmt.Errorf("failed to simplify content: %w", err)
	}

	return simplified, nil
}

