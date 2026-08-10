package masterdataadmin

import "strings"

type fieldKind string

const (
	fieldKindInt32  fieldKind = "int32"
	fieldKindInt64  fieldKind = "int64"
	fieldKindBool   fieldKind = "bool"
	fieldKindString fieldKind = "string"
)

type columnSpec struct {
	Name       string
	Index      int
	SchemaType string
	Kind       fieldKind
	PrimaryKey bool
	Datetime   bool
}

type tableSpec struct {
	Name       string
	EntityName string
	Primary    bool
	Fields     []columnSpec
	Times      []columnSpec
}

type timePair struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

func field(name string, index int, schemaType string) columnSpec {
	kind := fieldKindInt32
	switch schemaType {
	case "long":
		kind = fieldKindInt64
	case "bool":
		kind = fieldKindBool
	case "string":
		kind = fieldKindString
	}
	return columnSpec{
		Name: name, Index: index, SchemaType: schemaType, Kind: kind,
		Datetime: kind == fieldKindInt64 && strings.HasSuffix(name, "Datetime"),
	}
}

func activityTable(name, entityName string, primaryKeyCount int, primary bool, fields ...columnSpec) tableSpec {
	for index := range fields {
		fields[index].PrimaryKey = index < primaryKeyCount
	}
	spec := tableSpec{Name: name, EntityName: entityName, Primary: primary, Fields: fields}
	for _, field := range fields {
		if field.Datetime {
			spec.Times = append(spec.Times, field)
		}
	}
	return spec
}

func (s tableSpec) pairs() []timePair {
	fields := make(map[string]struct{}, len(s.Times))
	for _, field := range s.Times {
		fields[field.Name] = struct{}{}
	}
	var pairs []timePair
	for _, field := range s.Times {
		if !strings.HasSuffix(field.Name, "StartDatetime") {
			continue
		}
		end := strings.TrimSuffix(field.Name, "StartDatetime") + "EndDatetime"
		if _, ok := fields[end]; ok {
			pairs = append(pairs, timePair{Start: field.Name, End: end})
		}
	}
	return pairs
}

// activityTableSpecs contains the selected operational schedules followed by
// the first-level non-schedule tables directly referenced by their fields.
// Column names and scalar types are generated from scripts/schemas.json; the
// primary-key prefixes are kept read-only by the admin API.
var activityTableSpecs = []tableSpec{
	activityTable("m_beginner_campaign", "EntityMBeginnerCampaign", 1, true, field("BeginnerCampaignId", 0, "int"), field("BeginnerJudgeStartDatetime", 1, "long"), field("BeginnerJudgeEndDatetime", 2, "long"), field("GrantCampaignTermDayCount", 3, "int"), field("CampaignUnlockQuestId", 4, "int")),
	activityTable("m_big_hunt_schedule", "EntityMBigHuntSchedule", 1, true, field("BigHuntScheduleId", 0, "int"), field("NoticeStartDatetime", 1, "long"), field("ChallengeStartDatetime", 2, "long"), field("ChallengeEndDatetime", 3, "long"), field("SeasonAssetId", 4, "int")),
	activityTable("m_comeback_campaign", "EntityMComebackCampaign", 1, true, field("ComebackCampaignId", 0, "int"), field("ComebackJudgeStartDatetime", 1, "long"), field("ComebackJudgeEndDatetime", 2, "long"), field("ComebackJudgeDayCount", 3, "int"), field("GrantCampaignTermDayCount", 4, "int"), field("CampaignUnlockQuestId", 5, "int"), field("ComebackCampaignGradeGroupId", 6, "int")),
	activityTable("m_consumable_item_term", "EntityMConsumableItemTerm", 1, true, field("ConsumableItemTermId", 0, "int"), field("StartDatetime", 1, "long"), field("EndDatetime", 2, "long")),
	activityTable("m_enhance_campaign", "EntityMEnhanceCampaign", 1, true, field("EnhanceCampaignId", 0, "int"), field("EnhanceCampaignTargetGroupId", 1, "int"), field("EnhanceCampaignEffectType", 2, "EnhanceCampaignEffectType"), field("EnhanceCampaignEffectValue", 3, "int"), field("StartDatetime", 4, "long"), field("EndDatetime", 5, "long"), field("TargetUserStatusType", 6, "TargetUserStatusType"), field("SortOrder", 7, "int")),
	activityTable("m_event_quest_chapter", "EntityMEventQuestChapter", 1, true, field("EventQuestChapterId", 0, "int"), field("EventQuestType", 1, "EventQuestType"), field("SortOrder", 2, "int"), field("NameEventQuestTextId", 3, "int"), field("BannerAssetId", 4, "int"), field("EventQuestLinkId", 5, "int"), field("EventQuestDisplayItemGroupId", 6, "int"), field("EventQuestSequenceGroupId", 7, "int"), field("StartDatetime", 8, "long"), field("EndDatetime", 9, "long"), field("DisplaySortOrder", 10, "int")),
	activityTable("m_event_quest_daily_group", "EntityMEventQuestDailyGroup", 1, true, field("EventQuestDailyGroupId", 0, "int"), field("StartDatetime", 1, "long"), field("EndDatetime", 2, "long"), field("EventQuestDailyGroupTargetChapterId", 3, "int"), field("EventQuestDailyGroupCompleteRewardId", 4, "int"), field("EventQuestDailyGroupMessageId", 5, "int")),
	activityTable("m_event_quest_labyrinth_season", "EntityMEventQuestLabyrinthSeason", 2, true, field("EventQuestChapterId", 0, "int"), field("SeasonNumber", 1, "int"), field("StartDatetime", 2, "long"), field("EndDatetime", 3, "long"), field("SeasonRewardGroupId", 4, "int")),
	activityTable("m_login_bonus", "EntityMLoginBonus", 1, true, field("LoginBonusId", 0, "int"), field("SortOrder", 1, "int"), field("LoginBonusStartConditionId", 2, "int"), field("TotalPageCount", 3, "int"), field("StartDatetime", 4, "long"), field("EndDatetime", 5, "long"), field("StampReceiveEndDatetime", 6, "long"), field("LoginBonusAssetName", 7, "string")),
	activityTable("m_maintenance", "EntityMMaintenance", 1, true, field("MaintenanceId", 0, "int"), field("StartDatetime", 1, "long"), field("EndDatetime", 2, "long"), field("MaintenanceGroupId", 3, "int")),
	activityTable("m_mom_banner", "EntityMMomBanner", 1, true, field("MomBannerId", 0, "int"), field("SortOrderDesc", 1, "int"), field("DestinationDomainType", 2, "DomainType"), field("DestinationDomainId", 3, "int"), field("BannerAssetName", 4, "string"), field("IsEmphasis", 5, "bool"), field("StartDatetime", 6, "long"), field("EndDatetime", 7, "long"), field("TargetUserStatusType", 8, "TargetUserStatusType")),
	activityTable("m_omikuji", "EntityMOmikuji", 1, true, field("OmikujiId", 0, "int"), field("StartDatetime", 1, "long"), field("EndDatetime", 2, "long"), field("OmikujiAssetId", 3, "int")),
	activityTable("m_pvp_season", "EntityMPvpSeason", 1, true, field("PvpSeasonId", 0, "int"), field("NameAssetPath", 1, "string"), field("SeasonStartDatetime", 2, "long"), field("SeasonEndDatetime", 3, "long"), field("PvpSeasonGroupingId", 4, "int"), field("IsInvalid", 5, "bool"), field("PvpWeeklyRankRewardRankGroupId", 6, "int"), field("PvpSeasonRankRewardRankGroupId", 7, "int"), field("PvpGradeGroupId", 8, "int"), field("PvpInitialPointAdditionGroupId", 9, "int"), field("PvpSeasonDeckPowerThresholdGroupingId", 10, "int")),
	activityTable("m_quest_campaign", "EntityMQuestCampaign", 1, true, field("QuestCampaignId", 0, "int"), field("QuestCampaignTargetGroupId", 1, "int"), field("QuestCampaignEffectGroupId", 2, "int"), field("StartDatetime", 3, "long"), field("EndDatetime", 4, "long"), field("TargetUserStatusType", 5, "TargetUserStatusType"), field("SortOrder", 6, "int")),
	activityTable("m_shop", "EntityMShop", 1, true, field("ShopId", 0, "int"), field("ShopGroupType", 1, "ShopGroupType"), field("SortOrderInShopGroup", 2, "int"), field("ShopType", 3, "ShopType"), field("NameShopTextId", 4, "int"), field("ShopUpdatableLabelType", 5, "ShopUpdatableLabelType"), field("ShopExchangeType", 6, "ShopExchangeType"), field("ShopItemCellGroupId", 7, "int"), field("RelatedMainFunctionType", 8, "MainFunctionType"), field("StartDatetime", 9, "long"), field("EndDatetime", 10, "long"), field("LimitedOpenId", 11, "int")),
	activityTable("m_shop_item_cell_term", "EntityMShopItemCellTerm", 1, true, field("ShopItemCellTermId", 0, "int"), field("StartDatetime", 1, "long"), field("EndDatetime", 2, "long")),

	activityTable("m_enhance_campaign_target_group", "EntityMEnhanceCampaignTargetGroup", 2, false, field("EnhanceCampaignTargetGroupId", 0, "int"), field("EnhanceCampaignTargetIndex", 1, "int"), field("EnhanceCampaignTargetType", 2, "EnhanceCampaignTargetType"), field("EnhanceCampaignTargetValue", 3, "int")),
	activityTable("m_event_quest_link", "EntityMEventQuestLink", 1, false, field("EventQuestLinkId", 0, "int"), field("DestinationDomainType", 1, "DomainType"), field("DestinationDomainId", 2, "int"), field("PossessionType", 3, "PossessionType"), field("PossessionId", 4, "int")),
	activityTable("m_event_quest_display_item_group", "EntityMEventQuestDisplayItemGroup", 2, false, field("EventQuestDisplayItemGroupId", 0, "int"), field("SortOrder", 1, "int"), field("PossessionType", 2, "PossessionType"), field("PossessionId", 3, "int")),
	activityTable("m_event_quest_sequence_group", "EntityMEventQuestSequenceGroup", 2, false, field("EventQuestSequenceGroupId", 0, "int"), field("DifficultyType", 1, "DifficultyType"), field("EventQuestSequenceId", 2, "int")),
	activityTable("m_event_quest_daily_group_target_chapter", "EntityMEventQuestDailyGroupTargetChapter", 2, false, field("EventQuestDailyGroupTargetChapterId", 0, "int"), field("SortOrder", 1, "int"), field("EventQuestChapterId", 2, "int")),
	activityTable("m_event_quest_daily_group_complete_reward", "EntityMEventQuestDailyGroupCompleteReward", 2, false, field("EventQuestDailyGroupCompleteRewardId", 0, "int"), field("SortOrder", 1, "int"), field("PossessionType", 2, "PossessionType"), field("PossessionId", 3, "int"), field("Count", 4, "int")),
	activityTable("m_event_quest_daily_group_message", "EntityMEventQuestDailyGroupMessage", 2, false, field("EventQuestDailyGroupMessageId", 0, "int"), field("OddsNumber", 1, "int"), field("Weight", 2, "int"), field("BeforeClearMessageTextId", 3, "int"), field("AfterClearMessageTextId", 4, "int")),
	activityTable("m_event_quest_labyrinth_season_reward_group", "EntityMEventQuestLabyrinthSeasonRewardGroup", 2, false, field("EventQuestLabyrinthSeasonRewardGroupId", 0, "int"), field("HeadQuestId", 1, "int"), field("EventQuestLabyrinthRewardGroupId", 2, "int")),
	activityTable("m_maintenance_group", "EntityMMaintenanceGroup", 2, false, field("MaintenanceGroupId", 0, "int"), field("ApiPath", 1, "string"), field("Priority", 2, "int"), field("ScreenTransitionType", 3, "ScreenTransitionType"), field("BlockFunctionType", 4, "MaintenanceBlockFunctionType"), field("BlockFunctionValue", 5, "string")),
	activityTable("m_pvp_season_grouping", "EntityMPvpSeasonGrouping", 2, false, field("PvpSeasonGroupingId", 0, "int"), field("GroupId", 1, "int"), field("DivideWeight", 2, "int")),
	activityTable("m_pvp_weekly_rank_reward_rank_group", "EntityMPvpWeeklyRankRewardRankGroup", 2, false, field("PvpWeeklyRankRewardRankGroupId", 0, "int"), field("RankLowerLimit", 1, "int"), field("PvpWeeklyRankRewardGroupId", 2, "int")),
	activityTable("m_pvp_season_rank_reward_rank_group", "EntityMPvpSeasonRankRewardRankGroup", 2, false, field("PvpSeasonRankRewardRankGroupId", 0, "int"), field("RankLowerLimit", 1, "int"), field("PvpSeasonRankRewardGroupId", 2, "int")),
	activityTable("m_pvp_grade_group", "EntityMPvpGradeGroup", 2, false, field("PvpGradeGroupId", 0, "int"), field("PvpGradeId", 1, "int"), field("NecessaryPvpPoint", 2, "int"), field("IconAssetId", 3, "int"), field("PvpGradeWeeklyRewardGroupId", 4, "int"), field("PvpGradeOneMatchRewardGroupId", 5, "int")),
	activityTable("m_quest_campaign_target_group", "EntityMQuestCampaignTargetGroup", 2, false, field("QuestCampaignTargetGroupId", 0, "int"), field("QuestCampaignTargetIndex", 1, "int"), field("QuestCampaignTargetType", 2, "QuestCampaignTargetType"), field("QuestCampaignTargetValue", 3, "int")),
	activityTable("m_quest_campaign_effect_group", "EntityMQuestCampaignEffectGroup", 1, false, field("QuestCampaignEffectGroupId", 0, "int"), field("QuestCampaignEffectType", 1, "QuestCampaignEffectType"), field("QuestCampaignEffectValue", 2, "int"), field("QuestCampaignTargetItemGroupId", 3, "int")),
	activityTable("m_shop_item_cell_group", "EntityMShopItemCellGroup", 2, false, field("ShopItemCellGroupId", 0, "int"), field("ShopItemCellId", 1, "int"), field("SortOrder", 2, "int"), field("ShopItemCellTermId", 3, "int")),
}
