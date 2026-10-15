package models

import (
	"encoding/json"
	"strconv"
	"testing"
)

// The Task Router event stream delivers attributes as a JSON string containing JSON.
func eventStreamAttributes(raw string) json.RawMessage {
	return json.RawMessage(strconv.Quote(raw))
}

func TestInboundTaskIsNotATransfer(t *testing.T) {
	var payload TwilioTaskRouterPayload
	payload.TaskAttributes = eventStreamAttributes(`{"isVirtualAgentTask":true,"call_sid":"CA1"}`)

	attributes, err := payload.TransferAttributes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The assignment callback already dials the assistant in for its own inbound task;
	// treating this as a transfer would connect it to the same call twice.
	if attributes.IsTransferToVirtualAgent() {
		t.Fatal("the assistant's own inbound task must not be treated as a transfer")
	}
}

func TestFlexColdTransferConnectsAndOwnsTheCall(t *testing.T) {
	var payload TwilioTaskRouterPayload
	payload.TaskAttributes = eventStreamAttributes(`{"transferMeta":{"mode":"COLD","originalTaskSid":"WTorig"}}`)

	attributes, err := payload.TransferAttributes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !attributes.IsTransferToVirtualAgent() {
		t.Fatal("expected a Flex cold transfer to be treated as a transfer")
	}

	if got := attributes.ConnectionMode(); got != VirtualAgentModeCold {
		t.Fatalf("expected mode %q, got %q", VirtualAgentModeCold, got)
	}

	if attributes.TransferMeta.OriginalTaskSid != "WTorig" {
		t.Fatalf("expected the original task sid to survive, got %q", attributes.TransferMeta.OriginalTaskSid)
	}
}

func TestFlexWarmTransferIsAConferenceIn(t *testing.T) {
	var payload TwilioTaskRouterPayload
	payload.TaskAttributes = eventStreamAttributes(`{"transferMeta":{"mode":"WARM","originalTaskSid":"WTorig"}}`)

	attributes, err := payload.TransferAttributes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := attributes.ConnectionMode(); got != VirtualAgentModeConference {
		t.Fatalf("expected a WARM transfer to be a conference-in, got %q", got)
	}
}

// Flex copies the original task's attributes onto a transfer task, so the inbound flag can
// ride along on a genuine transfer. transferMeta has to win.
func TestTransferMetaBeatsACopiedInboundFlag(t *testing.T) {
	var payload TwilioTaskRouterPayload
	payload.TaskAttributes = eventStreamAttributes(`{"isVirtualAgentTask":true,"transferMeta":{"mode":"WARM"}}`)

	attributes, err := payload.TransferAttributes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !attributes.IsTransferToVirtualAgent() {
		t.Fatal("transferMeta must mark this a transfer even with the inbound flag copied over")
	}

	if got := attributes.ConnectionMode(); got != VirtualAgentModeConference {
		t.Fatalf("expected conference mode, got %q", got)
	}
}

func TestTaskWithoutTheInboundFlagIsATransfer(t *testing.T) {
	var payload TwilioTaskRouterPayload
	payload.TaskAttributes = eventStreamAttributes(`{"call_sid":"CA1"}`)

	attributes, err := payload.TransferAttributes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !attributes.IsTransferToVirtualAgent() {
		t.Fatal("a task that is not the assistant's own inbound task is a transfer")
	}

	if got := attributes.ConnectionMode(); got != VirtualAgentModeCold {
		t.Fatalf("expected cold as the default mode, got %q", got)
	}
}

// Once the mode is recorded on the task it is authoritative, so a re-read agrees with the
// decision made when the assistant was connected.
func TestRecordedModeWins(t *testing.T) {
	var payload TwilioTaskRouterPayload
	payload.TaskAttributes = eventStreamAttributes(`{"virtual_agent_mode":"conference","transferMeta":{"mode":"COLD"}}`)

	attributes, err := payload.TransferAttributes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := attributes.ConnectionMode(); got != VirtualAgentModeConference {
		t.Fatalf("expected the recorded mode to win, got %q", got)
	}
}

func TestWorkerContactUriFromEventStream(t *testing.T) {
	var payload TwilioTaskRouterPayload
	payload.WorkerAttributes = eventStreamAttributes(`{"isVirtualAgent":true,"contact_uri":"sip:agent@example.sip.twilio.com"}`)

	contactUri, err := payload.WorkerContactUri()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if contactUri != "sip:agent@example.sip.twilio.com" {
		t.Fatalf("unexpected contact uri %q", contactUri)
	}
}

// The event stream double-encodes attributes; the REST API does not. Both have to parse.
func TestDecodeTwilioAttributesAcceptsBothShapes(t *testing.T) {
	for name, raw := range map[string]json.RawMessage{
		"event stream": eventStreamAttributes(`{"isVirtualAgentTask":true}`),
		"rest api":     json.RawMessage(`{"isVirtualAgentTask":true}`),
	} {
		t.Run(name, func(t *testing.T) {
			var attributes TransferredTaskAttributes

			if err := DecodeTwilioAttributes(raw, &attributes); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !attributes.IsVirtualAgentTask {
				t.Fatal("expected isVirtualAgentTask to decode as true")
			}
		})
	}
}

func TestDecodeTwilioAttributesToleratesEmptyInput(t *testing.T) {
	for name, raw := range map[string]json.RawMessage{
		"absent":       nil,
		"null":         json.RawMessage(`null`),
		"empty string": eventStreamAttributes(``),
	} {
		t.Run(name, func(t *testing.T) {
			var attributes TransferredTaskAttributes

			if err := DecodeTwilioAttributes(raw, &attributes); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSipUriWithHeaders(t *testing.T) {
	headers := [][2]string{
		{SipHeaderVirtualAgentMode, VirtualAgentModeConference},
		{SipHeaderTaskSid, "WT123"},
	}

	got := SipUriWithHeaders("sip:agent@example.sip.twilio.com", headers)
	want := "sip:agent@example.sip.twilio.com?X-Zenn-Virtual-Agent-Mode=conference&X-Zenn-Task-Sid=WT123"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSipUriWithHeadersAppendsToAnExistingQuery(t *testing.T) {
	got := SipUriWithHeaders("sip:agent@example.com?X-Existing=1", [][2]string{{SipHeaderTaskSid, "WT123"}})
	want := "sip:agent@example.com?X-Existing=1&X-Zenn-Task-Sid=WT123"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// A worker can be reachable at a client or phone address, where sip uri headers are
// meaningless and would corrupt the destination.
func TestSipUriWithHeadersLeavesNonSipContactsAlone(t *testing.T) {
	for _, contactUri := range []string{"client:agent1", "+15551234567"} {
		if got := SipUriWithHeaders(contactUri, [][2]string{{SipHeaderTaskSid, "WT123"}}); got != contactUri {
			t.Fatalf("expected %q untouched, got %q", contactUri, got)
		}
	}
}

func TestSipUriWithHeadersSkipsEmptyValues(t *testing.T) {
	got := SipUriWithHeaders("sip:agent@example.com", [][2]string{
		{SipHeaderVirtualAgentMode, ""},
		{SipHeaderTaskSid, "WT123"},
	})
	want := "sip:agent@example.com?X-Zenn-Task-Sid=WT123"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
