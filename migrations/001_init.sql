CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    first_name TEXT NOT NULL DEFAULT '',
    last_name TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT 'user',
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE rooms (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    is_active BOOLEAN NOT NULL DEFAULT true
);

-- Связь пользователей и комнат
CREATE TABLE room_participants (
   id BIGSERIAL PRIMARY KEY,
   room_id TEXT NOT NULL,
   user_id TEXT NOT NULL,
   joined_at TIMESTAMP NOT NULL DEFAULT now(),
   left_at TIMESTAMP,

   CONSTRAINT fk_room
       FOREIGN KEY(room_id)
           REFERENCES rooms(id)
           ON DELETE CASCADE,

   CONSTRAINT fk_user
       FOREIGN KEY(user_id)
           REFERENCES users(id)
           ON DELETE CASCADE,

   CONSTRAINT uq_room_user_active
       UNIQUE (room_id, user_id, joined_at)
);

CREATE INDEX idx_room_participants_room
    ON room_participants(room_id);

CREATE INDEX idx_room_participants_user
    ON room_participants(user_id);

CREATE INDEX idx_room_participants_active
    ON room_participants(room_id)
    WHERE left_at IS NULL;