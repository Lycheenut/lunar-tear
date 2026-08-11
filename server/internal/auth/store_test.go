package auth

import (
	"errors"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/database"
)

func TestAuthStoreAccountLifecycle(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store, err := NewAuthStore(db)
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.CreateUser("alice", "old password")
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.GetUser("alice")
	if err != nil {
		t.Fatal(err)
	}
	if got != created {
		t.Fatalf("GetUser() = %+v, want %+v", got, created)
	}

	if err := store.UpdatePassword("alice", "new password"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.VerifyUser("alice", "old password"); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("old password error = %v, want ErrInvalidCreds", err)
	}
	if _, err := store.VerifyUser("alice", "new password"); err != nil {
		t.Fatalf("verify new password: %v", err)
	}

	if err := store.DeleteUser(created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetUser("alice"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("GetUser() after delete error = %v, want ErrUserNotFound", err)
	}
}

func TestAuthStoreMissingUserMutations(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store, err := NewAuthStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdatePassword("missing", "password"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("UpdatePassword() error = %v, want ErrUserNotFound", err)
	}
	if err := store.DeleteUser(123); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("DeleteUser() error = %v, want ErrUserNotFound", err)
	}
}
