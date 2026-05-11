package sfu

import (
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/stazoloto/zync/internal/domain"
)

type Peer struct {
	ID string
	PC *webrtc.PeerConnection

	out chan<- domain.SignalMessage

	negMu       sync.Mutex
	negotiating bool
	pending     bool

	iceMu             sync.Mutex
	pendingCandidates []webrtc.ICECandidateInit
	remoteDescSet     bool

	sendersMu sync.Mutex
	senders   map[string]*webrtc.RTPSender // relay.trackID → sender
}

func (p *Peer) beginNegotiation() bool {
	p.negMu.Lock()
	defer p.negMu.Unlock()
	if p.negotiating {
		return false
	}
	p.negotiating = true
	return true
}

func (p *Peer) endNegotiation() {
	p.negMu.Lock()
	p.negotiating = false
	p.negMu.Unlock()
}

func (p *Peer) markPending() {
	p.negMu.Lock()
	p.pending = true
	p.negMu.Unlock()
}

func (p *Peer) takePending() bool {
	p.negMu.Lock()
	defer p.negMu.Unlock()

	if p.pending {
		p.pending = false
		return true
	}

	return false
}
