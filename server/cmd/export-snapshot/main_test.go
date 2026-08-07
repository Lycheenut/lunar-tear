package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func TestExportSnapshotRedactsAndRoundTrips(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "game.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrations.Up(context.Background(), db); err != nil {
		db.Close()
		t.Fatal(err)
	}

	repo := sqlite.New(db, nil)
	if _, err := repo.CreateUser("first", model.ClientPlatform{}); err != nil {
		db.Close()
		t.Fatal(err)
	}
	userID, err := repo.CreateUser("production-uuid", model.ClientPlatform{})
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	user, err := repo.UpdateUser(userID, func(user *store.UserState) {
		user.BirthYear = 1980
		user.BirthMonth = 12
		user.BackupToken = "production-backup-token"
		user.ChargeMoneyThisMonth = 999
		user.Profile.Name = "Production Name"
		user.Profile.Message = "Production Message"
		user.Friends[3] = store.FriendState{IsFriend: true}
		user.FriendRequests[4] = 10
		user.Notifications.FriendRequestReceiveCount = 2
		user.Decks[store.DeckKey{DeckType: 1, UserDeckNumber: 1}] = store.DeckState{Name: "Private deck"}
		user.TripleDecks[store.DeckKey{DeckType: 2, UserDeckNumber: 1}] = store.TripleDeckState{Name: "Private triple deck"}
		user.PartsPresets[1] = store.PartsPresetState{Name: "Private preset"}
		user.PartsPresetTags[1] = store.PartsPresetTagState{Name: "Private tag"}
		user.PartsStatusSubs[store.PartsStatusSubKey{UserPartsUuid: "part-uuid", StatusIndex: 2}] = store.PartsStatusSubState{Level: 3}
	})
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := repo.SetFacebookId(userID, 123); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	result, err := exportSnapshot(dbPath, user.PlayerId, &output)
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceUserID != userID || result.Bytes != output.Len() || result.SHA256 == "" {
		t.Fatalf("unexpected export result: %+v", result)
	}

	var snapshot store.UserState
	if err := json.Unmarshal(output.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.UserId != 1 || snapshot.PlayerId != 1 || snapshot.Uuid != "" {
		t.Fatalf("identity was not rebound: user=%d player=%d uuid=%q", snapshot.UserId, snapshot.PlayerId, snapshot.Uuid)
	}
	if snapshot.BirthYear != 2000 || snapshot.BirthMonth != 1 || snapshot.BackupToken != "mock-backup-token" || snapshot.ChargeMoneyThisMonth != 0 || snapshot.FacebookId != 0 {
		t.Fatal("account metadata was not redacted")
	}
	if snapshot.Profile.Name != "Local Test Player" || snapshot.Profile.Message != "" {
		t.Fatal("profile was not redacted")
	}
	if len(snapshot.Friends) != 0 || len(snapshot.FriendRequests) != 0 || snapshot.Notifications.FriendRequestReceiveCount != 0 {
		t.Fatal("social state was not cleared")
	}
	for _, deck := range snapshot.Decks {
		if deck.Name != "" {
			t.Fatal("deck name was not redacted")
		}
	}
	for _, deck := range snapshot.TripleDecks {
		if deck.Name != "" {
			t.Fatal("triple deck name was not redacted")
		}
	}
	for _, preset := range snapshot.PartsPresets {
		if preset.Name != "" {
			t.Fatal("parts preset name was not redacted")
		}
	}
	for _, tag := range snapshot.PartsPresetTags {
		if tag.Name != "" {
			t.Fatal("parts preset tag name was not redacted")
		}
	}
	key := store.PartsStatusSubKey{UserPartsUuid: "part-uuid", StatusIndex: 2}
	if snapshot.PartsStatusSubs[key].Level != 3 {
		t.Fatal("parts status sub-state did not survive JSON round trip")
	}
}

func TestOpenReadOnlyRejectsWrites(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "game.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE check_read_only (id INTEGER)`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	readOnly, err := openReadOnly(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer readOnly.Close()
	if _, err := readOnly.Exec(`INSERT INTO check_read_only (id) VALUES (1)`); err == nil {
		t.Fatal("read-only connection accepted a write")
	}
}
