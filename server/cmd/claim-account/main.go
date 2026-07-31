package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"

	"lunar-tear/server/internal/database"
)

var childTables = []string{
	"user_friend_requests",
	"user_friends",
	"user_cage_ornament_rewards",
	"user_shop_replaceable_lineup",
	"user_shop_items",
	"user_gacha_banner_box_drew_counts",
	"user_gacha_banners",
	"user_gacha_converted_medals",
	"user_gifts",
	"user_dokan_confirmed",
	"user_drawn_omikuji",
	"user_contents_stories",
	"user_viewed_movies",
	"user_navi_cutin_played",
	"user_auto_sale_settings",
	"user_explore_scores",
	"user_tutorials",
	"user_premium_items",
	"user_important_items",
	"user_materials",
	"user_consumable_items",
	"user_gimmick_unlocks",
	"user_gimmick_sequences",
	"user_gimmick_ornament_progress",
	"user_gimmick_progress",
	"user_big_hunt_weekly_statuses",
	"user_big_hunt_weekly_max_scores",
	"user_big_hunt_schedule_max_scores",
	"user_big_hunt_statuses",
	"user_big_hunt_max_scores",
	"user_quest_limit_content_status",
	"user_side_story_quests",
	"user_main_quest_season_routes",
	"user_missions",
	"user_quest_missions",
	"user_quests",
	"user_deck_type_notes",
	"user_deck_parts",
	"user_deck_sub_weapons",
	"user_decks",
	"user_deck_characters",
	"user_parts_presets",
	"user_parts_group_notes",
	"user_parts",
	"user_thoughts",
	"user_companions",
	"user_weapon_notes",
	"user_weapon_stories",
	"user_weapon_awakens",
	"user_weapon_abilities",
	"user_weapon_skills",
	"user_weapons",
	"user_costume_awaken_status_ups",
	"user_costume_active_skills",
	"user_costumes",
	"user_character_rebirths",
	"user_character_board_status_ups",
	"user_character_board_abilities",
	"user_character_boards",
	"user_characters",
	"user_gacha",
	"user_shop_replaceable",
	"user_explore",
	"user_guerrilla_free_open",
	"user_portal_cage",
	"user_notification",
	"user_battle",
	"user_big_hunt_state",
	"user_side_story_active",
	"user_extra_quest",
	"user_event_quest",
	"user_main_quest",
	"user_login_bonus",
	"user_login",
	"user_profile",
	"user_gem",
	"user_status",
	"user_setting",
	"sessions",
}

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

	for _, t := range childTables {
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
