package sfu

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/stazoloto/zync/internal/domain"
	"go.uber.org/zap"
)

// Join добавляет участника в комнату, создавая для него PeerConnection и настраивая обработчики событий
func (s *SFU) Join(roomID, peerID string, out chan<- domain.SignalMessage) error {
	room := s.createRoom(roomID)

	pc, err := s.api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return fmt.Errorf("create peer connection: %w", err)
	}

	peer := &Peer{
		ID:      peerID,
		PC:      pc,
		out:     out,
		senders: map[string]*webrtc.RTPSender{},
	}

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		s.logger.Debug("connection state changed",
			zap.String("peer", peerID),
			zap.String("room", roomID),
			zap.String("state", state.String()),
		)

		switch state {
		case webrtc.PeerConnectionStateConnected:
			for _, relay := range room.GetRelays() {
				s.subscribePeerToRelay(roomID, peer, relay)
			}
		case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed:
			s.removePeer(roomID, peerID)
		}
	})

	// Обработчик для новых ICE кандидатов от клиента
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		s.emitICECandidate(roomID, peer, candidate)
	})

	// Обработчик для новых треков от клиента
	pc.OnTrack(func(remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		log.Printf("SFU OnTrack room=%s from=%s kind=%s stream=%s track=%s",
			roomID,
			peerID,
			remote.Kind(),
			remote.StreamID(),
			remote.ID(),
		)

		isVideo := remote.Kind() == webrtc.RTPCodecTypeVideo

		requestKeyframe := func() {
			if !isVideo {
				return
			}

			_ = pc.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{MediaSSRC: uint32(remote.SSRC())},
			})
		}

		relay := NewRelay(remote, isVideo, requestKeyframe, peerID)
		room.AddRelay(relay)

		room.mu.RLock()
		for id, subscriber := range room.peers {
			if id == peerID {
				continue
			}
			s.subscribePeerToRelay(roomID, subscriber, relay)
		}
		room.mu.RUnlock()

		relay.Start()
	})

	room.AddPeer(peer)

	return nil
}

// Leave удаляет участника из комнаты и уведомляет оставшихся пиров.
func (s *SFU) Leave(roomID, peerID string) error {
	room := s.getRoom(roomID)
	if room == nil {
		return nil
	}

	peer := room.RemovePeer(peerID)
	if peer == nil {
		return nil
	}

	// Убираем уходящего пира из всех relay как подписчика
	for _, relay := range room.GetRelays() {
		relay.RemoveSub(peerID)
	}

	// Удаляем relays уходящего пира, чистим senders у оставшихся пиров
	// и рассылаем peer_left с track_ids — фронтенд удаляет тайл немедленно.
	removedRelays := room.RemoveRelaysBySourcePeer(peerID)

	var streamIDs []string
	for _, relay := range removedRelays {
		streamIDs = append(streamIDs, relay.streamID)
	}

	if len(streamIDs) > 0 {
		payload, err := json.Marshal(map[string][]string{"stream_ids": streamIDs})
		if err == nil {
			for _, otherPeer := range room.GetPeers() {
				s.emit(otherPeer, domain.SignalMessage{
					Type:    domain.SignalPeerLeft,
					Room:    roomID,
					From:    "sfu",
					Payload: payload,
				})
			}
		}

		// RemoveTrack + renegotiation убирает треки из SDP оставшихся пиров
		for _, relay := range removedRelays {
			for _, otherPeer := range room.GetPeers() {
				relay.RemoveSub(otherPeer.ID)

				otherPeer.sendersMu.Lock()
				sender, ok := otherPeer.senders[relay.trackID]
				delete(otherPeer.senders, relay.trackID)
				otherPeer.sendersMu.Unlock()

				if ok && otherPeer.PC != nil {
					_ = otherPeer.PC.RemoveTrack(sender)
					s.requestRenegotiation(roomID, otherPeer)
				}
			}
		}
	}

	if peer.PC != nil {
		_ = peer.PC.Close()
	}

	s.logger.Debug("peer left sfu",
		zap.String("room", roomID),
		zap.String("peer", peerID),
	)

	return nil
}

func (s *SFU) removePeer(roomID, peerID string) {
	_ = s.Leave(roomID, peerID)
}
