package app

import (
	"strings"

	"ai-tutor/internal/models"
	"ai-tutor/internal/study"
)

// GetPromptStudioTemplates returns the default template, active template, variants, and placeholder specification for all categories.
func (a *App) GetPromptStudioTemplates() map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}

	categories := []string{
		study.PromptCategoryNotes,
		study.PromptCategoryQuiz,
		study.PromptCategoryFlashcard,
		study.PromptCategoryTutorDuck,
		study.PromptCategorySimplify,
		study.PromptCategoryReaderQA,
	}

	// Fetch active profile textbooks for contextual assignment
	activeProfileID, _ := repo.GetActiveProfileID()
	notebooks, err := repo.GetNotebooks("", activeProfileID)
	if err != nil || len(notebooks) == 0 {
		notebooks, _ = repo.GetNotebooks("", "")
	}
	var bookList []map[string]interface{}
	for _, nb := range notebooks {
		bookList = append(bookList, map[string]interface{}{
			"id":         nb.ID,
			"title":      nb.Title,
			"profile_id": nb.ProfileID,
		})
	}

	result := make(map[string]interface{})
	for _, cat := range categories {
		spec := study.CategorySpecs[cat]
		defTmpl := study.DefaultTemplates[cat]
		activeTmpl := defTmpl

		variants, err := repo.GetPromptVariants(cat)
		if err != nil {
			variants = nil
		}

		// If no variants exist yet in database, seed them from built-ins
		if len(variants) == 0 {
			if presets, ok := study.BuiltInPresets[cat]; ok && len(presets) > 0 {
				for _, p := range presets {
					_ = repo.SavePromptVariant(p)
				}
				variants, _ = repo.GetPromptVariants(cat)
			}
			if len(variants) == 0 {
				defaultPreset := models.CustomTaskPromptVariant{
					ID:       cat + "_standard",
					Category: cat,
					Name:     "Standard Academic Template",
					Template: defTmpl,
					IsActive: true,
					IsSystem: true,
				}
				_ = repo.SavePromptVariant(defaultPreset)
				variants = []models.CustomTaskPromptVariant{defaultPreset}
			}
		}

		activeVariantID := ""
		for _, v := range variants {
			if v.IsActive {
				activeVariantID = v.ID
				activeTmpl = v.Template
				break
			}
		}
		if activeVariantID == "" && len(variants) > 0 {
			activeVariantID = variants[0].ID
			activeTmpl = variants[0].Template
			_ = repo.SetActivePromptVariant(cat, variants[0].ID)
		}

		// Fetch notebook overrides for this category
		assignments, _ := repo.GetPromptAssignmentsForCategory(cat)
		if assignments == nil {
			assignments = make(map[string]string)
		}

		result[cat] = map[string]interface{}{
			"category":               cat,
			"default_template":       defTmpl,
			"active_template":        activeTmpl,
			"active_variant_id":      activeVariantID,
			"variants":               variants,
			"assignments":            assignments,
			"is_custom":              activeTmpl != defTmpl,
			"mandatory_placeholders": spec.MandatoryPlaceholders,
			"optional_placeholders":  spec.OptionalPlaceholders,
		}
	}

	return map[string]interface{}{
		"ok":        true,
		"templates": result,
		"notebooks": bookList,
	}
}

// SetPromptNotebookAssignment assigns or unassigns a specific prompt variant to a notebook.
func (a *App) SetPromptNotebookAssignment(notebookID, category, variantID string) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	if err := repo.SetPromptAssignment(notebookID, category, variantID); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"ok": true, "notebook_id": notebookID, "category": category, "variant_id": variantID}
}

// SaveCustomPromptTemplate validates and saves a custom prompt variant for a category.
func (a *App) SaveCustomPromptTemplate(category, template string) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	category = strings.ToLower(strings.TrimSpace(category))
	if err := study.ValidateTemplate(category, template); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if err := repo.SetCustomPrompt(category, template); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"ok": true, "category": category}
}

// SavePromptVariant saves a named prompt variant.
func (a *App) SavePromptVariant(variant models.CustomTaskPromptVariant) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	if err := study.ValidateTemplate(variant.Category, variant.Template); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if err := repo.SavePromptVariant(variant); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"ok": true, "variant": variant}
}

// SetActivePromptVariant activates a prompt variant for a category.
func (a *App) SetActivePromptVariant(category, variantID string) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	if err := repo.SetActivePromptVariant(category, variantID); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"ok": true, "category": category, "variant_id": variantID}
}

// DeletePromptVariant removes a custom prompt variant.
func (a *App) DeletePromptVariant(variantID string) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	if err := repo.DeletePromptVariant(variantID); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"ok": true, "variant_id": variantID}
}

// ResetPromptTemplate removes custom prompt override for a category, restoring system default.
func (a *App) ResetPromptTemplate(category string) map[string]interface{} {
	repo := a.getRepo()
	if repo == nil {
		return map[string]interface{}{"error": errDatabaseNotInitialized}
	}
	category = strings.ToLower(strings.TrimSpace(category))
	if err := repo.DeleteCustomPrompt(category); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	defTmpl := study.DefaultTemplates[category]
	return map[string]interface{}{"ok": true, "category": category, "template": defTmpl}
}
