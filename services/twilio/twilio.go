package comms_twilio

import (
	"encoding/json"
	"errors"
	"maps"
	"zennify/zenn-ai-assistant/config"

	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
	conversations "github.com/twilio/twilio-go/rest/conversations/v1"
	taskrouter "github.com/twilio/twilio-go/rest/taskrouter/v1"
)

type TwilioClient interface {
	AcceptReservaton(reservationSid string, taskSid string, params taskrouter.UpdateTaskReservationParams) (*taskrouter.TaskrouterV1TaskReservation, error)
	FetchConversation(conversationSid string) (*conversations.ConversationsV1Conversation, error)
	UpdateConversationWebhook(conversationSid string, params conversations.CreateConversationScopedWebhookParams) (*conversations.ConversationsV1ConversationScopedWebhook, error)
	CloseConversation(conversationSid string) (*conversations.ConversationsV1Conversation, error)
	UpdateConversation(conversationSid string, params conversations.UpdateConversationParams) (*conversations.ConversationsV1Conversation, error)
	SendConversationMessage(conversationSid string, params conversations.CreateConversationMessageParams) (*conversations.ConversationsV1ConversationMessage, error)
	UpdateTask(taskSid string, params taskrouter.UpdateTaskParams) (*taskrouter.TaskrouterV1Task, error)
	EndTask(taskSid string) (*taskrouter.TaskrouterV1Task, error)
	FindTaskByCallSid(callSid string) (*taskrouter.TaskrouterV1Task, error)
	FetchTask(taskSid string) (*taskrouter.TaskrouterV1Task, error)
	UpdateTaskAttributes(taskSid string, updatedAttributes map[string]any) error
	FindTaskQueueByName(friendlyName string) (*taskrouter.TaskrouterV1TaskQueue, error)
	CreateTask(params taskrouter.CreateTaskParams) (*taskrouter.TaskrouterV1Task, error)
	FetchWorker(workerSid string) (*taskrouter.TaskrouterV1Worker, error)
	FindConferenceByName(friendlyName string) (*api.ApiV2010Conference, error)
	AddConferenceParticipant(conferenceSid string, params api.CreateParticipantParams) (*api.ApiV2010Participant, error)
	EndConference(conferenceSid string) (*api.ApiV2010Conference, error)
}

type twilioClientImpl struct {
	client         *twilio.RestClient
	workspaceSid   string
	webhookBaseUrl string
}

func NewTwilioClient(config config.TwilioConfig) TwilioClient {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username:   config.AccountSID,
		Password:   config.AuthToken,
		AccountSid: config.AccountSID,
	})

	return &twilioClientImpl{
		client:         client,
		workspaceSid:   config.WorkspaceSid,
		webhookBaseUrl: config.WebhookBaseUrl,
	}
}

func (s twilioClientImpl) AcceptReservaton(reservationSid string, taskSid string, params taskrouter.UpdateTaskReservationParams) (*taskrouter.TaskrouterV1TaskReservation, error) {
	params.SetReservationStatus("accepted")

	return s.client.TaskrouterV1.UpdateTaskReservation(s.workspaceSid,
		taskSid,
		reservationSid,
		&params)
}

func (s twilioClientImpl) FetchConversation(conversationSid string) (*conversations.ConversationsV1Conversation, error) {
	return s.client.ConversationsV1.FetchConversation(conversationSid)
}

func (s twilioClientImpl) UpdateConversation(conversationSid string, params conversations.UpdateConversationParams) (*conversations.ConversationsV1Conversation, error) {
	return s.client.ConversationsV1.UpdateConversation(conversationSid,
		&params)
}

func (s twilioClientImpl) CloseConversation(conversationSid string) (*conversations.ConversationsV1Conversation, error) {
	var params conversations.UpdateConversationParams

	params.SetState("closed")
	return s.UpdateConversation(conversationSid, params)
}

func (s twilioClientImpl) UpdateConversationWebhook(conversationSid string, params conversations.CreateConversationScopedWebhookParams) (*conversations.ConversationsV1ConversationScopedWebhook, error) {
	params.SetConfigurationUrl(s.webhookBaseUrl + "/twilio/conversation/event")
	return s.client.ConversationsV1.CreateConversationScopedWebhook(conversationSid,
		&params)
}

func (s twilioClientImpl) SendConversationMessage(conversationSid string, params conversations.CreateConversationMessageParams) (*conversations.ConversationsV1ConversationMessage, error) {
	return s.client.ConversationsV1.CreateConversationMessage(conversationSid,
		&params)
}

func (s twilioClientImpl) UpdateTask(taskSid string, params taskrouter.UpdateTaskParams) (*taskrouter.TaskrouterV1Task, error) {
	return s.client.TaskrouterV1.UpdateTask(s.workspaceSid,
		taskSid,
		&params)
}

func (s twilioClientImpl) EndTask(taskSid string) (*taskrouter.TaskrouterV1Task, error) {
	var params taskrouter.UpdateTaskParams

	params.SetAssignmentStatus("completed")
	return s.UpdateTask(taskSid, params)
}

// taskCallSidAttributes covers both shapes Task Router uses to record the worker's call
// leg: the dequeue instruction writes worker_call_sid, the conference instruction records
// the legs under conference.participants instead.
type taskCallSidAttributes struct {
	WorkerCallSid string `json:"worker_call_sid"`
	Conference    struct {
		Sid          string `json:"sid"`
		Participants struct {
			Worker   string `json:"worker"`
			Customer string `json:"customer"`
		} `json:"participants"`
	} `json:"conference"`
}

func (s twilioClientImpl) FindTaskByCallSid(callSid string) (*taskrouter.TaskrouterV1Task, error) {
	// List tasks in the workspace
	tasks, err := s.client.TaskrouterV1.ListTask(s.workspaceSid, &taskrouter.ListTaskParams{})
	if err != nil {
		return nil, err
	}

	// Search through tasks for one whose worker leg is this call
	for i := range tasks {
		task := &tasks[i]
		if task.Attributes == nil {
			continue
		}

		var attrs taskCallSidAttributes
		if err := json.Unmarshal([]byte(*task.Attributes), &attrs); err != nil {
			continue
		}

		if attrs.WorkerCallSid == callSid || attrs.Conference.Participants.Worker == callSid {
			return task, nil
		}
	}

	return nil, errors.New("task not found for worker call sid: " + callSid)
}

func (s twilioClientImpl) FetchTask(taskSid string) (*taskrouter.TaskrouterV1Task, error) {
	return s.client.TaskrouterV1.FetchTask(s.workspaceSid, taskSid)
}

func (s twilioClientImpl) UpdateTaskAttributes(taskSid string, updatedAttributes map[string]any) error {
	var params taskrouter.UpdateTaskParams
	var existingAttributes map[string]interface{}

	task, err := s.client.TaskrouterV1.FetchTask(s.workspaceSid, taskSid)
	if err != nil {
		return err
	}

	if task.Attributes != nil {
		if err := json.Unmarshal([]byte(*task.Attributes), &existingAttributes); err != nil {
			return err
		}
	}

	maps.Copy(existingAttributes, updatedAttributes)

	combinedAttributes, err := json.Marshal(existingAttributes)
	if err != nil {
		return err
	}

	params.SetAttributes(string(combinedAttributes))

	if _, err := s.UpdateTask(taskSid, params); err != nil {
		return err
	}

	return nil
}

// FindTaskQueueByName resolves a Task Router queue by its friendly name, e.g. "Everyone".
func (s twilioClientImpl) FindTaskQueueByName(friendlyName string) (*taskrouter.TaskrouterV1TaskQueue, error) {
	var params taskrouter.ListTaskQueueParams

	params.SetFriendlyName(friendlyName)
	params.SetLimit(1)

	queues, err := s.client.TaskrouterV1.ListTaskQueue(s.workspaceSid, &params)
	if err != nil {
		return nil, err
	}

	if len(queues) == 0 {
		return nil, errors.New("task queue not found with friendly name: " + friendlyName)
	}

	return &queues[0], nil
}

func (s twilioClientImpl) CreateTask(params taskrouter.CreateTaskParams) (*taskrouter.TaskrouterV1Task, error) {
	return s.client.TaskrouterV1.CreateTask(s.workspaceSid, &params)
}

func (s twilioClientImpl) FetchWorker(workerSid string) (*taskrouter.TaskrouterV1Worker, error) {
	return s.client.TaskrouterV1.FetchWorker(s.workspaceSid, workerSid)
}

// FindConferenceByName resolves the live conference Task Router created for a task. The
// conference assignment instruction names the conference after the task sid.
func (s twilioClientImpl) FindConferenceByName(friendlyName string) (*api.ApiV2010Conference, error) {
	var params api.ListConferenceParams

	params.SetFriendlyName(friendlyName)
	params.SetStatus("in-progress")
	params.SetLimit(1)

	conferences, err := s.client.Api.ListConference(&params)
	if err != nil {
		return nil, err
	}

	if len(conferences) == 0 {
		return nil, errors.New("no in-progress conference found with friendly name: " + friendlyName)
	}

	return &conferences[0], nil
}

func (s twilioClientImpl) AddConferenceParticipant(conferenceSid string, params api.CreateParticipantParams) (*api.ApiV2010Participant, error) {
	return s.client.Api.CreateParticipant(conferenceSid, &params)
}

// EndConference hangs up every remaining leg of a conference. The assistant's own leg is
// configured not to end the conference when it exits, so this is how a call that finishes
// with the assistant still on it gets torn down.
func (s twilioClientImpl) EndConference(conferenceSid string) (*api.ApiV2010Conference, error) {
	var params api.UpdateConferenceParams

	params.SetStatus("completed")

	return s.client.Api.UpdateConference(conferenceSid, &params)
}

func twilioString(s string) *string {
	return &s
}
