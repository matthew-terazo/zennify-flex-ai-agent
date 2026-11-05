package controllers

import (
	"net/http"
	"zennify/zenn-ai-assistant/middleware"
	"zennify/zenn-ai-assistant/models"
)

type ConferenceParticipantStatusController struct {
	App models.App
}

// ServeHTTP receives Twilio status callbacks for a participant leg this service dialled.
// It exists so a warm transfer's reservation is only finished once the assistant has
// actually answered: doing it earlier would resolve a reservation for a leg that might
// never come up. The reservation and task are carried on the callback url rather than held
// in memory, so this survives a restart and does not care which instance dialled the leg.
func (c ConferenceParticipantStatusController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	if err := r.ParseForm(); err != nil {
		logger.Error().Err(err).Msg("error parsing participant status callback body")
		writeJSONResponse(w, http.StatusBadRequest, nil)
		return
	}

	reservationSid := r.URL.Query().Get("reservationSid")
	taskSid := r.URL.Query().Get("taskSid")
	callStatus := r.PostForm.Get("CallStatus")

	logger = logger.With().
		Str("reservationSid", reservationSid).
		Str("taskSid", taskSid).
		Str("callSid", r.PostForm.Get("CallSid")).
		Str("callStatus", callStatus).
		Logger()

	logger.Info().Msg("received participant status callback")

	if reservationSid == "" || taskSid == "" {
		logger.Warn().Msg("participant status callback carries no reservation to accept")
		writeJSONResponse(w, http.StatusNoContent, nil)
		return
	}

	switch callStatus {
	case "in-progress", "answered":
		// The assistant is on the call as a conference participant, which is all a
		// consultation needs. The reservation has done its job, so complete it rather than
		// hold it open: the assistant is not the worker handling this task.
		reservation, err := c.App.Twilio.CompleteReservation(reservationSid, taskSid)
		if err != nil {
			logger.Err(err).Msg("error completing reservation after the virtual agent answered")
			break
		}

		logger.Info().Any("reservation", reservation).Msg("virtual agent answered; reservation completed")

	case "busy", "failed", "no-answer", "canceled":
		// Nothing answered, so there is nothing to finish. Hand the reservation back so a
		// later delivery can try to connect the assistant again.
		logger.Warn().Msg("virtual agent did not answer; leaving the reservation alone")
		c.App.Reservations.Release(reservationSid)

	case "completed":
		logger.Info().Msg("virtual agent leg ended")

	default:
		logger.Debug().Msg("unhandled participant call status")
	}

	writeJSONResponse(w, http.StatusNoContent, nil)
}
