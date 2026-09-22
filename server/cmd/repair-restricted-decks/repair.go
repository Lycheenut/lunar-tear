package main

import (
	"database/sql"
	"fmt"
	"sort"

	"lunar-tear/server/internal/model"
)

type deckChange struct {
	DeckType             model.DeckType
	UserDeckNumber       int32
	Name                 string
	PowerBefore          int32
	CharacterUUIDsBefore [3]string
}

type repairReport struct {
	Applied                  bool
	UserID, PlayerID         int64
	Decks                    []deckChange
	RemovedDeckCharacters    int64
	RemovedSubWeapons        int64
	RemovedParts             int64
	SharedDeckCharactersKept []string
}

func repair(db *sql.DB, playerID int64, apply bool, now int64) (repairReport, error) {
	report := repairReport{PlayerID: playerID, Decks: []deckChange{}, SharedDeckCharactersKept: []string{}}
	if playerID <= 0 {
		return report, fmt.Errorf("--player-id must be a positive integer")
	}
	tx, err := db.Begin()
	if err != nil {
		return report, err
	}
	defer tx.Rollback()

	// Imported accounts can have duplicate player IDs. Never choose one silently.
	var matches int
	if err := tx.QueryRow(`SELECT count(*), coalesce(min(user_id), 0) FROM users WHERE player_id = ?`, playerID).
		Scan(&matches, &report.UserID); err != nil {
		return report, fmt.Errorf("find player %d: %w", playerID, err)
	}
	if matches != 1 {
		return report, fmt.Errorf("player ID %d matches %d users; expected exactly one", playerID, matches)
	}

	rows, err := tx.Query(`SELECT deck_type, user_deck_number, name, power,
		user_deck_character_uuid01, user_deck_character_uuid02, user_deck_character_uuid03
		FROM user_decks WHERE user_id = ? AND deck_type IN (?, ?)
		AND (user_deck_character_uuid01 != '' OR user_deck_character_uuid02 != ''
			OR user_deck_character_uuid03 != '' OR power != 0)
		ORDER BY deck_type, user_deck_number`, report.UserID,
		model.DeckTypeRestrictedQuest, model.DeckTypeRestrictedLimitContentQuest)
	if err != nil {
		return report, fmt.Errorf("read restricted decks: %w", err)
	}
	defer rows.Close()
	characterSet := make(map[string]bool)
	for rows.Next() {
		var deck deckChange
		if err := rows.Scan(&deck.DeckType, &deck.UserDeckNumber, &deck.Name, &deck.PowerBefore,
			&deck.CharacterUUIDsBefore[0], &deck.CharacterUUIDsBefore[1], &deck.CharacterUUIDsBefore[2]); err != nil {
			return report, err
		}
		report.Decks = append(report.Decks, deck)
		for _, uuid := range deck.CharacterUUIDsBefore {
			if uuid != "" {
				characterSet[uuid] = true
			}
		}
	}
	if err := rows.Err(); err != nil {
		return report, err
	}
	if err := rows.Close(); err != nil {
		return report, err
	}

	characters := make([]string, 0, len(characterSet))
	for uuid := range characterSet {
		characters = append(characters, uuid)
	}
	sort.Strings(characters)
	for _, uuid := range characters {
		// A damaged save may share a deck-character row with an ordinary deck.
		// Detach the restricted slots but preserve that other deck's equipment.
		var shared bool
		if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM user_decks
			WHERE user_id = ? AND deck_type NOT IN (?, ?)
			AND ? IN (user_deck_character_uuid01, user_deck_character_uuid02, user_deck_character_uuid03))`,
			report.UserID, model.DeckTypeRestrictedQuest, model.DeckTypeRestrictedLimitContentQuest, uuid).
			Scan(&shared); err != nil {
			return report, err
		}
		if shared {
			report.SharedDeckCharactersKept = append(report.SharedDeckCharactersKept, uuid)
			continue
		}
		// Removing the deck-character row unequips its costume, main weapon,
		// companion, Thought (Debris), and dressup costume. Parts are memoirs.
		for _, table := range []struct {
			name  string
			count *int64
		}{
			{"user_deck_sub_weapons", &report.RemovedSubWeapons},
			{"user_deck_parts", &report.RemovedParts},
			{"user_deck_characters", &report.RemovedDeckCharacters},
		} {
			where := ` FROM ` + table.name + ` WHERE user_id = ? AND user_deck_character_uuid = ?`
			var count int64
			if err := tx.QueryRow(`SELECT count(*)`+where, report.UserID, uuid).Scan(&count); err != nil {
				return report, fmt.Errorf("count %s: %w", table.name, err)
			}
			*table.count += count
			if apply && count > 0 {
				if _, err := tx.Exec(`DELETE`+where, report.UserID, uuid); err != nil {
					return report, fmt.Errorf("clear %s: %w", table.name, err)
				}
			}
		}
	}
	if !apply {
		return report, nil
	}
	for _, deck := range report.Decks {
		if _, err := tx.Exec(`UPDATE user_decks SET user_deck_character_uuid01 = '',
			user_deck_character_uuid02 = '', user_deck_character_uuid03 = '', power = 0, latest_version = ?
			WHERE user_id = ? AND deck_type = ? AND user_deck_number = ?`,
			now, report.UserID, deck.DeckType, deck.UserDeckNumber); err != nil {
			return report, fmt.Errorf("clear deck %d/%d: %w", deck.DeckType, deck.UserDeckNumber, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return report, err
	}
	report.Applied = true
	return report, nil
}
