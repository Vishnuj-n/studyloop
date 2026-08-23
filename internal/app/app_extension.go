package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-tutor/internal/extension"
	"ai-tutor/internal/utils"
)

// ExtensionDTO represents an extension serialized for the frontend.
type ExtensionDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Runtime     string `json:"runtime"`
	Tier        string `json:"tier"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Dir         string `json:"dir"`
}

// ListExtensions returns all discovered local extensions.
func (a *App) ListExtensions() []ExtensionDTO {
	if a.extManager == nil {
		return []ExtensionDTO{}
	}
	_, _ = a.extManager.Discover()
	exts := a.extManager.List()

	dtos := make([]ExtensionDTO, 0, len(exts))
	for _, ext := range exts {
		dtos = append(dtos, ExtensionDTO{
			ID:          ext.ID(),
			Name:        ext.Name(),
			Version:     ext.Version(),
			Runtime:     ext.Runtime(),
			Tier:        extension.GetEffectiveTier(ext),
			Description: ext.Manifest.Description,
			Category:    ext.Manifest.Category,
			Dir:         ext.Dir,
		})
	}
	return dtos
}

// RunExtension executes an extension by ID.
// If the extension is marked "pro" and isPro is false, it returns an entitlement error.
func (a *App) RunExtension(id string, input string, isPro bool) map[string]interface{} {
	if a.extManager == nil || a.extRunner == nil {
		return map[string]interface{}{"error": "extension system not initialized"}
	}

	ext, ok := a.extManager.Get(id)
	if !ok {
		return map[string]interface{}{"error": fmt.Sprintf("extension %q not found", id)}
	}

	effectiveTier := extension.GetEffectiveTier(ext)
	// Authoritative security guard for pro extensions
	if effectiveTier == "pro" && !isPro {
		return map[string]interface{}{
			"error":           fmt.Sprintf("extension %q requires a Pro subscription", ext.Name()),
			"is_pro_required": true,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	output, err := a.extRunner.Run(ctx, ext, input)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"output": output,
		"id":     id,
	}
}

// InstallExtensionZip installs a packaged extension from a local zip file.
func (a *App) InstallExtensionZip(zipPath string) map[string]interface{} {
	if a.extManager == nil {
		return map[string]interface{}{"error": "extension system not initialized"}
	}

	ext, err := a.extManager.InstallZip(zipPath)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"success": true,
		"id":      ext.ID(),
		"name":    ext.Name(),
		"tier":    ext.Manifest.Tier,
	}
}

// UninstallExtension removes an extension from the system.
func (a *App) UninstallExtension(id string) map[string]interface{} {
	if a.extManager == nil {
		return map[string]interface{}{"error": "extension system not initialized"}
	}

	if err := a.extManager.Uninstall(id); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"success": true,
		"id":      id,
	}
}

// SimplifyReadingContent takes dense text content and simplifies it using the LLM
// while preserving all core technical concepts, formulas, and definitions.
func (a *App) SimplifyReadingContent(content string) map[string]interface{} {
	a.aiMutex.Lock()
	provider := a.fastLLMProvider
	if provider == nil {
		provider = a.heavyLLMProvider
	}
	a.aiMutex.Unlock()

	if provider == nil {
		return map[string]interface{}{"error": "AI provider not available. Please check your API key in Settings."}
	}

	if strings.TrimSpace(content) == "" {
		return map[string]interface{}{"error": "content cannot be empty"}
	}

	prompt := fmt.Sprintf(`You are an expert tutor and text clarifier.
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

	simplified, err := provider.GenerateAnswer(prompt)
	if err != nil {
		utils.Warnf("[SIMPLIFY] LLM simplification error: %v", err)
		return map[string]interface{}{"error": fmt.Sprintf("failed to simplify content: %v", err)}
	}

	return map[string]interface{}{
		"simplified": simplified,
	}
}
