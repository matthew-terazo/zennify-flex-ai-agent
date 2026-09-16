package agenttools

import "zennify/zenn-ai-assistant/models"

func EndTaskTool() models.AgentTool {
	return models.AgentTool{
		Name:        "end_task",
		Description: "End the task when the conversation is over",
		Type:        "custom",
		InputSchema: struct{}{},
	}
}
