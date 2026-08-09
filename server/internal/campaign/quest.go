package campaign

func (c *Catalog) QuestStamina(t QuestTarget, f Filter) StaminaMul {
	return questPermilMin(c.quest, QuestEffectStaminaConsume, t, f)
}

func (c *Catalog) QuestDropRate(t QuestTarget, f Filter) DropRateMul {
	return DropRateMul{bonusPermil: questPermilMax(c.quest, QuestEffectDropRate, t, f)}
}

func (c *Catalog) QuestDropCount(t QuestTarget, f Filter) DropCountMul {
	return DropCountMul{bonusPermil: questPermilMax(c.quest, QuestEffectDropCount, t, f)}
}

func (c *Catalog) QuestGold(t QuestTarget, f Filter) GoldMul {
	return GoldMul{bonusPermil: questPermilMax(c.quest, QuestEffectClearRewardGold, t, f)}
}

func questPermilMax(rows []questRow, want QuestCampaignEffectType, t QuestTarget, f Filter) int32 {
	var max int32
	for _, r := range rows {
		if !r.isActive(f) || r.effectType != want || !matchesQuest(r.targets, t) {
			continue
		}
		if r.effectValue > max {
			max = r.effectValue
		}
	}
	return max
}

func (c *Catalog) QuestBonusDrops(t QuestTarget, f Filter) []BonusDrop {
	var out []BonusDrop
	for _, r := range c.quest {
		if !r.isActive(f) || r.effectType != QuestEffectDropItemAdd {
			continue
		}
		if !matchesQuest(r.targets, t) {
			continue
		}
		out = append(out, r.bonusItems...)
	}
	return out
}

func questPermilMin(rows []questRow, want QuestCampaignEffectType, t QuestTarget, f Filter) StaminaMul {
	min := int32(1000)
	for _, r := range rows {
		if !r.isActive(f) || r.effectType != want {
			continue
		}
		if !matchesQuest(r.targets, t) {
			continue
		}
		if r.effectValue < min {
			min = r.effectValue
		}
	}
	return StaminaMul{permil: min}
}

func matchesQuest(targets []questMatch, t QuestTarget) bool {
	for _, m := range targets {
		switch m.t {
		case QuestTargetWholeQuest:
			return true
		case QuestTargetQuestType:
			if int32(t.QuestType) == m.v {
				return true
			}
		case QuestTargetEventQuestType:
			if t.QuestType == QuestTypeEventQuest && t.EventQuestType == m.v {
				return true
			}
		case QuestTargetMainQuestChapterId:
			if t.QuestType == QuestTypeMainQuest && t.ChapterId == m.v {
				return true
			}
		case QuestTargetMainQuestQuestId:
			if t.QuestType == QuestTypeMainQuest && t.QuestId == m.v {
				return true
			}
		case QuestTargetSubQuestChapterId:
			if t.QuestType == QuestTypeEventQuest && t.ChapterId == m.v {
				return true
			}
		case QuestTargetSubQuestQuestId:
			if t.QuestType == QuestTypeEventQuest && t.QuestId == m.v {
				return true
			}
		}
	}
	return false
}
