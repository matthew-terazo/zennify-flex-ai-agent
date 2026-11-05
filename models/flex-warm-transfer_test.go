package models

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// A real reservation.created event captured from a Flex agent warm transferring a live
// call to the virtual agent worker for a consultation.
func loadFlexWarmTransferEvent(t *testing.T) TwilioTaskRouterPayload {
	t.Helper()

	raw, err := os.ReadFile("testdata/flex-warm-transfer-reservation.json")
	if err != nil {
		t.Fatalf("reading testdata: %v", err)
	}

	var events []TwilioTaskRouterEvent
	if err := json.Unmarshal(raw, &events); err != nil {
		t.Fatalf("unmarshalling event: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != "com.twilio.taskrouter.reservation.created" {
		t.Fatalf("unexpected event type %q", events[0].Type)
	}

	return events[0].Data.Payload
}

func TestFlexWarmTransferIsRecognisedAsOurs(t *testing.T) {
	payload := loadFlexWarmTransferEvent(t)

	isVirtualAgent, err := payload.IsAssignedToVirtualAgent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !isVirtualAgent {
		t.Fatal("worker attributes say isVirtualAgent true; the reservation must be recognised as ours")
	}

	if payload.TaskChannelUniqueName != "voice" {
		t.Fatalf("expected the voice channel, got %q", payload.TaskChannelUniqueName)
	}
}

// Flex states the transfer on the event payload. It writes no transferMeta block onto the
// task, so detection cannot depend on one.
func TestFlexWarmTransferIsAConferenceIn(t *testing.T) {
	payload := loadFlexWarmTransferEvent(t)

	attributes, err := payload.TransferAttributes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attributes.TransferMeta != nil {
		t.Fatal("this payload carries no transferMeta; the fixture or the model has drifted")
	}

	if !attributes.IsTransferToVirtualAgent() {
		t.Fatal("expected the reservation to be treated as a hand-off to the assistant")
	}

	// The whole point: a WARM transfer is a consultation. Reading this as cold would have
	// the assistant greet a customer mid-conversation and let end_call tear down the
	// conference, hanging up on the agent who asked for it.
	if got := attributes.ConnectionMode(); got != VirtualAgentModeConference {
		t.Fatalf("expected %q for a WARM transfer, got %q", VirtualAgentModeConference, got)
	}
}

func TestFlexWarmTransferCarriesTheLiveConference(t *testing.T) {
	payload := loadFlexWarmTransferEvent(t)

	attributes, err := payload.TransferAttributes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attributes.Conference.Sid != "CFbce5d39958189f6a4ef2b22750359aa6" {
		t.Fatalf("expected the live conference sid off the task, got %q", attributes.Conference.Sid)
	}

	// The worker leg on the task is the human agent's, not the assistant's.
	if attributes.Conference.Participants.Worker != "CA7506eb107e2d4c987fe79fbe11bb6b13" {
		t.Fatalf("unexpected worker leg %q", attributes.Conference.Participants.Worker)
	}

	if attributes.Conference.Participants.Customer != "CAac43a8f3e74421bd32bc6d6bb3e555c4" {
		t.Fatalf("unexpected customer leg %q", attributes.Conference.Participants.Customer)
	}
}

func TestFlexWarmTransferContactUriTakesSipHeaders(t *testing.T) {
	payload := loadFlexWarmTransferEvent(t)

	contactUri, err := payload.WorkerContactUri()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Carries sip uri parameters already, and no query string.
	if !strings.HasPrefix(contactUri, "sip:proj_") || !strings.Contains(contactUri, ";transport=tls") {
		t.Fatalf("unexpected contact uri %q", contactUri)
	}

	dialled := SipUriWithHeaders(contactUri, [][2]string{
		{SipHeaderVirtualAgentMode, VirtualAgentModeConference},
		{SipHeaderTaskSid, payload.TaskSID},
	})

	// Headers go after the uri parameters, introduced by "?" since there is none yet.
	want := contactUri + "?X-Zenn-Virtual-Agent-Mode=conference&X-Zenn-Task-Sid=WTd750f40c5dfbda0f60c8200bdda57f3a"
	if dialled != want {
		t.Fatalf("expected\n  %s\ngot\n  %s", want, dialled)
	}

	// The worker sid parameter already on the uri must survive untouched.
	if !strings.Contains(dialled, ";x-worker-sid=WK1f520e38dc1d0f2118d946ff148b9a11?") {
		t.Fatalf("sip uri parameters were disturbed: %s", dialled)
	}
}

// Flex reserves against the agent's existing task on a transfer, so a mode recorded by an
// earlier consultation must not outrank the mode on a later event for the same task.
func TestEventModeOutranksAModeRecordedByAnEarlierTransfer(t *testing.T) {
	payload := loadFlexWarmTransferEvent(t)
	payload.TransferMode = "COLD"

	attributes, err := payload.TransferAttributes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attributes.VirtualAgentMode = VirtualAgentModeConference // left over from a prior consultation

	if got := attributes.ConnectionMode(); got != VirtualAgentModeCold {
		t.Fatalf("expected the event to win with %q, got %q", VirtualAgentModeCold, got)
	}
}
