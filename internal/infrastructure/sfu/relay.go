package sfu

import (
	"sync"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

const (
	pliThrottle      = 800 * time.Millisecond
	rtpVideoClockHz  = 90000
	maxVideoPaceWait = 50 * time.Millisecond
)

// TrackReader абстрагирует webrtc.TrackRemote для чтения RTP пакетов.
type TrackReader interface {
	Read(b []byte) (int, interceptor.Attributes, error)
	Codec() webrtc.RTPCodecParameters
	ID() string
	StreamID() string
}

// trackWriter абстрагирует запись RTP пакетов подписчику.
type trackWriter interface {
	Write(b []byte) (n int, err error)
}

type Relay struct {
	remote TrackReader

	codec        webrtc.RTPCodecCapability
	streamID     string
	trackID      string
	SourcePeerID string

	mu   sync.RWMutex
	subs map[string]trackWriter

	lastPLI         time.Time
	requestKeyframe func()
	isVideo         bool
}

func NewRelay(remote *webrtc.TrackRemote, isVideo bool, requestKeyframe func(), sourcePeerID string) *Relay {
	return &Relay{
		remote: remote,

		codec:        remote.Codec().RTPCodecCapability,
		trackID:      remote.ID(),
		streamID:     remote.StreamID(),
		SourcePeerID: sourcePeerID,

		subs:            map[string]trackWriter{},
		requestKeyframe: requestKeyframe,
		isVideo:         isVideo,
	}
}

// AddSub добавляет подписчика и возвращает его персональный TrackLocal
func (r *Relay) AddSub(peerID string) (*webrtc.TrackLocalStaticRTP, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if t, ok := r.subs[peerID]; ok {
		if local, ok := t.(*webrtc.TrackLocalStaticRTP); ok {
			return local, nil
		}
	}

	local, err := webrtc.NewTrackLocalStaticRTP(
		r.codec,
		r.trackID,
		r.streamID,
	)
	if err != nil {
		return nil, err
	}

	r.subs[peerID] = local

	if r.isVideo && r.requestKeyframe != nil {
		if time.Since(r.lastPLI) > pliThrottle {
			r.requestKeyframe()
			r.lastPLI = time.Now()
		}
	}

	return local, nil
}

func (r *Relay) RemoveSub(peerID string) {
	r.mu.Lock()
	delete(r.subs, peerID)
	r.mu.Unlock()
}

func (r *Relay) HasSub(peerID string) bool {
	r.mu.RLock()
	_, ok := r.subs[peerID]
	r.mu.RUnlock()
	return ok
}

// Start: 1 reader -> N writers (+ очень простой pacing для video)
func (r *Relay) Start() {
	go func() {
		buf := make([]byte, 1500)

		// минимальный pacing для видео (сглаживает рывки)
		var lastTS uint32
		lastWall := time.Now()

		for {
			n, _, err := r.remote.Read(buf)
			if err != nil {
				return
			}

			if r.isVideo {
				var pkt rtp.Packet
				if err := pkt.Unmarshal(buf[:n]); err == nil {
					if lastTS != 0 {
						delta := pkt.Timestamp - lastTS
						wait := time.Duration(delta) * time.Second / rtpVideoClockHz
						elapsed := time.Since(lastWall)
						if wait > elapsed && wait-elapsed < maxVideoPaceWait {
							time.Sleep(wait - elapsed)
						}
					}
					lastTS = pkt.Timestamp
					lastWall = time.Now()
				}
			}

			r.mu.Lock()
			for peerID, t := range r.subs {
				if _, err = t.Write(buf[:n]); err != nil {
					delete(r.subs, peerID)
				}
			}
			r.mu.Unlock()
		}
	}()
}

func (r *Relay) RequestKeyFrame() {
	if r.isVideo && r.requestKeyframe != nil {
		r.requestKeyframe()
	}
}
