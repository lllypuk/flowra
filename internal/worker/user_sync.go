package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
)

// Default configuration values for user sync.
const (
	defaultSyncInterval  = 15 * time.Minute
	defaultSyncBatchSize = 100
)

// UserSyncConfig contains configuration for the user sync worker.
type UserSyncConfig struct {
	// Interval is the time between sync runs.
	Interval time.Duration

	// BatchSize is the number of users to fetch per batch.
	BatchSize int

	// Enabled determines if the worker should run.
	Enabled bool
}

// DefaultUserSyncConfig returns sensible default configuration.
func DefaultUserSyncConfig() UserSyncConfig {
	return UserSyncConfig{
		Interval:  defaultSyncInterval,
		BatchSize: defaultSyncBatchSize,
		Enabled:   true,
	}
}

// KeycloakUserClient is the interface for fetching users from Keycloak.
type KeycloakUserClient interface {
	ListUsers(ctx context.Context, first, limit int) ([]keycloak.User, error)
	CountUsers(ctx context.Context) (int, error)
}

// SyncUserRepository is the interface for user persistence operations needed by sync.
type SyncUserRepository interface {
	FindByExternalID(ctx context.Context, externalID string) (*user.User, error)
	Save(ctx context.Context, user *user.User) error
	ListExternalIDs(ctx context.Context) ([]string, error)
}

// UserSyncWorker handles periodic synchronization of users from Keycloak to MongoDB.
type UserSyncWorker struct {
	keycloakClient KeycloakUserClient
	userRepo       SyncUserRepository
	logger         *slog.Logger
	config         UserSyncConfig
}

// NewUserSyncWorker creates a new user sync worker.
func NewUserSyncWorker(
	keycloakClient KeycloakUserClient,
	userRepo SyncUserRepository,
	logger *slog.Logger,
	config UserSyncConfig,
) *UserSyncWorker {
	if logger == nil {
		logger = slog.Default()
	}

	return &UserSyncWorker{
		keycloakClient: keycloakClient,
		userRepo:       userRepo,
		logger:         logger,
		config:         config,
	}
}

// Run starts the sync worker and runs periodically until the context is cancelled.
func (w *UserSyncWorker) Run(ctx context.Context) error {
	if !w.config.Enabled {
		w.logger.InfoContext(ctx, "user sync worker is disabled")
		return nil
	}

	w.logger.InfoContext(ctx, "starting user sync worker",
		slog.Duration("interval", w.config.Interval),
		slog.Int("batch_size", w.config.BatchSize),
	)

	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	// Run immediately on start
	if err := w.Sync(ctx); err != nil {
		w.logger.ErrorContext(ctx, "initial user sync failed", slog.String("error", err.Error()))
	}

	for {
		select {
		case <-ctx.Done():
			w.logger.InfoContext(ctx, "user sync worker stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := w.Sync(ctx); err != nil {
				w.logger.ErrorContext(ctx, "user sync failed", slog.String("error", err.Error()))
			}
		}
	}
}

// SyncResult contains statistics about a sync operation.
type SyncResult struct {
	Synced      int
	Created     int
	Updated     int
	Deactivated int
	Errors      int
	Duration    time.Duration
}

// Sync performs a single synchronization of all users from Keycloak.
func (w *UserSyncWorker) Sync(ctx context.Context) error {
	start := time.Now()
	w.logger.InfoContext(ctx, "starting user sync")

	// Get total count from Keycloak
	totalCount, err := w.keycloakClient.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to count keycloak users: %w", err)
	}

	w.logger.DebugContext(ctx, "keycloak user count", slog.Int("count", totalCount))

	// Track seen external IDs to detect deleted users
	seenExternalIDs := make(map[string]bool, totalCount)

	result := SyncResult{}

	// Fetch and sync in batches
	for offset := 0; offset < totalCount; offset += w.config.BatchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		kcUsers, listErr := w.keycloakClient.ListUsers(ctx, offset, w.config.BatchSize)
		if listErr != nil {
			return fmt.Errorf("failed to list keycloak users at offset %d: %w", offset, listErr)
		}

		for _, kcUser := range kcUsers {
			seenExternalIDs[kcUser.ID] = true

			syncResult, syncErr := w.syncUser(ctx, kcUser)
			if syncErr != nil {
				w.logger.WarnContext(ctx, "failed to sync user",
					slog.String("keycloak_id", kcUser.ID),
					slog.String("username", kcUser.Username),
					slog.String("error", syncErr.Error()),
				)
				result.Errors++
				continue
			}

			result.Synced++
			switch syncResult {
			case syncResultCreated:
				result.Created++
			case syncResultUpdated:
				result.Updated++
			case syncResultNoChange:
				// No action needed
			}
		}

		w.logger.DebugContext(ctx, "processed batch",
			slog.Int("offset", offset),
			slog.Int("batch_size", len(kcUsers)),
		)
	}

	// Deactivate users not found in Keycloak
	deactivated, deactErr := w.deactivateMissingUsers(ctx, seenExternalIDs)
	if deactErr != nil {
		w.logger.WarnContext(ctx, "failed to deactivate missing users", slog.String("error", deactErr.Error()))
	}
	result.Deactivated = deactivated
	result.Duration = time.Since(start)

	w.logger.InfoContext(ctx, "user sync completed",
		slog.Int("synced", result.Synced),
		slog.Int("created", result.Created),
		slog.Int("updated", result.Updated),
		slog.Int("deactivated", result.Deactivated),
		slog.Int("errors", result.Errors),
		slog.Duration("duration", result.Duration),
	)

	return nil
}

type syncResultType int

const (
	syncResultNoChange syncResultType = iota
	syncResultCreated
	syncResultUpdated
)

func (w *UserSyncWorker) syncUser(ctx context.Context, kcUser keycloak.User) (syncResultType, error) {
	// Try to find existing user by external ID
	existing, err := w.userRepo.FindByExternalID(ctx, kcUser.ID)
	if err != nil && !errors.Is(err, errs.ErrNotFound) {
		return syncResultNoChange, fmt.Errorf("failed to find user by external ID: %w", err)
	}

	displayName := buildDisplayName(kcUser)

	if existing == nil {
		// Create new user
		newUser, createErr := user.NewUser(
			kcUser.ID,
			kcUser.Username,
			kcUser.Email,
			displayName,
		)
		if createErr != nil {
			return syncResultNoChange, fmt.Errorf("failed to create user: %w", createErr)
		}

		// Set active status based on Keycloak enabled flag
		newUser.SetActive(kcUser.Enabled)

		if saveErr := w.userRepo.Save(ctx, newUser); saveErr != nil {
			return syncResultNoChange, fmt.Errorf("failed to save new user: %w", saveErr)
		}

		w.logger.DebugContext(ctx, "created user from keycloak",
			slog.String("user_id", newUser.ID().String()),
			slog.String("keycloak_id", kcUser.ID),
			slog.String("username", kcUser.Username),
		)

		return syncResultCreated, nil
	}

	// Update existing user if needed
	if existing.UpdateFromSync(kcUser.Username, kcUser.Email, displayName, kcUser.Enabled) {
		if saveErr := w.userRepo.Save(ctx, existing); saveErr != nil {
			return syncResultNoChange, fmt.Errorf("failed to update user: %w", saveErr)
		}

		w.logger.DebugContext(ctx, "updated user from keycloak",
			slog.String("user_id", existing.ID().String()),
			slog.String("keycloak_id", kcUser.ID),
			slog.String("username", kcUser.Username),
		)

		return syncResultUpdated, nil
	}

	return syncResultNoChange, nil
}

func (w *UserSyncWorker) deactivateMissingUsers(ctx context.Context, seenExternalIDs map[string]bool) (int, error) {
	// Get all external IDs from local database
	localExternalIDs, err := w.userRepo.ListExternalIDs(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list local external IDs: %w", err)
	}

	var deactivated int
	for _, externalID := range localExternalIDs {
		if seenExternalIDs[externalID] {
			continue // User exists in Keycloak
		}

		// User not found in Keycloak, deactivate them
		localUser, findErr := w.userRepo.FindByExternalID(ctx, externalID)
		if findErr != nil {
			w.logger.WarnContext(ctx, "failed to find user for deactivation",
				slog.String("external_id", externalID),
				slog.String("error", findErr.Error()),
			)
			continue
		}

		if !localUser.IsActive() {
			continue // Already deactivated
		}

		localUser.SetActive(false)
		if saveErr := w.userRepo.Save(ctx, localUser); saveErr != nil {
			w.logger.WarnContext(ctx, "failed to deactivate user",
				slog.String("user_id", localUser.ID().String()),
				slog.String("external_id", externalID),
				slog.String("error", saveErr.Error()),
			)
			continue
		}

		w.logger.InfoContext(ctx, "deactivated user not found in keycloak",
			slog.String("user_id", localUser.ID().String()),
			slog.String("external_id", externalID),
			slog.String("username", localUser.Username()),
		)

		deactivated++
	}

	return deactivated, nil
}

// buildDisplayName creates a display name from Keycloak user data.
func buildDisplayName(kcUser keycloak.User) string {
	name := strings.TrimSpace(kcUser.FirstName + " " + kcUser.LastName)
	if name == "" {
		return kcUser.Username
	}
	return name
}

// SyncSingleUser synchronizes a single user from Keycloak by their external ID.
// This is useful for on-demand sync after login or profile updates.
func (w *UserSyncWorker) SyncSingleUser(ctx context.Context, kcUser keycloak.User) error {
	result, err := w.syncUser(ctx, kcUser)
	if err != nil {
		return err
	}

	switch result {
	case syncResultCreated:
		w.logger.DebugContext(ctx, "single user sync: created",
			slog.String("keycloak_id", kcUser.ID),
			slog.String("username", kcUser.Username),
		)
	case syncResultUpdated:
		w.logger.DebugContext(ctx, "single user sync: updated",
			slog.String("keycloak_id", kcUser.ID),
			slog.String("username", kcUser.Username),
		)
	case syncResultNoChange:
		// No action needed
	}

	return nil
}
