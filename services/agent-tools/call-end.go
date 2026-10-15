package agenttools

import "zennify/zenn-ai-assistant/models"

func CallEndTool() models.AgentTool {
	return models.AgentTool{
		Name: "end_call",
		Description: "Leave the call when you are no longer needed. If you are alone with the caller this ends the call. " +
			"If you have transferred the caller to an agent, or were brought onto a call an agent is already running, " +
			"this only drops you from the call and leaves the others talking to each other. " +
			"Do not use it while someone is still asking you something.",
		Type:        "custom",
		InputSchema: struct{}{},
	}
}
