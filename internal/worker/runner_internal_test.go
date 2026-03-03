package worker

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/config"
)

func TestSetupUserSyncWorker_DisabledSkipsKeycloakValidation(t *testing.T) {
	t.Setenv("USER_SYNC_DISABLED", "true")
	t.Setenv("USER_SYNC_INTERVAL", "")

	cfg := config.DefaultConfig()
	cfg.Keycloak.URL = ""
	cfg.Keycloak.AdminUsername = ""

	userSyncWorker, syncConfig, err := setupUserSyncWorker(cfg, nil, slog.Default())
	require.NoError(t, err)
	require.NotNil(t, userSyncWorker)
	require.False(t, syncConfig.Enabled)
	require.NoError(t, userSyncWorker.Run(context.Background()))
}

func TestSetupUserSyncWorker_EnabledRequiresKeycloakConfig(t *testing.T) {
	t.Setenv("USER_SYNC_DISABLED", "false")
	t.Setenv("USER_SYNC_INTERVAL", "")

	cfg := config.DefaultConfig()
	cfg.Keycloak.URL = ""
	cfg.Keycloak.AdminUsername = ""

	userSyncWorker, syncConfig, err := setupUserSyncWorker(cfg, nil, slog.Default())
	require.Error(t, err)
	require.Nil(t, userSyncWorker)
	require.Equal(t, UserSyncConfig{}, syncConfig)
	require.EqualError(t, err, "keycloak configuration is required for user sync worker")
}
