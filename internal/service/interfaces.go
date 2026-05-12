package service

import (
	"context"

	"github.com/pion/webrtc/v3"
	"github.com/stazoloto/zync/internal/domain"
	"github.com/stazoloto/zync/internal/repository"
)

type RoomInfo struct {
	ID           string   `json:"id"`
	Participants []string `json:"participants"`
}

type AdminService interface {
	// ListUsers возвращает список всех пользователей. Доступно только администратору.
	ListUsers() ([]*domain.User, error)
	// DeleteUser удаляет пользователя из системы.
	DeleteUser(userID string) error
	// SetRole устанавливает роль пользователю.
	SetRole(userID, role string) error
	// ListActiveRooms возвращает активные комнаты с участниками из Redis.
	ListActiveRooms(ctx context.Context) ([]RoomInfo, error)
	// CloseRoom принудительно закрывает комнату.
	CloseRoom(ctx context.Context, roomID string) error
}

type UserService interface {
	// SendVerificationCode генерирует код, сохраняет его в репозитории и отправляет пользователю на email.
	SendVerificationCode(ctx context.Context, email string) error
	// VerifyCode проверяет код из email и возвращает JWT-токен для пользователя.
	VerifyCode(ctx context.Context, email, code string) (string, error)
	// Register регистрирует нового пользователя и возвращает JWT-токен для него.
	Register(ctx context.Context, verifiedToken, firstName, lastName, password string) (string, error)
	// Login проверяет email и пароль, и возвращает JWT-токен для пользователя.
	Login(ctx context.Context, email, password string) (string, error)
	// EmailExists проверяет, зарегистрирован ли пользователь с таким email.
	EmailExists(ctx context.Context, email string) (bool, error)
}

type RoomService interface {
	// JoinRoom добавляет пользователя в комнату и возвращает список активных участников.
	JoinRoom(ctx context.Context, roomID, userID string) error
	// GetActiveParticipants возвращает список ID пользователей, которые в данный момент находятся в комнате.
	GetActiveParticipants(ctx context.Context, roomID string) ([]string, error)
	// LeaveRoom удаляет пользователя из комнаты.
	LeaveRoom(ctx context.Context, roomID, userID string) error
	// OnDisconnect удаляет пользователя из всех комнат, в которых он находится. Вызывается при отключении от WebSocket.
	OnDisconnect(ctx context.Context, userID string) error
	// PublishMessage публикует сообщение в комнату, чтобы все участники получили его.
	PublishMessage(ctx context.Context, roomID string, payload []byte) error
	// SubscribeToRoom возвращает канал для получения сообщений из комнаты. Участник должен быть добавлен в комнату через JoinRoom.
	SubscribeToRoom(ctx context.Context, roomID string) (repository.ChatSubscription, error)
	// IsParticipant проверяет, является ли пользователь участником комнаты.
	IsParticipant(ctx context.Context, roomID, userID string) (bool, error)
	// GetParticipantRole возвращает роль пользователя в комнате.
	GetParticipantRole(ctx context.Context, roomID, userID string) (domain.ParticipantRole, error)
}

type MediaGateway interface {
	// Join добавляет участника в комнату и возвращает канал для отправки сигналов этому участнику.
	Join(roomID, peerID string, out chan<- domain.SignalMessage) error
	// Leave удаляет участника из комнаты и закрывает канал для отправки сигналов этому участнику.
	Leave(roomID, peerID string) error
	// HandleOffer обрабатывает SDP-оффер от участника и пересылает его другим участникам комнаты.
	HandleOffer(roomID, peerID string, offer webrtc.SessionDescription) error
	// HandleAnswer обрабатывает SDP-ответ от участника и пересылает его другим участникам комнаты.
	HandleAnswer(roomID, peerID string, answer webrtc.SessionDescription) error
	// HandleCandidate обрабатывает ICE-кандидата от участника и пересылает его другим участникам комнаты.
	HandleCandidate(roomID, peerID string, candidate webrtc.ICECandidateInit) error
}

type SignalingService interface {
	// Join добавляет участника в комнату и возвращает канал для отправки сигналов этому участнику.
	Join(ctx context.Context, roomID, peerID string, out chan<- domain.SignalMessage) error
	// Leave удаляет участника из комнаты и закрывает канал для отправки сигналов этому участнику.
	Leave(ctx context.Context, roomID, peerID string) error
	// HandleOffer обрабатывает SDP-оффер от участника и пересылает его другим участникам комнаты.
	HandleOffer(ctx context.Context, roomID, peerID string, offer webrtc.SessionDescription) error
	// HandleAnswer обрабатывает SDP-ответ от участника и пересылает его другим участникам комнаты.
	HandleAnswer(ctx context.Context, roomID, peerID string, answer webrtc.SessionDescription) error
	// HandleCandidate обрабатывает ICE-кандидата от участника и пересылает его другим участникам комнаты.
	HandleCandidate(ctx context.Context, roomID, peerID string, candidate webrtc.ICECandidateInit) error
}
