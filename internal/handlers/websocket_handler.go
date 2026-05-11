package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/stazoloto/zync/internal/domain"
	"github.com/stazoloto/zync/internal/service"
	"go.uber.org/zap"
)

type WebSocketHandler struct {
	rooms     service.RoomService
	signaling service.SignalingService
	logger    *zap.Logger
	upgrader  websocket.Upgrader
}

func NewWebSocketHandler(
	rooms service.RoomService,
	signaling service.SignalingService,
	logger *zap.Logger,
) *WebSocketHandler {
	return &WebSocketHandler{
		rooms:     rooms,
		signaling: signaling,
		logger:    logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *WebSocketHandler) ServeGin(c *gin.Context) {
	h.ServeHTTP(c.Writer, c.Request)
}

// ServeHTTP обслуживает WebSocket-подключение участника комнаты.
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Warn("websocket upgrade failed", zap.Error(err))
		return
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(r.Context())

	peerID := r.URL.Query().Get("client_id")
	roomID := r.URL.Query().Get("room_id")
	if roomID == "" {
		h.logger.Warn("empty room id", zap.String("peer", peerID))
		cancel()
		return
	}
	if peerID == "" {
		h.logger.Warn("empty peer id")
		cancel()
		return
	}

	out := make(chan domain.SignalMessage, 32)

	go h.writeLoop(ws, out)

	defer func() {
		cancel()
		close(out)

		if err := h.signaling.Leave(context.Background(), roomID, peerID); err != nil {
			h.logger.Error("signaling leave failed",
				zap.String("room", roomID),
				zap.String("peer", peerID),
				zap.Error(err),
			)
		}
	}()

	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			h.logger.Info("websocket closed",
				zap.String("room", roomID),
				zap.String("peer", peerID),
				zap.Error(err),
			)
			return
		}

		var msg domain.SignalMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			h.logger.Warn("bad websocket message",
				zap.String("room", roomID),
				zap.String("peer", peerID),
				zap.Error(err),
			)
			continue
		}

		if msg.Room != "" {
			roomID = msg.Room
		}

		if msg.From == "" {
			msg.From = peerID
		}

		if err := h.handleSignal(ctx, roomID, peerID, out, msg); err != nil {
			h.logger.Error("handle signal failed",
				zap.String("room", roomID),
				zap.String("peer", peerID),
				zap.String("type", string(msg.Type)),
				zap.Error(err),
			)
		}
	}
}

// handleChat публикует сообщение участника в чат комнаты.
func (h *WebSocketHandler) handleChat(
	ctx context.Context,
	roomID string,
	peerID string,
	msg domain.SignalMessage,
) error {
	var chat domain.ChatMessage
	if err := json.Unmarshal(msg.Payload, &chat); err != nil {
		return err
	}

	chat.Room = roomID
	chat.From = peerID
	chat.CreatedAt = time.Now().UTC()

	payload, err := json.Marshal(chat)
	if err != nil {
		return err
	}

	outgoing := domain.SignalMessage{
		Type:    domain.SignalChat,
		Room:    roomID,
		From:    peerID,
		Payload: payload,
	}

	data, err := json.Marshal(outgoing)
	if err != nil {
		return err
	}

	return h.rooms.PublishMessage(ctx, roomID, data)
}

// subscribeChat подписывает WebSocket-подключение на сообщения чата комнаты.
func (h *WebSocketHandler) subscribeChat(
	ctx context.Context,
	roomID string,
	out chan<- domain.SignalMessage,
) error {
	sub, err := h.rooms.SubscribeToRoom(ctx, roomID)
	if err != nil {
		return err
	}

	go func() {
		defer sub.Close()

		for payload := range sub.Channel() {
			var msg domain.SignalMessage
			if err := json.Unmarshal(payload, &msg); err != nil {
				h.logger.Warn("bad chat message", zap.Error(err))
				continue
			}

			select {
			case out <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// writeLoop отправляет исходящие signaling-сообщения в WebSocket.
func (h *WebSocketHandler) writeLoop(ws *websocket.Conn, out <-chan domain.SignalMessage) {
	for msg := range out {
		data, err := json.Marshal(msg)
		if err != nil {
			h.logger.Warn("marshal signaling message failed", zap.Error(err))
			continue
		}

		if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
			h.logger.Info("websocket write failed", zap.Error(err))
			return
		}
	}
}

// handleSignal разбирает входящее signaling-сообщение и вызывает нужный usecase.
func (h *WebSocketHandler) handleSignal(
	ctx context.Context,
	roomID string,
	peerID string,
	out chan<- domain.SignalMessage,
	msg domain.SignalMessage,
) error {
	switch msg.Type {
	case domain.SignalJoin:
		if err := h.rooms.JoinRoom(ctx, roomID, peerID); err != nil {
			return err
		}

		if err := h.signaling.Join(ctx, roomID, peerID, out); err != nil {
			return err
		}

		return h.subscribeChat(ctx, roomID, out)

	case domain.SignalOffer:
		var offer webrtc.SessionDescription
		if err := json.Unmarshal(msg.Payload, &offer); err != nil {
			return err
		}

		return h.signaling.HandleOffer(ctx, roomID, peerID, offer)

	case domain.SignalAnswer:
		var answer webrtc.SessionDescription
		if err := json.Unmarshal(msg.Payload, &answer); err != nil {
			return err
		}

		return h.signaling.HandleAnswer(ctx, roomID, peerID, answer)

	case domain.SignalCandidate:
		var candidate webrtc.ICECandidateInit
		if err := json.Unmarshal(msg.Payload, &candidate); err != nil {
			return err
		}

		return h.signaling.HandleCandidate(ctx, roomID, peerID, candidate)

	case domain.SignalChat:
		return h.handleChat(ctx, roomID, peerID, msg)
	}

	return nil
}
