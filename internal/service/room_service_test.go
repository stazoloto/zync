package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	mocksrepo "github.com/stazoloto/zync/internal/mocks/repository"
	"github.com/stazoloto/zync/internal/service"
)

// newRoomService создаёт RoomService с моками всех зависимостей.
func newRoomService(
	t *testing.T,
	users *mocksrepo.MockUserRepository,
	rooms *mocksrepo.MockRoomRepository,
	participants *mocksrepo.MockParticipantRepository,
	presence *mocksrepo.MockPresenceRepository,
	publisher *mocksrepo.MockChatPublisher,
	subscriber *mocksrepo.MockChatSubscriber,
) service.RoomService {
	t.Helper()
	return service.NewRoomService(
		users,
		rooms,
		participants,
		presence,
		publisher,
		subscriber,
		zap.NewNop(), // логгер который ничего не пишет
	)
}

// --- JoinRoom ---

// TestRoomService_JoinRoom_Success проверяет что при успешном join
// все репозитории вызываются в правильном порядке.
func TestRoomService_JoinRoom_Success(t *testing.T) {
	ctx := context.Background()

	users := mocksrepo.NewMockUserRepository(t)
	rooms := mocksrepo.NewMockRoomRepository(t)
	participants := mocksrepo.NewMockParticipantRepository(t)
	presence := mocksrepo.NewMockPresenceRepository(t)

	users.EXPECT().GetByID("user-1").Return(nil, nil)
	rooms.EXPECT().CreateIfNotExists("room-1").Return(nil)
	participants.EXPECT().JoinRoom("room-1", "user-1").Return(nil)
	presence.EXPECT().AddParticipant(ctx, "room-1", "user-1").Return(nil)

	svc := newRoomService(t, users, rooms, participants, presence,
		mocksrepo.NewMockChatPublisher(t),
		mocksrepo.NewMockChatSubscriber(t),
	)

	require.NoError(t, svc.JoinRoom(ctx, "room-1", "user-1"))
}

// TestRoomService_JoinRoom_UserError проверяет что ошибка создания пользователя
// останавливает выполнение и возвращается наверх.
func TestRoomService_JoinRoom_UserError(t *testing.T) {
	ctx := context.Background()
	users := mocksrepo.NewMockUserRepository(t)
	rooms := mocksrepo.NewMockRoomRepository(t)
	participants := mocksrepo.NewMockParticipantRepository(t)
	presence := mocksrepo.NewMockPresenceRepository(t)

	dbErr := errors.New("db error")
	users.EXPECT().GetByID("user-1").Return(nil, dbErr)
	// rooms, participants, presence не должны вызываться

	svc := newRoomService(t, users, rooms, participants, presence,
		mocksrepo.NewMockChatPublisher(t),
		mocksrepo.NewMockChatSubscriber(t),
	)

	err := svc.JoinRoom(ctx, "room-1", "user-1")
	require.ErrorIs(t, err, dbErr)
}

// TestRoomService_JoinRoom_PresenceError проверяет что ошибка Redis
// возвращается наверх после успешной записи в Postgres.
func TestRoomService_JoinRoom_PresenceError(t *testing.T) {
	ctx := context.Background()
	users := mocksrepo.NewMockUserRepository(t)
	rooms := mocksrepo.NewMockRoomRepository(t)
	participants := mocksrepo.NewMockParticipantRepository(t)
	presence := mocksrepo.NewMockPresenceRepository(t)

	redisErr := errors.New("redis error")
	users.EXPECT().GetByID("user-1").Return(nil, nil)
	rooms.EXPECT().CreateIfNotExists("room-1").Return(nil)
	participants.EXPECT().JoinRoom("room-1", "user-1").Return(nil)
	presence.EXPECT().AddParticipant(ctx, "room-1", "user-1").Return(redisErr)

	svc := newRoomService(t, users, rooms, participants, presence,
		mocksrepo.NewMockChatPublisher(t),
		mocksrepo.NewMockChatSubscriber(t),
	)

	err := svc.JoinRoom(ctx, "room-1", "user-1")
	require.ErrorIs(t, err, redisErr)
}

// --- LeaveRoom ---

// TestRoomService_LeaveRoom_LastUser проверяет что когда последний участник
// выходит — комната помечается неактивной.
func TestRoomService_LeaveRoom_LastUser(t *testing.T) {
	ctx := context.Background()
	users := mocksrepo.NewMockUserRepository(t)
	rooms := mocksrepo.NewMockRoomRepository(t)
	participants := mocksrepo.NewMockParticipantRepository(t)
	presence := mocksrepo.NewMockPresenceRepository(t)

	participants.EXPECT().LeaveRoom("room-1", "user-1").Return(nil)
	presence.EXPECT().RemoveParticipant(ctx, "room-1", "user-1").Return(nil)
	participants.EXPECT().HasActiveParticipants("room-1").Return(false, nil)
	rooms.EXPECT().SetInactive("room-1").Return(nil)

	svc := newRoomService(t, users, rooms, participants, presence,
		mocksrepo.NewMockChatPublisher(t),
		mocksrepo.NewMockChatSubscriber(t),
	)

	require.NoError(t, svc.LeaveRoom(ctx, "room-1", "user-1"))
}

// TestRoomService_LeaveRoom_NotLastUser проверяет что когда в комнате
// ещё есть участники — комната не помечается неактивной.
func TestRoomService_LeaveRoom_NotLastUser(t *testing.T) {
	ctx := context.Background()
	users := mocksrepo.NewMockUserRepository(t)
	rooms := mocksrepo.NewMockRoomRepository(t)
	participants := mocksrepo.NewMockParticipantRepository(t)
	presence := mocksrepo.NewMockPresenceRepository(t)

	participants.EXPECT().LeaveRoom("room-1", "user-1").Return(nil)
	presence.EXPECT().RemoveParticipant(ctx, "room-1", "user-1").Return(nil)
	participants.EXPECT().HasActiveParticipants("room-1").Return(true, nil)
	// rooms.SetInactive не должен вызываться

	svc := newRoomService(t, users, rooms, participants, presence,
		mocksrepo.NewMockChatPublisher(t),
		mocksrepo.NewMockChatSubscriber(t),
	)

	require.NoError(t, svc.LeaveRoom(ctx, "room-1", "user-1"))
}

// --- IsParticipant ---

func TestRoomService_IsParticipant_True(t *testing.T) {
	ctx := context.Background()
	users := mocksrepo.NewMockUserRepository(t)
	rooms := mocksrepo.NewMockRoomRepository(t)
	participants := mocksrepo.NewMockParticipantRepository(t)
	presence := mocksrepo.NewMockPresenceRepository(t)

	presence.EXPECT().GetActiveParticipants(ctx, "room-1").Return([]string{"user-1", "user-2"}, nil)

	svc := newRoomService(t, users, rooms, participants, presence,
		mocksrepo.NewMockChatPublisher(t),
		mocksrepo.NewMockChatSubscriber(t),
	)

	ok, err := svc.IsParticipant(ctx, "room-1", "user-1")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestRoomService_IsParticipant_False(t *testing.T) {
	ctx := context.Background()
	users := mocksrepo.NewMockUserRepository(t)
	rooms := mocksrepo.NewMockRoomRepository(t)
	participants := mocksrepo.NewMockParticipantRepository(t)
	presence := mocksrepo.NewMockPresenceRepository(t)

	presence.EXPECT().GetActiveParticipants(ctx, "room-1").Return([]string{"user-2"}, nil)

	svc := newRoomService(t, users, rooms, participants, presence,
		mocksrepo.NewMockChatPublisher(t),
		mocksrepo.NewMockChatSubscriber(t),
	)

	ok, err := svc.IsParticipant(ctx, "room-1", "user-1")
	require.NoError(t, err)
	require.False(t, ok)
}

// --- OnDisconnect ---

// TestRoomService_OnDisconnect проверяет что при дисконнекте пользователь
// удаляется из всех комнат и пустые комнаты помечаются неактивными.
func TestRoomService_OnDisconnect(t *testing.T) {
	ctx := context.Background()
	users := mocksrepo.NewMockUserRepository(t)
	rooms := mocksrepo.NewMockRoomRepository(t)
	participants := mocksrepo.NewMockParticipantRepository(t)
	presence := mocksrepo.NewMockPresenceRepository(t)

	presence.EXPECT().GetActiveParticipantRooms(ctx, "user-1").Return([]string{"room-1", "room-2"}, nil)

	participants.EXPECT().LeaveRoom("room-1", "user-1").Return(nil)
	presence.EXPECT().RemoveParticipant(ctx, "room-1", "user-1").Return(nil)
	participants.EXPECT().HasActiveParticipants("room-1").Return(false, nil)
	rooms.EXPECT().SetInactive("room-1").Return(nil)

	participants.EXPECT().LeaveRoom("room-2", "user-1").Return(nil)
	presence.EXPECT().RemoveParticipant(ctx, "room-2", "user-1").Return(nil)
	participants.EXPECT().HasActiveParticipants("room-2").Return(true, nil)
	// room-2 не помечается неактивной — там ещё есть участники

	svc := newRoomService(t, users, rooms, participants, presence,
		mocksrepo.NewMockChatPublisher(t),
		mocksrepo.NewMockChatSubscriber(t),
	)

	require.NoError(t, svc.OnDisconnect(ctx, "user-1"))
}
