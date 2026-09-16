package controllers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"zennify/zenn-ai-assistant/middleware"
	"zennify/zenn-ai-assistant/models"
	agenttools "zennify/zenn-ai-assistant/services/agent-tools"

	openairt "github.com/WqyJh/go-openai-realtime/v2"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/realtime"
	"github.com/openai/openai-go/v3/webhooks"
	"github.com/rs/zerolog"
)

type OpenAIEventController struct {
	App models.App
}

func (c OpenAIEventController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error().Err(err).Msg("error reading message body")
		writeJSONResponse(w, http.StatusBadRequest, nil)
		return
	}
	defer r.Body.Close()

	openaiClient := openai.NewClient(
		option.WithWebhookSecret(c.App.Config.OpenAI.WebhookSigningKey), // env var used by default; explicit here.
		option.WithAPIKey(c.App.Config.OpenAI.ApiKey),
	)

	webhookEvent, err := openaiClient.Webhooks.Unwrap(requestBody, r.Header)
	if err != nil {
		logger.Error().Err(err).Msg("error verifying webhook event")
		writeJSONResponse(w, http.StatusForbidden, nil)
		return
	}

	switch webhookEvent.AsAny().(type) {
	case webhooks.BatchCancelledWebhookEvent:
	case webhooks.BatchCompletedWebhookEvent:
	case webhooks.BatchExpiredWebhookEvent:
	case webhooks.BatchFailedWebhookEvent:
	case webhooks.EvalRunCanceledWebhookEvent:
	case webhooks.EvalRunFailedWebhookEvent:
	case webhooks.EvalRunSucceededWebhookEvent:
	case webhooks.FineTuningJobCancelledWebhookEvent:
	case webhooks.FineTuningJobFailedWebhookEvent:
	case webhooks.FineTuningJobSucceededWebhookEvent:
	case webhooks.RealtimeCallIncomingWebhookEvent:
		event := webhookEvent.AsRealtimeCallIncoming()
		logger.Info().Any("event", event).Msg("received realtime call incoming webhook event")
		writeJSONResponse(w, http.StatusNoContent, nil)

		go c.handleRealtimeCallIncomingWebhookEvent(event, logger.With().Str("eventId", event.ID).Logger())

		return

	case webhooks.ResponseCancelledWebhookEvent:
	case webhooks.ResponseCompletedWebhookEvent:
	case webhooks.ResponseFailedWebhookEvent:
	case webhooks.ResponseIncompleteWebhookEvent:
	default:
		logger.Error().Err(err).Msg("error unknown webhook event")
		writeJSONResponse(w, http.StatusForbidden, nil)
		return
	}
}

func (c OpenAIEventController) handleRealtimeCallIncomingWebhookEvent(event webhooks.RealtimeCallIncomingWebhookEvent, logger zerolog.Logger) {
	ctx := context.Background()

	openaiClient := openai.NewClient(
		option.WithWebhookSecret(c.App.Config.OpenAI.WebhookSigningKey), // env var used by default; explicit here.
		option.WithAPIKey(c.App.Config.OpenAI.ApiKey),
	)
	var params realtime.CallAcceptParams

	assistant, err := c.App.PromptConfigService.GetActiveConfiguration(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("error getting active assistant")
		return
	}
	endTask := agenttools.CallEndTool()
	transferTask := agenttools.QueueTransferTool()

	params.RealtimeSessionCreateRequest.Model = realtime.RealtimeSessionCreateRequestModelGPTRealtime
	params.RealtimeSessionCreateRequest.Instructions = openai.String(*assistant.SystemPrompt)

	params.RealtimeSessionCreateRequest.Tools = append(params.RealtimeSessionCreateRequest.Tools, realtime.RealtimeToolsConfigUnionParam{
		OfFunction: &realtime.RealtimeFunctionToolParam{
			Name:        openai.String(endTask.Name),
			Description: openai.String(endTask.Description),
			Type:        realtime.RealtimeFunctionToolTypeFunction,
		},
	})
	params.RealtimeSessionCreateRequest.Tools = append(params.RealtimeSessionCreateRequest.Tools, realtime.RealtimeToolsConfigUnionParam{
		OfFunction: &realtime.RealtimeFunctionToolParam{
			Name:        openai.String(transferTask.Name),
			Description: openai.String(transferTask.Description),
			Type:        realtime.RealtimeFunctionToolTypeFunction,
			Parameters:  transferTask.InputSchema,
		},
	})

	if err := openaiClient.Realtime.Calls.Accept(ctx, event.Data.CallID, params); err != nil {
		logger.Err(err).Msg("error accepting sip call")
		return
	}

	websocketURL, err := url.Parse("wss://api.openai.com/v1/realtime")
	if err != nil {
		logger.Err(err).Msg("error creating realtime websocket url")
		return
	}

	websocketURLQueryParams := websocketURL.Query()
	websocketURLQueryParams.Add("call_id", event.Data.CallID)
	websocketURL.RawQuery = websocketURLQueryParams.Encode()

	openAIToken := "Bearer " + c.App.Config.OpenAI.ApiKey
	dialer := ws.Dialer{
		Header: ws.HandshakeHeaderHTTP(http.Header{
			"Authorization": {openAIToken},
		}),
	}

	conn, _, _, err := dialer.Dial(ctx, websocketURL.String())
	if err != nil {
		logger.Err(err).Msg("error creating realtime websocket to openai")
		return
	}
	defer conn.Close()

	greeting := realtime.ResponseFormatJSONObject{
		Type: "response.create",
	}

	greetingBytes, _ := json.Marshal(greeting)

	// give time for call to establish
	time.Sleep(2 * time.Second)

	if err := wsutil.WriteClientText(conn, greetingBytes); err != nil {
		logger.Err(err).Msg("error sending greeting to openai")
	}

	for {
		data, _, err := wsutil.ReadServerData(conn)
		if err != nil {
			logger.Err(err).Msg("error reading websocket connection")
			if err == io.EOF {
				break
			}
		}

		rtEvent, err := openairt.UnmarshalServerEvent(data)
		if err != nil {
			logger.Error().Err(err).Msg("error unmarshalling realtime event")
			continue
		}

		logger.Info().Any("event", rtEvent).Msg("real time websocket event received")

		switch rtEvent.ServerEventType() {
		case openairt.ServerEventTypeResponseFunctionCallArgumentsDone:
			toolCallEvent := rtEvent.(openairt.ResponseFunctionCallArgumentsDoneEvent)

			if toolCallEvent.Name == "end_call" {
				var callSid string
				for _, header := range event.Data.SipHeaders {
					if strings.EqualFold(header.Name, "X-Twilio-CallSid") {
						callSid = header.Value
					}
				}

				if callSid == "" {
					logger.Error().Err(err).Msg("error getting call sid from sip headers")
					return
				}

				task, err := c.App.Twilio.FindTaskByCallSid(callSid)
				if err != nil {
					logger.Error().Err(err).Msg("error finding task by call sid")
					return
				}

				if _, err := c.App.Twilio.EndTask(*task.Sid); err != nil {
					logger.Error().Err(err).Msg("error ending task")
				}

				if err := openaiClient.Realtime.Calls.Hangup(ctx, event.Data.CallID); err != nil {
					logger.Err(err).Msg("error accepting sip call")
					return
				}
			} else if toolCallEvent.Name == "transfer_call" {
				var callSid string
				for _, header := range event.Data.SipHeaders {
					if strings.EqualFold(header.Name, "X-Twilio-CallSid") {
						callSid = header.Value
					}
				}

				if callSid == "" {
					logger.Error().Err(err).Msg("error getting call sid from sip headers")
					return
				}

				task, err := c.App.Twilio.FindTaskByCallSid(callSid)
				if err != nil {
					logger.Error().Err(err).Msg("error finding task by call sid")
					return
				}

				args, err := ParseToolCallArguments(toolCallEvent.Arguments)
				if err != nil {
					logger.Error().Err(err).Msg("error parsing transfer_call arguments")
					return
				}

				targetQueue, ok := args["target_queue"].(string)
				if !ok {
					logger.Error().Msg("transfer_call missing target_queue argument")
					return
				}

				logger.Info().Str("target_queue", targetQueue).Msg("transferring task to queue")

				updateAtts := map[string]interface{}{
					"target_queue": targetQueue,
				}

				if err := c.App.Twilio.UpdateTaskAttributes(*task.Sid, updateAtts); err != nil {
					logger.Error().Err(err).Msg("error updating task attributes")
					return
				}
			}
		}
	}
}
