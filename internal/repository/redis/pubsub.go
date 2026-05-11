package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/stazoloto/zync/internal/repository"
)

type PubSubBroker struct {
	client *redis.Client
}

type ChatSubscription struct {
	pubsub *redis.PubSub
	ch     chan []byte
}

func NewPubSubBroker(client *redis.Client) *PubSubBroker {
	return &PubSubBroker{client: client}
}

// Publish публикует сообщение в канал Redis для указанного roomID.
func (b *PubSubBroker) Publish(ctx context.Context, roomID string, payload []byte) error {
	roomKey := fmt.Sprintf("chat:room:%s", roomID)
	if err := b.client.Publish(ctx, roomKey, payload).Err(); err != nil {
		return err
	}
	return nil
}

// Subscribe подписывается на канал Redis для указанного roomID и возвращает канал для получения сообщений.
func (b *PubSubBroker) Subscribe(ctx context.Context, roomID string) (repository.ChatSubscription, error) {
	roomKey := fmt.Sprintf("chat:room:%s", roomID)
	pubsub := b.client.Subscribe(ctx, roomKey)

	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, err
	}

	ch := make(chan []byte)

	go func () {
		<-ctx.Done()
		pubsub.Close()
	}()

	go func() {
		defer close(ch)
		for msg := range pubsub.Channel() {
			ch <- []byte(msg.Payload)
		}
	}()

	return &ChatSubscription{pubsub: pubsub, ch: ch}, nil
}

func (s *ChatSubscription) Channel() <-chan []byte {
	return s.ch
}

func (s *ChatSubscription) Close() error {
	return s.pubsub.Close()
}
