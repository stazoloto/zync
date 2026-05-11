package postgres

import "database/sql"

type DB interface {
    Exec(query string, args ...any) (sql.Result, error)
    Query(query string, args ...any) (*sql.Rows, error)
    QueryRow(query string, args ...any) *sql.Row
}

type Repositories struct {
	Users        *UserRepo
	Rooms        *RoomRepo
	Participants *ParticipantRepo
}

func New(db *sql.DB) *Repositories {
	return &Repositories{
		Users:        &UserRepo{db: db},
		Rooms:        &RoomRepo{db: db},
		Participants: &ParticipantRepo{db: db},
	}
}
