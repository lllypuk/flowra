//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	messageapp "github.com/lllypuk/flowra/internal/application/message"
	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	messagedomain "github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/tag"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	mongodbinfra "github.com/lllypuk/flowra/internal/infrastructure/mongodb"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const e2eSystemBotUserID = "00000000-0000-0000-0000-000000000001"

type realE2EMessageService struct {
	send          *messageapp.SendMessageUseCase
	list          *messageapp.ListMessagesUseCase
	edit          *messageapp.EditMessageUseCase
	del           *messageapp.DeleteMessageUseCase
	get           *messageapp.GetMessageUseCase
	addAttachment *messageapp.AddAttachmentUseCase
}

func newRealE2EMessageService(t *testing.T, suite *E2ETestSuite) httphandler.MessageService {
	t.Helper()

	botUserID, err := uuid.ParseUUID(e2eSystemBotUserID)
	require.NoError(t, err)

	chatReadRepo := mongodb.NewMongoChatReadModelRepository(
		suite.MongoDB.Collection(mongodbinfra.CollectionChatReadModel),
		suite.EventStore,
	)

	tagUseCases := &tag.ChatUseCases{
		ChangeStatus: chatapp.NewChangeStatusUseCase(suite.ChatRepo),
		SetPriority:  chatapp.NewSetPriorityUseCase(suite.ChatRepo),
	}

	tagExecutor := tag.NewCommandExecutor(tagUseCases, suite.UserRepo)

	return &realE2EMessageService{
		send: messageapp.NewSendMessageUseCase(
			suite.MessageRepo,
			chatReadRepo,
			nil,
			suite.EventBus,
			tag.NewProcessor(),
			tagExecutor,
			botUserID,
		),
		list:          messageapp.NewListMessagesUseCase(suite.MessageRepo),
		edit:          messageapp.NewEditMessageUseCase(suite.MessageRepo, suite.EventBus),
		del:           messageapp.NewDeleteMessageUseCase(suite.MessageRepo, suite.EventBus),
		get:           messageapp.NewGetMessageUseCase(suite.MessageRepo),
		addAttachment: messageapp.NewAddAttachmentUseCase(suite.MessageRepo, suite.EventBus),
	}
}

func (s *realE2EMessageService) SendMessage(ctx context.Context, cmd messageapp.SendMessageCommand) (messageapp.Result, error) {
	return s.send.Execute(ctx, cmd)
}

func (s *realE2EMessageService) ListMessages(ctx context.Context, query messageapp.ListMessagesQuery) (messageapp.ListResult, error) {
	return s.list.Execute(ctx, query)
}

func (s *realE2EMessageService) EditMessage(ctx context.Context, cmd messageapp.EditMessageCommand) (messageapp.Result, error) {
	return s.edit.Execute(ctx, cmd)
}

func (s *realE2EMessageService) DeleteMessage(ctx context.Context, cmd messageapp.DeleteMessageCommand) (messageapp.Result, error) {
	return s.del.Execute(ctx, cmd)
}

func (s *realE2EMessageService) GetMessage(ctx context.Context, messageID uuid.UUID) (*messagedomain.Message, error) {
	result, err := s.get.Execute(ctx, messageapp.GetMessageQuery{MessageID: messageID})
	if err != nil {
		return nil, err
	}
	return result.Value, nil
}

func (s *realE2EMessageService) AddAttachment(ctx context.Context, cmd messageapp.AddAttachmentCommand) (messageapp.Result, error) {
	return s.addAttachment.Execute(ctx, cmd)
}

func NewRealMessageE2ETestSuite(t *testing.T) *E2ETestSuite {
	t.Helper()
	return newE2ETestSuite(t, func(suite *E2ETestSuite) {
		suite.MessageService = newRealE2EMessageService(t, suite)
	})
}

func createRealTaskChat(t *testing.T, suite *E2ETestSuite, workspaceID, creatorID uuid.UUID, title string) *chatdomain.Chat {
	t.Helper()

	c, err := chatdomain.NewChat(workspaceID, chatdomain.TypeDiscussion, true, creatorID)
	require.NoError(t, err)
	require.NoError(t, c.ConvertToTask(title, creatorID))
	require.NoError(t, suite.ChatRepo.Save(context.Background(), c))

	return c
}

type botFlowMessageListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Messages []botFlowMessageListResponseMessage `json:"messages"`
	} `json:"data"`
}

type botFlowMessageListResponseMessage struct {
	ID       string `json:"id"`
	ChatID   string `json:"chat_id"`
	SenderID string `json:"sender_id"`
	Content  string `json:"content"`
	Type     string `json:"type"`
	IsSystem bool   `json:"is_system"`
}

func waitForBotFlowMessages(
	t *testing.T,
	client *HTTPClient,
	workspaceID uuid.UUID,
	chatID uuid.UUID,
	predicate func(botFlowMessageListResponse) bool,
) botFlowMessageListResponse {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	var last botFlowMessageListResponse

	for time.Now().Before(deadline) {
		resp := client.Get("/workspaces/" + workspaceID.String() + "/chats/" + chatID.String() + "/messages")
		AssertStatus(t, resp, 200)
		last = ParseResponse[botFlowMessageListResponse](t, resp)

		if predicate(last) {
			return last
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for bot flow messages, got %d messages", len(last.Data.Messages))
	return botFlowMessageListResponse{}
}

func splitUserAndBotMessages(
	resp botFlowMessageListResponse,
) (userMsgs []botFlowMessageListResponseMessage, botMsgs []botFlowMessageListResponseMessage) {
	for _, msg := range resp.Data.Messages {
		if msg.Type == "bot" {
			botMsgs = append(botMsgs, msg)
			continue
		}
		userMsgs = append(userMsgs, msg)
	}
	return userMsgs, botMsgs
}

func TestBotResponse_TaggedMessageProducesBotResponse(t *testing.T) {
	suite := NewRealMessageE2ETestSuite(t)

	user := suite.CreateTestUser("botflowstatus")
	ws := suite.CreateTestWorkspace("Bot Flow Workspace", user)
	chat := createRealTaskChat(t, suite, ws.ID(), user.ID, "Bot Flow Task")

	client := suite.NewHTTPClient(user.Token)

	sendResp := client.Post("/workspaces/"+ws.ID().String()+"/chats/"+chat.ID().String()+"/messages", map[string]string{
		"content": "#status Done",
	})
	AssertStatus(t, sendResp, 201)

	msgs := waitForBotFlowMessages(t, client, ws.ID(), chat.ID(), func(resp botFlowMessageListResponse) bool {
		if !resp.Success || len(resp.Data.Messages) < 2 {
			return false
		}
		_, botMsgs := splitUserAndBotMessages(resp)
		return len(botMsgs) >= 1
	})

	userMsgs, botMsgs := splitUserAndBotMessages(msgs)
	require.Len(t, userMsgs, 1, "expected exactly one user message")
	require.Len(t, botMsgs, 1, "expected exactly one bot message")

	assert.Equal(t, "#status Done", userMsgs[0].Content)
	assert.Equal(t, "bot", botMsgs[0].Type)
	assert.True(t, botMsgs[0].IsSystem)
	assert.NotEmpty(t, botMsgs[0].Content)
	assert.Contains(t, botMsgs[0].Content, "Status changed to")
	assert.Contains(t, botMsgs[0].Content, "Done")
}

func TestBotResponse_InvalidTagProducesBotErrorMessage(t *testing.T) {
	suite := NewRealMessageE2ETestSuite(t)

	user := suite.CreateTestUser("botflowinvalid")
	ws := suite.CreateTestWorkspace("Bot Flow Invalid Workspace", user)
	chat := createRealTaskChat(t, suite, ws.ID(), user.ID, "Invalid Tag Task")

	client := suite.NewHTTPClient(user.Token)

	sendResp := client.Post("/workspaces/"+ws.ID().String()+"/chats/"+chat.ID().String()+"/messages", map[string]string{
		"content": "#status Completed",
	})
	AssertStatus(t, sendResp, 201)

	msgs := waitForBotFlowMessages(t, client, ws.ID(), chat.ID(), func(resp botFlowMessageListResponse) bool {
		if len(resp.Data.Messages) < 2 {
			return false
		}
		_, botMsgs := splitUserAndBotMessages(resp)
		return len(botMsgs) >= 1 && strings.Contains(strings.ToLower(botMsgs[0].Content), "invalid status")
	})

	userMsgs, botMsgs := splitUserAndBotMessages(msgs)
	require.Len(t, userMsgs, 1)
	require.Len(t, botMsgs, 1)
	assert.Equal(t, "#status Completed", userMsgs[0].Content)
	assert.Equal(t, "bot", botMsgs[0].Type)
	assert.True(t, botMsgs[0].IsSystem)
	assert.Contains(t, strings.ToLower(botMsgs[0].Content), "invalid status")
}

func TestBotResponse_MultipleTagsProduceSingleBotResponse(t *testing.T) {
	suite := NewRealMessageE2ETestSuite(t)

	user := suite.CreateTestUser("botflowmulti")
	ws := suite.CreateTestWorkspace("Bot Flow Multi Workspace", user)
	chat := createRealTaskChat(t, suite, ws.ID(), user.ID, "Multi Tag Task")

	client := suite.NewHTTPClient(user.Token)

	sendResp := client.Post("/workspaces/"+ws.ID().String()+"/chats/"+chat.ID().String()+"/messages", map[string]string{
		"content": "#status Done #priority High",
	})
	AssertStatus(t, sendResp, 201)

	msgs := waitForBotFlowMessages(t, client, ws.ID(), chat.ID(), func(resp botFlowMessageListResponse) bool {
		if len(resp.Data.Messages) < 2 {
			return false
		}
		_, botMsgs := splitUserAndBotMessages(resp)
		if len(botMsgs) != 1 {
			return false
		}
		content := botMsgs[0].Content
		return strings.Contains(content, "Status changed to") && strings.Contains(content, "Priority changed to")
	})

	userMsgs, botMsgs := splitUserAndBotMessages(msgs)
	require.Len(t, userMsgs, 1)
	require.Len(t, botMsgs, 1)
	assert.Equal(t, "#status Done #priority High", userMsgs[0].Content)
	assert.Equal(t, "bot", botMsgs[0].Type)
	assert.True(t, botMsgs[0].IsSystem)
	assert.Contains(t, botMsgs[0].Content, "Status changed to")
	assert.Contains(t, botMsgs[0].Content, "Done")
	assert.Contains(t, botMsgs[0].Content, "Priority changed to")
	assert.Contains(t, botMsgs[0].Content, "High")
}
