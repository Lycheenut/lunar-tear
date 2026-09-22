// repair-restricted-decks empties one player's condition-restricted formations.
// Stop the game server and back up its database before using --apply.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("repair-restricted-decks", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dbPath := flags.String("db", "db/game.db", "Existing SQLite database (stop the game server before applying)")
	playerID := flags.Int64("player-id", 0, "In-game player ID to repair (required)")
	apply := flags.Bool("apply", false, "Commit the repair; default is a read-only preview")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *playerID <= 0 {
		return fmt.Errorf("--player-id must be a positive integer")
	}
	db, err := openDatabase(*dbPath, *apply)
	if err != nil {
		return err
	}
	defer db.Close()
	report, err := repair(db, *playerID, *apply, time.Now().UnixMilli())
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
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
