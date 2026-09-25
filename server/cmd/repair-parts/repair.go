package main

import (
	"database/sql"
	"fmt"
	"math"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
)

type subStatusChange struct {
	StatusIndex, PartsStatusSubLotteryID  int32
	StatusKindType, StatusCalculationType int32
	LevelBefore, LevelAfter               int32
	ValueBefore, ValueAfter               int32
}

type partChange struct {
	UUID                    string
	PartsID                 int32
	LevelBefore, LevelAfter int32
	GoldRefund              int64
	SubStatuses             []subStatusChange
}

type playerChange struct {
	UserID, PlayerID                  int64
	GoldBefore, GoldRefund, GoldAfter int64
	Parts                             []partChange
}

type repairReport struct {
	Applied        bool
	ScannedPlayers int
	ResetParts     int
	GoldItemID     int32
	GoldRefund     int64
	Players        []playerChange
}

func repair(db *sql.DB, catalog *masterdata.PartsCatalog, goldItemID int32, apply bool, now int64) (repairReport, error) {
	report := repairReport{GoldItemID: goldItemID, Players: []playerChange{}}
	if goldItemID <= 0 {
		return report, fmt.Errorf("master data is missing a valid CONSUMABLE_ITEM_ID_FOR_GOLD")
	}
	tx, err := db.Begin()
	if err != nil {
		return report, err
	}
	defer tx.Rollback()
	if err := tx.QueryRow(`SELECT count(*) FROM users`).Scan(&report.ScannedPlayers); err != nil {
		return report, err
	}
	report.Players, err = readEnhancedParts(tx, goldItemID)
	if err != nil {
		return report, err
	}
	for i := range report.Players {
		player := &report.Players[i]
		for j := range player.Parts {
			part := &player.Parts[j]
			part.GoldRefund, err = enhancementRefund(catalog, part.PartsID, part.LevelBefore)
			if err != nil {
				return report, fmt.Errorf("user %d part %s: %w", player.UserID, part.UUID, err)
			}
			part.SubStatuses, err = rerollSubStatuses(tx, catalog, player.UserID, part.UUID)
			if err != nil {
				return report, fmt.Errorf("user %d part %s: %w", player.UserID, part.UUID, err)
			}
			player.GoldRefund += part.GoldRefund
			report.ResetParts++
		}
		// Inventory is int32 in the server even though SQLite stores int64.
		if player.GoldBefore < 0 || player.GoldBefore > math.MaxInt32-player.GoldRefund {
			return report, fmt.Errorf("user %d: gold refund %d would overflow balance %d", player.UserID, player.GoldRefund, player.GoldBefore)
		}
		player.GoldAfter = player.GoldBefore + player.GoldRefund
		report.GoldRefund += player.GoldRefund
		if apply {
			if err := applyPlayerRepair(tx, *player, goldItemID, now); err != nil {
				return report, fmt.Errorf("user %d: %w", player.UserID, err)
			}
		}
	}
	if !apply {
		return report, nil
	}
	if err := tx.Commit(); err != nil {
		return report, err
	}
	report.Applied = true
	return report, nil
}

func readEnhancedParts(tx *sql.Tx, goldItemID int32) ([]playerChange, error) {
	rows, err := tx.Query(`SELECT p.user_id, u.player_id, coalesce(g.count, 0),
		p.user_parts_uuid, p.parts_id, p.level
		FROM user_parts p JOIN users u ON u.user_id = p.user_id
		LEFT JOIN user_consumable_items g ON g.user_id = p.user_id AND g.consumable_item_id = ?
		WHERE p.level > 1 ORDER BY p.user_id, p.user_parts_uuid`, goldItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	players := []playerChange{}
	for rows.Next() {
		var userID, playerID, gold int64
		part := partChange{LevelAfter: 1}
		if err := rows.Scan(&userID, &playerID, &gold, &part.UUID, &part.PartsID, &part.LevelBefore); err != nil {
			return nil, err
		}
		if len(players) == 0 || players[len(players)-1].UserID != userID {
			players = append(players, playerChange{UserID: userID, PlayerID: playerID, GoldBefore: gold})
		}
		player := &players[len(players)-1]
		player.Parts = append(player.Parts, part)
	}
	return players, rows.Err()
}

func enhancementRefund(catalog *masterdata.PartsCatalog, partsID, level int32) (int64, error) {
	if level < 2 || level > model.PartsMaxLevel {
		return 0, fmt.Errorf("invalid enhanced level %d", level)
	}
	part, ok := catalog.PartsById[partsID]
	if !ok {
		return 0, fmt.Errorf("unknown parts ID %d", partsID)
	}
	rarity, ok := catalog.RarityByRarityType[part.RarityType]
	if !ok {
		return 0, fmt.Errorf("unknown rarity %d", part.RarityType)
	}
	prices := catalog.PriceByGroupAndLevel[rarity.PartsLevelUpPriceGroupId]
	var total int64
	// Enhance charges the price at the CURRENT level, then increments on success.
	// Count one successful attempt per level; failed attempts are not reimbursed.
	for current := int32(1); current < level; current++ {
		price, ok := prices[current]
		if !ok || price < 0 {
			return 0, fmt.Errorf("missing or invalid enhancement price for group %d level %d", rarity.PartsLevelUpPriceGroupId, current)
		}
		total += int64(price)
	}
	return total, nil
}

func rerollSubStatuses(tx *sql.Tx, catalog *masterdata.PartsCatalog, userID int64, uuid string) ([]subStatusChange, error) {
	rows, err := tx.Query(`SELECT status_index, parts_status_sub_lottery_id, status_kind_type,
		status_calculation_type, level, status_change_value FROM user_parts_status_subs
		WHERE user_id = ? AND user_parts_uuid = ? ORDER BY status_index`, userID, uuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	subs := []subStatusChange{}
	for rows.Next() {
		sub := subStatusChange{LevelAfter: 1}
		if err := rows.Scan(&sub.StatusIndex, &sub.PartsStatusSubLotteryID, &sub.StatusKindType,
			&sub.StatusCalculationType, &sub.LevelBefore, &sub.ValueBefore); err != nil {
			return nil, err
		}
		def, ok := catalog.PartsStatusSubById[sub.PartsStatusSubLotteryID]
		if !ok || def.StatusKindType != sub.StatusKindType || def.StatusCalculationType != sub.StatusCalculationType {
			return nil, fmt.Errorf("sub-status slot %d has unknown or mismatched definition %d", sub.StatusIndex, sub.PartsStatusSubLotteryID)
		}
		sub.ValueAfter = def.Initial.Roll()
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func applyPlayerRepair(tx *sql.Tx, player playerChange, goldItemID int32, now int64) error {
	for _, part := range player.Parts {
		if _, err := tx.Exec(`UPDATE user_parts SET level = 1, latest_version = ?
			WHERE user_id = ? AND user_parts_uuid = ?`, now, player.UserID, part.UUID); err != nil {
			return err
		}
		for _, sub := range part.SubStatuses {
			if _, err := tx.Exec(`UPDATE user_parts_status_subs SET level = 1, status_change_value = ?, latest_version = ?
				WHERE user_id = ? AND user_parts_uuid = ? AND status_index = ?`,
				sub.ValueAfter, now, player.UserID, part.UUID, sub.StatusIndex); err != nil {
				return err
			}
		}
	}
	if player.GoldRefund > 0 {
		if _, err := tx.Exec(`INSERT INTO user_consumable_items (user_id, consumable_item_id, count) VALUES (?, ?, ?)
			ON CONFLICT(user_id, consumable_item_id) DO UPDATE SET count = excluded.count`,
			player.UserID, goldItemID, player.GoldAfter); err != nil {
			return err
		}
	}
	return nil
}
