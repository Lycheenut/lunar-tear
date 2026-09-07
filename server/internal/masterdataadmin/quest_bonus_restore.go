package masterdataadmin

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"lunar-tear/server/internal/masterdata/memorydb"
)

const (
	bonusCostumes = "m_quest_bonus_costume_setting_group"
	bonusWeapons  = "m_quest_bonus_weapon_group"
	bonusTerms    = "m_quest_bonus_term_group"
	bonusEffects  = "m_quest_bonus_effect_group"
	bonusDrops    = "m_quest_bonus_drop_reward"
)

type QuestBonusWeaponInput struct {
	WeaponID         int64 `json:"weaponId"`
	TemplateWeaponID int64 `json:"templateWeaponId"`
}
type QuestBonusRuleInput struct {
	QuestIDs    []int64 `json:"questIds"`
	RuleBonusID int64   `json:"ruleBonusId"`
}
type QuestBonusCurrencyInput struct {
	FromID int64 `json:"fromId"`
	ToID   int64 `json:"toId"`
}
type QuestBonusRestoreInput struct {
	ChapterID     int64                     `json:"chapterId"`
	SourceBonusID int64                     `json:"sourceBonusId"`
	Mode          string                    `json:"mode,omitempty"`
	CostumeIDs    []int64                   `json:"costumeIds"`
	RuleChapterID int64                     `json:"ruleChapterId"`
	Weapons       []QuestBonusWeaponInput   `json:"weapons"`
	Groups        []QuestBonusRuleInput     `json:"groups,omitempty"`
	Currencies    []QuestBonusCurrencyInput `json:"currencies,omitempty"`
}
type QuestBonusRewardPreview struct {
	PossessionType int64 `json:"possessionType"`
	PossessionID   int64 `json:"possessionId"`
	Count          int64 `json:"count"`
}
type QuestBonusWeaponTierPreview struct {
	WeaponID   int64                     `json:"weaponId"`
	LimitBreak int64                     `json:"limitBreak"`
	Rewards    []QuestBonusRewardPreview `json:"rewards"`
}
type QuestBonusPhasePreview struct {
	BeforeBonusID int64                         `json:"beforeBonusId"`
	AfterBonusID  int64                         `json:"afterBonusId"`
	RuleBonusID   int64                         `json:"ruleBonusId"`
	QuestIDs      []int64                       `json:"questIds"`
	Weapons       []QuestBonusWeaponTierPreview `json:"weapons"`
}
type QuestBonusRestorePreview struct {
	ChapterID     int64                    `json:"chapterId"`
	SourceBonusID int64                    `json:"sourceBonusId"`
	Mode          string                   `json:"mode,omitempty"`
	ScheduleOnly  bool                     `json:"scheduleOnly"`
	StartDatetime int64                    `json:"startDatetime"`
	EndDatetime   int64                    `json:"endDatetime"`
	TermGroupID   int64                    `json:"termGroupId"`
	CostumeIDs    []int64                  `json:"costumeIds"`
	Groups        []QuestBonusPhasePreview `json:"groups"`
}

// Medals are identified by the consumable type, never by translated names or
// numeric ID offsets. The pickup chain supplies each quest's available medals.
func questBonusMedals(file *memorydb.File) ([][]interface{}, map[int64][]int64) {
	var medals [][]interface{}
	ids := make(map[int64]bool)
	for _, row := range readRows(file, "m_consumable_item") {
		if bonusInt(row, 1) == 110 {
			medals = append(medals, row)
			ids[bonusInt(row, 0)] = true
		}
	}
	rewards := make(map[int64]int64)
	for _, row := range readRows(file, "m_battle_drop_reward") {
		if bonusInt(row, 1) == 6 && ids[bonusInt(row, 2)] {
			rewards[bonusInt(row, 0)] = bonusInt(row, 2)
		}
	}
	groups := make(map[int64]map[int64]bool)
	for _, row := range readRows(file, "m_quest_pickup_reward_group") {
		if id := rewards[bonusInt(row, 2)]; id != 0 {
			gid := bonusInt(row, 0)
			if groups[gid] == nil {
				groups[gid] = make(map[int64]bool)
			}
			groups[gid][id] = true
		}
	}
	result := make(map[int64][]int64)
	// QuestPickupRewardGroupId is column 8 in the serialized quest schema.
	for _, row := range readRows(file, questTable) {
		result[bonusInt(row, 0)] = sortedBonusIDs(groups[bonusInt(row, 8)])
	}
	return medals, result
}

func sortedBonusIDs(set map[int64]bool) []int64 {
	result := make([]int64, 0, len(set))
	for id := range set {
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
func bonusNumber(s string) int64 { n, _ := strconv.ParseInt(s, 10, 64); return n }
func bonusString(n int64) string { return strconv.FormatInt(n, 10) }

type bonusRestorePlanner struct {
	file         *memorydb.File
	groups       map[string]map[int64][][]string
	next         map[string]int64
	dirty        map[string]QuestBonusGroupInput
	fingerprints map[string]map[string]int64
	family       map[int64]int64
	order        map[int64]int64
	quests       []QuestBonusQuest
	questBonuses map[int64]int64
	medals       map[int64][]int64
}

func newBonusRestorePlanner(file *memorydb.File) *bonusRestorePlanner {
	p := &bonusRestorePlanner{file: file, groups: make(map[string]map[int64][][]string), next: make(map[string]int64), dirty: make(map[string]QuestBonusGroupInput), fingerprints: make(map[string]map[string]int64), family: make(map[int64]int64), order: make(map[int64]int64), quests: questBonusQuests(file)}
	for _, spec := range questBonusTableSpecs {
		p.groups[spec.Name] = make(map[int64][][]string)
		for _, row := range readRows(file, spec.Name) {
			id := bonusInt(row, 0)
			p.groups[spec.Name][id] = append(p.groups[spec.Name][id], bonusStringRows([][]interface{}{row})[0])
			if id > p.next[spec.Name] {
				p.next[spec.Name] = id
			}
		}
	}
	// Also reserve dangling IDs so allocation cannot repair unrelated history by accident.
	graph, _ := bonusReferenceGraph(file)
	for _, children := range graph {
		for _, key := range children {
			table, value, _ := strings.Cut(key, ":")
			if id := bonusNumber(value); id > p.next[table] {
				p.next[table] = id
			}
		}
	}
	// Legacy costume links are not in the validated graph, but still reserve
	// their historical IDs when cloning the legacy groups that do exist.
	for _, row := range readRows(file, questBonusTable) {
		if id := bonusInt(row, 2); id > p.next["m_quest_bonus_costume_group"] {
			p.next["m_quest_bonus_costume_group"] = id
		}
	}
	for _, row := range readRows(file, "m_weapon") {
		p.family[bonusInt(row, 0)] = bonusInt(row, 0)
	}
	for _, row := range readRows(file, "m_weapon_evolution_group") {
		p.family[bonusInt(row, 2)] = bonusInt(row, 0)
		p.order[bonusInt(row, 2)] = bonusInt(row, 1)
	}
	_, p.medals = questBonusMedals(file)
	p.questBonuses = make(map[int64]int64)
	for _, row := range readRows(file, questTable) {
		p.questBonuses[bonusInt(row, 0)] = bonusInt(row, 19)
	}
	return p
}

func bonusBody(rows [][]string) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		encoded, _ := json.Marshal(row[1:])
		parts = append(parts, string(encoded))
	}
	sort.Strings(parts)
	return strings.Join(parts, "\n")
}
func (p *bonusRestorePlanner) put(table string, id int64, rows [][]string) {
	if p.groups[table][id] != nil {
		delete(p.fingerprints, table)
	}
	copied := make([][]string, len(rows))
	for i, row := range rows {
		copied[i] = append([]string(nil), row...)
		copied[i][0] = bonusString(id)
	}
	p.groups[table][id] = copied
	if index := p.fingerprints[table]; index != nil {
		body := bonusBody(copied)
		if old := index[body]; old == 0 || id < old {
			index[body] = id
		}
	}
	p.dirty[fmt.Sprintf("%s:%d", table, id)] = QuestBonusGroupInput{Table: table, GroupID: id, Rows: copied}
}
func (p *bonusRestorePlanner) intern(table string, rows [][]string) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	if p.fingerprints[table] == nil {
		index := make(map[string]int64)
		for id, existing := range p.groups[table] {
			body := bonusBody(existing)
			if old := index[body]; old == 0 || id < old {
				index[body] = id
			}
		}
		p.fingerprints[table] = index
	}
	match := p.fingerprints[table][bonusBody(rows)]
	if match != 0 {
		return match, nil
	}
	p.next[table]++
	if p.next[table] > 2147483647 {
		return 0, fmt.Errorf("%s 没有可分配的 ID", table)
	}
	p.put(table, p.next[table], rows)
	return p.next[table], nil
}

var timedBonusLinks = []struct {
	column int
	table  string
	term   int
}{
	{1, "m_quest_bonus_character_group", 3}, {2, "m_quest_bonus_costume_group", 3}, {3, bonusWeapons, 4}, {4, bonusCostumes, 4}, {5, "m_quest_bonus_ally_character", 2},
}

func (p *bonusRestorePlanner) terms(bonus int64) map[int64]bool {
	set := make(map[int64]bool)
	rows := p.groups[questBonusTable][bonus]
	if len(rows) == 0 {
		return set
	}
	for _, link := range timedBonusLinks {
		for _, row := range p.groups[link.table][bonusNumber(rows[0][link.column])] {
			set[bonusNumber(row[link.term])] = true
		}
	}
	return set
}
func (p *bonusRestorePlanner) activityTerm(quests map[int64]QuestBonusQuest, start, end int64) (int64, error) {
	set := make(map[int64]bool)
	for _, q := range quests {
		for id := range p.terms(q.BonusID) {
			set[id] = true
		}
	}
	if len(set) == 1 && !set[0] {
		id := sortedBonusIDs(set)[0]
		private := true
		for qid, bonusID := range p.questBonuses {
			if _, own := quests[qid]; !own && p.terms(bonusID)[id] {
				private = false
				break
			}
		}
		if private && len(p.groups[bonusTerms][id]) == 1 {
			rows := [][]string{{bonusString(id), "1", bonusString(start), bonusString(end)}}
			if bonusBody(rows) != bonusBody(p.groups[bonusTerms][id]) {
				p.put(bonusTerms, id, rows)
			}
			return id, nil
		}
	}
	// A fresh term must remain activity-specific even when other dates are identical.
	p.next[bonusTerms]++
	if p.next[bonusTerms] > 2147483647 {
		return 0, fmt.Errorf("共鸣期限 ID 已用尽")
	}
	id := p.next[bonusTerms]
	p.put(bonusTerms, id, [][]string{{"0", "1", bonusString(start), bonusString(end)}})
	return id, nil
}
func (p *bonusRestorePlanner) timedGroup(table string, id, term int64, column int, legacy bool) (int64, error) {
	if id == 0 {
		return 0, nil
	}
	rows := p.groups[table][id]
	if len(rows) == 0 {
		if legacy {
			return id, nil
		}
		return 0, fmt.Errorf("%s:%d 缺少成员配置", table, id)
	}
	copyRows := make([][]string, len(rows))
	for i, row := range rows {
		copyRows[i] = append([]string(nil), row...)
		copyRows[i][column] = bonusString(term)
	}
	return p.intern(table, copyRows)
}

func (p *bonusRestorePlanner) mappedEffect(id int64, mappings map[int64]int64, available map[int64]bool, external bool) (int64, error) {
	rows := p.groups[bonusEffects][id]
	if len(rows) == 0 {
		return 0, fmt.Errorf("效果组 %d 不存在", id)
	}
	var result [][]string
	for _, row := range rows {
		if bonusNumber(row[2]) != 3 {
			return 0, fmt.Errorf("武器规则 %d 包含非掉落效果，请选择其他规则", id)
		}
		drops := p.groups[bonusDrops][bonusNumber(row[3])]
		if len(drops) != 1 {
			return 0, fmt.Errorf("掉落效果 %s 不存在", row[3])
		}
		drop := append([]string(nil), drops[0]...)
		old := bonusNumber(drop[2])
		target := old
		if mapped, ok := mappings[old]; ok {
			target = mapped
		}
		if bonusNumber(drop[1]) != 6 || (external && !available[target]) {
			return 0, fmt.Errorf("奖章 %d 未映射到这组关卡的实际奖章", old)
		}
		drop[2] = bonusString(target)
		did, err := p.intern(bonusDrops, [][]string{drop})
		if err != nil {
			return 0, err
		}
		copyRow := append([]string(nil), row...)
		copyRow[3] = bonusString(did)
		result = append(result, copyRow)
	}
	return p.intern(bonusEffects, result)
}

func (p *bonusRestorePlanner) costumes(source, current, term int64, selected map[int64]bool) (int64, error) {
	var result [][]string
	for index, gid := range []int64{current, source} {
		rows := p.groups[bonusCostumes][gid]
		if gid != 0 && len(rows) == 0 {
			return 0, fmt.Errorf("服装组 %d 不存在", gid)
		}
		for _, row := range rows {
			if index == 1 && !selected[bonusNumber(row[1])] {
				continue
			}
			copyRow := append([]string(nil), row...)
			copyRow[4] = bonusString(term)
			result = append(result, copyRow)
		}
	}
	return p.intern(bonusCostumes, result)
}

func (p *bonusRestorePlanner) weapons(source, rule, current, term int64, choices map[int64]int64, currencies map[int64]int64, available map[int64]bool, external bool) (int64, error) {
	sourceRows := p.groups[bonusWeapons][source]
	if source != 0 && len(sourceRows) == 0 {
		return 0, fmt.Errorf("来源武器组 %d 不存在", source)
	}
	forms := make(map[int64]bool)
	for _, row := range sourceRows {
		id := bonusNumber(row[1])
		if choices[p.family[id]] != 0 {
			forms[id] = true
		}
	}
	var result [][]string
	if current != 0 && len(p.groups[bonusWeapons][current]) == 0 {
		return 0, fmt.Errorf("当前武器组 %d 不存在", current)
	}
	for _, row := range p.groups[bonusWeapons][current] {
		copyRow := append([]string(nil), row...)
		copyRow[4] = bonusString(term)
		result = append(result, copyRow)
	}
	for _, wid := range sortedBonusIDs(forms) {
		family := p.family[wid]
		template := choices[family]
		if family == 0 || template == 0 {
			return 0, fmt.Errorf("武器 %d 尚未选择加成规则", wid)
		}
		order := int64(-1)
		for _, row := range p.groups[bonusWeapons][rule] {
			id := bonusNumber(row[1])
			if p.family[id] == p.family[template] && p.order[id] <= p.order[wid] && p.order[id] > order {
				order = p.order[id]
			}
		}
		if order < 0 {
			return 0, fmt.Errorf("所选规则缺少适用于武器 %d 的进化形态", wid)
		}
		seen := make(map[string]bool)
		for _, row := range p.groups[bonusWeapons][rule] {
			id := bonusNumber(row[1])
			if p.family[id] != p.family[template] || p.order[id] != order {
				continue
			}
			effect, err := p.mappedEffect(bonusNumber(row[3]), currencies, available, external)
			if err != nil {
				return 0, err
			}
			if seen[row[2]] {
				return 0, fmt.Errorf("所选规则的进化形态不能唯一匹配武器 %d", wid)
			}
			seen[row[2]] = true
			result = append(result, []string{"0", bonusString(wid), row[2], bonusString(effect), bonusString(term)})
		}
	}
	return p.intern(bonusWeapons, result)
}

func (p *bonusRestorePlanner) weaponPreview(gid int64) []QuestBonusWeaponTierPreview {
	var result []QuestBonusWeaponTierPreview
	for _, row := range p.groups[bonusWeapons][gid] {
		tier := QuestBonusWeaponTierPreview{WeaponID: bonusNumber(row[1]), LimitBreak: bonusNumber(row[2])}
		for _, effect := range p.groups[bonusEffects][bonusNumber(row[3])] {
			if bonusNumber(effect[2]) == 3 {
				for _, drop := range p.groups[bonusDrops][bonusNumber(effect[3])] {
					tier.Rewards = append(tier.Rewards, QuestBonusRewardPreview{bonusNumber(drop[1]), bonusNumber(drop[2]), bonusNumber(drop[3])})
				}
			}
		}
		result = append(result, tier)
	}
	return result
}

// Both preview and apply expand the same semantic request against its locked
// snapshot. Date edits also pass here, so synchronization needs no sidecar state.
func planQuestBonusUpdates(file *memorydb.File, request UpdateRequest) (UpdateRequest, []QuestBonusRestorePreview, error) {
	inputs := make(map[int64]*QuestBonusRestoreInput)
	for i := range request.QuestBonusRestores {
		input := &request.QuestBonusRestores[i]
		if input.ChapterID <= 0 || inputs[input.ChapterID] != nil {
			return request, nil, fmt.Errorf("活动还原请求重复或无效")
		}
		inputs[input.ChapterID] = input
	}
	chapters := readRows(file, "m_event_quest_chapter")
	dates := make(map[int64][2]int64)
	for _, change := range request.Changes {
		if change.Table != "m_event_quest_chapter" || (change.Field != "StartDatetime" && change.Field != "EndDatetime") {
			continue
		}
		if change.Row < 0 || change.Row >= len(chapters) {
			return request, nil, fmt.Errorf("活动行索引无效")
		}
		row := chapters[change.Row]
		id := bonusInt(row, 0)
		pair, ok := dates[id]
		if !ok {
			pair = [2]int64{bonusInt(row, 8), bonusInt(row, 9)}
		}
		spec, _ := findActivitySpec(change.Table)
		field, _ := findField(spec, change.Field)
		value, err := parseChangeValue(field, change.Value)
		if err != nil {
			return request, nil, err
		}
		n, _ := valueAsInt64(value)
		column := 0
		if change.Field == "EndDatetime" {
			column = 1
		}
		pair[column] = n
		dates[id] = pair
		if n != bonusInt(row, 8+column) {
			if _, exists := inputs[id]; !exists {
				inputs[id] = nil
			}
		}
	}
	if len(inputs) == 0 {
		return request, nil, nil
	}
	if len(inputs) > 100 {
		return request, nil, fmt.Errorf("一次最多处理 100 个活动")
	}
	if len(request.QuestBonusGroups) > 0 {
		return request, nil, fmt.Errorf("活动还原或时间联动不能与加成组原始编辑同时提交")
	}
	p := newBonusRestorePlanner(file)
	active := make(map[int64]bool)
	for id := range inputs {
		active[id] = true
	}
	var previews []QuestBonusRestorePreview
	assigned := make(map[int]int64)
	for _, chapterID := range sortedBonusIDs(active) {
		input := inputs[chapterID]
		var chapter []interface{}
		for _, row := range chapters {
			if bonusInt(row, 0) == chapterID {
				chapter = row
				break
			}
		}
		if chapter == nil {
			return request, nil, fmt.Errorf("活动 %d 不存在", chapterID)
		}
		quests := make(map[int64]QuestBonusQuest)
		for _, q := range p.quests {
			if q.ChapterID == chapterID {
				quests[q.QuestID] = q
			}
		}
		if len(quests) == 0 {
			if input != nil {
				return request, nil, fmt.Errorf("活动 %d 没有可还原关卡", chapterID)
			}
			continue
		}
		for _, q := range p.quests {
			if _, shared := quests[q.QuestID]; shared && q.ChapterID != chapterID {
				return request, nil, fmt.Errorf("关卡 %d 与活动 %d 共用，无法设置独立活动期限；请先分离关卡引用", q.QuestID, q.ChapterID)
			}
		}
		pair, ok := dates[chapterID]
		if !ok {
			pair = [2]int64{bonusInt(chapter, 8), bonusInt(chapter, 9)}
		}
		if pair[0] < 0 || pair[1] < 0 || pair[0] > maxDatetimeMillis || pair[1] > maxDatetimeMillis || (pair[1] != 0 && pair[0] > pair[1]) {
			return request, nil, fmt.Errorf("活动 %d 起止时间无效", chapterID)
		}
		var source []string
		ruleChapter := chapterID
		choices := make(map[int64]int64)
		selectedCostumes := make(map[int64]bool)
		currencies := make(map[int64]int64)
		ruleByQuest := make(map[int64]int64)
		if input != nil {
			if input.Mode != "" && input.Mode != "replace" && input.Mode != "append" {
				return request, nil, fmt.Errorf("无效的名单编辑模式")
			}
			if input.Mode != "" && input.CostumeIDs == nil {
				return request, nil, fmt.Errorf("请明确选择来源服装，空名单使用空数组")
			}
			rows := p.groups[questBonusTable][input.SourceBonusID]
			if len(rows) != 1 {
				return request, nil, fmt.Errorf("来源加成 %d 不存在", input.SourceBonusID)
			}
			source = rows[0]
			currentCostumes, currentFamilies := make(map[int64]bool), make(map[int64]bool)
			if input.Mode == "append" {
				for _, q := range quests {
					if rows := p.groups[questBonusTable][q.BonusID]; len(rows) == 1 {
						for _, row := range p.groups[bonusCostumes][bonusNumber(rows[0][4])] {
							currentCostumes[bonusNumber(row[1])] = true
						}
						for _, row := range p.groups[bonusWeapons][bonusNumber(rows[0][3])] {
							currentFamilies[p.family[bonusNumber(row[1])]] = true
						}
					}
				}
			}
			sourceCostumes := make(map[int64]bool)
			for _, row := range p.groups[bonusCostumes][bonusNumber(source[4])] {
				sourceCostumes[bonusNumber(row[1])] = true
			}
			if input.Mode == "" {
				selectedCostumes = sourceCostumes
			} else {
				for _, id := range input.CostumeIDs {
					if !sourceCostumes[id] || selectedCostumes[id] {
						return request, nil, fmt.Errorf("来源服装选择无效/重复：%d", id)
					}
					selectedCostumes[id] = true
				}
				for id := range currentCostumes {
					delete(selectedCostumes, id)
				}
			}
			if input.RuleChapterID != 0 {
				ruleChapter = input.RuleChapterID
			}
			sourceFamilies := make(map[int64]bool)
			for _, row := range p.groups[bonusWeapons][bonusNumber(source[3])] {
				sourceFamilies[p.family[bonusNumber(row[1])]] = true
			}
			for _, choice := range input.Weapons {
				family := p.family[choice.WeaponID]
				if family == 0 || !sourceFamilies[family] || choices[family] != 0 || p.family[choice.TemplateWeaponID] == 0 {
					return request, nil, fmt.Errorf("来源武器或规则选择无效/重复：%d", choice.WeaponID)
				}
				choices[family] = choice.TemplateWeaponID
			}
			if input.Mode == "" {
				for family := range sourceFamilies {
					if choices[family] == 0 {
						return request, nil, fmt.Errorf("武器组 %d 尚未选择规则", family)
					}
				}
			}
			for family := range currentFamilies {
				delete(choices, family)
			}
			// An empty supplement must not allocate groups or synchronize dates
			// unless the same request actually changes the activity schedule.
			if input.Mode == "append" && len(choices) == 0 && len(selectedCostumes) == 0 {
				if pair == [2]int64{bonusInt(chapter, 8), bonusInt(chapter, 9)} {
					continue
				}
				input = nil
			}
		}
		if input != nil && len(choices) > 0 {
			for _, mapping := range input.Currencies {
				if mapping.FromID <= 0 || mapping.ToID <= 0 || currencies[mapping.FromID] != 0 {
					return request, nil, fmt.Errorf("奖章映射无效或重复")
				}
				currencies[mapping.FromID] = mapping.ToID
			}
			validRules := make(map[int64]bool)
			for _, q := range p.quests {
				if q.ChapterID == ruleChapter && q.BonusID != 0 {
					validRules[q.BonusID] = true
				}
			}
			for _, group := range input.Groups {
				if !validRules[group.RuleBonusID] {
					return request, nil, fmt.Errorf("加成 %d 不属于所选规则参考活动", group.RuleBonusID)
				}
				for _, qid := range group.QuestIDs {
					if _, ok := quests[qid]; !ok || ruleByQuest[qid] != 0 {
						return request, nil, fmt.Errorf("关卡分组重复或包含其他活动")
					}
					ruleByQuest[qid] = group.RuleBonusID
				}
			}
			if ruleChapter != chapterID && len(ruleByQuest) != len(quests) {
				return request, nil, fmt.Errorf("请为全部目标关卡指定参考规则")
			}
			for qid, q := range quests {
				if ruleByQuest[qid] == 0 {
					if !validRules[q.BonusID] {
						return request, nil, fmt.Errorf("活动缺少现有加成，请选择规则参考活动并映射关卡及奖章")
					}
					ruleByQuest[qid] = q.BonusID
				}
				if ruleChapter == chapterID && ruleByQuest[qid] != q.BonusID {
					return request, nil, fmt.Errorf("使用本活动规则时必须保留原关卡分组")
				}
			}
		}
		hasBonus := false
		for _, q := range quests {
			if q.BonusID != 0 {
				hasBonus = true
			}
		}
		if input == nil && !hasBonus {
			continue
		}
		term, err := p.activityTerm(quests, pair[0], pair[1])
		if err != nil {
			return request, nil, err
		}
		preview := QuestBonusRestorePreview{ChapterID: chapterID, ScheduleOnly: input == nil, StartDatetime: pair[0], EndDatetime: pair[1], TermGroupID: term}
		if input != nil {
			preview.SourceBonusID = input.SourceBonusID
			preview.Mode = input.Mode
		}
		phaseQuests := make(map[string][]int64)
		for qid, q := range quests {
			if input == nil && q.BonusID == 0 {
				continue
			}
			// External references may split quests that originally all used bonus zero.
			key := fmt.Sprintf("%d:%d", q.BonusID, ruleByQuest[qid])
			phaseQuests[key] = append(phaseQuests[key], qid)
		}
		phaseKeys := make([]string, 0, len(phaseQuests))
		for key := range phaseQuests {
			phaseKeys = append(phaseKeys, key)
		}
		sort.Strings(phaseKeys)
		costumes := make(map[int64]bool)
		for _, key := range phaseKeys {
			qids := phaseQuests[key]
			sort.Slice(qids, func(i, j int) bool { return qids[i] < qids[j] })
			oldID := quests[qids[0]].BonusID
			ruleID := ruleByQuest[qids[0]]
			base := []string{"0", "0", "0", "0", "0", "0"}
			if rows := p.groups[questBonusTable][oldID]; len(rows) == 1 {
				base = append([]string(nil), rows[0]...)
			} else if oldID != 0 {
				return request, nil, fmt.Errorf("目标加成 %d 不存在", oldID)
			}
			for _, link := range timedBonusLinks {
				var gid int64
				current := int64(0)
				if input != nil && input.Mode == "append" {
					current = bonusNumber(base[link.column])
				}
				if link.column == 4 && input != nil {
					gid, err = p.costumes(bonusNumber(source[4]), current, term, selectedCostumes)
				} else if link.column == 3 && input != nil {
					ruleWeapons := int64(0)
					if len(choices) > 0 {
						rules := p.groups[questBonusTable][ruleID]
						if len(rules) != 1 {
							return request, nil, fmt.Errorf("参考加成 %d 不存在", ruleID)
						}
						ruleWeapons = bonusNumber(rules[0][3])
					}
					available := make(map[int64]bool)
					for _, id := range p.medals[qids[0]] {
						available[id] = true
					}
					for _, qid := range qids[1:] {
						here := make(map[int64]bool)
						for _, id := range p.medals[qid] {
							here[id] = true
						}
						for id := range available {
							if !here[id] {
								delete(available, id)
							}
						}
					}
					gid, err = p.weapons(bonusNumber(source[3]), ruleWeapons, current, term, choices, currencies, available, ruleChapter != chapterID)
				} else {
					gid, err = p.timedGroup(link.table, bonusNumber(base[link.column]), term, link.term, link.column == 2)
				}
				if err != nil {
					return request, nil, fmt.Errorf("活动 %d：%w", chapterID, err)
				}
				base[link.column] = bonusString(gid)
			}
			bid, err := p.intern(questBonusTable, [][]string{base})
			if err != nil {
				return request, nil, err
			}
			for _, qid := range qids {
				if bid != quests[qid].BonusID {
					assigned[quests[qid].Row] = bid
				}
			}
			for _, row := range p.groups[bonusCostumes][bonusNumber(base[4])] {
				costumes[bonusNumber(row[1])] = true
			}
			preview.Groups = append(preview.Groups, QuestBonusPhasePreview{BeforeBonusID: oldID, AfterBonusID: bid, RuleBonusID: ruleID, QuestIDs: qids, Weapons: p.weaponPreview(bonusNumber(base[3]))})
		}
		preview.CostumeIDs = sortedBonusIDs(costumes)
		previews = append(previews, preview)
	}
	rows := make([]int, 0, len(assigned))
	for row := range assigned {
		rows = append(rows, row)
	}
	sort.Ints(rows)
	for _, row := range rows {
		for _, change := range request.Changes {
			if change.Table == questTable && change.Row == row {
				return request, nil, fmt.Errorf("关卡加成还原与直接编辑冲突")
			}
		}
		request.Changes = append(request.Changes, Change{Table: questTable, Row: row, Field: "QuestBonusId", Value: bonusString(assigned[row])})
	}
	keys := make([]string, 0, len(p.dirty))
	for key := range p.dirty {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		request.QuestBonusGroups = append(request.QuestBonusGroups, p.dirty[key])
	}
	request.QuestBonusRestores = nil
	return request, previews, nil
}
