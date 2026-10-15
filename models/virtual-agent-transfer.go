package models

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

// How the assistant came to be on a call. This decides whether it greets the customer and
// whether it is allowed to end the call when it is done.
const (
	// VirtualAgentModeCold is a fresh inbound call, or a cold transfer where the human
	// agent is handing the customer over and leaving. The assistant owns the call.
	VirtualAgentModeCold = "cold"

	// VirtualAgentModeConference is a conference-in, where a human agent is still on the
	// line. The assistant joins an in-progress conversation and must not hang up on them.
	VirtualAgentModeConference = "conference"
)

// SIP headers set on the assistant's leg when it is dialed into an existing call, so the
// realtime session knows what it is joining without having to race a task attribute write.
const (
	SipHeaderVirtualAgentMode = "X-Zenn-Virtual-Agent-Mode"
	SipHeaderTaskSid          = "X-Zenn-Task-Sid"
)

// FlexTransferMeta is the block Twilio Flex writes onto a task it transfers. Mode is COLD
// when the transferring agent is leaving and WARM when they stay on to introduce the call.
type FlexTransferMeta struct {
	Mode                string `json:"mode"`
	Type                string `json:"type"`
	OriginalTaskSid     string `json:"originalTaskSid"`
	InitiatingWorkerSid string `json:"initiatingWorkerSid"`
}

// TransferredTaskAttributes is the slice of a task's attributes needed to decide whether a
// reservation is a hand-off to the assistant and how it should behave once connected.
type TransferredTaskAttributes struct {
	IsVirtualAgentTask bool                     `json:"isVirtualAgentTask"`
	VirtualAgentMode   string                   `json:"virtual_agent_mode"`
	TransferMeta       *FlexTransferMeta        `json:"transferMeta"`
	Conference         TaskConferenceAttributes `json:"conference"`
}

// IsTransferToVirtualAgent distinguishes a hand-off to the assistant from the inbound call
// the assistant was originally reserved for. The inbound reservation is answered by the
// queue assignment callback, which dials the assistant in itself; accepting that one here
// as well would connect the assistant to the same call twice.
func (a TransferredTaskAttributes) IsTransferToVirtualAgent() bool {
	return a.TransferMeta != nil || !a.IsVirtualAgentTask
}

// ConnectionMode reports how the assistant should behave on this task. A Flex WARM
// transfer is a conference-in: the transferring agent stays on the call.
func (a TransferredTaskAttributes) ConnectionMode() string {
	if a.VirtualAgentMode != "" {
		return a.VirtualAgentMode
	}

	if a.TransferMeta != nil && strings.EqualFold(a.TransferMeta.Mode, "WARM") {
		return VirtualAgentModeConference
	}

	return VirtualAgentModeCold
}

// DecodeTwilioAttributes unmarshals a Twilio attributes field. The Task Router event
// stream delivers attributes as a JSON string containing JSON, while the REST API returns
// them as plain JSON, so both shapes are accepted.
func DecodeTwilioAttributes(raw []byte, target any) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}

	if trimmed[0] != '"' {
		return json.Unmarshal(trimmed, target)
	}

	unquoted, err := strconv.Unquote(string(trimmed))
	if err != nil {
		return err
	}

	if strings.TrimSpace(unquoted) == "" {
		return nil
	}

	return json.Unmarshal([]byte(unquoted), target)
}

// SipUriWithHeaders appends custom SIP headers to a worker's contact uri. Twilio passes
// headers given as uri query parameters through on the INVITE, which is how the assistant
// learns what kind of call it is joining at the moment it connects rather than having to
// race a task attribute write. Non-SIP contact uris are returned untouched.
func SipUriWithHeaders(contactUri string, headers [][2]string) string {
	lowered := strings.ToLower(contactUri)
	if !strings.HasPrefix(lowered, "sip:") && !strings.HasPrefix(lowered, "sips:") {
		return contactUri
	}

	separator := "?"
	if strings.Contains(contactUri, "?") {
		separator = "&"
	}

	var builder strings.Builder

	builder.WriteString(contactUri)

	for _, header := range headers {
		if header[0] == "" || header[1] == "" {
			continue
		}

		builder.WriteString(separator)
		builder.WriteString(url.QueryEscape(header[0]))
		builder.WriteString("=")
		builder.WriteString(url.QueryEscape(header[1]))

		separator = "&"
	}

	return builder.String()
}
