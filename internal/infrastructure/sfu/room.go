package sfu

import "sync"

type Room struct {
	mu     sync.RWMutex
	peers  map[string]*Peer
	relays []*Relay
}

func (r *Room) AddPeer(peer *Peer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.peers[peer.ID] = peer
}

func (r *Room) RemovePeer(peerID string) *Peer {
	r.mu.Lock()
	defer r.mu.Unlock()

	p := r.peers[peerID]
	delete(r.peers, peerID)
	return p
}

func (r *Room) GetPeer(peerID string) *Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.peers[peerID]
}

func (r *Room) GetPeers() []*Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	peers := make([]*Peer, 0, len(r.peers))
	for _, p := range r.peers {
		peers = append(peers, p)
	}
	return peers
}

func (r *Room) AddRelay(relay *Relay) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.relays = append(r.relays, relay)
}

func (r *Room) GetRelays() []*Relay {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]*Relay(nil), r.relays...)
}

// RemoveRelaysBySourcePeer удаляет все relays источника peerID и возвращает их.
func (r *Room) RemoveRelaysBySourcePeer(peerID string) []*Relay {
	r.mu.Lock()
	defer r.mu.Unlock()

	var removed []*Relay
	kept := r.relays[:0]
	for _, relay := range r.relays {
		if relay.SourcePeerID == peerID {
			removed = append(removed, relay)
		} else {
			kept = append(kept, relay)
		}
	}
	r.relays = kept
	return removed
}
