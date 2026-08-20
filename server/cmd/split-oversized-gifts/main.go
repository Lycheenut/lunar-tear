package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/giftbox"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

const defaultMasterDataPath = "assets/release/20240404193219.bin.e"

type repairStats struct {
	OversizedRecords int
	CreatedRecords   int
	AffectedUsers    int
}

type giftRecord struct {
	userID                int64
	userGiftUUID          string
	possessionType        int32
	possessionID          int32
	count                 int32
	grantDatetime         int64
	descriptionGiftTextID int32
	equipmentData         []byte
	expirationDatetime    sql.NullInt64
}

func main() {
	dbPath := flag.String("db", "db/game.db", "SQLite database path")
	masterDataPath := flag.String("master-data", defaultMasterDataPath, "master data binary path")
	dryRun := flag.Bool("dry-run", false, "report changes without writing them")
	flag.Parse()

	if err := memorydb.Init(*masterDataPath); err != nil {
		log.Fatalf("load master data: %v", err)
	}
	config, err := masterdata.LoadGameConfig()
	if err != nil {
		log.Fatalf("load game config: %v", err)
	}
	db, err := database.Open(*dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	stats, err := repairOversizedGifts(context.Background(), db, config, *dryRun)
	if err != nil {
		log.Fatalf("split oversized gifts: %v", err)
	}
	if *dryRun {
		log.Printf("dry run: would split %d oversized gifts into %d additional records for %d users; no changes written",
			stats.OversizedRecords, stats.CreatedRecords, stats.AffectedUsers)
		return
	}
	log.Printf("split %d oversized gifts into %d additional records for %d users",
		stats.OversizedRecords, stats.CreatedRecords, stats.AffectedUsers)
}

func repairOversizedGifts(ctx context.Context, db *sql.DB, config *masterdata.GameConfig, dryRun bool) (repairStats, error) {
	if err := validateGiftLimits(config); err != nil {
		return repairStats{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return repairStats{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	records, err := loadStackableGiftRecords(ctx, tx)
	if err != nil {
		return repairStats{}, err
	}
	stats := repairStats{}
	affectedUsers := make(map[int64]struct{})
	for _, record := range records {
		gift := store.NotReceivedGiftState{
			GiftCommon: store.GiftCommonState{
				PossessionType:        record.possessionType,
				PossessionId:          record.possessionID,
				Count:                 record.count,
				GrantDatetime:         record.grantDatetime,
				DescriptionGiftTextId: record.descriptionGiftTextID,
				EquipmentData:         record.equipmentData,
			},
			UserGiftUuid: record.userGiftUUID,
		}
		if record.expirationDatetime.Valid {
			gift.ExpirationDatetime = record.expirationDatetime.Int64
		}
		parts := giftbox.SplitNotReceived(gift, config)
		if len(parts) == 1 {
			continue
		}

		stats.OversizedRecords++
		stats.CreatedRecords += len(parts) - 1
		affectedUsers[record.userID] = struct{}{}
		if dryRun {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE user_gifts SET count=? WHERE user_id=? AND user_gift_uuid=?`,
			parts[0].GiftCommon.Count, record.userID, record.userGiftUUID); err != nil {
			return repairStats{}, fmt.Errorf("update gift %s: %w", record.userGiftUUID, err)
		}
		for _, part := range parts[1:] {
			if _, err := tx.ExecContext(ctx, `INSERT INTO user_gifts
				(user_id, user_gift_uuid, is_received, possession_type, possession_id, count, grant_datetime,
				 description_gift_text_id, equipment_data, expiration_datetime)
				VALUES (?,?,0,?,?,?,?,?,?,?)`,
				record.userID, part.UserGiftUuid, record.possessionType, record.possessionID, part.GiftCommon.Count,
				record.grantDatetime, record.descriptionGiftTextID, record.equipmentData, record.expirationDatetime); err != nil {
				return repairStats{}, fmt.Errorf("insert split for gift %s: %w", record.userGiftUUID, err)
			}
		}
	}
	stats.AffectedUsers = len(affectedUsers)
	if dryRun {
		return stats, nil
	}
	for userID := range affectedUsers {
		if _, err := tx.ExecContext(ctx, `UPDATE user_notification
			SET gift_not_receive_count=(SELECT COUNT(*) FROM user_gifts WHERE user_id=? AND is_received=0)
			WHERE user_id=?`, userID, userID); err != nil {
			return repairStats{}, fmt.Errorf("update gift notification for user %d: %w", userID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return repairStats{}, fmt.Errorf("commit transaction: %w", err)
	}
	return stats, nil
}

func validateGiftLimits(config *masterdata.GameConfig) error {
	if config == nil {
		return fmt.Errorf("game config is nil")
	}
	if config.PossessionCountLimitMaterial <= 0 || config.PossessionCountLimitConsumableItem <= 0 || config.PossessionCountLimitMoney <= 0 {
		return fmt.Errorf("invalid gift limits: material=%d consumable=%d money=%d",
			config.PossessionCountLimitMaterial, config.PossessionCountLimitConsumableItem, config.PossessionCountLimitMoney)
	}
	return nil
}

func loadStackableGiftRecords(ctx context.Context, tx *sql.Tx) ([]giftRecord, error) {
	rows, err := tx.QueryContext(ctx, `SELECT user_id, user_gift_uuid, possession_type, possession_id, count,
		grant_datetime, description_gift_text_id, equipment_data, expiration_datetime
		FROM user_gifts
		WHERE is_received=0 AND possession_type IN (?,?) AND count>0
		ORDER BY user_id, user_gift_uuid`, model.PossessionTypeMaterial, model.PossessionTypeConsumableItem)
	if err != nil {
		return nil, fmt.Errorf("query gifts: %w", err)
	}
	defer rows.Close()

	records := make([]giftRecord, 0)
	for rows.Next() {
		var record giftRecord
		if err := rows.Scan(&record.userID, &record.userGiftUUID, &record.possessionType, &record.possessionID, &record.count,
			&record.grantDatetime, &record.descriptionGiftTextID, &record.equipmentData, &record.expirationDatetime); err != nil {
			return nil, fmt.Errorf("scan gift: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gifts: %w", err)
	}
	return records, nil
}
