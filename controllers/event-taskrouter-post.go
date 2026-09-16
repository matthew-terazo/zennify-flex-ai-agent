package controllers

import (
	"encoding/json"
	"errors"
	"net/http"

	"zennify/zenn-ai-assistant/middleware"
	"zennify/zenn-ai-assistant/models"

	"github.com/rs/zerolog"
	conversations "github.com/twilio/twilio-go/rest/conversations/v1"
	taskrouter "github.com/twilio/twilio-go/rest/taskrouter/v1"
)

type TaskRouterEventController struct {
	App models.App
}

// TaskRouterEventHandler handles incoming Task Router events from Twilio
func (c TaskRouterEventController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	var events []models.TwilioTaskRouterEvent

	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		logger.Error().Err(err).Msg("Failed to parse Task Router event")
		writeJSONResponse(w, http.StatusBadRequest, nil)
		return
	}

	logger.Info().
		Any("events", events).
		Msg("Received Task Router events")

	for _, event := range events {
		// if this is not a virtual event acknowledge and move on
		isVirtualAgent, err := event.Data.Payload.IsAssignedToVirtualAgent()
		if err != nil {
			logger.Err(err).Msg("error unmarshalling virtual agent task attributes")
			continue
		}
		if !isVirtualAgent {
			logger.Debug().Msg("task is not a virtual task")
			continue
		}

		if event.Data.Payload.TaskChannelUniqueName != "chat" {
			writeJSONResponse(w, http.StatusNoContent, nil)
			return
		}

		// Process the event based on type
		switch event.Type {
		case "com.twilio.taskrouter.reservation.created":
			if err := c.handleReservationCreated(event, logger); err != nil {
				writeJSONResponse(w, http.StatusBadRequest, nil)
				return
			}
		default:
			logger.Debug().Str("event_type", event.Type).Msg("Unhandled Task Router event type")
		}
	}

	writeJSONResponse(w, http.StatusNoContent, nil)
}

func (c TaskRouterEventController) handleVoiceReservationCreated(event models.TwilioTaskRouterEvent, logger zerolog.Logger) error {
	var reservationParams taskrouter.UpdateTaskReservationParams

	res, err := c.App.Twilio.AcceptReservaton(event.Data.Payload.ReservationSID, event.Data.Payload.TaskSID, reservationParams)
	if err != nil {
		logger.Err(err).Msg("error accepting reservation")
		return err
	}
	logger.Info().Any("reservation", res).Msg("reservation accepted")

	return nil
}

func (c TaskRouterEventController) handleChatReservationCreated(event models.TwilioTaskRouterEvent, logger zerolog.Logger) error {
	var reservationParams taskrouter.UpdateTaskReservationParams
	var conversationParams conversations.UpdateConversationParams
	var conversationWebhookParams conversations.CreateConversationScopedWebhookParams
	var conversationAttributes models.TwilioConversationAttributes

	res, err := c.App.Twilio.AcceptReservaton(event.Data.Payload.ReservationSID, event.Data.Payload.TaskSID, reservationParams)
	if err != nil {
		logger.Err(err).Msg("error accepting reservation")
		return err
	}
	logger.Info().Any("reservation", res).Msg("reservation accepted")

	conversationSid, err := event.Data.Payload.GetConversationSid()
	if err != nil {
		logger.Err(err).Msg("error getting conversationSid from task")
		return err
	}
	if conversationSid == "" {
		err := errors.New("conversationSid not set on task attributes")
		logger.Err(err).Msg("error conversationSid not set on task")
		return err
	}

	conversationAttributes.TaskSid = *res.TaskSid

	conversationAttrs, err := json.Marshal(conversationAttributes)
	if err != nil {
		logger.Err(err).Msg("error marshalling conversation attributes")
		return err
	}

	conversationParams.SetAttributes(string(conversationAttrs))

	if _, err := c.App.Twilio.UpdateConversation(conversationSid, conversationParams); err != nil {
		logger.Err(err).Msg("error updatingconversation attributes")
		return err
	}

	conversationWebhookParams.SetConfigurationReplayAfter(0)
	conversationWebhookParams.SetConfigurationMethod("post")
	conversationWebhookParams.SetTarget("webhook")
	conversationWebhookParams.SetConfigurationUrl(c.App.Config.Twilio.WebhookBaseUrl + "/twilio/conversation/event")
	conversationWebhookParams.SetConfigurationFilters([]string{"onMessageAdded"})

	if _, err := c.App.Twilio.UpdateConversationWebhook(conversationSid, conversationWebhookParams); err != nil {
		logger.Err(err).Msg("error setting conversation webhook")
		return err
	}

	return nil
}

// accept the reservation for the virtual worker and assign a webhook to the conversation
func (c TaskRouterEventController) handleReservationCreated(event models.TwilioTaskRouterEvent, logger zerolog.Logger) error {

	switch event.Data.Payload.TaskChannelUniqueName {
	case "chat":
		return c.handleChatReservationCreated(event, logger)
	case "voice":
		return c.handleVoiceReservationCreated(event, logger)
	default:
		return errors.New("error unhandled task router channel type")
	}
}
