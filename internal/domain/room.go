package domain

import "time"

type Room struct {
	ID        string     `json:"id"`
	OwnerID   string     `json:"owner_id"`
	Name      string     `json:"name,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	IsActive  bool       `json:"is_active"`
}

func NewRoom(id, ownerID, name string, now time.Time) *Room {
	now = now.UTC()

	return &Room{
		ID:        id,
		OwnerID:   ownerID,
		Name:      name,
		CreatedAt: now,
		StartedAt: now,
		IsActive:  true,
	}
}

func (r *Room) Activate() {
	r.IsActive = true
	r.EndedAt = nil
}

func (r *Room) End(now time.Time) {
	endedAt := now.UTC()
	r.EndedAt = &endedAt
	r.IsActive = false
}

func (r *Room) IsEnded() bool {
	return r.EndedAt != nil
}

func (r *Room) CanBeJoined() bool {
	return r.IsActive && !r.IsEnded()
}
