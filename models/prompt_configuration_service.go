package models

import (
	"context"

	"github.com/google/uuid"
)

// PromptConfigurationService defines the interface for prompt configuration operations
type PromptConfigurationService interface {
	GetActiveConfiguration(ctx context.Context) (PromptConfiguration, error)
	GetConfigurationByID(ctx context.Context, id uuid.UUID) (PromptConfiguration, error)
	GetConfigurationByName(ctx context.Context, name string) (PromptConfiguration, error)
	ListConfigurations(ctx context.Context, status string) ([]PromptConfiguration, error)
	CreateConfiguration(ctx context.Context, name string, systemPrompt *string, metadata MetadataJSON) (PromptConfiguration, error)
	UpdateConfiguration(ctx context.Context, id uuid.UUID, systemPrompt *string, metadata MetadataJSON) error
	GetKnowledgeEntriesByConfig(ctx context.Context, configID uuid.UUID) ([]KnowledgeEntry, error)
	AddKnowledgeEntry(ctx context.Context, configID uuid.UUID, category *string, content string, metadata MetadataJSON) (KnowledgeEntry, error)
	GetKnowledgeVersions(ctx context.Context, entryID uuid.UUID) ([]KnowledgeVersion, error)
	UpdateKnowledgeEntry(ctx context.Context, entryID uuid.UUID, content string) error
	ActivateConfiguration(ctx context.Context, configID uuid.UUID) error
}
