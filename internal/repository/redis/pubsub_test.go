package redis

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newTestBroker(t *testing.T) (*PubSubBroker, context.Context, context.CancelFunc) {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() { client.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	return NewPubSubBroker(client), ctx, cancel
}

// TestPubSubBroker_PublishSubscribe проверяет что опубликованное сообщение
// приходит подписчику.
func TestPubSubBroker_PublishSubscribe(t *testing.T) {
	broker, ctx, cancel := newTestBroker(t)
	defer cancel()

	roomID := "room-pubsub"
	payload := []byte(`{"text":"hello"}`)

	sub, err := broker.Subscribe(ctx, roomID)
	require.NoError(t, err)
	defer sub.Close()

	require.NoError(t, broker.Publish(ctx, roomID, payload))

	select {
	case msg := <-sub.Channel():
		require.Equal(t, payload, msg)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: message not received")
	}
}

// TestPubSubBroker_MultipleMessages проверяет что несколько сообщений
// приходят в правильном порядке.
func TestPubSubBroker_MultipleMessages(t *testing.T) {
	broker, ctx, cancel := newTestBroker(t)
	defer cancel()

	roomID := "room-multi-msg"
	messages := [][]byte{
		[]byte(`{"text":"first"}`),
		[]byte(`{"text":"second"}`),
		[]byte(`{"text":"third"}`),
	}

	sub, err := broker.Subscribe(ctx, roomID)
	require.NoError(t, err)
	defer sub.Close()

	for _, msg := range messages {
		require.NoError(t, broker.Publish(ctx, roomID, msg))
	}

	for _, expected := range messages {
		select {
		case msg := <-sub.Channel():
			require.Equal(t, expected, msg)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout: message not received")
		}
	}
}

// TestPubSubBroker_ContextCancel проверяет что отмена контекста
// закрывает канал подписки.
func TestPubSubBroker_ContextCancel(t *testing.T) {
	broker, ctx, cancel := newTestBroker(t)

	roomID := "room-cancel"

	sub, err := broker.Subscribe(ctx, roomID)
	require.NoError(t, err)

	cancel() // отменяем контекст

	select {
	case _, ok := <-sub.Channel():
		require.False(t, ok, "channel should be closed after context cancel")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: channel was not closed after context cancel")
	}
}

// TestPubSubBroker_NoMessageToOtherRoom проверяет что сообщение
// не приходит подписчику другой комнаты.
func TestPubSubBroker_NoMessageToOtherRoom(t *testing.T) {
	broker, ctx, cancel := newTestBroker(t)
	defer cancel()

	sub, err := broker.Subscribe(ctx, "room-A")
	require.NoError(t, err)
	defer sub.Close()

	// публикуем в другую комнату
	require.NoError(t, broker.Publish(ctx, "room-B", []byte(`{"text":"hello"}`)))

	select {
	case msg := <-sub.Channel():
		t.Fatalf("received unexpected message: %s", msg)
	case <-time.After(300 * time.Millisecond):
		// правильно — сообщение не должно прийти
	}
}
