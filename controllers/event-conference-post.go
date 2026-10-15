package controllers

import (
	"net/http"
	"zennify/zenn-ai-assistant/middleware"
	"zennify/zenn-ai-assistant/models"
)

type ConferenceEventController struct {
	App models.App
}

// ServeHTTP receives Twilio conference status callbacks. Its job is to notice when a
// human agent's leg joins a conference an assistant is still on, so the assistant can
// introduce the caller before it drops off.
func (c ConferenceEventController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	if err := r.ParseForm(); err != nil {
		logger.Error().Err(err).Msg("error parsing conference callback body")
		writeJSONResponse(w, http.StatusBadRequest, nil)
		return
	}

	event := r.PostForm.Get("StatusCallbackEvent")
	conferenceName := r.PostForm.Get("FriendlyName")
	callSid := r.PostForm.Get("CallSid")

	logger.Info().
		Str("event", event).
		Str("conferenceName", conferenceName).
		Str("conferenceSid", r.PostForm.Get("ConferenceSid")).
		Str("callSid", callSid).
		Msg("received conference status callback")

	if event == "participant-join" && conferenceName != "" {
		// A no-op unless a warm transfer is waiting on this conference.
		c.App.WarmTransfers.Signal(conferenceName, callSid)
	}

	writeJSONResponse(w, http.StatusNoContent, nil)
}
