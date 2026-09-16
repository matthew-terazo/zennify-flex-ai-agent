package models

import "encoding/json"

// TaskRouterEvent represents a Twilio Task Router event
type TaskRouterEvent struct {
	EventType      string          `json:"EventType"`
	TaskSid        string          `json:"TaskSid"`
	WorkspaceSid   string          `json:"WorkspaceSid"`
	AccountSid     string          `json:"AccountSid"`
	Timestamp      string          `json:"Timestamp"`
	TaskAttributes json.RawMessage `json:"TaskAttributes"`
	EventData      json.RawMessage `json:"EventData"`
}

func (m TaskRouterEvent) IsVirtualAgentTask() bool {
	var attributes TaskAttributes

	if err := json.Unmarshal(m.TaskAttributes, &attributes); err != nil {
		return false
	}

	return attributes.IsVirtualAgentTask
}
