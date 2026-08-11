package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists   = errors.New("username already taken")
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidCreds = errors.New("invalid username or password")
)

type AuthUser struct {
	ID       int64
	Username string
}

type AuthStore struct {
	db *sql.DB
}

func NewAuthStore(db *sql.DB) (*AuthStore, error) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS auth_users (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			username   TEXT    NOT NULL UNIQUE,
			password   BLOB   NOT NULL,
			created_at TEXT    NOT NULL
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("create auth_users table: %w", err)
	}
	return &AuthStore{db: db}, nil
}

func (s *AuthStore) CreateUser(username, password string) (AuthUser, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return AuthUser{}, err
	}

	res, err := s.db.Exec(
		`INSERT INTO auth_users (username, password, created_at) VALUES (?, ?, ?)`,
		username, hash, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return AuthUser{}, ErrUserExists
		}
		return AuthUser{}, fmt.Errorf("insert user: %w", err)
	}

	id, _ := res.LastInsertId()
	return AuthUser{ID: id, Username: username}, nil
}

func (s *AuthStore) GetUser(username string) (AuthUser, error) {
	var user AuthUser
	err := s.db.QueryRow(
		`SELECT id, username FROM auth_users WHERE username = ?`, username,
	).Scan(&user.ID, &user.Username)
	if err == sql.ErrNoRows {
		return AuthUser{}, ErrUserNotFound
	}
	if err != nil {
		return AuthUser{}, fmt.Errorf("query user: %w", err)
	}
	return user, nil
}

func (s *AuthStore) UpdatePassword(username, password string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}

	result, err := s.db.Exec(`UPDATE auth_users SET password = ? WHERE username = ?`, hash, username)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated row count: %w", err)
	}
	if updated == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *AuthStore) DeleteUser(id int64) error {
	result, err := s.db.Exec(`DELETE FROM auth_users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted row count: %w", err)
	}
	if deleted == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *AuthStore) VerifyUser(username, password string) (AuthUser, error) {
	var id int64
	var hash []byte
	err := s.db.QueryRow(
		`SELECT id, password FROM auth_users WHERE username = ?`, username,
	).Scan(&id, &hash)
	if err != nil {
		return AuthUser{}, ErrInvalidCreds
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		return AuthUser{}, ErrInvalidCreds
	}

	return AuthUser{ID: id, Username: username}, nil
}

func (s *AuthStore) UserExists(username string) bool {
	var n int
	err := s.db.QueryRow(`SELECT 1 FROM auth_users WHERE username = ?`, username).Scan(&n)
	return err == nil
}

func hashPassword(password string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	return hash, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && (errors.Is(err, sql.ErrNoRows) ||
		// modernc.org/sqlite returns error strings containing "UNIQUE constraint failed"
		fmt.Sprintf("%v", err) == fmt.Sprintf("%v", err) &&
			contains(err.Error(), "UNIQUE constraint failed"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
