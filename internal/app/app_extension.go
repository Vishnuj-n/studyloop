package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"ai-tutor/internal/extension"
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
func (a *App) ListExtensions() ([]ExtensionDTO, error) {
	if a.extManager == nil {
		if a.extInitError != "" {
			return nil, fmt.Errorf("extension manager not initialized: %s", a.extInitError)
		}
		return nil, fmt.Errorf("extension manager not initialized")
	}
	if _, err := a.extManager.Discover(); err != nil {
		return nil, fmt.Errorf("failed to discover extensions: %w", err)
	}
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
	return dtos, nil
}

// RunExtension executes an extension by ID.
// If the extension is marked "pro" and isPro is false, it returns an entitlement error.
func (a *App) RunExtension(id string, input string, isPro bool) map[string]interface{} {
	if a.extManager == nil || a.extRunner == nil {
		if a.extInitError != "" {
			return map[string]interface{}{"error": fmt.Sprintf("extension system not initialized: %s", a.extInitError)}
		}
		return map[string]interface{}{"error": "extension system not initialized"}
	}

	ext, ok := a.extManager.Get(id)
	if !ok {
		if a.extInitError != "" {
			return map[string]interface{}{"error": fmt.Sprintf("extension %q not found (initialization error: %s)", id, a.extInitError)}
		}
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

	var output []byte
	var err error

	runtimeType := strings.ToLower(strings.TrimSpace(ext.Runtime()))
	switch runtimeType {
	case "internal", "builtin":
		if ext.ID() == "text_simplifier" {
			sampleText := strings.TrimSpace(input)
			if sampleText == "" {
				sampleText = "StudyLoop is an intelligent study queue platform designed to help students master complex subjects through active recall, spaced repetition, and clear conceptual explanations."
			}
			res := a.SimplifyReadingContent(sampleText)
			if errStr, ok := res["error"].(string); ok && errStr != "" {
				return map[string]interface{}{"error": errStr, "id": id}
			}
			if simplified, ok := res["simplified"].(string); ok {
				return map[string]interface{}{"output": simplified, "id": id}
			}
			return map[string]interface{}{"output": "No output generated", "id": id}
		}
		return map[string]interface{}{"output": "Built-in extension executed successfully.", "id": id}
	case "python", "py":
		venvDir := extension.ResolveExtensionVenvDir(ext)
		pyExe := extension.GetVenvPython(venvDir)
		if info, sErr := os.Stat(pyExe); sErr != nil || info.IsDir() {
			return map[string]interface{}{"error": fmt.Sprintf("Python virtual environment not initialized for extension %q. Please initialize environment via Setup.", ext.Name())}
		}
		var args []string
		args = append(args, ext.EntrypointPath())
		if strings.TrimSpace(input) != "" {
			args = append(args, strings.TrimSpace(input))
		}
		output, err = a.extRunner.Run(ctx, ext, pyExe, args...)
	case "binary", "executable", "exe":
		var args []string
		if strings.TrimSpace(input) != "" {
			args = append(args, strings.TrimSpace(input))
		}
		output, err = a.extRunner.Run(ctx, ext, ext.EntrypointPath(), args...)
	default:
		output, err = a.extRunner.Run(ctx, ext, ext.EntrypointPath())
	}

	if err != nil {
		return map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
			"id":     id,
		}
	}

	return map[string]interface{}{
		"output": string(output),
		"id":     id,
	}
}

// CheckExtensionReadiness inspects an extension's environment, virtual environment, and smoke test status.
func (a *App) CheckExtensionReadiness(id string) extension.ReadinessStatus {
	if a.extManager == nil {
		return extension.ReadinessStatus{
			ID:      id,
			IsReady: false,
			Error:   "extension manager not initialized",
		}
	}

	ext, ok := a.extManager.Get(id)
	if !ok {
		return extension.ReadinessStatus{
			ID:      id,
			IsReady: false,
			Error:   fmt.Sprintf("extension %q not found", id),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return extension.CheckReadiness(ctx, ext)
}

// SetupExtension automatically provisions the Python virtual environment via uv,
// installs requirements, and runs verification self-tests.
func (a *App) SetupExtension(id string) map[string]interface{} {
	if a.extManager == nil {
		return map[string]interface{}{"error": "extension manager not initialized"}
	}

	ext, ok := a.extManager.Get(id)
	if !ok {
		return map[string]interface{}{"error": fmt.Sprintf("extension %q not found", id)}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var logs []string
	logCallback := func(line string) {
		logs = append(logs, line)
	}

	err := extension.SetupExtensionEnv(ctx, ext, logCallback)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"logs":    logs,
			"id":      id,
		}
	}

	return map[string]interface{}{
		"success": true,
		"logs":    logs,
		"id":      id,
		"name":    ext.Name(),
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
	if a.studyService == nil {
		return map[string]interface{}{"error": "study service not available"}
	}
	simplified, err := a.studyService.SimplifyReadingContent(context.Background(), content)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"simplified": simplified,
	}
}
