package domain

import "time"

type ChatMessage struct {
	Room      string    `json:"room"`
	From      string    `json:"from"`
	FromName  string    `json:"from_name,omitempty"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
