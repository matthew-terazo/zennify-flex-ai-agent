package agenttools

import (
	"zennify/zenn-ai-assistant/models"
)

type QueueTransferInputSchema struct {
	TargetQueue string `json:"target_queue"`
	Summary     string `json:"summary"`
	Reason      string `json:"reason"`
}

func QueueTransferTool() models.AgentTool {
	return models.AgentTool{
		Name:           "transfer_call",
		Description:    "Warm transfer the caller to a live agent in a Task Router queue. The caller stays on the line and you stay with them until an agent joins, at which point you introduce the caller to the agent and then leave the call.",
		Type:           "custom",
		RequiredFields: []string{"target_queue", "summary"},
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"target_queue", "summary"},
			"properties": map[string]any{
				"target_queue": map[string]any{
					"type":        "string",
					"enum":        []string{models.TransferQueueEveryone},
					"description": "The name of the queue that represents where the caller would like to be transferred",
				},
				"summary": map[string]any{
					"type":        "string",
					"description": "A brief handoff summary for the agent who picks up: who is calling, what they need, and anything already tried or promised. Two or three sentences, written to be read aloud to the agent.",
				},
				"reason": map[string]any{
					"type":        "string",
					"description": "Short reason the call is being transferred",
				},
			},
		},
	}
}
