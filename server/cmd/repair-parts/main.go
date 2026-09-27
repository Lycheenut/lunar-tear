package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"
	_ "modernc.org/sqlite"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil && !errors.Is(err, flag.ErrHelp) {
		log.Fatal(err)
	}
}

func run(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("repair-parts", flag.ContinueOnError)
	flags.SetOutput(out)
	dbPath := flags.String("db", "db/game.db", "SQLite database path (must already exist)")
	masterPath := flags.String("master", "assets/release/20240404193219.bin.e", "Master data path")
	userID := flags.Int64("user-id", 0, "Only repair this player; 0 selects all players")
	apply := flags.Bool("apply", false, "Write repairs; defaults to a read-only preview. Stop the server and back up the database first")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *userID < 0 || flags.NArg() != 0 {
		return fmt.Errorf("user-id must be nonnegative and positional arguments are not supported")
	}
	if err := memorydb.Init(*masterPath); err != nil {
		return err
	}
	catalog, err := masterdata.LoadPartsCatalog()
	if err != nil {
		return err
	}
	granter := questflow.BuildGranter(&masterdata.QuestCatalog{PartsCatalog: catalog}, nil)
	absPath, err := filepath.Abs(*dbPath)
	if err != nil {
		return err
	}
	mode, action := "ro", "would-repair"
	if *apply {
		mode, action = "rw", "repair"
	}
	dsn := url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}
	if !strings.HasPrefix(dsn.Path, "/") {
		dsn.Path = "/" + dsn.Path // file:///C:/... on Windows, not file://C:/...
	}
	dsn.RawQuery = url.Values{"mode": {mode}, "_pragma": {"busy_timeout(5000)"}}.Encode()
	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT p.user_id, p.user_parts_uuid, p.parts_id, p.level, p.parts_status_main_id
		FROM user_parts p WHERE (? = 0 OR p.user_id = ?) AND NOT EXISTS (
			SELECT 1 FROM user_parts_status_subs s WHERE s.user_id = p.user_id AND s.user_parts_uuid = p.user_parts_uuid
		) ORDER BY p.user_id, p.user_parts_uuid`, *userID, *userID)
	if err != nil {
		return err
	}
	type candidate struct {
		userID int64
		part   store.PartsState
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.userID, &c.part.UserPartsUuid, &c.part.PartsId, &c.part.Level, &c.part.PartsStatusMainId); err != nil {
			rows.Close()
			return err
		}
		candidates = append(candidates, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	nowMillis := time.Now().UnixMilli()
	repaired, skipped := 0, 0
	for _, c := range candidates {
		user := store.UserState{
			Parts:           map[string]store.PartsState{c.part.UserPartsUuid: c.part},
			PartsStatusSubs: make(map[store.PartsStatusSubKey]store.PartsStatusSubState),
		}
		_, mainOK := catalog.PartsStatusMainById[c.part.PartsStatusMainId]
		if !mainOK || granter.RepairPartsWithoutSubStatuses(&user, nowMillis) == 0 {
			skipped++
			fmt.Fprintf(out, "skip user=%d uuid=%s parts_id=%d: missing definitions or fewer than four valid sub-statuses\n", c.userID, c.part.UserPartsUuid, c.part.PartsId)
			continue
		}
		if *apply {
			for slot := int32(1); slot <= model.PartsMaxSubStatusCount; slot++ {
				sub := user.PartsStatusSubs[store.PartsStatusSubKey{UserPartsUuid: c.part.UserPartsUuid, StatusIndex: slot}]
				_, err := tx.Exec(`INSERT INTO user_parts_status_subs
				(user_id, user_parts_uuid, status_index, parts_status_sub_lottery_id, level,
				 status_kind_type, status_calculation_type, status_change_value, latest_version)
				VALUES (?,?,?,?,?,?,?,?,?)`, c.userID, sub.UserPartsUuid, sub.StatusIndex,
					sub.PartsStatusSubLotteryId, sub.Level, sub.StatusKindType, sub.StatusCalculationType, sub.StatusChangeValue, sub.LatestVersion)
				if err != nil {
					return fmt.Errorf("repair user=%d uuid=%s (transaction rolled back): %w", c.userID, c.part.UserPartsUuid, err)
				}
			}
		}
		repaired++
		fmt.Fprintf(out, "%s user=%d uuid=%s parts_id=%d level=%d: total-stats 1 -> 5\n", action, c.userID, c.part.UserPartsUuid, c.part.PartsId, c.part.Level)
	}
	if *apply {
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "apply=%t candidates=%d repaired=%d skipped=%d\n", *apply, len(candidates), repaired, skipped)
	return nil
}
