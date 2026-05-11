package sfu

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestRoom() *Room {
	return &Room{
		peers:  map[string]*Peer{},
		relays: []*Relay{},
	}
}

// --- AddPeer / GetPeer / RemovePeer ---

func TestRoom_AddAndGetPeer(t *testing.T) {
	r := newTestRoom()
	peer := newTestPeer("peer-1")

	r.AddPeer(peer)

	got := r.GetPeer("peer-1")
	require.Equal(t, peer, got)
}

func TestRoom_GetPeer_NotExist(t *testing.T) {
	r := newTestRoom()

	got := r.GetPeer("ghost")
	require.Nil(t, got)
}

func TestRoom_RemovePeer(t *testing.T) {
	r := newTestRoom()
	peer := newTestPeer("peer-1")

	r.AddPeer(peer)
	removed := r.RemovePeer("peer-1")

	require.Equal(t, peer, removed)
	require.Nil(t, r.GetPeer("peer-1"))
}

// TestRoom_RemovePeer_NotExist проверяет что удаление несуществующего пира
// возвращает nil без паники.
func TestRoom_RemovePeer_NotExist(t *testing.T) {
	r := newTestRoom()

	removed := r.RemovePeer("ghost")
	require.Nil(t, removed)
}

func TestRoom_AddPeer_Multiple(t *testing.T) {
	r := newTestRoom()

	r.AddPeer(newTestPeer("peer-1"))
	r.AddPeer(newTestPeer("peer-2"))
	r.AddPeer(newTestPeer("peer-3"))

	require.NotNil(t, r.GetPeer("peer-1"))
	require.NotNil(t, r.GetPeer("peer-2"))
	require.NotNil(t, r.GetPeer("peer-3"))
}

// --- AddRelay / GetRelays ---

func TestRoom_AddAndGetRelays(t *testing.T) {
	r := newTestRoom()
	relay := newTestRelay(false)

	r.AddRelay(relay)

	relays := r.GetRelays()
	require.Len(t, relays, 1)
	require.Equal(t, relay, relays[0])
}

// TestRoom_GetRelays_ReturnsCopy проверяет что GetRelays возвращает копию —
// изменение результата не затрагивает внутренний список.
func TestRoom_GetRelays_ReturnsCopy(t *testing.T) {
	r := newTestRoom()
	r.AddRelay(newTestRelay(false))

	relays := r.GetRelays()
	relays[0] = nil // меняем копию

	original := r.GetRelays()
	require.NotNil(t, original[0])
}

func TestRoom_GetRelays_Empty(t *testing.T) {
	r := newTestRoom()

	relays := r.GetRelays()
	require.Empty(t, relays)
}
