package masterdataadmin

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"lunar-tear/server/internal/activitygroup"
	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

type ActivityMemberKind struct {
	Kind       string `json:"kind"`
	Label      string `json:"label"`
	Table      string `json:"table,omitempty"`
	Types      []int  `json:"types"`
	Redemption bool   `json:"redemption"`
}

var activityMemberKinds = []ActivityMemberKind{
	{"chapter", "EventQuestChapter", "m_event_quest_chapter", []int{2, 3}, false},
	{"premium", "Premium Gacha", "", []int{1}, false},
	{"banner", "MomBanner", "m_mom_banner", []int{1, 2, 3}, false},
	{"navi", "NaviCutIn", "m_navi_cut_in", []int{2, 3}, false},
	{"shop", "兑换商店", "m_shop", []int{1, 2}, true},
	{"mission", "活动任务", "m_mission_term", []int{2, 3}, true},
	{"event", "Event Gacha", "", []int{3}, true},
	{"term", "道具有效期", "m_consumable_item_term", []int{1, 2, 3}, true},
	{"medal", "碎片自动转换期限", "m_gacha_medal", []int{1}, true},
	{"tip", "Tip", "m_tip", []int{2, 3}, false},
}

type ActivityMemberOption struct {
	activitygroup.ActivityMember
	Titles           map[string]string `json:"titles,omitempty"`
	StartDatetime    int64             `json:"startDatetime"`
	EndDatetime      int64             `json:"endDatetime"`
	ChapterType      int64             `json:"chapterType,omitempty"`
	RelatedChapterID int64             `json:"relatedChapterId,omitempty"`
	PreviewPath      []string          `json:"previewPath,omitempty"`
	row              int
}

type ActivityGroupCatalog struct {
	Version string                 `json:"version"`
	Config  *activitygroup.Config  `json:"config"`
	Kinds   []ActivityMemberKind   `json:"kinds"`
	Options []ActivityMemberOption `json:"options"`
}

func LoadActivityGroups(path string, groups *activitygroup.Config, config *gacha.Config, entries []store.GachaCatalogEntry) (*ActivityGroupCatalog, error) {
	file, err := memorydb.OpenFile(path)
	if err != nil {
		return nil, err
	}
	resolver := newTitleResolver(file, loadLocalizationIndex(path))
	bannerPaths, err := activityBannerPreviewPaths(file)
	if err != nil {
		return nil, err
	}
	catalog := &ActivityGroupCatalog{Version: file.Version(), Config: groups, Kinds: activityMemberKinds}
	for _, kind := range activityMemberKinds {
		if kind.Table == "" {
			continue
		}
		spec, _ := findActivitySpec(kind.Table)
		table, _, err := tableFromFile(file, resolver, spec, true)
		if err != nil {
			return nil, err
		}
		for _, row := range table.Rows {
			id, _ := strconv.ParseInt(row.Identity[0].Value, 10, 64)
			option := ActivityMemberOption{ActivityMember: activitygroup.ActivityMember{Kind: kind.Kind, ID: id}, Titles: row.Titles, StartDatetime: row.Times["StartDatetime"], EndDatetime: row.Times["EndDatetime"], row: row.Index}
			if kind.Kind == "banner" {
				option.PreviewPath = bannerPaths[id]
			}
			if kind.Kind == "medal" {
				option.EndDatetime = row.Times["AutoConvertDatetime"]
			}
			if kind.Kind == "chapter" {
				option.ChapterType, _ = strconv.ParseInt(row.Values["EventQuestType"], 10, 64)
				if activitySourceType(option) == 0 {
					continue
				}
			}
			catalog.Options = append(catalog.Options, option)
		}
	}
	for id, banner := range config.Banners {
		titles := resolver.byKey("gacha.title." + banner.BannerAssetName)
		if len(titles) == 0 {
			if suffix, ok := strings.CutPrefix(banner.BannerAssetName, "limited_"); ok {
				titles = resolver.byKey("gacha.title.limitd_" + suffix)
			}
		}
		if len(titles) == 0 {
			titles = map[string]string{"en": banner.BannerAssetName}
		}
		catalog.Options = append(catalog.Options, ActivityMemberOption{ActivityMember: activitygroup.ActivityMember{Kind: "premium", ID: int64(id)}, Titles: titles, StartDatetime: banner.StartDatetime, EndDatetime: banner.EndDatetime, PreviewPath: []string{"gacha", banner.BannerAssetName, "banner.png"}})
	}
	for _, entry := range entries {
		if entry.GachaLabelType != model.GachaLabelEvent {
			continue
		}
		start, end := entry.StartDatetime, entry.EndDatetime
		if schedule, ok := config.EventSchedules[entry.GachaId]; ok {
			start, end = schedule.StartDatetime, schedule.EndDatetime
		}
		catalog.Options = append(catalog.Options, ActivityMemberOption{ActivityMember: activitygroup.ActivityMember{Kind: "event", ID: int64(entry.GachaId)}, Titles: resolver.byKey(fmt.Sprintf("quest.event.chapter_title.%d", entry.DescriptionTextId)), StartDatetime: start, EndDatetime: end, RelatedChapterID: int64(entry.RelatedEventQuestChapterId)})
	}
	sort.Slice(catalog.Options, func(i, j int) bool {
		if catalog.Options[i].Kind != catalog.Options[j].Kind {
			return catalog.Options[i].Kind < catalog.Options[j].Kind
		}
		return catalog.Options[i].ID < catalog.Options[j].ID
	})
	return catalog, nil
}

// Only initial generation infers currency terms from shops, Gacha, and quest
// drops. Saved configurations and schedule edits use explicit members.
func initialActivityCurrencyTerms(file *memorydb.File, index *relationIndex, entries []store.GachaCatalogEntry) map[int64][]int64 {
	termIDs := make(map[rowRef]int64)
	for id, ref := range index.termsByID {
		termIDs[ref] = id
	}
	shopTerms := make(map[rowRef][]int64)
	for currency, shops := range index.shopsByCurrency {
		if term, ok := index.termByCurrency[currency]; ok {
			for _, shop := range shops {
				shopTerms[shop] = append(shopTerms[shop], termIDs[term])
			}
		}
	}
	// Event Gacha links expose only the first ticket tier. The quest pickup
	// chain includes the silver/gold tickets used by the same Variation.
	tickets := make(map[int64]bool)
	for _, row := range readRows(file, "m_consumable_item") {
		if bonusInt(row, 1) == 200 {
			tickets[bonusInt(row, 0)] = true
		}
	}
	rewardTerms := make(map[int64]int64)
	for _, row := range readRows(file, "m_battle_drop_reward") {
		currency := bonusInt(row, 2)
		if bonusInt(row, 1) == int64(model.PossessionTypeConsumableItem) && tickets[currency] {
			if term, ok := index.termByCurrency[currency]; ok {
				rewardTerms[bonusInt(row, 0)] = termIDs[term]
			}
		}
	}
	pickupTerms := make(map[int64][]int64)
	for _, row := range readRows(file, "m_quest_pickup_reward_group") {
		if term := rewardTerms[bonusInt(row, 2)]; term != 0 {
			pickupTerms[bonusInt(row, 0)] = append(pickupTerms[bonusInt(row, 0)], term)
		}
	}
	questRows := readRows(file, "m_quest")
	chapterTerms := make(map[int64][]int64)
	for _, quest := range questBonusQuests(file) {
		chapterTerms[quest.ChapterID] = append(chapterTerms[quest.ChapterID], pickupTerms[bonusInt(questRows[quest.Row], 8)]...)
	}
	result := make(map[int64][]int64)
	for _, chapter := range readRows(file, "m_event_quest_chapter") {
		chapterID, _ := integerAt(chapter, 0)
		chapterType, _ := integerAt(chapter, 1)
		linkID, _ := integerAt(chapter, 5)
		link := index.eventLinks[linkID]
		domain, _ := integerAt(link, 1)
		destination, _ := integerAt(link, 2)
		if chapterType == 1 && domain == eventLinkDomainShop {
			if shop, ok := index.shopsByID[destination]; ok {
				result[chapterID] = append(result[chapterID], shopTerms[shop]...)
			}
		} else if chapterType == 2 && domain == int64(model.MomBannerDomainGacha) {
			currency, _ := integerAt(link, 4)
			if tickets[currency] {
				result[chapterID] = append(result[chapterID], chapterTerms[chapterID]...)
			}
		}
	}
	for _, entry := range entries {
		if entry.GachaLabelType != model.GachaLabelEvent {
			continue
		}
		ids := []int64{termIDs[index.termByCurrency[int64(entry.RequiredConsumableItemId)]]}
		for _, phase := range entry.PricePhases {
			if phase.PriceType == int32(model.PriceTypeConsumableItem) {
				ids = append(ids, termIDs[index.termByCurrency[int64(phase.PriceId)]])
			}
		}
		chapterID := int64(entry.RelatedEventQuestChapterId)
		result[chapterID] = append(result[chapterID], ids...)
	}
	for chapterID, ids := range result {
		seen := make(map[int64]bool)
		for _, id := range ids {
			if id != 0 {
				seen[id] = true
			}
		}
		result[chapterID] = sortedBonusIDs(seen)
	}
	return result
}

func activityKey(member activitygroup.ActivityMember) string {
	return fmt.Sprintf("%s:%d", member.Kind, member.ID)
}

// Paths omit the language segment, which the admin's shared preview renderer supplies.
func activityBannerPreviewPaths(file *memorydb.File) (map[int64][]string, error) {
	paths := make(map[int64][]string)
	loginAssets := make(map[int64]string)
	chapterAssets := make(map[int64]int64)
	for _, table := range []string{"m_login_bonus", "m_event_quest_chapter", "m_mom_banner"} {
		rows, _, err := file.TableRows(table)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			id, _ := integerAt(row, 0)
			switch table {
			case "m_login_bonus":
				loginAssets[id], _ = stringAt(row, 7)
			case "m_event_quest_chapter":
				chapterAssets[id], _ = integerAt(row, 4)
			case "m_mom_banner":
				domain, _ := integerAt(row, 2)
				destination, _ := integerAt(row, 3)
				asset, _ := stringAt(row, 4)
				switch {
				case domain == 1 && asset != "":
					paths[id] = []string{"gacha", asset, "mom_banner.png"}
				case domain == 21:
					if asset := loginAssets[destination]; asset != "" {
						paths[id] = []string{"login_bonus", "banner", asset + ".png"}
					}
				case domain == 25:
					if asset, ok := chapterAssets[destination]; ok {
						paths[id] = []string{"quest", "mom_banner", fmt.Sprintf("event_mom_banner_%03d.png", asset)}
					}
				case asset != "":
					paths[id] = []string{"mom_banner", "mom_banner_" + asset + ".png"}
				}
			}
		}
	}
	return paths, nil
}

func activitySourceType(option ActivityMemberOption) int {
	if option.Kind == "premium" {
		return activitygroup.TypePremium
	}
	if option.Kind == "chapter" {
		switch option.ChapterType {
		case 1:
			return activitygroup.TypeRecord
		case 2:
			return activitygroup.TypeVariation
		}
	}
	return 0
}

func activityMemberAllowed(option ActivityMemberOption, unitType int) bool {
	if option.Kind == "chapter" || option.Kind == "premium" {
		return activitySourceType(option) == unitType
	}
	for _, kind := range activityMemberKinds {
		if kind.Kind == option.Kind {
			for _, allowed := range kind.Types {
				if allowed == unitType {
					return true
				}
			}
		}
	}
	return false
}

func ValidateActivityGroups(config *activitygroup.Config, catalog *ActivityGroupCatalog) error {
	if config == nil || config.Version != activitygroup.ConfigVersion {
		return fmt.Errorf("活动组配置版本必须为 %d", activitygroup.ConfigVersion)
	}
	if len(config.Units) > 2000 || len(config.Groups) > 2000 {
		return fmt.Errorf("活动组或单位数量不能超过 2000")
	}
	options := make(map[string]ActivityMemberOption)
	for _, option := range catalog.Options {
		options[activityKey(option.ActivityMember)] = option
	}
	units := make(map[string]bool)
	for _, unit := range config.Units {
		if strings.TrimSpace(unit.ID) == "" || strings.TrimSpace(unit.Name) == "" || units[unit.ID] {
			return fmt.Errorf("活动单位 ID / 名称不能为空且 ID 不能重复")
		}
		if unit.Type < activitygroup.TypePremium || unit.Type > activitygroup.TypeVariation {
			return fmt.Errorf("活动单位 %s 必须为 Premium Gacha、Record 或 Variation", unit.Name)
		}
		units[unit.ID] = true
		seen := make(map[string]bool)
		kindCounts := make(map[string]int)
		sourceCount := 0
		variationChapters := make(map[int64]bool)
		for _, member := range unit.Members {
			key := activityKey(member)
			option, exists := options[key]
			if !exists || seen[key] {
				return fmt.Errorf("活动单位 %s 的成员 %s 不存在或重复", unit.Name, key)
			}
			seen[key] = true
			kindCounts[member.Kind]++
			if (member.Kind == "shop" || member.Kind == "event") && kindCounts[member.Kind] > 1 {
				return fmt.Errorf("活动单位 %s 的 %s 最多只能选择 1 个条目", unit.Name, member.Kind)
			}
			if !activityMemberAllowed(option, unit.Type) {
				return fmt.Errorf("活动单位 %s 不支持成员类型 %s", unit.Name, member.Kind)
			}
			if activitySourceType(option) == unit.Type {
				sourceCount++
			}
			if member.Kind == "chapter" && option.ChapterType == 2 {
				variationChapters[member.ID] = true
			}
		}
		if sourceCount != 1 {
			return fmt.Errorf("活动单位 %s 必须且只能选择 1 个对应类型的 Premium Gacha、Record 或 Variation 主条目", unit.Name)
		}
		for _, member := range unit.Members {
			if member.Kind == "event" && !variationChapters[options[activityKey(member)].RelatedChapterID] {
				return fmt.Errorf("Event Gacha %d 必须包含对应的 Variation 活动副本", member.ID)
			}
		}
	}
	groups := make(map[string]bool)
	for _, group := range config.Groups {
		if strings.TrimSpace(group.ID) == "" || strings.TrimSpace(group.Name) == "" || groups[group.ID] {
			return fmt.Errorf("活动组 ID / 名称不能为空且 ID 不能重复")
		}
		groups[group.ID] = true
		if len(group.UnitIDs) == 0 {
			return fmt.Errorf("活动组 %s 至少需要 1 个活动单位", group.Name)
		}
		seen := make(map[string]bool)
		for _, id := range group.UnitIDs {
			if !units[id] || seen[id] {
				return fmt.Errorf("活动组 %s 引用了不存在或重复的活动单位 %s", group.Name, id)
			}
			seen[id] = true
		}
	}
	return nil
}

// GenerateActivityGroups captures relationships once. It never changes a
// schedule. Version 1 membership is migrated once to the three unit types.
func GenerateActivityGroups(path string, existing *activitygroup.Config, config *gacha.Config, entries []store.GachaCatalogEntry) (*activitygroup.Config, error) {
	if existing != nil && existing.Version == activitygroup.ConfigVersion {
		return existing, nil
	}
	catalog, err := LoadActivityGroups(path, nil, config, entries)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return migrateActivityGroups(existing, catalog, config)
	}
	file, err := memorydb.OpenFile(path)
	if err != nil {
		return nil, err
	}
	p := &activityRelationReader{file: file, rows: make(map[string][][]interface{})}
	index, err := p.relations()
	if err != nil {
		return nil, err
	}
	groups := &activitygroup.Config{Version: activitygroup.ConfigVersion, Units: []activitygroup.ActivityUnit{}, Groups: []activitygroup.ActivityGroup{}}
	currencyTerms := initialActivityCurrencyTerms(file, index, entries)
	byRef := make(map[rowRef]activitygroup.ActivityMember)
	for _, option := range catalog.Options {
		for _, kind := range activityMemberKinds {
			if option.Kind == kind.Kind && kind.Table != "" {
				byRef[rowRef{kind.Table, option.row}] = option.ActivityMember
			}
		}
	}
	chapters, err := p.tableRows("m_event_quest_chapter")
	if err != nil {
		return nil, err
	}
	banners, err := p.tableRows("m_mom_banner")
	if err != nil {
		return nil, err
	}
	medals, err := p.tableRows("m_gacha_medal")
	if err != nil {
		return nil, err
	}
	for _, source := range catalog.Options {
		if source.Kind != "chapter" && source.Kind != "premium" {
			continue
		}
		unit := activitygroup.ActivityUnit{ID: activityKey(source.ActivityMember), Type: activitySourceType(source), Name: fmt.Sprintf("EventQuestChapter %d", source.ID), Members: []activitygroup.ActivityMember{source.ActivityMember}}
		for _, language := range []string{"en", "ja", "ko"} {
			if source.Titles[language] != "" {
				unit.Name = source.Titles[language]
				break
			}
		}
		seen := map[string]bool{activityKey(source.ActivityMember): true}
		addMember := func(member activitygroup.ActivityMember) {
			if member.Kind != "" && !seen[activityKey(member)] {
				seen[activityKey(member)] = true
				unit.Members = append(unit.Members, member)
			}
		}
		add := func(ref rowRef) { addMember(byRef[ref]) }
		addCurrency := func(id int64) {
			if term, ok := index.termByCurrency[id]; ok {
				add(term)
			}
		}
		if source.Kind == "chapter" {
			row := chapters[source.row]
			textID, _ := integerAt(row, 3)
			linkID, _ := integerAt(row, 5)
			for _, banner := range index.eventBannersByText[textID] {
				if overlap, _ := p.rowsOverlap(rowRef{"m_event_quest_chapter", source.row}, banner); overlap {
					add(banner)
				}
			}
			if link := index.eventLinks[linkID]; link != nil {
				domain, _ := integerAt(link, 1)
				destination, _ := integerAt(link, 2)
				if domain == eventLinkDomainShop && unit.Type == activitygroup.TypeRecord {
					if shop, ok := index.shopsByID[destination]; ok {
						add(shop)
					}
				}
				possessionType, _ := integerAt(link, 3)
				currencyID, _ := integerAt(link, 4)
				if possessionType == int64(model.PossessionTypeConsumableItem) {
					addCurrency(currencyID)
				}
			}
			for _, id := range currencyTerms[source.ID] {
				add(index.termsByID[id])
			}
			for _, term := range index.missionTermsByChapter[source.ID] {
				add(term)
				for _, banner := range index.missionBannersByTerm[byRef[term].ID] {
					add(banner)
				}
			}
			navis, err := p.selectNaviCutIns(index.naviCutInsByChapter[source.ID], source.StartDatetime)
			if err != nil {
				return nil, err
			}
			for _, navi := range navis {
				add(navi)
			}
			for _, option := range catalog.Options {
				if option.Kind == "event" && option.RelatedChapterID == source.ID && source.ChapterType == 2 {
					addMember(option.ActivityMember)
				}
				// Time equality is only a safe seed when exactly one chapter owns
				// the interval. Ambiguous Tips remain available for manual selection.
				if option.Kind == "tip" && source.StartDatetime > 0 && source.StartDatetime == option.StartDatetime && source.EndDatetime == option.EndDatetime {
					matches := 0
					for _, chapter := range catalog.Options {
						if chapter.Kind == "chapter" && chapter.StartDatetime == option.StartDatetime && chapter.EndDatetime == option.EndDatetime {
							matches++
						}
					}
					if matches == 1 {
						addMember(option.ActivityMember)
					}
				}
			}
		} else {
			banner := config.Banners[int32(source.ID)]
			for i, row := range banners {
				domain, _ := integerAt(row, 2)
				asset, _ := stringAt(row, 4)
				if domain == int64(model.MomBannerDomainGacha) && asset == banner.BannerAssetName {
					add(rowRef{"m_mom_banner", i})
				}
			}
			for i, row := range medals {
				id, _ := integerAt(row, 0)
				if id != int64(banner.GachaMedalId) || id == 0 {
					continue
				}
				add(rowRef{"m_gacha_medal", i})
				currencyID, _ := integerAt(row, 2)
				addCurrency(currencyID)
				for _, shop := range index.shopsByCurrency[currencyID] {
					add(shop)
				}
			}
		}
		groups.Units = append(groups.Units, unit)
		groups.Groups = append(groups.Groups, activitygroup.ActivityGroup{ID: unit.ID, Name: unit.Name, UnitIDs: []string{unit.ID}})
	}
	if err := ValidateActivityGroups(groups, catalog); err != nil {
		return nil, err
	}
	return groups, nil
}

type ActivityScheduleChange struct {
	activitygroup.ActivityMember
	Field  string `json:"field"`
	Before int64  `json:"before"`
	After  int64  `json:"after"`
}

// BuildActivitySchedule visits configured members and binds Premium shard terms
// to same-ID conversion deadlines. Shared references are updated only once.
func BuildActivitySchedule(path string, groups *activitygroup.Config, config *gacha.Config, catalog *ActivityGroupCatalog, groupID string, start, end int64) ([]byte, *gacha.Config, []ActivityScheduleChange, error) {
	if err := ValidateActivityGroups(groups, catalog); err != nil {
		return nil, nil, nil, err
	}
	if start <= 0 || start >= end || end > maxDatetimeMillis-claimRedemptionGraceMillis {
		return nil, nil, nil, fmt.Errorf("活动时间必须为有效的起止时间，且结束后 48 小时不能超出日期范围")
	}
	var group *activitygroup.ActivityGroup
	for i := range groups.Groups {
		if groups.Groups[i].ID == groupID {
			group = &groups.Groups[i]
			break
		}
	}
	if group == nil {
		return nil, nil, nil, fmt.Errorf("活动组不存在")
	}
	selected := make(map[string]bool)
	for _, id := range group.UnitIDs {
		selected[id] = true
	}
	options := make(map[string]ActivityMemberOption)
	kinds := make(map[string]ActivityMemberKind)
	for _, option := range catalog.Options {
		options[activityKey(option.ActivityMember)] = option
	}
	for _, kind := range activityMemberKinds {
		kinds[kind.Kind] = kind
	}
	updated := *config
	updated.Banners = make(map[int32]gacha.BannerConfig, len(config.Banners))
	for id, banner := range config.Banners {
		updated.Banners[id] = banner
	}
	if config.EventSchedules != nil {
		updated.EventSchedules = make(map[int32]gacha.EventSchedule, len(config.EventSchedules))
		for id, schedule := range config.EventSchedules {
			updated.EventSchedules[id] = schedule
		}
	}
	seen := make(map[string]bool)
	changes := []Change{}
	preview := []ActivityScheduleChange{}
	for _, unit := range groups.Units {
		if !selected[unit.ID] {
			continue
		}
		members := append([]activitygroup.ActivityMember(nil), unit.Members...)
		if unit.Type == activitygroup.TypePremium {
			for _, member := range unit.Members {
				paired := activitygroup.ActivityMember{ID: member.ID}
				switch member.Kind {
				case "term":
					paired.Kind = "medal"
				case "medal":
					paired.Kind = "term"
				default:
					continue
				}
				if _, exists := options[activityKey(paired)]; exists {
					members = append(members, paired)
				}
			}
		}
		for _, member := range members {
			key := activityKey(member)
			if seen[key] {
				continue
			}
			seen[key] = true
			option, kind := options[key], kinds[member.Kind]
			targetEnd := end
			if kind.Redemption {
				targetEnd += claimRedemptionGraceMillis
			}
			add := func(field string, before, after int64) {
				if before == after {
					return
				}
				preview = append(preview, ActivityScheduleChange{ActivityMember: member, Field: field, Before: before, After: after})
				if kind.Table != "" {
					changes = append(changes, Change{Table: kind.Table, Row: option.row, Field: field, Value: after})
				}
			}
			if member.Kind == "medal" {
				add("AutoConvertDatetime", option.EndDatetime, targetEnd)
			} else {
				add("StartDatetime", option.StartDatetime, start)
				add("EndDatetime", option.EndDatetime, targetEnd)
			}
			if member.Kind == "premium" {
				banner := updated.Banners[int32(member.ID)]
				banner.StartDatetime = start
				banner.EndDatetime = targetEnd
				updated.Banners[int32(member.ID)] = banner
			}
			if member.Kind == "event" {
				if updated.EventSchedules == nil {
					updated.EventSchedules = make(map[int32]gacha.EventSchedule)
				}
				updated.EventSchedules[int32(member.ID)] = gacha.EventSchedule{StartDatetime: start, EndDatetime: targetEnd}
			}
		}
	}
	var candidate []byte
	if len(changes) > 0 {
		var err error
		candidate, _, err = BuildUpdate(path, UpdateRequest{ExpectedVersion: catalog.Version, Changes: changes})
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return candidate, &updated, preview, nil
}
