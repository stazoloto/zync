package sfu

func (s *SFU) subscribePeerToRelay(roomID string, peer *Peer, relay *Relay) {
	local, err := relay.AddSub(peer.ID)
	if err != nil {
		return
	}

	sender, err := peer.PC.AddTrack(local)
	if err != nil {
		relay.RemoveSub(peer.ID)
		return
	}

	peer.sendersMu.Lock()
	peer.senders[relay.trackID] = sender
	peer.sendersMu.Unlock()

	go func() {
		buf := make([]byte, 1500)
		for {
			if _, _, err := sender.Read(buf); err != nil {
				return
			}
		}
	}()

	s.requestRenegotiation(roomID, peer)
}
