package user

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

const (
	// MaxListLimit maximum count users in odnom zaprose
	MaxListLimit = 100
)

// ListUsersUseCase handles retrieval list users
type ListUsersUseCase struct {
	userRepo Repository
}

// NewListUsersUseCase creates New ListUsersUseCase
func NewListUsersUseCase(userRepo Repository) *ListUsersUseCase {
	return &ListUsersUseCase{userRepo: userRepo}
}

// Execute performs retrieval list users
func (uc *ListUsersUseCase) Execute(
	ctx context.Context,
	query ListUsersQuery,
) (UsersListResult, error) {
	// validation
	if err := uc.validate(query); err != nil {
		return UsersListResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// retrieval obschego kolichestva
	totalCount, err := uc.userRepo.Count(ctx)
	if err != nil {
		return UsersListResult{}, fmt.Errorf("failed to get users count: %w", err)
	}

	// retrieval list users
	users, err := uc.userRepo.List(ctx, query.Offset, query.Limit)
	if err != nil {
		return UsersListResult{}, fmt.Errorf("failed to list users: %w", err)
	}

	return UsersListResult{
		Users:      users,
		TotalCount: totalCount,
		Offset:     query.Offset,
		Limit:      query.Limit,
	}, nil
}

func (uc *ListUsersUseCase) validate(query ListUsersQuery) error {
	if err := appcore.ValidateNonNegative("offset", query.Offset); err != nil {
		return err
	}
	if err := appcore.ValidatePositive("limit", query.Limit); err != nil {
		return err
	}
	if err := appcore.ValidateRange("limit", query.Limit, 1, MaxListLimit); err != nil {
		return err
	}
	return nil
}
