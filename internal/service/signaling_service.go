package service

import (
	"context"
	"fmt"

	"github.com/pion/webrtc/v3"
	"github.com/stazoloto/zync/internal/domain"
	"go.uber.org/zap"
)

type signalingService struct {
	media  MediaGateway
	rooms  RoomService
	logger *zap.Logger
}

func NewSignalingService(media MediaGateway, rooms RoomService, logger *zap.Logger) SignalingService {
	return &signalingService{
		media:  media,
		rooms:  rooms,
		logger: logger,
	}
}

// Join добавляет участника в комнату и сохраняет его присутствие
func (s *signalingService) Join(ctx context.Context, roomID, peerID string, out chan<- domain.SignalMessage) error {
	if roomID == "" || peerID == "" {
		return fmt.Errorf("roomID and peerID are required")
	}

	s.logger.Info("join signaling", zap.String("peer", peerID), zap.String("room", roomID))

	if err := s.media.Join(roomID, peerID, out); err != nil {
		return err
	}

	return nil
}

func (s *signalingService) Leave(ctx context.Context, roomID, peerID string) error {
	s.logger.Info("leave signaling", zap.String("peer", peerID), zap.String("room", roomID))
	return s.media.Leave(roomID, peerID)
}

func (s *signalingService) HandleOffer(ctx context.Context, roomID, peerID string, offer webrtc.SessionDescription) error {
	s.logger.Info("handle offer", zap.String("peer", peerID), zap.String("room", roomID))

	if roomID == "" || peerID == "" {
		return fmt.Errorf("roomID and peerID are required")
	}

	ok, err := s.rooms.IsParticipant(ctx, roomID, peerID)
	if err != nil {
		return fmt.Errorf("check participant: %w", err)
	}
	if !ok {
		return fmt.Errorf("peer %s is not a participant in room %s", peerID, roomID)
	}

	if offer.Type != webrtc.SDPTypeOffer {
		return fmt.Errorf("invalid SDP type: expected offer, got %s", offer.Type)
	}

	return s.media.HandleOffer(roomID, peerID, offer)
}

func (s *signalingService) HandleAnswer(ctx context.Context, roomID, peerID string, answer webrtc.SessionDescription) error {
	s.logger.Info("handle answer", zap.String("peer", peerID), zap.String("room", roomID))

	if roomID == "" || peerID == "" {
		return fmt.Errorf("roomID and peerID are required")
	}

	ok, err := s.rooms.IsParticipant(ctx, roomID, peerID)
	if err != nil {
		return fmt.Errorf("check participant: %w", err)
	}
	if !ok {
		return fmt.Errorf("peer %s is not a participant in room %s", peerID, roomID)
	}

	if answer.Type != webrtc.SDPTypeAnswer {
		return fmt.Errorf("invalid SDP type: expected answer, got %s", answer.Type)
	}

	return s.media.HandleAnswer(roomID, peerID, answer)
}

func (s *signalingService) HandleCandidate(ctx context.Context, roomID, peerID string, candidate webrtc.ICECandidateInit) error {
	s.logger.Info("handle candidate", zap.String("peer", peerID), zap.String("room", roomID))

	if roomID == "" || peerID == "" {
		return fmt.Errorf("roomID and peerID are required")
	}

	ok, err := s.rooms.IsParticipant(ctx, roomID, peerID)
	if err != nil {
		return fmt.Errorf("check participant: %w", err)
	}
	if !ok {
		return fmt.Errorf("peer %s is not a participant in room %s", peerID, roomID)
	}

	return s.media.HandleCandidate(roomID, peerID, candidate)
}
