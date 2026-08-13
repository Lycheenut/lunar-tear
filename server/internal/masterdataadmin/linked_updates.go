package masterdataadmin

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
)

const (
	momBannerDomainLoginBonus int64 = 21
	momBannerDomainMission    int64 = 22
	momBannerDomainEvent      int64 = 23
	eventLinkDomainShop       int64 = 3
	missionLinkDomainEvent    int64 = 4
	naviCutInFunctionEvent    int64 = 2
	mamaMedalItemType         int64 = 110
	mamaMedalAssetCategory    int64 = 117
)

type CellChangePreview struct {
	Field     string `json:"field"`
	Before    string `json:"before"`
	After     string `json:"after"`
	Datetime  bool   `json:"datetime"`
	Generated bool   `json:"generated"`
}

type RecordPreview struct {
	Relation string              `json:"relation,omitempty"`
	Note     string              `json:"note,omitempty"`
	Table    string              `json:"table"`
	Row      int                 `json:"row"`
	Identity []FieldValue        `json:"identity"`
	Titles   map[string]string   `json:"titles,omitempty"`
	Changes  []CellChangePreview `json:"changes,omitempty"`
}

type UpdateImpactPreview struct {
	Kind       string          `json:"kind"`
	Upstream   RecordPreview   `json:"upstream"`
	Downstream []RecordPreview `json:"downstream,omitempty"`
}

type UpdatePreview struct {
	Impacts          []UpdateImpactPreview `json:"impacts,omitempty"`
	OtherChanges     []RecordPreview       `json:"otherChanges,omitempty"`
	RequestedChanges int                   `json:"requestedChanges"`
	GeneratedChanges int                   `json:"generatedChanges"`
	TotalChanges     int                   `json:"totalChanges"`
	ChangedRows      int                   `json:"changedRows"`
}

type rowRef struct {
	table string
	row   int
}

type linkedTarget struct {
	ref      rowRef
	relation string
	note     string
}

type linkedImpact struct {
	kind       string
	upstream   rowRef
	targets    []linkedTarget
	targetKeys map[string]bool
}

type relationIndex struct {
	shopsByID                      map[int64]rowRef
	shopsByCurrency                map[int64][]rowRef
	termsByID                      map[int64]rowRef
	termByCurrency                 map[int64]rowRef
	gachaCurrencies                map[int64][]int64
	eventLinks                     map[int64][]interface{}
	missionTermsByChapter          map[int64][]rowRef
	missionTermsByCurrency         map[int64][]rowRef
	monthlyCurrenciesByMissionTerm map[int64][]int64
	eventBannersByText             map[int64][]rowRef
	loginBannersByID               map[int64][]rowRef
	missionBannersByTerm           map[int64][]rowRef
	naviCutInsByChapter            map[int64][]rowRef
	monthlyCurrenciesByLoginBonus  map[int64][]int64
}

type linkedUpdatePlanner struct {
	file      *memorydb.File
	rows      map[string][][]interface{}
	explicit  map[string]Change
	effective map[string]interface{}
	generated map[string]Change
	impacts   []linkedImpact
	index     *relationIndex
}

func PreviewUpdate(path string, request UpdateRequest) (UpdatePreview, error) {
	if err := validateUpdateEnvelope(request); err != nil {
		return UpdatePreview{}, err
	}
	file, err := memorydb.OpenFile(path)
	if err != nil {
		return UpdatePreview{}, err
	}
	if file.Version() != request.ExpectedVersion {
		return UpdatePreview{}, ErrVersionConflict
	}
	planned, impacts, generated, err := expandLinkedChanges(file, request.Changes)
	if err != nil {
		return UpdatePreview{}, err
	}
	validated := request
	validated.Changes = planned
	_, result, err := buildUpdate(file, validated)
	if err != nil {
		return UpdatePreview{}, err
	}
	resolver := newTitleResolver(file, loadLocalizationIndex(path))
	catalog, err := catalogFromFile(file, resolver)
	if err != nil {
		return UpdatePreview{}, err
	}
	return assembleUpdatePreview(catalog, request.Changes, planned, impacts, generated, result), nil
}

func validateUpdateEnvelope(request UpdateRequest) error {
	if request.ExpectedVersion == "" {
		return fmt.Errorf("expectedVersion is required")
	}
	if len(request.Changes) == 0 {
		return fmt.Errorf("at least one change is required")
	}
	if len(request.Changes) > 10000 {
		return fmt.Errorf("too many changes")
	}
	return nil
}

func expandLinkedChanges(file *memorydb.File, requested []Change) ([]Change, []linkedImpact, map[string]bool, error) {
	planner := &linkedUpdatePlanner{
		file:      file,
		rows:      make(map[string][][]interface{}),
		explicit:  make(map[string]Change, len(requested)),
		effective: make(map[string]interface{}, len(requested)),
		generated: make(map[string]Change),
	}
	changedRows := make(map[string]rowRef)
	for _, change := range requested {
		spec, ok := findActivitySpec(change.Table)
		if !ok {
			return nil, nil, nil, fmt.Errorf("table %q is not an editable activity table", change.Table)
		}
		field, ok := findField(spec, change.Field)
		if !ok {
			return nil, nil, nil, fmt.Errorf("field %q does not exist on table %q", change.Field, change.Table)
		}
		if field.PrimaryKey {
			return nil, nil, nil, fmt.Errorf("primary key field %q is read-only on table %q", change.Field, change.Table)
		}
		rows, err := planner.tableRows(change.Table)
		if err != nil {
			return nil, nil, nil, err
		}
		if change.Row < 0 || change.Row >= len(rows) {
			return nil, nil, nil, fmt.Errorf("row %d is outside table %q", change.Row, change.Table)
		}
		value, err := parseChangeValue(field, change.Value)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s row %d field %s: %w", change.Table, change.Row, change.Field, err)
		}
		key := previewCellKey(change.Table, change.Row, change.Field)
		if _, duplicate := planner.explicit[key]; duplicate {
			return nil, nil, nil, fmt.Errorf("duplicate change for %s row %d field %s", change.Table, change.Row, change.Field)
		}
		planner.explicit[key] = change
		planner.effective[key] = value
		ref := rowRef{table: change.Table, row: change.Row}
		changedRows[previewRecordKey(ref)] = ref
	}

	refs := make([]rowRef, 0, len(changedRows))
	for _, ref := range changedRows {
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].table != refs[j].table {
			return refs[i].table < refs[j].table
		}
		return refs[i].row < refs[j].row
	})
	for _, ref := range refs {
		switch ref.table {
		case "m_mom_banner":
			if err := planner.planGachaMomBanner(ref); err != nil {
				return nil, nil, nil, err
			}
		case "m_event_quest_chapter":
			if err := planner.planEventQuestChapter(ref); err != nil {
				return nil, nil, nil, err
			}
		case "m_login_bonus":
			if err := planner.planLoginBonus(ref); err != nil {
				return nil, nil, nil, err
			}
		}
	}

	generatedKeys := make([]string, 0, len(planner.generated))
	for key := range planner.generated {
		generatedKeys = append(generatedKeys, key)
	}
	sort.Strings(generatedKeys)
	planned := append([]Change(nil), requested...)
	generatedSet := make(map[string]bool, len(generatedKeys))
	for _, key := range generatedKeys {
		planned = append(planned, planner.generated[key])
		generatedSet[key] = true
	}
	if len(planned) > 10000 {
		return nil, nil, nil, fmt.Errorf("linked update expands to too many changes")
	}
	return planned, planner.impacts, generatedSet, nil
}

func (p *linkedUpdatePlanner) planGachaMomBanner(ref rowRef) error {
	rows, err := p.tableRows(ref.table)
	if err != nil {
		return err
	}
	row := rows[ref.row]
	oldDomain, _ := integerAt(row, 2)
	oldGachaID, _ := integerAt(row, 3)
	newDomain, err := p.effectiveInt(ref, "DestinationDomainType")
	if err != nil {
		return err
	}
	newGachaID, err := p.effectiveInt(ref, "DestinationDomainId")
	if err != nil {
		return err
	}
	if oldDomain != int64(model.MomBannerDomainGacha) && newDomain != int64(model.MomBannerDomainGacha) {
		return nil
	}
	impact := p.newImpact("Gacha", ref)
	index, err := p.relations()
	if err != nil {
		return err
	}
	stable := oldDomain == int64(model.MomBannerDomainGacha) && newDomain == oldDomain && newGachaID == oldGachaID
	ids := []int64{oldGachaID}
	if newGachaID != oldGachaID {
		ids = append(ids, newGachaID)
	}
	seen := make(map[string]bool)
	for _, gachaID := range ids {
		for _, currencyID := range index.gachaCurrencies[gachaID] {
			for _, shop := range index.shopsByCurrency[currencyID] {
				key := previewRecordKey(shop)
				if seen[key] {
					continue
				}
				seen[key] = true
				overlaps, err := p.rowsOverlap(ref, shop)
				if err != nil {
					return err
				}
				note := ""
				cascade := stable && gachaID == oldGachaID && overlaps
				if !overlaps {
					note = "通过 Gacha 天井币确定关联，但当前档期不重叠，因此不自动调整。"
				}
				if !stable {
					note = "Gacha 关联字段发生变化，本次仅展示关联内容，不自动调整档期。"
				}
				target := p.addTarget(impact, shop, "天井兑换商店", note)
				if cascade {
					if err := p.cascadePair(ref, target); err != nil {
						return err
					}
				}
			}
			if term, ok := index.termByCurrency[currencyID]; ok {
				key := previewRecordKey(term)
				if seen[key] {
					continue
				}
				seen[key] = true
				overlaps, err := p.rowsOverlap(ref, term)
				if err != nil {
					return err
				}
				note := ""
				cascade := stable && gachaID == oldGachaID && overlaps
				if !overlaps {
					note = "通过 Gacha 天井币确定关联，但当前档期不重叠，因此不自动调整。"
				}
				if !stable {
					note = "Gacha 关联字段发生变化，本次仅展示关联内容，不自动调整档期。"
				}
				target := p.addTarget(impact, term, "天井币有效期", note)
				if cascade {
					if err := p.cascadePair(ref, target); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (p *linkedUpdatePlanner) planEventQuestChapter(ref rowRef) error {
	rows, err := p.tableRows(ref.table)
	if err != nil {
		return err
	}
	row := rows[ref.row]
	chapterID, _ := integerAt(row, 0)
	nameTextID, _ := integerAt(row, 3)
	eventLinkID, _ := integerAt(row, 5)
	newNameTextID, err := p.effectiveInt(ref, "NameEventQuestTextId")
	if err != nil {
		return err
	}
	newEventLinkID, err := p.effectiveInt(ref, "EventQuestLinkId")
	if err != nil {
		return err
	}
	impact := p.newImpact("EventQuestChapter", ref)
	index, err := p.relations()
	if err != nil {
		return err
	}

	textIDs := []int64{nameTextID}
	if newNameTextID != nameTextID {
		textIDs = append(textIDs, newNameTextID)
	}
	for _, textID := range textIDs {
		for _, banner := range index.eventBannersByText[textID] {
			overlaps, err := p.rowsOverlap(ref, banner)
			if err != nil {
				return err
			}
			if !overlaps {
				continue
			}
			stable := newNameTextID == nameTextID
			note := ""
			if !stable {
				note = "活动标题文本 ID 已变化，旧、新资源命名关系仅作提示，不自动调整档期。"
			}
			target := p.addTarget(impact, banner, "活动 MomBanner", note)
			if stable && textID == nameTextID {
				if err := p.cascadePair(ref, target); err != nil {
					return err
				}
			}
		}
	}

	eventLinkIDs := []int64{eventLinkID}
	if newEventLinkID != eventLinkID {
		eventLinkIDs = append(eventLinkIDs, newEventLinkID)
	}
	for _, linkID := range eventLinkIDs {
		link := index.eventLinks[linkID]
		if link == nil {
			continue
		}
		domain, _ := integerAt(link, 1)
		destination, _ := integerAt(link, 2)
		possessionType, _ := integerAt(link, 3)
		currencyID, _ := integerAt(link, 4)
		stable := newEventLinkID == eventLinkID
		if domain == eventLinkDomainShop {
			if shop, ok := index.shopsByID[destination]; ok {
				note := ""
				if !stable {
					note = "EventQuestLinkId 已变化，旧、新活动商店仅作提示，不自动调整档期。"
				}
				target := p.addTarget(impact, shop, "活动商店", note)
				if stable && linkID == eventLinkID {
					if err := p.cascadePair(ref, target); err != nil {
						return err
					}
				}
			}
		}
		if possessionType == int64(model.PossessionTypeConsumableItem) {
			if term, ok := index.termByCurrency[currencyID]; ok {
				note := ""
				if !stable {
					note = "EventQuestLinkId 已变化，旧、新活动币有效期仅作提示，不自动调整。"
				}
				target := p.addTarget(impact, term, "活动币有效期", note)
				if stable && linkID == eventLinkID {
					if err := p.cascadePair(ref, target); err != nil {
						return err
					}
				}
			}
		}
	}

	for _, term := range index.missionTermsByChapter[chapterID] {
		target := p.addTarget(impact, term, "限时任务档期", "")
		if err := p.cascadePair(ref, target); err != nil {
			return err
		}
		termRows, err := p.tableRows(term.table)
		if err != nil {
			return err
		}
		termID, _ := integerAt(termRows[term.row], 0)
		for _, banner := range index.missionBannersByTerm[termID] {
			bannerTarget := p.addTarget(impact, banner, "限时任务 MomBanner", "")
			if err := p.cascadePair(ref, bannerTarget); err != nil {
				return err
			}
		}
		for _, currencyID := range index.monthlyCurrenciesByMissionTerm[termID] {
			note := fmt.Sprintf("关联限时任务奖励包含月度妈妈兑换币 %d；该档期由多个奖励来源共享，不自动调整。", currencyID)
			for _, shop := range index.shopsByCurrency[currencyID] {
				p.addTarget(impact, shop, "限时任务关联的月度妈妈兑换商店", note)
			}
			if currencyTerm, ok := index.termByCurrency[currencyID]; ok {
				p.addTarget(impact, currencyTerm, "月度妈妈兑换币有效期", note)
			}
		}
	}
	for _, navi := range index.naviCutInsByChapter[chapterID] {
		target := p.addTarget(impact, navi, "活动 NaviCutIn", "")
		if err := p.cascadePair(ref, target); err != nil {
			return err
		}
	}
	return nil
}

func (p *linkedUpdatePlanner) planLoginBonus(ref rowRef) error {
	rows, err := p.tableRows(ref.table)
	if err != nil {
		return err
	}
	loginBonusID, _ := integerAt(rows[ref.row], 0)
	impact := p.newImpact("LoginBonus", ref)
	index, err := p.relations()
	if err != nil {
		return err
	}
	for _, banner := range index.loginBannersByID[loginBonusID] {
		overlaps, err := p.rowsOverlap(ref, banner)
		if err != nil {
			return err
		}
		note := ""
		if !overlaps {
			note = "通过 LoginBonusId 确定关联，但当前档期不重叠，因此不自动调整。"
		}
		target := p.addTarget(impact, banner, "签到 MomBanner", note)
		if overlaps {
			if err := p.cascadePair(ref, target); err != nil {
				return err
			}
		}
	}
	for _, currencyID := range index.monthlyCurrenciesByLoginBonus[loginBonusID] {
		note := fmt.Sprintf("签到奖励包含月度妈妈兑换币 %d；该档期由多个奖励来源共享，不自动调整。", currencyID)
		for _, shop := range index.shopsByCurrency[currencyID] {
			p.addTarget(impact, shop, "月度妈妈兑换商店", note)
		}
		if term, ok := index.termByCurrency[currencyID]; ok {
			p.addTarget(impact, term, "月度妈妈兑换币有效期", note)
		}
		for _, missionTerm := range index.missionTermsByCurrency[currencyID] {
			p.addTarget(impact, missionTerm, "共享月度兑换币的限时任务", note)
		}
	}
	return nil
}

func (p *linkedUpdatePlanner) newImpact(kind string, upstream rowRef) *linkedImpact {
	p.impacts = append(p.impacts, linkedImpact{
		kind:       kind,
		upstream:   upstream,
		targetKeys: make(map[string]bool),
	})
	return &p.impacts[len(p.impacts)-1]
}

func (p *linkedUpdatePlanner) addTarget(impact *linkedImpact, ref rowRef, relation, note string) linkedTarget {
	key := previewRecordKey(ref) + "\x00" + relation
	if !impact.targetKeys[key] {
		impact.targetKeys[key] = true
		impact.targets = append(impact.targets, linkedTarget{ref: ref, relation: relation, note: note})
	}
	return linkedTarget{ref: ref, relation: relation, note: note}
}

func (p *linkedUpdatePlanner) cascadePair(source rowRef, target linkedTarget) error {
	sourceSpec, _ := findActivitySpec(source.table)
	targetSpec, _ := findActivitySpec(target.ref.table)
	sourcePairs := sourceSpec.pairs()
	targetPairs := targetSpec.pairs()
	if len(sourcePairs) == 0 || len(targetPairs) == 0 {
		return nil
	}
	if err := p.cascadeTime(source, sourcePairs[0].Start, target, targetPairs[0].Start); err != nil {
		return err
	}
	return p.cascadeTime(source, sourcePairs[0].End, target, targetPairs[0].End)
}

func (p *linkedUpdatePlanner) cascadeTime(source rowRef, sourceField string, target linkedTarget, targetField string) error {
	sourceKey := previewCellKey(source.table, source.row, sourceField)
	if _, changed := p.explicit[sourceKey]; !changed {
		return nil
	}
	sourceRows, err := p.tableRows(source.table)
	if err != nil {
		return err
	}
	targetRows, err := p.tableRows(target.ref.table)
	if err != nil {
		return err
	}
	sourceSpec, _ := findActivitySpec(source.table)
	targetSpec, _ := findActivitySpec(target.ref.table)
	sourceColumn, _ := findField(sourceSpec, sourceField)
	targetColumn, _ := findField(targetSpec, targetField)
	oldSource, err := valueAsInt64(sourceRows[source.row][sourceColumn.Index])
	if err != nil {
		return err
	}
	newSource, err := valueAsInt64(p.effective[sourceKey])
	if err != nil {
		return err
	}
	oldTarget, err := valueAsInt64(targetRows[target.ref.row][targetColumn.Index])
	if err != nil {
		return err
	}
	newTarget, ok := cascadedTime(oldSource, newSource, oldTarget)
	if !ok || newTarget == oldTarget {
		return nil
	}
	if newTarget < 0 || newTarget > maxDatetimeMillis {
		return fmt.Errorf("级联更新 %s row %d field %s 超出支持的日期范围", target.ref.table, target.ref.row, targetField)
	}
	key := previewCellKey(target.ref.table, target.ref.row, targetField)
	if _, explicit := p.explicit[key]; explicit {
		return nil
	}
	change := Change{Table: target.ref.table, Row: target.ref.row, Field: targetField, Value: strconv.FormatInt(newTarget, 10)}
	if existing, duplicate := p.generated[key]; duplicate {
		if fmt.Sprint(existing.Value) != fmt.Sprint(change.Value) {
			return fmt.Errorf("多个上游修改会把 %s row %d field %s 设置为不同值，请拆分修改", target.ref.table, target.ref.row, targetField)
		}
		return nil
	}
	p.generated[key] = change
	return nil
}

func cascadedTime(oldSource, newSource, oldTarget int64) (int64, bool) {
	if oldSource == newSource || oldTarget == 0 {
		return oldTarget, false
	}
	if newSource == 0 {
		return 0, true
	}
	if oldSource == 0 {
		return newSource, true
	}
	return oldTarget + (newSource - oldSource), true
}

func (p *linkedUpdatePlanner) effectiveInt(ref rowRef, fieldName string) (int64, error) {
	key := previewCellKey(ref.table, ref.row, fieldName)
	if value, ok := p.effective[key]; ok {
		return valueAsInt64(value)
	}
	spec, _ := findActivitySpec(ref.table)
	field, ok := findField(spec, fieldName)
	if !ok {
		return 0, fmt.Errorf("field %q does not exist on table %q", fieldName, ref.table)
	}
	rows, err := p.tableRows(ref.table)
	if err != nil {
		return 0, err
	}
	return valueAsInt64(rows[ref.row][field.Index])
}

func (p *linkedUpdatePlanner) rowsOverlap(left, right rowRef) (bool, error) {
	leftStart, leftEnd, err := p.rowTimes(left)
	if err != nil {
		return false, err
	}
	rightStart, rightEnd, err := p.rowTimes(right)
	if err != nil {
		return false, err
	}
	if leftEnd == 0 || rightEnd == 0 {
		return false, nil
	}
	return leftStart <= rightEnd && rightStart <= leftEnd, nil
}

func (p *linkedUpdatePlanner) rowTimes(ref rowRef) (int64, int64, error) {
	spec, ok := findActivitySpec(ref.table)
	if !ok || len(spec.pairs()) == 0 {
		return 0, 0, fmt.Errorf("table %q has no schedule", ref.table)
	}
	rows, err := p.tableRows(ref.table)
	if err != nil {
		return 0, 0, err
	}
	pair := spec.pairs()[0]
	startField, _ := findField(spec, pair.Start)
	endField, _ := findField(spec, pair.End)
	start, err := valueAsInt64(rows[ref.row][startField.Index])
	if err != nil {
		return 0, 0, err
	}
	end, err := valueAsInt64(rows[ref.row][endField.Index])
	return start, end, err
}

func (p *linkedUpdatePlanner) tableRows(name string) ([][]interface{}, error) {
	if rows, ok := p.rows[name]; ok {
		return rows, nil
	}
	rows, exists, err := p.file.TableRows(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("table %q is absent from the current master data", name)
	}
	p.rows[name] = rows
	return rows, nil
}

func (p *linkedUpdatePlanner) relations() (*relationIndex, error) {
	if p.index != nil {
		return p.index, nil
	}
	index := &relationIndex{
		shopsByID:                      make(map[int64]rowRef),
		shopsByCurrency:                make(map[int64][]rowRef),
		termsByID:                      make(map[int64]rowRef),
		termByCurrency:                 make(map[int64]rowRef),
		gachaCurrencies:                make(map[int64][]int64),
		eventLinks:                     make(map[int64][]interface{}),
		missionTermsByChapter:          make(map[int64][]rowRef),
		missionTermsByCurrency:         make(map[int64][]rowRef),
		monthlyCurrenciesByMissionTerm: make(map[int64][]int64),
		eventBannersByText:             make(map[int64][]rowRef),
		loginBannersByID:               make(map[int64][]rowRef),
		missionBannersByTerm:           make(map[int64][]rowRef),
		naviCutInsByChapter:            make(map[int64][]rowRef),
		monthlyCurrenciesByLoginBonus:  make(map[int64][]int64),
	}

	shopRows, err := p.tableRows("m_shop")
	if err != nil {
		return nil, err
	}
	shopsByGroup := make(map[int64][]rowRef)
	for rowIndex, row := range shopRows {
		shopID, _ := integerAt(row, 0)
		groupID, _ := integerAt(row, 7)
		ref := rowRef{table: "m_shop", row: rowIndex}
		index.shopsByID[shopID] = ref
		shopsByGroup[groupID] = append(shopsByGroup[groupID], ref)
	}
	termRows, err := p.tableRows("m_consumable_item_term")
	if err != nil {
		return nil, err
	}
	for rowIndex, row := range termRows {
		termID, _ := integerAt(row, 0)
		index.termsByID[termID] = rowRef{table: "m_consumable_item_term", row: rowIndex}
	}

	cellRows, err := p.tableRows("m_shop_item_cell")
	if err != nil {
		return nil, err
	}
	itemByCell := make(map[int64]int64)
	for _, row := range cellRows {
		cellID, _ := integerAt(row, 0)
		itemID, _ := integerAt(row, 2)
		itemByCell[cellID] = itemID
	}
	itemRows, err := p.tableRows("m_shop_item")
	if err != nil {
		return nil, err
	}
	currencyByItem := make(map[int64]int64)
	for _, row := range itemRows {
		itemID, _ := integerAt(row, 0)
		priceType, _ := integerAt(row, 4)
		priceID, _ := integerAt(row, 5)
		if priceType == int64(model.PriceTypeConsumableItem) {
			currencyByItem[itemID] = priceID
		}
	}
	groupRows, err := p.tableRows("m_shop_item_cell_group")
	if err != nil {
		return nil, err
	}
	shopCurrencySeen := make(map[string]bool)
	for _, row := range groupRows {
		groupID, _ := integerAt(row, 0)
		cellID, _ := integerAt(row, 1)
		currencyID := currencyByItem[itemByCell[cellID]]
		if currencyID == 0 {
			continue
		}
		for _, shop := range shopsByGroup[groupID] {
			key := fmt.Sprintf("%d\x00%s", currencyID, previewRecordKey(shop))
			if shopCurrencySeen[key] {
				continue
			}
			shopCurrencySeen[key] = true
			index.shopsByCurrency[currencyID] = append(index.shopsByCurrency[currencyID], shop)
		}
	}

	monthlyCurrencies := make(map[int64]bool)
	consumableRows, err := p.tableRows("m_consumable_item")
	if err != nil {
		return nil, err
	}
	for _, row := range consumableRows {
		itemID, _ := integerAt(row, 0)
		itemType, _ := integerAt(row, 1)
		termID, _ := integerAt(row, 4)
		assetCategory, _ := integerAt(row, 6)
		if term, ok := index.termsByID[termID]; ok && termID != 0 {
			index.termByCurrency[itemID] = term
		}
		if itemType == mamaMedalItemType && assetCategory == mamaMedalAssetCategory {
			monthlyCurrencies[itemID] = true
		}
	}
	gachaMedalRows, err := p.tableRows("m_gacha_medal")
	if err != nil {
		return nil, err
	}
	for _, row := range gachaMedalRows {
		currencyID, _ := integerAt(row, 2)
		gachaID, _ := integerAt(row, 3)
		index.gachaCurrencies[gachaID] = appendUniqueInt64(index.gachaCurrencies[gachaID], currencyID)
	}
	eventLinkRows, err := p.tableRows("m_event_quest_link")
	if err != nil {
		return nil, err
	}
	for _, row := range eventLinkRows {
		linkID, _ := integerAt(row, 0)
		index.eventLinks[linkID] = row
	}

	missionRewardRows, err := p.tableRows("m_mission_reward")
	if err != nil {
		return nil, err
	}
	monthlyCurrencyByReward := make(map[int64]int64)
	for _, row := range missionRewardRows {
		rewardID, _ := integerAt(row, 0)
		possessionType, _ := integerAt(row, 1)
		possessionID, _ := integerAt(row, 2)
		if possessionType == int64(model.PossessionTypeConsumableItem) && monthlyCurrencies[possessionID] {
			monthlyCurrencyByReward[rewardID] = possessionID
		}
	}
	missionLinkRows, err := p.tableRows("m_mission_link")
	if err != nil {
		return nil, err
	}
	chapterByMissionLink := make(map[int64]int64)
	for _, row := range missionLinkRows {
		linkID, _ := integerAt(row, 0)
		domain, _ := integerAt(row, 1)
		chapterID, _ := integerAt(row, 2)
		if domain == missionLinkDomainEvent {
			chapterByMissionLink[linkID] = chapterID
		}
	}
	missionTermRows, err := p.tableRows("m_mission_term")
	if err != nil {
		return nil, err
	}
	missionTermByID := make(map[int64]rowRef)
	for rowIndex, row := range missionTermRows {
		termID, _ := integerAt(row, 0)
		missionTermByID[termID] = rowRef{table: "m_mission_term", row: rowIndex}
	}
	missionRows, err := p.tableRows("m_mission")
	if err != nil {
		return nil, err
	}
	missionChapterSeen := make(map[string]bool)
	missionCurrencySeen := make(map[string]bool)
	for _, row := range missionRows {
		linkID, _ := integerAt(row, 6)
		rewardID, _ := integerAt(row, 11)
		termID, _ := integerAt(row, 12)
		term, ok := missionTermByID[termID]
		if !ok || termID == 0 {
			continue
		}
		if chapterID := chapterByMissionLink[linkID]; chapterID != 0 {
			key := fmt.Sprintf("%d\x00%s", chapterID, previewRecordKey(term))
			if !missionChapterSeen[key] {
				missionChapterSeen[key] = true
				index.missionTermsByChapter[chapterID] = append(index.missionTermsByChapter[chapterID], term)
			}
		}
		if currencyID := monthlyCurrencyByReward[rewardID]; currencyID != 0 {
			key := fmt.Sprintf("%d\x00%s", currencyID, previewRecordKey(term))
			if !missionCurrencySeen[key] {
				missionCurrencySeen[key] = true
				index.missionTermsByCurrency[currencyID] = append(index.missionTermsByCurrency[currencyID], term)
				index.monthlyCurrenciesByMissionTerm[termID] = appendUniqueInt64(index.monthlyCurrenciesByMissionTerm[termID], currencyID)
			}
		}
	}

	momBannerRows, err := p.tableRows("m_mom_banner")
	if err != nil {
		return nil, err
	}
	for rowIndex, row := range momBannerRows {
		domain, _ := integerAt(row, 2)
		destination, _ := integerAt(row, 3)
		assetName, _ := stringAt(row, 4)
		ref := rowRef{table: "m_mom_banner", row: rowIndex}
		switch domain {
		case momBannerDomainLoginBonus:
			index.loginBannersByID[destination] = append(index.loginBannersByID[destination], ref)
		case momBannerDomainMission:
			index.missionBannersByTerm[destination] = append(index.missionBannersByTerm[destination], ref)
		case momBannerDomainEvent:
			if textID, ok := eventBannerTextID(assetName); ok {
				index.eventBannersByText[textID] = append(index.eventBannersByText[textID], ref)
			}
		}
	}
	naviRows, err := p.tableRows("m_navi_cut_in")
	if err != nil {
		return nil, err
	}
	for rowIndex, row := range naviRows {
		functionType, _ := integerAt(row, 1)
		chapterID, _ := integerAt(row, 6)
		if functionType == naviCutInFunctionEvent {
			index.naviCutInsByChapter[chapterID] = append(index.naviCutInsByChapter[chapterID], rowRef{table: "m_navi_cut_in", row: rowIndex})
		}
	}
	loginStampRows, err := p.tableRows("m_login_bonus_stamp")
	if err != nil {
		return nil, err
	}
	for _, row := range loginStampRows {
		loginBonusID, _ := integerAt(row, 0)
		possessionType, _ := integerAt(row, 3)
		currencyID, _ := integerAt(row, 4)
		if possessionType == int64(model.PossessionTypeConsumableItem) && monthlyCurrencies[currencyID] {
			index.monthlyCurrenciesByLoginBonus[loginBonusID] = appendUniqueInt64(index.monthlyCurrenciesByLoginBonus[loginBonusID], currencyID)
		}
	}

	p.index = index
	return index, nil
}

func eventBannerTextID(assetName string) (int64, bool) {
	const prefix = "event_mom_banner_"
	if !strings.HasPrefix(assetName, prefix) {
		return 0, false
	}
	value := strings.TrimPrefix(assetName, prefix)
	if number, _, found := strings.Cut(value, "_"); found {
		value = number
	}
	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil
}

func appendUniqueInt64(values []int64, value int64) []int64 {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func findActivitySpec(name string) (tableSpec, bool) {
	for _, spec := range activityTableSpecs {
		if spec.Name == name {
			return spec, true
		}
	}
	return tableSpec{}, false
}

func assembleUpdatePreview(catalog *Catalog, requested, planned []Change, impacts []linkedImpact, generated map[string]bool, result UpdateResult) UpdatePreview {
	rows := make(map[string]Row)
	fields := make(map[string]map[string]Field)
	for _, table := range catalog.Tables {
		fields[table.Name] = make(map[string]Field, len(table.Fields))
		for _, field := range table.Fields {
			fields[table.Name][field.Name] = field
		}
		for _, row := range table.Rows {
			rows[previewRecordKey(rowRef{table: table.Name, row: row.Index})] = row
		}
	}
	requestedByRecord := changesByRecord(requested)
	plannedByRecord := changesByRecord(planned)
	coveredRequested := make(map[string]bool)
	preview := UpdatePreview{
		RequestedChanges: len(requested),
		GeneratedChanges: len(generated),
		TotalChanges:     len(planned),
		ChangedRows:      result.ChangedRows,
	}
	for _, impact := range impacts {
		upstreamKey := previewRecordKey(impact.upstream)
		coveredRequested[upstreamKey] = true
		entry := UpdateImpactPreview{
			Kind:     impact.kind,
			Upstream: makeRecordPreview(impact.upstream, "", "", rows, fields, requestedByRecord[upstreamKey], generated),
		}
		sort.Slice(impact.targets, func(i, j int) bool {
			if impact.targets[i].relation != impact.targets[j].relation {
				return impact.targets[i].relation < impact.targets[j].relation
			}
			if impact.targets[i].ref.table != impact.targets[j].ref.table {
				return impact.targets[i].ref.table < impact.targets[j].ref.table
			}
			return impact.targets[i].ref.row < impact.targets[j].ref.row
		})
		for _, target := range impact.targets {
			key := previewRecordKey(target.ref)
			if len(requestedByRecord[key]) != 0 {
				coveredRequested[key] = true
			}
			entry.Downstream = append(entry.Downstream,
				makeRecordPreview(target.ref, target.relation, target.note, rows, fields, plannedByRecord[key], generated))
		}
		preview.Impacts = append(preview.Impacts, entry)
	}
	otherKeys := make([]string, 0)
	for key := range requestedByRecord {
		if !coveredRequested[key] {
			otherKeys = append(otherKeys, key)
		}
	}
	sort.Strings(otherKeys)
	for _, key := range otherKeys {
		ref := parsePreviewRecordKey(key)
		preview.OtherChanges = append(preview.OtherChanges,
			makeRecordPreview(ref, "", "", rows, fields, requestedByRecord[key], generated))
	}
	return preview
}

func changesByRecord(changes []Change) map[string][]Change {
	result := make(map[string][]Change)
	for _, change := range changes {
		key := previewRecordKey(rowRef{table: change.Table, row: change.Row})
		result[key] = append(result[key], change)
	}
	return result
}

func makeRecordPreview(ref rowRef, relation, note string, rows map[string]Row, fields map[string]map[string]Field, changes []Change, generated map[string]bool) RecordPreview {
	row := rows[previewRecordKey(ref)]
	record := RecordPreview{
		Relation: relation,
		Note:     note,
		Table:    ref.table,
		Row:      ref.row,
		Identity: append([]FieldValue(nil), row.Identity...),
		Titles:   cloneTitles(row.Titles),
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Field < changes[j].Field })
	for _, change := range changes {
		field := fields[ref.table][change.Field]
		record.Changes = append(record.Changes, CellChangePreview{
			Field:     change.Field,
			Before:    row.Values[change.Field],
			After:     fmt.Sprint(change.Value),
			Datetime:  field.Datetime,
			Generated: generated[previewCellKey(ref.table, ref.row, change.Field)],
		})
	}
	return record
}

func previewCellKey(table string, row int, field string) string {
	return fmt.Sprintf("%s\x00%d\x00%s", table, row, field)
}

func previewRecordKey(ref rowRef) string {
	return fmt.Sprintf("%s\x00%d", ref.table, ref.row)
}

func parsePreviewRecordKey(key string) rowRef {
	table, rowText, _ := strings.Cut(key, "\x00")
	row, _ := strconv.Atoi(rowText)
	return rowRef{table: table, row: row}
}
