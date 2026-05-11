package service

import (
	"context"

	"github.com/stazoloto/zync/internal/repository"
	"go.uber.org/zap"
)

type roomService struct {
	users          repository.UserRepository
	rooms          repository.RoomRepository
	participants   repository.ParticipantRepository
	presence       repository.PresenceRepository
	chatPublisher  repository.ChatPublisher
	chatSubscriber repository.ChatSubscriber
	logger         *zap.Logger
}

func NewRoomService(
	users repository.UserRepository,
	rooms repository.RoomRepository,
	participants repository.ParticipantRepository,
	presence repository.PresenceRepository,
	chatPublisher repository.ChatPublisher,
	chatSubscriber repository.ChatSubscriber,
	logger *zap.Logger,
) RoomService {
	return &roomService{
		users:          users,
		rooms:          rooms,
		participants:   participants,
		presence:       presence,
		chatPublisher:  chatPublisher,
		chatSubscriber: chatSubscriber,
		logger:         logger,
	}
}

// JoinRoom добавляет участника в комнату и сохраняет его присутствие
func (s *roomService) JoinRoom(ctx context.Context, roomID, userID string) error {
	s.logger.Info("join room", zap.String("user", userID), zap.String("room", roomID))

	if _, err := s.users.GetByID(userID); err != nil {
		return err
	}

	if err := s.rooms.CreateIfNotExists(roomID); err != nil {
		return err
	}

	if err := s.participants.JoinRoom(roomID, userID); err != nil {
		return err
	}

	if err := s.presence.AddParticipant(ctx, roomID, userID); err != nil {
		return err
	}

	return nil
}

// GetActiveParticipants возвращает список активных участников комнаты
func (s *roomService) GetActiveParticipants(ctx context.Context, roomID string) ([]string, error) {
	s.logger.Info("get active participants", zap.String("room", roomID))

	return s.presence.GetActiveParticipants(ctx, roomID)
}

// LeaveRoom удяляет участника из комнаты и помечает комнату неактивной, если участников больше нет
func (s *roomService) LeaveRoom(ctx context.Context, roomID, userID string) error {
	s.logger.Info("leave room", zap.String("user", userID), zap.String("room", roomID))

	if err := s.participants.LeaveRoom(roomID, userID); err != nil {
		return err
	}

	if err := s.presence.RemoveParticipant(ctx, roomID, userID); err != nil {
		s.logger.Error("remove participant from presence failed",
			zap.String("room", roomID),
			zap.String("user", userID),
			zap.Error(err),
		)
	}

	active, err := s.participants.HasActiveParticipants(roomID)
	if err != nil {
		return err
	}

	if !active {
		return s.rooms.SetInactive(roomID)
	}

	return nil
}

// OnDisconnect удаляет участника из всех комнат, в которых он был активен, и помечает эти комнаты неактивными, если участников больше нет
func (s *roomService) OnDisconnect(ctx context.Context, userID string) error {
	s.logger.Info("disconnect user", zap.String("user", userID))

	rooms, err := s.presence.GetActiveParticipantRooms(ctx, userID)
	if err != nil {
		return err
	}

	for _, roomID := range rooms {
		if err := s.participants.LeaveRoom(roomID, userID); err != nil {
			s.logger.Error("error leaving room", zap.String("room", roomID), zap.String("user", userID), zap.Error(err))
			continue
		}

		if err := s.presence.RemoveParticipant(ctx, roomID, userID); err != nil {
			s.logger.Error("remove participant from presence failed",
				zap.String("room", roomID),
				zap.String("user", userID),
				zap.Error(err),
			)
			continue
		}

		// Проверяем, активна ли комната
		active, err := s.participants.HasActiveParticipants(roomID)
		if err != nil {
			s.logger.Error("error checking active participants", zap.String("room", roomID), zap.String("user", userID), zap.Error(err))
			continue
		}
		if !active {
			if err := s.rooms.SetInactive(roomID); err != nil {
				s.logger.Error("error setting room inactive", zap.String("room", roomID), zap.Error(err))
			}
		}
	}

	return nil
}

func (s *roomService) PublishMessage(ctx context.Context, roomID string, payload []byte) error {
	s.logger.Info("publish message", zap.String("room", roomID), zap.Int("payload_size", len(payload)))
	return s.chatPublisher.Publish(ctx, roomID, payload)
}

func (s *roomService) SubscribeToRoom(ctx context.Context, roomID string) (repository.ChatSubscription, error) {
	s.logger.Info("subscribe to room", zap.String("room", roomID))
	return s.chatSubscriber.Subscribe(ctx, roomID)
}

func (s *roomService) IsParticipant(ctx context.Context, roomID, userID string) (bool, error) {
	participants, err := s.presence.GetActiveParticipants(ctx, roomID)
	if err != nil {
		return false, err
	}
	for _, id := range participants {
		if id == userID {
			return true, nil
		}
	}
	return false, nil

}
