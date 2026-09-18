package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
)

const (
	repairID = "20260918-missions-3708-3711-login-bonus-1"
	// Verified against the Japanese text asset: material.name.315002.
	stoneID = 315002 // 真暗ノ天命石
)

type stampReward struct {
	Page, Stamp int32
	Type, ID    int32
	Count       int64
}

// Expand lower-page templates just as LoginBonusCatalog.LookupStampReward does.
// This also supports templates with different numbers of stamps per page.
func makeSchedule(bonuses []masterdata.EntityMLoginBonus, stamps []masterdata.EntityMLoginBonusStamp) ([]stampReward, error) {
	var totalPages int32
	for _, bonus := range bonuses {
		if bonus.LoginBonusId == 1 {
			totalPages = bonus.TotalPageCount
		}
	}
	if totalPages <= 0 {
		return nil, fmt.Errorf("login bonus 1 needs a finite positive page count")
	}
	groups := make(map[int32][]masterdata.EntityMLoginBonusStamp)
	for _, stamp := range stamps {
		if stamp.LoginBonusId != 1 {
			continue
		}
		if stamp.LowerPageNumber < 1 || stamp.LowerPageNumber > totalPages || stamp.RewardCount <= 0 {
			return nil, fmt.Errorf("invalid login bonus 1 stamp: %+v", stamp)
		}
		switch model.PossessionType(stamp.RewardPossessionType) {
		case model.PossessionTypeMaterial, model.PossessionTypeConsumableItem, model.PossessionTypeFreeGem:
		default:
			return nil, fmt.Errorf("unsupported login reward type %d; refusing to skip a reward", stamp.RewardPossessionType)
		}
		groups[stamp.LowerPageNumber] = append(groups[stamp.LowerPageNumber], stamp)
	}
	for page, group := range groups {
		sort.Slice(group, func(i, j int) bool { return group[i].StampNumber < group[j].StampNumber })
		for i, stamp := range group {
			if stamp.StampNumber != int32(i+1) {
				return nil, fmt.Errorf("login bonus 1 template page %d has missing or duplicate stamps", page)
			}
		}
	}
	var schedule []stampReward
	var group []masterdata.EntityMLoginBonusStamp
	for page := int32(1); page <= totalPages; page++ {
		if next, ok := groups[page]; ok {
			group = next
		}
		if len(group) == 0 {
			return nil, fmt.Errorf("login bonus 1 is missing page %d rewards", page)
		}
		for _, stamp := range group {
			schedule = append(schedule, stampReward{page, stamp.StampNumber, stamp.RewardPossessionType, stamp.RewardPossessionId, int64(stamp.RewardCount)})
		}
	}
	return schedule, nil
}

type inventoryChange struct {
	Type, ID             int32
	Before, Delta, After int64
}

type playerChange struct {
	UserID, PlayerID             int64
	DeductStones                 bool
	ResetMissions                []int32
	TotalLoginDays, ReceivedDays int64
	FromPage, FromStamp          int32
	ToPage, ToStamp              int32
	GrantDays                    int64
	ReceiveDatetime              int64
	Inventory                    []inventoryChange
	lastLogin, previousReceive   int64
}

type repairReport struct {
	RepairID       string
	Applied        bool
	AlreadyApplied bool
	ScannedPlayers int
	StonePlayers   int
	ResetMissions  int
	LoginPlayers   int
	GrantedDays    int64
	AheadPlayers   []int64
	Players        []playerChange
}

func repair(db *sql.DB, schedule []stampReward, apply bool, now int64) (repairReport, error) {
	report := repairReport{RepairID: repairID}
	tx, err := db.Begin()
	if err != nil {
		return report, err
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='player_data_repairs'`).Scan(&exists); err != nil {
		return report, err
	}
	if exists != 0 {
		if err := tx.QueryRow(`SELECT count(*) FROM player_data_repairs WHERE repair_id=?`, repairID).Scan(&exists); err != nil {
			return report, err
		}
		if exists != 0 {
			report.AlreadyApplied = true
			return report, nil
		}
	}
	players, err := readPlayers(tx)
	if err != nil {
		return report, err
	}
	report.ScannedPlayers = len(players)
	for _, player := range players {
		if err := planPlayer(tx, schedule, &player); err != nil {
			return report, fmt.Errorf("user %d: %w", player.UserID, err)
		}
		if player.ReceivedDays > player.TotalLoginDays {
			report.AheadPlayers = append(report.AheadPlayers, player.UserID)
		}
		if player.DeductStones {
			report.StonePlayers++
		}
		report.ResetMissions += len(player.ResetMissions)
		if player.GrantDays > 0 {
			report.LoginPlayers++
			report.GrantedDays += player.GrantDays
		}
		if player.DeductStones || len(player.ResetMissions) > 0 || player.GrantDays > 0 {
			report.Players = append(report.Players, player)
		}
	}
	if !apply {
		return report, nil
	}
	for _, player := range report.Players {
		if err := applyPlayer(tx, player, now); err != nil {
			return report, fmt.Errorf("user %d: %w", player.UserID, err)
		}
	}
	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS player_data_repairs (
		repair_id TEXT PRIMARY KEY, applied_at INTEGER NOT NULL, report_json TEXT NOT NULL
	)`); err != nil {
		return report, err
	}
	report.Applied = true
	audit, err := json.Marshal(report)
	if err != nil {
		return report, err
	}
	if _, err := tx.Exec(`INSERT INTO player_data_repairs VALUES (?,?,?)`, repairID, now, string(audit)); err != nil {
		return report, err
	}
	return report, tx.Commit()
}

func readPlayers(tx *sql.Tx) ([]playerChange, error) {
	rows, err := tx.Query(`SELECT u.user_id, u.player_id, COALESCE(l.total_login_count,0),
		COALESCE(l.last_login_datetime,0), COALESCE(b.current_page_number,0),
		COALESCE(b.current_stamp_number,0), COALESCE(b.latest_reward_receive_datetime,0)
		FROM users u LEFT JOIN user_login l ON l.user_id=u.user_id
		LEFT JOIN user_login_bonus b ON b.user_id=u.user_id AND b.login_bonus_id=1
		ORDER BY u.user_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var players []playerChange
	for rows.Next() {
		var p playerChange
		if err := rows.Scan(&p.UserID, &p.PlayerID, &p.TotalLoginDays, &p.lastLogin, &p.FromPage, &p.FromStamp, &p.previousReceive); err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, rows.Err()
}

func planPlayer(tx *sql.Tx, schedule []stampReward, p *playerChange) error {
	p.ToPage, p.ToStamp = p.FromPage, p.FromStamp
	p.ReceiveDatetime = p.previousReceive
	rows, err := tx.Query(`SELECT mission_id, mission_progress_status_type FROM user_missions
		WHERE user_id=? AND mission_id BETWEEN 3708 AND 3711 ORDER BY mission_id`, p.UserID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int32
		var status model.MissionProgressStatusType
		if err := rows.Scan(&id, &status); err != nil {
			rows.Close()
			return err
		}
		if id == 3711 {
			p.DeductStones = status == model.MissionProgressStatusTypeClear || status == model.MissionProgressStatusTypeRewardReceived
		} else if status == model.MissionProgressStatusTypeRewardReceived {
			p.ResetMissions = append(p.ResetMissions, id)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if p.TotalLoginDays < 0 || p.TotalLoginDays > int64(len(schedule)) {
		return fmt.Errorf("total login days %d exceed the configured reward schedule", p.TotalLoginDays)
	}
	validPosition := p.FromStamp == 0 && (p.FromPage == 0 || p.FromPage == 1)
	for i, reward := range schedule {
		if reward.Page == p.FromPage && reward.Stamp == p.FromStamp {
			p.ReceivedDays, validPosition = int64(i+1), true
		}
		if reward.Page == p.FromPage && reward.Stamp == 1 && p.FromStamp == 0 {
			p.ReceivedDays, validPosition = int64(i), true
		}
	}
	if !validPosition {
		return fmt.Errorf("invalid login bonus position page=%d stamp=%d", p.FromPage, p.FromStamp)
	}
	deltas := make(map[[2]int32]int64)
	if p.DeductStones {
		deltas[[2]int32{int32(model.PossessionTypeMaterial), stoneID}] = -630
	}
	if p.TotalLoginDays > p.ReceivedDays {
		if p.lastLogin <= 0 {
			return fmt.Errorf("cannot align rewards without a last login timestamp")
		}
		p.GrantDays = p.TotalLoginDays - p.ReceivedDays
		last := schedule[p.TotalLoginDays-1]
		p.ToPage, p.ToStamp = last.Page, last.Stamp
		// Preserve a newer stamp timestamp, but do not consume an absent player's
		// next login day by setting this to the maintenance execution time.
		p.ReceiveDatetime = max(p.lastLogin, p.previousReceive)
		for _, reward := range schedule[p.ReceivedDays:p.TotalLoginDays] {
			deltas[[2]int32{reward.Type, reward.ID}] += reward.Count
		}
	}
	for key, delta := range deltas {
		if delta == 0 {
			continue
		}
		change := inventoryChange{Type: key[0], ID: key[1], Delta: delta}
		table, idColumn, countColumn, err := inventoryTable(change.Type)
		if err != nil {
			return err
		}
		query := fmt.Sprintf("SELECT %s FROM %s WHERE user_id=?", countColumn, table)
		args := []any{p.UserID}
		if idColumn != "" {
			query += " AND " + idColumn + "=?"
			args = append(args, change.ID)
		}
		err = tx.QueryRow(query, args...).Scan(&change.Before)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		change.After = change.Before + change.Delta
		if change.After < math.MinInt32 || change.After > math.MaxInt32 {
			return fmt.Errorf("inventory type=%d id=%d would overflow int32", change.Type, change.ID)
		}
		p.Inventory = append(p.Inventory, change)
	}
	sort.Slice(p.Inventory, func(i, j int) bool {
		a, b := p.Inventory[i], p.Inventory[j]
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.ID < b.ID
	})
	return nil
}

func inventoryTable(possessionType int32) (table, idColumn, countColumn string, err error) {
	switch model.PossessionType(possessionType) {
	case model.PossessionTypeMaterial:
		return "user_materials", "material_id", "count", nil
	case model.PossessionTypeConsumableItem:
		return "user_consumable_items", "consumable_item_id", "count", nil
	case model.PossessionTypeFreeGem:
		return "user_gem", "", "free_gem", nil
	default:
		return "", "", "", fmt.Errorf("unsupported inventory type %d", possessionType)
	}
}

func applyPlayer(tx *sql.Tx, p playerChange, now int64) error {
	for _, change := range p.Inventory {
		table, idColumn, countColumn, err := inventoryTable(change.Type)
		if err != nil {
			return err
		}
		query := `INSERT INTO user_gem (user_id, free_gem) VALUES (?,?)
			ON CONFLICT(user_id) DO UPDATE SET free_gem=excluded.free_gem`
		args := []any{p.UserID, change.After}
		if idColumn != "" {
			query = fmt.Sprintf(`INSERT INTO %s (user_id,%s,%s) VALUES (?,?,?)
				ON CONFLICT(user_id,%s) DO UPDATE SET %s=excluded.%s`, table, idColumn, countColumn, idColumn, countColumn, countColumn)
			args = []any{p.UserID, change.ID, change.After}
		}
		if _, err := tx.Exec(query, args...); err != nil {
			return err
		}
	}
	for _, id := range p.ResetMissions {
		if _, err := tx.Exec(`UPDATE user_missions SET mission_progress_status_type=?,
			latest_version=MAX(latest_version+1,?) WHERE user_id=? AND mission_id=?`,
			model.MissionProgressStatusTypeClear, now, p.UserID, id); err != nil {
			return err
		}
	}
	if p.GrantDays > 0 {
		if _, err := tx.Exec(`INSERT INTO user_login_bonus
			(user_id,login_bonus_id,current_page_number,current_stamp_number,latest_reward_receive_datetime,latest_version)
			VALUES (?,1,?,?,?,?) ON CONFLICT(user_id,login_bonus_id) DO UPDATE SET
			current_page_number=excluded.current_page_number, current_stamp_number=excluded.current_stamp_number,
			latest_reward_receive_datetime=excluded.latest_reward_receive_datetime,
			latest_version=MAX(user_login_bonus.latest_version+1,excluded.latest_version)`,
			p.UserID, p.ToPage, p.ToStamp, p.ReceiveDatetime, now); err != nil {
			return err
		}
	}
	_, err := tx.Exec(`UPDATE users SET latest_version=MAX(latest_version+1,?) WHERE user_id=?`, now, p.UserID)
	return err
}
