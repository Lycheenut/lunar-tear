package sqlite

import (
	"database/sql"

	"lunar-tear/server/internal/store"
)

func writeMechanismTables(tx *sql.Tx, uid int64, u *store.UserState) error {
	if err := upsertBattleMissionDetail(tx, uid, u.Battle.MissionDetail); err != nil {
		return err
	}
	for id, v := range u.QuestReplayFlowRewards {
		if _, err := tx.Exec(`INSERT INTO user_quest_replay_flow_rewards (user_id, quest_replay_flow_reward_group_id, reward_receive_datetime, latest_version) VALUES (?,?,?,?)`,
			uid, id, v.RewardReceiveDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for groupingId, v := range u.QuestSceneChoices {
		if _, err := tx.Exec(`INSERT INTO user_quest_scene_choices (user_id, quest_scene_choice_grouping_id, quest_scene_choice_effect_id, latest_version) VALUES (?,?,?,?)`,
			uid, groupingId, v.QuestSceneChoiceEffectId, v.LatestVersion); err != nil {
			return err
		}
	}
	for effectId, v := range u.QuestSceneChoiceHistory {
		if _, err := tx.Exec(`INSERT INTO user_quest_scene_choice_history (user_id, quest_scene_choice_effect_id, choice_datetime, latest_version) VALUES (?,?,?,?)`,
			uid, effectId, v.ChoiceDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for id, v := range u.EventQuestDailyRewards {
		if _, err := tx.Exec(`INSERT INTO user_event_quest_daily_rewards (user_id, event_quest_daily_group_id, reward_receive_datetime, latest_version) VALUES (?,?,?,?)`,
			uid, id, v.RewardReceiveDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for id, v := range u.MissionPassPoints {
		if _, err := tx.Exec(`INSERT INTO user_mission_pass_points (user_id, mission_pass_id, point, latest_version) VALUES (?,?,?,?)`,
			uid, id, v.Point, v.LatestVersion); err != nil {
			return err
		}
	}
	for key, v := range u.MissionPassRewards {
		if _, err := tx.Exec(`INSERT INTO user_mission_pass_rewards (user_id, mission_pass_id, level, is_premium, reward_receive_datetime, latest_version) VALUES (?,?,?,?,?,?)`,
			uid, key.MissionPassId, key.Level, boolToInt(key.IsPremium), v.RewardReceiveDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for id, v := range u.MissionPassRemaining {
		if _, err := tx.Exec(`INSERT INTO user_mission_pass_remaining (user_id, mission_pass_id, reward_received, reward_receive_datetime, latest_version) VALUES (?,?,?,?,?)`,
			uid, id, boolToInt(v.RewardReceived), v.RewardReceiveDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for id, v := range u.WebviewPanelMissions {
		if _, err := tx.Exec(`INSERT INTO user_webview_panel_missions (user_id, webview_panel_mission_page_id, reward_receive_datetime, latest_version) VALUES (?,?,?,?)`,
			uid, id, v.RewardReceiveDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	return nil
}

func upsertBattleMissionDetail(tx *sql.Tx, uid int64, v store.BattleMissionDetailState) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO user_battle_mission_details (
		user_id, is_valid, character_death_count, max_damage, costume_skill_use_count,
		weapon_skill_use_count, companion_skill_use_count, critical_count, combo_count,
		combo_max_damage, total_recover_point, costume_result_count,
		costume_1_is_alive, costume_1_max_hp, costume_1_remaining_hp,
		costume_2_is_alive, costume_2_max_hp, costume_2_remaining_hp,
		costume_3_is_alive, costume_3_max_hp, costume_3_remaining_hp
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		uid, boolToInt(v.IsValid), v.CharacterDeathCount, v.MaxDamage, v.CostumeSkillUseCount,
		v.WeaponSkillUseCount, v.CompanionSkillUseCount, v.CriticalCount, v.ComboCount,
		v.ComboMaxDamage, v.TotalRecoverPoint, v.CostumeResultCount,
		boolToInt(v.CostumeResults[0].IsAlive), v.CostumeResults[0].MaxHp, v.CostumeResults[0].RemainingHp,
		boolToInt(v.CostumeResults[1].IsAlive), v.CostumeResults[1].MaxHp, v.CostumeResults[1].RemainingHp,
		boolToInt(v.CostumeResults[2].IsAlive), v.CostumeResults[2].MaxHp, v.CostumeResults[2].RemainingHp)
	return err
}

func loadMechanismTables(db *sql.DB, uid int64, u *store.UserState) {
	var valid, alive1, alive2, alive3 int
	_ = db.QueryRow(`SELECT is_valid, character_death_count, max_damage, costume_skill_use_count,
		weapon_skill_use_count, companion_skill_use_count, critical_count, combo_count,
		combo_max_damage, total_recover_point, costume_result_count,
		costume_1_is_alive, costume_1_max_hp, costume_1_remaining_hp,
		costume_2_is_alive, costume_2_max_hp, costume_2_remaining_hp,
		costume_3_is_alive, costume_3_max_hp, costume_3_remaining_hp
		FROM user_battle_mission_details WHERE user_id=?`, uid).Scan(
		&valid, &u.Battle.MissionDetail.CharacterDeathCount, &u.Battle.MissionDetail.MaxDamage,
		&u.Battle.MissionDetail.CostumeSkillUseCount, &u.Battle.MissionDetail.WeaponSkillUseCount,
		&u.Battle.MissionDetail.CompanionSkillUseCount, &u.Battle.MissionDetail.CriticalCount,
		&u.Battle.MissionDetail.ComboCount, &u.Battle.MissionDetail.ComboMaxDamage,
		&u.Battle.MissionDetail.TotalRecoverPoint, &u.Battle.MissionDetail.CostumeResultCount,
		&alive1, &u.Battle.MissionDetail.CostumeResults[0].MaxHp, &u.Battle.MissionDetail.CostumeResults[0].RemainingHp,
		&alive2, &u.Battle.MissionDetail.CostumeResults[1].MaxHp, &u.Battle.MissionDetail.CostumeResults[1].RemainingHp,
		&alive3, &u.Battle.MissionDetail.CostumeResults[2].MaxHp, &u.Battle.MissionDetail.CostumeResults[2].RemainingHp)
	u.Battle.MissionDetail.IsValid = valid != 0
	u.Battle.MissionDetail.CostumeResults[0].IsAlive = alive1 != 0
	u.Battle.MissionDetail.CostumeResults[1].IsAlive = alive2 != 0
	u.Battle.MissionDetail.CostumeResults[2].IsAlive = alive3 != 0

	queryRows(db, `SELECT quest_replay_flow_reward_group_id, reward_receive_datetime, latest_version FROM user_quest_replay_flow_rewards WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.QuestReplayFlowRewardState
		_ = rows.Scan(&v.QuestReplayFlowRewardGroupId, &v.RewardReceiveDatetime, &v.LatestVersion)
		u.QuestReplayFlowRewards[v.QuestReplayFlowRewardGroupId] = v
	})
	queryRows(db, `SELECT quest_scene_choice_grouping_id, quest_scene_choice_effect_id, latest_version FROM user_quest_scene_choices WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.QuestSceneChoiceState
		_ = rows.Scan(&v.QuestSceneChoiceGroupingId, &v.QuestSceneChoiceEffectId, &v.LatestVersion)
		u.QuestSceneChoices[v.QuestSceneChoiceGroupingId] = v
	})
	queryRows(db, `SELECT quest_scene_choice_effect_id, choice_datetime, latest_version FROM user_quest_scene_choice_history WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.QuestSceneChoiceHistoryState
		_ = rows.Scan(&v.QuestSceneChoiceEffectId, &v.ChoiceDatetime, &v.LatestVersion)
		u.QuestSceneChoiceHistory[v.QuestSceneChoiceEffectId] = v
	})
	queryRows(db, `SELECT event_quest_daily_group_id, reward_receive_datetime, latest_version FROM user_event_quest_daily_rewards WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.EventQuestDailyRewardState
		_ = rows.Scan(&v.EventQuestDailyGroupId, &v.RewardReceiveDatetime, &v.LatestVersion)
		u.EventQuestDailyRewards[v.EventQuestDailyGroupId] = v
	})
	queryRows(db, `SELECT mission_pass_id, point, latest_version FROM user_mission_pass_points WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.MissionPassPointState
		_ = rows.Scan(&v.MissionPassId, &v.Point, &v.LatestVersion)
		u.MissionPassPoints[v.MissionPassId] = v
	})
	queryRows(db, `SELECT mission_pass_id, level, is_premium, reward_receive_datetime, latest_version FROM user_mission_pass_rewards WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.MissionPassRewardState
		var premium int
		_ = rows.Scan(&v.MissionPassId, &v.Level, &premium, &v.RewardReceiveDatetime, &v.LatestVersion)
		v.IsPremium = premium != 0
		u.MissionPassRewards[store.MissionPassRewardKey{MissionPassId: v.MissionPassId, Level: v.Level, IsPremium: v.IsPremium}] = v
	})
	queryRows(db, `SELECT mission_pass_id, reward_received, reward_receive_datetime, latest_version FROM user_mission_pass_remaining WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.MissionPassRemainingState
		var received int
		_ = rows.Scan(&v.MissionPassId, &received, &v.RewardReceiveDatetime, &v.LatestVersion)
		v.RewardReceived = received != 0
		u.MissionPassRemaining[v.MissionPassId] = v
	})
	queryRows(db, `SELECT webview_panel_mission_page_id, reward_receive_datetime, latest_version FROM user_webview_panel_missions WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.WebviewPanelMissionState
		_ = rows.Scan(&v.WebviewPanelMissionPageId, &v.RewardReceiveDatetime, &v.LatestVersion)
		u.WebviewPanelMissions[v.WebviewPanelMissionPageId] = v
	})
}

func diffMechanismTables(tx *sql.Tx, uid int64, before, after *store.UserState) error {
	if before.Battle.MissionDetail != after.Battle.MissionDetail {
		if err := upsertBattleMissionDetail(tx, uid, after.Battle.MissionDetail); err != nil {
			return err
		}
	}
	if err := diffMapInt32(tx, uid, before.QuestReplayFlowRewards, after.QuestReplayFlowRewards,
		"user_quest_replay_flow_rewards", "quest_replay_flow_reward_group_id",
		func(v store.QuestReplayFlowRewardState) []any {
			return []any{v.QuestReplayFlowRewardGroupId, v.RewardReceiveDatetime, v.LatestVersion}
		},
		"quest_replay_flow_reward_group_id, reward_receive_datetime, latest_version"); err != nil {
		return err
	}
	if err := diffQuestSceneChoices(tx, uid, before.QuestSceneChoices, after.QuestSceneChoices); err != nil {
		return err
	}
	if err := diffQuestSceneChoiceHistory(tx, uid, before.QuestSceneChoiceHistory, after.QuestSceneChoiceHistory); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.EventQuestDailyRewards, after.EventQuestDailyRewards,
		"user_event_quest_daily_rewards", "event_quest_daily_group_id",
		func(v store.EventQuestDailyRewardState) []any {
			return []any{v.EventQuestDailyGroupId, v.RewardReceiveDatetime, v.LatestVersion}
		},
		"event_quest_daily_group_id, reward_receive_datetime, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.MissionPassPoints, after.MissionPassPoints,
		"user_mission_pass_points", "mission_pass_id",
		func(v store.MissionPassPointState) []any { return []any{v.MissionPassId, v.Point, v.LatestVersion} },
		"mission_pass_id, point, latest_version"); err != nil {
		return err
	}
	if err := diffMissionPassRewards(tx, uid, before.MissionPassRewards, after.MissionPassRewards); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.MissionPassRemaining, after.MissionPassRemaining,
		"user_mission_pass_remaining", "mission_pass_id",
		func(v store.MissionPassRemainingState) []any {
			return []any{v.MissionPassId, boolToInt(v.RewardReceived), v.RewardReceiveDatetime, v.LatestVersion}
		},
		"mission_pass_id, reward_received, reward_receive_datetime, latest_version"); err != nil {
		return err
	}
	return diffMapInt32(tx, uid, before.WebviewPanelMissions, after.WebviewPanelMissions,
		"user_webview_panel_missions", "webview_panel_mission_page_id",
		func(v store.WebviewPanelMissionState) []any {
			return []any{v.WebviewPanelMissionPageId, v.RewardReceiveDatetime, v.LatestVersion}
		},
		"webview_panel_mission_page_id, reward_receive_datetime, latest_version")
}

func diffQuestSceneChoices(tx *sql.Tx, uid int64, before, after map[int32]store.QuestSceneChoiceState) error {
	for groupingId, v := range after {
		if old, ok := before[groupingId]; ok && old == v {
			continue
		}
		if _, err := tx.Exec(`INSERT OR REPLACE INTO user_quest_scene_choices (user_id, quest_scene_choice_grouping_id, quest_scene_choice_effect_id, latest_version) VALUES (?,?,?,?)`,
			uid, groupingId, v.QuestSceneChoiceEffectId, v.LatestVersion); err != nil {
			return err
		}
	}
	for groupingId := range before {
		if _, ok := after[groupingId]; !ok {
			if _, err := tx.Exec(`DELETE FROM user_quest_scene_choices WHERE user_id=? AND quest_scene_choice_grouping_id=?`, uid, groupingId); err != nil {
				return err
			}
		}
	}
	return nil
}

func diffQuestSceneChoiceHistory(tx *sql.Tx, uid int64, before, after map[int32]store.QuestSceneChoiceHistoryState) error {
	for effectId, v := range after {
		if old, ok := before[effectId]; ok && old == v {
			continue
		}
		if _, err := tx.Exec(`INSERT OR REPLACE INTO user_quest_scene_choice_history (user_id, quest_scene_choice_effect_id, choice_datetime, latest_version) VALUES (?,?,?,?)`,
			uid, effectId, v.ChoiceDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for effectId := range before {
		if _, ok := after[effectId]; !ok {
			if _, err := tx.Exec(`DELETE FROM user_quest_scene_choice_history WHERE user_id=? AND quest_scene_choice_effect_id=?`, uid, effectId); err != nil {
				return err
			}
		}
	}
	return nil
}

func diffMissionPassRewards(tx *sql.Tx, uid int64, before, after map[store.MissionPassRewardKey]store.MissionPassRewardState) error {
	for key, v := range after {
		if old, ok := before[key]; ok && old == v {
			continue
		}
		if _, err := tx.Exec(`INSERT OR REPLACE INTO user_mission_pass_rewards (user_id, mission_pass_id, level, is_premium, reward_receive_datetime, latest_version) VALUES (?,?,?,?,?,?)`,
			uid, key.MissionPassId, key.Level, boolToInt(key.IsPremium), v.RewardReceiveDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for key := range before {
		if _, ok := after[key]; !ok {
			if _, err := tx.Exec(`DELETE FROM user_mission_pass_rewards WHERE user_id=? AND mission_pass_id=? AND level=? AND is_premium=?`, uid, key.MissionPassId, key.Level, boolToInt(key.IsPremium)); err != nil {
				return err
			}
		}
	}
	return nil
}
