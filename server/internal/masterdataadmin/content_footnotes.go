package masterdataadmin

import (
	"fmt"

	"lunar-tear/server/internal/masterdata/memorydb"
)

type eventChapterReference struct {
	chapterID int64
	startTime int64
	endTime   int64
}

func (r *titleResolver) loadCampaignReferences(file *memorydb.File) {
	for _, row := range readRows(file, "m_enhance_campaign_target_group") {
		r.appendCampaignTarget(r.enhanceTargets, row)
	}
	for _, row := range readRows(file, "m_quest_campaign_effect_group") {
		groupID, groupOK := integerAt(row, 0)
		effectType, typeOK := integerAt(row, 1)
		effectValue, valueOK := integerAt(row, 2)
		if groupOK && typeOK && valueOK {
			r.questEffects[groupID] = append(r.questEffects[groupID], questCampaignEffect{
				effectType: effectType, effectValue: effectValue,
			})
		}
	}
	for _, row := range readRows(file, "m_quest_campaign_target_group") {
		r.appendCampaignTarget(r.questTargets, row)
	}

	for _, row := range readRows(file, "m_character") {
		characterID, idOK := integerAt(row, 0)
		textID, textOK := integerAt(row, 2)
		if !idOK || !textOK {
			continue
		}
		if titles := r.firstTitles(fmt.Sprintf("character.name.%d", textID), fmt.Sprintf("character.name.%d", characterID)); len(titles) != 0 {
			r.characterTitlesByID[characterID] = titles
		}
	}

	for _, row := range readRows(file, "m_costume") {
		costumeID, idOK := integerAt(row, 0)
		characterID, characterOK := integerAt(row, 1)
		skeletonID, skeletonOK := integerAt(row, 4)
		variationID, variationOK := integerAt(row, 5)
		weaponType, weaponTypeOK := integerAt(row, 6)
		if !idOK || !skeletonOK || !variationOK {
			continue
		}
		assetName := fmt.Sprintf("ch%03d%03d", skeletonID, variationID)
		titles := r.firstTitles("costume.name.replace."+assetName, "costume.name."+assetName)
		if len(titles) == 0 {
			continue
		}
		r.costumeTitlesByID[costumeID] = titles
		if characterOK {
			r.costumesByCharacter[characterID] = append(r.costumesByCharacter[characterID], titles)
		}
		if weaponTypeOK {
			r.costumesByWeaponType[weaponType] = append(r.costumesByWeaponType[weaponType], titles)
		}
	}

	for _, row := range readRows(file, "m_weapon") {
		weaponID, idOK := integerAt(row, 0)
		categoryType, categoryOK := integerAt(row, 1)
		weaponType, typeOK := integerAt(row, 2)
		variationID, variationOK := integerAt(row, 3)
		attributeType, attributeOK := integerAt(row, 5)
		if !idOK || !categoryOK || !typeOK || !variationOK {
			continue
		}
		prefix := "wp"
		if categoryType == 2 {
			prefix = "mw"
		}
		assetName := fmt.Sprintf("%s%03d%03d", prefix, weaponType, variationID)
		titles := r.firstTitles(
			"weapon.name.replace."+assetName+".1",
			"weapon.name."+assetName+".1",
			"weapon.name.replace."+assetName+".2",
			"weapon.name."+assetName+".2",
		)
		if len(titles) == 0 {
			continue
		}
		r.weaponTitlesByID[weaponID] = titles
		r.weaponsByType[weaponType] = append(r.weaponsByType[weaponType], titles)
		if attributeOK {
			r.weaponsByAttribute[attributeType] = append(r.weaponsByAttribute[attributeType], titles)
		}
	}

	partsSeriesByGroup := make(map[int64]int64)
	for _, row := range readRows(file, "m_parts_group") {
		r.putPair(partsSeriesByGroup, row, 0, 1)
	}
	for _, row := range readRows(file, "m_parts") {
		partsID, idOK := integerAt(row, 0)
		partsGroupID, groupOK := integerAt(row, 2)
		seriesID, seriesOK := partsSeriesByGroup[partsGroupID]
		if idOK && groupOK && seriesOK {
			r.partsSeriesByPart[partsID] = seriesID
		}
	}

	eventQuestsBySequence := make(map[int64][]int64)
	for _, row := range readRows(file, "m_event_quest_sequence") {
		sequenceID, sequenceOK := integerAt(row, 0)
		questID, questOK := integerAt(row, 2)
		if sequenceOK && questOK {
			eventQuestsBySequence[sequenceID] = append(eventQuestsBySequence[sequenceID], questID)
		}
	}
	eventQuestsByGroup := make(map[int64][]int64)
	for _, row := range readRows(file, "m_event_quest_sequence_group") {
		groupID, groupOK := integerAt(row, 0)
		sequenceID, sequenceOK := integerAt(row, 2)
		if groupOK && sequenceOK {
			eventQuestsByGroup[groupID] = append(eventQuestsByGroup[groupID], eventQuestsBySequence[sequenceID]...)
		}
	}
	for _, row := range readRows(file, "m_event_quest_chapter") {
		chapterID, chapterOK := integerAt(row, 0)
		eventType, typeOK := integerAt(row, 1)
		groupID, groupOK := integerAt(row, 7)
		startTime, startOK := integerAt(row, 8)
		endTime, endOK := integerAt(row, 9)
		if !chapterOK {
			continue
		}
		if typeOK && startOK && endOK {
			r.eventChaptersByType[eventType] = append(r.eventChaptersByType[eventType], eventChapterReference{
				chapterID: chapterID, startTime: startTime, endTime: endTime,
			})
		}
		if groupOK {
			for _, questID := range eventQuestsByGroup[groupID] {
				r.eventChaptersByQuest[questID] = append(r.eventChaptersByQuest[questID], chapterID)
			}
		}
	}
}

func (r *titleResolver) appendCampaignTarget(target map[int64][]campaignTarget, row []interface{}) {
	groupID, groupOK := integerAt(row, 0)
	targetType, typeOK := integerAt(row, 2)
	targetValue, valueOK := integerAt(row, 3)
	if groupOK && typeOK && valueOK {
		target[groupID] = append(target[groupID], campaignTarget{targetType: targetType, targetValue: targetValue})
	}
}

func (r *titleResolver) firstTitles(keys ...string) map[string]string {
	for _, key := range keys {
		if titles := r.byKey(key); len(titles) != 0 {
			return titles
		}
	}
	return nil
}

func (r *titleResolver) resolveContentFootnotes(table string, row []interface{}, relations []ShopRelation) []map[string]string {
	var footnotes []map[string]string
	switch table {
	case "m_beginner_campaign":
		footnotes = append(footnotes, targetUserStatusTitles(3))
	case "m_comeback_campaign":
		footnotes = append(footnotes, targetUserStatusTitles(2))
	case "m_enhance_campaign":
		footnotes = append(footnotes, r.enhanceEffectFootnotes(row)...)
		if groupID, ok := integerAt(row, 1); ok {
			if targets := r.enhanceTargetTitles(r.enhanceTargets[groupID]); len(targets) != 0 {
				footnotes = append(footnotes, targets)
			}
		}
		if userStatus, ok := integerAt(row, 6); ok {
			if titles := targetUserStatusTitles(userStatus); len(titles) != 0 {
				footnotes = append(footnotes, titles)
			}
		}
	case "m_quest_campaign":
		footnotes = append(footnotes, r.questEffectFootnotes(row)...)
		if groupID, ok := integerAt(row, 1); ok {
			if targets := r.questTargetTitles(r.questTargets[groupID], row); len(targets) != 0 {
				footnotes = append(footnotes, targets)
			}
		}
		if userStatus, ok := integerAt(row, 5); ok {
			if titles := targetUserStatusTitles(userStatus); len(titles) != 0 {
				footnotes = append(footnotes, titles)
			}
		}
	case "m_login_bonus":
		if startCondition, ok := integerAt(row, 2); ok {
			if titles := loginBonusStartConditionTitles(startCondition); len(titles) != 0 {
				footnotes = append(footnotes, titles)
			}
		}
	case "m_shop_item_cell_term":
		var shops []map[string]string
		for _, relation := range relations {
			shops = append(shops, relation.ShopTitles)
		}
		if titles := combineLocalizedTitles(shops); len(titles) != 0 {
			footnotes = append(footnotes, titles)
		}
	}
	return footnotes
}

func loginBonusStartConditionTitles(startCondition int64) map[string]string {
	values := map[int64]map[string]string{
		0: {"en": "All Users", "ja": "全ユーザー", "ko": "전체 사용자"},
		4: {"en": "Returning Users", "ja": "カムバックユーザー", "ko": "복귀 사용자"},
		5: {"en": "New Users", "ja": "新規ユーザー", "ko": "신규 사용자"},
		6: {"en": "Returning Users (Grade Group 1)", "ja": "カムバックユーザー（グレードグループ1）", "ko": "복귀 사용자 (등급 그룹 1)"},
	}
	return cloneTitles(values[startCondition])
}

func targetUserStatusTitles(userStatus int64) map[string]string {
	values := map[int64]map[string]string{
		1: {"en": "All Users", "ja": "全ユーザー", "ko": "전체 사용자"},
		2: {"en": "Returning Users", "ja": "カムバックユーザー", "ko": "복귀 사용자"},
		3: {"en": "New Users", "ja": "新規ユーザー", "ko": "신규 사용자"},
	}
	return cloneTitles(values[userStatus])
}

func (r *titleResolver) enhanceEffectFootnotes(row []interface{}) []map[string]string {
	groupID, groupOK := integerAt(row, 1)
	effectType, typeOK := integerAt(row, 2)
	effectValue, valueOK := integerAt(row, 3)
	if !groupOK || !typeOK || !valueOK {
		return nil
	}
	targets := r.enhanceTargets[groupID]
	if len(targets) == 0 {
		if effectType == 1 {
			return neutralLocalizedTexts([]string{formatDecimal(effectValue, 100) + "%"})
		}
		if effectType == 2 {
			return neutralLocalizedTexts([]string{"+" + formatDecimal(effectValue, 100) + "%"})
		}
		return nil
	}
	values := make([]string, 0, len(targets))
	for _, target := range targets {
		switch effectType {
		case 1:
			values = append(values, formatDecimal(effectValue, 100)+"%")
		case 2:
			if target.targetType == 3 || target.targetType == 31 || target.targetType == 32 {
				values = append(values, "+"+formatDecimal(effectValue, 100)+"%")
			} else {
				values = append(values, "×"+formatDecimal(effectValue+enhanceGreatSuccessBaseValue, enhanceGreatSuccessBaseValue))
			}
		}
	}
	return neutralLocalizedTexts(values)
}

func (r *titleResolver) questEffectFootnotes(row []interface{}) []map[string]string {
	effectGroupID, ok := integerAt(row, 2)
	if !ok {
		return nil
	}
	var values []string
	for _, effect := range r.questEffects[effectGroupID] {
		switch effect.effectType {
		case 1, 2, 4:
			values = append(values, "×"+formatDecimal(1000+effect.effectValue, 1000))
		case 3:
			values = append(values, "×"+formatDecimal(effect.effectValue, 1000))
		}
	}
	return neutralLocalizedTexts(values)
}

func neutralLocalizedTexts(values []string) []map[string]string {
	seen := make(map[string]bool)
	var footnotes []map[string]string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		titles := make(map[string]string, len(supportedLanguages))
		for _, language := range supportedLanguages {
			titles[language] = value
		}
		footnotes = append(footnotes, titles)
	}
	return footnotes
}

func (r *titleResolver) enhanceTargetTitles(targets []campaignTarget) map[string]string {
	var titles []map[string]string
	for _, target := range targets {
		switch target.targetType {
		case 1, 2, 3:
			titles = append(titles, r.byKey(fmt.Sprintf("campaign.target.02.%02d", enhanceTargetCategory(target.targetType))))
		case 11:
			characterTitles := r.characterTitlesByID[target.targetValue]
			if len(characterTitles) == 0 {
				characterTitles = r.byKey(fmt.Sprintf("character.name.%d", target.targetValue))
			}
			if len(characterTitles) != 0 {
				titles = append(titles, characterTitles)
			} else {
				titles = append(titles, r.costumesByCharacter[target.targetValue]...)
			}
		case 12:
			titles = append(titles, r.costumesByWeaponType[target.targetValue]...)
		case 13:
			titles = append(titles, r.costumeTitlesByID[target.targetValue])
		case 21:
			titles = append(titles, r.weaponsByType[target.targetValue]...)
		case 22:
			titles = append(titles, r.weaponsByAttribute[target.targetValue]...)
		case 23:
			titles = append(titles, r.weaponTitlesByID[target.targetValue])
		case 31:
			titles = append(titles, r.byKey(fmt.Sprintf("parts.series.name.%d", target.targetValue)))
		case 32:
			if seriesID := r.partsSeriesByPart[target.targetValue]; seriesID != 0 {
				titles = append(titles, r.byKey(fmt.Sprintf("parts.series.name.%d", seriesID)))
			}
		}
	}
	return combineLocalizedTitles(titles)
}

func (r *titleResolver) questTargetTitles(targets []campaignTarget, row []interface{}) map[string]string {
	startTime, _ := integerAt(row, 3)
	endTime, _ := integerAt(row, 4)
	var titles []map[string]string
	for _, target := range targets {
		switch target.targetType {
		case 1:
			titles = append(titles, r.byKey("campaign.target.01.01"))
		case 2:
			titles = append(titles, questTypeTitles(target.targetValue))
		case 3:
			var chapterIDs []int64
			var allChapterIDs []int64
			for _, chapter := range r.eventChaptersByType[target.targetValue] {
				allChapterIDs = append(allChapterIDs, chapter.chapterID)
				if schedulesOverlap(startTime, endTime, chapter.startTime, chapter.endTime) {
					chapterIDs = append(chapterIDs, chapter.chapterID)
				}
			}
			if len(chapterIDs) == 0 {
				chapterIDs = allChapterIDs
			}
			chapterTitles := r.eventChapterTitles(chapterIDs)
			if len(chapterTitles) == 0 {
				chapterTitles = questTypeTitles(2)
			}
			titles = append(titles, chapterTitles)
		case 6:
			titles = append(titles, r.eventChapterTitles([]int64{target.targetValue}))
		case 7:
			titles = append(titles, r.eventChapterTitles(r.eventChaptersByQuest[target.targetValue]))
		}
	}
	return combineLocalizedTitles(titles)
}

func questTypeTitles(questType int64) map[string]string {
	values := map[int64]map[string]string{
		1: {"en": "Main Quests", "ja": "メインクエスト", "ko": "메인 퀘스트"},
		2: {"en": "Event Quests", "ja": "イベントクエスト", "ko": "이벤트 퀘스트"},
		3: {"en": "Extra Quests", "ja": "サブクエスト", "ko": "서브 퀘스트"},
		4: {"en": "Subjugations", "ja": "討伐戦", "ko": "토벌전"},
	}
	return cloneTitles(values[questType])
}

func schedulesOverlap(leftStart, leftEnd, rightStart, rightEnd int64) bool {
	return leftStart <= rightEnd && rightStart <= leftEnd
}

func combineLocalizedTitles(parts []map[string]string) map[string]string {
	combined := make(map[string]string)
	for _, language := range supportedLanguages {
		seen := make(map[string]bool)
		var values []string
		for _, part := range parts {
			value := localizedMapText(part, language)
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			values = append(values, value)
		}
		if len(values) != 0 {
			combined[language] = joinLocalizedValues(values)
		}
	}
	if len(combined) == 0 {
		return nil
	}
	return combined
}

func localizedMapText(titles map[string]string, language string) string {
	if value := titles[language]; value != "" {
		return value
	}
	if value := titles["en"]; value != "" {
		return value
	}
	for _, value := range titles {
		if value != "" {
			return value
		}
	}
	return ""
}

func joinLocalizedValues(values []string) string {
	result := ""
	for _, value := range values {
		if result != "" {
			result += " / "
		}
		result += value
	}
	return result
}
