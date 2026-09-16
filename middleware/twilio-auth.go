package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/twilio/twilio-go/client"
)

// TwilioAuth validates Twilio webhook requests using the X-Twilio-Signature header
func TwilioAuth(authToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := GetLogger(r.Context())

			// Read the body for validation and then for the handler
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to read request body")
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}

			// Restore the body for the handler
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// Get the signature from the header
			signature := r.Header.Get("X-Twilio-Signature")
			if signature == "" {
				logger.Warn().Msg("Missing X-Twilio-Signature header")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			url := r.Header.Get("X-Forwarded-Proto") + "://" + r.Host + r.URL.String()

			// Validate the request
			requestValidator := client.NewRequestValidator(authToken)

			if !requestValidator.ValidateBody(url, bodyBytes, signature) {
				logger.Warn().Str("signature", signature).Msg("Invalid Twilio signature")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			logger.Debug().Msg("Twilio webhook validated successfully")
			next.ServeHTTP(w, r)
		})
	}
}
