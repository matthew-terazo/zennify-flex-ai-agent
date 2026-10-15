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

// TaskQueueCallbackResponse is a Task Router assignment instruction. The "conference"
// instruction drops the caller into a conference named after the task sid and dials the
// reserved worker into it, which is what makes a third leg (a warm transfer) possible
// later. See https://www.twilio.com/docs/taskrouter/js-sdk/reservation#conference.
type TaskQueueCallbackResponse struct {
	Instruction         string `json:"instruction"`
	PostWorkActivitySid string `json:"post_work_activity_sid,omitempty"`
	From                string `json:"from,omitempty"`
	To                  string `json:"to,omitempty"`
	Timeout             int    `json:"timeout,omitempty"`

	// Pointers so an explicit false is sent rather than omitted. EndConferenceOnExit
	// false is load-bearing: it is what lets the assistant hang up its own leg without
	// tearing down the conference the caller and agent are still on.
	StartConferenceOnEnter      *bool `json:"start_conference_on_enter,omitempty"`
	EndConferenceOnExit         *bool `json:"end_conference_on_exit,omitempty"`
	EndConferenceOnCustomerExit *bool `json:"end_conference_on_customer_exit,omitempty"`

	Beep                           string `json:"beep,omitempty"`
	ConferenceStatusCallback       string `json:"conference_status_callback,omitempty"`
	ConferenceStatusCallbackMethod string `json:"conference_status_callback_method,omitempty"`
	// Comma separated, matching Twilio's other *_status_callback_event instruction fields.
	ConferenceStatusCallbackEvent string `json:"conference_status_callback_event,omitempty"`
}

// func (m *TaskQueueCallbackResponse) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(m)
// }
