package agenttools

import "zennify/zenn-ai-assistant/models"

func CloseConversationTool() models.AgentTool {
	return models.AgentTool{
		Name:        "close_conversation",
		Description: "End the conversation when the conversation is over",
		Type:        "custom",
		InputSchema: struct{}{},
	}
}
