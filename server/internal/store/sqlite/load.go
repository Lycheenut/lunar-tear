package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func (s *SQLiteStore) LoadUser(userId int64) (store.UserState, error) {
	var u store.UserState
	var fbId sql.NullInt64

	err := s.db.QueryRow(`SELECT user_id, uuid, player_id, os_type, platform_type, user_restriction_type,
		register_datetime, game_start_datetime, latest_version, birth_year, birth_month,
		backup_token, charge_money_this_month, facebook_id FROM users WHERE user_id = ?`, userId).Scan(
		&u.UserId, &u.Uuid, &u.PlayerId, &u.OsType, &u.PlatformType, &u.UserRestrictionType,
		&u.RegisterDatetime, &u.GameStartDatetime, &u.LatestVersion, &u.BirthYear, &u.BirthMonth,
		&u.BackupToken, &u.ChargeMoneyThisMonth, &fbId)
	if err == sql.ErrNoRows {
		return u, store.ErrNotFound
	}
	if err != nil {
		return u, fmt.Errorf("load users: %w", err)
	}
	u.FacebookId = fbId.Int64

	initMaps(&u)

	load1to1(s.db, userId, &u)
	loadMechanismTables(s.db, userId, &u)
	loadMapTables(s.db, userId, &u)

	return u, nil
}

func initMaps(u *store.UserState) {
	u.LoginBonuses = make(map[int32]store.UserLoginBonusState)
	u.Friends = make(map[int64]store.FriendState)
	u.FriendRequests = make(map[int64]int64)
	u.Tutorials = make(map[int32]store.TutorialProgressState)
	u.Characters = make(map[int32]store.CharacterState)
	u.Costumes = make(map[string]store.CostumeState)
	u.Weapons = make(map[string]store.WeaponState)
	u.Companions = make(map[string]store.CompanionState)
	u.Thoughts = make(map[string]store.ThoughtState)
	u.DeckCharacters = make(map[string]store.DeckCharacterState)
	u.Decks = make(map[store.DeckKey]store.DeckState)
	u.TripleDecks = make(map[store.DeckKey]store.TripleDeckState)
	u.DeckLimitContentRestricted = make(map[string]store.DeckLimitContentRestrictedState)
	u.DeckSubWeapons = make(map[string][]string)
	u.DeckParts = make(map[string][]string)
	u.Quests = make(map[int32]store.UserQuestState)
	u.QuestMissions = make(map[store.QuestMissionKey]store.UserQuestMissionState)
	u.Missions = make(map[int32]store.UserMissionState)
	u.WeaponStories = make(map[int32]store.WeaponStoryState)
	u.WeaponNotes = make(map[int32]store.WeaponNoteState)
	u.WeaponSkills = make(map[string][]store.WeaponSkillState)
	u.WeaponAbilities = make(map[string][]store.WeaponAbilityState)
	u.WeaponAwakens = make(map[string]store.WeaponAwakenState)
	u.CostumeActiveSkills = make(map[string]store.CostumeActiveSkillState)
	u.CostumeAwakenStatusUps = make(map[store.CostumeAwakenStatusKey]store.CostumeAwakenStatusUpState)
	u.CostumeLevelBonusReleaseStatuses = make(map[int32]store.CostumeLevelBonusReleaseStatusState)
	u.CostumeLotteryEffects = make(map[store.CostumeLotteryEffectKey]store.CostumeLotteryEffectState)
	u.CostumeLotteryEffectAbilities = make(map[store.CostumeLotteryEffectKey]store.CostumeLotteryEffectAbilityState)
	u.CostumeLotteryEffectStatusUps = make(map[store.CostumeLotteryEffectStatusKey]store.CostumeLotteryEffectStatusUpState)
	u.CostumeLotteryEffectPending = make(map[string]store.CostumeLotteryEffectPendingState)
	u.Parts = make(map[string]store.PartsState)
	u.PartsGroupNotes = make(map[int32]store.PartsGroupNoteState)
	u.PartsPresets = make(map[int32]store.PartsPresetState)
	u.PartsPresetTags = make(map[int32]store.PartsPresetTagState)
	u.PartsStatusSubs = make(map[store.PartsStatusSubKey]store.PartsStatusSubState)
	u.DeckTypeNotes = make(map[model.DeckType]store.DeckTypeNoteState)
	u.ConsumableItems = make(map[int32]int32)
	u.Materials = make(map[int32]int32)
	u.ImportantItems = make(map[int32]int32)
	u.PremiumItems = make(map[int32]int64)
	u.NaviCutInPlayed = make(map[int32]bool)
	u.ViewedMovies = make(map[int32]int64)
	u.ContentsStories = make(map[int32]int64)
	u.DrawnOmikuji = make(map[int32]int64)
	u.DokanConfirmed = make(map[int32]bool)
	u.ShopItems = make(map[int32]store.UserShopItemState)
	u.ShopReplaceableLineup = make(map[int32]store.UserShopReplaceableLineupState)
	u.ExploreScores = make(map[int32]store.ExploreScoreState)
	u.CageOrnamentRewards = make(map[int32]store.CageOrnamentRewardState)
	u.CageOrnamentAccesses = make(map[int32]store.CageOrnamentAccessState)
	u.TowerAccumulationRewards = make(map[int32]store.TowerAccumulationRewardState)
	u.LabyrinthSeasons = make(map[int32]store.LabyrinthSeasonState)
	u.LabyrinthStages = make(map[store.LabyrinthStageKey]store.LabyrinthStageState)
	u.CharacterBoards = make(map[int32]store.CharacterBoardState)
	u.CharacterBoardAbilities = make(map[store.CharacterBoardAbilityKey]store.CharacterBoardAbilityState)
	u.CharacterBoardStatusUps = make(map[store.CharacterBoardStatusUpKey]store.CharacterBoardStatusUpState)
	u.CharacterRebirths = make(map[int32]store.CharacterRebirthState)
	u.AutoSaleSettings = make(map[int32]store.AutoSaleSettingState)
	u.SideStoryQuests = make(map[int32]store.SideStoryQuestProgress)
	u.QuestLimitContentStatus = make(map[int32]store.QuestLimitContentStatus)
	u.BigHuntMaxScores = make(map[int32]store.BigHuntMaxScore)
	u.BigHuntStatuses = make(map[int32]store.BigHuntStatus)
	u.BigHuntScheduleMaxScores = make(map[store.BigHuntScheduleScoreKey]store.BigHuntScheduleMaxScore)
	u.BigHuntWeeklyMaxScores = make(map[store.BigHuntWeeklyScoreKey]store.BigHuntWeeklyMaxScore)
	u.BigHuntWeeklyStatuses = make(map[int64]store.BigHuntWeeklyStatus)
	u.Gacha.BannerStates = make(map[int32]store.GachaBannerState)
	u.Gacha.ConvertedGachaMedal.ConvertedMedalPossession = []store.ConsumableItemState{}
	u.Gifts.NotReceived = []store.NotReceivedGiftState{}
	u.Gifts.Received = []store.ReceivedGiftState{}
	u.Gimmick.Progress = make(map[store.GimmickKey]store.GimmickProgressState)
	u.Gimmick.OrnamentProgress = make(map[store.GimmickOrnamentKey]store.GimmickOrnamentProgressState)
	u.Gimmick.Sequences = make(map[store.GimmickSequenceKey]store.GimmickSequenceState)
	u.Gimmick.Unlocks = make(map[store.GimmickKey]store.GimmickUnlockState)
	u.EnsureMaps()
}

func load1to1(db *sql.DB, uid int64, u *store.UserState) {
	var b int
	_ = db.QueryRow(`SELECT is_notify_purchase_alert, latest_version FROM user_setting WHERE user_id=?`, uid).
		Scan(&b, &u.Setting.LatestVersion)
	u.Setting.IsNotifyPurchaseAlert = b != 0

	_ = db.QueryRow(`SELECT level, exp, stamina_milli_value, stamina_update_datetime, latest_version FROM user_status WHERE user_id=?`, uid).
		Scan(&u.Status.Level, &u.Status.Exp, &u.Status.StaminaMilliValue, &u.Status.StaminaUpdateDatetime, &u.Status.LatestVersion)

	_ = db.QueryRow(`SELECT paid_gem, free_gem FROM user_gem WHERE user_id=?`, uid).
		Scan(&u.Gem.PaidGem, &u.Gem.FreeGem)

	_ = db.QueryRow(`SELECT name, name_update_datetime, message, message_update_datetime, favorite_costume_id,
		favorite_costume_id_update_datetime, current_pvp_rank, current_pvp_grade_id,
		max_pvp_season_rank, latest_version FROM user_profile WHERE user_id=?`, uid).
		Scan(&u.Profile.Name, &u.Profile.NameUpdateDatetime, &u.Profile.Message, &u.Profile.MessageUpdateDatetime,
			&u.Profile.FavoriteCostumeId, &u.Profile.FavoriteCostumeIdUpdateDatetime,
			&u.Profile.CurrentPvpRank, &u.Profile.CurrentPvpGradeId, &u.Profile.MaxPvpSeasonRank,
			&u.Profile.LatestVersion)

	_ = db.QueryRow(`SELECT total_login_count, continual_login_count, max_continual_login_count,
		last_login_datetime, last_comeback_login_datetime, latest_version FROM user_login WHERE user_id=?`, uid).
		Scan(&u.Login.TotalLoginCount, &u.Login.ContinualLoginCount, &u.Login.MaxContinualLoginCount,
			&u.Login.LastLoginDatetime, &u.Login.LastComebackLoginDatetime, &u.Login.LatestVersion)

	_ = db.QueryRow(`SELECT beginner_campaign_id, campaign_register_datetime, latest_version
		FROM user_beginner_campaign WHERE user_id=?`, uid).
		Scan(&u.BeginnerCampaign.BeginnerCampaignId, &u.BeginnerCampaign.CampaignRegisterDatetime,
			&u.BeginnerCampaign.LatestVersion)

	_ = db.QueryRow(`SELECT comeback_campaign_id, comeback_datetime, latest_version
		FROM user_comeback_campaign WHERE user_id=?`, uid).
		Scan(&u.ComebackCampaign.ComebackCampaignId, &u.ComebackCampaign.ComebackDatetime,
			&u.ComebackCampaign.LatestVersion)

	var ctxActive, ctxIsLast, ctxCage int
	_ = db.QueryRow(`SELECT current_quest_flow_type, current_main_quest_route_id, current_quest_scene_id,
		head_quest_scene_id, is_reached_last_quest_scene, progress_quest_scene_id, progress_head_quest_scene_id,
		progress_quest_flow_type, main_quest_season_id, latest_version,
		saved_ctx_active, saved_ctx_current_quest_scene_id, saved_ctx_head_quest_scene_id,
		saved_ctx_current_main_quest_route_id, saved_ctx_main_quest_season_id,
		saved_ctx_is_reached_last_quest_scene, saved_ctx_portal_cage_in_progress, saved_ctx_current_quest_flow_type,
		replay_flow_current_quest_scene_id, replay_flow_head_quest_scene_id
		FROM user_main_quest WHERE user_id=?`, uid).
		Scan(&u.MainQuest.CurrentQuestFlowType, &u.MainQuest.CurrentMainQuestRouteId, &u.MainQuest.CurrentQuestSceneId,
			&u.MainQuest.HeadQuestSceneId, &b, &u.MainQuest.ProgressQuestSceneId, &u.MainQuest.ProgressHeadQuestSceneId,
			&u.MainQuest.ProgressQuestFlowType, &u.MainQuest.MainQuestSeasonId, &u.MainQuest.LatestVersion,
			&ctxActive, &u.MainQuest.SavedContext.CurrentQuestSceneId, &u.MainQuest.SavedContext.HeadQuestSceneId,
			&u.MainQuest.SavedContext.CurrentMainQuestRouteId, &u.MainQuest.SavedContext.MainQuestSeasonId,
			&ctxIsLast, &ctxCage, &u.MainQuest.SavedContext.CurrentQuestFlowType,
			&u.MainQuest.ReplayFlowCurrentQuestSceneId, &u.MainQuest.ReplayFlowHeadQuestSceneId)
	u.MainQuest.IsReachedLastQuestScene = b != 0
	u.MainQuest.SavedContext.Active = ctxActive != 0
	u.MainQuest.SavedContext.IsReachedLastQuestScene = ctxIsLast != 0
	u.MainQuest.SavedContext.PortalCageInProgress = ctxCage != 0

	_ = db.QueryRow(`SELECT current_event_quest_chapter_id, current_quest_id, current_quest_scene_id,
		head_quest_scene_id, latest_version FROM user_event_quest WHERE user_id=?`, uid).
		Scan(&u.EventQuest.CurrentEventQuestChapterId, &u.EventQuest.CurrentQuestId,
			&u.EventQuest.CurrentQuestSceneId, &u.EventQuest.HeadQuestSceneId, &u.EventQuest.LatestVersion)

	_ = db.QueryRow(`SELECT current_quest_id, current_quest_scene_id, head_quest_scene_id, latest_version
		FROM user_extra_quest WHERE user_id=?`, uid).
		Scan(&u.ExtraQuest.CurrentQuestId, &u.ExtraQuest.CurrentQuestSceneId,
			&u.ExtraQuest.HeadQuestSceneId, &u.ExtraQuest.LatestVersion)

	_ = db.QueryRow(`SELECT current_side_story_quest_id, current_side_story_quest_scene_id, latest_version
		FROM user_side_story_active WHERE user_id=?`, uid).
		Scan(&u.SideStoryActiveProgress.CurrentSideStoryQuestId,
			&u.SideStoryActiveProgress.CurrentSideStoryQuestSceneId, &u.SideStoryActiveProgress.LatestVersion)

	var isDryRun int
	_ = db.QueryRow(`SELECT current_big_hunt_boss_quest_id, current_big_hunt_quest_id, current_quest_scene_id,
		is_dry_run, latest_version, deck_type, user_triple_deck_number, boss_knock_down_count,
		max_combo_count, total_damage, deck_number, battle_binary
		FROM user_big_hunt_state WHERE user_id=?`, uid).
		Scan(&u.BigHuntProgress.CurrentBigHuntBossQuestId, &u.BigHuntProgress.CurrentBigHuntQuestId,
			&u.BigHuntProgress.CurrentQuestSceneId, &isDryRun, &u.BigHuntProgress.LatestVersion,
			&u.BigHuntBattleDetail.DeckType, &u.BigHuntBattleDetail.UserTripleDeckNumber,
			&u.BigHuntBattleDetail.BossKnockDownCount, &u.BigHuntBattleDetail.MaxComboCount,
			&u.BigHuntBattleDetail.TotalDamage, &u.BigHuntDeckNumber, &u.BigHuntBattleBinary)
	u.BigHuntProgress.IsDryRun = isDryRun != 0

	queryRows(db, `SELECT wave_index, costume_id, total_damage, hit_count, random_display_value_type, random_display_value
		FROM user_big_hunt_costume_battle_infos WHERE user_id=? ORDER BY wave_index, sort_order`, uid, func(rows *sql.Rows) {
		var ci store.BigHuntCostumeBattleInfo
		rows.Scan(&ci.WaveIndex, &ci.CostumeId, &ci.TotalDamage, &ci.HitCount, &ci.RandomDisplayValueType, &ci.RandomDisplayValue)
		u.BigHuntBattleDetail.CostumeBattleInfo = append(u.BigHuntBattleDetail.CostumeBattleInfo, ci)
	})

	var isActive, isUnread int
	_ = db.QueryRow(`SELECT is_active, start_count, finish_count, last_started_at, last_finished_at,
		last_user_party_count, last_npc_party_count, last_battle_binary_size, last_elapsed_frame_count, battle_binary
		FROM user_battle WHERE user_id=?`, uid).
		Scan(&isActive, &u.Battle.StartCount, &u.Battle.FinishCount, &u.Battle.LastStartedAt,
			&u.Battle.LastFinishedAt, &u.Battle.LastUserPartyCount, &u.Battle.LastNpcPartyCount,
			&u.Battle.LastBattleBinarySize, &u.Battle.LastElapsedFrameCount, &u.BattleBinary)
	u.Battle.IsActive = isActive != 0

	_ = db.QueryRow(`SELECT gift_not_receive_count, friend_request_receive_count, is_exist_unread_information
		FROM user_notification WHERE user_id=?`, uid).
		Scan(&u.Notifications.GiftNotReceiveCount, &u.Notifications.FriendRequestReceiveCount, &isUnread)
	u.Notifications.IsExistUnreadInformation = isUnread != 0

	var isCP int
	_ = db.QueryRow(`SELECT is_current_progress, drop_item_start_datetime, current_drop_item_count, latest_version
		FROM user_portal_cage WHERE user_id=?`, uid).
		Scan(&isCP, &u.PortalCageStatus.DropItemStartDatetime, &u.PortalCageStatus.CurrentDropItemCount,
			&u.PortalCageStatus.LatestVersion)
	u.PortalCageStatus.IsCurrentProgress = isCP != 0

	_ = db.QueryRow(`SELECT start_datetime, open_minutes, daily_opened_count, latest_version
		FROM user_guerrilla_free_open WHERE user_id=?`, uid).
		Scan(&u.GuerrillaFreeOpen.StartDatetime, &u.GuerrillaFreeOpen.OpenMinutes,
			&u.GuerrillaFreeOpen.DailyOpenedCount, &u.GuerrillaFreeOpen.LatestVersion)

	var accumulatedDropsJSON string
	_ = db.QueryRow(`SELECT quest_type, chapter_id, quest_id, max_auto_orbit_count, cleared_auto_orbit_count, last_clear_datetime, latest_version, accumulated_drops_json
		FROM user_quest_auto_orbit WHERE user_id=?`, uid).
		Scan(&u.QuestAutoOrbit.QuestType, &u.QuestAutoOrbit.ChapterId, &u.QuestAutoOrbit.QuestId,
			&u.QuestAutoOrbit.MaxAutoOrbitCount, &u.QuestAutoOrbit.ClearedAutoOrbitCount,
			&u.QuestAutoOrbit.LastClearDatetime, &u.QuestAutoOrbit.LatestVersion,
			&accumulatedDropsJSON)
	if accumulatedDropsJSON != "" && accumulatedDropsJSON != "[]" {
		_ = json.Unmarshal([]byte(accumulatedDropsJSON), &u.QuestAutoOrbit.AccumulatedDrops)
	}

	var isTicket int
	_ = db.QueryRow(`SELECT is_use_explore_ticket, playing_explore_id, latest_play_datetime, latest_version
		FROM user_explore WHERE user_id=?`, uid).
		Scan(&isTicket, &u.Explore.PlayingExploreId, &u.Explore.LatestPlayDatetime, &u.Explore.LatestVersion)
	u.Explore.IsUseExploreTicket = isTicket != 0

	_ = db.QueryRow(`SELECT lineup_update_count, latest_lineup_update_datetime, latest_version
		FROM user_shop_replaceable WHERE user_id=?`, uid).
		Scan(&u.ShopReplaceable.LineupUpdateCount, &u.ShopReplaceable.LatestLineupUpdateDatetime,
			&u.ShopReplaceable.LatestVersion)

	var rewardAvail int
	var obtainItemId, obtainCount sql.NullInt64
	_ = db.QueryRow(`SELECT reward_available, todays_current_draw_count, daily_max_count,
		last_reward_draw_date, obtain_consumable_item_id, obtain_count
		FROM user_gacha WHERE user_id=?`, uid).
		Scan(&rewardAvail, &u.Gacha.TodaysCurrentDrawCount, &u.Gacha.DailyMaxCount,
			&u.Gacha.LastRewardDrawDate, &obtainItemId, &obtainCount)
	u.Gacha.RewardAvailable = rewardAvail != 0
	if obtainItemId.Valid {
		u.Gacha.ConvertedGachaMedal.ObtainPossession = &store.ConsumableItemState{
			ConsumableItemId: int32(obtainItemId.Int64),
			Count:            int32(obtainCount.Int64),
		}
	}

}

func loadMapTables(db *sql.DB, uid int64, u *store.UserState) {
	queryRows(db, `SELECT login_bonus_id, current_page_number, current_stamp_number,
		latest_reward_receive_datetime, latest_version FROM user_login_bonus WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.UserLoginBonusState
			rows.Scan(&v.LoginBonusId, &v.CurrentPageNumber, &v.CurrentStampNumber,
				&v.LatestRewardReceiveDatetime, &v.LatestVersion)
			u.LoginBonuses[v.LoginBonusId] = v
		})

	queryRows(db, `SELECT friend_user_id, is_friend, cheer_sent_datetime, cheer_received_datetime,
		stamina_received_datetime FROM user_friends WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var friendUserId int64
			var isFriend int
			var friend store.FriendState
			rows.Scan(&friendUserId, &isFriend, &friend.CheerSentDatetime, &friend.CheerReceivedDatetime,
				&friend.StaminaReceivedDatetime)
			friend.IsFriend = isFriend != 0
			u.Friends[friendUserId] = friend
		})
	queryRows(db, `SELECT requester_user_id, request_datetime FROM user_friend_requests WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var requesterUserId, requestDatetime int64
			rows.Scan(&requesterUserId, &requestDatetime)
			u.FriendRequests[requesterUserId] = requestDatetime
		})

	queryRows(db, `SELECT character_id, level, exp, latest_version FROM user_characters WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.CharacterState
			rows.Scan(&v.CharacterId, &v.Level, &v.Exp, &v.LatestVersion)
			u.Characters[v.CharacterId] = v
		})

	queryRows(db, `SELECT user_costume_uuid, costume_id, limit_break_count, level, exp,
		headup_display_view_id, acquisition_datetime, awaken_count,
		costume_lottery_effect_unlocked_slot_count, latest_version
		FROM user_costumes WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.CostumeState
		rows.Scan(&v.UserCostumeUuid, &v.CostumeId, &v.LimitBreakCount, &v.Level, &v.Exp,
			&v.HeadupDisplayViewId, &v.AcquisitionDatetime, &v.AwakenCount,
			&v.CostumeLotteryEffectUnlockedSlotCount, &v.LatestVersion)
		u.Costumes[v.UserCostumeUuid] = v
	})

	queryRows(db, `SELECT user_weapon_uuid, weapon_id, level, exp, limit_break_count,
		is_protected, acquisition_datetime, latest_version FROM user_weapons WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.WeaponState
			var prot int
			rows.Scan(&v.UserWeaponUuid, &v.WeaponId, &v.Level, &v.Exp, &v.LimitBreakCount,
				&prot, &v.AcquisitionDatetime, &v.LatestVersion)
			v.IsProtected = prot != 0
			u.Weapons[v.UserWeaponUuid] = v
		})

	queryRows(db, `SELECT user_companion_uuid, companion_id, headup_display_view_id, level,
		acquisition_datetime, latest_version FROM user_companions WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.CompanionState
			rows.Scan(&v.UserCompanionUuid, &v.CompanionId, &v.HeadupDisplayViewId, &v.Level,
				&v.AcquisitionDatetime, &v.LatestVersion)
			u.Companions[v.UserCompanionUuid] = v
		})

	queryRows(db, `SELECT user_thought_uuid, thought_id, acquisition_datetime, latest_version
		FROM user_thoughts WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.ThoughtState
		rows.Scan(&v.UserThoughtUuid, &v.ThoughtId, &v.AcquisitionDatetime, &v.LatestVersion)
		u.Thoughts[v.UserThoughtUuid] = v
	})

	queryRows(db, `SELECT user_deck_character_uuid, user_costume_uuid, main_user_weapon_uuid,
		user_companion_uuid, power, user_thought_uuid, dressup_costume_id, latest_version
		FROM user_deck_characters WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.DeckCharacterState
		rows.Scan(&v.UserDeckCharacterUuid, &v.UserCostumeUuid, &v.MainUserWeaponUuid,
			&v.UserCompanionUuid, &v.Power, &v.UserThoughtUuid, &v.DressupCostumeId, &v.LatestVersion)
		u.DeckCharacters[v.UserDeckCharacterUuid] = v
	})

	queryRows(db, `SELECT deck_type, user_deck_number, user_deck_character_uuid01, user_deck_character_uuid02,
		user_deck_character_uuid03, name, power, latest_version FROM user_decks WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.DeckState
			var dt int32
			rows.Scan(&dt, &v.UserDeckNumber, &v.UserDeckCharacterUuid01, &v.UserDeckCharacterUuid02,
				&v.UserDeckCharacterUuid03, &v.Name, &v.Power, &v.LatestVersion)
			v.DeckType = model.DeckType(dt)
			u.Decks[store.DeckKey{DeckType: v.DeckType, UserDeckNumber: v.UserDeckNumber}] = v
		})

	queryRows(db, `SELECT deck_type, user_deck_number, name, deck_number01, deck_number02, deck_number03, latest_version
		FROM user_triple_decks WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.TripleDeckState
			var dt int32
			rows.Scan(&dt, &v.UserDeckNumber, &v.Name, &v.DeckNumber01, &v.DeckNumber02, &v.DeckNumber03, &v.LatestVersion)
			v.DeckType = model.DeckType(dt)
			u.TripleDecks[store.DeckKey{DeckType: v.DeckType, UserDeckNumber: v.UserDeckNumber}] = v
		})

	queryRows(db, `SELECT deck_restricted_uuid, event_quest_chapter_id, quest_id, possession_type,
		target_uuid, latest_version FROM user_deck_limit_content_restricted WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.DeckLimitContentRestrictedState
			rows.Scan(&v.DeckRestrictedUuid, &v.EventQuestChapterId, &v.QuestId, &v.PossessionType,
				&v.TargetUuid, &v.LatestVersion)
			u.DeckLimitContentRestricted[v.DeckRestrictedUuid] = v
		})

	queryRows(db, `SELECT user_deck_character_uuid, ordinal, user_weapon_uuid
		FROM user_deck_sub_weapons WHERE user_id=? ORDER BY user_deck_character_uuid, ordinal`, uid,
		func(rows *sql.Rows) {
			var key, val string
			var ord int
			rows.Scan(&key, &ord, &val)
			u.DeckSubWeapons[key] = append(u.DeckSubWeapons[key], val)
		})

	queryRows(db, `SELECT user_deck_character_uuid, ordinal, user_parts_uuid
		FROM user_deck_parts WHERE user_id=? ORDER BY user_deck_character_uuid, ordinal`, uid,
		func(rows *sql.Rows) {
			var key, val string
			var ord int
			rows.Scan(&key, &ord, &val)
			u.DeckParts[key] = append(u.DeckParts[key], val)
		})

	queryRows(db, `SELECT quest_id, quest_state_type, is_battle_only, user_deck_number, latest_start_datetime,
		clear_count, daily_clear_count, last_clear_datetime, shortest_clear_frames, is_reward_granted, latest_version
		FROM user_quests WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.UserQuestState
		var bo, rg int
		rows.Scan(&v.QuestId, &v.QuestStateType, &bo, &v.UserDeckNumber, &v.LatestStartDatetime,
			&v.ClearCount, &v.DailyClearCount, &v.LastClearDatetime, &v.ShortestClearFrames, &rg, &v.LatestVersion)
		v.IsBattleOnly = bo != 0
		v.IsRewardGranted = rg != 0
		u.Quests[v.QuestId] = v
	})

	queryRows(db, `SELECT quest_id, quest_mission_id, progress_value, is_clear, latest_clear_datetime, latest_version
		FROM user_quest_missions WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.UserQuestMissionState
		var ic int
		rows.Scan(&v.QuestId, &v.QuestMissionId, &v.ProgressValue, &ic, &v.LatestClearDatetime, &v.LatestVersion)
		v.IsClear = ic != 0
		u.QuestMissions[store.QuestMissionKey{QuestId: v.QuestId, QuestMissionId: v.QuestMissionId}] = v
	})

	queryRows(db, `SELECT mission_id, start_datetime, progress_value, mission_progress_status_type,
		clear_datetime, latest_version FROM user_missions WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.UserMissionState
		rows.Scan(&v.MissionId, &v.StartDatetime, &v.ProgressValue, &v.MissionProgressStatusType,
			&v.ClearDatetime, &v.LatestVersion)
		u.Missions[v.MissionId] = v
	})

	queryRows(db, `SELECT tutorial_type, progress_phase, choice_id, latest_version
		FROM user_tutorials WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.TutorialProgressState
		rows.Scan(&v.TutorialType, &v.ProgressPhase, &v.ChoiceId, &v.LatestVersion)
		u.Tutorials[v.TutorialType] = v
	})

	queryRows(db, `SELECT side_story_quest_id, head_side_story_quest_scene_id, side_story_quest_state_type, latest_version
		FROM user_side_story_quests WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var id, head, st int32
		var lv int64
		rows.Scan(&id, &head, &st, &lv)
		u.SideStoryQuests[id] = store.SideStoryQuestProgress{
			HeadSideStoryQuestSceneId: head, SideStoryQuestStateType: model.SideStoryQuestStateType(st), LatestVersion: lv,
		}
	})

	queryRows(db, `SELECT limit_content_id, limit_content_quest_status_type, event_quest_chapter_id, latest_version
		FROM user_quest_limit_content_status WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var id int32
		var v store.QuestLimitContentStatus
		rows.Scan(&id, &v.LimitContentQuestStatusType, &v.EventQuestChapterId, &v.LatestVersion)
		u.QuestLimitContentStatus[id] = v
	})

	queryRows(db, `SELECT weapon_id, released_max_story_index, latest_version FROM user_weapon_stories WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.WeaponStoryState
			rows.Scan(&v.WeaponId, &v.ReleasedMaxStoryIndex, &v.LatestVersion)
			u.WeaponStories[v.WeaponId] = v
		})

	queryRows(db, `SELECT weapon_id, max_level, max_limit_break_count, first_acquisition_datetime, latest_version
		FROM user_weapon_notes WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.WeaponNoteState
		rows.Scan(&v.WeaponId, &v.MaxLevel, &v.MaxLimitBreakCount, &v.FirstAcquisitionDatetime, &v.LatestVersion)
		u.WeaponNotes[v.WeaponId] = v
	})

	queryRows(db, `SELECT user_weapon_uuid, slot_number, level FROM user_weapon_skills WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.WeaponSkillState
			rows.Scan(&v.UserWeaponUuid, &v.SlotNumber, &v.Level)
			u.WeaponSkills[v.UserWeaponUuid] = append(u.WeaponSkills[v.UserWeaponUuid], v)
		})

	queryRows(db, `SELECT user_weapon_uuid, slot_number, level FROM user_weapon_abilities WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.WeaponAbilityState
			rows.Scan(&v.UserWeaponUuid, &v.SlotNumber, &v.Level)
			u.WeaponAbilities[v.UserWeaponUuid] = append(u.WeaponAbilities[v.UserWeaponUuid], v)
		})

	queryRows(db, `SELECT user_weapon_uuid, latest_version FROM user_weapon_awakens WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.WeaponAwakenState
			rows.Scan(&v.UserWeaponUuid, &v.LatestVersion)
			u.WeaponAwakens[v.UserWeaponUuid] = v
		})

	queryRows(db, `SELECT user_costume_uuid, level, acquisition_datetime, latest_version
		FROM user_costume_active_skills WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.CostumeActiveSkillState
		rows.Scan(&v.UserCostumeUuid, &v.Level, &v.AcquisitionDatetime, &v.LatestVersion)
		u.CostumeActiveSkills[v.UserCostumeUuid] = v
	})

	queryRows(db, `SELECT user_costume_uuid, status_calculation_type, hp, attack, vitality, agility,
		critical_ratio, critical_attack, latest_version FROM user_costume_awaken_status_ups WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.CostumeAwakenStatusUpState
			var sct int32
			rows.Scan(&v.UserCostumeUuid, &sct, &v.Hp, &v.Attack, &v.Vitality, &v.Agility,
				&v.CriticalRatio, &v.CriticalAttack, &v.LatestVersion)
			v.StatusCalculationType = model.StatusCalculationType(sct)
			u.CostumeAwakenStatusUps[store.CostumeAwakenStatusKey{
				UserCostumeUuid: v.UserCostumeUuid, StatusCalculationType: v.StatusCalculationType,
			}] = v
		})

	queryRows(db, `SELECT costume_id, last_released_bonus_level, confirmed_bonus_level, latest_version
		FROM user_costume_level_bonus_release_statuses WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.CostumeLevelBonusReleaseStatusState
		rows.Scan(&v.CostumeId, &v.LastReleasedBonusLevel, &v.ConfirmedBonusLevel, &v.LatestVersion)
		u.CostumeLevelBonusReleaseStatuses[v.CostumeId] = v
	})

	queryRows(db, `SELECT user_costume_uuid, slot_number, odds_number, latest_version
		FROM user_costume_lottery_effects WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.CostumeLotteryEffectState
			rows.Scan(&v.UserCostumeUuid, &v.SlotNumber, &v.OddsNumber, &v.LatestVersion)
			u.CostumeLotteryEffects[store.CostumeLotteryEffectKey{
				UserCostumeUuid: v.UserCostumeUuid, SlotNumber: v.SlotNumber,
			}] = v
		})

	queryRows(db, `SELECT user_costume_uuid, slot_number, ability_id, ability_level, latest_version
		FROM user_costume_lottery_effect_abilities WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.CostumeLotteryEffectAbilityState
			rows.Scan(&v.UserCostumeUuid, &v.SlotNumber, &v.AbilityId, &v.AbilityLevel, &v.LatestVersion)
			u.CostumeLotteryEffectAbilities[store.CostumeLotteryEffectKey{
				UserCostumeUuid: v.UserCostumeUuid, SlotNumber: v.SlotNumber,
			}] = v
		})

	queryRows(db, `SELECT user_costume_uuid, status_calculation_type, hp, attack, vitality, agility,
		critical_ratio, critical_attack, latest_version FROM user_costume_lottery_effect_status_ups WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.CostumeLotteryEffectStatusUpState
			var calculationType int32
			rows.Scan(&v.UserCostumeUuid, &calculationType, &v.Hp, &v.Attack, &v.Vitality, &v.Agility,
				&v.CriticalRatio, &v.CriticalAttack, &v.LatestVersion)
			v.StatusCalculationType = model.StatusCalculationType(calculationType)
			u.CostumeLotteryEffectStatusUps[store.CostumeLotteryEffectStatusKey{
				UserCostumeUuid: v.UserCostumeUuid, StatusCalculationType: v.StatusCalculationType,
			}] = v
		})

	queryRows(db, `SELECT user_costume_uuid, slot_number, odds_number, latest_version
		FROM user_costume_lottery_effect_pending WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.CostumeLotteryEffectPendingState
			rows.Scan(&v.UserCostumeUuid, &v.SlotNumber, &v.OddsNumber, &v.LatestVersion)
			u.CostumeLotteryEffectPending[v.UserCostumeUuid] = v
		})

	queryRows(db, `SELECT user_parts_uuid, parts_id, level, parts_status_main_id, is_protected,
		acquisition_datetime, latest_version FROM user_parts WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.PartsState
			var prot int
			rows.Scan(&v.UserPartsUuid, &v.PartsId, &v.Level, &v.PartsStatusMainId, &prot,
				&v.AcquisitionDatetime, &v.LatestVersion)
			v.IsProtected = prot != 0
			u.Parts[v.UserPartsUuid] = v
		})

	queryRows(db, `SELECT parts_group_id, first_acquisition_datetime, latest_version
		FROM user_parts_group_notes WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.PartsGroupNoteState
		rows.Scan(&v.PartsGroupId, &v.FirstAcquisitionDatetime, &v.LatestVersion)
		u.PartsGroupNotes[v.PartsGroupId] = v
	})

	queryRows(db, `SELECT user_parts_preset_number, user_parts_uuid01, user_parts_uuid02, user_parts_uuid03,
		name, user_parts_preset_tag_number, latest_version FROM user_parts_presets WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.PartsPresetState
			rows.Scan(&v.UserPartsPresetNumber, &v.UserPartsUuid01, &v.UserPartsUuid02, &v.UserPartsUuid03,
				&v.Name, &v.UserPartsPresetTagNumber, &v.LatestVersion)
			u.PartsPresets[v.UserPartsPresetNumber] = v
		})

	queryRows(db, `SELECT user_parts_preset_tag_number, name, latest_version
		FROM user_parts_preset_tags WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.PartsPresetTagState
			rows.Scan(&v.UserPartsPresetTagNumber, &v.Name, &v.LatestVersion)
			u.PartsPresetTags[v.UserPartsPresetTagNumber] = v
		})

	queryRows(db, `SELECT user_parts_uuid, status_index, parts_status_sub_lottery_id, level,
		status_kind_type, status_calculation_type, status_change_value, latest_version
		FROM user_parts_status_subs WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.PartsStatusSubState
			rows.Scan(&v.UserPartsUuid, &v.StatusIndex, &v.PartsStatusSubLotteryId, &v.Level,
				&v.StatusKindType, &v.StatusCalculationType, &v.StatusChangeValue, &v.LatestVersion)
			u.PartsStatusSubs[store.PartsStatusSubKey{UserPartsUuid: v.UserPartsUuid, StatusIndex: v.StatusIndex}] = v
		})

	queryRows(db, `SELECT deck_type, max_deck_power, latest_version FROM user_deck_type_notes WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var dt int32
			var v store.DeckTypeNoteState
			rows.Scan(&dt, &v.MaxDeckPower, &v.LatestVersion)
			v.DeckType = model.DeckType(dt)
			u.DeckTypeNotes[v.DeckType] = v
		})

	loadSimpleMap(db, uid, `SELECT consumable_item_id, count FROM user_consumable_items WHERE user_id=?`, u.ConsumableItems)
	loadSimpleMap(db, uid, `SELECT material_id, count FROM user_materials WHERE user_id=?`, u.Materials)
	loadSimpleMap(db, uid, `SELECT important_item_id, count FROM user_important_items WHERE user_id=?`, u.ImportantItems)

	queryRows(db, `SELECT premium_item_id, count FROM user_premium_items WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var k int32
			var v int64
			rows.Scan(&k, &v)
			u.PremiumItems[k] = v
		})

	queryRows(db, `SELECT explore_id, max_score, max_score_update_datetime, latest_version
		FROM user_explore_scores WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.ExploreScoreState
		rows.Scan(&v.ExploreId, &v.MaxScore, &v.MaxScoreUpdateDatetime, &v.LatestVersion)
		u.ExploreScores[v.ExploreId] = v
	})

	queryRows(db, `SELECT possession_auto_sale_item_type, possession_auto_sale_item_value
		FROM user_auto_sale_settings WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.AutoSaleSettingState
		rows.Scan(&v.PossessionAutoSaleItemType, &v.PossessionAutoSaleItemValue)
		u.AutoSaleSettings[v.PossessionAutoSaleItemType] = v
	})

	queryRows(db, `SELECT navi_cutin_id FROM user_navi_cutin_played WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var id int32
			rows.Scan(&id)
			u.NaviCutInPlayed[id] = true
		})

	loadTimestampMap(db, uid, `SELECT movie_id, timestamp FROM user_viewed_movies WHERE user_id=?`, u.ViewedMovies)
	loadTimestampMap(db, uid, `SELECT contents_story_id, timestamp FROM user_contents_stories WHERE user_id=?`, u.ContentsStories)
	loadTimestampMap(db, uid, `SELECT omikuji_id, timestamp FROM user_drawn_omikuji WHERE user_id=?`, u.DrawnOmikuji)

	queryRows(db, `SELECT dokan_id FROM user_dokan_confirmed WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var id int32
			rows.Scan(&id)
			u.DokanConfirmed[id] = true
		})

	queryRows(db, `SELECT user_gift_uuid, is_received, possession_type, possession_id, count, grant_datetime,
		description_gift_text_id, equipment_data, expiration_datetime, received_datetime
		FROM user_gifts WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var uuid string
		var isRecv int
		var gc store.GiftCommonState
		var expDt, recvDt sql.NullInt64
		var equipData []byte
		rows.Scan(&uuid, &isRecv, &gc.PossessionType, &gc.PossessionId, &gc.Count, &gc.GrantDatetime,
			&gc.DescriptionGiftTextId, &equipData, &expDt, &recvDt)
		gc.EquipmentData = equipData
		if isRecv == 0 {
			u.Gifts.NotReceived = append(u.Gifts.NotReceived, store.NotReceivedGiftState{
				GiftCommon: gc, ExpirationDatetime: expDt.Int64, UserGiftUuid: uuid,
			})
		} else {
			u.Gifts.Received = append(u.Gifts.Received, store.ReceivedGiftState{
				GiftCommon: gc, ReceivedDatetime: recvDt.Int64,
			})
		}
	})

	queryRows(db, `SELECT consumable_item_id, count FROM user_gacha_converted_medals WHERE user_id=? ORDER BY ordinal`, uid,
		func(rows *sql.Rows) {
			var v store.ConsumableItemState
			rows.Scan(&v.ConsumableItemId, &v.Count)
			u.Gacha.ConvertedGachaMedal.ConvertedMedalPossession = append(u.Gacha.ConvertedGachaMedal.ConvertedMedalPossession, v)
		})

	queryRows(db, `SELECT gacha_id, medal_count, step_number, loop_count, draw_count, box_number
		FROM user_gacha_banners WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.GachaBannerState
		rows.Scan(&v.GachaId, &v.MedalCount, &v.StepNumber, &v.LoopCount, &v.DrawCount, &v.BoxNumber)
		v.BoxDrewCounts = make(map[int32]int32)
		u.Gacha.BannerStates[v.GachaId] = v
	})
	queryRows(db, `SELECT gacha_id, box_item_id, count FROM user_gacha_banner_box_drew_counts WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var gachaId, boxItemId, count int32
			rows.Scan(&gachaId, &boxItemId, &count)
			if bs, ok := u.Gacha.BannerStates[gachaId]; ok {
				bs.BoxDrewCounts[boxItemId] = count
				u.Gacha.BannerStates[gachaId] = bs
			}
		})

	queryRows(db, `SELECT character_board_id, panel_release_bit1, panel_release_bit2, panel_release_bit3,
		panel_release_bit4, latest_version FROM user_character_boards WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.CharacterBoardState
			rows.Scan(&v.CharacterBoardId, &v.PanelReleaseBit1, &v.PanelReleaseBit2,
				&v.PanelReleaseBit3, &v.PanelReleaseBit4, &v.LatestVersion)
			u.CharacterBoards[v.CharacterBoardId] = v
		})

	queryRows(db, `SELECT character_id, ability_id, level, latest_version
		FROM user_character_board_abilities WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.CharacterBoardAbilityState
		rows.Scan(&v.CharacterId, &v.AbilityId, &v.Level, &v.LatestVersion)
		u.CharacterBoardAbilities[store.CharacterBoardAbilityKey{CharacterId: v.CharacterId, AbilityId: v.AbilityId}] = v
	})

	queryRows(db, `SELECT character_id, status_calculation_type, hp, attack, vitality, agility,
		critical_ratio, critical_attack, latest_version FROM user_character_board_status_ups WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.CharacterBoardStatusUpState
			rows.Scan(&v.CharacterId, &v.StatusCalculationType, &v.Hp, &v.Attack, &v.Vitality, &v.Agility,
				&v.CriticalRatio, &v.CriticalAttack, &v.LatestVersion)
			u.CharacterBoardStatusUps[store.CharacterBoardStatusUpKey{
				CharacterId: v.CharacterId, StatusCalculationType: v.StatusCalculationType,
			}] = v
		})

	queryRows(db, `SELECT character_id, rebirth_count, latest_version FROM user_character_rebirths WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.CharacterRebirthState
			rows.Scan(&v.CharacterId, &v.RebirthCount, &v.LatestVersion)
			u.CharacterRebirths[v.CharacterId] = v
		})

	queryRows(db, `SELECT cage_ornament_id, acquisition_datetime, latest_version
		FROM user_cage_ornament_rewards WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.CageOrnamentRewardState
		rows.Scan(&v.CageOrnamentId, &v.AcquisitionDatetime, &v.LatestVersion)
		u.CageOrnamentRewards[v.CageOrnamentId] = v
	})

	queryRows(db, `SELECT cage_ornament_id, first_access_datetime, latest_access_datetime, latest_version
		FROM user_cage_ornament_accesses WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.CageOrnamentAccessState
		rows.Scan(&v.CageOrnamentId, &v.FirstAccessDatetime, &v.LatestAccessDatetime, &v.LatestVersion)
		u.CageOrnamentAccesses[v.CageOrnamentId] = v
	})

	queryRows(db, `SELECT event_quest_chapter_id, latest_reward_receive_quest_mission_clear_count, latest_version
		FROM user_event_quest_tower_accumulation_rewards WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.TowerAccumulationRewardState
		rows.Scan(&v.EventQuestChapterId, &v.LatestRewardReceiveQuestMissionClearCount, &v.LatestVersion)
		u.TowerAccumulationRewards[v.EventQuestChapterId] = v
	})

	queryRows(db, `SELECT event_quest_chapter_id, last_join_season_number, last_season_reward_received_season_number, latest_version
		FROM user_event_quest_labyrinth_seasons WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.LabyrinthSeasonState
		rows.Scan(&v.EventQuestChapterId, &v.LastJoinSeasonNumber, &v.LastSeasonRewardReceivedSeasonNumber, &v.LatestVersion)
		u.LabyrinthSeasons[v.EventQuestChapterId] = v
	})

	queryRows(db, `SELECT event_quest_chapter_id, stage_order, is_received_stage_clear_reward, accumulation_reward_received_quest_mission_count, latest_version
		FROM user_event_quest_labyrinth_stages WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.LabyrinthStageState
		var rcvd int
		rows.Scan(&v.EventQuestChapterId, &v.StageOrder, &rcvd, &v.AccumulationRewardReceivedQuestMissionCount, &v.LatestVersion)
		v.IsReceivedStageClearReward = rcvd != 0
		u.LabyrinthStages[store.LabyrinthStageKey{EventQuestChapterId: v.EventQuestChapterId, StageOrder: v.StageOrder}] = v
	})

	queryRows(db, `SELECT shop_item_id, bought_count, latest_bought_count_changed_datetime, latest_version
		FROM user_shop_items WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.UserShopItemState
		rows.Scan(&v.ShopItemId, &v.BoughtCount, &v.LatestBoughtCountChangedDatetime, &v.LatestVersion)
		u.ShopItems[v.ShopItemId] = v
	})

	queryRows(db, `SELECT slot_number, shop_item_id, latest_version FROM user_shop_replaceable_lineup WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.UserShopReplaceableLineupState
			rows.Scan(&v.SlotNumber, &v.ShopItemId, &v.LatestVersion)
			u.ShopReplaceableLineup[v.SlotNumber] = v
		})

	queryRows(db, `SELECT gimmick_sequence_schedule_id, gimmick_sequence_id, gimmick_id,
		is_gimmick_cleared, start_datetime, latest_version FROM user_gimmick_progress WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.GimmickProgressState
			var ic int
			rows.Scan(&v.Key.GimmickSequenceScheduleId, &v.Key.GimmickSequenceId, &v.Key.GimmickId,
				&ic, &v.StartDatetime, &v.LatestVersion)
			v.IsGimmickCleared = ic != 0
			u.Gimmick.Progress[v.Key] = v
		})

	queryRows(db, `SELECT gimmick_sequence_schedule_id, gimmick_sequence_id, gimmick_id,
		gimmick_ornament_index, progress_value_bit, base_datetime, latest_version
		FROM user_gimmick_ornament_progress WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var v store.GimmickOrnamentProgressState
		rows.Scan(&v.Key.GimmickSequenceScheduleId, &v.Key.GimmickSequenceId, &v.Key.GimmickId,
			&v.Key.GimmickOrnamentIndex, &v.ProgressValueBit, &v.BaseDatetime, &v.LatestVersion)
		u.Gimmick.OrnamentProgress[v.Key] = v
	})

	queryRows(db, `SELECT gimmick_sequence_schedule_id, gimmick_sequence_id,
		is_gimmick_sequence_cleared, clear_datetime, latest_version FROM user_gimmick_sequences WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.GimmickSequenceState
			var ic int
			rows.Scan(&v.Key.GimmickSequenceScheduleId, &v.Key.GimmickSequenceId,
				&ic, &v.ClearDatetime, &v.LatestVersion)
			v.IsGimmickSequenceCleared = ic != 0
			u.Gimmick.Sequences[v.Key] = v
		})

	queryRows(db, `SELECT gimmick_sequence_schedule_id, gimmick_sequence_id, gimmick_id,
		is_unlocked, latest_version FROM user_gimmick_unlocks WHERE user_id=?`, uid,
		func(rows *sql.Rows) {
			var v store.GimmickUnlockState
			var iu int
			rows.Scan(&v.Key.GimmickSequenceScheduleId, &v.Key.GimmickSequenceId, &v.Key.GimmickId,
				&iu, &v.LatestVersion)
			v.IsUnlocked = iu != 0
			u.Gimmick.Unlocks[v.Key] = v
		})

	queryRows(db, `SELECT big_hunt_boss_id, max_score, max_score_update_datetime, latest_version
		FROM user_big_hunt_max_scores WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var id int32
		var v store.BigHuntMaxScore
		rows.Scan(&id, &v.MaxScore, &v.MaxScoreUpdateDatetime, &v.LatestVersion)
		u.BigHuntMaxScores[id] = v
	})

	queryRows(db, `SELECT big_hunt_boss_id, daily_challenge_count, latest_challenge_datetime, last_daily_reward_received_day_version, latest_version
		FROM user_big_hunt_statuses WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var id int32
		var v store.BigHuntStatus
		rows.Scan(&id, &v.DailyChallengeCount, &v.LatestChallengeDatetime, &v.LastDailyRewardReceivedDayVersion, &v.LatestVersion)
		u.BigHuntStatuses[id] = v
	})

	queryRows(db, `SELECT big_hunt_schedule_id, big_hunt_boss_id, max_score, max_score_update_datetime, latest_version
		FROM user_big_hunt_schedule_max_scores WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var k store.BigHuntScheduleScoreKey
		var v store.BigHuntScheduleMaxScore
		rows.Scan(&k.BigHuntScheduleId, &k.BigHuntBossId, &v.MaxScore, &v.MaxScoreUpdateDatetime, &v.LatestVersion)
		u.BigHuntScheduleMaxScores[k] = v
	})

	queryRows(db, `SELECT big_hunt_weekly_version, attribute_type, max_score, latest_version
		FROM user_big_hunt_weekly_max_scores WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var k store.BigHuntWeeklyScoreKey
		var v store.BigHuntWeeklyMaxScore
		rows.Scan(&k.BigHuntWeeklyVersion, &k.AttributeType, &v.MaxScore, &v.LatestVersion)
		u.BigHuntWeeklyMaxScores[k] = v
	})

	queryRows(db, `SELECT big_hunt_weekly_version, is_received_weekly_reward, latest_version
		FROM user_big_hunt_weekly_statuses WHERE user_id=?`, uid, func(rows *sql.Rows) {
		var ver int64
		var ir int
		var lv int64
		rows.Scan(&ver, &ir, &lv)
		u.BigHuntWeeklyStatuses[ver] = store.BigHuntWeeklyStatus{IsReceivedWeeklyReward: ir != 0, LatestVersion: lv}
	})
}

func queryRows(db *sql.DB, query string, uid int64, scan func(*sql.Rows)) {
	rows, err := db.Query(query, uid)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		scan(rows)
	}
}

func loadSimpleMap(db *sql.DB, uid int64, query string, m map[int32]int32) {
	queryRows(db, query, uid, func(rows *sql.Rows) {
		var k, v int32
		rows.Scan(&k, &v)
		m[k] = v
	})
}

func loadTimestampMap(db *sql.DB, uid int64, query string, m map[int32]int64) {
	queryRows(db, query, uid, func(rows *sql.Rows) {
		var k int32
		var v int64
		rows.Scan(&k, &v)
		m[k] = v
	})
}
