package models

import (
	"context"
)

// Agent registry keys. Use these instead of raw strings to avoid typos.
const (
	AgentAnthropic = "anthropic"
)

// AiBotFactory returns a fresh AiBotService instance. Storing factories
// (rather than a shared instance) keeps each request's message/system state
// isolated, which is required because AiBotService implementations are stateful.
type AiBotFactory func() AiBotService

type AiBotService interface {
	SetModel(modelName string)
	SetSystemPrompt(prompt string)
	SetSystemPrompts(prompts ...string)
	SetUserMessage(message string)
	SetUserMessages(messages ...string)
	SetAssistantMessage(message string)
	SetAssistantMessages(messages ...string)
	SetTools(agentTools ...AgentTool)
	SendMessage(ctx context.Context) (AiBotResponse, error)
}

type AiBotToolCall struct {
	ID     string
	Name   string
	Inputs any
}

type AiBotResponse struct {
	ID        string
	Messages  []string
	ToolCalls []AiBotToolCall
}
