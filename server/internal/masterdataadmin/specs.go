package masterdataadmin

import "strings"

type columnSpec struct {
	Name  string
	Index int
}

type tableSpec struct {
	Name       string
	EntityName string
	Keys       []columnSpec
	Times      []columnSpec
}

type timePair struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

func column(name string, index int) columnSpec {
	return columnSpec{Name: name, Index: index}
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

// scheduleTableSpecs is generated from scripts/schemas.json. A table is in
// scope when its schema has a matching *StartDatetime/*EndDatetime pair. All
// datetime columns on an in-scope table are exposed so secondary boundaries
// such as NoticeStartDatetime and StampReceiveEndDatetime are not hidden.
var scheduleTableSpecs = []tableSpec{
	{Name: "m_appeal_dialog", EntityName: "EntityMAppealDialog", Keys: []columnSpec{column("AppealDialogId", 0), column("SortOrder", 1), column("AppealTargetType", 2), column("TitleTextId", 6), column("AssetId", 7)}, Times: []columnSpec{column("StartDatetime", 4), column("EndDatetime", 5)}},
	{Name: "m_beginner_campaign", EntityName: "EntityMBeginnerCampaign", Keys: []columnSpec{column("BeginnerCampaignId", 0)}, Times: []columnSpec{column("BeginnerJudgeStartDatetime", 1), column("BeginnerJudgeEndDatetime", 2)}},
	{Name: "m_big_hunt_schedule", EntityName: "EntityMBigHuntSchedule", Keys: []columnSpec{column("BigHuntScheduleId", 0)}, Times: []columnSpec{column("NoticeStartDatetime", 1), column("ChallengeStartDatetime", 2), column("ChallengeEndDatetime", 3)}},
	{Name: "m_cage_ornament", EntityName: "EntityMCageOrnament", Keys: []columnSpec{column("CageOrnamentId", 0)}, Times: []columnSpec{column("StartDatetime", 1), column("EndDatetime", 2)}},
	{Name: "m_comeback_campaign", EntityName: "EntityMComebackCampaign", Keys: []columnSpec{column("ComebackCampaignId", 0)}, Times: []columnSpec{column("ComebackJudgeStartDatetime", 1), column("ComebackJudgeEndDatetime", 2)}},
	{Name: "m_consumable_item_term", EntityName: "EntityMConsumableItemTerm", Keys: []columnSpec{column("ConsumableItemTermId", 0)}, Times: []columnSpec{column("StartDatetime", 1), column("EndDatetime", 2)}},
	{Name: "m_costume_collection_bonus", EntityName: "EntityMCostumeCollectionBonus", Keys: []columnSpec{column("CollectionBonusId", 0), column("CollectionBonusTextId", 1), column("CollectionBonusGroupId", 2)}, Times: []columnSpec{column("StartDatetime", 5), column("EndDatetime", 6)}},
	{Name: "m_dokan", EntityName: "EntityMDokan", Keys: []columnSpec{column("DokanId", 0), column("SortOrder", 1), column("DokanType", 2)}, Times: []columnSpec{column("StartDatetime", 3), column("EndDatetime", 4)}},
	{Name: "m_enhance_campaign", EntityName: "EntityMEnhanceCampaign", Keys: []columnSpec{column("EnhanceCampaignId", 0), column("EnhanceCampaignTargetGroupId", 1), column("EnhanceCampaignEffectType", 2)}, Times: []columnSpec{column("StartDatetime", 4), column("EndDatetime", 5)}},
	{Name: "m_event_quest_chapter", EntityName: "EntityMEventQuestChapter", Keys: []columnSpec{column("EventQuestChapterId", 0), column("EventQuestType", 1), column("SortOrder", 2), column("NameEventQuestTextId", 3), column("BannerAssetId", 4)}, Times: []columnSpec{column("StartDatetime", 8), column("EndDatetime", 9)}},
	{Name: "m_event_quest_daily_group", EntityName: "EntityMEventQuestDailyGroup", Keys: []columnSpec{column("EventQuestDailyGroupId", 0)}, Times: []columnSpec{column("StartDatetime", 1), column("EndDatetime", 2)}},
	{Name: "m_event_quest_guerrilla_free_open", EntityName: "EntityMEventQuestGuerrillaFreeOpen", Keys: []columnSpec{column("EventQuestGuerrillaFreeOpenId", 0), column("OpenMinutes", 1), column("DailyOpenMaxCount", 2)}, Times: []columnSpec{column("StartDatetime", 3), column("EndDatetime", 4)}},
	{Name: "m_event_quest_labyrinth_season", EntityName: "EntityMEventQuestLabyrinthSeason", Keys: []columnSpec{column("EventQuestChapterId", 0), column("SeasonNumber", 1)}, Times: []columnSpec{column("StartDatetime", 2), column("EndDatetime", 3)}},
	{Name: "m_event_quest_limit_content", EntityName: "EntityMEventQuestLimitContent", Keys: []columnSpec{column("EventQuestLimitContentId", 0), column("CostumeId", 1), column("UnlockEvaluateConditionId", 2)}, Times: []columnSpec{column("StartDatetime", 5), column("EndDatetime", 6)}},
	{Name: "m_event_quest_limit_content_deck_restriction", EntityName: "EntityMEventQuestLimitContentDeckRestriction", Keys: []columnSpec{column("EventQuestLimitContentDeckRestrictionId", 0), column("GroupIndex", 1), column("EventQuestLimitContentDeckRestrictionTargetId", 2)}, Times: []columnSpec{column("StartDatetime", 3), column("EndDatetime", 4)}},
	{Name: "m_gimmick_sequence_schedule", EntityName: "EntityMGimmickSequenceSchedule", Keys: []columnSpec{column("GimmickSequenceScheduleId", 0)}, Times: []columnSpec{column("StartDatetime", 1), column("EndDatetime", 2)}},
	{Name: "m_important_item_effect", EntityName: "EntityMImportantItemEffect", Keys: []columnSpec{column("ImportantItemEffectId", 0), column("ImportantItemEffectGroupingId", 1), column("Priority", 2)}, Times: []columnSpec{column("StartDatetime", 5), column("EndDatetime", 6)}},
	{Name: "m_login_bonus", EntityName: "EntityMLoginBonus", Keys: []columnSpec{column("LoginBonusId", 0), column("SortOrder", 1), column("LoginBonusStartConditionId", 2), column("LoginBonusAssetName", 7)}, Times: []columnSpec{column("StartDatetime", 4), column("EndDatetime", 5), column("StampReceiveEndDatetime", 6)}},
	{Name: "m_maintenance", EntityName: "EntityMMaintenance", Keys: []columnSpec{column("MaintenanceId", 0)}, Times: []columnSpec{column("StartDatetime", 1), column("EndDatetime", 2)}},
	{Name: "m_mission_pass", EntityName: "EntityMMissionPass", Keys: []columnSpec{column("MissionPassId", 0)}, Times: []columnSpec{column("StartDatetime", 1), column("EndDatetime", 2)}},
	{Name: "m_mission_term", EntityName: "EntityMMissionTerm", Keys: []columnSpec{column("MissionTermId", 0)}, Times: []columnSpec{column("StartDatetime", 1), column("EndDatetime", 2)}},
	{Name: "m_mom_banner", EntityName: "EntityMMomBanner", Keys: []columnSpec{column("MomBannerId", 0), column("SortOrderDesc", 1), column("DestinationDomainType", 2), column("DestinationDomainId", 3), column("BannerAssetName", 4)}, Times: []columnSpec{column("StartDatetime", 6), column("EndDatetime", 7)}},
	{Name: "m_mom_point_banner", EntityName: "EntityMMomPointBanner", Keys: []columnSpec{column("MomPointBannerId", 0), column("BannerAssetId", 1), column("DestinationInformationId", 2)}, Times: []columnSpec{column("StartDatetime", 3), column("EndDatetime", 4)}},
	{Name: "m_navi_cut_in", EntityName: "EntityMNaviCutIn", Keys: []columnSpec{column("NaviCutInId", 0), column("RelatedCutInFunctionType", 1), column("SortOrder", 2)}, Times: []columnSpec{column("StartDatetime", 3), column("EndDatetime", 4)}},
	{Name: "m_omikuji", EntityName: "EntityMOmikuji", Keys: []columnSpec{column("OmikujiId", 0)}, Times: []columnSpec{column("StartDatetime", 1), column("EndDatetime", 2)}},
	{Name: "m_portal_cage_access_point_function_group_schedule", EntityName: "EntityMPortalCageAccessPointFunctionGroupSchedule", Keys: []columnSpec{column("PortalCageAccessPointFunctionGroupScheduleId", 0), column("PriorityDesc", 1), column("AccessPointType", 2)}, Times: []columnSpec{column("StartDatetime", 4), column("EndDatetime", 5)}},
	{Name: "m_possession_acquisition_route", EntityName: "EntityMPossessionAcquisitionRoute", Keys: []columnSpec{column("PossessionType", 0), column("PossessionId", 1), column("SortOrder", 2)}, Times: []columnSpec{column("StartDatetime", 6), column("EndDatetime", 7)}},
	{Name: "m_premium_item", EntityName: "EntityMPremiumItem", Keys: []columnSpec{column("PremiumItemId", 0), column("PremiumItemType", 1)}, Times: []columnSpec{column("StartDatetime", 2), column("EndDatetime", 3)}},
	{Name: "m_pvp_season", EntityName: "EntityMPvpSeason", Keys: []columnSpec{column("PvpSeasonId", 0), column("NameAssetPath", 1)}, Times: []columnSpec{column("SeasonStartDatetime", 2), column("SeasonEndDatetime", 3)}},
	{Name: "m_quest_bonus_term_group", EntityName: "EntityMQuestBonusTermGroup", Keys: []columnSpec{column("QuestBonusTermGroupId", 0), column("SortOrder", 1)}, Times: []columnSpec{column("StartDatetime", 2), column("EndDatetime", 3)}},
	{Name: "m_quest_campaign", EntityName: "EntityMQuestCampaign", Keys: []columnSpec{column("QuestCampaignId", 0), column("QuestCampaignTargetGroupId", 1), column("QuestCampaignEffectGroupId", 2)}, Times: []columnSpec{column("StartDatetime", 3), column("EndDatetime", 4)}},
	{Name: "m_quest_schedule", EntityName: "EntityMQuestSchedule", Keys: []columnSpec{column("QuestScheduleId", 0), column("QuestScheduleCronExpression", 1)}, Times: []columnSpec{column("StartDatetime", 2), column("EndDatetime", 3)}},
	{Name: "m_shop", EntityName: "EntityMShop", Keys: []columnSpec{column("ShopId", 0), column("ShopGroupType", 1), column("SortOrderInShopGroup", 2), column("NameShopTextId", 4), column("ShopItemCellGroupId", 7)}, Times: []columnSpec{column("StartDatetime", 9), column("EndDatetime", 10)}},
	{Name: "m_shop_item_cell_term", EntityName: "EntityMShopItemCellTerm", Keys: []columnSpec{column("ShopItemCellTermId", 0)}, Times: []columnSpec{column("StartDatetime", 1), column("EndDatetime", 2)}},
	{Name: "m_tip", EntityName: "EntityMTip", Keys: []columnSpec{column("TipId", 0), column("TitleTipTextId", 1), column("ContentTipTextId", 2)}, Times: []columnSpec{column("StartDatetime", 5), column("EndDatetime", 6)}},
	{Name: "m_title_flow_movie", EntityName: "EntityMTitleFlowMovie", Keys: []columnSpec{column("TitleFlowMovieId", 0), column("MovieId", 1)}, Times: []columnSpec{column("StartDatetime", 2), column("EndDatetime", 3)}},
	{Name: "m_webview_mission", EntityName: "EntityMWebviewMission", Keys: []columnSpec{column("WebviewMissionId", 0), column("TitleTextId", 1), column("WebviewMissionType", 2)}, Times: []columnSpec{column("StartDatetime", 4), column("EndDatetime", 5)}},
	{Name: "m_webview_panel_mission", EntityName: "EntityMWebviewPanelMission", Keys: []columnSpec{column("WebviewPanelMissionId", 0), column("Page", 1), column("WebviewPanelMissionPageId", 2)}, Times: []columnSpec{column("StartDatetime", 3), column("EndDatetime", 4)}},
}
