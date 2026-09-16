package controllers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"zennify/zenn-ai-assistant/middleware"
	"zennify/zenn-ai-assistant/models"
	agenttools "zennify/zenn-ai-assistant/services/agent-tools"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/rs/zerolog"
	conversations "github.com/twilio/twilio-go/rest/conversations/v1"
)

type ConversationEventController struct {
	App models.App
}

// ServeHTTP handles incoming Twilio Conversations webhooks.
// For onMessageAdded events, it feeds the message to an AI agent and sends back the response.
func (c ConversationEventController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error().Err(err).Msg("error reading message body")
		writeJSONResponse(w, http.StatusBadRequest, nil)
		return
	}
	defer r.Body.Close()

	event, err := models.ParseTwilioConversationsWebhook(string(requestBody))
	if err != nil {
		logger.Error().Err(err).Msg("error parsing message body")
		writeJSONResponse(w, http.StatusBadRequest, nil)
		return
	}

	logger.Info().
		Str("event_type", event.EventType).
		Str("conversation_sid", event.ConversationSid).
		Str("author", event.Author).
		Msg("Received conversation event")

	// Only process onMessageAdded events
	if event.EventType != "onMessageAdded" {
		writeJSONResponse(w, http.StatusNoContent, nil)
		return
	}

	// Process the message with an AI agent
	if err := c.processMessage(r.Context(), logger, event); err != nil {
		logger.Error().Err(err).Msg("Failed to process conversation message")
		// Still return 204 to prevent Twilio from retrying
		writeJSONResponse(w, http.StatusNoContent, nil)
		return
	}

	writeJSONResponse(w, http.StatusNoContent, nil)
}

// processMessage feeds the message to an AI agent and sends the response back to the conversation.
func (c ConversationEventController) processMessage(ctx context.Context, logger zerolog.Logger, webhook models.TwilioConversationsWebhook) error {
	var conversationAttributes models.TwilioConversationAttributes

	conversation, err := c.App.Twilio.FetchConversation(webhook.ConversationSid)
	if err != nil {
		logger.Error().Err(err).Msg("error retrieving conversation")
		return err
	}

	if err := json.Unmarshal([]byte(*conversation.Attributes), &conversationAttributes); err != nil {
		logger.Err(err).Msg("error unmarshalling conversation attributes")
		return err
	}

	assistant, err := c.App.PromptConfigService.GetActiveConfiguration(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("error getting active assistant")
		return err
	}

	// Get the Anthropic agent
	factory, ok := c.App.Agents[models.AgentAnthropic]
	if !ok {
		logger.Warn().Msg("Anthropic agent not configured")
		return nil // Not an error, just not available
	}

	// Create a fresh agent instance for this request
	agent := factory()

	// Configure the agent
	agent.SetModel(anthropic.ModelClaudeHaiku4_5)
	agent.SetSystemPrompt(*assistant.SystemPrompt)

	// Feed the user message to the agent
	agent.SetUserMessage(webhook.Body)
	agent.SetTools(agenttools.EndTaskTool(), agenttools.CloseConversationTool())

	// Get the AI response
	resp, err := agent.SendMessage(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get response from AI agent")
		return err
	}

	logger.Info().
		Any("response", resp).
		Msg("Received response from AI agent")

	for _, message := range resp.Messages {

		// Send the response back to the conversation
		params := conversations.CreateConversationMessageParams{}
		params.SetBody(message)
		params.SetAuthor("bot")

		msg, err := c.App.Twilio.SendConversationMessage(webhook.ConversationSid, params)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to send message to conversation")
			return err
		}

		logger.Info().
			Any("messageMeta", msg).
			Str("conversation_sid", webhook.ConversationSid).
			Msg("Sent AI response to conversation")
	}

	for _, toolCall := range resp.ToolCalls {

		switch toolCall.Name {
		case "end_task":
			if _, err := c.App.Twilio.EndTask(conversationAttributes.TaskSid); err != nil {
				logger.Err(err).Msg("error ending task")
				return err
			}
		case "close_conversation":
			if _, err := c.App.Twilio.CloseConversation(webhook.ConversationSid); err != nil {
				logger.Err(err).Msg("error closing conversation")
				return err
			}
		}
	}

	return nil
}
