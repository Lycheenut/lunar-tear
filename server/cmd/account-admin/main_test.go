package main

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"lunar-tear/server/internal/auth"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
)

func TestAccountAdminLifecycle(t *testing.T) {
	directory := t.TempDir()
	gameDBPath := filepath.Join(directory, "game.db")
	authDBPath := filepath.Join(directory, "auth.db")
	var output bytes.Buffer

	if err := run([]string{
		"create", "--name", "alice", "--platform", "ios", "--password-stdin",
		"--db", gameDBPath, "--auth-db", authDBPath,
	}, strings.NewReader("old password\n"), &output, &output); err != nil {
		t.Fatal(err)
	}

	authDB, err := database.Open(authDBPath)
	if err != nil {
		t.Fatal(err)
	}
	authStore, err := auth.NewAuthStore(authDB)
	if err != nil {
		t.Fatal(err)
	}
	authUser, err := authStore.VerifyUser("alice", "old password")
	if err != nil {
		t.Fatal(err)
	}
	gameDB, err := database.Open(gameDBPath)
	if err != nil {
		t.Fatal(err)
	}
	gameStore := sqlite.New(gameDB, nil)
	userID, err := gameStore.GetUserByFacebookId(authUser.ID)
	if err != nil {
		t.Fatal(err)
	}
	user, err := gameStore.LoadUser(userID)
	if err != nil {
		t.Fatal(err)
	}
	if user.OsType != 1 || user.PlatformType != 1 {
		t.Fatalf("created platform = %d/%d, want 1/1", user.OsType, user.PlatformType)
	}
	gameDB.Close()
	authDB.Close()

	if err := run([]string{
		"password", "--name", "alice", "--password-stdin", "--auth-db", authDBPath,
	}, strings.NewReader("new password\n"), &output, &output); err != nil {
		t.Fatal(err)
	}
	authDB, err = database.Open(authDBPath)
	if err != nil {
		t.Fatal(err)
	}
	authStore, err = auth.NewAuthStore(authDB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authStore.VerifyUser("alice", "new password"); err != nil {
		t.Fatal(err)
	}
	authDB.Close()

	if err := run([]string{
		"delete", "--name", "alice", "--yes", "--db", gameDBPath, "--auth-db", authDBPath,
	}, strings.NewReader(""), &output, &output); err != nil {
		t.Fatal(err)
	}
	authDB, err = database.Open(authDBPath)
	if err != nil {
		t.Fatal(err)
	}
	authStore, err = auth.NewAuthStore(authDB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authStore.GetUser("alice"); !errors.Is(err, auth.ErrUserNotFound) {
		t.Fatalf("auth user after delete error = %v, want ErrUserNotFound", err)
	}
	authDB.Close()
	gameDB, err = database.Open(gameDBPath)
	if err != nil {
		t.Fatal(err)
	}
	gameStore = sqlite.New(gameDB, nil)
	if _, err := gameStore.LoadUser(userID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("game user after delete error = %v, want ErrNotFound", err)
	}
	gameDB.Close()
}

func TestDeleteRequiresMatchingConfirmation(t *testing.T) {
	directory := t.TempDir()
	gameDBPath := filepath.Join(directory, "game.db")
	authDBPath := filepath.Join(directory, "auth.db")
	var output bytes.Buffer
	if err := run([]string{
		"create", "--name", "alice", "--password-stdin", "--db", gameDBPath, "--auth-db", authDBPath,
	}, strings.NewReader("password\n"), &output, &output); err != nil {
		t.Fatal(err)
	}
	err := run([]string{
		"delete", "--name", "alice", "--db", gameDBPath, "--auth-db", authDBPath,
	}, strings.NewReader("wrong\n"), &output, &output)
	if err == nil || !strings.Contains(err.Error(), "confirmation did not match") {
		t.Fatalf("delete error = %v, want confirmation mismatch", err)
	}
}

func TestReadPasswordFromStdinRejectsEmptyPassword(t *testing.T) {
	_, err := readNewPassword(strings.NewReader("\n"), &bytes.Buffer{}, true)
	if err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("readNewPassword() error = %v, want empty password error", err)
	}
}
