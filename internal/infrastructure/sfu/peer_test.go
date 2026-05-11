package sfu

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestPeer(id string) *Peer {
	return &Peer{ID: id}
}

// --- beginNegotiation ---

// TestPeer_BeginNegotiation проверяет что первый вызов возвращает true и устанавливает флаг.
func TestPeer_BeginNegotiation(t *testing.T) {
	p := newTestPeer("peer-1")

	ok := p.beginNegotiation()
	require.True(t, ok)
	require.True(t, p.negotiating)
}

// TestPeer_BeginNegotiation_AlreadyNegotiating проверяет что повторный вызов
// возвращает false — переговоры уже идут.
func TestPeer_BeginNegotiation_AlreadyNegotiating(t *testing.T) {
	p := newTestPeer("peer-1")

	require.True(t, p.beginNegotiation())
	require.False(t, p.beginNegotiation())
}

// --- endNegotiation ---

func TestPeer_EndNegotiation(t *testing.T) {
	p := newTestPeer("peer-1")

	p.beginNegotiation()
	p.endNegotiation()

	require.False(t, p.negotiating)
	// после завершения можно начать снова
	require.True(t, p.beginNegotiation())
}

// --- markPending / takePending ---

func TestPeer_MarkAndTakePending(t *testing.T) {
	p := newTestPeer("peer-1")

	p.markPending()
	require.True(t, p.takePending())
}

// TestPeer_TakePending_ClearsFlag проверяет что takePending сбрасывает флаг —
// второй вызов возвращает false.
func TestPeer_TakePending_ClearsFlag(t *testing.T) {
	p := newTestPeer("peer-1")

	p.markPending()
	require.True(t, p.takePending())
	require.False(t, p.takePending())
}

func TestPeer_TakePending_WithoutMark(t *testing.T) {
	p := newTestPeer("peer-1")
	require.False(t, p.takePending())
}

// TestPeer_NegotiationConcurrent проверяет что state machine безопасна
// при одновременном доступе из нескольких горутин.
func TestPeer_NegotiationConcurrent(t *testing.T) {
	p := newTestPeer("peer-1")

	var wg sync.WaitGroup
	wins := 0
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if p.beginNegotiation() {
				mu.Lock()
				wins++
				mu.Unlock()
				p.endNegotiation()
			}
		}()
	}

	wg.Wait()
	// каждый beginNegotiation должен был завершиться endNegotiation
	// итоговое состояние — не negotiating
	require.False(t, p.negotiating)
}
