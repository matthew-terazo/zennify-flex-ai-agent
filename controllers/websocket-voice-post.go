package controllers

import (
	"net/http"
	"zennify/zenn-ai-assistant/middleware"
	"zennify/zenn-ai-assistant/models"
	agenttools "zennify/zenn-ai-assistant/services/agent-tools"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type WebsocketVoiceController struct {
	App models.App
}

func (c WebsocketVoiceController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		logger.Err(err).Msg("error upgrading webhook connection")
		return
	}

	// Get the Anthropic agent
	factory, ok := c.App.Agents[models.AgentAnthropic]
	if !ok {
		logger.Warn().Msg("Anthropic agent not configured")
		return // Not an error, just not available
	}

	// Create a fresh agent instance for this request
	agent := factory()
	// Configure the agent
	agent.SetModel(anthropic.ModelClaudeHaiku4_5)

	agent.SetSystemPrompt(`
		You are a helpful customer service assistant in a Twilio Conversation. 
		Keep responses concise, friendly, and professional. 
		Respond directly to user messages without prefacing your response.
		Ask if there is anything else you can do for the customer.
		If the customer says no, thank the customer and then use the end_task tool.
	`)

	agent.SetTools(agenttools.EndTaskTool())

	go func() {
		defer conn.Close()

		for {
			msg, op, err := wsutil.ReadClientData(conn)
			if err != nil {
				logger.Err(err).Msg("error reading webhook message")
			}

			// Feed the user message to the agent
			agent.SetUserMessage(string(msg))

			// Get the AI response
			resp, err := agent.SendMessage(r.Context())
			if err != nil {
				logger.Error().Err(err).Msg("Failed to get response from AI agent")
			}

			logger.Info().
				Any("response", resp).
				Msg("Received response from AI agent")

			for _, message := range resp.Messages {
				if err := wsutil.WriteServerMessage(conn, op, []byte(message)); err != nil {
					logger.Err(err).Msg("error writing webhook message")
					return
				}
				logger.Info().
					Any("messageMeta", msg).
					Msg("Sent AI response to conversation")
			}

			for _, toolCall := range resp.ToolCalls {

				switch toolCall.Name {
				case "end_task":
					// if _, err := c.App.Twilio.EndTask(conversationAttributes.TaskSid); err != nil {
					// 	logger.Err(err).Msg("error ending task")
					// }
					return
				}
			}
		}
	}()
}
