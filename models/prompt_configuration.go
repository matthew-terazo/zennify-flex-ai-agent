package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type PromptConfiguration struct {
	ID           uuid.UUID    `db:"id" json:"id"`
	Name         string       `db:"name" json:"name"`
	SystemPrompt *string      `db:"system_prompt" json:"systemPrompt"`
	Metadata     MetadataJSON `db:"metadata" json:"metadata"`
	Version      int          `db:"version" json:"version"`
	IsActive     bool         `db:"is_active" json:"isActive"`
	CreatedAt    time.Time    `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time    `db:"updated_at" json:"updatedAt"`
}

type KnowledgeEntry struct {
	ID        uuid.UUID    `db:"id"`
	ConfigID  uuid.UUID    `db:"config_id"`
	Category  *string      `db:"category"`
	Content   string       `db:"content"`
	Metadata  MetadataJSON `db:"metadata"`
	CreatedAt time.Time    `db:"created_at"`
	UpdatedAt time.Time    `db:"updated_at"`
}

type KnowledgeVersion struct {
	ID             uuid.UUID `db:"id"`
	EntryID        uuid.UUID `db:"entry_id"`
	ContentVersion int       `db:"content_version"`
	Content        string    `db:"content"`
	ChangedAt      time.Time `db:"changed_at"`
}

// MetadataJSON is a JSONB wrapper for arbitrary metadata
type MetadataJSON map[string]interface{}

func (m MetadataJSON) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *MetadataJSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion failed")
	}
	return json.Unmarshal(bytes, &m)
}
