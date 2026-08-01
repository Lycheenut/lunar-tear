package sqlite

import (
	"database/sql"
	"time"

	"lunar-tear/server/internal/store"
)

type SQLiteStore struct {
	db        *sql.DB
	clock     store.Clock
	userLocks userLockManager
}

var (
	_ store.UserRepository    = (*SQLiteStore)(nil)
	_ store.SessionRepository = (*SQLiteStore)(nil)
)

func New(db *sql.DB, clock store.Clock) *SQLiteStore {
	if clock == nil {
		clock = time.Now
	}
	return &SQLiteStore{db: db, clock: clock}
}
