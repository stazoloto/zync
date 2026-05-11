package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/stazoloto/zync/internal/domain"
	"github.com/stazoloto/zync/internal/handlers"
	mocksrepo "github.com/stazoloto/zync/internal/mocks/repository"
	mocksservice "github.com/stazoloto/zync/internal/mocks/service"
)

// newTestServer создаёт реальный HTTP сервер на случайном порту.
// httptest.NewServer — стандартный инструмент Go для тестирования HTTP хендлеров.
func newTestServer(t *testing.T, rooms *mocksservice.MockRoomService, signaling *mocksservice.MockSignalingService) *httptest.Server {
	t.Helper()
	h := handlers.NewWebSocketHandler(rooms, signaling, zap.NewNop())
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

// wsURL конвертирует http:// адрес в ws:// с параметрами.
func wsURL(srv *httptest.Server, clientID, roomID string) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") +
		"?client_id=" + clientID + "&room_id=" + roomID
}

// wsConnect подключается к WebSocket серверу.
func wsConnect(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

// sendMessage отправляет SignalMessage через WebSocket.
func sendMessage(t *testing.T, conn *websocket.Conn, msg domain.SignalMessage) {
	t.Helper()
	data, err := json.Marshal(msg)
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, data))
}

// --- Валидация параметров ---

// TestWebSocketHandler_MissingRoomID проверяет что сервер закрывает соединение
// если не передан room_id.
// WebSocket сначала устанавливает соединение (HTTP 101), потом сервер закрывает —
// поэтому Dial успешен, но следующий Read возвращает ошибку.
func TestWebSocketHandler_MissingRoomID(t *testing.T) {
	rooms := mocksservice.NewMockRoomService(t)
	signaling := mocksservice.NewMockSignalingService(t)
	srv := newTestServer(t, rooms, signaling)

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "?client_id=peer-1"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	// сервер закрыл соединение — Read должен вернуть ошибку
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, _, err = conn.ReadMessage()
	require.Error(t, err)
}

// TestWebSocketHandler_MissingPeerID проверяет то же самое для client_id.
func TestWebSocketHandler_MissingPeerID(t *testing.T) {
	rooms := mocksservice.NewMockRoomService(t)
	signaling := mocksservice.NewMockSignalingService(t)
	srv := newTestServer(t, rooms, signaling)

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "?room_id=room-1"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, _, err = conn.ReadMessage()
	require.Error(t, err)
}

// --- Join ---

// TestWebSocketHandler_Join проверяет что сообщение "join" вызывает
// JoinRoom, signaling.Join и SubscribeToRoom в правильном порядке.
// mock.Anything используется для аргументов которые мы не можем предсказать:
// контекст создаётся внутри хендлера, канал out тоже.
func TestWebSocketHandler_Join(t *testing.T) {
	rooms := mocksservice.NewMockRoomService(t)
	signaling := mocksservice.NewMockSignalingService(t)

	// ChatSubscription — мок подписки на чат.
	// Возвращаем уже закрытый канал — subscribeChat горутина сразу выйдет
	// из range и вызовет defer sub.Close(). Если вернуть незакрытый канал,
	// горутина заблокируется навсегда и Close() никогда не вызовется.
	ch := make(chan []byte)
	close(ch)
	chatSub := mocksrepo.NewMockChatSubscription(t)
	chatSub.EXPECT().Channel().Return(ch)
	chatSub.EXPECT().Close().Return(nil)

	rooms.EXPECT().JoinRoom(mock.Anything, "room-1", "peer-1").Return(nil)
	signaling.EXPECT().Join(mock.Anything, "room-1", "peer-1", mock.Anything).Return(nil)
	rooms.EXPECT().SubscribeToRoom(mock.Anything, "room-1").Return(chatSub, nil)
	signaling.EXPECT().Leave(mock.Anything, "room-1", "peer-1").Return(nil)

	srv := newTestServer(t, rooms, signaling)
	conn := wsConnect(t, wsURL(srv, "peer-1", "room-1"))

	sendMessage(t, conn, domain.SignalMessage{
		Type: domain.SignalJoin,
		Room: "room-1",
		From: "peer-1",
	})

	// даём время обработать join
	time.Sleep(100 * time.Millisecond)

	// явно закрываем соединение — это триггерит defer в ServeHTTP:
	// cancel() → ctx.Done() → subscribeChat горутина вызывает sub.Close()
	// без этого t.Cleanup закрыл бы соединение ПОСЛЕ проверки моков (LIFO порядок)
	conn.Close()
	time.Sleep(100 * time.Millisecond)
}

// --- Chat ---

// TestWebSocketHandler_Chat проверяет что сообщение "chat"
// вызывает PublishMessage.
// mock.Anything для payload — мы не знаем точный байтовый формат.
func TestWebSocketHandler_Chat(t *testing.T) {
	rooms := mocksservice.NewMockRoomService(t)
	signaling := mocksservice.NewMockSignalingService(t)

	rooms.EXPECT().PublishMessage(mock.Anything, "room-1", mock.Anything).Return(nil)
	signaling.EXPECT().Leave(mock.Anything, "room-1", "peer-1").Return(nil)

	srv := newTestServer(t, rooms, signaling)
	conn := wsConnect(t, wsURL(srv, "peer-1", "room-1"))

	chatPayload, _ := json.Marshal(domain.ChatMessage{Text: "hello"})
	sendMessage(t, conn, domain.SignalMessage{
		Type:    domain.SignalChat,
		Room:    "room-1",
		From:    "peer-1",
		Payload: chatPayload,
	})

	time.Sleep(100 * time.Millisecond)

	// закрываем явно чтобы defer в ServeHTTP успел вызвать Leave
	// до того как мок проверит ожидания через t.Cleanup
	conn.Close()
	time.Sleep(100 * time.Millisecond)
}

// --- Leave ---

// TestWebSocketHandler_Leave проверяет что при закрытии соединения
// вызывается signaling.Leave — это критично для cleanup WebRTC peer connection.
// Leave вызывается в defer внутри ServeHTTP.
func TestWebSocketHandler_Leave(t *testing.T) {
	rooms := mocksservice.NewMockRoomService(t)
	signaling := mocksservice.NewMockSignalingService(t)

	signaling.EXPECT().Leave(mock.Anything, "room-1", "peer-1").Return(nil)

	srv := newTestServer(t, rooms, signaling)
	conn := wsConnect(t, wsURL(srv, "peer-1", "room-1"))

	// закрываем соединение явно — это триггерит defer в ServeHTTP
	conn.Close()

	// даём defer время выполниться
	time.Sleep(100 * time.Millisecond)
}
