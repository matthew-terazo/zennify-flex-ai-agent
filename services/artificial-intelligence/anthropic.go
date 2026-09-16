package artificial_intelligence

import (
	"context"

	"zennify/zenn-ai-assistant/models"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/rs/zerolog"
)

// anthropicServiceImpl is the concrete implementation of AnthropicService
type anthropicServiceImpl struct {
	logger   zerolog.Logger
	model    anthropic.Model
	system   []anthropic.TextBlockParam
	messages []anthropic.MessageParam
	tools    []anthropic.ToolUnionParam
	client   *anthropic.Client
}

// NewAnthropicService creates a new AnthropicService instance
func NewAnthropicService(apiKey string, logger zerolog.Logger) models.AiBotService {
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)
	return &anthropicServiceImpl{
		client: &client,
		logger: logger,
	}
}

func (s *anthropicServiceImpl) SetLogger(logger zerolog.Logger) {
	s.logger = logger
}

func (s *anthropicServiceImpl) SetModel(modelName string) {
	s.model = modelName
}

func (s *anthropicServiceImpl) SetSystemPrompt(prompt string) {
	s.system = append(s.system, anthropic.TextBlockParam{Text: prompt})
}

func (s *anthropicServiceImpl) SetSystemPrompts(prompts ...string) {
	for _, prompt := range prompts {
		s.SetSystemPrompt(prompt)
	}
}

func (s *anthropicServiceImpl) SetUserMessage(message string) {
	text := anthropic.NewTextBlock(message)
	s.messages = append(s.messages, anthropic.NewUserMessage(text))
}

func (s *anthropicServiceImpl) SetUserMessages(messages ...string) {
	for _, message := range messages {
		s.SetUserMessage(message)
	}
}

func (s *anthropicServiceImpl) SetAssistantMessage(message string) {
	text := anthropic.NewTextBlock(message)
	s.messages = append(s.messages, anthropic.NewAssistantMessage(text))
}

func (s *anthropicServiceImpl) SetAssistantMessages(messages ...string) {
	for _, message := range messages {
		s.SetAssistantMessage(message)
	}
}

func (s *anthropicServiceImpl) SetTools(agentTools ...models.AgentTool) {
	tools := make([]anthropic.ToolUnionParam, len(agentTools))

	for i, agentTool := range agentTools {
		tools[i] = anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        agentTool.Name,
				Type:        anthropic.ToolType(agentTool.Type),
				Description: anthropic.String(agentTool.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: agentTool.InputSchema,
					Required:   agentTool.RequiredFields,
				},
			},
		}
	}
	s.tools = tools
}

func (s *anthropicServiceImpl) SendMessage(ctx context.Context) (models.AiBotResponse, error) {
	var response models.AiBotResponse

	message, err := s.client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 1024,
		Model:     s.model,
		Messages:  s.messages,
		System:    s.system,
		Tools:     s.tools,
	})
	if err != nil {
		return response, err
	}

	s.logger.Info().Any("message", message).Msg("anthropic response")

	for _, block := range message.Content {
		switch block := block.AsAny().(type) {
		case anthropic.TextBlock:
			p := block.ToParam()
			response.Messages = append(response.Messages, p.Text)
			s.SetAssistantMessage(p.Text)
		case anthropic.ToolUseBlock:
			param := block.ToParam()

			response.ToolCalls = append(response.ToolCalls, models.AiBotToolCall{
				ID:     param.ID,
				Name:   param.Name,
				Inputs: param.Input,
			})
		}
	}

	return response, nil
}
