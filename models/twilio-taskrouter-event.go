package models

import (
	"encoding/json"
	"strconv"
)

// TwilioTaskRouterEvent represents a Twilio Task Router event in CloudEvents format
type TwilioTaskRouterEvent struct {
	SpecVersion     string          `json:"specversion"`
	Type            string          `json:"type"`
	Source          string          `json:"source"`
	ID              string          `json:"id"`
	DataSchema      string          `json:"dataschema"`
	DataContentType string          `json:"datacontenttype"`
	Data            TwilioEventData `json:"data"`
}

// TwilioEventData represents the data field of the event
type TwilioEventData struct {
	SID                 string                  `json:"sid"`
	Name                string                  `json:"name"`
	AccountSID          string                  `json:"account_sid"`
	AccountFriendlyName string                  `json:"account_friendly_name"`
	ParentAccountSID    string                  `json:"parent_account_sid"`
	ParentFriendlyName  string                  `json:"parent_friendly_name"`
	Timestamp           string                  `json:"timestamp"`
	Level               string                  `json:"level"`
	Group               string                  `json:"group"`
	ProductName         string                  `json:"product_name"`
	Publisher           string                  `json:"publisher"`
	PublisherMetadata   string                  `json:"publisher_metadata"`
	PayloadType         string                  `json:"payload_type"`
	Payload             TwilioTaskRouterPayload `json:"payload"`
}

// TwilioTaskRouterPayload represents the payload field containing the actual event data
type TwilioTaskRouterPayload struct {
	SID                              string          `json:"sid"`
	EventType                        string          `json:"eventtype"`
	AccountSID                       string          `json:"account_sid"`
	Timestamp                        string          `json:"timestamp"`
	WorkspaceSID                     string          `json:"workspace_sid"`
	WorkspaceName                    string          `json:"workspace_name"`
	TaskSID                          string          `json:"task_sid"`
	TaskAttributes                   json.RawMessage `json:"task_attributes"`
	TaskAge                          int             `json:"task_age"`
	TaskDateCreated                  string          `json:"task_date_created"`
	TaskPriority                     int             `json:"task_priority"`
	TaskAssignmentStatus             string          `json:"task_assignment_status"`
	TaskCompletedReason              string          `json:"task_completed_reason"`
	TaskCanceledReason               string          `json:"task_canceled_reason"`
	Reason                           string          `json:"reason"`
	TaskChannelSID                   string          `json:"task_channel_sid"`
	TaskChannelUniqueName            string          `json:"task_channel_unique_name"`
	ReservationSID                   string          `json:"reservation_sid"`
	WorkerSID                        string          `json:"worker_sid"`
	WorkerName                       string          `json:"worker_name"`
	WorkerAttributes                 json.RawMessage `json:"worker_attributes"`
	WorkerActivitySID                string          `json:"worker_activity_sid"`
	WorkerActivityName               string          `json:"worker_activity_name"`
	WorkerTimeInPreviousActivity     int             `json:"worker_time_in_previous_activity"`
	WorkerTimeInPreviousActivityMs   int             `json:"worker_time_in_previous_activity_ms"`
	WorkerPreviousActivitySID        string          `json:"worker_previous_activity_sid"`
	WorkerPreviousActivityName       string          `json:"worker_previous_activity_name"`
	WorkflowSID                      string          `json:"workflow_sid"`
	WorkflowName                     string          `json:"workflow_name"`
	TaskQueueSID                     string          `json:"task_queue_sid"`
	TaskQueueName                    string          `json:"task_queue_name"`
	WorkerChannelAvailable           int             `json:"worker_channel_available"`
	WorkerChannelCapacity            int             `json:"worker_channel_capacity"`
	WorkerChannelAvailableCapacity   float64         `json:"worker_channel_available_capacity"`
	WorkerChannelPreviousCapacity    float64         `json:"worker_channel_previous_capacity"`
	WorkerChannelTaskCount           int             `json:"worker_channel_task_count"`
	WorkflowFilterName               string          `json:"workflow_filter_name"`
	WorkflowFilterExpression         string          `json:"workflow_filter_expression"`
	WorkflowFilterTargetExpression   string          `json:"workflow_filter_target_expression"`
	WorkflowFilterTargetName         string          `json:"workflow_filter_target_name"`
	PreviousTaskQueueSID             string          `json:"previous_task_queue_sid"`
	PreviousTaskQueueName            string          `json:"previous_task_queue_name"`
	PreviousTaskPriority             int             `json:"previous_task_priority"`
	TargetChangedReason              string          `json:"target_changed_reason"`
	TaskReEvaluatedReason            string          `json:"task_re_evaluated_reason"`
	TaskQueueTargetExpression        string          `json:"task_queue_target_expression"`
	TaskTransferSID                  string          `json:"task_transfer_sid"`
	TransferType                     string          `json:"transfer_type"`
	TransferInitiatingWorkerSID      string          `json:"transfer_initiating_worker_sid"`
	TransferInitiatingReservationSID string          `json:"transfer_initiating_reservation_sid"`
	TransferTo                       string          `json:"transfer_to"`
	TransferMode                     string          `json:"transfer_mode"`
	TransferStarted                  string          `json:"transfer_started"`
	TransferFailedReason             string          `json:"transfer_failed_reason"`
	TransferStatus                   string          `json:"transfer_status"`
	EventDescription                 string          `json:"event_description"`
	ResourceType                     string          `json:"resource_type"`
	ResourceSID                      string          `json:"resource_sid"`
	TaskRoutingTarget                string          `json:"task_routing_target"`
	ReservationReasonCode            int             `json:"reservation_reason_code"`
	TaskAgeInQueue                   int             `json:"task_age_in_queue"`
	TaskQueueEnteredDate             string          `json:"task_queue_entered_date"`
	TaskVersion                      int             `json:"task_version"`
	ReservationVersion               int             `json:"reservation_version"`
	TaskVirtualStartTime             string          `json:"task_virtual_start_time"`
	WorkerChannelConsumedCapacity    int             `json:"worker_channel_consumed_capacity"`
	WorkerVersion                    int             `json:"worker_version"`
	TaskChannelName                  string          `json:"task_channel_name"`
	TaskChannelOptimizedRouting      bool            `json:"task_channel_optimized_routing"`
	KnownWorkerSID                   string          `json:"known_worker_sid"`
	ActivitySID                      string          `json:"activity_sid"`
	ActivityName                     string          `json:"activity_name"`
	ActivityAvailable                bool            `json:"activity_available"`
	OperatingUnitSID                 string          `json:"operating_unit_sid"`
	OperatingUnitFriendlyName        string          `json:"operating_unit_friendly_name"`
	OperatingUnitDateCreated         string          `json:"operating_unit_date_created"`
	OperatingUnitDateUpdated         string          `json:"operating_unit_date_updated"`
	TaskAddOnAttributes              string          `json:"task_add_on_attributes"`
	TaskTimeout                      int             `json:"task_timeout"`
	WorkerAvailable                  bool            `json:"worker_available"`
	ReservationTimeout               int             `json:"reservation_timeout"`
	ReservationStatus                string          `json:"reservation_status"`
}

type TaskAttributes struct {
	IsVirtualAgentTask bool   `json:"isVirtualAgentTask"`
	ConversationSid    string `json:"conversationSid"`
	CallSid            string `json:"call_sid"`
}

type WorkerAttributes struct {
	IsVirtualAgent bool `json:"isVirtualAgent"`
}

func (m TwilioTaskRouterPayload) IsAssignedToVirtualAgent() (bool, error) {
	var attributes WorkerAttributes

	s, err := strconv.Unquote(string(m.WorkerAttributes))
	if err != nil {
		return false, err
	}

	if err := json.Unmarshal([]byte(s), &attributes); err != nil {
		return attributes.IsVirtualAgent, err
	}

	return attributes.IsVirtualAgent, nil
}

func (m TwilioTaskRouterPayload) IsVirtualAgentTask() (bool, error) {
	var attributes TaskAttributes

	s, err := strconv.Unquote(string(m.TaskAttributes))
	if err != nil {
		return false, err
	}

	if err := json.Unmarshal([]byte(s), &attributes); err != nil {
		return attributes.IsVirtualAgentTask, err
	}

	return attributes.IsVirtualAgentTask, nil
}

func (m TwilioTaskRouterPayload) GetCallSid() (string, error) {
	var attributes TaskAttributes

	s, err := strconv.Unquote(string(m.TaskAttributes))
	if err != nil {
		return attributes.CallSid, err
	}

	if err := json.Unmarshal([]byte(s), &attributes); err != nil {
		return attributes.CallSid, err
	}

	return attributes.CallSid, nil
}

func (m TwilioTaskRouterPayload) GetConversationSid() (string, error) {
	var attributes TaskAttributes

	s, err := strconv.Unquote(string(m.TaskAttributes))
	if err != nil {
		return attributes.ConversationSid, err
	}

	if err := json.Unmarshal([]byte(s), &attributes); err != nil {
		return attributes.ConversationSid, err
	}

	return attributes.ConversationSid, nil
}
