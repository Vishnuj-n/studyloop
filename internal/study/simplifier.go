package study

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ai-tutor/internal/utils"
)

// SimplifyReadingContent takes dense text content and simplifies it using the fast LLM provider,
// loading the prompt template dynamically from extensions/text_simplifier/prompt.md if available.
func (s *StudyService) SimplifyReadingContent(ctx context.Context, content string) (string, error) {
	if s.fastLLMProvider == nil {
		return "", fmt.Errorf("AI provider not available. Please check your API key in Settings")
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("content cannot be empty")
	}

	// Attempt to load dynamic prompt template from extensions directory
	promptTemplate := ""
	promptPath := filepath.Join("extensions", "text_simplifier", "prompt.md")
	if data, err := os.ReadFile(promptPath); err == nil {
		promptTemplate = string(data)
	}

	var prompt string
	if promptTemplate != "" && strings.Contains(promptTemplate, "{{content}}") {
		prompt = strings.ReplaceAll(promptTemplate, "{{content}}", content)
	} else if promptTemplate != "" {
		prompt = promptTemplate + "\n\nReading Material:\n\"\"\"\n" + content + "\n\"\"\"\n"
	} else {
		prompt = fmt.Sprintf(`You are an expert tutor and text clarifier.
Your goal is to rewrite and simplify the following reading material so that it is intuitive, crystal clear, and easy to understand, WITHOUT LOSING any technical accuracy, formulas, key definitions, or essential details.

Format your output in clean Markdown with:
1. **TL;DR Overview**: 1-2 sentence core intuition.
2. **Key Concepts Explained Simply**: Clear, intuitive explanations using real-world analogies where helpful.
3. **Step-by-Step Breakdown**: Detailed, structured explanation with all core facts intact.
4. **Quick Summary / Key Takeaways**: Bullet points of what to remember.

Reading Material:
"""
%s
"""

Return only the markdown response without meta-commentary.`, content)
	}

	simplified, err := s.fastLLMProvider.GenerateAnswer(prompt)
	if err != nil {
		utils.Warnf("[SIMPLIFY] LLM simplification error: %v", err)
		return "", fmt.Errorf("failed to simplify content: %w", err)
	}

	return simplified, nil
}
