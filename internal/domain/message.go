package domain

import "encoding/json"

type SignalType string

const (
	SignalJoin      SignalType = "join"
	SignalOffer     SignalType = "offer"
	SignalAnswer    SignalType = "answer"
	SignalCandidate SignalType = "candidate"
	SignalChat      SignalType = "chat"
	SignalPeerLeft  SignalType = "peer_left"
)

type SignalMessage struct {
	Type    SignalType      `json:"type"`
	Room    string          `json:"room"`
	From    string          `json:"from"`
	To      string          `json:"to,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}
