package controllers

import (
	"encoding/json"
	"errors"
	"net/http"

	"zennify/zenn-ai-assistant/middleware"
	"zennify/zenn-ai-assistant/models"

	"github.com/rs/zerolog"
	api "github.com/twilio/twilio-go/rest/api/v2010"
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

// handleVoiceReservationCreated answers a voice reservation offered to the virtual agent
// worker. This is how a Flex agent hands a live call to the assistant: whether they cold
// transfer it or conference the assistant in, Task Router reserves the virtual agent
// worker and nothing on the Flex side will ever accept it, so the reservation is accepted
// here and the assistant's SIP address is dialed into the call the customer is already on.
func (c TaskRouterEventController) handleVoiceReservationCreated(event models.TwilioTaskRouterEvent, logger zerolog.Logger) error {
	payload := event.Data.Payload

	attributes, err := payload.TransferAttributes()
	if err != nil {
		logger.Err(err).Msg("error reading task attributes on voice reservation")
		return err
	}

	// The inbound call's own reservation is answered by the queue assignment callback,
	// which dials the assistant in with a conference instruction. Accepting it here too
	// would put the assistant on the same call twice.
	if !attributes.IsTransferToVirtualAgent() {
		logger.Debug().Msg("voice reservation is the assistant's own inbound task; the assignment callback owns it")
		return nil
	}

	contactUri, err := payload.WorkerContactUri()
	if err != nil {
		logger.Err(err).Msg("error reading virtual agent worker attributes")
		return err
	}

	if contactUri == "" {
		err := errors.New("virtual agent worker has no contact_uri to dial")
		logger.Err(err).Str("workerSid", payload.WorkerSID).Msg("cannot connect virtual agent")
		return err
	}

	mode := attributes.ConnectionMode()
	logger = logger.With().Str("mode", mode).Str("taskSid", payload.TaskSID).Logger()

	conferenceSid, err := c.conferenceForReservation(attributes, payload, logger)
	if err != nil {
		logger.Err(err).Msg("error finding the call to connect the virtual agent to")
		return err
	}

	// Accept first so Task Router stops offering this reservation elsewhere.
	res, err := c.App.Twilio.AcceptReservaton(payload.ReservationSID, payload.TaskSID, taskrouter.UpdateTaskReservationParams{})
	if err != nil {
		logger.Err(err).Msg("error accepting reservation")
		return err
	}
	logger.Info().Any("reservation", res).Msg("reservation accepted")

	var participantParams api.CreateParticipantParams

	participantParams.SetFrom(assignmentCallerId)
	participantParams.SetTo(models.SipUriWithHeaders(contactUri, [][2]string{
		{models.SipHeaderVirtualAgentMode, mode},
		{models.SipHeaderTaskSid, payload.TaskSID},
	}))
	participantParams.SetStartConferenceOnEnter(true)
	// On a conference-in a human agent is still talking to the customer, so the assistant
	// leaving must not end their call. On a cold transfer the assistant is the only agent
	// left, so its exit ends the call.
	participantParams.SetEndConferenceOnExit(mode == models.VirtualAgentModeCold)
	participantParams.SetBeep("false")

	participant, err := c.App.Twilio.AddConferenceParticipant(conferenceSid, participantParams)
	if err != nil {
		logger.Err(err).Str("conferenceSid", conferenceSid).Msg("error dialing the virtual agent into the conference")
		return err
	}

	logger.Info().
		Str("conferenceSid", conferenceSid).
		Any("participant", participant).
		Msg("virtual agent connected to the call")

	// Record the assistant's leg on the task so it can be found by call sid later, the way
	// the inbound flow finds it.
	updates := map[string]any{"virtual_agent_mode": mode}
	if participant.CallSid != nil {
		updates["worker_call_sid"] = *participant.CallSid
	}

	if err := c.App.Twilio.UpdateTaskAttributes(payload.TaskSID, updates); err != nil {
		// The assistant is already on the call and carries the task sid on its SIP leg, so
		// this is worth knowing about but not worth failing the reservation over.
		logger.Warn().Err(err).Msg("error recording the virtual agent leg on the task")
	}

	return nil
}

// conferenceForReservation resolves the live conference the customer is on. Flex records
// it on the task; failing that, Task Router names a conference after the task it was
// created for, so a transfer can be traced back through the original task sid.
func (c TaskRouterEventController) conferenceForReservation(attributes models.TransferredTaskAttributes, payload models.TwilioTaskRouterPayload, logger zerolog.Logger) (string, error) {
	if attributes.Conference.Sid != "" {
		return attributes.Conference.Sid, nil
	}

	candidates := []string{}
	if attributes.TransferMeta != nil && attributes.TransferMeta.OriginalTaskSid != "" {
		candidates = append(candidates, attributes.TransferMeta.OriginalTaskSid)
	}
	candidates = append(candidates, payload.TaskSID)

	for _, name := range candidates {
		conference, err := c.App.Twilio.FindConferenceByName(name)
		if err != nil {
			logger.Debug().Err(err).Str("conferenceName", name).Msg("no live conference under this name")
			continue
		}

		return *conference.Sid, nil
	}

	return "", errors.New("no live conference found for the reserved task")
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
		logger.Debug().
			Str("channel", event.Data.Payload.TaskChannelUniqueName).
			Msg("unhandled task router channel type")
		return nil
	}
}
