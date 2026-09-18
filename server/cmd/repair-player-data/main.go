// repair-player-data performs the one-time mission/login compensation repair.
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

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"

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
	if err := memorydb.Init(*masterPath); err != nil {
		log.Fatal(err)
	}
	bonuses, err := memorydb.ReadTable[masterdata.EntityMLoginBonus]("m_login_bonus")
	if err != nil {
		log.Fatal(err)
	}
	stamps, err := memorydb.ReadTable[masterdata.EntityMLoginBonusStamp]("m_login_bonus_stamp")
	if err != nil {
		log.Fatal(err)
	}
	schedule, err := makeSchedule(bonuses, stamps)
	if err != nil {
		log.Fatal(err)
	}
	db, err := openDatabase(*dbPath, *apply)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	report, err := repair(db, schedule, *apply, time.Now().UnixMilli())
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		log.Fatal(err)
	}
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
		// Acquire the write lock before reading so the plan and commit share
		// one snapshot. mode=rw must never create an empty database on a typo.
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
