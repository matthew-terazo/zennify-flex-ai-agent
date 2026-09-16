package models

type AgentTool struct {
	Name           string
	Description    string
	Type           string
	RequiredFields []string
	InputSchema    any // must be valid json schema
}
