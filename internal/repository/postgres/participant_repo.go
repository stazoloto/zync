package postgres

import (
	"time"

	"github.com/stazoloto/zync/internal/domain"
)

type ParticipantRepo struct {
	db DB
}

func (r *ParticipantRepo) JoinRoom(roomID, userID string) error {
	_, err := r.db.Exec(`
		INSERT INTO room_participants (room_id, user_id, role)
		VALUES ($1, $2, CASE
			WHEN NOT EXISTS (
				SELECT 1 FROM room_participants
				WHERE room_id = $1 AND left_at IS NULL
			) THEN 'host'
			ELSE 'participant'
		END)
	`, roomID, userID)
	return err
}

func (r *ParticipantRepo) LeaveRoom(roomID, userID string) error {
	_, err := r.db.Exec(`
		UPDATE room_participants
		SET left_at = $3
		WHERE room_id = $1
		  AND user_id = $2
		  AND left_at IS NULL
	`, roomID, userID, time.Now())
	return err
}

func (r *ParticipantRepo) GetActiveParticipants(roomID string) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT user_id
		FROM room_participants
		WHERE room_id = $1
		  AND left_at IS NULL
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		users = append(users, id)
	}
	return users, nil
}

func (r *ParticipantRepo) HasActiveParticipants(roomID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM room_participants
			WHERE room_id = $1
			  AND left_at IS NULL
		)
	`, roomID).Scan(&exists)
	return exists, err
}

func (r *ParticipantRepo) GetRole(roomID, userID string) (domain.ParticipantRole, error) {
	var role domain.ParticipantRole
	err := r.db.QueryRow(`
		SELECT role FROM room_participants
		WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL
		ORDER BY joined_at DESC LIMIT 1
	`, roomID, userID).Scan(&role)
	return role, err
}
