package models

import (
	"database/sql"
	"net/http"
	"zennify/zenn-ai-assistant/config"
	comms_twilio "zennify/zenn-ai-assistant/services/twilio"
	warmtransfer "zennify/zenn-ai-assistant/services/warm-transfer"
)

type App struct {
	Config              config.Config
	Mux                 *http.ServeMux
	Agents              map[string]AiBotFactory
	Twilio              comms_twilio.TwilioClient
	DB                  *sql.DB
	PromptConfigService PromptConfigurationService
	WarmTransfers       *warmtransfer.Broker
}
