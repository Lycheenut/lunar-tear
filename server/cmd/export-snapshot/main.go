package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"

	_ "modernc.org/sqlite"
)

const localTestUserID int64 = 1

type exportResult struct {
	SourceUserID int64
	Bytes        int
	SHA256       string
}

func main() {
	dbPath := flag.String("db", "db/game.db", "SQLite database path")
	playerID := flag.Int64("player-id", 0, "In-game player ID to export (required)")
	flag.Parse()

	if *playerID <= 0 {
		log.Fatal("--player-id must be a positive integer")
	}

	result, err := exportSnapshot(*dbPath, *playerID, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("exported source user %d as local test user %d (%d bytes, sha256=%s)",
		result.SourceUserID, localTestUserID, result.Bytes, result.SHA256)
}

func exportSnapshot(dbPath string, playerID int64, out io.Writer) (exportResult, error) {
	db, err := openReadOnly(dbPath)
	if err != nil {
		return exportResult{}, fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	repo := sqlite.New(db, nil)
	userID, err := repo.GetUserByPlayerId(playerID)
	if err != nil {
		return exportResult{}, fmt.Errorf("find player %d: %w", playerID, err)
	}

	user, err := repo.LoadUser(userID)
	if err != nil {
		return exportResult{}, fmt.Errorf("load user %d: %w", userID, err)
	}
	prepareForLocalTest(&user)

	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return exportResult{}, fmt.Errorf("marshal snapshot: %w", err)
	}
	data = append(data, '\n')
	if _, err := out.Write(data); err != nil {
		return exportResult{}, fmt.Errorf("write snapshot: %w", err)
	}

	digest := sha256.Sum256(data)
	return exportResult{
		SourceUserID: userID,
		Bytes:        len(data),
		SHA256:       hex.EncodeToString(digest[:]),
	}, nil
}

func openReadOnly(path string) (*sql.DB, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	uriPath := filepath.ToSlash(absPath)
	if filepath.VolumeName(absPath) != "" {
		uriPath = "/" + uriPath
	}
	dsn := (&url.URL{
		Scheme:   "file",
		Path:     uriPath,
		RawQuery: "mode=ro&_pragma=busy_timeout(5000)",
	}).String()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func prepareForLocalTest(user *store.UserState) {
	user.UserId = localTestUserID
	user.PlayerId = localTestUserID
	user.Uuid = ""
	user.BirthYear = 2000
	user.BirthMonth = 1
	user.BackupToken = "mock-backup-token"
	user.ChargeMoneyThisMonth = 0
	user.FacebookId = 0

	user.Profile.Name = "Local Test Player"
	user.Profile.Message = ""
	user.Friends = make(map[int64]store.FriendState)
	user.FriendRequests = make(map[int64]int64)
	user.Notifications.FriendRequestReceiveCount = 0

	for key, deck := range user.Decks {
		deck.Name = ""
		user.Decks[key] = deck
	}
	for key, deck := range user.TripleDecks {
		deck.Name = ""
		user.TripleDecks[key] = deck
	}
	for key, preset := range user.PartsPresets {
		preset.Name = ""
		user.PartsPresets[key] = preset
	}
	for key, tag := range user.PartsPresetTags {
		tag.Name = ""
		user.PartsPresetTags[key] = tag
	}
}
