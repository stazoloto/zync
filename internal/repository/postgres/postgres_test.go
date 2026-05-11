package postgres

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// newTestDB открывает соединение с тестовой БД.
// Каждый тест оборачивает свои операции в транзакцию и откатывает её —
// так база остаётся чистой без явного DELETE.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", "postgres://sfu_user:sfu_pswrd@localhost:5432/sfu_db?sslmode=disable")
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	t.Cleanup(func() { db.Close() })
	return db
}

// withTx открывает транзакцию и откатывает её после теста.
// Все операции внутри теста идут через эту транзакцию.
func withTx(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := db.Begin()
	require.NoError(t, err)
	t.Cleanup(func() { tx.Rollback() })
	return tx
}

// --- UserRepo ---

func TestUserRepo_Register(t *testing.T) {
	db := newTestDB(t)
	tx := withTx(t, db)
	repo := &UserRepo{db: tx}

	id, err := repo.Register("test@example.com", "Test", "User", "hashedpassword")
	require.NoError(t, err)
	require.NotEmpty(t, id)

	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", id).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestUserRepo_GetByEmail(t *testing.T) {
	db := newTestDB(t)
	tx := withTx(t, db)
	repo := &UserRepo{db: tx}

	_, err := repo.Register("lookup@example.com", "Lookup", "User", "hashedpassword")
	require.NoError(t, err)

	user, err := repo.GetByEmail("lookup@example.com")
	require.NoError(t, err)
	require.Equal(t, "lookup@example.com", user.Email)
	require.NotEmpty(t, user.ID)
}

// --- RoomRepo ---

func TestRoomRepo_CreateIfNotExists(t *testing.T) {
	db := newTestDB(t)
	tx := withTx(t, db)
	repo := &RoomRepo{db: tx}

	err := repo.CreateIfNotExists("room-1")
	require.NoError(t, err)

	var isActive bool
	err = tx.QueryRow("SELECT is_active FROM rooms WHERE id = $1", "room-1").Scan(&isActive)
	require.NoError(t, err)
	require.True(t, isActive)
}

func TestRoomRepo_CreateIfNotExists_Double(t *testing.T) {
	db := newTestDB(t)
	tx := withTx(t, db)
	repo := &RoomRepo{db: tx}

	require.NoError(t, repo.CreateIfNotExists("room-dup"))
	require.NoError(t, repo.CreateIfNotExists("room-dup"))

	var count int
	err := tx.QueryRow("SELECT COUNT(*) FROM rooms WHERE id = $1", "room-dup").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestRoomRepo_SetInactive(t *testing.T) {
	db := newTestDB(t)
	tx := withTx(t, db)
	repo := &RoomRepo{db: tx}

	require.NoError(t, repo.CreateIfNotExists("room-inactive"))
	require.NoError(t, repo.SetInactive("room-inactive"))

	var isActive bool
	err := tx.QueryRow("SELECT is_active FROM rooms WHERE id = $1", "room-inactive").Scan(&isActive)
	require.NoError(t, err)
	require.False(t, isActive)
}

// --- ParticipantRepo ---

// setupParticipantRepo создаёт пользователя и комнату внутри транзакции
// — они нужны из-за foreign key constraints.
func setupParticipantRepo(t *testing.T, tx *sql.Tx, roomID, userID string) *ParticipantRepo {
	t.Helper()
	_, err := tx.Exec("INSERT INTO users (id, email, password_hash) VALUES ($1, $2, 'hash') ON CONFLICT DO NOTHING",
		userID, userID+"@test.com")
	require.NoError(t, err)
	_, err = tx.Exec("INSERT INTO rooms (id) VALUES ($1) ON CONFLICT DO NOTHING", roomID)
	require.NoError(t, err)
	return &ParticipantRepo{db: tx}
}

func TestParticipantRepo_JoinRoom(t *testing.T) {
	db := newTestDB(t)
	tx := withTx(t, db)
	repo := setupParticipantRepo(t, tx, "room-join", "user-join")

	require.NoError(t, repo.JoinRoom("room-join", "user-join"))

	participants, err := repo.GetActiveParticipants("room-join")
	require.NoError(t, err)
	require.Contains(t, participants, "user-join")
}

func TestParticipantRepo_LeaveRoom(t *testing.T) {
	db := newTestDB(t)
	tx := withTx(t, db)
	repo := setupParticipantRepo(t, tx, "room-leave", "user-leave")

	require.NoError(t, repo.JoinRoom("room-leave", "user-leave"))
	require.NoError(t, repo.LeaveRoom("room-leave", "user-leave"))

	participants, err := repo.GetActiveParticipants("room-leave")
	require.NoError(t, err)
	require.Empty(t, participants)
}

// TestParticipantRepo_LeaveRoom_NotExist проверяет что выход несуществующего
// участника не возвращает ошибку.
func TestParticipantRepo_LeaveRoom_NotExist(t *testing.T) {
	db := newTestDB(t)
	tx := withTx(t, db)
	setupParticipantRepo(t, tx, "room-leave-ghost", "user-leave-ghost")

	err := (&ParticipantRepo{db: tx}).LeaveRoom("room-leave-ghost", "user-leave-ghost")
	require.NoError(t, err)
}

func TestParticipantRepo_HasActiveParticipants_True(t *testing.T) {
	db := newTestDB(t)
	tx := withTx(t, db)
	repo := setupParticipantRepo(t, tx, "room-has", "user-has")

	require.NoError(t, repo.JoinRoom("room-has", "user-has"))

	has, err := repo.HasActiveParticipants("room-has")
	require.NoError(t, err)
	require.True(t, has)
}

func TestParticipantRepo_HasActiveParticipants_False(t *testing.T) {
	db := newTestDB(t)
	tx := withTx(t, db)
	repo := setupParticipantRepo(t, tx, "room-empty", "user-empty")

	require.NoError(t, repo.JoinRoom("room-empty", "user-empty"))
	require.NoError(t, repo.LeaveRoom("room-empty", "user-empty"))

	has, err := repo.HasActiveParticipants("room-empty")
	require.NoError(t, err)
	require.False(t, has)
}

func TestParticipantRepo_GetActiveParticipants_Multiple(t *testing.T) {
	db := newTestDB(t)
	tx := withTx(t, db)

	_, err := tx.Exec("INSERT INTO rooms (id) VALUES ($1) ON CONFLICT DO NOTHING", "room-multi")
	require.NoError(t, err)

	users := []string{"user-m1", "user-m2", "user-m3"}
	for _, u := range users {
		_, err = tx.Exec("INSERT INTO users (id, email, password_hash) VALUES ($1, $2, 'hash') ON CONFLICT DO NOTHING",
			u, u+"@test.com")
		require.NoError(t, err)
	}

	repo := &ParticipantRepo{db: tx}
	for _, u := range users {
		require.NoError(t, repo.JoinRoom("room-multi", u))
	}

	participants, err := repo.GetActiveParticipants("room-multi")
	require.NoError(t, err)
	require.Len(t, participants, 3)
	for _, u := range users {
		require.Contains(t, participants, u)
	}
}
