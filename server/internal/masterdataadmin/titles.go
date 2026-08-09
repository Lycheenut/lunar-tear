package masterdataadmin

import (
	"fmt"
	"strconv"
	"strings"

	"lunar-tear/server/internal/masterdata/memorydb"
)

type translatedText map[int64]map[string]string

type questCampaignEffect struct {
	effectType  int64
	effectValue int64
}

type localizedTitlePart struct {
	key       string
	targetKey string
	parameter string
	suffix    string
}

// Costume and weapon enhancement use a 20‰ base Glorious Success rate.
// Campaign values are stored at ten times the probability-per-mille value.
const enhanceGreatSuccessBaseValue int64 = 200

type titleResolver struct {
	texts                localizationIndex
	webviewTitles        translatedText
	dokanTitles          translatedText
	chapterTextIDs       map[int64]int64
	dailyGroupChapters   map[int64][]int64
	limitContentChapters map[int64][]int64
	restrictionContents  map[int64][]int64
	missionTermTextIDs   map[int64]int64
	shopTermTextIDs      map[int64]int64
	shopTextIDs          map[int64]int64
	consumableTermKeys   map[int64][]string
	importantEffectTexts map[int64]int64
	shopRelationsByShop  map[int64][]ShopRelation
	shopRelationsByTerm  map[int64][]ShopRelation
	dokanGroupTextIDs    map[int64]int64
	enhanceTargetTypes   map[int64][]int64
	questEffects         map[int64]questCampaignEffect
	questTargetTypes     map[int64][]int64
}

func newTitleResolver(file *memorydb.File, texts localizationIndex) *titleResolver {
	resolver := &titleResolver{
		texts:                texts,
		webviewTitles:        readTranslatedText(file, "m_webview_mission_title_text", 0, 1, 2),
		dokanTitles:          readTranslatedText(file, "m_dokan_text", 0, 1, 2),
		chapterTextIDs:       make(map[int64]int64),
		dailyGroupChapters:   make(map[int64][]int64),
		limitContentChapters: make(map[int64][]int64),
		restrictionContents:  make(map[int64][]int64),
		missionTermTextIDs:   make(map[int64]int64),
		shopTermTextIDs:      make(map[int64]int64),
		shopTextIDs:          make(map[int64]int64),
		consumableTermKeys:   make(map[int64][]string),
		importantEffectTexts: make(map[int64]int64),
		dokanGroupTextIDs:    make(map[int64]int64),
		enhanceTargetTypes:   make(map[int64][]int64),
		questEffects:         make(map[int64]questCampaignEffect),
		questTargetTypes:     make(map[int64][]int64),
	}

	for _, row := range readRows(file, "m_event_quest_chapter") {
		resolver.putPair(resolver.chapterTextIDs, row, 0, 3)
	}
	for _, row := range readRows(file, "m_event_quest_daily_group_target_chapter") {
		resolver.appendPair(resolver.dailyGroupChapters, row, 0, 2)
	}
	for _, row := range readRows(file, "m_event_quest_chapter_limit_content_relation") {
		resolver.appendPair(resolver.limitContentChapters, row, 1, 0)
	}
	for _, row := range readRows(file, "m_event_quest_limit_content") {
		resolver.appendPair(resolver.restrictionContents, row, 7, 0)
	}
	for _, row := range readRows(file, "m_mission") {
		resolver.putPairIfAbsent(resolver.missionTermTextIDs, row, 12, 5)
	}
	for _, row := range readRows(file, "m_shop") {
		resolver.putPair(resolver.shopTextIDs, row, 0, 4)
	}
	for _, row := range readRows(file, "m_consumable_item") {
		termID, termOK := integerAt(row, 4)
		categoryID, categoryOK := integerAt(row, 6)
		variationID, variationOK := integerAt(row, 7)
		if termOK && categoryOK && variationOK && termID != 0 {
			key := fmt.Sprintf("consumable_item.name.%d", categoryID*1000+variationID)
			resolver.consumableTermKeys[termID] = append(resolver.consumableTermKeys[termID], key)
		}
	}
	for _, row := range readRows(file, "m_important_item") {
		resolver.putPair(resolver.importantEffectTexts, row, 6, 1)
	}

	cellItems := make(map[int64]int64)
	for _, row := range readRows(file, "m_shop_item_cell") {
		resolver.putPair(cellItems, row, 0, 2)
	}
	itemTextIDs := make(map[int64]int64)
	for _, row := range readRows(file, "m_shop_item") {
		resolver.putPair(itemTextIDs, row, 0, 1)
	}
	for _, row := range readRows(file, "m_shop_item_cell_group") {
		termID, termOK := integerAt(row, 3)
		cellID, cellOK := integerAt(row, 1)
		itemID, itemOK := cellItems[cellID]
		textID, textOK := itemTextIDs[itemID]
		if termOK && cellOK && itemOK && textOK {
			if _, exists := resolver.shopTermTextIDs[termID]; !exists {
				resolver.shopTermTextIDs[termID] = textID
			}
		}
	}
	for _, row := range readRows(file, "m_dokan_content_group") {
		resolver.putPairIfAbsent(resolver.dokanGroupTextIDs, row, 0, 4)
	}
	for _, row := range readRows(file, "m_enhance_campaign_target_group") {
		resolver.appendPair(resolver.enhanceTargetTypes, row, 0, 2)
	}
	for _, row := range readRows(file, "m_quest_campaign_effect_group") {
		groupID, groupOK := integerAt(row, 0)
		effectType, typeOK := integerAt(row, 1)
		effectValue, valueOK := integerAt(row, 2)
		if groupOK && typeOK && valueOK {
			resolver.questEffects[groupID] = questCampaignEffect{effectType: effectType, effectValue: effectValue}
		}
	}
	for _, row := range readRows(file, "m_quest_campaign_target_group") {
		resolver.appendPair(resolver.questTargetTypes, row, 0, 2)
	}
	resolver.loadShopRelations(file)
	return resolver
}

func (r *titleResolver) resolve(table string, row []interface{}) map[string]string {
	var key string
	switch table {
	case "m_appeal_dialog":
		if id, ok := integerAt(row, 6); ok {
			key = fmt.Sprintf("appeal.dialog.title.%06d", id)
		}
	case "m_costume_collection_bonus":
		if id, ok := integerAt(row, 1); ok {
			key = fmt.Sprintf("CollectionBonus.Effect.%d", id)
		}
	case "m_event_quest_chapter":
		key = integerKey(row, 3, "quest.event.chapter_title.%d")
	case "m_enhance_campaign":
		if titles := r.enhanceCampaignTitles(row); len(titles) != 0 {
			return titles
		}
	case "m_event_quest_daily_group":
		if groupID, ok := integerAt(row, 3); ok {
			return r.eventChapterTitles(r.dailyGroupChapters[groupID])
		}
	case "m_event_quest_labyrinth_season":
		if chapterID, ok := integerAt(row, 0); ok {
			return r.eventChapterTitles([]int64{chapterID})
		}
	case "m_event_quest_limit_content":
		if contentID, ok := integerAt(row, 0); ok {
			return r.eventChapterTitles(r.limitContentChapters[contentID])
		}
	case "m_event_quest_limit_content_deck_restriction":
		if restrictionID, ok := integerAt(row, 0); ok {
			var chapterIDs []int64
			for _, contentID := range r.restrictionContents[restrictionID] {
				chapterIDs = append(chapterIDs, r.limitContentChapters[contentID]...)
			}
			return r.eventChapterTitles(chapterIDs)
		}
	case "m_login_bonus":
		if assetName, ok := stringAt(row, 7); ok {
			key = "loginbonus.title." + assetName
		}
	case "m_mission_term":
		if termID, ok := integerAt(row, 0); ok {
			key = fmt.Sprintf("mission.name.%d", r.missionTermTextIDs[termID])
		}
	case "m_pvp_season":
		if assetName, ok := stringAt(row, 1); ok {
			if number, err := strconv.Atoi(assetName); err == nil {
				key = fmt.Sprintf("pvp.season.name.%03d", number)
			}
		}
	case "m_quest_campaign":
		if titles := r.questCampaignTitles(row); len(titles) != 0 {
			return titles
		}
	case "m_consumable_item_term":
		if termID, ok := integerAt(row, 0); ok {
			return r.titlesForKeys(r.consumableTermKeys[termID])
		}
	case "m_important_item_effect":
		if effectID, ok := integerAt(row, 0); ok {
			key = fmt.Sprintf("important_item.name.%d", r.importantEffectTexts[effectID])
		}
	case "m_mom_banner":
		return r.momBannerTitles(row)
	case "m_shop":
		key = integerKey(row, 4, "shop.name.%d")
	case "m_shop_item_cell_term":
		if termID, ok := integerAt(row, 0); ok {
			key = fmt.Sprintf("shop.item.name.%d", r.shopTermTextIDs[termID])
		}
	case "m_tip":
		key = integerKey(row, 1, "tip.%d")
	case "m_webview_mission":
		if textID, ok := integerAt(row, 1); ok {
			return cloneTitles(r.webviewTitles[textID])
		}
	case "m_dokan":
		if groupID, ok := integerAt(row, 5); ok {
			return cloneTitles(r.dokanTitles[r.dokanGroupTextIDs[groupID]])
		}
	}
	if key == "" {
		return nil
	}
	return r.byKey(key)
}

func (r *titleResolver) enhanceCampaignTitles(row []interface{}) map[string]string {
	targetGroupID, groupOK := integerAt(row, 1)
	effectType, typeOK := integerAt(row, 2)
	effectValue, valueOK := integerAt(row, 3)
	if !groupOK || !typeOK || !valueOK {
		return nil
	}

	targetTypes := r.enhanceTargetTypes[targetGroupID]
	if len(targetTypes) == 0 {
		return nil
	}
	if effectType != 1 && effectType != 2 {
		return nil
	}

	parts := make([]localizedTitlePart, 0, len(targetTypes))
	for _, targetType := range targetTypes {
		if effectType == 1 {
			parts = append(parts, localizedTitlePart{
				key:       "campaign.description.02.01",
				targetKey: fmt.Sprintf("campaign.target.02.%02d", enhanceTargetCategory(targetType)),
				suffix:    formatDecimal(effectValue, 100) + "%",
			})
			continue
		}
		suffix := "×" + formatDecimal(effectValue+enhanceGreatSuccessBaseValue, enhanceGreatSuccessBaseValue)
		if targetType == 3 || targetType == 31 || targetType == 32 {
			suffix = "+" + formatDecimal(effectValue, 100) + "%"
		}
		parts = append(parts, localizedTitlePart{
			key:    fmt.Sprintf("campaign.description.02.%02d.%02d", effectType, targetType),
			suffix: suffix,
		})
	}
	return r.localizedTitles(parts)
}

func (r *titleResolver) questCampaignTitles(row []interface{}) map[string]string {
	targetGroupID, groupOK := integerAt(row, 1)
	effectGroupID, effectOK := integerAt(row, 2)
	effect, found := r.questEffects[effectGroupID]
	if !groupOK || !effectOK || !found {
		return nil
	}

	parts := make([]localizedTitlePart, 0, len(r.questTargetTypes[targetGroupID]))
	parameter := formatDecimal(1000+effect.effectValue, 1000)
	suffix := ""
	if effect.effectType == 3 {
		parameter = formatDecimal(effect.effectValue, 1000)
		suffix = "×" + parameter
	} else if effect.effectType == 5 {
		parameter = ""
	}

	for _, targetType := range r.questTargetTypes[targetGroupID] {
		parts = append(parts, localizedTitlePart{
			key:       fmt.Sprintf("campaign.description.01.%02d.%02d", effect.effectType, targetType),
			parameter: parameter,
			suffix:    suffix,
		})
	}
	return r.localizedTitles(parts)
}

func enhanceTargetCategory(targetType int64) int64 {
	switch targetType {
	case 1, 11, 12, 13:
		return 1
	case 2, 21, 22, 23:
		return 2
	case 3, 31, 32:
		return 3
	default:
		return 0
	}
}

func formatDecimal(numerator, denominator int64) string {
	if denominator == 0 {
		return ""
	}
	return strconv.FormatFloat(float64(numerator)/float64(denominator), 'f', -1, 64)
}

func (r *titleResolver) momBannerTitles(row []interface{}) map[string]string {
	domainType, typeOK := integerAt(row, 2)
	domainID, idOK := integerAt(row, 3)
	assetName, assetOK := stringAt(row, 4)
	if typeOK && domainType == 1 && assetOK {
		if titles := r.byKey("gacha.title." + assetName); len(titles) != 0 {
			return titles
		}
		if value, ok := strings.CutPrefix(assetName, "limited_"); ok {
			return r.byKey("gacha.title.limitd_" + value)
		}
		return nil
	}
	if typeOK && idOK {
		switch domainType {
		case 2:
			return r.byKey(fmt.Sprintf("shop.name.%d", r.shopTextIDs[domainID]))
		case 22:
			if titles := r.byKey(fmt.Sprintf("mission.name.%d", r.missionTermTextIDs[domainID])); len(titles) != 0 {
				return titles
			}
		}
	}
	if assetOK {
		if key := numberedAssetKey(assetName, "event_mom_banner_", "quest.event.chapter_title.%d"); key != "" {
			return r.byKey(key)
		}
		for _, prefix := range []string{"mission_mom_banner_", "mission_"} {
			if key := numberedAssetKey(assetName, prefix, "mission.name.%d"); key != "" {
				return r.byKey(key)
			}
		}
	}
	return nil
}

func numberedAssetKey(assetName, prefix, format string) string {
	value, ok := strings.CutPrefix(assetName, prefix)
	if !ok {
		return ""
	}
	if number, _, found := strings.Cut(value, "_"); found {
		value = number
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return ""
	}
	return fmt.Sprintf(format, id)
}

func (r *titleResolver) titlesForKeys(keys []string) map[string]string {
	parts := make([]localizedTitlePart, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, localizedTitlePart{key: key})
	}
	return r.localizedTitles(parts)
}

func (r *titleResolver) localizedTitles(parts []localizedTitlePart) map[string]string {
	titles := make(map[string]string)
	for _, language := range supportedLanguages {
		var localized []string
		seen := make(map[string]bool)
		for _, part := range parts {
			value := r.texts[language][part.key]
			if value == "" {
				continue
			}
			value = strings.ReplaceAll(value, "{0}", part.parameter)
			if target := r.texts[language][part.targetKey]; target != "" {
				value += " (" + target + ")"
			}
			if part.suffix != "" {
				value += " · " + part.suffix
			}
			if !seen[value] {
				seen[value] = true
				localized = append(localized, value)
			}
		}
		if len(localized) != 0 {
			titles[language] = strings.Join(localized, " / ")
		}
	}
	if len(titles) == 0 {
		return nil
	}
	return titles
}

func (r *titleResolver) eventChapterTitles(chapterIDs []int64) map[string]string {
	keys := make([]string, 0, len(chapterIDs))
	for _, chapterID := range chapterIDs {
		textID, ok := r.chapterTextIDs[chapterID]
		if ok {
			keys = append(keys, fmt.Sprintf("quest.event.chapter_title.%d", textID))
		}
	}
	return r.titlesForKeys(keys)
}

func (r *titleResolver) byKey(key string) map[string]string {
	titles := make(map[string]string)
	for _, language := range supportedLanguages {
		if title := r.texts[language][key]; title != "" {
			titles[language] = title
		}
	}
	if len(titles) == 0 {
		return nil
	}
	return titles
}

func (r *titleResolver) putPair(target map[int64]int64, row []interface{}, keyIndex, valueIndex int) {
	key, keyOK := integerAt(row, keyIndex)
	value, valueOK := integerAt(row, valueIndex)
	if keyOK && valueOK && key != 0 {
		target[key] = value
	}
}

func (r *titleResolver) putPairIfAbsent(target map[int64]int64, row []interface{}, keyIndex, valueIndex int) {
	key, keyOK := integerAt(row, keyIndex)
	value, valueOK := integerAt(row, valueIndex)
	if keyOK && valueOK && key != 0 {
		if _, exists := target[key]; !exists {
			target[key] = value
		}
	}
}

func (r *titleResolver) appendPair(target map[int64][]int64, row []interface{}, keyIndex, valueIndex int) {
	key, keyOK := integerAt(row, keyIndex)
	value, valueOK := integerAt(row, valueIndex)
	if keyOK && valueOK && key != 0 {
		target[key] = append(target[key], value)
	}
}

func readRows(file *memorydb.File, table string) [][]interface{} {
	rows, exists, err := file.TableRows(table)
	if err != nil || !exists {
		return nil
	}
	return rows
}

func readTranslatedText(file *memorydb.File, table string, idIndex, languageIndex, textIndex int) translatedText {
	result := make(translatedText)
	for _, row := range readRows(file, table) {
		id, idOK := integerAt(row, idIndex)
		languageType, languageOK := integerAt(row, languageIndex)
		text, textOK := stringAt(row, textIndex)
		language := map[int64]string{1: "ja", 2: "en", 3: "ko"}[languageType]
		if !idOK || !languageOK || !textOK || language == "" || text == "" {
			continue
		}
		if result[id] == nil {
			result[id] = make(map[string]string)
		}
		result[id][language] = text
	}
	return result
}

func integerKey(row []interface{}, index int, format string) string {
	if value, ok := integerAt(row, index); ok && value != 0 {
		return fmt.Sprintf(format, value)
	}
	return ""
}

func integerAt(row []interface{}, index int) (int64, bool) {
	if index < 0 || index >= len(row) {
		return 0, false
	}
	value, err := valueAsInt64(row[index])
	return value, err == nil
}

func stringAt(row []interface{}, index int) (string, bool) {
	if index < 0 || index >= len(row) {
		return "", false
	}
	value, ok := row[index].(string)
	return value, ok && value != ""
}

func cloneTitles(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]string, len(source))
	for language, title := range source {
		result[language] = title
	}
	return result
}
