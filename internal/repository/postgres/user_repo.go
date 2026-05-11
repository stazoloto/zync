package postgres

import (
	"database/sql"
	"errors"

	"github.com/stazoloto/zync/internal/domain"
)

type UserRepo struct {
	db DB
}

func (r *UserRepo) Register(email, firstName, lastName, passwordHash string) (string, error) {
	var id string
	err := r.db.QueryRow(`
		INSERT INTO users (id, email, first_name, last_name, password_hash)
		VALUES (gen_random_uuid(), $1, $2, $3, $4)
		RETURNING id
	`, email, firstName, lastName, passwordHash).Scan(&id)
	return id, err
}

func (r *UserRepo) GetByID(userID string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(`
		SELECT id, email, password_hash, first_name, last_name, role, created_at
		FROM users WHERE id = $1
	`, userID).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepo) GetByEmail(email string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(`
		SELECT id, email, password_hash, first_name, last_name, role, created_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role, &u.CreatedAt)
	return &u, err
}

func (r *UserRepo) ListUsers() ([]*domain.User, error) {
	rows, err := r.db.Query(`
		SELECT id, email, first_name, last_name, role, created_at
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func (r *UserRepo) DeleteUser(userID string) error {
	_, err := r.db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	return err
}

func (r *UserRepo) SetRole(userID, role string) error {
	_, err := r.db.Exec(`UPDATE users SET role = $1 WHERE id = $2`, role, userID)
	return err
}
