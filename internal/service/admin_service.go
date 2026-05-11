package service

import (
	"context"
	"errors"

	"github.com/stazoloto/zync/internal/domain"
	"github.com/stazoloto/zync/internal/repository"
	"go.uber.org/zap"
)

type adminService struct {
	users    repository.UserRepository
	rooms    repository.RoomRepository
	presence repository.PresenceRepository
	logger   *zap.Logger
}

func NewAdminService(
	users repository.UserRepository,
	rooms repository.RoomRepository,
	presence repository.PresenceRepository,
	logger *zap.Logger,
) AdminService {
	return &adminService{users: users, rooms: rooms, presence: presence, logger: logger}
}

func (s *adminService) ListUsers() ([]*domain.User, error) {
	return s.users.ListUsers()
}

func (s *adminService) DeleteUser(userID string) error {
	s.logger.Info("admin: delete user", zap.String("user_id", userID))
	return s.users.DeleteUser(userID)
}

func (s *adminService) SetRole(userID, role string) error {
	if role != domain.RoleUser && role != domain.RoleAdmin {
		return errors.New("invalid role")
	}
	s.logger.Info("admin: set role", zap.String("user_id", userID), zap.String("role", role))
	return s.users.SetRole(userID, role)
}

func (s *adminService) ListActiveRooms(ctx context.Context) ([]RoomInfo, error) {
	ids, err := s.rooms.ListActive()
	if err != nil {
		return nil, err
	}

	result := make([]RoomInfo, 0, len(ids))
	for _, id := range ids {
		participants, err := s.presence.GetActiveParticipants(ctx, id)
		if err != nil {
			participants = []string{}
		}
		result = append(result, RoomInfo{ID: id, Participants: participants})
	}
	return result, nil
}

func (s *adminService) CloseRoom(ctx context.Context, roomID string) error {
	s.logger.Info("admin: close room", zap.String("room_id", roomID))
	return s.rooms.SetInactive(roomID)
}
