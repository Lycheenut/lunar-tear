// repair-parts resets enhanced memoirs and refunds successful enhancement costs.
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

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"

	_ "modernc.org/sqlite"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("repair-parts", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dbPath := flags.String("db", "db/game.db", "Existing SQLite database (stop the game server before applying)")
	masterPath := flags.String("master-data", "assets/release/20240404193219.bin.e", "Master data used by the target game server")
	apply := flags.Bool("apply", false, "Commit the repair for all players; default is a read-only preview")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if err := memorydb.Init(*masterPath); err != nil {
		return err
	}
	catalog, err := masterdata.LoadPartsCatalog()
	if err != nil {
		return err
	}
	config, err := masterdata.LoadGameConfig()
	if err != nil {
		return err
	}
	db, err := openDatabase(*dbPath, *apply)
	if err != nil {
		return err
	}
	defer db.Close()
	report, err := repair(db, catalog, config.ConsumableItemIdForGold, *apply, time.Now().UnixMilli())
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
