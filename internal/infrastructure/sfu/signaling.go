package sfu

import (
	"encoding/json"
	"fmt"

	"github.com/pion/webrtc/v3"
	"github.com/stazoloto/zync/internal/domain"
	"go.uber.org/zap"
)

func (s *SFU) HandleOffer(roomID, peerID string, offer webrtc.SessionDescription) error {
	room := s.getRoom(roomID)
	if room == nil {
		return fmt.Errorf("room %s not found", roomID)
	}

	peer := room.GetPeer(peerID)
	if peer == nil {
		return fmt.Errorf("peer %s not found in room %s", peerID, roomID)
	}

	if err := peer.PC.SetRemoteDescription(offer); err != nil {
		return fmt.Errorf("set remote offer: %w", err)
	}

	peer.iceMu.Lock()
	peer.remoteDescSet = true

	for _, candidate := range peer.pendingCandidates {
		if err := peer.PC.AddICECandidate(candidate); err != nil {
			s.logger.Error("add pending ICE candidate error",
				zap.String("peer", peerID),
				zap.String("room", roomID),
				zap.Error(err),
			)
		}
	}

	peer.pendingCandidates = nil
	peer.iceMu.Unlock()

	answer, err := peer.PC.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("create answer: %w", err)
	}

	if err := peer.PC.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("set local answer: %w", err)
	}

	payload, err := json.Marshal(answer)
	if err != nil {
		return fmt.Errorf("marshal answer: %w", err)
	}

	s.emit(peer, domain.SignalMessage{
		Type:    domain.SignalAnswer,
		Room:    roomID,
		From:    "sfu",
		To:      peer.ID,
		Payload: payload,
	})

	return nil
}

func (s *SFU) HandleAnswer(roomID, peerID string, answer webrtc.SessionDescription) error {
	room := s.getRoom(roomID)
	if room == nil {
		return fmt.Errorf("room %s not found", roomID)
	}

	peer := room.GetPeer(peerID)
	if peer == nil {
		return fmt.Errorf("peer %s not found in room %s", peerID, roomID)
	}

	if err := peer.PC.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("set remote answer: %w", err)
	}

	peer.iceMu.Lock()
	peer.remoteDescSet = true

	// Добавляем все отложенные кандидаты, которые пришли до установки удаленного описания
	for _, candidate := range peer.pendingCandidates {
		if err := peer.PC.AddICECandidate(candidate); err != nil {
			s.logger.Error("add pending ICE candidate error",
				zap.String("peer", peerID),
				zap.String("room", roomID),
				zap.Error(err),
			)
		}
	}

	peer.pendingCandidates = nil
	peer.iceMu.Unlock()

	s.OnClientAnswer(roomID, peerID)
	return nil
}

func (s *SFU) HandleCandidate(roomID, peerID string, candidate webrtc.ICECandidateInit) error {
	room := s.getRoom(roomID)
	if room == nil {
		return fmt.Errorf("room %s not found", roomID)
	}

	peer := room.GetPeer(peerID)
	if peer == nil {
		return fmt.Errorf("peer %s not found in room %s", peerID, roomID)
	}

	peer.iceMu.Lock()
	defer peer.iceMu.Unlock()

	if !peer.remoteDescSet {
		peer.pendingCandidates = append(peer.pendingCandidates, candidate)
		return nil
	}

	if err := peer.PC.AddICECandidate(candidate); err != nil {
		return fmt.Errorf("add ice candidate: %w", err)
	}

	return nil
}

func (s *SFU) emitICECandidate(roomID string, peer *Peer, candidate *webrtc.ICECandidate) {
	payload, err := json.Marshal(candidate.ToJSON())
	if err != nil {
		s.logger.Error("marshal ICE candidate error", zap.Error(err))
		return
	}

	s.emit(peer, domain.SignalMessage{
		Type:    domain.SignalCandidate,
		Room:    roomID,
		From:    "sfu",
		To:      peer.ID,
		Payload: payload,
	})
}
