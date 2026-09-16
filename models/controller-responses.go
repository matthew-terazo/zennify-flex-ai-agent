package models

import "encoding/json"

type ResponseJSON interface {
	MarshalJSON() ([]byte, error)
}

type ResponseError struct {
	Errors []string `json:"errors"`
}

func (m *ResponseError) AddErrorMessage(errorMessage string) {
	m.Errors = append(m.Errors, errorMessage)
}

func (m *ResponseError) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}

type TaskQueueCallbackResponse struct {
	Instruction         string `json:"instruction"`
	PostWorkActivitySid string `json:"post_work_activity_sid"`
	From                string `json:"from"`
}

// func (m *TaskQueueCallbackResponse) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(m)
// }
