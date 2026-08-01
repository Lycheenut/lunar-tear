package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/store/sqlite"
)

func main() {
	dbPath := flag.String("db", "db/game.db", "SQLite database path")
	name := flag.String("name", "", "In-game player name to look up in user_profile (required)")
	flag.Parse()

	if *name == "" {
		log.Fatal("--name flag is required")
	}

	db, err := database.Open(*dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	var targetId int64
	err = db.QueryRow(`SELECT user_id FROM user_profile WHERE name = ?`, *name).Scan(&targetId)
	if err == sql.ErrNoRows {
		log.Fatalf("no user found with name %q", *name)
	}
	if err != nil {
		log.Fatalf("query user_profile: %v", err)
	}

	var targetUuid string
	err = db.QueryRow(`SELECT uuid FROM users WHERE user_id = ?`, targetId).Scan(&targetUuid)
	if err != nil {
		log.Fatalf("query target uuid: %v", err)
	}

	var latestId int64
	var latestUuid string
	err = db.QueryRow(`SELECT user_id, uuid FROM users ORDER BY user_id DESC LIMIT 1`).Scan(&latestId, &latestUuid)
	if err != nil {
		log.Fatalf("query latest user: %v", err)
	}

	if targetId == latestId {
		log.Printf("user %q (id=%d) is already the most recent user, nothing to do", *name, targetId)
		return
	}

	log.Printf("target:  id=%d uuid=%s (name=%q)", targetId, targetUuid, *name)
	log.Printf("latest:  id=%d uuid=%s (will be deleted)", latestId, latestUuid)

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM user_friend_requests WHERE requester_user_id = ?`, latestId); err != nil {
		log.Fatalf("delete outgoing friend requests: %v", err)
	}
	if _, err := tx.Exec(`DELETE FROM user_friends WHERE friend_user_id = ?`, latestId); err != nil {
		log.Fatalf("delete inbound friend relationships: %v", err)
	}

	for _, t := range sqlite.UserOwnedTables() {
		if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE user_id = ?`, t), latestId); err != nil {
			log.Fatalf("delete from %s: %v", t, err)
		}
	}
	if _, err := tx.Exec(`DELETE FROM users WHERE user_id = ?`, latestId); err != nil {
		log.Fatalf("delete latest user: %v", err)
	}

	if _, err := tx.Exec(`UPDATE users SET uuid = ? WHERE user_id = ?`, latestUuid, targetId); err != nil {
		log.Fatalf("update target uuid: %v", err)
	}

	if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id = ?`, targetId); err != nil {
		log.Fatalf("delete stale sessions: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("commit: %v", err)
	}

	fmt.Printf("claimed account:\n")
	fmt.Printf("  user %d (%s): uuid changed %s -> %s\n", targetId, *name, targetUuid, latestUuid)
	fmt.Printf("  user %d: deleted\n", latestId)
}
