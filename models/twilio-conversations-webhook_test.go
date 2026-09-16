package models

import (
	"testing"
	"time"
)

func TestParseTwilioConversationsWebhook(t *testing.T) {
	// The provided URL-encoded payload
	payload := "MessagingServiceSid=MG188e7b6696e8726b20431db61ad0dd15&EventType=onMessageAdded&Attributes=%7B%7D&DateCreated=2026-05-29T02%3A04%3A12.264Z&Index=0&ChatServiceSid=ISe8090e8d1163455a88572f2e704b44cf&MessageSid=IM2c646eeb5eb14c65b3f0bed37714f113&AccountSid=AC8e3875079b184b53ed6c8fbf133899df&Source=API&RetryCount=0&WebhookType=webhook&Author=FX0599f649dc7a3a3f2b834e2a13b38396&ParticipantSid=MB9ca58ca912e2450eb4289675c8c0f431&Body=testing&WebhookSid=WH930960287b3240d486cba9c8ee5cdb9a&ConversationSid=CH366b1857fcd544239e5496cd1bcdbd53"

	webhook, err := ParseTwilioConversationsWebhook(payload)
	if err != nil {
		t.Fatalf("failed to parse webhook: %v", err)
	}

	// Verify key fields
	if webhook.MessagingServiceSid != "MG188e7b6696e8726b20431db61ad0dd15" {
		t.Errorf("MessagingServiceSid mismatch: got %q", webhook.MessagingServiceSid)
	}

	if webhook.EventType != "onMessageAdded" {
		t.Errorf("EventType mismatch: got %q", webhook.EventType)
	}

	if webhook.Body != "testing" {
		t.Errorf("Body mismatch: got %q", webhook.Body)
	}

	if webhook.ConversationSid != "CH366b1857fcd544239e5496cd1bcdbd53" {
		t.Errorf("ConversationSid mismatch: got %q", webhook.ConversationSid)
	}

	if webhook.Index != 0 {
		t.Errorf("Index mismatch: got %d", webhook.Index)
	}

	if webhook.RetryCount != 0 {
		t.Errorf("RetryCount mismatch: got %d", webhook.RetryCount)
	}

	// Verify DateCreated parsing
	expectedTime := time.Date(2026, 5, 29, 2, 4, 12, 264000000, time.UTC)
	if !webhook.DateCreated.Equal(expectedTime) {
		t.Errorf("DateCreated mismatch: got %v, expected %v", webhook.DateCreated, expectedTime)
	}
}
