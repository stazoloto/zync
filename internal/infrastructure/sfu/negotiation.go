package sfu

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
	"github.com/stazoloto/zync/internal/domain"
	"go.uber.org/zap"
)

func (s *SFU) requestRenegotiation(roomID string, peer *Peer) {
	if peer.PC.SignalingState() != webrtc.SignalingStateStable {
		peer.markPending()
		return
	}

	if !peer.beginNegotiation() {
		peer.markPending()
		return
	}

	peer.iceMu.Lock()
	peer.remoteDescSet = false
	peer.iceMu.Unlock()

	offer, err := peer.PC.CreateOffer(nil)
	if err != nil {
		s.logger.Error("create offer failed", zap.String("peer", peer.ID), zap.Error(err))
		peer.endNegotiation()
		return
	}

	if err := peer.PC.SetLocalDescription(offer); err != nil {
		s.logger.Error("set local description failed", zap.String("peer", peer.ID), zap.Error(err))
		peer.endNegotiation()
		return
	}

	payload, err := json.Marshal(offer)
	if err != nil {
		s.logger.Error("marshal offer failed", zap.String("peer", peer.ID), zap.Error(err))
		peer.endNegotiation()
		return
	}

	s.emit(peer, domain.SignalMessage{
		Type:    domain.SignalOffer,
		Room:    roomID,
		From:    "sfu",
		To:      peer.ID,
		Payload: payload,
	})
}

func (s *SFU) OnClientAnswer(roomID, peerID string) {
	room := s.getRoom(roomID)
	if room == nil {
		return
	}

	peer := room.GetPeer(peerID)
	if peer == nil {
		return
	}

	peer.endNegotiation()

	if peer.takePending() {
		s.requestRenegotiation(roomID, peer)
		return
	}

	// Все раунды renegotiation завершены — запрашиваем keyframe для всех видео-relay
	// у которых есть этот peer как подписчик. Keyframe нужен чтобы браузер смог начать
	// декодировать видео после того как SDP с SSRC уже согласован.
	for _, relay := range room.GetRelays() {
		if relay.isVideo && relay.HasSub(peerID) {
			relay.RequestKeyFrame()
		}
	}
}
