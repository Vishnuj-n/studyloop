package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"ai-tutor/internal/models"

	"github.com/google/uuid"
)

// GetActiveCustomPrompt retrieves the currently active prompt variant for a category.
func (r *Repository) GetActiveCustomPrompt(category string) (string, error) {
	category = strings.ToLower(strings.TrimSpace(category))
	var tmpl string
	err := r.db.QueryRow(`SELECT template FROM custom_task_prompts WHERE category = ? AND is_active = 1 LIMIT 1`, category).Scan(&tmpl)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return tmpl, err
}

// GetPromptVariants returns all prompt variations for a category.
func (r *Repository) GetPromptVariants(category string) ([]models.CustomTaskPromptVariant, error) {
	category = strings.ToLower(strings.TrimSpace(category))
	rows, err := r.db.Query(`
		SELECT id, category, name, template, is_active, is_system, created_at, updated_at
		FROM custom_task_prompts
		WHERE category = ?
		ORDER BY is_system DESC, created_at ASC
	`, category)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var variants []models.CustomTaskPromptVariant
	for rows.Next() {
		var v models.CustomTaskPromptVariant
		if err := rows.Scan(&v.ID, &v.Category, &v.Name, &v.Template, &v.IsActive, &v.IsSystem, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		variants = append(variants, v)
	}
	return variants, rows.Err()
}

// GetAllPromptVariants returns all prompt variants across all categories.
func (r *Repository) GetAllPromptVariants() ([]models.CustomTaskPromptVariant, error) {
	rows, err := r.db.Query(`
		SELECT id, category, name, template, is_active, is_system, created_at, updated_at
		FROM custom_task_prompts
		ORDER BY category ASC, is_system DESC, created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var variants []models.CustomTaskPromptVariant
	for rows.Next() {
		var v models.CustomTaskPromptVariant
		if err := rows.Scan(&v.ID, &v.Category, &v.Name, &v.Template, &v.IsActive, &v.IsSystem, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		variants = append(variants, v)
	}
	return variants, rows.Err()
}

// SavePromptVariant inserts or updates a prompt variant.
func (r *Repository) SavePromptVariant(v models.CustomTaskPromptVariant) error {
	v.Category = strings.ToLower(strings.TrimSpace(v.Category))
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	now := time.Now().Unix()
	if v.CreatedAt == 0 {
		v.CreatedAt = now
	}
	v.UpdatedAt = now

	// If this variant is set to active, deactivate other variants in the same category
	if v.IsActive {
		if _, err := r.db.Exec(`UPDATE custom_task_prompts SET is_active = 0 WHERE category = ?`, v.Category); err != nil {
			return err
		}
	}

	_, err := r.db.Exec(`
		INSERT INTO custom_task_prompts (id, category, name, template, is_active, is_system, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			template = excluded.template,
			is_active = excluded.is_active,
			is_system = excluded.is_system,
			updated_at = excluded.updated_at
	`, v.ID, v.Category, v.Name, v.Template, v.IsActive, v.IsSystem, v.CreatedAt, v.UpdatedAt)
	return err
}

// SetActivePromptVariant marks a specific variant as active for its category.
func (r *Repository) SetActivePromptVariant(category, variantID string) error {
	category = strings.ToLower(strings.TrimSpace(category))
	if _, err := r.db.Exec(`UPDATE custom_task_prompts SET is_active = 0 WHERE category = ?`, category); err != nil {
		return err
	}
	_, err := r.db.Exec(`UPDATE custom_task_prompts SET is_active = 1, updated_at = ? WHERE id = ? AND category = ?`, time.Now().Unix(), variantID, category)
	return err
}

// DeletePromptVariant removes a custom prompt variant (system variants cannot be deleted).
func (r *Repository) DeletePromptVariant(variantID string) error {
	var isSystem bool
	var category string
	err := r.db.QueryRow(`SELECT is_system, category FROM custom_task_prompts WHERE id = ?`, variantID).Scan(&isSystem, &category)
	if err != nil {
		return err
	}
	if isSystem {
		return fmt.Errorf("cannot delete a system preset variant")
	}

	_, err = r.db.Exec(`DELETE FROM custom_task_prompts WHERE id = ?`, variantID)
	if err != nil {
		return err
	}

	// Clean up any notebook assignments referencing this deleted variant
	_, _ = r.db.Exec(`DELETE FROM prompt_notebook_assignments WHERE variant_id = ?`, variantID)

	// If no active variant remains in category, set default system one active
	var activeCount int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM custom_task_prompts WHERE category = ? AND is_active = 1`, category).Scan(&activeCount)
	if activeCount == 0 {
		_, _ = r.db.Exec(`
			UPDATE custom_task_prompts
			SET is_active = 1
			WHERE id = (SELECT id FROM custom_task_prompts WHERE category = ? ORDER BY is_system DESC, created_at ASC LIMIT 1)
		`, category)
	}
	return nil
}

// GetPromptAssignmentsForCategory returns map of notebook_id -> variant_id for a category.
func (r *Repository) GetPromptAssignmentsForCategory(category string) (map[string]string, error) {
	category = strings.ToLower(strings.TrimSpace(category))
	rows, err := r.db.Query(`SELECT notebook_id, variant_id FROM prompt_notebook_assignments WHERE category = ?`, category)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	res := make(map[string]string)
	for rows.Next() {
		var nbID, vID string
		if err := rows.Scan(&nbID, &vID); err != nil {
			return nil, err
		}
		res[nbID] = vID
	}
	return res, rows.Err()
}

// SetPromptAssignment locks or unlinks a prompt variant for a specific notebook.
func (r *Repository) SetPromptAssignment(notebookID, category, variantID string) error {
	notebookID = strings.TrimSpace(notebookID)
	category = strings.ToLower(strings.TrimSpace(category))
	variantID = strings.TrimSpace(variantID)
	if variantID == "" {
		_, err := r.db.Exec(`DELETE FROM prompt_notebook_assignments WHERE notebook_id = ? AND category = ?`, notebookID, category)
		return err
	}
	now := time.Now().Unix()
	_, err := r.db.Exec(`
		INSERT INTO prompt_notebook_assignments (notebook_id, category, variant_id, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(notebook_id, category) DO UPDATE SET
			variant_id = excluded.variant_id,
			updated_at = excluded.updated_at
	`, notebookID, category, variantID, now)
	return err
}

// GetAssignedPromptTemplate returns the notebook-scoped template if set, otherwise global active template.
func (r *Repository) GetAssignedPromptTemplate(notebookID, category string) (string, error) {
	category = strings.ToLower(strings.TrimSpace(category))
	var tmpl string
	if strings.TrimSpace(notebookID) != "" {
		err := r.db.QueryRow(`
			SELECT p.template
			FROM prompt_notebook_assignments a
			JOIN custom_task_prompts p ON a.variant_id = p.id
			WHERE a.notebook_id = ? AND a.category = ?
		`, notebookID, category).Scan(&tmpl)
		if err == nil && strings.TrimSpace(tmpl) != "" {
			return tmpl, nil
		}
	}
	return r.GetActiveCustomPrompt(category)
}

// Backward compatibility methods for PromptRepository interface
func (r *Repository) GetCustomPrompt(category string) (string, error) {
	return r.GetActiveCustomPrompt(category)
}

func (r *Repository) SetCustomPrompt(category string, template string) error {
	category = strings.ToLower(strings.TrimSpace(category))
	variants, err := r.GetPromptVariants(category)
	if err == nil {
		for _, v := range variants {
			if !v.IsSystem && v.IsActive {
				v.Template = template
				v.UpdatedAt = time.Now().Unix()
				return r.SavePromptVariant(v)
			}
		}
	}
	return r.SavePromptVariant(models.CustomTaskPromptVariant{
		ID:        uuid.New().String(),
		Category:  category,
		Name:      "Custom Prompt",
		Template:  template,
		IsActive:  true,
		IsSystem:  false,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	})
}

func (r *Repository) DeleteCustomPrompt(category string) error {
	category = strings.ToLower(strings.TrimSpace(category))
	if _, err := r.db.Exec(`UPDATE custom_task_prompts SET is_active = 0 WHERE category = ?`, category); err != nil {
		return err
	}
	_, err := r.db.Exec(`
		UPDATE custom_task_prompts
		SET is_active = 1
		WHERE id = (SELECT id FROM custom_task_prompts WHERE category = ? AND is_system = 1 LIMIT 1)
	`, category)
	return err
}

func (r *Repository) GetAllCustomPrompts() (map[string]string, error) {
	rows, err := r.db.Query(`SELECT category, template FROM custom_task_prompts WHERE is_active = 1`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	prompts := make(map[string]string)
	for rows.Next() {
		var cat, tmpl string
		if err := rows.Scan(&cat, &tmpl); err != nil {
			return nil, err
		}
		prompts[cat] = tmpl
	}
	return prompts, rows.Err()
}
