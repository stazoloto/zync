package sfu

import (
	"sync"

	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

// SFU управляет медиа-комнатами и хранит общий WebRTC API.
type SFU struct {
	mu     sync.RWMutex
	rooms  map[string]*Room
	api    *webrtc.API
	logger *zap.Logger
}

// NewSFU создает SFU и регистрирует стандартные WebRTC-кодеки.
func NewSFU(logger *zap.Logger) *SFU {
	var mediaEngine webrtc.MediaEngine
	_ = mediaEngine.RegisterDefaultCodecs()

	api := webrtc.NewAPI(webrtc.WithMediaEngine(&mediaEngine))

	return &SFU{
		rooms:  make(map[string]*Room),
		api:    api,
		logger: logger,
	}
}

// createRoom создает новую медиа-комнату.
// Если комната уже существует, возвращает существующую.
func (s *SFU) createRoom(roomID string) *Room {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room, ok := s.rooms[roomID]; ok {
		return room
	}

	room := &Room{
		peers:  make(map[string]*Peer),
		relays: []*Relay{},
	}

	s.rooms[roomID] = room
	return room
}

func (s *SFU) getRoom(roomID string) *Room {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.rooms[roomID]
}
