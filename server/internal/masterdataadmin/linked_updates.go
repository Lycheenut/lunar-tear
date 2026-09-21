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
	momBannerDomainMission     int64 = 22
	momBannerDomainEvent       int64 = 23
	eventLinkDomainShop        int64 = 3
	missionLinkDomainEvent     int64 = 4
	naviCutInFunctionEvent     int64 = 2
	claimRedemptionGraceMillis       = int64(48 * 60 * 60 * 1000)
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

type TableReplacementPreview struct {
	Table       string `json:"table"`
	BeforeRows  int    `json:"beforeRows"`
	AfterRows   int    `json:"afterRows"`
	ChangedRows int    `json:"changedRows"`
}

type UpdatePreview struct {
	Impacts            []UpdateImpactPreview      `json:"impacts,omitempty"`
	OtherChanges       []RecordPreview            `json:"otherChanges,omitempty"`
	TableReplacements  []TableReplacementPreview  `json:"tableReplacements,omitempty"`
	RequestedChanges   int                        `json:"requestedChanges"`
	GeneratedChanges   int                        `json:"generatedChanges"`
	TotalChanges       int                        `json:"totalChanges"`
	ChangedRows        int                        `json:"changedRows"`
	QuestBonusGroups   []QuestBonusGroupPreview   `json:"questBonusGroups,omitempty"`
	QuestBonusRestores []QuestBonusRestorePreview `json:"questBonusRestores,omitempty"`
}

type rowRef struct {
	table string
	row   int
}

type relationIndex struct {
	shopsByID             map[int64]rowRef
	shopsByCurrency       map[int64][]rowRef
	termsByID             map[int64]rowRef
	termByCurrency        map[int64]rowRef
	eventLinks            map[int64][]interface{}
	missionTermsByChapter map[int64][]rowRef
	eventBannersByText    map[int64][]rowRef
	missionBannersByTerm  map[int64][]rowRef
	naviCutInsByChapter   map[int64][]rowRef
}

type activityRelationReader struct {
	file  *memorydb.File
	rows  map[string][][]interface{}
	index *relationIndex
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
	validated := request
	validated, bonusPreviews, err := planQuestBonusUpdates(file, validated)
	if err != nil {
		return UpdatePreview{}, err
	}
	candidate, result, err := buildUpdate(file, validated)
	if err != nil {
		return UpdatePreview{}, err
	}
	resolver := newTitleResolver(file, loadLocalizationIndex(path))
	catalog, err := catalogFromFile(file, resolver)
	if err != nil {
		return UpdatePreview{}, err
	}
	for _, change := range validated.Changes {
		if change.Table == questTable {
			spec, _ := findActivitySpec(questTable)
			table, _, err := tableFromFile(file, resolver, spec, true)
			if err != nil {
				return UpdatePreview{}, err
			}
			catalog.Tables = append(catalog.Tables, table)
			break
		}
	}
	preview := assembleUpdatePreview(catalog, request.Changes, validated.Changes, result)
	preview.QuestBonusRestores = bonusPreviews
	if request.MissionRewards != nil {
		current, _, readErr := file.TableRows(missionRewardTable)
		if readErr != nil {
			return UpdatePreview{}, readErr
		}
		replacement, _, changedRows, _, replace, replacementErr :=
			prepareMissionRewardReplacement(file, request.MissionRewards, nil, false)
		if replacementErr != nil {
			return UpdatePreview{}, replacementErr
		}
		if replace {
			preview.TableReplacements = append(preview.TableReplacements, TableReplacementPreview{
				Table: missionRewardTable, BeforeRows: len(current), AfterRows: len(replacement), ChangedRows: changedRows,
			})
		}
	}
	if request.ShopItemCellGroups != nil {
		current, _, readErr := file.TableRows(shopItemCellGroupTable)
		if readErr != nil {
			return UpdatePreview{}, readErr
		}
		_, changedRows, replacementErr := buildShopItemCellGroupReplacement(file, request.ShopItemCellGroups)
		if replacementErr != nil {
			return UpdatePreview{}, replacementErr
		}
		preview.TableReplacements = append(preview.TableReplacements, TableReplacementPreview{
			Table: shopItemCellGroupTable, BeforeRows: len(current),
			AfterRows: len(*request.ShopItemCellGroups), ChangedRows: changedRows,
		})
	}
	if request.ShopItemCells != nil && (len(request.ShopItemCells.Additions) != 0 || len(request.ShopItemCells.Deletes) != 0) {
		current, _, readErr := file.TableRows(shopItemCellTable)
		if readErr != nil {
			return UpdatePreview{}, readErr
		}
		replacement, _, changedRows, _, replacementErr := buildShopItemCellReplacement(file, request.ShopItemCells, nil)
		if replacementErr != nil {
			return UpdatePreview{}, replacementErr
		}
		preview.TableReplacements = append(preview.TableReplacements, TableReplacementPreview{
			Table: shopItemCellTable, BeforeRows: len(current), AfterRows: len(replacement), ChangedRows: changedRows,
		})
	}
	if request.ShopItems != nil && (len(request.ShopItems.Copies) != 0 || len(request.ShopItems.DeleteIDs) != 0) {
		current, _, readErr := file.TableRows(shopItemTable)
		if readErr != nil {
			return UpdatePreview{}, readErr
		}
		replacement, _, changedRows, _, replacementErr := buildShopItemReplacement(file, request.ShopItems, nil)
		if replacementErr != nil {
			return UpdatePreview{}, replacementErr
		}
		preview.TableReplacements = append(preview.TableReplacements, TableReplacementPreview{
			Table: shopItemTable, BeforeRows: len(current), AfterRows: len(replacement), ChangedRows: changedRows,
		})
		possessionCurrent, _, possessionReadErr := file.TableRows(shopItemContentPossessionTable)
		if possessionReadErr != nil {
			return UpdatePreview{}, possessionReadErr
		}
		possessionReplacement, _, possessionChangedRows, replacePossessions, possessionReplacementErr :=
			buildShopItemContentPossessionReplacement(file, request.ShopItems, nil)
		if possessionReplacementErr != nil {
			return UpdatePreview{}, possessionReplacementErr
		}
		if replacePossessions {
			preview.TableReplacements = append(preview.TableReplacements, TableReplacementPreview{
				Table: shopItemContentPossessionTable, BeforeRows: len(possessionCurrent),
				AfterRows: len(possessionReplacement), ChangedRows: possessionChangedRows,
			})
		}
	}
	if len(validated.QuestBonusGroups) != 0 && len(bonusPreviews) == 0 {
		candidateFile, err := memorydb.OpenBytes(candidate)
		if err != nil {
			return UpdatePreview{}, err
		}
		if err := appendQuestBonusPreview(&preview, file, candidateFile, validated); err != nil {
			return UpdatePreview{}, err
		}
	}
	return preview, nil
}

func validateUpdateEnvelope(request UpdateRequest) error {
	if request.ExpectedVersion == "" {
		return fmt.Errorf("expectedVersion is required")
	}
	if len(request.Changes) == 0 && request.MissionRewards == nil && request.ShopItemCellGroups == nil && request.ShopItemCells == nil && request.ShopItems == nil && len(request.QuestBonusGroups) == 0 && len(request.QuestBonusRestores) == 0 {
		return fmt.Errorf("at least one change is required")
	}
	if len(request.Changes) > 10000 {
		return fmt.Errorf("too many changes")
	}
	return nil
}

func (p *activityRelationReader) selectNaviCutIns(candidates []rowRef, sourceStart int64) ([]rowRef, error) {
	// Event reruns reuse both the chapter and content-group IDs, and SortOrder is
	// usually identical. The occurrence closest to the chapter start is unique
	// in the current master data; ID only makes an unexpected tie deterministic.
	type selection struct {
		ref      rowRef
		distance uint64
		id       int64
	}
	byContentGroup := make(map[int64]selection)
	for _, candidate := range candidates {
		rows, err := p.tableRows(candidate.table)
		if err != nil {
			return nil, err
		}
		row := rows[candidate.row]
		id, _ := integerAt(row, 0)
		start, _ := integerAt(row, 3)
		contentGroupID, _ := integerAt(row, 5)
		distance := datetimeDistance(start, sourceStart)
		current, exists := byContentGroup[contentGroupID]
		if !exists || distance < current.distance || distance == current.distance && id < current.id {
			byContentGroup[contentGroupID] = selection{ref: candidate, distance: distance, id: id}
		}
	}
	result := make([]rowRef, 0, len(byContentGroup))
	for _, selected := range byContentGroup {
		result = append(result, selected.ref)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].row < result[j].row })
	return result, nil
}

func datetimeDistance(left, right int64) uint64 {
	if left >= right {
		return uint64(left - right)
	}
	return uint64(right - left)
}

func (p *activityRelationReader) rowsOverlap(left, right rowRef) (bool, error) {
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

func (p *activityRelationReader) rowTimes(ref rowRef) (int64, int64, error) {
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

func (p *activityRelationReader) tableRows(name string) ([][]interface{}, error) {
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

func (p *activityRelationReader) relations() (*relationIndex, error) {
	if p.index != nil {
		return p.index, nil
	}
	index := &relationIndex{
		shopsByID:             make(map[int64]rowRef),
		shopsByCurrency:       make(map[int64][]rowRef),
		termsByID:             make(map[int64]rowRef),
		termByCurrency:        make(map[int64]rowRef),
		eventLinks:            make(map[int64][]interface{}),
		missionTermsByChapter: make(map[int64][]rowRef),
		eventBannersByText:    make(map[int64][]rowRef),
		missionBannersByTerm:  make(map[int64][]rowRef),
		naviCutInsByChapter:   make(map[int64][]rowRef),
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

	consumableRows, err := p.tableRows("m_consumable_item")
	if err != nil {
		return nil, err
	}
	for _, row := range consumableRows {
		itemID, _ := integerAt(row, 0)
		termID, _ := integerAt(row, 4)
		if term, ok := index.termsByID[termID]; ok && termID != 0 {
			index.termByCurrency[itemID] = term
		}

	}
	eventLinkRows, err := p.tableRows("m_event_quest_link")
	if err != nil {
		return nil, err
	}
	for _, row := range eventLinkRows {
		linkID, _ := integerAt(row, 0)
		index.eventLinks[linkID] = row
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
	for _, row := range missionRows {
		linkID, _ := integerAt(row, 6)
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

func findActivitySpec(name string) (tableSpec, bool) {
	for _, spec := range editableTableSpecs {
		if spec.Name == name {
			return spec, true
		}
	}
	return tableSpec{}, false
}

func assembleUpdatePreview(catalog *Catalog, requested, planned []Change, result UpdateResult) UpdatePreview {
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
	fields["m_mission"] = map[string]Field{
		"MissionId":       {Name: "MissionId", Type: "int", Kind: string(fieldKindInt32), PrimaryKey: true},
		"MissionRewardId": {Name: "MissionRewardId", Type: "int", Kind: string(fieldKindInt32)},
		"MissionTermId":   {Name: "MissionTermId", Type: "int", Kind: string(fieldKindInt32)},
	}
	for _, mission := range catalog.MissionSources.Missions {
		rows[previewRecordKey(rowRef{table: "m_mission", row: mission.Row})] = Row{
			Index:    mission.Row,
			Identity: []FieldValue{{Name: "MissionId", Value: strconv.FormatInt(mission.MissionID, 10)}},
			Values: map[string]string{
				"MissionId":       strconv.FormatInt(mission.MissionID, 10),
				"MissionRewardId": strconv.FormatInt(mission.MissionRewardID, 10),
				"MissionTermId":   strconv.FormatInt(mission.MissionTermID, 10),
			},
			Titles: cloneTitles(mission.Names),
		}
	}
	fields["m_shop_item_cell"] = map[string]Field{
		"ShopItemCellId": {Name: "ShopItemCellId", Type: "int", Kind: string(fieldKindInt32), PrimaryKey: true},
		"StepNumber":     {Name: "StepNumber", Type: "int", Kind: string(fieldKindInt32), PrimaryKey: true},
		"ShopItemId":     {Name: "ShopItemId", Type: "int", Kind: string(fieldKindInt32)},
	}
	for _, cell := range catalog.ShopEditor.Cells {
		rows[previewRecordKey(rowRef{table: "m_shop_item_cell", row: int(cell.Row)})] = Row{
			Index: int(cell.Row),
			Identity: []FieldValue{
				{Name: "ShopItemCellId", Value: strconv.FormatInt(cell.ShopItemCellID, 10)},
				{Name: "StepNumber", Value: strconv.FormatInt(cell.StepNumber, 10)},
			},
			Values: map[string]string{
				"ShopItemCellId": strconv.FormatInt(cell.ShopItemCellID, 10),
				"StepNumber":     strconv.FormatInt(cell.StepNumber, 10),
				"ShopItemId":     strconv.FormatInt(cell.ShopItemID, 10),
			},
		}
	}
	fields["m_shop_item"] = map[string]Field{
		"ShopItemId":             {Name: "ShopItemId", Type: "int", Kind: string(fieldKindInt32), PrimaryKey: true},
		"PriceType":              {Name: "PriceType", Type: "PriceType", Kind: string(fieldKindInt32)},
		"PriceId":                {Name: "PriceId", Type: "int", Kind: string(fieldKindInt32)},
		"Price":                  {Name: "Price", Type: "int", Kind: string(fieldKindInt32)},
		"RegularPrice":           {Name: "RegularPrice", Type: "int", Kind: string(fieldKindInt32)},
		"ShopItemLimitedStockId": {Name: "ShopItemLimitedStockId", Type: "int", Kind: string(fieldKindInt32)},
	}
	for _, item := range catalog.ShopEditor.Items {
		rows[previewRecordKey(rowRef{table: "m_shop_item", row: int(item.Row)})] = Row{
			Index:    int(item.Row),
			Identity: []FieldValue{{Name: "ShopItemId", Value: strconv.FormatInt(item.ShopItemID, 10)}},
			Values: map[string]string{
				"ShopItemId":             strconv.FormatInt(item.ShopItemID, 10),
				"PriceType":              strconv.FormatInt(item.PriceType, 10),
				"PriceId":                strconv.FormatInt(item.PriceID, 10),
				"Price":                  strconv.FormatInt(item.Price, 10),
				"RegularPrice":           strconv.FormatInt(item.RegularPrice, 10),
				"ShopItemLimitedStockId": strconv.FormatInt(item.ShopItemLimitedStockID, 10),
			},
			Titles: cloneTitles(item.Names),
		}
	}
	requestedByRecord := changesByRecord(requested)
	preview := UpdatePreview{
		RequestedChanges: len(requested),
		TotalChanges:     len(planned),
		ChangedRows:      result.ChangedRows,
	}
	otherKeys := make([]string, 0)
	for key := range requestedByRecord {
		otherKeys = append(otherKeys, key)
	}
	sort.Strings(otherKeys)
	for _, key := range otherKeys {
		ref := parsePreviewRecordKey(key)
		preview.OtherChanges = append(preview.OtherChanges,
			makeRecordPreview(ref, "", "", rows, fields, requestedByRecord[key], nil))
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
