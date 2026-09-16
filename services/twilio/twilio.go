package comms_twilio

import (
	"encoding/json"
	"errors"
	"maps"
	"zennify/zenn-ai-assistant/config"

	"github.com/twilio/twilio-go"
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
	UpdateTaskAttributes(taskSid string, updatedAttributes map[string]any) error
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

func (s twilioClientImpl) FindTaskByCallSid(callSid string) (*taskrouter.TaskrouterV1Task, error) {
	// List tasks in the workspace
	tasks, err := s.client.TaskrouterV1.ListTask(s.workspaceSid, &taskrouter.ListTaskParams{})
	if err != nil {
		return nil, err
	}

	// Search through tasks for one with matching call_sid in attributes
	for i := range tasks {
		task := &tasks[i]
		if task.Attributes != nil {
			var attrs map[string]interface{}
			err := json.Unmarshal([]byte(*task.Attributes), &attrs)
			if err != nil {
				continue
			}

			// Check if call_sid matches
			if callSidAttr, ok := attrs["worker_call_sid"]; ok {
				if callSidAttr == callSid {
					return task, nil
				}
			}
		}
	}

	return nil, errors.New("task not found with worker_call_sid: " + callSid)
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

func twilioString(s string) *string {
	return &s
}
