package mongodb_test

import (
	"context"
	"testing"
	"time"

	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/tests/mocks"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Helper functions

// setupTestRepository создает тестовые репозитории с mock event store
// Возвращает также readModelColl для тестов read model
func setupTestRepository(t *testing.T) (
	*mongodb.MongoChatRepository,
	*mongodb.MongoChatReadModelRepository,
	*mocks.MockEventStore,
	*mongo.Collection,
) {
	t.Helper()

	eventStore := mocks.NewMockEventStore()

	// Используем testcontainers для MongoDB 6
	_, db := testutil.SetupTestMongoDBWithClient(t)
	readModelColl := db.Collection("chat_read_model")

	// Создаем индексы
	ctx := context.Background()
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "chat_id", Value: 1}},
	}
	_, _ = readModelColl.Indexes().CreateOne(ctx, indexModel)

	commandRepo := mongodb.NewMongoChatRepository(eventStore, readModelColl)
	queryRepo := mongodb.NewMongoChatReadModelRepository(readModelColl, eventStore)

	return commandRepo, queryRepo, eventStore, readModelColl
}

// addChatToReadModel добавляет чат в read model для тестов
func addChatToReadModel(ctx context.Context, t *testing.T, coll *mongo.Collection, c *chat.Chat) {
	t.Helper()

	// Преобразуем участников в строки
	participantStrs := make([]string, len(c.Participants()))
	for i, p := range c.Participants() {
		participantStrs[i] = p.UserID().String()
	}

	// Формируем документ для read model
	doc := bson.M{
		"chat_id":      c.ID().String(),
		"workspace_id": c.WorkspaceID().String(),
		"type":         string(c.Type()),
		"is_public":    c.IsPublic(),
		"created_by":   c.CreatedBy().String(),
		"created_at":   c.CreatedAt(),
		"participants": participantStrs,
	}

	// Добавляем дополнительные поля для typed чатов
	if c.Type() != chat.TypeDiscussion {
		doc["title"] = c.Title()
		doc["status"] = c.Status()
		doc["priority"] = c.Priority()

		if c.AssigneeID() != nil {
			doc["assigned_to"] = c.AssigneeID().String()
		}

		if c.DueDate() != nil {
			doc["due_date"] = *c.DueDate()
		}

		if c.Type() == chat.TypeBug {
			doc["severity"] = c.Severity()
		}
	}

	_, err := coll.InsertOne(ctx, doc)
	require.NoError(t, err)
}

// Tests for MongoChatRepository (Command Repository)

// TestMongoChatRepository_Load_SaveRoundTrip проверяет сохранение и загрузку чата
func TestMongoChatRepository_Load_SaveRoundTrip(t *testing.T) {
	commandRepo, _, eventStore, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	// Создаем новый чат
	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()
	c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)

	// Добавляем участника
	err = c.AddParticipant(uuid.NewUUID(), chat.RoleMember)
	require.NoError(t, err)

	// Save
	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Проверяем, что события были сохранены
	assert.NotEmpty(t, eventStore.AllEvents()[c.ID().String()])

	// Load
	loaded, err := commandRepo.Load(ctx, c.ID())
	require.NoError(t, err)

	assert.Equal(t, c.ID(), loaded.ID())
	assert.Equal(t, c.WorkspaceID(), loaded.WorkspaceID())
	assert.Equal(t, c.Type(), loaded.Type())
	assert.Equal(t, c.IsPublic(), loaded.IsPublic())
	assert.Len(t, loaded.Participants(), len(c.Participants()))
}

// TestMongoChatRepository_Load_NotFound проверяет обработку missing chat
func TestMongoChatRepository_Load_NotFound(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	_, err := commandRepo.Load(ctx, uuid.NewUUID())
	assert.Equal(t, errs.ErrNotFound, err)
}

// TestMongoChatRepository_Load_InvalidInput проверяет валидацию входных данных
func TestMongoChatRepository_Load_InvalidInput(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	// Zero UUID
	_, err := commandRepo.Load(ctx, uuid.UUID(""))
	assert.Equal(t, errs.ErrInvalidInput, err)
}

// TestMongoChatRepository_Save_NoChanges проверяет сохранение без изменений
func TestMongoChatRepository_Save_NoChanges(t *testing.T) {
	commandRepo, _, eventStore, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()
	c, err := chat.NewChat(workspaceID, chat.TypeTask, false, userID)
	require.NoError(t, err)

	// Save once
	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	eventCountBefore := len(eventStore.AllEvents()[c.ID().String()])

	// Save again without changes
	err = commandRepo.Save(ctx, c)
	require.NoError(t, err) // Should be idempotent

	eventCountAfter := len(eventStore.AllEvents()[c.ID().String()])
	assert.Equal(t, eventCountBefore, eventCountAfter)
}

// TestMongoChatRepository_Save_InvalidInput проверяет валидацию nil чата
func TestMongoChatRepository_Save_InvalidInput(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	// Nil chat
	err := commandRepo.Save(ctx, nil)
	assert.Equal(t, errs.ErrInvalidInput, err)
}

// TestMongoChatRepository_GetEvents проверяет получение событий чата
func TestMongoChatRepository_GetEvents(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	// Создаем чат
	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()
	c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)

	// Добавляем участника
	err = c.AddParticipant(uuid.NewUUID(), chat.RoleMember)
	require.NoError(t, err)

	// Сохраняем
	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Получаем события
	events, err := commandRepo.GetEvents(ctx, c.ID())
	require.NoError(t, err)

	assert.NotEmpty(t, events)
	assert.GreaterOrEqual(t, len(events), 2) // ChatCreated + ParticipantAdded
}

// TestMongoChatRepository_GetEvents_NotFound проверяет обработку несуществующего чата
func TestMongoChatRepository_GetEvents_NotFound(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	_, err := commandRepo.GetEvents(ctx, uuid.NewUUID())
	assert.Equal(t, errs.ErrNotFound, err)
}

// TestMongoChatRepository_GetEvents_InvalidInput проверяет валидацию входных данных
func TestMongoChatRepository_GetEvents_InvalidInput(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	_, err := commandRepo.GetEvents(ctx, uuid.UUID(""))
	assert.Equal(t, errs.ErrInvalidInput, err)
}

// TestMongoChatRepository_TypedChats проверяет работу с typed чатами</parameter>
func TestMongoChatRepository_TypedChats(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	// Тестируем Task чат - создаем Discussion и конвертируем в Task
	taskChat, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)

	err = taskChat.ConvertToTask("Test Task", userID)
	require.NoError(t, err)

	err = taskChat.SetPriority("Medium", userID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, taskChat)
	require.NoError(t, err)

	loaded, err := commandRepo.Load(ctx, taskChat.ID())
	require.NoError(t, err)

	assert.Equal(t, chat.TypeTask, loaded.Type())
	assert.Equal(t, "Test Task", loaded.Title())
	assert.Equal(t, "Medium", loaded.Priority())
}

// TestMongoChatRepository_ParticipantManagement проверяет управление участниками
func TestMongoChatRepository_ParticipantManagement(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)

	// Добавляем участников
	participant1 := uuid.NewUUID()
	participant2 := uuid.NewUUID()

	err = c.AddParticipant(participant1, chat.RoleMember)
	require.NoError(t, err)

	err = c.AddParticipant(participant2, chat.RoleAdmin)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Загружаем и проверяем
	loaded, err := commandRepo.Load(ctx, c.ID())
	require.NoError(t, err)

	assert.Len(t, loaded.Participants(), 3) // Создатель + 2 участника
	assert.True(t, loaded.HasParticipant(userID))
	assert.True(t, loaded.HasParticipant(participant1))
	assert.True(t, loaded.HasParticipant(participant2))

	// Удаляем участника из загруженного агрегата
	err = loaded.RemoveParticipant(participant1)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, loaded)
	require.NoError(t, err)

	// Загружаем снова
	loaded, err = commandRepo.Load(ctx, c.ID())
	require.NoError(t, err)

	assert.Len(t, loaded.Participants(), 2)
	assert.False(t, loaded.HasParticipant(participant1))
}

// TestMongoChatRepository_ChatStatusChanges проверяет изменение статуса чата
func TestMongoChatRepository_ChatStatusChanges(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	// Создаем Discussion чат и конвертируем в Task
	c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)

	err = c.ConvertToTask("Test Task", userID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Изменяем статус
	err = c.ChangeStatus("In Progress", userID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Загружаем и проверяем
	loaded, err := commandRepo.Load(ctx, c.ID())
	require.NoError(t, err)

	assert.Equal(t, "In Progress", loaded.Status())

	// Изменяем приоритет используя loaded агрегат
	err = loaded.SetPriority("High", userID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, loaded)
	require.NoError(t, err)

	loaded2, err := commandRepo.Load(ctx, c.ID())
	require.NoError(t, err)

	assert.Equal(t, "High", loaded2.Priority())
}

// Tests for MongoChatReadModelRepository (Query Repository)

// TestMongoChatReadModelRepository_FindByWorkspace проверяет поиск чатов по workspace
func TestMongoChatReadModelRepository_FindByWorkspace(t *testing.T) {
	_, queryRepo, _, readModelColl := setupTestRepository(t)
	if queryRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	// Создаем несколько чатов в одном workspace
	for range 3 {
		c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
		require.NoError(t, err)

		addChatToReadModel(ctx, t, readModelColl, c)
	}

	// Создаем чат в другом workspace
	otherWorkspaceID := uuid.NewUUID()
	otherChat, err := chat.NewChat(otherWorkspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)

	addChatToReadModel(ctx, t, readModelColl, otherChat)

	filters := chatapp.Filters{
		Offset: 0,
		Limit:  10,
	}

	chats, err := queryRepo.FindByWorkspace(ctx, workspaceID, filters)
	require.NoError(t, err)

	assert.NotNil(t, chats)
	assert.Len(t, chats, 3)

	for _, c := range chats {
		assert.Equal(t, workspaceID, c.WorkspaceID)
	}
}

// TestMongoChatReadModelRepository_FindByWorkspace_WithTypeFilter проверяет фильтрацию по типу
func TestMongoChatReadModelRepository_FindByWorkspace_WithTypeFilter(t *testing.T) {
	_, queryRepo, _, readModelColl := setupTestRepository(t)
	if queryRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	// Создаем чаты разных типов
	taskChat, err := chat.NewChat(workspaceID, chat.TypeTask, false, userID)
	require.NoError(t, err)
	addChatToReadModel(ctx, t, readModelColl, taskChat)

	discussionChat, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)
	addChatToReadModel(ctx, t, readModelColl, discussionChat)

	chatType := chat.TypeTask
	filters := chatapp.Filters{
		Type:   &chatType,
		Offset: 0,
		Limit:  10,
	}

	chats, err := queryRepo.FindByWorkspace(ctx, workspaceID, filters)
	require.NoError(t, err)

	assert.NotNil(t, chats)
	assert.Len(t, chats, 1)
	assert.Equal(t, chat.TypeTask, chats[0].Type)
}

// TestMongoChatReadModelRepository_FindByParticipant проверяет поиск чатов по участнику
func TestMongoChatReadModelRepository_FindByParticipant(t *testing.T) {
	_, queryRepo, _, readModelColl := setupTestRepository(t)
	if queryRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	// Создаем несколько чатов и добавляем пользователя
	participantID := uuid.NewUUID()

	for range 3 {
		c, err := chat.NewChat(workspaceID, chat.TypeTask, false, userID)
		require.NoError(t, err)

		// Добавляем участника
		err = c.AddParticipant(participantID, chat.RoleMember)
		require.NoError(t, err)

		addChatToReadModel(ctx, t, readModelColl, c)
	}

	// Добавляем чат без этого участника
	otherChat, err := chat.NewChat(workspaceID, chat.TypeTask, false, userID)
	require.NoError(t, err)
	addChatToReadModel(ctx, t, readModelColl, otherChat)

	chats, err := queryRepo.FindByParticipant(ctx, participantID, 0, 10)
	require.NoError(t, err)

	assert.NotNil(t, chats)
	assert.Len(t, chats, 3)
}

// TestMongoChatReadModelRepository_Count проверяет подсчет чатов
func TestMongoChatReadModelRepository_Count(t *testing.T) {
	_, queryRepo, _, readModelColl := setupTestRepository(t)
	if queryRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	// Добавляем чаты в workspace
	for range 5 {
		c, err := chat.NewChat(workspaceID, chat.TypeTask, false, userID)
		require.NoError(t, err)

		addChatToReadModel(ctx, t, readModelColl, c)
	}

	// Добавляем чат в другой workspace
	otherWorkspaceID := uuid.NewUUID()
	otherChat, err := chat.NewChat(otherWorkspaceID, chat.TypeTask, false, userID)
	require.NoError(t, err)

	addChatToReadModel(ctx, t, readModelColl, otherChat)

	count, err := queryRepo.Count(ctx, workspaceID)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, count, 5)
}

// TestMongoChatRepository_InputValidation проверяет валидацию входных данных
func TestMongoChatRepository_InputValidation(t *testing.T) {
	commandRepo, queryRepo, _, _ := setupTestRepository(t)
	if commandRepo == nil || queryRepo == nil {
		return
	}

	ctx := context.Background()

	// Load с нулевым UUID
	_, err := commandRepo.Load(ctx, uuid.UUID(""))
	assert.Equal(t, errs.ErrInvalidInput, err)

	// Save с nil чатом
	err = commandRepo.Save(ctx, nil)
	assert.Equal(t, errs.ErrInvalidInput, err)

	// FindByWorkspace с нулевым UUID
	filters := chatapp.Filters{Offset: 0, Limit: 10}
	_, err = queryRepo.FindByWorkspace(ctx, uuid.UUID(""), filters)
	assert.Equal(t, errs.ErrInvalidInput, err)

	// FindByParticipant с нулевым UUID
	_, err = queryRepo.FindByParticipant(ctx, uuid.UUID(""), 0, 10)
	assert.Equal(t, errs.ErrInvalidInput, err)

	// Count с нулевым UUID
	_, err = queryRepo.Count(ctx, uuid.UUID(""))
	assert.Equal(t, errs.ErrInvalidInput, err)
}

// TestMongoChatRepository_ComplexWorkflow проверяет сложный workflow создания и обновления чата
func TestMongoChatRepository_ComplexWorkflow(t *testing.T) {
	commandRepo, queryRepo, _, _ := setupTestRepository(t)
	if commandRepo == nil || queryRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	creatorID := uuid.NewUUID()

	// 1. Создаем Discussion чат и конвертируем в Task
	c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, creatorID)
	require.NoError(t, err)

	err = c.ConvertToTask("Complex Task", creatorID)
	require.NoError(t, err)

	err = c.SetPriority("Medium", creatorID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Назначаем исполнителя
	assigneeID := uuid.NewUUID()
	err = c.AssignUser(&assigneeID, creatorID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Устанавливаем срок
	dueDate := time.Now().Add(24 * time.Hour)
	err = c.SetDueDate(&dueDate, creatorID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Добавляем участников
	participantID := uuid.NewUUID()
	err = c.AddParticipant(participantID, chat.RoleMember)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Изменяем статус
	err = c.ChangeStatus("In Progress", creatorID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Загружаем финальное состояние
	loaded, err := commandRepo.Load(ctx, c.ID())
	require.NoError(t, err)

	// Проверяем все изменения
	assert.Equal(t, chat.TypeTask, loaded.Type())
	assert.Equal(t, "Complex Task", loaded.Title())
	assert.Equal(t, "Medium", loaded.Priority())
	assert.Equal(t, "In Progress", loaded.Status())
	assert.Equal(t, assigneeID, *loaded.AssigneeID())
	assert.Len(t, loaded.Participants(), 2) // Создатель + участник

	// Проверяем события
	events, err := commandRepo.GetEvents(ctx, c.ID())
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(events), 6) // Минимум 6 событий для такого workflow
}
