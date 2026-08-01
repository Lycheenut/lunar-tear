package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func marshalAutoOrbitDrops(drops []store.AutoOrbitDropEntry) string {
	if len(drops) == 0 {
		return "[]"
	}
	b, err := json.Marshal(drops)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Precondition: the users row must already exist.
func writeUserState(tx *sql.Tx, uid int64, u *store.UserState) error {
	exec := func(query string, args ...any) error {
		_, err := tx.Exec(query, args...)
		return err
	}
	if err := writeMechanismTables(tx, uid, u); err != nil {
		return err
	}

	if err := exec(`INSERT INTO user_setting (user_id, is_notify_purchase_alert, latest_version) VALUES (?,?,?)`,
		uid, boolToInt(u.Setting.IsNotifyPurchaseAlert), u.Setting.LatestVersion); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_status (user_id, level, exp, stamina_milli_value, stamina_update_datetime, latest_version) VALUES (?,?,?,?,?,?)`,
		uid, u.Status.Level, u.Status.Exp, u.Status.StaminaMilliValue, u.Status.StaminaUpdateDatetime, u.Status.LatestVersion); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_gem (user_id, paid_gem, free_gem) VALUES (?,?,?)`,
		uid, u.Gem.PaidGem, u.Gem.FreeGem); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_profile (user_id, name, name_update_datetime, message, message_update_datetime, favorite_costume_id, favorite_costume_id_update_datetime, current_pvp_rank, current_pvp_grade_id, max_pvp_season_rank, latest_version) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		uid, u.Profile.Name, u.Profile.NameUpdateDatetime, u.Profile.Message, u.Profile.MessageUpdateDatetime,
		u.Profile.FavoriteCostumeId, u.Profile.FavoriteCostumeIdUpdateDatetime,
		u.Profile.CurrentPvpRank, u.Profile.CurrentPvpGradeId, u.Profile.MaxPvpSeasonRank,
		u.Profile.LatestVersion); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_login (user_id, total_login_count, continual_login_count, max_continual_login_count, last_login_datetime, last_comeback_login_datetime, latest_version) VALUES (?,?,?,?,?,?,?)`,
		uid, u.Login.TotalLoginCount, u.Login.ContinualLoginCount, u.Login.MaxContinualLoginCount,
		u.Login.LastLoginDatetime, u.Login.LastComebackLoginDatetime, u.Login.LatestVersion); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_login_bonus (user_id, login_bonus_id, current_page_number, current_stamp_number, latest_reward_receive_datetime, latest_version) VALUES (?,?,?,?,?,?)`,
		uid, u.LoginBonus.LoginBonusId, u.LoginBonus.CurrentPageNumber, u.LoginBonus.CurrentStampNumber,
		u.LoginBonus.LatestRewardReceiveDatetime, u.LoginBonus.LatestVersion); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_main_quest (user_id, current_quest_flow_type, current_main_quest_route_id, current_quest_scene_id, head_quest_scene_id, is_reached_last_quest_scene, progress_quest_scene_id, progress_head_quest_scene_id, progress_quest_flow_type, main_quest_season_id, latest_version, saved_ctx_active, saved_ctx_current_quest_scene_id, saved_ctx_head_quest_scene_id, saved_ctx_current_main_quest_route_id, saved_ctx_main_quest_season_id, saved_ctx_is_reached_last_quest_scene, saved_ctx_portal_cage_in_progress, saved_ctx_current_quest_flow_type, replay_flow_current_quest_scene_id, replay_flow_head_quest_scene_id) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		uid, u.MainQuest.CurrentQuestFlowType, u.MainQuest.CurrentMainQuestRouteId, u.MainQuest.CurrentQuestSceneId,
		u.MainQuest.HeadQuestSceneId, boolToInt(u.MainQuest.IsReachedLastQuestScene), u.MainQuest.ProgressQuestSceneId,
		u.MainQuest.ProgressHeadQuestSceneId, u.MainQuest.ProgressQuestFlowType, u.MainQuest.MainQuestSeasonId,
		u.MainQuest.LatestVersion,
		boolToInt(u.MainQuest.SavedContext.Active),
		u.MainQuest.SavedContext.CurrentQuestSceneId, u.MainQuest.SavedContext.HeadQuestSceneId,
		u.MainQuest.SavedContext.CurrentMainQuestRouteId, u.MainQuest.SavedContext.MainQuestSeasonId,
		boolToInt(u.MainQuest.SavedContext.IsReachedLastQuestScene),
		boolToInt(u.MainQuest.SavedContext.PortalCageInProgress),
		u.MainQuest.SavedContext.CurrentQuestFlowType,
		u.MainQuest.ReplayFlowCurrentQuestSceneId, u.MainQuest.ReplayFlowHeadQuestSceneId); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_event_quest (user_id, current_event_quest_chapter_id, current_quest_id, current_quest_scene_id, head_quest_scene_id, latest_version) VALUES (?,?,?,?,?,?)`,
		uid, u.EventQuest.CurrentEventQuestChapterId, u.EventQuest.CurrentQuestId, u.EventQuest.CurrentQuestSceneId,
		u.EventQuest.HeadQuestSceneId, u.EventQuest.LatestVersion); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_extra_quest (user_id, current_quest_id, current_quest_scene_id, head_quest_scene_id, latest_version) VALUES (?,?,?,?,?)`,
		uid, u.ExtraQuest.CurrentQuestId, u.ExtraQuest.CurrentQuestSceneId, u.ExtraQuest.HeadQuestSceneId, u.ExtraQuest.LatestVersion); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_side_story_active (user_id, current_side_story_quest_id, current_side_story_quest_scene_id, latest_version) VALUES (?,?,?,?)`,
		uid, u.SideStoryActiveProgress.CurrentSideStoryQuestId, u.SideStoryActiveProgress.CurrentSideStoryQuestSceneId, u.SideStoryActiveProgress.LatestVersion); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_big_hunt_state (user_id, current_big_hunt_boss_quest_id, current_big_hunt_quest_id, current_quest_scene_id, is_dry_run, latest_version, deck_type, user_triple_deck_number, boss_knock_down_count, max_combo_count, total_damage, deck_number, battle_binary) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		uid, u.BigHuntProgress.CurrentBigHuntBossQuestId, u.BigHuntProgress.CurrentBigHuntQuestId,
		u.BigHuntProgress.CurrentQuestSceneId, boolToInt(u.BigHuntProgress.IsDryRun), u.BigHuntProgress.LatestVersion,
		u.BigHuntBattleDetail.DeckType, u.BigHuntBattleDetail.UserTripleDeckNumber, u.BigHuntBattleDetail.BossKnockDownCount,
		u.BigHuntBattleDetail.MaxComboCount, u.BigHuntBattleDetail.TotalDamage, u.BigHuntDeckNumber, u.BigHuntBattleBinary); err != nil {
		return err
	}
	for i, ci := range u.BigHuntBattleDetail.CostumeBattleInfo {
		if err := exec(`INSERT INTO user_big_hunt_costume_battle_infos (user_id, wave_index, sort_order, costume_id, total_damage, hit_count, random_display_value_type, random_display_value) VALUES (?,?,?,?,?,?,?,?)`,
			uid, ci.WaveIndex, i, ci.CostumeId, ci.TotalDamage, ci.HitCount, ci.RandomDisplayValueType, ci.RandomDisplayValue); err != nil {
			return err
		}
	}
	if err := exec(`INSERT INTO user_battle (user_id, is_active, start_count, finish_count, last_started_at, last_finished_at, last_user_party_count, last_npc_party_count, last_battle_binary_size, last_elapsed_frame_count) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		uid, boolToInt(u.Battle.IsActive), u.Battle.StartCount, u.Battle.FinishCount, u.Battle.LastStartedAt,
		u.Battle.LastFinishedAt, u.Battle.LastUserPartyCount, u.Battle.LastNpcPartyCount,
		u.Battle.LastBattleBinarySize, u.Battle.LastElapsedFrameCount); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_notification (user_id, gift_not_receive_count, friend_request_receive_count, is_exist_unread_information) VALUES (?,?,?,?)`,
		uid, u.Notifications.GiftNotReceiveCount, u.Notifications.FriendRequestReceiveCount,
		boolToInt(u.Notifications.IsExistUnreadInformation)); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_portal_cage (user_id, is_current_progress, drop_item_start_datetime, current_drop_item_count, latest_version) VALUES (?,?,?,?,?)`,
		uid, boolToInt(u.PortalCageStatus.IsCurrentProgress), u.PortalCageStatus.DropItemStartDatetime,
		u.PortalCageStatus.CurrentDropItemCount, u.PortalCageStatus.LatestVersion); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_guerrilla_free_open (user_id, start_datetime, open_minutes, daily_opened_count, latest_version) VALUES (?,?,?,?,?)`,
		uid, u.GuerrillaFreeOpen.StartDatetime, u.GuerrillaFreeOpen.OpenMinutes, u.GuerrillaFreeOpen.DailyOpenedCount, u.GuerrillaFreeOpen.LatestVersion); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_quest_auto_orbit (user_id, quest_type, chapter_id, quest_id, max_auto_orbit_count, cleared_auto_orbit_count, last_clear_datetime, latest_version, accumulated_drops_json) VALUES (?,?,?,?,?,?,?,?,?)`,
		uid, u.QuestAutoOrbit.QuestType, u.QuestAutoOrbit.ChapterId, u.QuestAutoOrbit.QuestId,
		u.QuestAutoOrbit.MaxAutoOrbitCount, u.QuestAutoOrbit.ClearedAutoOrbitCount,
		u.QuestAutoOrbit.LastClearDatetime, u.QuestAutoOrbit.LatestVersion,
		marshalAutoOrbitDrops(u.QuestAutoOrbit.AccumulatedDrops)); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_explore (user_id, is_use_explore_ticket, playing_explore_id, latest_play_datetime, latest_version) VALUES (?,?,?,?,?)`,
		uid, boolToInt(u.Explore.IsUseExploreTicket), u.Explore.PlayingExploreId, u.Explore.LatestPlayDatetime, u.Explore.LatestVersion); err != nil {
		return err
	}
	if err := exec(`INSERT INTO user_shop_replaceable (user_id, lineup_update_count, latest_lineup_update_datetime, latest_version) VALUES (?,?,?,?)`,
		uid, u.ShopReplaceable.LineupUpdateCount, u.ShopReplaceable.LatestLineupUpdateDatetime, u.ShopReplaceable.LatestVersion); err != nil {
		return err
	}

	var obtainItemId, obtainCount sql.NullInt64
	if u.Gacha.ConvertedGachaMedal.ObtainPossession != nil {
		obtainItemId = sql.NullInt64{Int64: int64(u.Gacha.ConvertedGachaMedal.ObtainPossession.ConsumableItemId), Valid: true}
		obtainCount = sql.NullInt64{Int64: int64(u.Gacha.ConvertedGachaMedal.ObtainPossession.Count), Valid: true}
	}
	if err := exec(`INSERT INTO user_gacha (user_id, reward_available, todays_current_draw_count, daily_max_count, last_reward_draw_date, obtain_consumable_item_id, obtain_count) VALUES (?,?,?,?,?,?,?)`,
		uid, boolToInt(u.Gacha.RewardAvailable), u.Gacha.TodaysCurrentDrawCount, u.Gacha.DailyMaxCount,
		u.Gacha.LastRewardDrawDate, obtainItemId, obtainCount); err != nil {
		return err
	}

	for friendUserId, friend := range u.Friends {
		if err := exec(`INSERT INTO user_friends (user_id, friend_user_id, is_friend, cheer_sent_datetime,
			cheer_received_datetime, stamina_received_datetime) VALUES (?,?,?,?,?,?)`,
			uid, friendUserId, boolToInt(friend.IsFriend), friend.CheerSentDatetime, friend.CheerReceivedDatetime,
			friend.StaminaReceivedDatetime); err != nil {
			return err
		}
	}
	for requesterUserId, requestDatetime := range u.FriendRequests {
		if err := exec(`INSERT INTO user_friend_requests (user_id, requester_user_id, request_datetime) VALUES (?,?,?)`,
			uid, requesterUserId, requestDatetime); err != nil {
			return err
		}
	}

	for _, v := range u.Characters {
		if err := exec(`INSERT INTO user_characters (user_id, character_id, level, exp, latest_version) VALUES (?,?,?,?,?)`,
			uid, v.CharacterId, v.Level, v.Exp, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.Costumes {
		if err := exec(`INSERT INTO user_costumes (user_id, user_costume_uuid, costume_id, limit_break_count, level, exp, headup_display_view_id, acquisition_datetime, awaken_count, costume_lottery_effect_unlocked_slot_count, latest_version) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			uid, v.UserCostumeUuid, v.CostumeId, v.LimitBreakCount, v.Level, v.Exp, v.HeadupDisplayViewId, v.AcquisitionDatetime, v.AwakenCount, v.CostumeLotteryEffectUnlockedSlotCount, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.Weapons {
		if err := exec(`INSERT INTO user_weapons (user_id, user_weapon_uuid, weapon_id, level, exp, limit_break_count, is_protected, acquisition_datetime, latest_version) VALUES (?,?,?,?,?,?,?,?,?)`,
			uid, v.UserWeaponUuid, v.WeaponId, v.Level, v.Exp, v.LimitBreakCount, boolToInt(v.IsProtected), v.AcquisitionDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.Companions {
		if err := exec(`INSERT INTO user_companions (user_id, user_companion_uuid, companion_id, headup_display_view_id, level, acquisition_datetime, latest_version) VALUES (?,?,?,?,?,?,?)`,
			uid, v.UserCompanionUuid, v.CompanionId, v.HeadupDisplayViewId, v.Level, v.AcquisitionDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.Thoughts {
		if err := exec(`INSERT INTO user_thoughts (user_id, user_thought_uuid, thought_id, acquisition_datetime, latest_version) VALUES (?,?,?,?,?)`,
			uid, v.UserThoughtUuid, v.ThoughtId, v.AcquisitionDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.DeckCharacters {
		if err := exec(`INSERT INTO user_deck_characters (user_id, user_deck_character_uuid, user_costume_uuid, main_user_weapon_uuid, user_companion_uuid, power, user_thought_uuid, dressup_costume_id, latest_version) VALUES (?,?,?,?,?,?,?,?,?)`,
			uid, v.UserDeckCharacterUuid, v.UserCostumeUuid, v.MainUserWeaponUuid, v.UserCompanionUuid, v.Power, v.UserThoughtUuid, v.DressupCostumeId, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.Decks {
		if err := exec(`INSERT INTO user_decks (user_id, deck_type, user_deck_number, user_deck_character_uuid01, user_deck_character_uuid02, user_deck_character_uuid03, name, power, latest_version) VALUES (?,?,?,?,?,?,?,?,?)`,
			uid, int32(k.DeckType), k.UserDeckNumber, v.UserDeckCharacterUuid01, v.UserDeckCharacterUuid02, v.UserDeckCharacterUuid03, v.Name, v.Power, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.TripleDecks {
		if err := exec(`INSERT INTO user_triple_decks (user_id, deck_type, user_deck_number, name, deck_number01, deck_number02, deck_number03, latest_version) VALUES (?,?,?,?,?,?,?,?)`,
			uid, int32(k.DeckType), k.UserDeckNumber, v.Name, v.DeckNumber01, v.DeckNumber02, v.DeckNumber03, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.DeckLimitContentRestricted {
		if err := exec(`INSERT INTO user_deck_limit_content_restricted (user_id, deck_restricted_uuid, event_quest_chapter_id, quest_id, possession_type, target_uuid, latest_version) VALUES (?,?,?,?,?,?,?)`,
			uid, v.DeckRestrictedUuid, v.EventQuestChapterId, v.QuestId, v.PossessionType, v.TargetUuid, v.LatestVersion); err != nil {
			return err
		}
	}
	for key, uuids := range u.DeckSubWeapons {
		for i, uuid := range uuids {
			if err := exec(`INSERT INTO user_deck_sub_weapons (user_id, user_deck_character_uuid, ordinal, user_weapon_uuid) VALUES (?,?,?,?)`,
				uid, key, i, uuid); err != nil {
				return err
			}
		}
	}
	for key, uuids := range u.DeckParts {
		for i, uuid := range uuids {
			if err := exec(`INSERT INTO user_deck_parts (user_id, user_deck_character_uuid, ordinal, user_parts_uuid) VALUES (?,?,?,?)`,
				uid, key, i, uuid); err != nil {
				return err
			}
		}
	}
	for _, v := range u.Quests {
		if err := exec(`INSERT INTO user_quests (user_id, quest_id, quest_state_type, is_battle_only, user_deck_number, latest_start_datetime, clear_count, daily_clear_count, last_clear_datetime, shortest_clear_frames, is_reward_granted, latest_version) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			uid, v.QuestId, int32(v.QuestStateType), boolToInt(v.IsBattleOnly), v.UserDeckNumber, v.LatestStartDatetime,
			v.ClearCount, v.DailyClearCount, v.LastClearDatetime, v.ShortestClearFrames, boolToInt(v.IsRewardGranted), v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.QuestMissions {
		if err := exec(`INSERT INTO user_quest_missions (user_id, quest_id, quest_mission_id, progress_value, is_clear, latest_clear_datetime, latest_version) VALUES (?,?,?,?,?,?,?)`,
			uid, k.QuestId, k.QuestMissionId, v.ProgressValue, boolToInt(v.IsClear), v.LatestClearDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.Missions {
		if err := exec(`INSERT INTO user_missions (user_id, mission_id, start_datetime, progress_value, mission_progress_status_type, clear_datetime, latest_version) VALUES (?,?,?,?,?,?,?)`,
			uid, v.MissionId, v.StartDatetime, v.ProgressValue, v.MissionProgressStatusType, v.ClearDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.Tutorials {
		if err := exec(`INSERT INTO user_tutorials (user_id, tutorial_type, progress_phase, choice_id, latest_version) VALUES (?,?,?,?,?)`,
			uid, v.TutorialType, v.ProgressPhase, v.ChoiceId, v.LatestVersion); err != nil {
			return err
		}
	}
	for id, v := range u.SideStoryQuests {
		if err := exec(`INSERT INTO user_side_story_quests (user_id, side_story_quest_id, head_side_story_quest_scene_id, side_story_quest_state_type, latest_version) VALUES (?,?,?,?,?)`,
			uid, id, v.HeadSideStoryQuestSceneId, int32(v.SideStoryQuestStateType), v.LatestVersion); err != nil {
			return err
		}
	}
	for id, v := range u.QuestLimitContentStatus {
		if err := exec(`INSERT INTO user_quest_limit_content_status (user_id, limit_content_id, limit_content_quest_status_type, event_quest_chapter_id, latest_version) VALUES (?,?,?,?,?)`,
			uid, id, v.LimitContentQuestStatusType, v.EventQuestChapterId, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.WeaponStories {
		if err := exec(`INSERT INTO user_weapon_stories (user_id, weapon_id, released_max_story_index, latest_version) VALUES (?,?,?,?)`,
			uid, v.WeaponId, v.ReleasedMaxStoryIndex, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.WeaponNotes {
		if err := exec(`INSERT INTO user_weapon_notes (user_id, weapon_id, max_level, max_limit_break_count, first_acquisition_datetime, latest_version) VALUES (?,?,?,?,?,?)`,
			uid, v.WeaponId, v.MaxLevel, v.MaxLimitBreakCount, v.FirstAcquisitionDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, skills := range u.WeaponSkills {
		for _, v := range skills {
			if err := exec(`INSERT INTO user_weapon_skills (user_id, user_weapon_uuid, slot_number, level) VALUES (?,?,?,?)`,
				uid, v.UserWeaponUuid, v.SlotNumber, v.Level); err != nil {
				return err
			}
		}
	}
	for _, abilities := range u.WeaponAbilities {
		for _, v := range abilities {
			if err := exec(`INSERT INTO user_weapon_abilities (user_id, user_weapon_uuid, slot_number, level) VALUES (?,?,?,?)`,
				uid, v.UserWeaponUuid, v.SlotNumber, v.Level); err != nil {
				return err
			}
		}
	}
	for _, v := range u.WeaponAwakens {
		if err := exec(`INSERT INTO user_weapon_awakens (user_id, user_weapon_uuid, latest_version) VALUES (?,?,?)`,
			uid, v.UserWeaponUuid, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.CostumeActiveSkills {
		if err := exec(`INSERT INTO user_costume_active_skills (user_id, user_costume_uuid, level, acquisition_datetime, latest_version) VALUES (?,?,?,?,?)`,
			uid, v.UserCostumeUuid, v.Level, v.AcquisitionDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.CostumeAwakenStatusUps {
		if err := exec(`INSERT INTO user_costume_awaken_status_ups (user_id, user_costume_uuid, status_calculation_type, hp, attack, vitality, agility, critical_ratio, critical_attack, latest_version) VALUES (?,?,?,?,?,?,?,?,?,?)`,
			uid, k.UserCostumeUuid, int32(k.StatusCalculationType), v.Hp, v.Attack, v.Vitality, v.Agility, v.CriticalRatio, v.CriticalAttack, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.CostumeLevelBonusReleaseStatuses {
		if err := exec(`INSERT INTO user_costume_level_bonus_release_statuses (user_id, costume_id, last_released_bonus_level, confirmed_bonus_level, latest_version) VALUES (?,?,?,?,?)`,
			uid, v.CostumeId, v.LastReleasedBonusLevel, v.ConfirmedBonusLevel, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.CostumeLotteryEffects {
		if err := exec(`INSERT INTO user_costume_lottery_effects (user_id, user_costume_uuid, slot_number, odds_number, latest_version) VALUES (?,?,?,?,?)`,
			uid, k.UserCostumeUuid, k.SlotNumber, v.OddsNumber, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.CostumeLotteryEffectAbilities {
		if err := exec(`INSERT INTO user_costume_lottery_effect_abilities (user_id, user_costume_uuid, slot_number, ability_id, ability_level, latest_version) VALUES (?,?,?,?,?,?)`,
			uid, k.UserCostumeUuid, k.SlotNumber, v.AbilityId, v.AbilityLevel, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.CostumeLotteryEffectStatusUps {
		if err := exec(`INSERT INTO user_costume_lottery_effect_status_ups (user_id, user_costume_uuid, status_calculation_type, hp, attack, vitality, agility, critical_ratio, critical_attack, latest_version) VALUES (?,?,?,?,?,?,?,?,?,?)`,
			uid, k.UserCostumeUuid, int32(k.StatusCalculationType), v.Hp, v.Attack, v.Vitality, v.Agility, v.CriticalRatio, v.CriticalAttack, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.CostumeLotteryEffectPending {
		if err := exec(`INSERT INTO user_costume_lottery_effect_pending (user_id, user_costume_uuid, slot_number, odds_number, latest_version) VALUES (?,?,?,?,?)`,
			uid, v.UserCostumeUuid, v.SlotNumber, v.OddsNumber, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.Parts {
		if err := exec(`INSERT INTO user_parts (user_id, user_parts_uuid, parts_id, level, parts_status_main_id, is_protected, acquisition_datetime, latest_version) VALUES (?,?,?,?,?,?,?,?)`,
			uid, v.UserPartsUuid, v.PartsId, v.Level, v.PartsStatusMainId, boolToInt(v.IsProtected), v.AcquisitionDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.PartsGroupNotes {
		if err := exec(`INSERT INTO user_parts_group_notes (user_id, parts_group_id, first_acquisition_datetime, latest_version) VALUES (?,?,?,?)`,
			uid, v.PartsGroupId, v.FirstAcquisitionDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.PartsPresets {
		if err := exec(`INSERT INTO user_parts_presets (user_id, user_parts_preset_number, user_parts_uuid01, user_parts_uuid02, user_parts_uuid03, name, user_parts_preset_tag_number, latest_version) VALUES (?,?,?,?,?,?,?,?)`,
			uid, v.UserPartsPresetNumber, v.UserPartsUuid01, v.UserPartsUuid02, v.UserPartsUuid03, v.Name, v.UserPartsPresetTagNumber, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.PartsPresetTags {
		if err := exec(`INSERT INTO user_parts_preset_tags (user_id, user_parts_preset_tag_number, name, latest_version) VALUES (?,?,?,?)`,
			uid, v.UserPartsPresetTagNumber, v.Name, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.PartsStatusSubs {
		if err := exec(`INSERT INTO user_parts_status_subs (user_id, user_parts_uuid, status_index, parts_status_sub_lottery_id, level, status_kind_type, status_calculation_type, status_change_value, latest_version) VALUES (?,?,?,?,?,?,?,?,?)`,
			uid, v.UserPartsUuid, v.StatusIndex, v.PartsStatusSubLotteryId, v.Level, v.StatusKindType, v.StatusCalculationType, v.StatusChangeValue, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.DeckTypeNotes {
		if err := exec(`INSERT INTO user_deck_type_notes (user_id, deck_type, max_deck_power, latest_version) VALUES (?,?,?,?)`,
			uid, int32(v.DeckType), v.MaxDeckPower, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.ConsumableItems {
		if err := exec(`INSERT INTO user_consumable_items (user_id, consumable_item_id, count) VALUES (?,?,?)`, uid, k, v); err != nil {
			return err
		}
	}
	for k, v := range u.Materials {
		if err := exec(`INSERT INTO user_materials (user_id, material_id, count) VALUES (?,?,?)`, uid, k, v); err != nil {
			return err
		}
	}
	for k, v := range u.ImportantItems {
		if err := exec(`INSERT INTO user_important_items (user_id, important_item_id, count) VALUES (?,?,?)`, uid, k, v); err != nil {
			return err
		}
	}
	for k, v := range u.PremiumItems {
		if err := exec(`INSERT INTO user_premium_items (user_id, premium_item_id, count) VALUES (?,?,?)`, uid, k, v); err != nil {
			return err
		}
	}
	for _, v := range u.ExploreScores {
		if err := exec(`INSERT INTO user_explore_scores (user_id, explore_id, max_score, max_score_update_datetime, latest_version) VALUES (?,?,?,?,?)`,
			uid, v.ExploreId, v.MaxScore, v.MaxScoreUpdateDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.AutoSaleSettings {
		if err := exec(`INSERT INTO user_auto_sale_settings (user_id, possession_auto_sale_item_type, possession_auto_sale_item_value) VALUES (?,?,?)`,
			uid, v.PossessionAutoSaleItemType, v.PossessionAutoSaleItemValue); err != nil {
			return err
		}
	}
	for k := range u.NaviCutInPlayed {
		if err := exec(`INSERT INTO user_navi_cutin_played (user_id, navi_cutin_id) VALUES (?,?)`, uid, k); err != nil {
			return err
		}
	}
	for k, v := range u.ViewedMovies {
		if err := exec(`INSERT INTO user_viewed_movies (user_id, movie_id, timestamp) VALUES (?,?,?)`, uid, k, v); err != nil {
			return err
		}
	}
	for k, v := range u.ContentsStories {
		if err := exec(`INSERT INTO user_contents_stories (user_id, contents_story_id, timestamp) VALUES (?,?,?)`, uid, k, v); err != nil {
			return err
		}
	}
	for k, v := range u.DrawnOmikuji {
		if err := exec(`INSERT INTO user_drawn_omikuji (user_id, omikuji_id, timestamp) VALUES (?,?,?)`, uid, k, v); err != nil {
			return err
		}
	}
	for k := range u.DokanConfirmed {
		if err := exec(`INSERT INTO user_dokan_confirmed (user_id, dokan_id) VALUES (?,?)`, uid, k); err != nil {
			return err
		}
	}
	for _, g := range u.Gifts.NotReceived {
		var expDt sql.NullInt64
		if g.ExpirationDatetime != 0 {
			expDt = sql.NullInt64{Int64: g.ExpirationDatetime, Valid: true}
		}
		if err := exec(`INSERT INTO user_gifts (user_id, user_gift_uuid, is_received, possession_type, possession_id, count, grant_datetime, description_gift_text_id, equipment_data, expiration_datetime) VALUES (?,?,0,?,?,?,?,?,?,?)`,
			uid, g.UserGiftUuid, g.GiftCommon.PossessionType, g.GiftCommon.PossessionId, g.GiftCommon.Count,
			g.GiftCommon.GrantDatetime, g.GiftCommon.DescriptionGiftTextId, g.GiftCommon.EquipmentData, expDt); err != nil {
			return err
		}
	}
	for i, g := range u.Gifts.Received {
		uuid := fmt.Sprintf("received-%d-%d", uid, i)
		if err := exec(`INSERT INTO user_gifts (user_id, user_gift_uuid, is_received, possession_type, possession_id, count, grant_datetime, description_gift_text_id, equipment_data, received_datetime) VALUES (?,?,1,?,?,?,?,?,?,?)`,
			uid, uuid, g.GiftCommon.PossessionType, g.GiftCommon.PossessionId, g.GiftCommon.Count,
			g.GiftCommon.GrantDatetime, g.GiftCommon.DescriptionGiftTextId, g.GiftCommon.EquipmentData, g.ReceivedDatetime); err != nil {
			return err
		}
	}
	for i, v := range u.Gacha.ConvertedGachaMedal.ConvertedMedalPossession {
		if err := exec(`INSERT INTO user_gacha_converted_medals (user_id, ordinal, consumable_item_id, count) VALUES (?,?,?,?)`,
			uid, i, v.ConsumableItemId, v.Count); err != nil {
			return err
		}
	}
	for _, v := range u.Gacha.BannerStates {
		if err := exec(`INSERT INTO user_gacha_banners (user_id, gacha_id, medal_count, step_number, loop_count, draw_count, box_number) VALUES (?,?,?,?,?,?,?)`,
			uid, v.GachaId, v.MedalCount, v.StepNumber, v.LoopCount, v.DrawCount, v.BoxNumber); err != nil {
			return err
		}
		for itemId, count := range v.BoxDrewCounts {
			if err := exec(`INSERT INTO user_gacha_banner_box_drew_counts (user_id, gacha_id, box_item_id, count) VALUES (?,?,?,?)`,
				uid, v.GachaId, itemId, count); err != nil {
				return err
			}
		}
	}
	for _, v := range u.CharacterBoards {
		if err := exec(`INSERT INTO user_character_boards (user_id, character_board_id, panel_release_bit1, panel_release_bit2, panel_release_bit3, panel_release_bit4, latest_version) VALUES (?,?,?,?,?,?,?)`,
			uid, v.CharacterBoardId, v.PanelReleaseBit1, v.PanelReleaseBit2, v.PanelReleaseBit3, v.PanelReleaseBit4, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.CharacterBoardAbilities {
		if err := exec(`INSERT INTO user_character_board_abilities (user_id, character_id, ability_id, level, latest_version) VALUES (?,?,?,?,?)`,
			uid, k.CharacterId, k.AbilityId, v.Level, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.CharacterBoardStatusUps {
		if err := exec(`INSERT INTO user_character_board_status_ups (user_id, character_id, status_calculation_type, hp, attack, vitality, agility, critical_ratio, critical_attack, latest_version) VALUES (?,?,?,?,?,?,?,?,?,?)`,
			uid, k.CharacterId, k.StatusCalculationType, v.Hp, v.Attack, v.Vitality, v.Agility, v.CriticalRatio, v.CriticalAttack, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.CharacterRebirths {
		if err := exec(`INSERT INTO user_character_rebirths (user_id, character_id, rebirth_count, latest_version) VALUES (?,?,?,?)`,
			uid, v.CharacterId, v.RebirthCount, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.CageOrnamentRewards {
		if err := exec(`INSERT INTO user_cage_ornament_rewards (user_id, cage_ornament_id, acquisition_datetime, latest_version) VALUES (?,?,?,?)`,
			uid, v.CageOrnamentId, v.AcquisitionDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.CageOrnamentAccesses {
		if err := exec(`INSERT INTO user_cage_ornament_accesses (user_id, cage_ornament_id, first_access_datetime, latest_access_datetime, latest_version) VALUES (?,?,?,?,?)`,
			uid, v.CageOrnamentId, v.FirstAccessDatetime, v.LatestAccessDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.TowerAccumulationRewards {
		if err := exec(`INSERT INTO user_event_quest_tower_accumulation_rewards (user_id, event_quest_chapter_id, latest_reward_receive_quest_mission_clear_count, latest_version) VALUES (?,?,?,?)`,
			uid, v.EventQuestChapterId, v.LatestRewardReceiveQuestMissionClearCount, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.LabyrinthSeasons {
		if err := exec(`INSERT INTO user_event_quest_labyrinth_seasons (user_id, event_quest_chapter_id, last_join_season_number, last_season_reward_received_season_number, latest_version) VALUES (?,?,?,?,?)`,
			uid, v.EventQuestChapterId, v.LastJoinSeasonNumber, v.LastSeasonRewardReceivedSeasonNumber, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.LabyrinthStages {
		if err := exec(`INSERT INTO user_event_quest_labyrinth_stages (user_id, event_quest_chapter_id, stage_order, is_received_stage_clear_reward, accumulation_reward_received_quest_mission_count, latest_version) VALUES (?,?,?,?,?,?)`,
			uid, k.EventQuestChapterId, k.StageOrder, boolToInt(v.IsReceivedStageClearReward), v.AccumulationRewardReceivedQuestMissionCount, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.ShopItems {
		if err := exec(`INSERT INTO user_shop_items (user_id, shop_item_id, bought_count, latest_bought_count_changed_datetime, latest_version) VALUES (?,?,?,?,?)`,
			uid, v.ShopItemId, v.BoughtCount, v.LatestBoughtCountChangedDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for _, v := range u.ShopReplaceableLineup {
		if err := exec(`INSERT INTO user_shop_replaceable_lineup (user_id, slot_number, shop_item_id, latest_version) VALUES (?,?,?,?)`,
			uid, v.SlotNumber, v.ShopItemId, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.Gimmick.Progress {
		if err := exec(`INSERT INTO user_gimmick_progress (user_id, gimmick_sequence_schedule_id, gimmick_sequence_id, gimmick_id, is_gimmick_cleared, start_datetime, latest_version) VALUES (?,?,?,?,?,?,?)`,
			uid, k.GimmickSequenceScheduleId, k.GimmickSequenceId, k.GimmickId, boolToInt(v.IsGimmickCleared), v.StartDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.Gimmick.OrnamentProgress {
		if err := exec(`INSERT INTO user_gimmick_ornament_progress (user_id, gimmick_sequence_schedule_id, gimmick_sequence_id, gimmick_id, gimmick_ornament_index, progress_value_bit, base_datetime, latest_version) VALUES (?,?,?,?,?,?,?,?)`,
			uid, k.GimmickSequenceScheduleId, k.GimmickSequenceId, k.GimmickId, k.GimmickOrnamentIndex, v.ProgressValueBit, v.BaseDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.Gimmick.Sequences {
		if err := exec(`INSERT INTO user_gimmick_sequences (user_id, gimmick_sequence_schedule_id, gimmick_sequence_id, is_gimmick_sequence_cleared, clear_datetime, latest_version) VALUES (?,?,?,?,?,?)`,
			uid, k.GimmickSequenceScheduleId, k.GimmickSequenceId, boolToInt(v.IsGimmickSequenceCleared), v.ClearDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.Gimmick.Unlocks {
		if err := exec(`INSERT INTO user_gimmick_unlocks (user_id, gimmick_sequence_schedule_id, gimmick_sequence_id, gimmick_id, is_unlocked, latest_version) VALUES (?,?,?,?,?,?)`,
			uid, k.GimmickSequenceScheduleId, k.GimmickSequenceId, k.GimmickId, boolToInt(v.IsUnlocked), v.LatestVersion); err != nil {
			return err
		}
	}
	for id, v := range u.BigHuntMaxScores {
		if err := exec(`INSERT INTO user_big_hunt_max_scores (user_id, big_hunt_boss_id, max_score, max_score_update_datetime, latest_version) VALUES (?,?,?,?,?)`,
			uid, id, v.MaxScore, v.MaxScoreUpdateDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for id, v := range u.BigHuntStatuses {
		if err := exec(`INSERT INTO user_big_hunt_statuses (user_id, big_hunt_boss_id, daily_challenge_count, latest_challenge_datetime, last_daily_reward_received_day_version, latest_version) VALUES (?,?,?,?,?,?)`,
			uid, id, v.DailyChallengeCount, v.LatestChallengeDatetime, v.LastDailyRewardReceivedDayVersion, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.BigHuntScheduleMaxScores {
		if err := exec(`INSERT INTO user_big_hunt_schedule_max_scores (user_id, big_hunt_schedule_id, big_hunt_boss_id, max_score, max_score_update_datetime, latest_version) VALUES (?,?,?,?,?,?)`,
			uid, k.BigHuntScheduleId, k.BigHuntBossId, v.MaxScore, v.MaxScoreUpdateDatetime, v.LatestVersion); err != nil {
			return err
		}
	}
	for k, v := range u.BigHuntWeeklyMaxScores {
		if err := exec(`INSERT INTO user_big_hunt_weekly_max_scores (user_id, big_hunt_weekly_version, attribute_type, max_score, latest_version) VALUES (?,?,?,?,?)`,
			uid, k.BigHuntWeeklyVersion, k.AttributeType, v.MaxScore, v.LatestVersion); err != nil {
			return err
		}
	}
	for ver, v := range u.BigHuntWeeklyStatuses {
		if err := exec(`INSERT INTO user_big_hunt_weekly_statuses (user_id, big_hunt_weekly_version, is_received_weekly_reward, latest_version) VALUES (?,?,?,?)`,
			uid, ver, boolToInt(v.IsReceivedWeeklyReward), v.LatestVersion); err != nil {
			return err
		}
	}

	return nil
}

// 1:1 tables update on field-change; maps INSERT OR REPLACE + DELETE; slice tables DELETE-all then INSERT-all.
func diffAndSave(tx *sql.Tx, uid int64, before, after *store.UserState) error {
	var firstErr error
	exec := func(query string, args ...any) error {
		if firstErr != nil {
			return firstErr
		}
		_, err := tx.Exec(query, args...)
		if err != nil {
			firstErr = err
		}
		return err
	}
	if err := diffMechanismTables(tx, uid, before, after); err != nil {
		return err
	}

	if before.PlayerId != after.PlayerId || before.OsType != after.OsType || before.PlatformType != after.PlatformType ||
		before.UserRestrictionType != after.UserRestrictionType || before.RegisterDatetime != after.RegisterDatetime ||
		before.GameStartDatetime != after.GameStartDatetime || before.LatestVersion != after.LatestVersion ||
		before.BirthYear != after.BirthYear || before.BirthMonth != after.BirthMonth ||
		before.BackupToken != after.BackupToken || before.ChargeMoneyThisMonth != after.ChargeMoneyThisMonth {
		if err := exec(`UPDATE users SET player_id=?, os_type=?, platform_type=?, user_restriction_type=?,
			register_datetime=?, game_start_datetime=?, latest_version=?, birth_year=?, birth_month=?,
			backup_token=?, charge_money_this_month=? WHERE user_id=?`,
			after.PlayerId, after.OsType, after.PlatformType, after.UserRestrictionType,
			after.RegisterDatetime, after.GameStartDatetime, after.LatestVersion, after.BirthYear, after.BirthMonth,
			after.BackupToken, after.ChargeMoneyThisMonth, uid); err != nil {
			return err
		}
	}

	if before.Setting != after.Setting {
		if err := exec(`UPDATE user_setting SET is_notify_purchase_alert=?, latest_version=? WHERE user_id=?`,
			boolToInt(after.Setting.IsNotifyPurchaseAlert), after.Setting.LatestVersion, uid); err != nil {
			return err
		}
	}
	if before.Status != after.Status {
		if err := exec(`UPDATE user_status SET level=?, exp=?, stamina_milli_value=?, stamina_update_datetime=?, latest_version=? WHERE user_id=?`,
			after.Status.Level, after.Status.Exp, after.Status.StaminaMilliValue, after.Status.StaminaUpdateDatetime, after.Status.LatestVersion, uid); err != nil {
			return err
		}
	}
	if before.Gem != after.Gem {
		if err := exec(`UPDATE user_gem SET paid_gem=?, free_gem=? WHERE user_id=?`, after.Gem.PaidGem, after.Gem.FreeGem, uid); err != nil {
			return err
		}
	}
	if before.Profile != after.Profile {
		if err := exec(`UPDATE user_profile SET name=?, name_update_datetime=?, message=?, message_update_datetime=?, favorite_costume_id=?, favorite_costume_id_update_datetime=?, current_pvp_rank=?, current_pvp_grade_id=?, max_pvp_season_rank=?, latest_version=? WHERE user_id=?`,
			after.Profile.Name, after.Profile.NameUpdateDatetime, after.Profile.Message, after.Profile.MessageUpdateDatetime,
			after.Profile.FavoriteCostumeId, after.Profile.FavoriteCostumeIdUpdateDatetime,
			after.Profile.CurrentPvpRank, after.Profile.CurrentPvpGradeId, after.Profile.MaxPvpSeasonRank,
			after.Profile.LatestVersion, uid); err != nil {
			return err
		}
	}
	if before.Login != after.Login {
		if err := exec(`UPDATE user_login SET total_login_count=?, continual_login_count=?, max_continual_login_count=?, last_login_datetime=?, last_comeback_login_datetime=?, latest_version=? WHERE user_id=?`,
			after.Login.TotalLoginCount, after.Login.ContinualLoginCount, after.Login.MaxContinualLoginCount,
			after.Login.LastLoginDatetime, after.Login.LastComebackLoginDatetime, after.Login.LatestVersion, uid); err != nil {
			return err
		}
	}
	if before.LoginBonus != after.LoginBonus {
		if err := exec(`UPDATE user_login_bonus SET login_bonus_id=?, current_page_number=?, current_stamp_number=?, latest_reward_receive_datetime=?, latest_version=? WHERE user_id=?`,
			after.LoginBonus.LoginBonusId, after.LoginBonus.CurrentPageNumber, after.LoginBonus.CurrentStampNumber,
			after.LoginBonus.LatestRewardReceiveDatetime, after.LoginBonus.LatestVersion, uid); err != nil {
			return err
		}
	}
	if before.MainQuest != after.MainQuest {
		if err := exec(`UPDATE user_main_quest SET current_quest_flow_type=?, current_main_quest_route_id=?, current_quest_scene_id=?, head_quest_scene_id=?, is_reached_last_quest_scene=?, progress_quest_scene_id=?, progress_head_quest_scene_id=?, progress_quest_flow_type=?, main_quest_season_id=?, latest_version=?, saved_ctx_active=?, saved_ctx_current_quest_scene_id=?, saved_ctx_head_quest_scene_id=?, saved_ctx_current_main_quest_route_id=?, saved_ctx_main_quest_season_id=?, saved_ctx_is_reached_last_quest_scene=?, saved_ctx_portal_cage_in_progress=?, saved_ctx_current_quest_flow_type=?, replay_flow_current_quest_scene_id=?, replay_flow_head_quest_scene_id=? WHERE user_id=?`,
			after.MainQuest.CurrentQuestFlowType, after.MainQuest.CurrentMainQuestRouteId, after.MainQuest.CurrentQuestSceneId,
			after.MainQuest.HeadQuestSceneId, boolToInt(after.MainQuest.IsReachedLastQuestScene), after.MainQuest.ProgressQuestSceneId,
			after.MainQuest.ProgressHeadQuestSceneId, after.MainQuest.ProgressQuestFlowType, after.MainQuest.MainQuestSeasonId,
			after.MainQuest.LatestVersion,
			boolToInt(after.MainQuest.SavedContext.Active),
			after.MainQuest.SavedContext.CurrentQuestSceneId, after.MainQuest.SavedContext.HeadQuestSceneId,
			after.MainQuest.SavedContext.CurrentMainQuestRouteId, after.MainQuest.SavedContext.MainQuestSeasonId,
			boolToInt(after.MainQuest.SavedContext.IsReachedLastQuestScene),
			boolToInt(after.MainQuest.SavedContext.PortalCageInProgress),
			after.MainQuest.SavedContext.CurrentQuestFlowType,
			after.MainQuest.ReplayFlowCurrentQuestSceneId, after.MainQuest.ReplayFlowHeadQuestSceneId, uid); err != nil {
			return err
		}
	}
	if before.EventQuest != after.EventQuest {
		if err := exec(`UPDATE user_event_quest SET current_event_quest_chapter_id=?, current_quest_id=?, current_quest_scene_id=?, head_quest_scene_id=?, latest_version=? WHERE user_id=?`,
			after.EventQuest.CurrentEventQuestChapterId, after.EventQuest.CurrentQuestId, after.EventQuest.CurrentQuestSceneId, after.EventQuest.HeadQuestSceneId, after.EventQuest.LatestVersion, uid); err != nil {
			return err
		}
	}
	if before.ExtraQuest != after.ExtraQuest {
		if err := exec(`UPDATE user_extra_quest SET current_quest_id=?, current_quest_scene_id=?, head_quest_scene_id=?, latest_version=? WHERE user_id=?`,
			after.ExtraQuest.CurrentQuestId, after.ExtraQuest.CurrentQuestSceneId, after.ExtraQuest.HeadQuestSceneId, after.ExtraQuest.LatestVersion, uid); err != nil {
			return err
		}
	}
	if before.SideStoryActiveProgress != after.SideStoryActiveProgress {
		if err := exec(`UPDATE user_side_story_active SET current_side_story_quest_id=?, current_side_story_quest_scene_id=?, latest_version=? WHERE user_id=?`,
			after.SideStoryActiveProgress.CurrentSideStoryQuestId, after.SideStoryActiveProgress.CurrentSideStoryQuestSceneId, after.SideStoryActiveProgress.LatestVersion, uid); err != nil {
			return err
		}
	}
	if before.BigHuntProgress != after.BigHuntProgress || !bigHuntBattleDetailEqual(before.BigHuntBattleDetail, after.BigHuntBattleDetail) || before.BigHuntDeckNumber != after.BigHuntDeckNumber {
		if err := exec(`UPDATE user_big_hunt_state SET current_big_hunt_boss_quest_id=?, current_big_hunt_quest_id=?, current_quest_scene_id=?, is_dry_run=?, latest_version=?, deck_type=?, user_triple_deck_number=?, boss_knock_down_count=?, max_combo_count=?, total_damage=?, deck_number=?, battle_binary=? WHERE user_id=?`,
			after.BigHuntProgress.CurrentBigHuntBossQuestId, after.BigHuntProgress.CurrentBigHuntQuestId,
			after.BigHuntProgress.CurrentQuestSceneId, boolToInt(after.BigHuntProgress.IsDryRun), after.BigHuntProgress.LatestVersion,
			after.BigHuntBattleDetail.DeckType, after.BigHuntBattleDetail.UserTripleDeckNumber, after.BigHuntBattleDetail.BossKnockDownCount,
			after.BigHuntBattleDetail.MaxComboCount, after.BigHuntBattleDetail.TotalDamage, after.BigHuntDeckNumber, after.BigHuntBattleBinary, uid); err != nil {
			return err
		}
		if err := exec(`DELETE FROM user_big_hunt_costume_battle_infos WHERE user_id=?`, uid); err != nil {
			return err
		}
		for i, ci := range after.BigHuntBattleDetail.CostumeBattleInfo {
			if err := exec(`INSERT INTO user_big_hunt_costume_battle_infos (user_id, wave_index, sort_order, costume_id, total_damage, hit_count, random_display_value_type, random_display_value) VALUES (?,?,?,?,?,?,?,?)`,
				uid, ci.WaveIndex, i, ci.CostumeId, ci.TotalDamage, ci.HitCount, ci.RandomDisplayValueType, ci.RandomDisplayValue); err != nil {
				return err
			}
		}
	}
	if before.Battle != after.Battle {
		if err := exec(`UPDATE user_battle SET is_active=?, start_count=?, finish_count=?, last_started_at=?, last_finished_at=?, last_user_party_count=?, last_npc_party_count=?, last_battle_binary_size=?, last_elapsed_frame_count=? WHERE user_id=?`,
			boolToInt(after.Battle.IsActive), after.Battle.StartCount, after.Battle.FinishCount, after.Battle.LastStartedAt,
			after.Battle.LastFinishedAt, after.Battle.LastUserPartyCount, after.Battle.LastNpcPartyCount,
			after.Battle.LastBattleBinarySize, after.Battle.LastElapsedFrameCount, uid); err != nil {
			return err
		}
	}
	if before.Notifications != after.Notifications {
		if err := exec(`UPDATE user_notification SET gift_not_receive_count=?, friend_request_receive_count=?, is_exist_unread_information=? WHERE user_id=?`,
			after.Notifications.GiftNotReceiveCount, after.Notifications.FriendRequestReceiveCount, boolToInt(after.Notifications.IsExistUnreadInformation), uid); err != nil {
			return err
		}
	}
	if before.PortalCageStatus != after.PortalCageStatus {
		if err := exec(`UPDATE user_portal_cage SET is_current_progress=?, drop_item_start_datetime=?, current_drop_item_count=?, latest_version=? WHERE user_id=?`,
			boolToInt(after.PortalCageStatus.IsCurrentProgress), after.PortalCageStatus.DropItemStartDatetime, after.PortalCageStatus.CurrentDropItemCount, after.PortalCageStatus.LatestVersion, uid); err != nil {
			return err
		}
	}
	if before.GuerrillaFreeOpen != after.GuerrillaFreeOpen {
		if err := exec(`UPDATE user_guerrilla_free_open SET start_datetime=?, open_minutes=?, daily_opened_count=?, latest_version=? WHERE user_id=?`,
			after.GuerrillaFreeOpen.StartDatetime, after.GuerrillaFreeOpen.OpenMinutes, after.GuerrillaFreeOpen.DailyOpenedCount, after.GuerrillaFreeOpen.LatestVersion, uid); err != nil {
			return err
		}
	}
	if !before.QuestAutoOrbit.Equal(after.QuestAutoOrbit) {
		if err := exec(`UPDATE user_quest_auto_orbit SET quest_type=?, chapter_id=?, quest_id=?, max_auto_orbit_count=?, cleared_auto_orbit_count=?, last_clear_datetime=?, latest_version=?, accumulated_drops_json=? WHERE user_id=?`,
			after.QuestAutoOrbit.QuestType, after.QuestAutoOrbit.ChapterId, after.QuestAutoOrbit.QuestId,
			after.QuestAutoOrbit.MaxAutoOrbitCount, after.QuestAutoOrbit.ClearedAutoOrbitCount,
			after.QuestAutoOrbit.LastClearDatetime, after.QuestAutoOrbit.LatestVersion,
			marshalAutoOrbitDrops(after.QuestAutoOrbit.AccumulatedDrops), uid); err != nil {
			return err
		}
	}
	if before.Explore != after.Explore {
		if err := exec(`UPDATE user_explore SET is_use_explore_ticket=?, playing_explore_id=?, latest_play_datetime=?, latest_version=? WHERE user_id=?`,
			boolToInt(after.Explore.IsUseExploreTicket), after.Explore.PlayingExploreId, after.Explore.LatestPlayDatetime, after.Explore.LatestVersion, uid); err != nil {
			return err
		}
	}
	if before.ShopReplaceable != after.ShopReplaceable {
		if err := exec(`UPDATE user_shop_replaceable SET lineup_update_count=?, latest_lineup_update_datetime=?, latest_version=? WHERE user_id=?`,
			after.ShopReplaceable.LineupUpdateCount, after.ShopReplaceable.LatestLineupUpdateDatetime, after.ShopReplaceable.LatestVersion, uid); err != nil {
			return err
		}
	}

	if before.Gacha.RewardAvailable != after.Gacha.RewardAvailable || before.Gacha.TodaysCurrentDrawCount != after.Gacha.TodaysCurrentDrawCount ||
		before.Gacha.DailyMaxCount != after.Gacha.DailyMaxCount || before.Gacha.LastRewardDrawDate != after.Gacha.LastRewardDrawDate {
		var obtainItemId, obtainCount sql.NullInt64
		if after.Gacha.ConvertedGachaMedal.ObtainPossession != nil {
			obtainItemId = sql.NullInt64{Int64: int64(after.Gacha.ConvertedGachaMedal.ObtainPossession.ConsumableItemId), Valid: true}
			obtainCount = sql.NullInt64{Int64: int64(after.Gacha.ConvertedGachaMedal.ObtainPossession.Count), Valid: true}
		}
		if err := exec(`UPDATE user_gacha SET reward_available=?, todays_current_draw_count=?, daily_max_count=?, last_reward_draw_date=?, obtain_consumable_item_id=?, obtain_count=? WHERE user_id=?`,
			boolToInt(after.Gacha.RewardAvailable), after.Gacha.TodaysCurrentDrawCount, after.Gacha.DailyMaxCount,
			after.Gacha.LastRewardDrawDate, obtainItemId, obtainCount, uid); err != nil {
			return err
		}
	}

	for friendUserId, friend := range after.Friends {
		if old, exists := before.Friends[friendUserId]; !exists || old != friend {
			if err := exec(`INSERT OR REPLACE INTO user_friends (user_id, friend_user_id, is_friend, cheer_sent_datetime,
				cheer_received_datetime, stamina_received_datetime) VALUES (?,?,?,?,?,?)`,
				uid, friendUserId, boolToInt(friend.IsFriend), friend.CheerSentDatetime, friend.CheerReceivedDatetime,
				friend.StaminaReceivedDatetime); err != nil {
				return err
			}
		}
	}
	for friendUserId := range before.Friends {
		if _, exists := after.Friends[friendUserId]; !exists {
			if err := exec(`DELETE FROM user_friends WHERE user_id=? AND friend_user_id=?`, uid, friendUserId); err != nil {
				return err
			}
		}
	}
	for requesterUserId, requestDatetime := range after.FriendRequests {
		if old, exists := before.FriendRequests[requesterUserId]; !exists || old != requestDatetime {
			if err := exec(`INSERT OR REPLACE INTO user_friend_requests (user_id, requester_user_id, request_datetime) VALUES (?,?,?)`,
				uid, requesterUserId, requestDatetime); err != nil {
				return err
			}
		}
	}
	for requesterUserId := range before.FriendRequests {
		if _, exists := after.FriendRequests[requesterUserId]; !exists {
			if err := exec(`DELETE FROM user_friend_requests WHERE user_id=? AND requester_user_id=?`, uid, requesterUserId); err != nil {
				return err
			}
		}
	}

	if err := diffMapInt32(tx, uid, before.Characters, after.Characters, "user_characters", "character_id",
		func(v store.CharacterState) []any { return []any{v.CharacterId, v.Level, v.Exp, v.LatestVersion} },
		"character_id, level, exp, latest_version"); err != nil {
		return err
	}
	if err := diffMapStr(tx, uid, before.Costumes, after.Costumes, "user_costumes", "user_costume_uuid",
		func(v store.CostumeState) []any {
			return []any{v.UserCostumeUuid, v.CostumeId, v.LimitBreakCount, v.Level, v.Exp, v.HeadupDisplayViewId, v.AcquisitionDatetime, v.AwakenCount, v.CostumeLotteryEffectUnlockedSlotCount, v.LatestVersion}
		}, "user_costume_uuid, costume_id, limit_break_count, level, exp, headup_display_view_id, acquisition_datetime, awaken_count, costume_lottery_effect_unlocked_slot_count, latest_version"); err != nil {
		return err
	}
	if err := diffMapStr(tx, uid, before.Weapons, after.Weapons, "user_weapons", "user_weapon_uuid",
		func(v store.WeaponState) []any {
			return []any{v.UserWeaponUuid, v.WeaponId, v.Level, v.Exp, v.LimitBreakCount, boolToInt(v.IsProtected), v.AcquisitionDatetime, v.LatestVersion}
		}, "user_weapon_uuid, weapon_id, level, exp, limit_break_count, is_protected, acquisition_datetime, latest_version"); err != nil {
		return err
	}
	if err := diffMapStr(tx, uid, before.Companions, after.Companions, "user_companions", "user_companion_uuid",
		func(v store.CompanionState) []any {
			return []any{v.UserCompanionUuid, v.CompanionId, v.HeadupDisplayViewId, v.Level, v.AcquisitionDatetime, v.LatestVersion}
		}, "user_companion_uuid, companion_id, headup_display_view_id, level, acquisition_datetime, latest_version"); err != nil {
		return err
	}
	if err := diffMapStr(tx, uid, before.Thoughts, after.Thoughts, "user_thoughts", "user_thought_uuid",
		func(v store.ThoughtState) []any {
			return []any{v.UserThoughtUuid, v.ThoughtId, v.AcquisitionDatetime, v.LatestVersion}
		}, "user_thought_uuid, thought_id, acquisition_datetime, latest_version"); err != nil {
		return err
	}
	if err := diffMapStr(tx, uid, before.DeckCharacters, after.DeckCharacters, "user_deck_characters", "user_deck_character_uuid",
		func(v store.DeckCharacterState) []any {
			return []any{v.UserDeckCharacterUuid, v.UserCostumeUuid, v.MainUserWeaponUuid, v.UserCompanionUuid, v.Power, v.UserThoughtUuid, v.DressupCostumeId, v.LatestVersion}
		}, "user_deck_character_uuid, user_costume_uuid, main_user_weapon_uuid, user_companion_uuid, power, user_thought_uuid, dressup_costume_id, latest_version"); err != nil {
		return err
	}

	for k, v := range after.Decks {
		if old, ok := before.Decks[k]; !ok || old != v {
			exec(fmt.Sprintf(`INSERT OR REPLACE INTO user_decks (user_id, deck_type, user_deck_number, user_deck_character_uuid01, user_deck_character_uuid02, user_deck_character_uuid03, name, power, latest_version) VALUES (?,?,?,?,?,?,?,?,?)`),
				uid, int32(k.DeckType), k.UserDeckNumber, v.UserDeckCharacterUuid01, v.UserDeckCharacterUuid02, v.UserDeckCharacterUuid03, v.Name, v.Power, v.LatestVersion)
		}
	}
	for k := range before.Decks {
		if _, ok := after.Decks[k]; !ok {
			exec(`DELETE FROM user_decks WHERE user_id=? AND deck_type=? AND user_deck_number=?`, uid, int32(k.DeckType), k.UserDeckNumber)
		}
	}

	for k, v := range after.TripleDecks {
		if old, ok := before.TripleDecks[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_triple_decks (user_id, deck_type, user_deck_number, name, deck_number01, deck_number02, deck_number03, latest_version) VALUES (?,?,?,?,?,?,?,?)`,
				uid, int32(k.DeckType), k.UserDeckNumber, v.Name, v.DeckNumber01, v.DeckNumber02, v.DeckNumber03, v.LatestVersion)
		}
	}
	for k := range before.TripleDecks {
		if _, ok := after.TripleDecks[k]; !ok {
			exec(`DELETE FROM user_triple_decks WHERE user_id=? AND deck_type=? AND user_deck_number=?`, uid, int32(k.DeckType), k.UserDeckNumber)
		}
	}
	if err := diffMapStr(tx, uid, before.DeckLimitContentRestricted, after.DeckLimitContentRestricted,
		"user_deck_limit_content_restricted", "deck_restricted_uuid",
		func(v store.DeckLimitContentRestrictedState) []any {
			return []any{v.DeckRestrictedUuid, v.EventQuestChapterId, v.QuestId, v.PossessionType, v.TargetUuid, v.LatestVersion}
		}, "deck_restricted_uuid, event_quest_chapter_id, quest_id, possession_type, target_uuid, latest_version"); err != nil {
		return err
	}

	if err := replaceSliceTable(tx, uid, "user_deck_sub_weapons", after.DeckSubWeapons, func(key string, uuids []string) {
		for i, uuid := range uuids {
			exec(`INSERT INTO user_deck_sub_weapons (user_id, user_deck_character_uuid, ordinal, user_weapon_uuid) VALUES (?,?,?,?)`, uid, key, i, uuid)
		}
	}); err != nil {
		return err
	}
	if err := replaceSliceTable(tx, uid, "user_deck_parts", after.DeckParts, func(key string, uuids []string) {
		for i, uuid := range uuids {
			exec(`INSERT INTO user_deck_parts (user_id, user_deck_character_uuid, ordinal, user_parts_uuid) VALUES (?,?,?,?)`, uid, key, i, uuid)
		}
	}); err != nil {
		return err
	}

	if err := diffMapInt32(tx, uid, before.Quests, after.Quests, "user_quests", "quest_id",
		func(v store.UserQuestState) []any {
			return []any{v.QuestId, int32(v.QuestStateType), boolToInt(v.IsBattleOnly), v.UserDeckNumber, v.LatestStartDatetime, v.ClearCount, v.DailyClearCount, v.LastClearDatetime, v.ShortestClearFrames, boolToInt(v.IsRewardGranted), v.LatestVersion}
		}, "quest_id, quest_state_type, is_battle_only, user_deck_number, latest_start_datetime, clear_count, daily_clear_count, last_clear_datetime, shortest_clear_frames, is_reward_granted, latest_version"); err != nil {
		return err
	}

	for k, v := range after.QuestMissions {
		if old, ok := before.QuestMissions[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_quest_missions (user_id, quest_id, quest_mission_id, progress_value, is_clear, latest_clear_datetime, latest_version) VALUES (?,?,?,?,?,?,?)`,
				uid, k.QuestId, k.QuestMissionId, v.ProgressValue, boolToInt(v.IsClear), v.LatestClearDatetime, v.LatestVersion)
		}
	}
	for k := range before.QuestMissions {
		if _, ok := after.QuestMissions[k]; !ok {
			exec(`DELETE FROM user_quest_missions WHERE user_id=? AND quest_id=? AND quest_mission_id=?`, uid, k.QuestId, k.QuestMissionId)
		}
	}

	if err := diffMapInt32(tx, uid, before.Missions, after.Missions, "user_missions", "mission_id",
		func(v store.UserMissionState) []any {
			return []any{v.MissionId, v.StartDatetime, v.ProgressValue, v.MissionProgressStatusType, v.ClearDatetime, v.LatestVersion}
		}, "mission_id, start_datetime, progress_value, mission_progress_status_type, clear_datetime, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.Tutorials, after.Tutorials, "user_tutorials", "tutorial_type",
		func(v store.TutorialProgressState) []any {
			return []any{v.TutorialType, v.ProgressPhase, v.ChoiceId, v.LatestVersion}
		},
		"tutorial_type, progress_phase, choice_id, latest_version"); err != nil {
		return err
	}

	if err := diffMapInt32(tx, uid, before.SideStoryQuests, after.SideStoryQuests, "user_side_story_quests", "side_story_quest_id",
		func(v store.SideStoryQuestProgress) []any {
			return []any{0, v.HeadSideStoryQuestSceneId, int32(v.SideStoryQuestStateType), v.LatestVersion}
		}, "side_story_quest_id, head_side_story_quest_scene_id, side_story_quest_state_type, latest_version"); err != nil {
		return err
	}

	if err := diffMapInt32(tx, uid, before.QuestLimitContentStatus, after.QuestLimitContentStatus, "user_quest_limit_content_status", "limit_content_id",
		func(v store.QuestLimitContentStatus) []any {
			return []any{0, v.LimitContentQuestStatusType, v.EventQuestChapterId, v.LatestVersion}
		}, "limit_content_id, limit_content_quest_status_type, event_quest_chapter_id, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.WeaponStories, after.WeaponStories, "user_weapon_stories", "weapon_id",
		func(v store.WeaponStoryState) []any {
			return []any{v.WeaponId, v.ReleasedMaxStoryIndex, v.LatestVersion}
		},
		"weapon_id, released_max_story_index, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.WeaponNotes, after.WeaponNotes, "user_weapon_notes", "weapon_id",
		func(v store.WeaponNoteState) []any {
			return []any{v.WeaponId, v.MaxLevel, v.MaxLimitBreakCount, v.FirstAcquisitionDatetime, v.LatestVersion}
		}, "weapon_id, max_level, max_limit_break_count, first_acquisition_datetime, latest_version"); err != nil {
		return err
	}

	exec(`DELETE FROM user_weapon_skills WHERE user_id=?`, uid)
	for _, skills := range after.WeaponSkills {
		for _, v := range skills {
			exec(`INSERT INTO user_weapon_skills (user_id, user_weapon_uuid, slot_number, level) VALUES (?,?,?,?)`, uid, v.UserWeaponUuid, v.SlotNumber, v.Level)
		}
	}
	exec(`DELETE FROM user_weapon_abilities WHERE user_id=?`, uid)
	for _, abilities := range after.WeaponAbilities {
		for _, v := range abilities {
			exec(`INSERT INTO user_weapon_abilities (user_id, user_weapon_uuid, slot_number, level) VALUES (?,?,?,?)`, uid, v.UserWeaponUuid, v.SlotNumber, v.Level)
		}
	}

	if err := diffMapStr(tx, uid, before.WeaponAwakens, after.WeaponAwakens, "user_weapon_awakens", "user_weapon_uuid",
		func(v store.WeaponAwakenState) []any { return []any{v.UserWeaponUuid, v.LatestVersion} },
		"user_weapon_uuid, latest_version"); err != nil {
		return err
	}
	if err := diffMapStr(tx, uid, before.CostumeActiveSkills, after.CostumeActiveSkills, "user_costume_active_skills", "user_costume_uuid",
		func(v store.CostumeActiveSkillState) []any {
			return []any{v.UserCostumeUuid, v.Level, v.AcquisitionDatetime, v.LatestVersion}
		}, "user_costume_uuid, level, acquisition_datetime, latest_version"); err != nil {
		return err
	}

	for k, v := range after.CostumeAwakenStatusUps {
		if old, ok := before.CostumeAwakenStatusUps[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_costume_awaken_status_ups (user_id, user_costume_uuid, status_calculation_type, hp, attack, vitality, agility, critical_ratio, critical_attack, latest_version) VALUES (?,?,?,?,?,?,?,?,?,?)`,
				uid, k.UserCostumeUuid, int32(k.StatusCalculationType), v.Hp, v.Attack, v.Vitality, v.Agility, v.CriticalRatio, v.CriticalAttack, v.LatestVersion)
		}
	}
	for k := range before.CostumeAwakenStatusUps {
		if _, ok := after.CostumeAwakenStatusUps[k]; !ok {
			exec(`DELETE FROM user_costume_awaken_status_ups WHERE user_id=? AND user_costume_uuid=? AND status_calculation_type=?`, uid, k.UserCostumeUuid, int32(k.StatusCalculationType))
		}
	}
	if err := diffMapInt32(tx, uid, before.CostumeLevelBonusReleaseStatuses, after.CostumeLevelBonusReleaseStatuses,
		"user_costume_level_bonus_release_statuses", "costume_id",
		func(v store.CostumeLevelBonusReleaseStatusState) []any {
			return []any{v.CostumeId, v.LastReleasedBonusLevel, v.ConfirmedBonusLevel, v.LatestVersion}
		}, "costume_id, last_released_bonus_level, confirmed_bonus_level, latest_version"); err != nil {
		return err
	}

	for k, v := range after.CostumeLotteryEffects {
		if old, ok := before.CostumeLotteryEffects[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_costume_lottery_effects (user_id, user_costume_uuid, slot_number, odds_number, latest_version) VALUES (?,?,?,?,?)`,
				uid, k.UserCostumeUuid, k.SlotNumber, v.OddsNumber, v.LatestVersion)
		}
	}
	for k := range before.CostumeLotteryEffects {
		if _, ok := after.CostumeLotteryEffects[k]; !ok {
			exec(`DELETE FROM user_costume_lottery_effects WHERE user_id=? AND user_costume_uuid=? AND slot_number=?`, uid, k.UserCostumeUuid, k.SlotNumber)
		}
	}
	for k, v := range after.CostumeLotteryEffectAbilities {
		if old, ok := before.CostumeLotteryEffectAbilities[k]; !ok || old != v {
			if err := exec(`INSERT OR REPLACE INTO user_costume_lottery_effect_abilities (user_id, user_costume_uuid, slot_number, ability_id, ability_level, latest_version) VALUES (?,?,?,?,?,?)`,
				uid, k.UserCostumeUuid, k.SlotNumber, v.AbilityId, v.AbilityLevel, v.LatestVersion); err != nil {
				return err
			}
		}
	}
	for k := range before.CostumeLotteryEffectAbilities {
		if _, ok := after.CostumeLotteryEffectAbilities[k]; !ok {
			if err := exec(`DELETE FROM user_costume_lottery_effect_abilities WHERE user_id=? AND user_costume_uuid=? AND slot_number=?`, uid, k.UserCostumeUuid, k.SlotNumber); err != nil {
				return err
			}
		}
	}
	for k, v := range after.CostumeLotteryEffectStatusUps {
		if old, ok := before.CostumeLotteryEffectStatusUps[k]; !ok || old != v {
			if err := exec(`INSERT OR REPLACE INTO user_costume_lottery_effect_status_ups (user_id, user_costume_uuid, status_calculation_type, hp, attack, vitality, agility, critical_ratio, critical_attack, latest_version) VALUES (?,?,?,?,?,?,?,?,?,?)`,
				uid, k.UserCostumeUuid, int32(k.StatusCalculationType), v.Hp, v.Attack, v.Vitality, v.Agility, v.CriticalRatio, v.CriticalAttack, v.LatestVersion); err != nil {
				return err
			}
		}
	}
	for k := range before.CostumeLotteryEffectStatusUps {
		if _, ok := after.CostumeLotteryEffectStatusUps[k]; !ok {
			if err := exec(`DELETE FROM user_costume_lottery_effect_status_ups WHERE user_id=? AND user_costume_uuid=? AND status_calculation_type=?`, uid, k.UserCostumeUuid, int32(k.StatusCalculationType)); err != nil {
				return err
			}
		}
	}

	if err := diffMapStr(tx, uid, before.CostumeLotteryEffectPending, after.CostumeLotteryEffectPending, "user_costume_lottery_effect_pending", "user_costume_uuid",
		func(v store.CostumeLotteryEffectPendingState) []any {
			return []any{v.UserCostumeUuid, v.SlotNumber, v.OddsNumber, v.LatestVersion}
		}, "user_costume_uuid, slot_number, odds_number, latest_version"); err != nil {
		return err
	}

	if err := diffMapStr(tx, uid, before.Parts, after.Parts, "user_parts", "user_parts_uuid",
		func(v store.PartsState) []any {
			return []any{v.UserPartsUuid, v.PartsId, v.Level, v.PartsStatusMainId, boolToInt(v.IsProtected), v.AcquisitionDatetime, v.LatestVersion}
		}, "user_parts_uuid, parts_id, level, parts_status_main_id, is_protected, acquisition_datetime, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.PartsGroupNotes, after.PartsGroupNotes, "user_parts_group_notes", "parts_group_id",
		func(v store.PartsGroupNoteState) []any {
			return []any{v.PartsGroupId, v.FirstAcquisitionDatetime, v.LatestVersion}
		},
		"parts_group_id, first_acquisition_datetime, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.PartsPresets, after.PartsPresets, "user_parts_presets", "user_parts_preset_number",
		func(v store.PartsPresetState) []any {
			return []any{v.UserPartsPresetNumber, v.UserPartsUuid01, v.UserPartsUuid02, v.UserPartsUuid03, v.Name, v.UserPartsPresetTagNumber, v.LatestVersion}
		}, "user_parts_preset_number, user_parts_uuid01, user_parts_uuid02, user_parts_uuid03, name, user_parts_preset_tag_number, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.PartsPresetTags, after.PartsPresetTags, "user_parts_preset_tags", "user_parts_preset_tag_number",
		func(v store.PartsPresetTagState) []any {
			return []any{v.UserPartsPresetTagNumber, v.Name, v.LatestVersion}
		}, "user_parts_preset_tag_number, name, latest_version"); err != nil {
		return err
	}

	for k, v := range after.PartsStatusSubs {
		if old, ok := before.PartsStatusSubs[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_parts_status_subs (user_id, user_parts_uuid, status_index, parts_status_sub_lottery_id, level, status_kind_type, status_calculation_type, status_change_value, latest_version) VALUES (?,?,?,?,?,?,?,?,?)`,
				uid, k.UserPartsUuid, k.StatusIndex, v.PartsStatusSubLotteryId, v.Level, v.StatusKindType, v.StatusCalculationType, v.StatusChangeValue, v.LatestVersion)
		}
	}
	for k := range before.PartsStatusSubs {
		if _, ok := after.PartsStatusSubs[k]; !ok {
			exec(`DELETE FROM user_parts_status_subs WHERE user_id=? AND user_parts_uuid=? AND status_index=?`, uid, k.UserPartsUuid, k.StatusIndex)
		}
	}

	for k, v := range after.DeckTypeNotes {
		if old, ok := before.DeckTypeNotes[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_deck_type_notes (user_id, deck_type, max_deck_power, latest_version) VALUES (?,?,?,?)`,
				uid, int32(k), v.MaxDeckPower, v.LatestVersion)
		}
	}
	for k := range before.DeckTypeNotes {
		if _, ok := after.DeckTypeNotes[k]; !ok {
			exec(`DELETE FROM user_deck_type_notes WHERE user_id=? AND deck_type=?`, uid, int32(k))
		}
	}

	if err := diffSimpleMap(tx, uid, before.ConsumableItems, after.ConsumableItems, "user_consumable_items", "consumable_item_id", "count"); err != nil {
		return err
	}
	if err := diffSimpleMap(tx, uid, before.Materials, after.Materials, "user_materials", "material_id", "count"); err != nil {
		return err
	}
	if err := diffSimpleMap(tx, uid, before.ImportantItems, after.ImportantItems, "user_important_items", "important_item_id", "count"); err != nil {
		return err
	}
	if err := diffInt64Map(tx, uid, before.PremiumItems, after.PremiumItems, "user_premium_items", "premium_item_id", "count"); err != nil {
		return err
	}

	if err := diffMapInt32(tx, uid, before.ExploreScores, after.ExploreScores, "user_explore_scores", "explore_id",
		func(v store.ExploreScoreState) []any {
			return []any{v.ExploreId, v.MaxScore, v.MaxScoreUpdateDatetime, v.LatestVersion}
		},
		"explore_id, max_score, max_score_update_datetime, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.AutoSaleSettings, after.AutoSaleSettings, "user_auto_sale_settings", "possession_auto_sale_item_type",
		func(v store.AutoSaleSettingState) []any {
			return []any{v.PossessionAutoSaleItemType, v.PossessionAutoSaleItemValue}
		},
		"possession_auto_sale_item_type, possession_auto_sale_item_value"); err != nil {
		return err
	}
	diffBoolMap(tx, uid, before.NaviCutInPlayed, after.NaviCutInPlayed, "user_navi_cutin_played", "navi_cutin_id")
	diffTimestampMap(tx, uid, before.ViewedMovies, after.ViewedMovies, "user_viewed_movies", "movie_id")
	diffTimestampMap(tx, uid, before.ContentsStories, after.ContentsStories, "user_contents_stories", "contents_story_id")
	diffTimestampMap(tx, uid, before.DrawnOmikuji, after.DrawnOmikuji, "user_drawn_omikuji", "omikuji_id")
	diffBoolMap(tx, uid, before.DokanConfirmed, after.DokanConfirmed, "user_dokan_confirmed", "dokan_id")

	exec(`DELETE FROM user_gifts WHERE user_id=?`, uid)
	for _, g := range after.Gifts.NotReceived {
		var expDt sql.NullInt64
		if g.ExpirationDatetime != 0 {
			expDt = sql.NullInt64{Int64: g.ExpirationDatetime, Valid: true}
		}
		exec(`INSERT INTO user_gifts (user_id, user_gift_uuid, is_received, possession_type, possession_id, count, grant_datetime, description_gift_text_id, equipment_data, expiration_datetime) VALUES (?,?,0,?,?,?,?,?,?,?)`,
			uid, g.UserGiftUuid, g.GiftCommon.PossessionType, g.GiftCommon.PossessionId, g.GiftCommon.Count, g.GiftCommon.GrantDatetime, g.GiftCommon.DescriptionGiftTextId, g.GiftCommon.EquipmentData, expDt)
	}
	for i, g := range after.Gifts.Received {
		uuid := fmt.Sprintf("received-%d-%d", uid, i)
		exec(`INSERT INTO user_gifts (user_id, user_gift_uuid, is_received, possession_type, possession_id, count, grant_datetime, description_gift_text_id, equipment_data, received_datetime) VALUES (?,?,1,?,?,?,?,?,?,?)`,
			uid, uuid, g.GiftCommon.PossessionType, g.GiftCommon.PossessionId, g.GiftCommon.Count, g.GiftCommon.GrantDatetime, g.GiftCommon.DescriptionGiftTextId, g.GiftCommon.EquipmentData, g.ReceivedDatetime)
	}

	exec(`DELETE FROM user_gacha_converted_medals WHERE user_id=?`, uid)
	for i, v := range after.Gacha.ConvertedGachaMedal.ConvertedMedalPossession {
		exec(`INSERT INTO user_gacha_converted_medals (user_id, ordinal, consumable_item_id, count) VALUES (?,?,?,?)`, uid, i, v.ConsumableItemId, v.Count)
	}

	for id, v := range after.Gacha.BannerStates {
		if old, ok := before.Gacha.BannerStates[id]; !ok || old.MedalCount != v.MedalCount || old.StepNumber != v.StepNumber || old.LoopCount != v.LoopCount || old.DrawCount != v.DrawCount || old.BoxNumber != v.BoxNumber {
			exec(`INSERT OR REPLACE INTO user_gacha_banners (user_id, gacha_id, medal_count, step_number, loop_count, draw_count, box_number) VALUES (?,?,?,?,?,?,?)`,
				uid, v.GachaId, v.MedalCount, v.StepNumber, v.LoopCount, v.DrawCount, v.BoxNumber)
		}
		exec(`DELETE FROM user_gacha_banner_box_drew_counts WHERE user_id=? AND gacha_id=?`, uid, id)
		for itemId, count := range v.BoxDrewCounts {
			exec(`INSERT INTO user_gacha_banner_box_drew_counts (user_id, gacha_id, box_item_id, count) VALUES (?,?,?,?)`, uid, id, itemId, count)
		}
	}
	for id := range before.Gacha.BannerStates {
		if _, ok := after.Gacha.BannerStates[id]; !ok {
			exec(`DELETE FROM user_gacha_banners WHERE user_id=? AND gacha_id=?`, uid, id)
			exec(`DELETE FROM user_gacha_banner_box_drew_counts WHERE user_id=? AND gacha_id=?`, uid, id)
		}
	}

	if err := diffMapInt32(tx, uid, before.CharacterBoards, after.CharacterBoards, "user_character_boards", "character_board_id",
		func(v store.CharacterBoardState) []any {
			return []any{v.CharacterBoardId, v.PanelReleaseBit1, v.PanelReleaseBit2, v.PanelReleaseBit3, v.PanelReleaseBit4, v.LatestVersion}
		}, "character_board_id, panel_release_bit1, panel_release_bit2, panel_release_bit3, panel_release_bit4, latest_version"); err != nil {
		return err
	}

	for k, v := range after.CharacterBoardAbilities {
		if old, ok := before.CharacterBoardAbilities[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_character_board_abilities (user_id, character_id, ability_id, level, latest_version) VALUES (?,?,?,?,?)`,
				uid, k.CharacterId, k.AbilityId, v.Level, v.LatestVersion)
		}
	}
	for k := range before.CharacterBoardAbilities {
		if _, ok := after.CharacterBoardAbilities[k]; !ok {
			exec(`DELETE FROM user_character_board_abilities WHERE user_id=? AND character_id=? AND ability_id=?`, uid, k.CharacterId, k.AbilityId)
		}
	}

	for k, v := range after.CharacterBoardStatusUps {
		if old, ok := before.CharacterBoardStatusUps[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_character_board_status_ups (user_id, character_id, status_calculation_type, hp, attack, vitality, agility, critical_ratio, critical_attack, latest_version) VALUES (?,?,?,?,?,?,?,?,?,?)`,
				uid, k.CharacterId, k.StatusCalculationType, v.Hp, v.Attack, v.Vitality, v.Agility, v.CriticalRatio, v.CriticalAttack, v.LatestVersion)
		}
	}
	for k := range before.CharacterBoardStatusUps {
		if _, ok := after.CharacterBoardStatusUps[k]; !ok {
			exec(`DELETE FROM user_character_board_status_ups WHERE user_id=? AND character_id=? AND status_calculation_type=?`, uid, k.CharacterId, k.StatusCalculationType)
		}
	}
	if err := diffMapInt32(tx, uid, before.CharacterRebirths, after.CharacterRebirths, "user_character_rebirths", "character_id",
		func(v store.CharacterRebirthState) []any {
			return []any{v.CharacterId, v.RebirthCount, v.LatestVersion}
		},
		"character_id, rebirth_count, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.CageOrnamentRewards, after.CageOrnamentRewards, "user_cage_ornament_rewards", "cage_ornament_id",
		func(v store.CageOrnamentRewardState) []any {
			return []any{v.CageOrnamentId, v.AcquisitionDatetime, v.LatestVersion}
		},
		"cage_ornament_id, acquisition_datetime, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.CageOrnamentAccesses, after.CageOrnamentAccesses, "user_cage_ornament_accesses", "cage_ornament_id",
		func(v store.CageOrnamentAccessState) []any {
			return []any{v.CageOrnamentId, v.FirstAccessDatetime, v.LatestAccessDatetime, v.LatestVersion}
		},
		"cage_ornament_id, first_access_datetime, latest_access_datetime, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.TowerAccumulationRewards, after.TowerAccumulationRewards, "user_event_quest_tower_accumulation_rewards", "event_quest_chapter_id",
		func(v store.TowerAccumulationRewardState) []any {
			return []any{v.EventQuestChapterId, v.LatestRewardReceiveQuestMissionClearCount, v.LatestVersion}
		},
		"event_quest_chapter_id, latest_reward_receive_quest_mission_clear_count, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.LabyrinthSeasons, after.LabyrinthSeasons, "user_event_quest_labyrinth_seasons", "event_quest_chapter_id",
		func(v store.LabyrinthSeasonState) []any {
			return []any{v.EventQuestChapterId, v.LastJoinSeasonNumber, v.LastSeasonRewardReceivedSeasonNumber, v.LatestVersion}
		},
		"event_quest_chapter_id, last_join_season_number, last_season_reward_received_season_number, latest_version"); err != nil {
		return err
	}
	for k, v := range after.LabyrinthStages {
		if old, ok := before.LabyrinthStages[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_event_quest_labyrinth_stages (user_id, event_quest_chapter_id, stage_order, is_received_stage_clear_reward, accumulation_reward_received_quest_mission_count, latest_version) VALUES (?,?,?,?,?,?)`,
				uid, k.EventQuestChapterId, k.StageOrder, boolToInt(v.IsReceivedStageClearReward), v.AccumulationRewardReceivedQuestMissionCount, v.LatestVersion)
		}
	}
	for k := range before.LabyrinthStages {
		if _, ok := after.LabyrinthStages[k]; !ok {
			exec(`DELETE FROM user_event_quest_labyrinth_stages WHERE user_id=? AND event_quest_chapter_id=? AND stage_order=?`, uid, k.EventQuestChapterId, k.StageOrder)
		}
	}
	if err := diffMapInt32(tx, uid, before.ShopItems, after.ShopItems, "user_shop_items", "shop_item_id",
		func(v store.UserShopItemState) []any {
			return []any{v.ShopItemId, v.BoughtCount, v.LatestBoughtCountChangedDatetime, v.LatestVersion}
		}, "shop_item_id, bought_count, latest_bought_count_changed_datetime, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.ShopReplaceableLineup, after.ShopReplaceableLineup, "user_shop_replaceable_lineup", "slot_number",
		func(v store.UserShopReplaceableLineupState) []any {
			return []any{v.SlotNumber, v.ShopItemId, v.LatestVersion}
		},
		"slot_number, shop_item_id, latest_version"); err != nil {
		return err
	}

	for k, v := range after.Gimmick.Progress {
		if old, ok := before.Gimmick.Progress[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_gimmick_progress (user_id, gimmick_sequence_schedule_id, gimmick_sequence_id, gimmick_id, is_gimmick_cleared, start_datetime, latest_version) VALUES (?,?,?,?,?,?,?)`,
				uid, k.GimmickSequenceScheduleId, k.GimmickSequenceId, k.GimmickId, boolToInt(v.IsGimmickCleared), v.StartDatetime, v.LatestVersion)
		}
	}
	for k := range before.Gimmick.Progress {
		if _, ok := after.Gimmick.Progress[k]; !ok {
			exec(`DELETE FROM user_gimmick_progress WHERE user_id=? AND gimmick_sequence_schedule_id=? AND gimmick_sequence_id=? AND gimmick_id=?`,
				uid, k.GimmickSequenceScheduleId, k.GimmickSequenceId, k.GimmickId)
		}
	}
	for k, v := range after.Gimmick.OrnamentProgress {
		if old, ok := before.Gimmick.OrnamentProgress[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_gimmick_ornament_progress (user_id, gimmick_sequence_schedule_id, gimmick_sequence_id, gimmick_id, gimmick_ornament_index, progress_value_bit, base_datetime, latest_version) VALUES (?,?,?,?,?,?,?,?)`,
				uid, k.GimmickSequenceScheduleId, k.GimmickSequenceId, k.GimmickId, k.GimmickOrnamentIndex, v.ProgressValueBit, v.BaseDatetime, v.LatestVersion)
		}
	}
	for k := range before.Gimmick.OrnamentProgress {
		if _, ok := after.Gimmick.OrnamentProgress[k]; !ok {
			exec(`DELETE FROM user_gimmick_ornament_progress WHERE user_id=? AND gimmick_sequence_schedule_id=? AND gimmick_sequence_id=? AND gimmick_id=? AND gimmick_ornament_index=?`,
				uid, k.GimmickSequenceScheduleId, k.GimmickSequenceId, k.GimmickId, k.GimmickOrnamentIndex)
		}
	}
	for k, v := range after.Gimmick.Sequences {
		if old, ok := before.Gimmick.Sequences[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_gimmick_sequences (user_id, gimmick_sequence_schedule_id, gimmick_sequence_id, is_gimmick_sequence_cleared, clear_datetime, latest_version) VALUES (?,?,?,?,?,?)`,
				uid, k.GimmickSequenceScheduleId, k.GimmickSequenceId, boolToInt(v.IsGimmickSequenceCleared), v.ClearDatetime, v.LatestVersion)
		}
	}
	for k := range before.Gimmick.Sequences {
		if _, ok := after.Gimmick.Sequences[k]; !ok {
			exec(`DELETE FROM user_gimmick_sequences WHERE user_id=? AND gimmick_sequence_schedule_id=? AND gimmick_sequence_id=?`,
				uid, k.GimmickSequenceScheduleId, k.GimmickSequenceId)
		}
	}
	for k, v := range after.Gimmick.Unlocks {
		if old, ok := before.Gimmick.Unlocks[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_gimmick_unlocks (user_id, gimmick_sequence_schedule_id, gimmick_sequence_id, gimmick_id, is_unlocked, latest_version) VALUES (?,?,?,?,?,?)`,
				uid, k.GimmickSequenceScheduleId, k.GimmickSequenceId, k.GimmickId, boolToInt(v.IsUnlocked), v.LatestVersion)
		}
	}
	for k := range before.Gimmick.Unlocks {
		if _, ok := after.Gimmick.Unlocks[k]; !ok {
			exec(`DELETE FROM user_gimmick_unlocks WHERE user_id=? AND gimmick_sequence_schedule_id=? AND gimmick_sequence_id=? AND gimmick_id=?`,
				uid, k.GimmickSequenceScheduleId, k.GimmickSequenceId, k.GimmickId)
		}
	}

	if err := diffMapInt32(tx, uid, before.BigHuntMaxScores, after.BigHuntMaxScores, "user_big_hunt_max_scores", "big_hunt_boss_id",
		func(v store.BigHuntMaxScore) []any {
			return []any{0, v.MaxScore, v.MaxScoreUpdateDatetime, v.LatestVersion}
		},
		"big_hunt_boss_id, max_score, max_score_update_datetime, latest_version"); err != nil {
		return err
	}
	if err := diffMapInt32(tx, uid, before.BigHuntStatuses, after.BigHuntStatuses, "user_big_hunt_statuses", "big_hunt_boss_id",
		func(v store.BigHuntStatus) []any {
			return []any{0, v.DailyChallengeCount, v.LatestChallengeDatetime, v.LastDailyRewardReceivedDayVersion, v.LatestVersion}
		},
		"big_hunt_boss_id, daily_challenge_count, latest_challenge_datetime, last_daily_reward_received_day_version, latest_version"); err != nil {
		return err
	}

	for k, v := range after.BigHuntScheduleMaxScores {
		if old, ok := before.BigHuntScheduleMaxScores[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_big_hunt_schedule_max_scores (user_id, big_hunt_schedule_id, big_hunt_boss_id, max_score, max_score_update_datetime, latest_version) VALUES (?,?,?,?,?,?)`,
				uid, k.BigHuntScheduleId, k.BigHuntBossId, v.MaxScore, v.MaxScoreUpdateDatetime, v.LatestVersion)
		}
	}
	for k := range before.BigHuntScheduleMaxScores {
		if _, ok := after.BigHuntScheduleMaxScores[k]; !ok {
			exec(`DELETE FROM user_big_hunt_schedule_max_scores WHERE user_id=? AND big_hunt_schedule_id=? AND big_hunt_boss_id=?`, uid, k.BigHuntScheduleId, k.BigHuntBossId)
		}
	}
	for k, v := range after.BigHuntWeeklyMaxScores {
		if old, ok := before.BigHuntWeeklyMaxScores[k]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_big_hunt_weekly_max_scores (user_id, big_hunt_weekly_version, attribute_type, max_score, latest_version) VALUES (?,?,?,?,?)`,
				uid, k.BigHuntWeeklyVersion, k.AttributeType, v.MaxScore, v.LatestVersion)
		}
	}
	for k := range before.BigHuntWeeklyMaxScores {
		if _, ok := after.BigHuntWeeklyMaxScores[k]; !ok {
			exec(`DELETE FROM user_big_hunt_weekly_max_scores WHERE user_id=? AND big_hunt_weekly_version=? AND attribute_type=?`, uid, k.BigHuntWeeklyVersion, k.AttributeType)
		}
	}
	for ver, v := range after.BigHuntWeeklyStatuses {
		if old, ok := before.BigHuntWeeklyStatuses[ver]; !ok || old != v {
			exec(`INSERT OR REPLACE INTO user_big_hunt_weekly_statuses (user_id, big_hunt_weekly_version, is_received_weekly_reward, latest_version) VALUES (?,?,?,?)`,
				uid, ver, boolToInt(v.IsReceivedWeeklyReward), v.LatestVersion)
		}
	}
	for ver := range before.BigHuntWeeklyStatuses {
		if _, ok := after.BigHuntWeeklyStatuses[ver]; !ok {
			exec(`DELETE FROM user_big_hunt_weekly_statuses WHERE user_id=? AND big_hunt_weekly_version=?`, uid, ver)
		}
	}

	return firstErr
}

func bigHuntBattleDetailEqual(a, b store.BigHuntBattleDetail) bool {
	if a.DeckType != b.DeckType || a.UserTripleDeckNumber != b.UserTripleDeckNumber ||
		a.BossKnockDownCount != b.BossKnockDownCount || a.MaxComboCount != b.MaxComboCount ||
		a.TotalDamage != b.TotalDamage {
		return false
	}
	if len(a.CostumeBattleInfo) != len(b.CostumeBattleInfo) {
		return false
	}
	for i := range a.CostumeBattleInfo {
		if a.CostumeBattleInfo[i] != b.CostumeBattleInfo[i] {
			return false
		}
	}
	return true
}

func diffMapInt32[V comparable](tx *sql.Tx, uid int64, before, after map[int32]V, table, keyCol string, vals func(V) []any, cols string) error {
	for k, v := range after {
		if old, ok := before[k]; !ok || old != v {
			allVals := vals(v)
			allVals[0] = k
			args := append([]any{uid}, allVals...)
			placeholders := "?"
			for range allVals {
				placeholders += ",?"
			}
			if _, err := tx.Exec(fmt.Sprintf(`INSERT OR REPLACE INTO %s (user_id, %s) VALUES (%s)`, table, cols, placeholders), args...); err != nil {
				return err
			}
		}
	}
	for k := range before {
		if _, ok := after[k]; !ok {
			if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE user_id=? AND %s=?`, table, keyCol), uid, k); err != nil {
				return err
			}
		}
	}
	return nil
}

func diffMapStr[V comparable](tx *sql.Tx, uid int64, before, after map[string]V, table, keyCol string, vals func(V) []any, cols string) error {
	for k, v := range after {
		if old, ok := before[k]; !ok || old != v {
			allVals := vals(v)
			args := append([]any{uid}, allVals...)
			placeholders := "?"
			for range allVals {
				placeholders += ",?"
			}
			if _, err := tx.Exec(fmt.Sprintf(`INSERT OR REPLACE INTO %s (user_id, %s) VALUES (%s)`, table, cols, placeholders), args...); err != nil {
				return err
			}
		}
	}
	for k := range before {
		if _, ok := after[k]; !ok {
			if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE user_id=? AND %s=?`, table, keyCol), uid, k); err != nil {
				return err
			}
		}
	}
	return nil
}

func diffSimpleMap(tx *sql.Tx, uid int64, before, after map[int32]int32, table, keyCol, valCol string) error {
	for k, v := range after {
		if old, ok := before[k]; !ok || old != v {
			if _, err := tx.Exec(fmt.Sprintf(`INSERT OR REPLACE INTO %s (user_id, %s, %s) VALUES (?,?,?)`, table, keyCol, valCol), uid, k, v); err != nil {
				return err
			}
		}
	}
	for k := range before {
		if _, ok := after[k]; !ok {
			if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE user_id=? AND %s=?`, table, keyCol), uid, k); err != nil {
				return err
			}
		}
	}
	return nil
}

func diffInt64Map(tx *sql.Tx, uid int64, before, after map[int32]int64, table, keyCol, valCol string) error {
	for k, v := range after {
		if old, ok := before[k]; !ok || old != v {
			if _, err := tx.Exec(fmt.Sprintf(`INSERT OR REPLACE INTO %s (user_id, %s, %s) VALUES (?,?,?)`, table, keyCol, valCol), uid, k, v); err != nil {
				return err
			}
		}
	}
	for k := range before {
		if _, ok := after[k]; !ok {
			if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE user_id=? AND %s=?`, table, keyCol), uid, k); err != nil {
				return err
			}
		}
	}
	return nil
}

func diffBoolMap(tx *sql.Tx, uid int64, before, after map[int32]bool, table, keyCol string) {
	for k := range after {
		if !before[k] {
			tx.Exec(fmt.Sprintf(`INSERT OR IGNORE INTO %s (user_id, %s) VALUES (?,?)`, table, keyCol), uid, k)
		}
	}
	for k := range before {
		if !after[k] {
			tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE user_id=? AND %s=?`, table, keyCol), uid, k)
		}
	}
}

func diffTimestampMap(tx *sql.Tx, uid int64, before, after map[int32]int64, table, keyCol string) {
	for k, v := range after {
		if old, ok := before[k]; !ok || old != v {
			tx.Exec(fmt.Sprintf(`INSERT OR REPLACE INTO %s (user_id, %s, timestamp) VALUES (?,?,?)`, table, keyCol), uid, k, v)
		}
	}
	for k := range before {
		if _, ok := after[k]; !ok {
			tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE user_id=? AND %s=?`, table, keyCol), uid, k)
		}
	}
}

func replaceSliceTable(tx *sql.Tx, uid int64, table string, data map[string][]string, insertFn func(string, []string)) error {
	if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE user_id=?`, table), uid); err != nil {
		return err
	}
	for key, vals := range data {
		insertFn(key, vals)
	}
	return nil
}

// suppress unused import
var _ = model.DeckTypeQuest
