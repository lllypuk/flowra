package service

import (
	"context"
	"errors"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// MemberCommandRepository defines interface for commands (change state) членов workspace.
// interface declared on the consumer side according to principles Go interface design.
type MemberCommandRepository interface {
	// AddMember добавляет члена in workspace
	AddMember(ctx context.Context, member *workspace.Member) error

	// RemoveMember удаляет члена from workspace
	RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error

	// UpdateMember обновляет data члена workspace
	UpdateMember(ctx context.Context, member *workspace.Member) error
}

// MemberQueryRepository defines interface for запросов (only reading) членов workspace.
// interface declared on the consumer side according to principles Go interface design.
type MemberQueryRepository interface {
	// FindByID finds workspaceее пространство по ID (for проверки существования)
	FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

	// GetMember returns члена workspace по userID
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)

	// ListMembers returns all членов workspace
	ListMembers(ctx context.Context, workspaceID uuid.UUID, offset, limit int) ([]*workspace.Member, error)

	// CountMembers returns count членов workspace
	CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

// MemberService реализует httphandler.MemberService
type MemberService struct {
	commandRepo MemberCommandRepository
	queryRepo   MemberQueryRepository
}

// NewMemberService создаёт New MemberService.
func NewMemberService(
	commandRepo MemberCommandRepository,
	queryRepo MemberQueryRepository,
) *MemberService {
	return &MemberService{
		commandRepo: commandRepo,
		queryRepo:   queryRepo,
	}
}

// AddMember добавляет user in workspace.
func (s *MemberService) AddMember(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
	role workspace.Role,
) (*workspace.Member, error) {
	// verify, that workspace существует
	ws, err := s.queryRepo.FindByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errs.ErrNotFound
	}

	// verify, that userель ещё not член
	existing, err := s.queryRepo.GetMember(ctx, workspaceID, userID)
	if err != nil && !errors.Is(err, errs.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, errs.ErrAlreadyExists
	}

	// create member
	member := workspace.NewMember(userID, workspaceID, role)

	if addErr := s.commandRepo.AddMember(ctx, &member); addErr != nil {
		return nil, addErr
	}

	return &member, nil
}

// RemoveMember удаляет user from workspace.
func (s *MemberService) RemoveMember(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) error {
	// verify, that member существует
	member, err := s.queryRepo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		return err
	}
	if member == nil {
		return errs.ErrNotFound
	}

	// Нельзя delete owner
	if member.Role() == workspace.RoleOwner {
		return errs.ErrForbidden
	}

	return s.commandRepo.RemoveMember(ctx, workspaceID, userID)
}

// UpdateMemberRole обновляет роль participant.
func (s *MemberService) UpdateMemberRole(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
	role workspace.Role,
) (*workspace.Member, error) {
	member, err := s.queryRepo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, errs.ErrNotFound
	}

	// Нельзя change роль owner
	if member.Role() == workspace.RoleOwner {
		return nil, errs.ErrForbidden
	}

	// Нельзя наvalueить owner via it isт method
	if role == workspace.RoleOwner {
		return nil, errs.ErrForbidden
	}

	// update роль (immutable update)
	updatedMember := member.WithRole(role)

	if updateErr := s.commandRepo.UpdateMember(ctx, &updatedMember); updateErr != nil {
		return nil, updateErr
	}

	return &updatedMember, nil
}

// GetMember returns информацию об участнике.
func (s *MemberService) GetMember(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) (*workspace.Member, error) {
	return s.queryRepo.GetMember(ctx, workspaceID, userID)
}

// ListMembers returns list participants workspace.
func (s *MemberService) ListMembers(
	ctx context.Context,
	workspaceID uuid.UUID,
	offset, limit int,
) ([]*workspace.Member, int, error) {
	members, err := s.queryRepo.ListMembers(ctx, workspaceID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.queryRepo.CountMembers(ctx, workspaceID)
	if err != nil {
		return nil, 0, err
	}

	return members, total, nil
}

// IsOwner checks, is ли userель владельцем workspace.
func (s *MemberService) IsOwner(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) (bool, error) {
	member, err := s.queryRepo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if member == nil {
		return false, nil
	}

	return member.Role() == workspace.RoleOwner, nil
}
