package controllers

import (
	"encoding/json"
	"errors"
)

// RealtimeMessage represents a parsed OpenAI Realtime API message
type RealtimeMessage struct {
	Type  string          `json:"type"`
	Event RealtimeEvent   `json:"event,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// RealtimeEvent represents event details in a Realtime message
type RealtimeEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// ToolCall represents a tool call request from OpenAI
type ToolCall struct {
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	ID        string                 `json:"id"`
	Arguments map[string]interface{} `json:"arguments"`
}

// FunctionCallArgumentsDelta represents partial function call arguments
type FunctionCallArgumentsDelta struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta string `json:"delta"`
}

// ResponseFunctionCall represents a complete function call in a response
type ResponseFunctionCall struct {
	Type        string `json:"type"`
	EventID     string `json:"event_id"`
	ResponseID  string `json:"response_id"`
	ItemID      string `json:"item_id"`
	OutputIndex int    `json:"output_index"`
	CallID      string `json:"call_id"`
	Name        string `json:"name"`
	Arguments   string `json:"arguments"`
}

// ParseRealtimeMessage parses raw websocket data into a RealtimeMessage
func ParseRealtimeMessage(data []byte) (*RealtimeMessage, error) {
	var msg RealtimeMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// IsToolCallRequest checks if a message contains a tool call request
func IsToolCallRequest(msg *RealtimeMessage) bool {
	if msg == nil {
		return false
	}

	// Check for function call argument deltas (streaming tool calls)
	if msg.Type == "response.function_call_arguments.delta" {
		return true
	}

	// Check for complete function call in response
	if msg.Type == "response.function_call_arguments.done" {
		return true
	}

	// Check event-based tool calls
	if msg.Event.Type != "" {
		switch msg.Event.Type {
		case "function_call_arguments.delta", "function_call":
			return true
		}
	}

	return false
}

// ExtractToolCallFromDelta extracts tool call info from streaming arguments delta
func ExtractToolCallFromDelta(data []byte) (*FunctionCallArgumentsDelta, error) {
	var delta FunctionCallArgumentsDelta
	if err := json.Unmarshal(data, &delta); err != nil {
		return nil, err
	}
	return &delta, nil
}

// ExtractToolCall extracts tool call info from a complete function call
func ExtractToolCall(data []byte) (*ResponseFunctionCall, error) {
	var call ResponseFunctionCall
	if err := json.Unmarshal(data, &call); err != nil {
		return nil, err
	}

	if call.Name == "" {
		return nil, errors.New("tool call missing function name")
	}

	return &call, nil
}

// ParseToolCallArguments parses the JSON arguments string from a tool call
func ParseToolCallArguments(argumentsJSON string) (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return nil, err
	}
	return args, nil
}

// IsResponseDone checks if a message indicates the response is complete
func IsResponseDone(msg *RealtimeMessage) bool {
	if msg == nil {
		return false
	}

	return msg.Type == "response.done" || msg.Type == "response.completed"
}

// IsInputAudioBufferCommitted checks if audio input buffer was committed
func IsInputAudioBufferCommitted(msg *RealtimeMessage) bool {
	if msg == nil {
		return false
	}

	return msg.Type == "input_audio_buffer.committed"
}
