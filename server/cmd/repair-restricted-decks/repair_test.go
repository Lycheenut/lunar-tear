package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func fixture(t *testing.T) (*sql.DB, string, []store.UserState) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "game #1.db")
	db, err := database.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	var users []store.UserState
	for id := int64(1); id <= 2; id++ {
		user := store.SeedUserState(id, fmt.Sprintf("repair-%d", id), 10, model.ClientPlatform{})
		user.PlayerId = 100 + id
		user.Costumes["costume"] = store.CostumeState{UserCostumeUuid: "costume", CostumeId: 1}
		user.Weapons["main"] = store.WeaponState{UserWeaponUuid: "main", WeaponId: 1}
		user.Weapons["sub1"] = store.WeaponState{UserWeaponUuid: "sub1", WeaponId: 2}
		user.Weapons["sub2"] = store.WeaponState{UserWeaponUuid: "sub2", WeaponId: 3}
		user.Companions["companion"] = store.CompanionState{UserCompanionUuid: "companion", CompanionId: 1}
		user.Thoughts["debris"] = store.ThoughtState{UserThoughtUuid: "debris", ThoughtId: 1}
		for i := 1; i <= 3; i++ {
			uuid := fmt.Sprintf("part%d", i)
			user.Parts[uuid] = store.PartsState{UserPartsUuid: uuid, PartsId: int32(i)}
		}
		for _, uuid := range []string{"r1", "r2", "r3", "r4", "shared", "keep2", "keep3", "keep5", "orphan", "missing"} {
			if uuid != "missing" {
				user.DeckCharacters[uuid] = store.DeckCharacterState{
					UserDeckCharacterUuid: uuid, UserCostumeUuid: "costume", MainUserWeaponUuid: "main",
					UserCompanionUuid: "companion", UserThoughtUuid: "debris", DressupCostumeId: 123,
					Power: 900, LatestVersion: 10,
				}
			}
			user.DeckSubWeapons[uuid] = []string{"sub1", "sub2"}
			user.DeckParts[uuid] = []string{"part1", "part2", "part3"}
		}
		user.Decks = map[store.DeckKey]store.DeckState{}
		for _, deck := range []store.DeckState{
			{DeckType: 4, UserDeckNumber: 1, UserDeckCharacterUuid01: "r1", UserDeckCharacterUuid02: "r2", UserDeckCharacterUuid03: "r3", Power: 2700},
			{DeckType: 4, UserDeckNumber: 2, Power: 100}, // Empty slots with stale power.
			{DeckType: 4, UserDeckNumber: 3},             // Already empty: leave its version alone.
			{DeckType: 6, UserDeckNumber: 1, UserDeckCharacterUuid01: "r1", UserDeckCharacterUuid02: "r4", UserDeckCharacterUuid03: "shared", Power: 2700},
			{DeckType: 6, UserDeckNumber: 2, UserDeckCharacterUuid02: "missing", Power: 900},
			{DeckType: 1, UserDeckNumber: 1, UserDeckCharacterUuid03: "shared", Power: 900},
			{DeckType: 2, UserDeckNumber: 1, UserDeckCharacterUuid01: "keep2", Power: 900},
			{DeckType: 3, UserDeckNumber: 1, UserDeckCharacterUuid02: "keep3", Power: 900},
			{DeckType: 5, UserDeckNumber: 1, UserDeckCharacterUuid03: "keep5", Power: 900},
		} {
			deck.Name = fmt.Sprintf("编队 %d/%d", deck.DeckType, deck.UserDeckNumber)
			deck.LatestVersion = 10
			user.Decks[store.DeckKey{DeckType: deck.DeckType, UserDeckNumber: deck.UserDeckNumber}] = deck
		}
		user.DeckTypeNotes[4] = store.DeckTypeNoteState{DeckType: 4, MaxDeckPower: 9000, LatestVersion: 10}
		user.TripleDecks[store.DeckKey{DeckType: 5, UserDeckNumber: 1}] = store.TripleDeckState{
			DeckType: 5, UserDeckNumber: 1, Name: "Triple", DeckNumber01: 1, LatestVersion: 10,
		}
		user.DeckLimitContentRestricted["lock"] = store.DeckLimitContentRestrictedState{
			DeckRestrictedUuid: "lock", EventQuestChapterId: 1, QuestId: 1,
			PossessionType: 1, TargetUuid: "costume", LatestVersion: 10,
		}
		user.Quests[1] = store.UserQuestState{QuestId: 1, ClearCount: 2, UserDeckNumber: 1}
		if err := repo.ImportUser(user); err != nil {
			t.Fatal(err)
		}
		loaded, err := repo.LoadUser(id)
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, loaded)
	}
	return db, path, users
}

func assertUsers(t *testing.T, db *sql.DB, want []store.UserState) {
	t.Helper()
	for _, user := range want {
		after, err := sqlite.New(db, nil).LoadUser(user.UserId)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(user, after) {
			t.Fatalf("unexpected changes to player %d", user.PlayerId)
		}
	}
}

func TestRepairPreviewApplyAndIdempotence(t *testing.T) {
	db, path, before := fixture(t)
	readOnly, err := openDatabase(path, false)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := repair(readOnly, before[0].PlayerId, false, 1000)
	readOnly.Close()
	if err != nil {
		t.Fatal(err)
	}
	if preview.Applied || preview.UserID != 1 || preview.PlayerID != 101 || len(preview.Decks) != 4 ||
		preview.RemovedDeckCharacters != 4 || preview.RemovedSubWeapons != 10 || preview.RemovedParts != 15 ||
		!reflect.DeepEqual(preview.SharedDeckCharactersKept, []string{"shared"}) {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	assertUsers(t, db, before)
	writable, err := openDatabase(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer writable.Close()
	applied, err := repair(writable, before[0].PlayerId, true, 1000)
	if err != nil {
		t.Fatal(err)
	}
	preview.Applied = true
	if !reflect.DeepEqual(preview, applied) {
		t.Fatalf("application differs from preview: %+v", applied)
	}

	want := before[0]
	want.Decks = maps.Clone(want.Decks)
	want.DeckCharacters = maps.Clone(want.DeckCharacters)
	want.DeckSubWeapons = maps.Clone(want.DeckSubWeapons)
	want.DeckParts = maps.Clone(want.DeckParts)
	for _, key := range []store.DeckKey{{DeckType: 4, UserDeckNumber: 1}, {DeckType: 4, UserDeckNumber: 2},
		{DeckType: 6, UserDeckNumber: 1}, {DeckType: 6, UserDeckNumber: 2}} {
		deck := want.Decks[key]
		deck.UserDeckCharacterUuid01 = ""
		deck.UserDeckCharacterUuid02 = ""
		deck.UserDeckCharacterUuid03 = ""
		deck.Power = 0
		deck.LatestVersion = 1000
		want.Decks[key] = deck
	}
	for _, uuid := range []string{"r1", "r2", "r3", "r4", "missing"} {
		delete(want.DeckCharacters, uuid)
		delete(want.DeckSubWeapons, uuid)
		delete(want.DeckParts, uuid)
	}
	// Compare the complete saved states, including inventory, progression, and
	// another player using the same deck-character UUIDs.
	assertUsers(t, db, []store.UserState{want, before[1]})
	repeated, err := repair(writable, before[0].PlayerId, true, 2000)
	if err != nil {
		t.Fatal(err)
	}
	if !repeated.Applied || len(repeated.Decks) != 0 || repeated.RemovedDeckCharacters != 0 ||
		repeated.RemovedSubWeapons != 0 || repeated.RemovedParts != 0 || len(repeated.SharedDeckCharactersKept) != 0 {
		t.Fatalf("second repair should be a no-op: %+v", repeated)
	}
	assertUsers(t, db, []store.UserState{want, before[1]})
}

func TestRepairRollsBackOnFailure(t *testing.T) {
	db, path, before := fixture(t)
	if _, err := db.Exec(`CREATE TRIGGER fail_repair BEFORE UPDATE ON user_decks
		WHEN NEW.user_id = 1 AND NEW.deck_type = 6
		BEGIN SELECT RAISE(ABORT, 'injected failure'); END`); err != nil {
		t.Fatal(err)
	}
	writable, err := openDatabase(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer writable.Close()
	report, err := repair(writable, 101, true, 1000)
	if err == nil || !strings.Contains(err.Error(), "injected failure") || report.Applied {
		t.Fatalf("expected failed transaction: report=%+v err=%v", report, err)
	}
	assertUsers(t, db, before)
}

func TestRepairRejectsUnknownAndAmbiguousPlayer(t *testing.T) {
	db, _, before := fixture(t)
	if _, err := repair(db, 999, true, 1000); err == nil {
		t.Fatal("unknown player ID accepted")
	}
	assertUsers(t, db, before)
	if _, err := db.Exec(`UPDATE users SET player_id = 101 WHERE user_id = 2`); err != nil {
		t.Fatal(err)
	}
	before[1].PlayerId = 101
	if _, err := repair(db, 101, true, 1000); err == nil {
		t.Fatal("ambiguous player ID accepted")
	}
	assertUsers(t, db, before)
}

func TestCommand(t *testing.T) {
	db, path, before := fixture(t)
	var output bytes.Buffer
	args := []string{"--db", path, "--player-id", "101"}
	if err := run(args, &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	var preview repairReport
	if err := json.Unmarshal(output.Bytes(), &preview); err != nil || preview.Applied || len(preview.Decks) != 4 {
		t.Fatalf("invalid preview JSON: %s, err=%v", output.String(), err)
	}
	assertUsers(t, db, before)
	output.Reset()
	if err := run(append(args, "--apply"), &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	var applied repairReport
	if err := json.Unmarshal(output.Bytes(), &applied); err != nil || !applied.Applied {
		t.Fatalf("invalid application JSON: %s, err=%v", output.String(), err)
	}
	preview.Applied = true
	if !reflect.DeepEqual(preview, applied) {
		t.Fatal("command application does not match preview")
	}
	output.Reset()
	if err := run(args, &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	var empty repairReport
	if err := json.Unmarshal(output.Bytes(), &empty); err != nil || len(empty.Decks) != 0 {
		t.Fatalf("decks were not cleared: %s, err=%v", output.String(), err)
	}
}

func TestCommandRequiresExplicitPlayerAndExistingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	for _, args := range [][]string{
		{"--db", path},
		{"--db", path, "--player-id", "0"},
		{"--db", path, "--player-id", "-1"},
		{"--db", path, "--player-id", "abc"},
		{"--db", path, "--player-id", "101", "extra"},
		{"--db", path, "--player-id", "101"},
		{"--db", path, "--player-id", "101", "--apply"},
	} {
		if err := run(args, io.Discard, io.Discard); err == nil {
			t.Errorf("expected failure for %v", args)
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("command created a missing database: %v", err)
	}
}
