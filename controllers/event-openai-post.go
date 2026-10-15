package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
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
	taskrouter "github.com/twilio/twilio-go/rest/taskrouter/v1"
)

// How long the assistant stays with the caller waiting for an agent to answer the
// transfer before it checks back in with them. It keeps waiting after that.
const warmTransferJoinTimeout = 2 * time.Minute

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

// realtimeSession serialises writes to the OpenAI realtime websocket. Both the read loop
// and the goroutine waiting on a warm transfer hand-off send events on it.
type realtimeSession struct {
	mu   sync.Mutex
	conn net.Conn
}

func (s *realtimeSession) send(event realtimeClientEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return wsutil.WriteClientText(s.conn, payload)
}

type realtimeClientEvent struct {
	Type     string                    `json:"type"`
	Response *realtimeResponseCreate   `json:"response,omitempty"`
	Item     *realtimeConversationItem `json:"item,omitempty"`
}

type realtimeResponseCreate struct {
	Instructions string `json:"instructions,omitempty"`
}

type realtimeConversationItem struct {
	Type    string                `json:"type"`
	Role    string                `json:"role,omitempty"`
	Content []realtimeContentPart `json:"content,omitempty"`
	CallID  string                `json:"call_id,omitempty"`
	Output  string                `json:"output,omitempty"`
}

type realtimeContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// warmTransfer tracks a transfer in flight on a call: the caller is still on the line
// with the assistant, a task is queued for the target queue, and the assistant is waiting
// to introduce the caller to whoever picks it up.
//
// Once an agent has joined, the assistant stays on the call. The agent usually has
// follow-up questions, and finishing the introduction is not the same as being finished,
// so leaving is a decision the assistant makes with end_call rather than something that
// happens to it when a response stops playing.
type warmTransfer struct {
	conferenceName string
	summary        string

	mu        sync.Mutex
	handedOff bool
}

func (t *warmTransfer) markHandedOff() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.handedOff = true
}

// hasHandedOff reports whether an agent has joined and been introduced. After that the
// assistant is a guest on someone else's call, so ending the call means leaving it rather
// than tearing it down.
func (t *warmTransfer) hasHandedOff() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.handedOff
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

	session := &realtimeSession{conn: conn}

	// Closed when this call is over, so a goroutine still waiting on a transfer hand-off
	// does not outlive it.
	callEnded := make(chan struct{})
	defer close(callEnded)

	var transfer *warmTransfer
	defer func() {
		if transfer != nil {
			c.App.WarmTransfers.Release(transfer.conferenceName)
		}
	}()

	// How the assistant got onto this call decides whether it greets and whether it may
	// end the call. A leg dialed in by a Flex transfer says so on its SIP invite; a plain
	// inbound call says nothing and is treated as the assistant's own call.
	mode := sipHeaderValue(event, models.SipHeaderVirtualAgentMode)
	if mode == "" {
		mode = models.VirtualAgentModeCold
	}

	logger = logger.With().Str("mode", mode).Logger()

	// give time for call to establish
	time.Sleep(2 * time.Second)

	if mode == models.VirtualAgentModeConference {
		// Conferenced in alongside a human agent. The customer is mid-conversation with
		// someone else, so the assistant listens until it is spoken to rather than talking
		// over them with a greeting.
		logger.Info().Msg("conferenced into a call in progress; listening rather than greeting")

		if err := session.send(realtimeClientEvent{
			Type: "conversation.item.create",
			Item: &realtimeConversationItem{
				Type: "message",
				Role: "system",
				Content: []realtimeContentPart{{
					Type: "input_text",
					Text: "You have been conferenced into a call that is already under way between a customer and a human agent. " +
						"Do not greet anyone and do not introduce yourself. Stay silent and listen until someone addresses you " +
						"directly or asks you a question, then answer briefly. The human agent is running this call, not you.",
				}},
			},
		}); err != nil {
			logger.Err(err).Msg("error briefing the assistant on the call it joined")
		}
	} else if err := session.send(realtimeClientEvent{Type: "response.create"}); err != nil {
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

			switch toolCallEvent.Name {
			case "end_call":
				task, _, err := c.taskForCall(event)
				if err != nil {
					logger.Error().Err(err).Msg("error finding the task to end")
					return
				}

				if _, err := c.App.Twilio.EndTask(*task.Sid); err != nil {
					logger.Error().Err(err).Msg("error ending task")
				}

				// Leaving is not the same as ending. Once the assistant has handed a caller
				// to an agent, or was conferenced in alongside one, other people are still
				// talking on this conference and tearing it down would hang up on them.
				// Otherwise the assistant's leg no longer ends the conference on exit, so it
				// has to be ended explicitly or the caller is left alone in an empty one.
				handedOver := transfer != nil && transfer.hasHandedOff()

				if mode == models.VirtualAgentModeConference || handedOver {
					logger.Info().Bool("afterWarmTransfer", handedOver).Msg("leaving the conference; the others keep the call")
				} else if conferenceSid := c.conferenceSidForTask(task, logger); conferenceSid != "" {
					if _, err := c.App.Twilio.EndConference(conferenceSid); err != nil {
						logger.Error().Err(err).Str("conferenceSid", conferenceSid).Msg("error ending conference")
					}
				}

				if err := openaiClient.Realtime.Calls.Hangup(ctx, event.Data.CallID); err != nil {
					logger.Err(err).Msg("error hanging up sip call")
				}

				return

			case "transfer_call":
				if transfer != nil {
					logger.Warn().Msg("transfer already in flight for this call; ignoring duplicate transfer_call")
					continue
				}

				// Unlike end_call, a transfer that cannot start is recoverable: the
				// assistant keeps the caller and carries on.
				var started *warmTransfer

				task, callSid, err := c.taskForCall(event)
				if err == nil {
					started, err = c.startWarmTransfer(toolCallEvent, task, callSid, session, callEnded, logger)
				}

				if err != nil {
					logger.Error().Err(err).Msg("error starting warm transfer")

					c.reportToolResult(session, toolCallEvent.CallID,
						`{"status":"failed","detail":"the transfer could not be started"}`,
						"Tell the caller you were not able to reach an agent right now and offer to keep helping them yourself.",
						logger)

					continue
				}

				transfer = started

				c.reportToolResult(session, toolCallEvent.CallID,
					`{"status":"queued","detail":"an agent has been requested and will join shortly"}`,
					"Tell the caller you are connecting them to an agent and that you will stay on the line until someone picks up. Keep it to one sentence. Ask the agent if they have any more questions. If not, you may the end_call tool.",
					logger)

			default:
				logger.Warn().Str("tool", toolCallEvent.Name).Msg("unhandled tool call")
			}
		}
	}
}

// startWarmTransfer queues a task for the target queue carrying the conference the caller
// is already on, and starts waiting for an agent to join it. The caller stays connected to
// the assistant throughout.
func (c OpenAIEventController) startWarmTransfer(
	toolCall openairt.ResponseFunctionCallArgumentsDoneEvent,
	task *taskrouter.TaskrouterV1Task,
	callSid string,
	session *realtimeSession,
	callEnded <-chan struct{},
	logger zerolog.Logger,
) (*warmTransfer, error) {
	args, err := ParseToolCallArguments(toolCall.Arguments)
	if err != nil {
		return nil, fmt.Errorf("parsing transfer_call arguments: %w", err)
	}

	targetQueue, _ := args["target_queue"].(string)
	if targetQueue == "" {
		targetQueue = models.TransferQueueEveryone
	}

	summary, _ := args["summary"].(string)
	reason, _ := args["reason"].(string)

	queue, err := c.App.Twilio.FindTaskQueueByName(targetQueue)
	if err != nil {
		return nil, fmt.Errorf("finding %q task queue: %w", targetQueue, err)
	}

	// The conference assignment instruction names the conference after the task sid, so
	// the task the assistant is working names the conference the caller is sitting in.
	// Twilio also records the conference sid and both legs on the task; pass those along
	// so the accepting worker does not have to look the conference up by name.
	conferenceName := *task.Sid

	var conference models.TaskConferenceAttributes
	if task.Attributes != nil {
		if conference, err = models.ParseTaskConferenceAttributes(*task.Attributes); err != nil {
			logger.Warn().Err(err).Msg("error reading conference details off the task; falling back to lookup by name")
		}
	}

	attributes := models.WarmTransferTaskAttributes{
		IsWarmTransfer:      true,
		ConferenceName:      conferenceName,
		ConferenceSid:       conference.Sid,
		TransferredFromTask: *task.Sid,
		TargetQueue:         targetQueue,
		Summary:             summary,
		Reason:              reason,
		CallSid:             callSid,
		CustomerCallSid:     conference.Participants.Customer,
	}

	attributesJSON, err := json.Marshal(attributes)
	if err != nil {
		return nil, fmt.Errorf("marshalling transfer task attributes: %w", err)
	}

	// Watch before the task exists: an agent can answer faster than this function
	// returns, and the join signal must not be missed.
	joined := c.App.WarmTransfers.Watch(conferenceName)

	var params taskrouter.CreateTaskParams

	params.SetTaskQueueSid(*queue.Sid)
	params.SetTaskChannel("voice")
	params.SetRoutingTarget(*queue.Sid)
	params.SetAttributes(string(attributesJSON))

	// Route through the same workflow as the call being transferred so the reservation
	// reaches the assignment callback that knows how to join the conference.
	if task.WorkflowSid != nil {
		params.SetWorkflowSid(*task.WorkflowSid)
	}

	transferTask, err := c.App.Twilio.CreateTask(params)
	if err != nil {
		c.App.WarmTransfers.Release(conferenceName)
		return nil, fmt.Errorf("creating transfer task in %q: %w", targetQueue, err)
	}

	logger.Info().
		Str("targetQueue", targetQueue).
		Str("transferTaskSid", *transferTask.Sid).
		Str("conferenceName", conferenceName).
		Msg("warm transfer task created; holding with the caller until an agent joins")

	transfer := &warmTransfer{conferenceName: conferenceName, summary: summary}

	go c.awaitAgentJoin(transfer, joined, session, callEnded, logger)

	return transfer, nil
}

// awaitAgentJoin waits for an agent's leg to join the caller's conference and then asks
// the assistant to introduce the caller to them.
func (c OpenAIEventController) awaitAgentJoin(
	transfer *warmTransfer,
	joined <-chan string,
	session *realtimeSession,
	callEnded <-chan struct{},
	logger zerolog.Logger,
) {
	waited := time.NewTimer(warmTransferJoinTimeout)
	defer waited.Stop()

	for {
		select {
		case participantCallSid := <-joined:
			logger.Info().
				Str("participantCallSid", participantCallSid).
				Str("conferenceName", transfer.conferenceName).
				Msg("agent joined the conference; briefing them")

			// From here on, ending the call means leaving it rather than ending it for
			// the caller and the agent.
			transfer.markHandedOff()

			briefing := "An agent has just joined the call. Speak to the agent, not the caller. " +
				"Introduce the caller and hand off with this summary: " + transfer.summary +
				" Keep the introduction under fifteen seconds. Then stay on the line and answer any " +
				"follow-up questions the agent asks you. Only once the agent has everything they need, " +
				"say goodbye and use the end_call tool to leave them to it. Do not use end_call while " +
				"the agent is still asking you things."

			if err := session.send(realtimeClientEvent{
				Type:     "response.create",
				Response: &realtimeResponseCreate{Instructions: briefing},
			}); err != nil {
				logger.Err(err).Msg("error sending warm transfer briefing to openai")
			}

			return

		case <-waited.C:
			logger.Warn().
				Dur("waited", warmTransferJoinTimeout).
				Str("conferenceName", transfer.conferenceName).
				Msg("no agent has joined the conference yet; still holding")

			if err := session.send(realtimeClientEvent{
				Type: "response.create",
				Response: &realtimeResponseCreate{
					Instructions: "No agent has picked up yet. Let the caller know you are still holding for someone and ask whether they would like to keep waiting.",
				},
			}); err != nil {
				logger.Err(err).Msg("error notifying caller of warm transfer delay")
			}

			waited.Reset(warmTransferJoinTimeout)

		case <-callEnded:
			return
		}
	}
}

// reportToolResult hands a tool call's outcome back to the model and asks it to say
// something about it, so the caller is not left in silence after a tool runs.
func (c OpenAIEventController) reportToolResult(session *realtimeSession, toolCallID string, output string, instructions string, logger zerolog.Logger) {
	if err := session.send(realtimeClientEvent{
		Type: "conversation.item.create",
		Item: &realtimeConversationItem{
			Type:   "function_call_output",
			CallID: toolCallID,
			Output: output,
		},
	}); err != nil {
		logger.Err(err).Msg("error sending tool result to openai")
		return
	}

	if err := session.send(realtimeClientEvent{
		Type:     "response.create",
		Response: &realtimeResponseCreate{Instructions: instructions},
	}); err != nil {
		logger.Err(err).Msg("error requesting response after tool result")
	}
}

// taskForCall finds the Task Router task this realtime call is the worker leg of, keyed
// off the Twilio call sid Twilio puts on the SIP invite.
func (c OpenAIEventController) taskForCall(event webhooks.RealtimeCallIncomingWebhookEvent) (*taskrouter.TaskrouterV1Task, string, error) {
	callSid := twilioCallSidFromSipHeaders(event)

	// A leg dialed in by a transfer carries its task sid, which saves scanning the
	// workspace and is set before the assistant's leg is recorded on the task.
	if taskSid := sipHeaderValue(event, models.SipHeaderTaskSid); taskSid != "" {
		task, err := c.App.Twilio.FetchTask(taskSid)
		if err != nil {
			return nil, callSid, err
		}

		return task, callSid, nil
	}

	if callSid == "" {
		return nil, "", errors.New("no X-Twilio-CallSid sip header on the call")
	}

	task, err := c.App.Twilio.FindTaskByCallSid(callSid)
	if err != nil {
		return nil, callSid, err
	}

	return task, callSid, nil
}

// conferenceSidForTask resolves the conference a task's legs are on, preferring the sid
// Task Router records on the task and falling back to the conference named after the task.
// Returns "" when the call is not on a conference at all.
func (c OpenAIEventController) conferenceSidForTask(task *taskrouter.TaskrouterV1Task, logger zerolog.Logger) string {
	if task.Attributes != nil {
		conference, err := models.ParseTaskConferenceAttributes(*task.Attributes)
		if err != nil {
			logger.Warn().Err(err).Msg("error reading conference details off the task")
		} else if conference.Sid != "" {
			return conference.Sid
		}
	}

	conference, err := c.App.Twilio.FindConferenceByName(*task.Sid)
	if err != nil {
		logger.Warn().Err(err).Str("conferenceName", *task.Sid).Msg("no conference found for this call")
		return ""
	}

	return *conference.Sid
}

func sipHeaderValue(event webhooks.RealtimeCallIncomingWebhookEvent, name string) string {
	for _, header := range event.Data.SipHeaders {
		if strings.EqualFold(header.Name, name) {
			return header.Value
		}
	}

	return ""
}

func twilioCallSidFromSipHeaders(event webhooks.RealtimeCallIncomingWebhookEvent) string {
	return sipHeaderValue(event, "X-Twilio-CallSid")
}
