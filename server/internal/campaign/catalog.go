package campaign

import (
	"fmt"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/utils"
)

type Catalog struct {
	enhance  []enhanceRow
	quest    []questRow
	beginner []beginnerRow
	comeback []comebackRow
}

type enhanceRow struct {
	effectType  EnhanceCampaignEffectType
	effectValue int32
	targets     []enhanceMatch
	startMillis int64
	endMillis   int64
	userStatus  TargetUserStatusType
}

type enhanceMatch struct {
	t EnhanceCampaignTargetType
	v int32
}

type questRow struct {
	effectType  QuestCampaignEffectType
	effectValue int32
	bonusItems  []BonusDrop
	targets     []questMatch
	startMillis int64
	endMillis   int64
	userStatus  TargetUserStatusType
}

type questMatch struct {
	t QuestCampaignTargetType
	v int32
}

func Load() (*Catalog, error) {
	enhance, err := loadEnhanceRows()
	if err != nil {
		return nil, fmt.Errorf("load enhance campaigns: %w", err)
	}
	quest, err := loadQuestRows()
	if err != nil {
		return nil, fmt.Errorf("load quest campaigns: %w", err)
	}
	beginner, err := loadBeginnerRows()
	if err != nil {
		return nil, fmt.Errorf("load beginner campaigns: %w", err)
	}
	comeback, err := loadComebackRows()
	if err != nil {
		return nil, fmt.Errorf("load comeback campaigns: %w", err)
	}
	return &Catalog{enhance: enhance, quest: quest, beginner: beginner, comeback: comeback}, nil
}

func (c *Catalog) EnhanceCount() int { return len(c.enhance) }
func (c *Catalog) QuestCount() int   { return len(c.quest) }

func loadBeginnerRows() ([]beginnerRow, error) {
	rows, err := utils.ReadTable[masterdata.EntityMBeginnerCampaign]("m_beginner_campaign")
	if err != nil {
		return nil, err
	}
	out := make([]beginnerRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, beginnerRow{
			campaignId:       r.BeginnerCampaignId,
			judgeStartMillis: r.BeginnerJudgeStartDatetime,
			judgeEndMillis:   r.BeginnerJudgeEndDatetime,
			grantDays:        r.GrantCampaignTermDayCount,
			unlockQuestId:    r.CampaignUnlockQuestId,
		})
	}
	return out, nil
}

func loadComebackRows() ([]comebackRow, error) {
	rows, err := utils.ReadTable[masterdata.EntityMComebackCampaign]("m_comeback_campaign")
	if err != nil {
		return nil, err
	}
	out := make([]comebackRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, comebackRow{
			campaignId:       r.ComebackCampaignId,
			judgeStartMillis: r.ComebackJudgeStartDatetime,
			judgeEndMillis:   r.ComebackJudgeEndDatetime,
			judgeDays:        r.ComebackJudgeDayCount,
			grantDays:        r.GrantCampaignTermDayCount,
			unlockQuestId:    r.CampaignUnlockQuestId,
			gradeGroupId:     r.ComebackCampaignGradeGroupId,
		})
	}
	return out, nil
}

func loadEnhanceRows() ([]enhanceRow, error) {
	campaigns, err := utils.ReadTable[masterdata.EntityMEnhanceCampaign]("m_enhance_campaign")
	if err != nil {
		return nil, err
	}
	targets, err := utils.ReadTable[masterdata.EntityMEnhanceCampaignTargetGroup]("m_enhance_campaign_target_group")
	if err != nil {
		return nil, err
	}

	byGroup := make(map[int32][]enhanceMatch, len(targets))
	for _, t := range targets {
		byGroup[t.EnhanceCampaignTargetGroupId] = append(byGroup[t.EnhanceCampaignTargetGroupId], enhanceMatch{
			t: EnhanceCampaignTargetType(t.EnhanceCampaignTargetType),
			v: t.EnhanceCampaignTargetValue,
		})
	}

	rows := make([]enhanceRow, 0, len(campaigns))
	for _, c := range campaigns {
		grp := byGroup[c.EnhanceCampaignTargetGroupId]
		if len(grp) == 0 {
			continue
		}
		rows = append(rows, enhanceRow{
			effectType:  EnhanceCampaignEffectType(c.EnhanceCampaignEffectType),
			effectValue: c.EnhanceCampaignEffectValue / 10,
			targets:     grp,
			startMillis: c.StartDatetime,
			endMillis:   c.EndDatetime,
			userStatus:  TargetUserStatusType(c.TargetUserStatusType),
		})
	}
	return rows, nil
}

func loadQuestRows() ([]questRow, error) {
	campaigns, err := utils.ReadTable[masterdata.EntityMQuestCampaign]("m_quest_campaign")
	if err != nil {
		return nil, err
	}
	targets, err := utils.ReadTable[masterdata.EntityMQuestCampaignTargetGroup]("m_quest_campaign_target_group")
	if err != nil {
		return nil, err
	}
	effects, err := utils.ReadTable[masterdata.EntityMQuestCampaignEffectGroup]("m_quest_campaign_effect_group")
	if err != nil {
		return nil, err
	}
	itemGroups, err := utils.ReadTable[masterdata.EntityMQuestCampaignTargetItemGroup]("m_quest_campaign_target_item_group")
	if err != nil {
		return nil, err
	}

	targetsByGroup := make(map[int32][]questMatch, len(targets))
	for _, t := range targets {
		targetsByGroup[t.QuestCampaignTargetGroupId] = append(targetsByGroup[t.QuestCampaignTargetGroupId], questMatch{
			t: QuestCampaignTargetType(t.QuestCampaignTargetType),
			v: t.QuestCampaignTargetValue,
		})
	}

	bonusByGroup := make(map[int32][]BonusDrop, len(itemGroups))
	for _, ig := range itemGroups {
		bonusByGroup[ig.QuestCampaignTargetItemGroupId] = append(bonusByGroup[ig.QuestCampaignTargetItemGroupId], BonusDrop{
			PossessionType: ig.PossessionType,
			PossessionId:   ig.PossessionId,
			Count:          ig.Count,
		})
	}

	effectByGroup := make(map[int32]masterdata.EntityMQuestCampaignEffectGroup, len(effects))
	for _, e := range effects {
		effectByGroup[e.QuestCampaignEffectGroupId] = e
	}

	var rows []questRow
	for _, c := range campaigns {
		grp := targetsByGroup[c.QuestCampaignTargetGroupId]
		if len(grp) == 0 {
			continue
		}
		eff, ok := effectByGroup[c.QuestCampaignEffectGroupId]
		if !ok {
			continue
		}
		rows = append(rows, questRow{
			effectType:  QuestCampaignEffectType(eff.QuestCampaignEffectType),
			effectValue: eff.QuestCampaignEffectValue,
			bonusItems:  bonusByGroup[eff.QuestCampaignTargetItemGroupId],
			targets:     grp,
			startMillis: c.StartDatetime,
			endMillis:   c.EndDatetime,
			userStatus:  TargetUserStatusType(c.TargetUserStatusType),
		})
	}
	return rows, nil
}

func (r enhanceRow) isActive(f Filter) bool {
	if f.NowMillis < r.startMillis || f.NowMillis > r.endMillis {
		return false
	}
	return r.userStatus == TargetUserStatusAll || r.userStatus == f.UserStatus
}

func (r questRow) isActive(f Filter) bool {
	if f.NowMillis < r.startMillis || f.NowMillis > r.endMillis {
		return false
	}
	return r.userStatus == TargetUserStatusAll || r.userStatus == f.UserStatus
}
