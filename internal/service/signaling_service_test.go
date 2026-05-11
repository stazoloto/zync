package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/stretchr/testify/mock"

	"github.com/stazoloto/zync/internal/domain"
	mocksservice "github.com/stazoloto/zync/internal/mocks/service"
	"github.com/stazoloto/zync/internal/service"
)

func newSignalingService(t *testing.T, media *mocksservice.MockMediaGateway, rooms *mocksservice.MockRoomService) service.SignalingService {
	t.Helper()
	return service.NewSignalingService(media, rooms, zap.NewNop())
}

// --- Join ---

// TestSignalingService_Join_EmptyRoomID проверяет что пустой roomID отклоняется.
func TestSignalingService_Join_EmptyRoomID(t *testing.T) {
	svc := newSignalingService(t,
		mocksservice.NewMockMediaGateway(t),
		mocksservice.NewMockRoomService(t),
	)

	err := svc.Join(context.Background(), "", "peer-1", nil)
	require.Error(t, err)
}

// TestSignalingService_Join_EmptyPeerID проверяет что пустой peerID отклоняется.
func TestSignalingService_Join_EmptyPeerID(t *testing.T) {
	svc := newSignalingService(t,
		mocksservice.NewMockMediaGateway(t),
		mocksservice.NewMockRoomService(t),
	)

	err := svc.Join(context.Background(), "room-1", "", nil)
	require.Error(t, err)
}

// TestSignalingService_Join_Success проверяет что при валидных данных
// вызывается media.Join.
func TestSignalingService_Join_Success(t *testing.T) {
	ctx := context.Background()
	media := mocksservice.NewMockMediaGateway(t)
	rooms := mocksservice.NewMockRoomService(t)

	out := make(chan domain.SignalMessage, 1)
	media.EXPECT().Join("room-1", "peer-1", mock.Anything).Return(nil)

	svc := newSignalingService(t, media, rooms)
	err := svc.Join(ctx, "room-1", "peer-1", out)
	require.NoError(t, err)
}

// TestSignalingService_Join_MediaError проверяет что ошибка media.Join
// возвращается и peer не считается joined.
func TestSignalingService_Join_MediaError(t *testing.T) {
	ctx := context.Background()
	media := mocksservice.NewMockMediaGateway(t)
	rooms := mocksservice.NewMockRoomService(t)

	out := make(chan domain.SignalMessage, 1)
	mediaErr := errors.New("media error")
	media.EXPECT().Join("room-1", "peer-1", mock.Anything).Return(mediaErr)

	svc := newSignalingService(t, media, rooms)
	err := svc.Join(ctx, "room-1", "peer-1", out)
	require.ErrorIs(t, err, mediaErr)
}

// --- HandleOffer ---

// TestSignalingService_HandleOffer_NotJoined проверяет что peer который
// не делал join не может слать offer.
func TestSignalingService_HandleOffer_NotJoined(t *testing.T) {
	ctx := context.Background()
	media := mocksservice.NewMockMediaGateway(t)
	rooms := mocksservice.NewMockRoomService(t)

	rooms.EXPECT().IsParticipant(ctx, "room-1", "peer-1").Return(false, nil)

	svc := newSignalingService(t, media, rooms)
	err := svc.HandleOffer(ctx, "room-1", "peer-1", webrtc.SessionDescription{Type: webrtc.SDPTypeOffer})
	require.Error(t, err)
}

// TestSignalingService_HandleOffer_WrongSDPType проверяет что answer вместо offer отклоняется.
func TestSignalingService_HandleOffer_WrongSDPType(t *testing.T) {
	ctx := context.Background()
	media := mocksservice.NewMockMediaGateway(t)
	rooms := mocksservice.NewMockRoomService(t)

	rooms.EXPECT().IsParticipant(ctx, "room-1", "peer-1").Return(true, nil)

	svc := newSignalingService(t, media, rooms)
	err := svc.HandleOffer(ctx, "room-1", "peer-1", webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer})
	require.Error(t, err)
}

// TestSignalingService_HandleOffer_Success проверяет успешный путь.
func TestSignalingService_HandleOffer_Success(t *testing.T) {
	ctx := context.Background()
	media := mocksservice.NewMockMediaGateway(t)
	rooms := mocksservice.NewMockRoomService(t)

	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer}
	rooms.EXPECT().IsParticipant(ctx, "room-1", "peer-1").Return(true, nil)
	media.EXPECT().HandleOffer("room-1", "peer-1", offer).Return(nil)

	svc := newSignalingService(t, media, rooms)
	err := svc.HandleOffer(ctx, "room-1", "peer-1", offer)
	require.NoError(t, err)
}

// --- HandleAnswer ---

// TestSignalingService_HandleAnswer_WrongSDPType проверяет что offer вместо answer отклоняется.
func TestSignalingService_HandleAnswer_WrongSDPType(t *testing.T) {
	ctx := context.Background()
	media := mocksservice.NewMockMediaGateway(t)
	rooms := mocksservice.NewMockRoomService(t)

	rooms.EXPECT().IsParticipant(ctx, "room-1", "peer-1").Return(true, nil)

	svc := newSignalingService(t, media, rooms)
	err := svc.HandleAnswer(ctx, "room-1", "peer-1", webrtc.SessionDescription{Type: webrtc.SDPTypeOffer})
	require.Error(t, err)
}

// TestSignalingService_HandleAnswer_Success проверяет успешный путь.
func TestSignalingService_HandleAnswer_Success(t *testing.T) {
	ctx := context.Background()
	media := mocksservice.NewMockMediaGateway(t)
	rooms := mocksservice.NewMockRoomService(t)

	answer := webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer}
	rooms.EXPECT().IsParticipant(ctx, "room-1", "peer-1").Return(true, nil)
	media.EXPECT().HandleAnswer("room-1", "peer-1", answer).Return(nil)

	svc := newSignalingService(t, media, rooms)
	err := svc.HandleAnswer(ctx, "room-1", "peer-1", answer)
	require.NoError(t, err)
}

// --- HandleCandidate ---

// TestSignalingService_HandleCandidate_NotJoined проверяет что peer
// который не делал join не может слать candidates.
func TestSignalingService_HandleCandidate_NotJoined(t *testing.T) {
	ctx := context.Background()
	media := mocksservice.NewMockMediaGateway(t)
	rooms := mocksservice.NewMockRoomService(t)

	rooms.EXPECT().IsParticipant(ctx, "room-1", "peer-1").Return(false, nil)

	svc := newSignalingService(t, media, rooms)
	err := svc.HandleCandidate(ctx, "room-1", "peer-1", webrtc.ICECandidateInit{})
	require.Error(t, err)
}

// TestSignalingService_HandleCandidate_Success проверяет успешный путь.
func TestSignalingService_HandleCandidate_Success(t *testing.T) {
	ctx := context.Background()
	media := mocksservice.NewMockMediaGateway(t)
	rooms := mocksservice.NewMockRoomService(t)

	candidate := webrtc.ICECandidateInit{Candidate: "candidate:123"}
	rooms.EXPECT().IsParticipant(ctx, "room-1", "peer-1").Return(true, nil)
	media.EXPECT().HandleCandidate("room-1", "peer-1", candidate).Return(nil)

	svc := newSignalingService(t, media, rooms)
	err := svc.HandleCandidate(ctx, "room-1", "peer-1", candidate)
	require.NoError(t, err)
}
