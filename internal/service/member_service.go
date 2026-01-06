package service

import (
	"context"
	"errors"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// MemberCommandRepository определяет интерфейс для команд (изменение состояния) членов workspace.
// Интерфейс объявлен на стороне потребителя согласно принципам Go interface design.
type MemberCommandRepository interface {
	// AddMember добавляет члена в workspace
	AddMember(ctx context.Context, member *workspace.Member) error

	// RemoveMember удаляет члена из workspace
	RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error

	// UpdateMember обновляет данные члена workspace
	UpdateMember(ctx context.Context, member *workspace.Member) error
}

// MemberQueryRepository определяет интерфейс для запросов (только чтение) членов workspace.
// Интерфейс объявлен на стороне потребителя согласно принципам Go interface design.
type MemberQueryRepository interface {
	// FindByID находит рабочее пространство по ID (для проверки существования)
	FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

	// GetMember возвращает члена workspace по userID
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)

	// ListMembers возвращает всех членов workspace
	ListMembers(ctx context.Context, workspaceID uuid.UUID, offset, limit int) ([]*workspace.Member, error)

	// CountMembers возвращает количество членов workspace
	CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

// MemberService реализует httphandler.MemberService
type MemberService struct {
	commandRepo MemberCommandRepository
	queryRepo   MemberQueryRepository
}

// NewMemberService создаёт новый MemberService.
func NewMemberService(
	commandRepo MemberCommandRepository,
	queryRepo MemberQueryRepository,
) *MemberService {
	return &MemberService{
		commandRepo: commandRepo,
		queryRepo:   queryRepo,
	}
}

// AddMember добавляет пользователя в workspace.
func (s *MemberService) AddMember(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
	role workspace.Role,
) (*workspace.Member, error) {
	// Проверить, что workspace существует
	ws, err := s.queryRepo.FindByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errs.ErrNotFound
	}

	// Проверить, что пользователь ещё не член
	existing, err := s.queryRepo.GetMember(ctx, workspaceID, userID)
	if err != nil && !errors.Is(err, errs.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, errs.ErrAlreadyExists
	}

	// Создать member
	member := workspace.NewMember(userID, workspaceID, role)

	if addErr := s.commandRepo.AddMember(ctx, &member); addErr != nil {
		return nil, addErr
	}

	return &member, nil
}

// RemoveMember удаляет пользователя из workspace.
func (s *MemberService) RemoveMember(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) error {
	// Проверить, что member существует
	member, err := s.queryRepo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		return err
	}
	if member == nil {
		return errs.ErrNotFound
	}

	// Нельзя удалить owner
	if member.Role() == workspace.RoleOwner {
		return errs.ErrForbidden
	}

	return s.commandRepo.RemoveMember(ctx, workspaceID, userID)
}

// UpdateMemberRole обновляет роль участника.
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

	// Нельзя изменить роль owner
	if member.Role() == workspace.RoleOwner {
		return nil, errs.ErrForbidden
	}

	// Нельзя назначить owner через этот метод
	if role == workspace.RoleOwner {
		return nil, errs.ErrForbidden
	}

	// Обновить роль (immutable update)
	updatedMember := member.WithRole(role)

	if updateErr := s.commandRepo.UpdateMember(ctx, &updatedMember); updateErr != nil {
		return nil, updateErr
	}

	return &updatedMember, nil
}

// GetMember возвращает информацию об участнике.
func (s *MemberService) GetMember(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) (*workspace.Member, error) {
	return s.queryRepo.GetMember(ctx, workspaceID, userID)
}

// ListMembers возвращает список участников workspace.
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

// IsOwner проверяет, является ли пользователь владельцем workspace.
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
