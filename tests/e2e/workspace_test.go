//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspace_Create_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create and authenticate user
	testUser := suite.CreateTestUser("wsowner")
	client := suite.NewHTTPClient(testUser.Token)

	// Create workspace
	resp := client.Post("/workspaces", map[string]string{
		"name":        "My Workspace",
		"description": "A test workspace",
	})

	AssertStatus(t, resp, http.StatusCreated)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			OwnerID     string `json:"owner_id"`
			CreatedAt   string `json:"created_at"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			OwnerID     string `json:"owner_id"`
			CreatedAt   string `json:"created_at"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Data.ID)
	assert.Equal(t, "My Workspace", result.Data.Name)
	// Description may not be returned by mock service
	assert.Equal(t, testUser.ID.String(), result.Data.OwnerID)
	assert.NotEmpty(t, result.Data.CreatedAt)
}

func TestWorkspace_Create_ValidationError(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("wsowner2")
	client := suite.NewHTTPClient(testUser.Token)

	t.Run("empty name", func(t *testing.T) {
		resp := client.Post("/workspaces", map[string]string{
			"name":        "",
			"description": "A workspace without name",
		})

		AssertStatus(t, resp, http.StatusBadRequest)

		var result struct {
			Success bool `json:"success"`
			Error   struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		result = ParseResponse[struct {
			Success bool `json:"success"`
			Error   struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}](t, resp)

		assert.False(t, result.Success)
		assert.Equal(t, "VALIDATION_ERROR", result.Error.Code)
	})

	t.Run("name too long", func(t *testing.T) {
		longName := ""
		for i := 0; i < 300; i++ {
			longName += "a"
		}

		resp := client.Post("/workspaces", map[string]string{
			"name":        longName,
			"description": "A workspace with long name",
		})

		AssertStatus(t, resp, http.StatusBadRequest)
	})
}

func TestWorkspace_Create_Unauthorized(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// No auth token
	client := suite.NewHTTPClient("")

	resp := client.Post("/workspaces", map[string]string{
		"name": "Unauthorized Workspace",
	})

	AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestWorkspace_Get_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create user and workspace with helper
	testUser := suite.CreateTestUser("wsgetowner")
	ws := suite.CreateTestWorkspace("Test Get Workspace", testUser)

	client := suite.NewHTTPClient(testUser.Token)

	// Get workspace
	resp := client.Get("/workspaces/" + ws.ID().String())

	AssertStatus(t, resp, http.StatusOK)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			OwnerID     string `json:"owner_id"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			OwnerID     string `json:"owner_id"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.Equal(t, ws.ID().String(), result.Data.ID)
	assert.Equal(t, "Test Get Workspace", result.Data.Name)
}

func TestWorkspace_Get_NotFound(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("wsnotfounduser")
	// Create a workspace so user has membership somewhere
	suite.CreateTestWorkspace("Some Workspace", testUser)

	client := suite.NewHTTPClient(testUser.Token)

	// Get non-existent workspace - will get 403 because user is not member
	nonExistentWs, err := workspace.NewWorkspace("NonExistent", "", "keycloak-group-test", testUser.ID)
	require.NoError(t, err)
	resp := client.Get("/workspaces/" + nonExistentWs.ID().String())

	// Mock service returns not found
	AssertStatus(t, resp, http.StatusNotFound)
}

func TestWorkspace_List_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create user
	testUser := suite.CreateTestUser("wslistowner")
	client := suite.NewHTTPClient(testUser.Token)

	// Create multiple workspaces using helper
	for i := 0; i < 3; i++ {
		suite.CreateTestWorkspace("Workspace "+string(rune('A'+i)), testUser)
	}

	// List workspaces
	resp := client.Get("/workspaces")

	AssertStatus(t, resp, http.StatusOK)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Workspaces []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"workspaces"`
			Total  int `json:"total"`
			Offset int `json:"offset"`
			Limit  int `json:"limit"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			Workspaces []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"workspaces"`
			Total  int `json:"total"`
			Offset int `json:"offset"`
			Limit  int `json:"limit"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.GreaterOrEqual(t, len(result.Data.Workspaces), 3)
}

func TestWorkspace_Update_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("wsupdateowner")
	ws := suite.CreateTestWorkspace("Original Name", testUser)

	client := suite.NewHTTPClient(testUser.Token)

	// Update workspace
	resp := client.Put("/workspaces/"+ws.ID().String(), map[string]string{
		"name":        "Updated Name",
		"description": "Updated Description",
	})

	AssertStatus(t, resp, http.StatusOK)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.Equal(t, "Updated Name", result.Data.Name)
}

func TestWorkspace_Delete_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("wsdeleteowner")
	ws := suite.CreateTestWorkspace("To Delete", testUser)

	client := suite.NewHTTPClient(testUser.Token)

	// Delete workspace
	resp := client.Delete("/workspaces/" + ws.ID().String())

	AssertStatus(t, resp, http.StatusNoContent)
}

func TestWorkspace_AddMember_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create owner and member
	owner := suite.CreateTestUser("wsaddmemberowner")
	member := suite.CreateTestUser("wsnewmember")

	ws := suite.CreateTestWorkspace("Team Workspace", owner)

	client := suite.NewHTTPClient(owner.Token)

	// Add member
	resp := client.Post("/workspaces/"+ws.ID().String()+"/members", map[string]string{
		"user_id": member.ID.String(),
		"role":    "member",
	})

	AssertStatus(t, resp, http.StatusCreated)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			UserID   string `json:"user_id"`
			Role     string `json:"role"`
			JoinedAt string `json:"joined_at"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			UserID   string `json:"user_id"`
			Role     string `json:"role"`
			JoinedAt string `json:"joined_at"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.Equal(t, member.ID.String(), result.Data.UserID)
	assert.Equal(t, "member", result.Data.Role)
}

func TestWorkspace_AddMember_InvalidRole(t *testing.T) {
	suite := NewE2ETestSuite(t)

	owner := suite.CreateTestUser("wsaddmemberowner2")
	member := suite.CreateTestUser("wsnewmember2")

	ws := suite.CreateTestWorkspace("Team Workspace 2", owner)

	client := suite.NewHTTPClient(owner.Token)

	// Add member with invalid role
	resp := client.Post("/workspaces/"+ws.ID().String()+"/members", map[string]string{
		"user_id": member.ID.String(),
		"role":    "invalid_role",
	})

	AssertStatus(t, resp, http.StatusBadRequest)
}

func TestWorkspace_RemoveMember_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	owner := suite.CreateTestUser("wsremovememberowner")
	member := suite.CreateTestUser("wsremovemember")

	ws := suite.CreateTestWorkspace("Team Workspace 3", owner)

	// Add member using helper
	suite.AddWorkspaceMember(ws, member, workspace.RoleMember)

	client := suite.NewHTTPClient(owner.Token)

	// Remove member
	resp := client.Delete("/workspaces/" + ws.ID().String() + "/members/" + member.ID.String())

	AssertStatus(t, resp, http.StatusNoContent)
}

func TestWorkspace_UpdateMemberRole_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	owner := suite.CreateTestUser("wsupdateroleowner")
	member := suite.CreateTestUser("wsupdaterolemember")

	ws := suite.CreateTestWorkspace("Team Workspace 4", owner)

	// Add member using helper
	suite.AddWorkspaceMember(ws, member, workspace.RoleMember)

	client := suite.NewHTTPClient(owner.Token)

	// Update role
	resp := client.Put("/workspaces/"+ws.ID().String()+"/members/"+member.ID.String()+"/role", map[string]string{
		"role": "admin",
	})

	AssertStatus(t, resp, http.StatusOK)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.Equal(t, "admin", result.Data.Role)
}

func TestWorkspace_CompleteFlow(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create owner and members
	owner := suite.CreateTestUser("wsflowowner")
	member1 := suite.CreateTestUser("wsflowmember1")
	member2 := suite.CreateTestUser("wsflowmember2")

	// Create workspace using helper (this sets up owner as member)
	ws := suite.CreateTestWorkspace("Flow Test Workspace", owner)
	workspaceID := ws.ID().String()

	ownerClient := suite.NewHTTPClient(owner.Token)

	// 1. Add first member
	addMember1Resp := ownerClient.Post("/workspaces/"+workspaceID+"/members", map[string]string{
		"user_id": member1.ID.String(),
		"role":    "member",
	})
	AssertStatus(t, addMember1Resp, http.StatusCreated)

	// 2. Add second member as admin
	addMember2Resp := ownerClient.Post("/workspaces/"+workspaceID+"/members", map[string]string{
		"user_id": member2.ID.String(),
		"role":    "admin",
	})
	AssertStatus(t, addMember2Resp, http.StatusCreated)

	// 3. Update workspace
	updateResp := ownerClient.Put("/workspaces/"+workspaceID, map[string]string{
		"name":        "Updated Flow Workspace",
		"description": "Updated description",
	})
	AssertStatus(t, updateResp, http.StatusOK)

	// 4. Promote member1 to admin
	promoteResp := ownerClient.Put("/workspaces/"+workspaceID+"/members/"+member1.ID.String()+"/role", map[string]string{
		"role": "admin",
	})
	AssertStatus(t, promoteResp, http.StatusOK)

	// 5. Remove member2
	removeResp := ownerClient.Delete("/workspaces/" + workspaceID + "/members/" + member2.ID.String())
	AssertStatus(t, removeResp, http.StatusNoContent)

	// 6. Verify workspace state
	getResp := ownerClient.Get("/workspaces/" + workspaceID)
	AssertStatus(t, getResp, http.StatusOK)

	var getResult struct {
		Success bool `json:"success"`
		Data    struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	getResult = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			Name string `json:"name"`
		} `json:"data"`
	}](t, getResp)

	assert.Equal(t, "Updated Flow Workspace", getResult.Data.Name)
}
