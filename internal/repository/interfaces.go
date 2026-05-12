package repository

import (
	"context"
	"time"

	"github.com/stazoloto/zync/internal/domain"
)

type UserRepository interface {
	// Register регистрирует нового пользователя и возвращает JWT-токен для него.
	Register(email, firstName, lastName, passwordHash string) (string, error)
	// GetByEmail возвращает пользователя по email. Если пользователь не найден, возвращает sql.ErrNoRows.
	GetByEmail(email string) (*domain.User, error)
	// GetByID возвращает пользователя по ID. Если пользователь не найден, возвращает sql.ErrNoRows.
	GetByID(userID string) (*domain.User, error)
	// ListUsers возвращает список всех пользователей. Доступно только администратору.
	ListUsers() ([]*domain.User, error)
	// DeleteUser удаляет пользователя из системы.
	DeleteUser(userID string) error
	// SetRole устанавливает роль пользователю.
	SetRole(userID, role string) error
}

type RoomRepository interface {
	// CreateIfNotExists создает комнату, если ее еще нет. Если комната уже существует, ничего не делает.
	CreateIfNotExists(roomID string) error
	// SetInactive устанавливает комнату как неактивную.
	SetInactive(roomID string) error
	// ListActive возвращает список ID активных комнат.
	ListActive() ([]string, error)
}

type ParticipantRepository interface {
	// JoinRoom добавляет пользователя в комнату. Если пользователь уже в комнате, обновляет время последнего присоединения.
	JoinRoom(roomID, userID string) error
	// LeaveRoom удаляет пользователя из комнаты. Если пользователь не в комнате, ничего не делает.
	LeaveRoom(roomID, userID string) error
	// GetActiveParticipants возвращает список ID пользователей, которые в данный момент находятся в комнате.
	GetActiveParticipants(roomID string) ([]string, error)
	// HasActiveParticipants проверяет, есть ли в комнате активные участники.
	HasActiveParticipants(roomID string) (bool, error)
	// GetRole возвращает роль участника в комнате.
	GetRole(roomID, userID string) (domain.ParticipantRole, error)
}

type PresenceRepository interface {
	// AddParticipant добавляет участника в комнату. Если участник уже в комнате, обновляет время последнего присоединения.
	AddParticipant(ctx context.Context, roomID, userID string) error
	RemoveParticipant(ctx context.Context, roomID, userID string) error
	GetActiveParticipants(ctx context.Context, roomID string) ([]string, error)
	GetActiveParticipantRooms(ctx context.Context, userID string) ([]string, error)
}

type VerifyCodeRepository interface {
	SaveCode(ctx context.Context, email, code string, ttl time.Duration) error
	GetCode(ctx context.Context, email string) (string, error)
	DeleteCode(ctx context.Context, email string) error
}

type ChatSubscription interface {
	Channel() <-chan []byte
	Close() error
}

type ChatPublisher interface {
	Publish(ctx context.Context, roomID string, payload []byte) error
}

type ChatSubscriber interface {
	Subscribe(ctx context.Context, roomID string) (ChatSubscription, error)
}

type Participant struct {
	RoomID   string
	UserID   string
	JoinedAt time.Time
	LeftAt   *time.Time
}
