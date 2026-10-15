package controllers

import (
	"net/http"
	"zennify/zenn-ai-assistant/middleware"
	"zennify/zenn-ai-assistant/models"

	"github.com/rs/zerolog"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

const (
	assignmentPostWorkActivitySid = "WA1a65e9e0f4f271d8c0d8ea5cc831ef9d"
	assignmentCallerId            = "+17039975700"
)

type TaskRouterQueueAssignmentCallbackController struct {
	App models.App
}

func (c TaskRouterQueueAssignmentCallbackController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	if err := r.ParseForm(); err != nil {
		logger.Error().Err(err).Msg("error parsing assignment callback body")
		writeJSONResponse(w, http.StatusBadRequest, nil)
		return
	}

	logger = logger.With().
		Str("taskSid", r.PostForm.Get("TaskSid")).
		Str("reservationSid", r.PostForm.Get("ReservationSid")).
		Logger()

	transfer, err := models.ParseWarmTransferTaskAttributes(r.PostForm.Get("TaskAttributes"))
	if err != nil {
		logger.Error().Err(err).Msg("error parsing task attributes on assignment callback")
	}

	// The caller on a transfer task is already sitting in a conference with the
	// assistant. Accept the reservation and dial this worker into that conference rather
	// than bridging a second call, so all three legs are up at once.
	if transfer.IsWarmTransfer && transfer.ConferenceName != "" {
		logger.Info().
			Str("conferenceName", transfer.ConferenceName).
			Msg("accepting warm transfer reservation")

		go c.joinWorkerToConference(transfer, r.PostForm.Get("WorkerSid"), r.PostForm.Get("WorkerAttributes"), logger)

		writeJSONResponse(w, http.StatusOK, &models.TaskQueueCallbackResponse{Instruction: "accept"})
		return
	}

	startConferenceOnEnter := true
	endConferenceOnExit := false
	endConferenceOnCustomerExit := true

	response := models.TaskQueueCallbackResponse{
		Instruction:            "conference",
		PostWorkActivitySid:    assignmentPostWorkActivitySid,
		From:                   assignmentCallerId,
		StartConferenceOnEnter: &startConferenceOnEnter,
		// The assistant is the worker on this leg. Leaving the conference open when it
		// exits is what lets it brief a human agent and then drop off mid-call.
		EndConferenceOnExit:            &endConferenceOnExit,
		EndConferenceOnCustomerExit:    &endConferenceOnCustomerExit,
		Beep:                           "false",
		ConferenceStatusCallback:       c.App.Config.Twilio.WebhookBaseUrl + "/twilio/conference/event",
		ConferenceStatusCallbackMethod: "POST",
		ConferenceStatusCallbackEvent:  "start,end,join,leave",
	}

	writeJSONResponse(w, http.StatusOK, &response)
}

// joinWorkerToConference dials the reserved worker into the conference the caller is
// already on. Twilio's participant-join callback for this leg is what tells the assistant
// the agent has arrived and it is time to brief them.
func (c TaskRouterQueueAssignmentCallbackController) joinWorkerToConference(transfer models.WarmTransferTaskAttributes, workerSid string, workerAttributes string, logger zerolog.Logger) {
	contact, err := models.ParseWorkerContactAttributes(workerAttributes)
	if err != nil {
		logger.Error().Err(err).Msg("error parsing worker attributes for warm transfer")
		return
	}

	// The assignment callback does not always carry worker attributes; fall back to the
	// worker resource itself.
	if contact.ContactUri == "" && workerSid != "" {
		worker, err := c.App.Twilio.FetchWorker(workerSid)
		if err != nil {
			logger.Error().Err(err).Str("workerSid", workerSid).Msg("error fetching worker for warm transfer")
			return
		}

		if worker.Attributes != nil {
			if contact, err = models.ParseWorkerContactAttributes(*worker.Attributes); err != nil {
				logger.Error().Err(err).Str("workerSid", workerSid).Msg("error parsing fetched worker attributes")
				return
			}
		}
	}

	if contact.ContactUri == "" {
		logger.Error().Str("workerSid", workerSid).Msg("worker has no contact_uri to join to the conference")
		return
	}

	// The transferring assistant passes the conference sid along; only fall back to a
	// lookup by name when it is missing.
	conferenceSid := transfer.ConferenceSid
	if conferenceSid == "" {
		conference, err := c.App.Twilio.FindConferenceByName(transfer.ConferenceName)
		if err != nil {
			logger.Error().Err(err).Str("conferenceName", transfer.ConferenceName).Msg("error finding conference for warm transfer")
			return
		}

		conferenceSid = *conference.Sid
	}

	var params api.CreateParticipantParams

	params.SetFrom(assignmentCallerId)
	params.SetTo(contact.ContactUri)
	params.SetStartConferenceOnEnter(true)
	// The agent is the last party on the call, so their hangup ends it.
	params.SetEndConferenceOnExit(true)
	params.SetBeep("false")

	participant, err := c.App.Twilio.AddConferenceParticipant(conferenceSid, params)
	if err != nil {
		logger.Error().Err(err).
			Str("conferenceSid", conferenceSid).
			Str("contactUri", contact.ContactUri).
			Msg("error adding worker to conference for warm transfer")
		return
	}

	logger.Info().
		Str("conferenceSid", conferenceSid).
		Any("participant", participant).
		Msg("worker dialed into conference for warm transfer")
}
