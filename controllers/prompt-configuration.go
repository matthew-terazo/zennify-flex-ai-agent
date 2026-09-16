package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"zennify/zenn-ai-assistant/middleware"
	"zennify/zenn-ai-assistant/models"
)

type PromptConfigurationController struct {
	Service models.PromptConfigurationService
}

type getActiveConfigurationHandler struct {
	controller *PromptConfigurationController
}

func (h getActiveConfigurationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.controller.GetActiveConfiguration(w, r)
}

type getConfigurationByNameHandler struct {
	controller *PromptConfigurationController
}

func (h getConfigurationByNameHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.controller.GetConfigurationByName(w, r)
}

type createConfigurationHandler struct {
	controller *PromptConfigurationController
}

func (h createConfigurationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.controller.CreateConfiguration(w, r)
}

type getKnowledgeEntriesHandler struct {
	controller *PromptConfigurationController
}

func (h getKnowledgeEntriesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.controller.GetKnowledgeEntries(w, r)
}

type addKnowledgeEntryHandler struct {
	controller *PromptConfigurationController
}

func (h addKnowledgeEntryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.controller.AddKnowledgeEntry(w, r)
}

type getKnowledgeVersionsHandler struct {
	controller *PromptConfigurationController
}

func (h getKnowledgeVersionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.controller.GetKnowledgeVersions(w, r)
}

type updateKnowledgeEntryHandler struct {
	controller *PromptConfigurationController
}

func (h updateKnowledgeEntryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.controller.UpdateKnowledgeEntry(w, r)
}

type activateConfigurationHandler struct {
	controller *PromptConfigurationController
}

func (h activateConfigurationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.controller.ActivateConfiguration(w, r)
}

type listConfigurationsHandler struct {
	controller *PromptConfigurationController
}

func (h listConfigurationsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.controller.ListConfigurations(w, r)
}

type updateConfigurationHandler struct {
	controller *PromptConfigurationController
}

func (h updateConfigurationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.controller.UpdateConfiguration(w, r)
}

type createConfigurationRequest struct {
	Name         string                 `json:"name"`
	SystemPrompt *string                `json:"system_prompt"`
	Metadata     models.MetadataJSON    `json:"metadata"`
}

type addKnowledgeEntryRequest struct {
	Category string              `json:"category"`
	Content  string              `json:"content"`
	Metadata models.MetadataJSON `json:"metadata"`
}

type updateKnowledgeEntryRequest struct {
	Content string `json:"content"`
}

type activateConfigurationRequest struct {
	ConfigurationID string `json:"configuration_id"`
}

type updateConfigurationRequest struct {
	SystemPrompt *string             `json:"system_prompt"`
	Metadata     models.MetadataJSON `json:"metadata"`
}

// ListConfigurations handles GET /api/prompt-configurations?status={active|inactive}
func (c *PromptConfigurationController) ListConfigurations(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	status := r.URL.Query().Get("status")

	configs, err := c.Service.ListConfigurations(r.Context(), status)
	if err != nil {
		logger.Error().Err(err).Str("status", status).Msg("Failed to list configurations")
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to list configurations"})
		return
	}

	if configs == nil {
		configs = []models.PromptConfiguration{}
	}

	writeJSONResponse(w, http.StatusOK, configs)
}

// GetActiveConfiguration handles GET /api/prompt-configurations/active
func (c *PromptConfigurationController) GetActiveConfiguration(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	cfg, err := c.Service.GetActiveConfiguration(r.Context())
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get active configuration")
		writeJSONResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSONResponse(w, http.StatusOK, cfg)
}

// GetConfigurationByName handles GET /api/prompt-configurations/{identifier} where identifier can be UUID or name
func (c *PromptConfigurationController) GetConfigurationByName(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	identifier := r.PathValue("name")

	// Try parsing as UUID first
	if id, err := uuid.Parse(identifier); err == nil {
		cfg, err := c.Service.GetConfigurationByID(r.Context(), id)
		if err == nil {
			writeJSONResponse(w, http.StatusOK, cfg)
			return
		}
		logger.Debug().Err(err).Str("id", id.String()).Msg("Configuration not found by ID")
	}

	// Fall back to name lookup
	cfg, err := c.Service.GetConfigurationByName(r.Context(), identifier)
	if err != nil {
		logger.Error().Err(err).Str("identifier", identifier).Msg("Failed to get configuration")
		writeJSONResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSONResponse(w, http.StatusOK, cfg)
}

// CreateConfiguration handles POST /api/prompt-configurations
func (c *PromptConfigurationController) CreateConfiguration(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	var req createConfigurationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error().Err(err).Msg("Failed to decode request body")
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	var metadata models.MetadataJSON
	if req.Metadata != nil {
		metadata = req.Metadata
	} else {
		metadata = make(models.MetadataJSON)
	}

	cfg, err := c.Service.CreateConfiguration(r.Context(), req.Name, req.SystemPrompt, metadata)
	if err != nil {
		logger.Error().Err(err).Str("name", req.Name).Msg("Failed to create configuration")
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to create configuration"})
		return
	}

	writeJSONResponse(w, http.StatusCreated, cfg)
}

// GetKnowledgeEntries handles GET /api/prompt-configurations/{id}/knowledge-entries
func (c *PromptConfigurationController) GetKnowledgeEntries(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	configIDStr := r.PathValue("id")

	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		logger.Error().Err(err).Str("id", configIDStr).Msg("Invalid configuration ID")
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid configuration id"})
		return
	}

	entries, err := c.Service.GetKnowledgeEntriesByConfig(r.Context(), configID)
	if err != nil {
		logger.Error().Err(err).Str("config_id", configID.String()).Msg("Failed to get knowledge entries")
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to get knowledge entries"})
		return
	}

	if entries == nil {
		entries = []models.KnowledgeEntry{}
	}

	writeJSONResponse(w, http.StatusOK, entries)
}

// AddKnowledgeEntry handles POST /api/prompt-configurations/{id}/knowledge-entries
func (c *PromptConfigurationController) AddKnowledgeEntry(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	configIDStr := r.PathValue("id")

	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		logger.Error().Err(err).Str("id", configIDStr).Msg("Invalid configuration ID")
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid configuration id"})
		return
	}

	var req addKnowledgeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error().Err(err).Msg("Failed to decode request body")
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	var category *string
	if req.Category != "" {
		category = &req.Category
	}

	var metadata models.MetadataJSON
	if req.Metadata != nil {
		metadata = req.Metadata
	} else {
		metadata = make(models.MetadataJSON)
	}

	entry, err := c.Service.AddKnowledgeEntry(r.Context(), configID, category, req.Content, metadata)
	if err != nil {
		logger.Error().Err(err).Str("config_id", configID.String()).Msg("Failed to add knowledge entry")
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to add knowledge entry"})
		return
	}

	writeJSONResponse(w, http.StatusCreated, entry)
}

// GetKnowledgeVersions handles GET /api/knowledge-entries/{id}/versions
func (c *PromptConfigurationController) GetKnowledgeVersions(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	entryIDStr := r.PathValue("id")

	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		logger.Error().Err(err).Str("id", entryIDStr).Msg("Invalid entry ID")
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid entry id"})
		return
	}

	versions, err := c.Service.GetKnowledgeVersions(r.Context(), entryID)
	if err != nil {
		logger.Error().Err(err).Str("entry_id", entryID.String()).Msg("Failed to get knowledge versions")
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to get knowledge versions"})
		return
	}

	if versions == nil {
		versions = []models.KnowledgeVersion{}
	}

	writeJSONResponse(w, http.StatusOK, versions)
}

// UpdateKnowledgeEntry handles PUT /api/knowledge-entries/{id}
func (c *PromptConfigurationController) UpdateKnowledgeEntry(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	entryIDStr := r.PathValue("id")

	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		logger.Error().Err(err).Str("id", entryIDStr).Msg("Invalid entry ID")
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid entry id"})
		return
	}

	var req updateKnowledgeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error().Err(err).Msg("Failed to decode request body")
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := c.Service.UpdateKnowledgeEntry(r.Context(), entryID, req.Content); err != nil {
		logger.Error().Err(err).Str("entry_id", entryID.String()).Msg("Failed to update knowledge entry")
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to update knowledge entry"})
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "knowledge entry updated"})
}

// UpdateConfiguration handles PUT /api/prompt-configurations/{identifier}
func (c *PromptConfigurationController) UpdateConfiguration(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	identifier := r.PathValue("identifier")

	var configID uuid.UUID

	// Try parsing as UUID first
	if id, parseErr := uuid.Parse(identifier); parseErr == nil {
		configID = id
	} else {
		// If not a UUID, try to fetch by name to get the ID
		cfg, getErr := c.Service.GetConfigurationByName(r.Context(), identifier)
		if getErr != nil {
			logger.Error().Err(getErr).Str("identifier", identifier).Msg("Configuration not found")
			writeJSONResponse(w, http.StatusNotFound, map[string]string{"error": "configuration not found"})
			return
		}
		configID = cfg.ID
	}

	var req updateConfigurationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error().Err(err).Msg("Failed to decode request body")
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.SystemPrompt == nil && req.Metadata == nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "at least one field must be provided"})
		return
	}

	if err := c.Service.UpdateConfiguration(r.Context(), configID, req.SystemPrompt, req.Metadata); err != nil {
		logger.Error().Err(err).Str("config_id", configID.String()).Msg("Failed to update configuration")
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to update configuration"})
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "configuration updated"})
}

// ActivateConfiguration handles POST /api/prompt-configurations/{id}/activate
func (c *PromptConfigurationController) ActivateConfiguration(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	configIDStr := r.PathValue("id")

	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		logger.Error().Err(err).Str("id", configIDStr).Msg("Invalid configuration ID")
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid configuration id"})
		return
	}

	if err := c.Service.ActivateConfiguration(r.Context(), configID); err != nil {
		logger.Error().Err(err).Str("config_id", configID.String()).Msg("Failed to activate configuration")
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to activate configuration"})
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "configuration activated"})
}
