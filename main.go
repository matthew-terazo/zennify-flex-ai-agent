package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"zennify/zenn-ai-assistant/config"
	"zennify/zenn-ai-assistant/controllers"
	"zennify/zenn-ai-assistant/db"
	"zennify/zenn-ai-assistant/models"
	ai "zennify/zenn-ai-assistant/services/artificial-intelligence"
	promptconfiguration "zennify/zenn-ai-assistant/services/prompt-configuration"
	comms_twilio "zennify/zenn-ai-assistant/services/twilio"
)

func main() {
	cfg := config.Load()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Apply configured log level
	if level, err := zerolog.ParseLevel(cfg.Log.Level); err == nil {
		zerolog.SetGlobalLevel(level)
	}

	database, err := db.Connect(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer database.Close()
	log.Info().Msg("Connected to database")

	mux := http.NewServeMux()
	twilio := comms_twilio.NewTwilioClient(cfg.Twilio)
	promptConfigService := promptconfiguration.NewService(database)

	app := models.App{
		Config:              cfg,
		Mux:                 mux,
		Agents:              ai.BuildAgents(cfg, log.Logger),
		Twilio:              twilio,
		DB:                  database,
		PromptConfigService: promptConfigService,
	}

	controllers.RegisterRoutes(app)

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Info().Str("address", addr).Msg("Starting server")
	if err := http.ListenAndServe(addr, corsMiddleware.Handler(mux)); err != nil {
		log.Fatal().Err(err).Msg("Server error")
	}
}
