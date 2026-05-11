package redis

import (
	"context"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newTestRepo(t *testing.T) (*PresenceRepo, context.Context) {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() { client.Close() })
	return NewPresenceRepo(client), context.Background()
}

func cleanupKeys(t *testing.T, repo *PresenceRepo, ctx context.Context, roomID, userID string) {
	t.Helper()
	t.Cleanup(func() {
		repo.client.Del(ctx, fmt.Sprintf("room:%s:participants", roomID))
		repo.client.Del(ctx, fmt.Sprintf("user:%s:rooms", userID))
	})
}

// TestPresenceRepo_AddParticipant проверяет что после добавления
// участник появляется в комнате и комната появляется у пользователя.
func TestPresenceRepo_AddParticipant(t *testing.T) {
	repo, ctx := newTestRepo(t)
	roomID := "room-add"
	userID := "user-add"
	cleanupKeys(t, repo, ctx, roomID, userID)

	require.NoError(t, repo.AddParticipant(ctx, roomID, userID))

	participants, err := repo.GetActiveParticipants(ctx, roomID)
	require.NoError(t, err)
	require.Contains(t, participants, userID)

	rooms, err := repo.GetActiveParticipantRooms(ctx, userID)
	require.NoError(t, err)
	require.Contains(t, rooms, roomID)
}

// TestPresenceRepo_AddParticipant_Double проверяет что добавление
// одного и того же участника дважды не дублирует его в списке.
func TestPresenceRepo_AddParticipant_Double(t *testing.T) {
	repo, ctx := newTestRepo(t)
	roomID := "room-double"
	userID := "user-double"
	cleanupKeys(t, repo, ctx, roomID, userID)

	require.NoError(t, repo.AddParticipant(ctx, roomID, userID))
	require.NoError(t, repo.AddParticipant(ctx, roomID, userID))

	participants, err := repo.GetActiveParticipants(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, participants, 1)
}

// TestPresenceRepo_RemoveParticipant проверяет что после удаления
// участник исчезает из комнаты и комната исчезает у пользователя.
func TestPresenceRepo_RemoveParticipant(t *testing.T) {
	repo, ctx := newTestRepo(t)
	roomID := "room-remove"
	userID := "user-remove"
	cleanupKeys(t, repo, ctx, roomID, userID)

	require.NoError(t, repo.AddParticipant(ctx, roomID, userID))
	require.NoError(t, repo.RemoveParticipant(ctx, roomID, userID))

	participants, err := repo.GetActiveParticipants(ctx, roomID)
	require.NoError(t, err)
	require.Empty(t, participants)

	rooms, err := repo.GetActiveParticipantRooms(ctx, userID)
	require.NoError(t, err)
	require.Empty(t, rooms)
}

// TestPresenceRepo_RemoveParticipant_NotExist проверяет что удаление
// несуществующего участника не возвращает ошибку.
func TestPresenceRepo_RemoveParticipant_NotExist(t *testing.T) {
	repo, ctx := newTestRepo(t)

	err := repo.RemoveParticipant(ctx, "room-ghost", "user-ghost")
	require.NoError(t, err)
}

// TestPresenceRepo_GetActiveParticipants_Multiple проверяет что
// несколько участников корректно сохраняются и возвращаются.
func TestPresenceRepo_GetActiveParticipants_Multiple(t *testing.T) {
	repo, ctx := newTestRepo(t)
	roomID := "room-multi"
	users := []string{"user-a", "user-b", "user-c"}

	t.Cleanup(func() {
		repo.client.Del(ctx, fmt.Sprintf("room:%s:participants", roomID))
		for _, u := range users {
			repo.client.Del(ctx, fmt.Sprintf("user:%s:rooms", u))
		}
	})

	for _, u := range users {
		require.NoError(t, repo.AddParticipant(ctx, roomID, u))
	}

	participants, err := repo.GetActiveParticipants(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, participants, 3)
	for _, u := range users {
		require.Contains(t, participants, u)
	}
}

// TestPresenceRepo_GetActiveParticipants_Empty проверяет что для
// несуществующей комнаты возвращается пустой список без ошибки.
func TestPresenceRepo_GetActiveParticipants_Empty(t *testing.T) {
	repo, ctx := newTestRepo(t)

	participants, err := repo.GetActiveParticipants(ctx, "room-empty-xyz")
	require.NoError(t, err)
	require.Empty(t, participants)
}

// TestPresenceRepo_GetActiveParticipantRooms проверяет что пользователь
// корректно отслеживается в нескольких комнатах одновременно.
func TestPresenceRepo_GetActiveParticipantRooms(t *testing.T) {
	repo, ctx := newTestRepo(t)
	userID := "user-multiroom"
	rooms := []string{"room-x", "room-y", "room-z"}

	t.Cleanup(func() {
		repo.client.Del(ctx, fmt.Sprintf("user:%s:rooms", userID))
		for _, r := range rooms {
			repo.client.Del(ctx, fmt.Sprintf("room:%s:participants", r))
		}
	})

	for _, r := range rooms {
		require.NoError(t, repo.AddParticipant(ctx, r, userID))
	}

	activeRooms, err := repo.GetActiveParticipantRooms(ctx, userID)
	require.NoError(t, err)
	require.Len(t, activeRooms, 3)
	for _, r := range rooms {
		require.Contains(t, activeRooms, r)
	}
}
