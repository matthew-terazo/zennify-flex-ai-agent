package controllers

import (
	"net/http"
	"zennify/zenn-ai-assistant/models"
)

type TaskRouterQueueAssignmentCallbackController struct {
	App models.App
}

func (c TaskRouterQueueAssignmentCallbackController) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	response := models.TaskQueueCallbackResponse{
		Instruction:         "dequeue",
		PostWorkActivitySid: "WA1a65e9e0f4f271d8c0d8ea5cc831ef9d",
		From:                "+17039975700",
	}

	writeJSONResponse(w, http.StatusOK, &response)
}
