package agenttools

import "zennify/zenn-ai-assistant/models"

func CallEndTool() models.AgentTool {
	return models.AgentTool{
		Name:        "end_call",
		Description: "End the call when the conversation is over",
		Type:        "custom",
		InputSchema: struct{}{},
	}
}
