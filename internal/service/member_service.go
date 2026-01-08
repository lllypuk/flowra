package service

import (
	"context"
	"errors"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// MemberCommandRepository defines interface for commands (change state) chlenov workspace.
// interface declared on the consumer side according to principles Go interface design.
type MemberCommandRepository interface {
	// AddMember adds chlena in workspace
	AddMember(ctx context.Context, member *workspace.Member) error

	// RemoveMember udalyaet chlena from workspace
	RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error

	// UpdateMember obnovlyaet data chlena workspace
	UpdateMember(ctx context.Context, member *workspace.Member) error
}

// MemberQueryRepository defines interface for zaprosov (only reading) chlenov workspace.
// interface declared on the consumer side according to principles Go interface design.
type MemberQueryRepository interface {
	// FindByID finds workspace space po ID (for proverki suschestvovaniya)
	FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

	// GetMember returns chlena workspace po userID
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)

	// ListMembers returns all chlenov workspace
	ListMembers(ctx context.Context, workspaceID uuid.UUID, offset, limit int) ([]*workspace.Member, error)

	// CountMembers returns count chlenov workspace
	CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

// MemberService realizuet httphandler.MemberService
type MemberService struct {
	commandRepo MemberCommandRepository
	queryRepo   MemberQueryRepository
}

// NewMemberService sozdayot New MemberService.
func NewMemberService(
	commandRepo MemberCommandRepository,
	queryRepo MemberQueryRepository,
) *MemberService {
	return &MemberService{
		commandRepo: commandRepo,
		queryRepo:   queryRepo,
	}
}

// AddMember adds user in workspace.
func (s *MemberService) AddMember(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
	role workspace.Role,
) (*workspace.Member, error) {
	// verify, that workspace suschestvuet
	ws, err := s.queryRepo.FindByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errs.ErrNotFound
	}

	// verify, that user eschyo not chlen
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

// RemoveMember udalyaet user from workspace.
func (s *MemberService) RemoveMember(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) error {
	// verify, that member suschestvuet
	member, err := s.queryRepo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		return err
	}
	if member == nil {
		return errs.ErrNotFound
	}

	// nelzya delete owner
	if member.Role() == workspace.RoleOwner {
		return errs.ErrForbidden
	}

	return s.commandRepo.RemoveMember(ctx, workspaceID, userID)
}

// UpdateMemberRole obnovlyaet role participant.
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

	// nelzya change role owner
	if member.Role() == workspace.RoleOwner {
		return nil, errs.ErrForbidden
	}

	// nelzya value owner via it is method
	if role == workspace.RoleOwner {
		return nil, errs.ErrForbidden
	}

	// update role (immutable update)
	updatedMember := member.WithRole(role)

	if updateErr := s.commandRepo.UpdateMember(ctx, &updatedMember); updateErr != nil {
		return nil, updateErr
	}

	return &updatedMember, nil
}

// GetMember returns informatsiyu ob uchastnike.
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

// IsOwner checks, is li user vladeltsem workspace.
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
