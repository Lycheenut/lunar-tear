// repair-player-data backfills login-unlock and companion rewards for eligible players.
// Stop the game server and back up its database before using --apply.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := flag.String("db", "db/game.db", "Existing SQLite database (stop the game server before applying)")
	masterPath := flag.String("master-data", "assets/release/20240404193219.bin.e", "Master data used by the target game server")
	apply := flag.Bool("apply", false, "Commit the repair; default is a read-only preview")
	flag.Parse()
	if flag.NArg() != 0 {
		log.Fatal("unexpected positional arguments")
	}
	granter, err := loadGranter(*masterPath)
	if err != nil {
		log.Fatal(err)
	}
	db, err := openDatabase(*dbPath, *apply)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	report, err := repair(db, granter, *apply, time.Now().UnixMilli())
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		log.Fatal(err)
	}
}

func loadGranter(masterPath string) (*store.PossessionGranter, error) {
	data, err := os.ReadFile(masterPath)
	if err != nil {
		return nil, err
	}
	// NewHolder touches its input file; keep the source master data unchanged.
	dir, err := os.MkdirTemp("", "login-unlock-backfill-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	copyPath := filepath.Join(dir, "master.bin.e")
	if err := os.WriteFile(copyPath, data, 0o600); err != nil {
		return nil, err
	}
	holder, err := runtime.NewHolder(copyPath)
	if err != nil {
		return nil, err
	}
	return holder.Get().QuestHandler.Granter, nil
}

func openDatabase(path string, apply bool) (*sql.DB, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(absPath)
	if filepath.VolumeName(absPath) != "" {
		uriPath = "/" + uriPath
	}
	query := "mode=ro&_pragma=busy_timeout(5000)"
	if apply {
		// mode=rw must never create an empty database on a typo.
		query = "mode=rw&_txlock=immediate&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	}
	db, err := sql.Open("sqlite", (&url.URL{Scheme: "file", Path: uriPath, RawQuery: query}).String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
