package controllers

import (
	"testing"
)

func TestExtractToolCall(t *testing.T) {
	// Sample response.function_call_arguments.done event from OpenAI
	sampleJSON := "{\"type\":\"response.function_call_arguments.done\",\"event_id\":\"event_DuP0kBoYpxuz7yxawKqBF\",\"response_id\":\"resp_DuP0kZFcIarUesa6wvfFg\",\"item_id\":\"item_DuP0keWoB4ZwIGqXcF9NV\",\"output_index\":0,\"call_id\":\"call_pNuPTkIfIlNGRj6Y\",\"name\":\"end_call\",\"arguments\":\"{}\"}"

	jsonBytes := []byte(sampleJSON)

	toolCall, err := ExtractToolCall(jsonBytes)
	if err != nil {
		t.Fatalf("ExtractToolCall failed: %v", err)
	}

	// Verify all fields are correctly parsed
	if toolCall.Type != "response.function_call_arguments.done" {
		t.Errorf("expected type 'response.function_call_arguments.done', got '%s'", toolCall.Type)
	}

	if toolCall.EventID != "event_DuP0kBoYpxuz7yxawKqBF" {
		t.Errorf("expected event_id 'event_DuP0kBoYpxuz7yxawKqBF', got '%s'", toolCall.EventID)
	}

	if toolCall.ResponseID != "resp_DuP0kZFcIarUesa6wvfFg" {
		t.Errorf("expected response_id 'resp_DuP0kZFcIarUesa6wvfFg', got '%s'", toolCall.ResponseID)
	}

	if toolCall.ItemID != "item_DuP0keWoB4ZwIGqXcF9NV" {
		t.Errorf("expected item_id 'item_DuP0keWoB4ZwIGqXcF9NV', got '%s'", toolCall.ItemID)
	}

	if toolCall.OutputIndex != 0 {
		t.Errorf("expected output_index 0, got %d", toolCall.OutputIndex)
	}

	if toolCall.CallID != "call_pNuPTkIfIlNGRj6Y" {
		t.Errorf("expected call_id 'call_pNuPTkIfIlNGRj6Y', got '%s'", toolCall.CallID)
	}

	if toolCall.Name != "end_call" {
		t.Errorf("expected name 'end_call', got '%s'", toolCall.Name)
	}

	if toolCall.Arguments != "{}" {
		t.Errorf("expected arguments '{}', got '%s'", toolCall.Arguments)
	}
}

func TestExtractToolCallWithArguments(t *testing.T) {
	// Test with non-empty arguments
	sampleJSON := "{\"type\":\"response.function_call_arguments.done\",\"event_id\":\"event_DuP0kBoYpxuz7yxawKqBF\",\"response_id\":\"resp_DuP0kZFcIarUesa6wvfFg\",\"item_id\":\"item_DuP0keWoB4ZwIGqXcF9NV\",\"output_index\":0,\"call_id\":\"call_pNuPTkIfIlNGRj6Y\",\"name\":\"end_call\",\"arguments\":\"{}\"}"

	jsonBytes := []byte(sampleJSON)

	toolCall, err := ExtractToolCall(jsonBytes)
	if err != nil {
		t.Fatalf("ExtractToolCall failed: %v", err)
	}

	if toolCall.Name != "transfer_call" {
		t.Errorf("expected name 'transfer_call', got '%s'", toolCall.Name)
	}

	// Verify arguments can be parsed
	args, err := ParseToolCallArguments(toolCall.Arguments)
	if err != nil {
		t.Fatalf("ParseToolCallArguments failed: %v", err)
	}

	destination, ok := args["destination"].(string)
	if !ok || destination != "+14155552671" {
		t.Errorf("expected destination '+14155552671', got '%v'", destination)
	}

	reason, ok := args["reason"].(string)
	if !ok || reason != "agent transfer" {
		t.Errorf("expected reason 'agent transfer', got '%v'", reason)
	}
}

func TestExtractToolCallMissingName(t *testing.T) {
	// Test error handling for missing name
	sampleJSON := `{
		"type": "response.function_call_arguments.done",
		"event_id": "event_test123",
		"response_id": "resp_test123",
		"item_id": "item_test123",
		"output_index": 0,
		"call_id": "call_test123",
		"arguments": "{}"
	}`

	jsonBytes := []byte(sampleJSON)

	_, err := ExtractToolCall(jsonBytes)
	if err == nil {
		t.Fatal("expected error for missing function name, got nil")
	}

	if err.Error() != "tool call missing function name" {
		t.Errorf("expected 'tool call missing function name', got '%s'", err.Error())
	}
}

func TestIsToolCallRequest(t *testing.T) {
	tests := []struct {
		name     string
		msg      *RealtimeMessage
		expected bool
	}{
		{
			name: "response.function_call_arguments.delta",
			msg: &RealtimeMessage{
				Type: "response.function_call_arguments.delta",
			},
			expected: true,
		},
		{
			name: "response.function_call_arguments.done",
			msg: &RealtimeMessage{
				Type: "response.function_call_arguments.done",
			},
			expected: true,
		},
		{
			name: "response.done",
			msg: &RealtimeMessage{
				Type: "response.done",
			},
			expected: false,
		},
		{
			name:     "nil message",
			msg:      nil,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsToolCallRequest(test.msg)
			if result != test.expected {
				t.Errorf("expected %v, got %v", test.expected, result)
			}
		})
	}
}

func TestParseRealtimeMessage(t *testing.T) {
	sampleJSON := `{
		"type": "response.function_call_arguments.done",
		"event_id": "event_test123",
		"call_id": "call_test123",
		"name": "end_call",
		"arguments": "{}"
	}`

	jsonBytes := []byte(sampleJSON)

	msg, err := ParseRealtimeMessage(jsonBytes)
	if err != nil {
		t.Fatalf("ParseRealtimeMessage failed: %v", err)
	}

	if msg.Type != "response.function_call_arguments.done" {
		t.Errorf("expected type 'response.function_call_arguments.done', got '%s'", msg.Type)
	}
}

func TestParseToolCallArgumentsWithExtractedToolCall(t *testing.T) {
	// First extract a tool call with arguments
	sampleJSON := `{
		"type": "response.function_call_arguments.done",
		"event_id": "event_test123",
		"response_id": "resp_test123",
		"item_id": "item_test123",
		"output_index": 0,
		"call_id": "call_test123",
		"name": "transfer_call",
		"arguments": "{\"destination\": \"+14155552671\", \"reason\": \"customer request\", \"priority\": 1}"
	}`

	// Extract the tool call
	toolCall, err := ExtractToolCall([]byte(sampleJSON))
	if err != nil {
		t.Fatalf("ExtractToolCall failed: %v", err)
	}

	// Now parse the arguments from the extracted tool call
	args, err := ParseToolCallArguments(toolCall.Arguments)
	if err != nil {
		t.Fatalf("ParseToolCallArguments failed: %v", err)
	}

	// Verify arguments
	if destination, ok := args["destination"].(string); !ok || destination != "+14155552671" {
		t.Errorf("expected destination '+14155552671', got '%v'", destination)
	}

	if reason, ok := args["reason"].(string); !ok || reason != "customer request" {
		t.Errorf("expected reason 'customer request', got '%v'", reason)
	}

	if priority, ok := args["priority"].(float64); !ok || priority != 1 {
		t.Errorf("expected priority 1.0, got '%v'", priority)
	}
}

func TestParseToolCallArgumentsEmptyObject(t *testing.T) {
	// Extract tool call with empty arguments
	sampleJSON := `{
		"type": "response.function_call_arguments.done",
		"event_id": "event_test123",
		"response_id": "resp_test123",
		"item_id": "item_test123",
		"output_index": 0,
		"call_id": "call_test123",
		"name": "end_call",
		"arguments": "{}"
	}`

	toolCall, err := ExtractToolCall([]byte(sampleJSON))
	if err != nil {
		t.Fatalf("ExtractToolCall failed: %v", err)
	}

	args, err := ParseToolCallArguments(toolCall.Arguments)
	if err != nil {
		t.Fatalf("ParseToolCallArguments failed: %v", err)
	}

	if len(args) != 0 {
		t.Errorf("expected empty map, got %d entries", len(args))
	}
}

func TestParseToolCallArgumentsComplexTypes(t *testing.T) {
	// Extract tool call with various data types
	sampleJSON := `{
		"type": "response.function_call_arguments.done",
		"event_id": "event_test123",
		"response_id": "resp_test123",
		"item_id": "item_test123",
		"output_index": 0,
		"call_id": "call_test123",
		"name": "complex_call",
		"arguments": "{\"string_val\": \"hello\", \"number_val\": 42, \"float_val\": 3.14, \"bool_val\": true, \"null_val\": null, \"array_val\": [1, 2, 3], \"object_val\": {\"nested\": \"value\"}}"
	}`

	toolCall, err := ExtractToolCall([]byte(sampleJSON))
	if err != nil {
		t.Fatalf("ExtractToolCall failed: %v", err)
	}

	args, err := ParseToolCallArguments(toolCall.Arguments)
	if err != nil {
		t.Fatalf("ParseToolCallArguments failed: %v", err)
	}

	// Test string
	if str, ok := args["string_val"].(string); !ok || str != "hello" {
		t.Errorf("expected string 'hello', got '%v'", args["string_val"])
	}

	// Test number
	if num, ok := args["number_val"].(float64); !ok || num != 42 {
		t.Errorf("expected number 42.0, got '%v'", args["number_val"])
	}

	// Test float
	if flt, ok := args["float_val"].(float64); !ok || flt != 3.14 {
		t.Errorf("expected float 3.14, got '%v'", args["float_val"])
	}

	// Test boolean
	if bln, ok := args["bool_val"].(bool); !ok || bln != true {
		t.Errorf("expected bool true, got '%v'", args["bool_val"])
	}

	// Test null
	if args["null_val"] != nil {
		t.Errorf("expected nil for null_val, got '%v'", args["null_val"])
	}

	// Test array
	if arr, ok := args["array_val"].([]interface{}); !ok || len(arr) != 3 {
		t.Errorf("expected array with 3 elements, got '%v'", args["array_val"])
	}

	// Test nested object
	if obj, ok := args["object_val"].(map[string]interface{}); !ok {
		t.Errorf("expected nested object, got '%v'", args["object_val"])
	} else if nested, ok := obj["nested"].(string); !ok || nested != "value" {
		t.Errorf("expected nested value 'value', got '%v'", nested)
	}
}

func TestParseToolCallArgumentsMultipleToolCalls(t *testing.T) {
	toolCalls := []struct {
		name        string
		jsonPayload string
		expectedKey string
		expectedVal interface{}
	}{
		{
			name: "end_call tool",
			jsonPayload: `{
				"type": "response.function_call_arguments.done",
				"call_id": "call_1",
				"name": "end_call",
				"arguments": "{}"
			}`,
			expectedKey: "no_args",
			expectedVal: nil,
		},
		{
			name: "transfer_call tool",
			jsonPayload: `{
				"type": "response.function_call_arguments.done",
				"call_id": "call_2",
				"name": "transfer_call",
				"arguments": "{\"destination\": \"+15551234567\", \"queue\": \"sales\"}"
			}`,
			expectedKey: "destination",
			expectedVal: "+15551234567",
		},
		{
			name: "note_call tool",
			jsonPayload: `{
				"type": "response.function_call_arguments.done",
				"call_id": "call_3",
				"name": "note_call",
				"arguments": "{\"note\": \"Customer requested callback\", \"priority\": 2}"
			}`,
			expectedKey: "note",
			expectedVal: "Customer requested callback",
		},
	}

	for _, tc := range toolCalls {
		t.Run(tc.name, func(t *testing.T) {
			toolCall, err := ExtractToolCall([]byte(tc.jsonPayload))
			if err != nil {
				t.Fatalf("ExtractToolCall failed: %v", err)
			}

			args, err := ParseToolCallArguments(toolCall.Arguments)
			if err != nil {
				t.Fatalf("ParseToolCallArguments failed: %v", err)
			}

			if tc.expectedKey == "no_args" {
				if len(args) != 0 {
					t.Errorf("expected no arguments, got %v", args)
				}
			} else {
				if val, ok := args[tc.expectedKey]; !ok || val != tc.expectedVal {
					t.Errorf("expected %s=%v, got %v", tc.expectedKey, tc.expectedVal, val)
				}
			}
		})
	}
}

func TestParseToolCallArgumentsPreservesTypes(t *testing.T) {
	// Test that ParseToolCallArguments preserves JSON types correctly
	sampleJSON := `{
		"type": "response.function_call_arguments.done",
		"call_id": "call_test",
		"name": "typed_call",
		"arguments": "{\"port\": 8080, \"enabled\": false, \"tags\": [\"urgent\", \"vip\"]}"
	}`

	toolCall, err := ExtractToolCall([]byte(sampleJSON))
	if err != nil {
		t.Fatalf("ExtractToolCall failed: %v", err)
	}

	args, err := ParseToolCallArguments(toolCall.Arguments)
	if err != nil {
		t.Fatalf("ParseToolCallArguments failed: %v", err)
	}

	// Verify port is a number, not a string
	port, ok := args["port"].(float64)
	if !ok {
		t.Fatalf("expected port to be float64, got %T", args["port"])
	}
	if port != 8080 {
		t.Errorf("expected port 8080, got %v", port)
	}

	// Verify enabled is a boolean, not a string
	enabled, ok := args["enabled"].(bool)
	if !ok {
		t.Fatalf("expected enabled to be bool, got %T", args["enabled"])
	}
	if enabled != false {
		t.Errorf("expected enabled false, got %v", enabled)
	}

	// Verify tags is an array
	tags, ok := args["tags"].([]interface{})
	if !ok {
		t.Fatalf("expected tags to be []interface{}, got %T", args["tags"])
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestParseToolCallArguments(t *testing.T) {
	argumentsJSON := `{"destination": "+14155552671", "reason": "customer request", "priority": 1}`

	args, err := ParseToolCallArguments(argumentsJSON)
	if err != nil {
		t.Fatalf("ParseToolCallArguments failed: %v", err)
	}

	if destination, ok := args["destination"].(string); !ok || destination != "+14155552671" {
		t.Errorf("expected destination '+14155552671', got '%v'", destination)
	}

	if reason, ok := args["reason"].(string); !ok || reason != "customer request" {
		t.Errorf("expected reason 'customer request', got '%v'", reason)
	}

	if priority, ok := args["priority"].(float64); !ok || priority != 1 {
		t.Errorf("expected priority 1.0, got '%v'", priority)
	}
}

func TestParseToolCallArgumentsInvalid(t *testing.T) {
	invalidJSON := `{invalid json}`

	_, err := ParseToolCallArguments(invalidJSON)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func BenchmarkExtractToolCall(b *testing.B) {
	sampleJSON := []byte(`{
		"type": "response.function_call_arguments.done",
		"event_id": "event_test123",
		"response_id": "resp_test123",
		"item_id": "item_test123",
		"output_index": 0,
		"call_id": "call_test123",
		"name": "end_call",
		"arguments": "{}"
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractToolCall(sampleJSON)
	}
}
