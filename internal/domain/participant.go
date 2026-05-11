package domain

import (
	"time"
)

type ParticipantRole string

const (
	ParticipantRoleHost        ParticipantRole = "host"
	ParticipantRoleParticipant ParticipantRole = "participant"
)

type Participant struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	RoomID          string          `json:"room_id"`
	Role            ParticipantRole `json:"role"`
	JoinedAt        time.Time       `json:"joined_at"`
	LeftAt          *time.Time      `json:"left_at,omitempty"`
	IsAudioMuted    bool            `json:"is_audio_muted"`
	IsVideoDisabled bool            `json:"is_video_disabled"`
	IsScreenSharing bool            `json:"is_screen_sharing"`
}

func NewParticipant(id, userID, roomID string, role ParticipantRole, now time.Time) *Participant {
	if role == "" {
		role = ParticipantRoleParticipant
	}

	return &Participant{
		ID:       id,
		UserID:   userID,
		RoomID:   roomID,
		Role:     role,
		JoinedAt: now.UTC(),
	}
}

func (p *Participant) Leave(now time.Time) {
	leftAt := now.UTC()
	p.LeftAt = &leftAt
	p.IsScreenSharing = false
}

func (p *Participant) MuteAudio() {
	p.IsAudioMuted = true
}

func (p *Participant) UnmuteAudio() {
	p.IsAudioMuted = false
}

func (p *Participant) DisableVideo() {
	p.IsVideoDisabled = true
}

func (p *Participant) EnableVideo() {
	p.IsVideoDisabled = false
}

func (p *Participant) StartScreenShare() {
	p.IsScreenSharing = true
}

func (p *Participant) StopScreenShare() {
	p.IsScreenSharing = false
}

func (p *Participant) IsActive() bool {
	return p.LeftAt == nil
}

func (p *Participant) IsHost() bool {
	return p.Role == ParticipantRoleHost
}
