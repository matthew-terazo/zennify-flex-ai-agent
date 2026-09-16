package promptconfiguration

import (
	"context"
	"database/sql"
	"errors"

	"zennify/zenn-ai-assistant/models"

	"github.com/google/uuid"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// ListConfigurations retrieves all prompt configurations, optionally filtered by status
func (s *Service) ListConfigurations(ctx context.Context, status string) ([]models.PromptConfiguration, error) {
	var query string
	var args []interface{}

	if status == "active" {
		query = `SELECT id, name, system_prompt, metadata, version, is_active, created_at, updated_at
		         FROM prompt_configurations WHERE is_active = true ORDER BY created_at DESC`
	} else if status == "inactive" {
		query = `SELECT id, name, system_prompt, metadata, version, is_active, created_at, updated_at
		         FROM prompt_configurations WHERE is_active = false ORDER BY created_at DESC`
	} else {
		query = `SELECT id, name, system_prompt, metadata, version, is_active, created_at, updated_at
		         FROM prompt_configurations ORDER BY created_at DESC`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []models.PromptConfiguration
	for rows.Next() {
		var cfg models.PromptConfiguration
		err := rows.Scan(&cfg.ID, &cfg.Name, &cfg.SystemPrompt, &cfg.Metadata, &cfg.Version,
			&cfg.IsActive, &cfg.CreatedAt, &cfg.UpdatedAt)
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}

	return configs, rows.Err()
}

// GetActiveConfiguration retrieves the currently active prompt configuration
func (s *Service) GetActiveConfiguration(ctx context.Context) (models.PromptConfiguration, error) {
	var cfg models.PromptConfiguration
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, system_prompt, metadata, version, is_active, created_at, updated_at
		 FROM prompt_configurations WHERE is_active = true LIMIT 1`,
	).Scan(
		&cfg.ID, &cfg.Name, &cfg.SystemPrompt, &cfg.Metadata, &cfg.Version,
		&cfg.IsActive, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.PromptConfiguration{}, errors.New("no active configuration found")
		}
		return models.PromptConfiguration{}, err
	}
	return cfg, nil
}

// GetConfigurationByID retrieves a configuration by ID
func (s *Service) GetConfigurationByID(ctx context.Context, id uuid.UUID) (models.PromptConfiguration, error) {
	var cfg models.PromptConfiguration
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, system_prompt, metadata, version, is_active, created_at, updated_at
		 FROM prompt_configurations WHERE id = $1`,
		id,
	).Scan(
		&cfg.ID, &cfg.Name, &cfg.SystemPrompt, &cfg.Metadata, &cfg.Version,
		&cfg.IsActive, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.PromptConfiguration{}, errors.New("configuration not found")
		}
		return models.PromptConfiguration{}, err
	}
	return cfg, nil
}

// GetConfigurationByName retrieves a configuration by name
func (s *Service) GetConfigurationByName(ctx context.Context, name string) (models.PromptConfiguration, error) {
	var cfg models.PromptConfiguration
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, system_prompt, metadata, version, is_active, created_at, updated_at
		 FROM prompt_configurations WHERE name = $1`,
		name,
	).Scan(
		&cfg.ID, &cfg.Name, &cfg.SystemPrompt, &cfg.Metadata, &cfg.Version,
		&cfg.IsActive, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.PromptConfiguration{}, errors.New("configuration not found")
		}
		return models.PromptConfiguration{}, err
	}
	return cfg, nil
}

// CreateConfiguration creates a new prompt configuration
func (s *Service) CreateConfiguration(ctx context.Context, name string, systemPrompt *string, metadata models.MetadataJSON) (models.PromptConfiguration, error) {
	var cfg models.PromptConfiguration
	cfg.ID = uuid.New()
	cfg.Name = name
	cfg.SystemPrompt = systemPrompt
	cfg.Metadata = metadata
	cfg.Version = 1
	cfg.IsActive = false

	err := s.db.QueryRowContext(
		ctx,
		`INSERT INTO prompt_configurations (id, name, system_prompt, metadata, version, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at, updated_at`,
		cfg.ID, cfg.Name, cfg.SystemPrompt, cfg.Metadata, cfg.Version, cfg.IsActive,
	).Scan(&cfg.CreatedAt, &cfg.UpdatedAt)

	if err != nil {
		return models.PromptConfiguration{}, err
	}
	return cfg, nil
}

// GetKnowledgeEntriesByConfig retrieves all knowledge entries for a configuration
func (s *Service) GetKnowledgeEntriesByConfig(ctx context.Context, configID uuid.UUID) ([]models.KnowledgeEntry, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, config_id, category, content, metadata, created_at, updated_at
		 FROM knowledge_entries WHERE config_id = $1 ORDER BY created_at DESC`,
		configID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.KnowledgeEntry
	for rows.Next() {
		var entry models.KnowledgeEntry
		err := rows.Scan(
			&entry.ID, &entry.ConfigID, &entry.Category, &entry.Content, &entry.Metadata,
			&entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// AddKnowledgeEntry adds a new knowledge entry to a configuration
func (s *Service) AddKnowledgeEntry(ctx context.Context, configID uuid.UUID, category *string, content string, metadata models.MetadataJSON) (models.KnowledgeEntry, error) {
	var entry models.KnowledgeEntry
	entry.ID = uuid.New()
	entry.ConfigID = configID
	entry.Category = category
	entry.Content = content
	entry.Metadata = metadata

	err := s.db.QueryRowContext(
		ctx,
		`INSERT INTO knowledge_entries (id, config_id, category, content, metadata)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING created_at, updated_at`,
		entry.ID, entry.ConfigID, entry.Category, entry.Content, entry.Metadata,
	).Scan(&entry.CreatedAt, &entry.UpdatedAt)

	if err != nil {
		return models.KnowledgeEntry{}, err
	}

	// Create version record
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO knowledge_versions (id, entry_id, content_version, content)
		 VALUES ($1, $2, $3, $4)`,
		uuid.New(), entry.ID, 1, content,
	)
	if err != nil {
		return models.KnowledgeEntry{}, err
	}

	return entry, nil
}

// GetKnowledgeVersions retrieves version history for a knowledge entry
func (s *Service) GetKnowledgeVersions(ctx context.Context, entryID uuid.UUID) ([]models.KnowledgeVersion, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, entry_id, content_version, content, changed_at
		 FROM knowledge_versions WHERE entry_id = $1 ORDER BY content_version DESC`,
		entryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []models.KnowledgeVersion
	for rows.Next() {
		var version models.KnowledgeVersion
		err := rows.Scan(&version.ID, &version.EntryID, &version.ContentVersion, &version.Content, &version.ChangedAt)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}

	return versions, rows.Err()
}

// UpdateKnowledgeEntry updates a knowledge entry and creates a new version
func (s *Service) UpdateKnowledgeEntry(ctx context.Context, entryID uuid.UUID, content string) error {
	// Get current version number
	var currentVersion int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(content_version), 0) FROM knowledge_versions WHERE entry_id = $1`,
		entryID,
	).Scan(&currentVersion)
	if err != nil {
		return err
	}

	var newVersion int
	newVersion = currentVersion + 1

	// Update the entry
	_, err = s.db.ExecContext(
		ctx,
		`UPDATE knowledge_entries SET content = $1, updated_at = now() WHERE id = $2`,
		content, entryID,
	)
	if err != nil {
		return err
	}

	// Create version record
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO knowledge_versions (id, entry_id, content_version, content)
		 VALUES ($1, $2, $3, $4)`,
		uuid.New(), entryID, newVersion, content,
	)

	return err
}

// UpdateConfiguration updates a prompt configuration
func (s *Service) UpdateConfiguration(ctx context.Context, id uuid.UUID, systemPrompt *string, metadata models.MetadataJSON) error {
	var query string
	var args []interface{}

	if systemPrompt != nil && metadata != nil {
		query = `UPDATE prompt_configurations SET system_prompt = $1, metadata = $2, updated_at = now() WHERE id = $3`
		args = []interface{}{*systemPrompt, metadata, id}
	} else if systemPrompt != nil {
		query = `UPDATE prompt_configurations SET system_prompt = $1, updated_at = now() WHERE id = $2`
		args = []interface{}{*systemPrompt, id}
	} else if metadata != nil {
		query = `UPDATE prompt_configurations SET metadata = $1, updated_at = now() WHERE id = $2`
		args = []interface{}{metadata, id}
	} else {
		return errors.New("at least one field must be provided for update")
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("configuration not found")
	}

	return nil
}

// ActivateConfiguration makes a configuration active and deactivates others
func (s *Service) ActivateConfiguration(ctx context.Context, configID uuid.UUID) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Deactivate all others
	_, err = tx.ExecContext(ctx, `UPDATE prompt_configurations SET is_active = false`)
	if err != nil {
		return err
	}

	// Activate this one
	_, err = tx.ExecContext(ctx, `UPDATE prompt_configurations SET is_active = true WHERE id = $1`, configID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
