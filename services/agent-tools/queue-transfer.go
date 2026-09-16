package agenttools

import (
	"zennify/zenn-ai-assistant/models"
)

type QueueTransferInputSchema struct {
	TargetQueue string `json:"target_queue"`
	Reason      string `json:"reason"`
}

func QueueTransferTool() models.AgentTool {
	return models.AgentTool{
		Name:           "transfer_call",
		Description:    "Transfer the call to a different queue in Task Router",
		Type:           "custom",
		RequiredFields: []string{"target_queue"},
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"target_queue"},
			"properties": map[string]any{
				"target_queue": map[string]any{
					"type":        "string",
					"enum":        []string{"Everyone"},
					"description": "The name of the queue that represents where the caller wuld like to be transfered",
				},
			},
		},
	}
}
