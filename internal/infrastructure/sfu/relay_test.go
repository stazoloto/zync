package sfu

import (
	"errors"
	"testing"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/require"
)

// mockTrackReader мокает TrackReader — позволяет создать Relay без реального WebRTC соединения.
type mockTrackReader struct {
	readFunc func(b []byte) (int, interceptor.Attributes, error)
}

func (m *mockTrackReader) Read(b []byte) (int, interceptor.Attributes, error) {
	return m.readFunc(b)
}

func (m *mockTrackReader) Codec() webrtc.RTPCodecParameters {
	return webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/VP8"},
	}
}

func (m *mockTrackReader) ID() string       { return "track-id" }
func (m *mockTrackReader) StreamID() string { return "stream-id" }

// mockTrackWriter мокает trackWriter — позволяет тестировать поведение при ошибках записи.
type mockTrackWriter struct {
	writeFunc func(b []byte) (int, error)
}

func (m *mockTrackWriter) Write(b []byte) (int, error) {
	return m.writeFunc(b)
}

// newTestRelay создаёт Relay с моком TrackReader.
func newTestRelay(isVideo bool) *Relay {
	reader := &mockTrackReader{
		readFunc: func(b []byte) (int, interceptor.Attributes, error) {
			// блокируется — используется только там где нужен Start
			select {}
		},
	}
	return &Relay{
		remote:   reader,
		codec:    webrtc.RTPCodecCapability{MimeType: "video/VP8"},
		trackID:  "track-id",
		streamID: "stream-id",
		subs:     map[string]trackWriter{},
		isVideo:  isVideo,
	}
}

// --- AddSub ---

func TestRelay_AddSub(t *testing.T) {
	r := newTestRelay(false)

	local, err := r.AddSub("peer-1")
	require.NoError(t, err)
	require.NotNil(t, local)
	require.Len(t, r.subs, 1)
}

// TestRelay_AddSub_Double проверяет что повторный AddSub возвращает тот же трек.
func TestRelay_AddSub_Double(t *testing.T) {
	r := newTestRelay(false)

	first, err := r.AddSub("peer-1")
	require.NoError(t, err)

	second, err := r.AddSub("peer-1")
	require.NoError(t, err)

	require.Equal(t, first, second)
	require.Len(t, r.subs, 1)
}

func TestRelay_AddSub_Multiple(t *testing.T) {
	r := newTestRelay(false)

	_, err := r.AddSub("peer-1")
	require.NoError(t, err)
	_, err = r.AddSub("peer-2")
	require.NoError(t, err)

	require.Len(t, r.subs, 2)
}

// --- RemoveSub ---

func TestRelay_RemoveSub(t *testing.T) {
	r := newTestRelay(false)

	_, err := r.AddSub("peer-1")
	require.NoError(t, err)

	r.RemoveSub("peer-1")
	require.Empty(t, r.subs)
}

// TestRelay_RemoveSub_NotExist проверяет что удаление несуществующего подписчика не паникует.
func TestRelay_RemoveSub_NotExist(t *testing.T) {
	r := newTestRelay(false)
	require.NotPanics(t, func() { r.RemoveSub("ghost") })
}

// --- Keyframe ---

// TestRelay_AddSub_RequestsKeyframe проверяет что при добавлении видео подписчика
// запрашивается keyframe.
func TestRelay_AddSub_RequestsKeyframe(t *testing.T) {
	r := newTestRelay(true)

	keyframeRequested := false
	r.requestKeyframe = func() { keyframeRequested = true }

	_, err := r.AddSub("peer-1")
	require.NoError(t, err)
	require.True(t, keyframeRequested)
}

// TestRelay_AddSub_KeyframeThrottle проверяет что повторный keyframe
// не запрашивается раньше pliThrottle.
func TestRelay_AddSub_KeyframeThrottle(t *testing.T) {
	r := newTestRelay(true)

	count := 0
	r.requestKeyframe = func() { count++ }
	r.lastPLI = time.Now() // симулируем недавний PLI

	_, err := r.AddSub("peer-1")
	require.NoError(t, err)
	require.Equal(t, 0, count) // keyframe не должен быть запрошен
}

// --- Start: ошибочный подписчик удаляется ---

// TestRelay_Start_RemovesFailedSub проверяет что подписчик который вернул ошибку
// при записи удаляется из subs.
func TestRelay_Start_RemovesFailedSub(t *testing.T) {
	done := make(chan struct{})

	r := newTestRelay(false)

	// reader отдаёт один пакет потом закрывается
	once := make(chan struct{}, 1)
	once <- struct{}{}
	r.remote = &mockTrackReader{
		readFunc: func(b []byte) (int, interceptor.Attributes, error) {
			if _, ok := <-once; ok {
				return 4, nil, nil
			}
			return 0, nil, errors.New("track closed")
		},
	}

	// writer всегда возвращает ошибку
	r.subs["peer-fail"] = &mockTrackWriter{
		writeFunc: func(b []byte) (int, error) {
			close(done)
			return 0, errors.New("write failed")
		},
	}

	r.Start()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: write was not called")
	}

	// даём горутине время удалить подписчика
	time.Sleep(10 * time.Millisecond)

	r.mu.Lock()
	defer r.mu.Unlock()
	require.Empty(t, r.subs)
}
