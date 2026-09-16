package models

import (
	"net/url"
	"strconv"
	"time"
)

// TwilioConversationsWebhook represents a webhook event from Twilio Conversations API.
// The webhook is sent as URL-encoded form data.
type TwilioConversationsWebhook struct {
	MessagingServiceSid string    `form:"MessagingServiceSid"`
	EventType           string    `form:"EventType"`
	Attributes          string    `form:"Attributes"`  // URL-encoded JSON string
	DateCreated         time.Time `form:"DateCreated"` // RFC3339 timestamp
	Index               int       `form:"Index"`
	ChatServiceSid      string    `form:"ChatServiceSid"`
	MessageSid          string    `form:"MessageSid"`
	AccountSid          string    `form:"AccountSid"`
	Source              string    `form:"Source"`
	RetryCount          int       `form:"RetryCount"`
	WebhookType         string    `form:"WebhookType"`
	Author              string    `form:"Author"`
	ParticipantSid      string    `form:"ParticipantSid"`
	Body                string    `form:"Body"`
	WebhookSid          string    `form:"WebhookSid"`
	ConversationSid     string    `form:"ConversationSid"`
}

// ParseTwilioConversationsWebhook parses a URL-encoded form payload into a webhook struct.
func ParseTwilioConversationsWebhook(encoded string) (TwilioConversationsWebhook, error) {
	values, err := url.ParseQuery(encoded)
	if err != nil {
		return TwilioConversationsWebhook{}, err
	}

	webhook := TwilioConversationsWebhook{
		MessagingServiceSid: values.Get("MessagingServiceSid"),
		EventType:           values.Get("EventType"),
		Attributes:          values.Get("Attributes"),
		ChatServiceSid:      values.Get("ChatServiceSid"),
		MessageSid:          values.Get("MessageSid"),
		AccountSid:          values.Get("AccountSid"),
		Source:              values.Get("Source"),
		WebhookType:         values.Get("WebhookType"),
		Author:              values.Get("Author"),
		ParticipantSid:      values.Get("ParticipantSid"),
		Body:                values.Get("Body"),
		WebhookSid:          values.Get("WebhookSid"),
		ConversationSid:     values.Get("ConversationSid"),
	}

	// Parse Index (int)
	if indexStr := values.Get("Index"); indexStr != "" {
		if idx, err := strconv.Atoi(indexStr); err == nil {
			webhook.Index = idx
		}
	}

	// Parse RetryCount (int)
	if retryStr := values.Get("RetryCount"); retryStr != "" {
		if retry, err := strconv.Atoi(retryStr); err == nil {
			webhook.RetryCount = retry
		}
	}

	// Parse DateCreated (RFC3339 timestamp)
	if dateStr := values.Get("DateCreated"); dateStr != "" {
		if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
			webhook.DateCreated = t
		}
	}

	return webhook, nil
}
