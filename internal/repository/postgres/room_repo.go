package postgres

type RoomRepo struct {
	db DB
}

func (r *RoomRepo) CreateIfNotExists(roomID string) error {
	_, err := r.db.Exec(`
		INSERT INTO rooms (id)
		VALUES ($1)
		ON CONFLICT (id) DO NOTHING
	`, roomID)
	return err
}

func (r *RoomRepo) SetInactive(roomID string) error {
	_, err := r.db.Exec(`
		UPDATE rooms
		SET is_active = false
		WHERE id = $1
	`, roomID)
	return err
}

func (r *RoomRepo) ListActive() ([]string, error) {
	rows, err := r.db.Query(`SELECT id FROM rooms WHERE is_active = true ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
