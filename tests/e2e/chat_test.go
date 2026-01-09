//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChat_Create_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create user and workspace with helper
	testUser := suite.CreateTestUser("chatcreateowner")
	ws := suite.CreateTestWorkspace("Chat Workspace", testUser)

	client := suite.NewHTTPClient(testUser.Token)

	// Create chat
	resp := client.Post("/workspaces/"+ws.ID().String()+"/chats", map[string]interface{}{
		"name":            "General",
		"type":            "discussion",
		"is_public":       true,
		"participant_ids": []string{testUser.ID.String()},
	})

	AssertStatus(t, resp, http.StatusCreated)

	result := ParseResponse[ChatResponse](t, resp)

	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Data.ID)
	assert.Equal(t, ws.ID().String(), result.Data.WorkspaceID)
	assert.Equal(t, "discussion", result.Data.Type)
	assert.True(t, result.Data.IsPublic)
}

func TestChat_Create_TaskChat(t *testing.T) {
	suite := NewE2ETestSuite(t)

	user1 := suite.CreateTestUser("chattask1")
	ws := suite.CreateTestWorkspace("Task Chat Workspace", user1)

	client := suite.NewHTTPClient(user1.Token)

	// Create task chat
	resp := client.Post("/workspaces/"+ws.ID().String()+"/chats", map[string]interface{}{
		"name":            "My Task",
		"type":            "task",
		"is_public":       false,
		"participant_ids": []string{user1.ID.String()},
	})

	AssertStatus(t, resp, http.StatusCreated)

	result := ParseResponse[ChatResponse](t, resp)

	assert.True(t, result.Success)
	assert.Equal(t, "task", result.Data.Type)
}

func TestChat_Create_ValidationError(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("chatvalidation")
	ws := suite.CreateTestWorkspace("Validation Workspace", testUser)

	client := suite.NewHTTPClient(testUser.Token)

	t.Run("invalid type", func(t *testing.T) {
		resp := client.Post("/workspaces/"+ws.ID().String()+"/chats", map[string]interface{}{
			"name": "Test Chat",
			"type": "invalid_type",
		})

		AssertStatus(t, resp, http.StatusBadRequest)
	})
}

func TestChat_Create_Unauthorized(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create a workspace first so we have a valid ID
	testUser := suite.CreateTestUser("chatunauthowner")
	ws := suite.CreateTestWorkspace("Unauth Workspace", testUser)

	// No auth token
	client := suite.NewHTTPClient("")

	resp := client.Post("/workspaces/"+ws.ID().String()+"/chats", map[string]interface{}{
		"name": "Unauthorized Chat",
		"type": "discussion",
	})

	AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestChat_Get_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("chatgetowner")
	ws := suite.CreateTestWorkspace("Get Chat Workspace", testUser)

	client := suite.NewHTTPClient(testUser.Token)

	// Create a chat and add to mock service
	c, err := chat.NewChat(ws.ID(), chat.TypeDiscussion, true, testUser.ID)
	require.NoError(t, err)
	suite.MockChatService.AddChat(c)

	// Get chat
	resp := client.Get("/workspaces/" + ws.ID().String() + "/chats/" + c.ID().String())

	AssertStatus(t, resp, http.StatusOK)

	result := ParseResponse[ChatResponse](t, resp)

	assert.True(t, result.Success)
	assert.Equal(t, c.ID().String(), result.Data.ID)
}

func TestChat_Get_NotFound(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("chatnotfound")
	ws := suite.CreateTestWorkspace("NotFound Workspace", testUser)

	client := suite.NewHTTPClient(testUser.Token)

	// Create a non-existent chat ID
	nonExistentChat, err := chat.NewChat(ws.ID(), chat.TypeDiscussion, true, testUser.ID)
	require.NoError(t, err)

	resp := client.Get("/workspaces/" + ws.ID().String() + "/chats/" + nonExistentChat.ID().String())

	AssertStatus(t, resp, http.StatusNotFound)
}

func TestChat_List_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("chatlistowner")
	ws := suite.CreateTestWorkspace("List Chats Workspace", testUser)

	client := suite.NewHTTPClient(testUser.Token)

	// Create multiple chats
	for i := 0; i < 3; i++ {
		c, err := chat.NewChat(ws.ID(), chat.TypeDiscussion, true, testUser.ID)
		require.NoError(t, err)
		suite.MockChatService.AddChat(c)
	}

	// List chats
	resp := client.Get("/workspaces/" + ws.ID().String() + "/chats")

	AssertStatus(t, resp, http.StatusOK)

	result := ParseResponse[ChatListResponse](t, resp)

	assert.True(t, result.Success)
	assert.GreaterOrEqual(t, len(result.Data.Chats), 3)
}

func TestChat_Update_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("chatupdateowner")
	ws := suite.CreateTestWorkspace("Update Chat Workspace", testUser)

	client := suite.NewHTTPClient(testUser.Token)

	// Create a task chat (only typed chats can be renamed)
	c, err := chat.NewChat(ws.ID(), chat.TypeTask, true, testUser.ID)
	require.NoError(t, err)
	_ = c.ConvertToTask("Original Name", testUser.ID)
	suite.MockChatService.AddChat(c)

	// Update chat name
	resp := client.Put("/workspaces/"+ws.ID().String()+"/chats/"+c.ID().String(), map[string]string{
		"name": "Updated Name",
	})

	AssertStatus(t, resp, http.StatusOK)

	result := ParseResponse[ChatResponse](t, resp)

	assert.True(t, result.Success)
	assert.Equal(t, "Updated Name", result.Data.Name)
}

func TestChat_Delete_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("chatdeleteowner")
	ws := suite.CreateTestWorkspace("Delete Chat Workspace", testUser)

	client := suite.NewHTTPClient(testUser.Token)

	c, err := chat.NewChat(ws.ID(), chat.TypeDiscussion, true, testUser.ID)
	require.NoError(t, err)
	suite.MockChatService.AddChat(c)

	// Delete chat
	resp := client.Delete("/workspaces/" + ws.ID().String() + "/chats/" + c.ID().String())

	AssertStatus(t, resp, http.StatusNoContent)
}

func TestChat_AddParticipant_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	owner := suite.CreateTestUser("chataddparticipantowner")
	newMember := suite.CreateTestUser("chataddparticipant")
	ws := suite.CreateTestWorkspace("Participant Workspace", owner)

	// Add new member to workspace
	suite.AddWorkspaceMember(ws, newMember, workspace.RoleMember)

	client := suite.NewHTTPClient(owner.Token)

	c, err := chat.NewChat(ws.ID(), chat.TypeDiscussion, true, owner.ID)
	require.NoError(t, err)
	suite.MockChatService.AddChat(c)

	// Add participant
	resp := client.Post("/workspaces/"+ws.ID().String()+"/chats/"+c.ID().String()+"/participants", map[string]string{
		"user_id": newMember.ID.String(),
		"role":    "member",
	})

	AssertStatus(t, resp, http.StatusCreated)

	result := ParseResponse[ChatResponse](t, resp)

	assert.True(t, result.Success)
	assert.Equal(t, c.ID().String(), result.Data.ID)
	// Check that the new member was added to participants
	found := false
	for _, p := range result.Data.Participants {
		if p.UserID == newMember.ID.String() {
			found = true
			break
		}
	}
	assert.True(t, found, "new member should be in participants")
}

func TestChat_RemoveParticipant_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	owner := suite.CreateTestUser("chatremoveparticipantowner")
	member := suite.CreateTestUser("chatremoveparticipant")
	ws := suite.CreateTestWorkspace("Remove Participant Workspace", owner)

	// Add member to workspace
	suite.AddWorkspaceMember(ws, member, workspace.RoleMember)

	client := suite.NewHTTPClient(owner.Token)

	c, err := chat.NewChat(ws.ID(), chat.TypeDiscussion, true, owner.ID)
	require.NoError(t, err)
	// Add member to chat
	_ = c.AddParticipant(member.ID, chat.RoleMember)
	suite.MockChatService.AddChat(c)

	// Remove participant
	resp := client.Delete("/workspaces/" + ws.ID().String() + "/chats/" + c.ID().String() + "/participants/" + member.ID.String())

	AssertStatus(t, resp, http.StatusNoContent)
}

func TestChat_CompleteFlow(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create users
	owner := suite.CreateTestUser("chatflowowner")
	member1 := suite.CreateTestUser("chatflowmember1")
	member2 := suite.CreateTestUser("chatflowmember2")

	// Create workspace and add all members
	ws := suite.CreateTestWorkspace("Flow Workspace", owner)
	suite.AddWorkspaceMember(ws, member1, workspace.RoleMember)
	suite.AddWorkspaceMember(ws, member2, workspace.RoleMember)

	ownerClient := suite.NewHTTPClient(owner.Token)

	// 1. Create group chat
	createResp := ownerClient.Post("/workspaces/"+ws.ID().String()+"/chats", map[string]interface{}{
		"name":            "Team Discussion",
		"type":            "discussion",
		"is_public":       true,
		"participant_ids": []string{owner.ID.String()},
	})
	AssertStatus(t, createResp, http.StatusCreated)

	createResult := ParseResponse[ChatResponse](t, createResp)

	chatID := createResult.Data.ID
	require.NotEmpty(t, chatID)

	// 2. Add member1
	addMember1Resp := ownerClient.Post("/workspaces/"+ws.ID().String()+"/chats/"+chatID+"/participants", map[string]string{
		"user_id": member1.ID.String(),
		"role":    "member",
	})
	AssertStatus(t, addMember1Resp, http.StatusCreated)

	// 3. Add member2
	addMember2Resp := ownerClient.Post("/workspaces/"+ws.ID().String()+"/chats/"+chatID+"/participants", map[string]string{
		"user_id": member2.ID.String(),
		"role":    "member",
	})
	AssertStatus(t, addMember2Resp, http.StatusCreated)

	// 4. Remove member2
	removeResp := ownerClient.Delete("/workspaces/" + ws.ID().String() + "/chats/" + chatID + "/participants/" + member2.ID.String())
	AssertStatus(t, removeResp, http.StatusNoContent)

	// 5. Verify chat state
	getResp := ownerClient.Get("/workspaces/" + ws.ID().String() + "/chats/" + chatID)
	AssertStatus(t, getResp, http.StatusOK)

	getResult := ParseResponse[ChatResponse](t, getResp)

	assert.Equal(t, chatID, getResult.Data.ID)

	// 6. Delete chat
	deleteResp := ownerClient.Delete("/workspaces/" + ws.ID().String() + "/chats/" + chatID)
	AssertStatus(t, deleteResp, http.StatusNoContent)

	// 7. Verify chat is gone
	getAfterDeleteResp := ownerClient.Get("/workspaces/" + ws.ID().String() + "/chats/" + chatID)
	AssertStatus(t, getAfterDeleteResp, http.StatusNotFound)
}
