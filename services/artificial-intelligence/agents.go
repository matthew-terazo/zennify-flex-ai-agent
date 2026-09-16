package artificial_intelligence

import (
	"zennify/zenn-ai-assistant/config"
	"zennify/zenn-ai-assistant/models"

	"github.com/rs/zerolog"
)

// BuildAgents constructs the registry of available AI agent factories from
// config. It has no side effects and is safe to unit test: pass a config with
// or without credentials and assert on the returned map.
//
// Each entry is a factory that returns a *fresh* AiBotService per call so that
// concurrent requests never share mutable message/system state.
func BuildAgents(cfg config.Config, logger zerolog.Logger) map[string]models.AiBotFactory {
	agents := make(map[string]models.AiBotFactory)

	if cfg.Anthropic.APIKey != "" {
		agents[models.AgentAnthropic] = func() models.AiBotService {
			return NewAnthropicService(cfg.Anthropic.APIKey, logger)
		}
	}

	return agents
}
