package artificial_intelligence

import (
	"context"
	"testing"

	"zennify/zenn-ai-assistant/config"
	"zennify/zenn-ai-assistant/models"

	"github.com/rs/zerolog"
)

// mockAgent is a test double for models.AiBotService. SendMessage returns
// canned values so handlers can be tested without hitting the Anthropic API.
type mockAgent struct {
	resp models.AiBotResponse
	err  error
}

func (m *mockAgent) SetModel(string)                {}
func (m *mockAgent) SetSystemPrompt(string)         {}
func (m *mockAgent) SetSystemPrompts(...string)     {}
func (m *mockAgent) SetUserMessage(string)          {}
func (m *mockAgent) SetUserMessages(...string)      {}
func (m *mockAgent) SetAssistantMessage(string)     {}
func (m *mockAgent) SetAssistantMessages(...string) {}
func (m *mockAgent) SendMessage(context.Context) (models.AiBotResponse, error) {
	return m.resp, m.err
}

func TestBuildAgents_NoAPIKey(t *testing.T) {
	agents := BuildAgents(config.Config{}, zerolog.Nop())
	if len(agents) != 0 {
		t.Fatalf("expected no agents without an API key, got %d", len(agents))
	}
}

func TestBuildAgents_WithAnthropicKey(t *testing.T) {
	cfg := config.Config{
		Anthropic: config.AnthropicConfig{
			APIKey: "sk-test",
			Model:  "claude-3-5-sonnet-20241022",
		},
	}

	agents := BuildAgents(cfg, zerolog.Nop())

	factory, ok := agents[models.AgentAnthropic]
	if !ok {
		t.Fatalf("expected %q agent to be registered", models.AgentAnthropic)
	}

	// The factory must return a non-nil, fresh instance on each call.
	first := factory()
	second := factory()
	if first == nil || second == nil {
		t.Fatal("factory returned a nil agent")
	}
	if first == second {
		t.Fatal("factory must return a fresh instance per call, not a shared one")
	}
}
