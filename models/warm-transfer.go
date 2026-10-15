package models

import "encoding/json"

// TransferQueueEveryone is the Task Router queue the assistant hands callers off to.
const TransferQueueEveryone = "Everyone"

// WarmTransferTaskAttributes are the attributes carried by the task the assistant creates
// when it warm transfers a caller. ConferenceName points at the conference the caller is
// already sitting in, which is what lets the accepting worker join the live call rather
// than start a new one.
type WarmTransferTaskAttributes struct {
	IsWarmTransfer      bool   `json:"is_warm_transfer"`
	ConferenceName      string `json:"conference_name"`
	ConferenceSid       string `json:"conference_sid"`
	TransferredFromTask string `json:"transferred_from_task_sid"`
	TargetQueue         string `json:"target_queue"`
	Summary             string `json:"summary"`
	Reason              string `json:"reason"`
	CallSid             string `json:"call_sid"`
	CustomerCallSid     string `json:"customer_call_sid"`
	IsVirtualAgentTask  bool   `json:"isVirtualAgentTask"`
}

// TaskConferenceAttributes is the conference block Task Router writes onto a task it
// assigned with the conference instruction.
type TaskConferenceAttributes struct {
	Sid          string `json:"sid"`
	Participants struct {
		Worker   string `json:"worker"`
		Customer string `json:"customer"`
	} `json:"participants"`
}

// ParseTaskConferenceAttributes pulls the conference Task Router put a task's legs into.
// A task assigned some other way yields the zero value.
func ParseTaskConferenceAttributes(raw string) (TaskConferenceAttributes, error) {
	var wrapper struct {
		Conference TaskConferenceAttributes `json:"conference"`
	}

	if raw == "" {
		return wrapper.Conference, nil
	}

	if err := json.Unmarshal([]byte(raw), &wrapper); err != nil {
		return wrapper.Conference, err
	}

	return wrapper.Conference, nil
}

// ParseWarmTransferTaskAttributes reads the warm transfer fields out of a task's raw
// attributes JSON. A task that is not a transfer unmarshals cleanly with IsWarmTransfer
// false, so callers can treat the zero value as "not a transfer".
func ParseWarmTransferTaskAttributes(raw string) (WarmTransferTaskAttributes, error) {
	var attributes WarmTransferTaskAttributes

	if raw == "" {
		return attributes, nil
	}

	if err := json.Unmarshal([]byte(raw), &attributes); err != nil {
		return attributes, err
	}

	return attributes, nil
}

// WorkerContactAttributes is the slice of a worker's attributes needed to dial them into
// a conference.
type WorkerContactAttributes struct {
	ContactUri string `json:"contact_uri"`
}

func ParseWorkerContactAttributes(raw string) (WorkerContactAttributes, error) {
	var attributes WorkerContactAttributes

	if raw == "" {
		return attributes, nil
	}

	if err := json.Unmarshal([]byte(raw), &attributes); err != nil {
		return attributes, err
	}

	return attributes, nil
}
