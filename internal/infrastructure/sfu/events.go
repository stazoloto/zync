package sfu

import (
	"github.com/stazoloto/zync/internal/domain"
	"go.uber.org/zap"
)

func (s *SFU) emit(peer *Peer, msg domain.SignalMessage) {
	select {
	case peer.out <- msg:
	default:
		s.logger.Warn("drop signaling message",
			zap.String("peer", peer.ID),
			zap.String("type", string(msg.Type)),
		)
	}
}
