package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type PresenceRepo struct {
	client *redis.Client
}

func NewPresenceRepo(client *redis.Client) *PresenceRepo {
	return &PresenceRepo{
		client: client,
	}
}

// AddParticipant отмечает участника как активного в комнате, добавляя его ID в множество Redis, связанное с ID комнаты.
func (r *PresenceRepo) AddParticipant(ctx context.Context, roomID, userID string) error {
	roomKey := fmt.Sprintf("room:%s:participants", roomID)
	if err := r.client.SAdd(ctx, roomKey, userID).Err(); err != nil {
		return err
	}

	userKey := fmt.Sprintf("user:%s:rooms", userID)
	if err := r.client.SAdd(ctx, userKey, roomID).Err(); err != nil {
		return err
	}

	return nil
}

// RemoveParticipant удаляет участника из множества активных пользователей в комнате.
func (r *PresenceRepo) RemoveParticipant(ctx context.Context, roomID, userID string) error {
	roomKey := fmt.Sprintf("room:%s:participants", roomID)
	if err := r.client.SRem(ctx, roomKey, userID).Err(); err != nil {
		return err
	}

	userKey := fmt.Sprintf("user:%s:rooms", userID)
	if err := r.client.SRem(ctx, userKey, roomID).Err(); err != nil {
		return err
	}

	return nil
}

// GetActiveParticipants возвращает список ID активных участников в комнате, извлекая их из множества Redis, связанного с ID комнаты.
func (r *PresenceRepo) GetActiveParticipants(ctx context.Context, roomID string) ([]string, error) {
	roomKey := fmt.Sprintf("room:%s:participants", roomID)
	return r.client.SMembers(ctx, roomKey).Result()
}

// GetActiveParticipantRooms возвращает список ID комнат, в которых пользователь является активным участником, извлекая их из множества Redis, связанного с ID пользователя.
func (r *PresenceRepo) GetActiveParticipantRooms(ctx context.Context, userID string) ([]string, error) {
	userKey := fmt.Sprintf("user:%s:rooms", userID)
	return r.client.SMembers(ctx, userKey).Result()
}
