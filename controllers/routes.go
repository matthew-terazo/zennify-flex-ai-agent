package controllers

import (
	"zennify/zenn-ai-assistant/middleware"
	"zennify/zenn-ai-assistant/models"
)

func RegisterRoutes(app models.App) {
	twilioAuth := middleware.TwilioAuth(app.Config.Twilio.AuthToken)

	// Conversation message route
	// Middleware order (left-to-right): RequestID (always first) → TwilioAuth → Logging → handler
	app.Mux.Handle("POST /twilio/conversation/event",
		middleware.Chain(ConversationEventController{App: app},
			twilioAuth,
			middleware.Logging))

	// Task Router events route
	// Middleware order (left-to-right): RequestID (always first) → TwilioAuth → Logging → handler
	app.Mux.Handle("POST /twilio/taskrouter/event",
		middleware.Chain(TaskRouterEventController{App: app},
			twilioAuth,
			middleware.Logging))

	app.Mux.Handle("POST /twilio/voice/websocket",
		middleware.Chain(WebsocketVoiceController{App: app},
			twilioAuth,
			middleware.Logging))

	app.Mux.Handle("POST /twilio/taskrouter/queue-assignment-callback",
		middleware.Chain(TaskRouterQueueAssignmentCallbackController{App: app},
			twilioAuth,
			middleware.Logging))

	// Conference status callbacks; drives the hand-off step of a warm transfer
	app.Mux.Handle("POST /twilio/conference/event",
		middleware.Chain(ConferenceEventController{App: app},
			twilioAuth,
			middleware.Logging))

	app.Mux.Handle("POST /openai/events/realtime-call-incoming",
		middleware.Chain(OpenAIEventController{App: app},
			middleware.Logging))

	// Prompt configuration routes
	promptConfigController := &PromptConfigurationController{Service: app.PromptConfigService}

	app.Mux.Handle("GET /api/prompt-configurations",
		middleware.Chain(
			listConfigurationsHandler{controller: promptConfigController},
			middleware.Logging))

	app.Mux.Handle("GET /api/prompt-configurations/{name}",
		middleware.Chain(getConfigurationByNameHandler{controller: promptConfigController},
			middleware.Logging))

	app.Mux.Handle("POST /api/prompt-configurations",
		middleware.Chain(createConfigurationHandler{controller: promptConfigController},
			middleware.Logging))

	app.Mux.Handle("GET /api/prompt-configurations/{id}/knowledge-entries",
		middleware.Chain(
			getKnowledgeEntriesHandler{controller: promptConfigController},
			middleware.Logging))

	app.Mux.Handle("POST /api/prompt-configurations/{id}/knowledge-entries",
		middleware.Chain(addKnowledgeEntryHandler{controller: promptConfigController},
			middleware.Logging))

	app.Mux.Handle("GET /api/knowledge-entries/{id}/versions",
		middleware.Chain(getKnowledgeVersionsHandler{controller: promptConfigController},
			middleware.Logging))

	app.Mux.Handle("PUT /api/knowledge-entries/{id}",
		middleware.Chain(updateKnowledgeEntryHandler{controller: promptConfigController},
			middleware.Logging))

	app.Mux.Handle("PUT /api/prompt-configurations/{identifier}",
		middleware.Chain(updateConfigurationHandler{controller: promptConfigController},
			middleware.Logging))

	app.Mux.Handle("POST /api/prompt-configurations/{id}/activate",
		middleware.Chain(activateConfigurationHandler{controller: promptConfigController},
			middleware.Logging))
}
