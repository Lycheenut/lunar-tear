// Package masterdataadmin exposes the operational-activity subset of master
// data and produces validated replacement binaries for the live admin service.
package masterdataadmin

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"lunar-tear/server/internal/masterdata/memorydb"
)

const maxDatetimeMillis int64 = 253402300799999

var ErrVersionConflict = errors.New("master data changed since it was loaded")

type FieldValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Row struct {
	Index            int                 `json:"index"`
	Identity         []FieldValue        `json:"identity"`
	Values           map[string]string   `json:"values"`
	Times            map[string]int64    `json:"times"`
	Titles           map[string]string   `json:"titles,omitempty"`
	ContentBody      map[string]string   `json:"contentBody,omitempty"`
	DokanImages      []DokanImage        `json:"dokanImages,omitempty"`
	ContentFootnotes []map[string]string `json:"contentFootnotes,omitempty"`
	ShopRelations    []ShopRelation      `json:"shopRelations,omitempty"`
}

type DokanImage struct {
	ContentIndex int64 `json:"contentIndex"`
	ImageID      int64 `json:"imageId"`
}

type Field struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Kind       string `json:"kind"`
	PrimaryKey bool   `json:"primaryKey"`
	Datetime   bool   `json:"datetime"`
}

type Table struct {
	Name       string     `json:"name"`
	EntityName string     `json:"entityName"`
	Primary    bool       `json:"primary"`
	Delivery   bool       `json:"delivery,omitempty"`
	Fields     []Field    `json:"fields"`
	TimeFields []string   `json:"timeFields"`
	Pairs      []timePair `json:"pairs"`
	RowCount   int        `json:"rowCount"`
	Rows       []Row      `json:"rows"`
}

type Catalog struct {
	Version         string                 `json:"version"`
	DefaultLanguage string                 `json:"defaultLanguage"`
	Languages       []string               `json:"languages"`
	TableCount      int                    `json:"tableCount"`
	PrimaryCount    int                    `json:"primaryCount"`
	RelatedCount    int                    `json:"relatedCount"`
	DeliveryCount   int                    `json:"deliveryCount"`
	RowCount        int                    `json:"rowCount"`
	Tables          []Table                `json:"tables"`
	MissionSources  MissionSourceCatalog   `json:"missionSources"`
	ShopEditor      ShopEditorCatalog      `json:"shopEditor"`
	QuestDropEditor QuestDropEditorCatalog `json:"questDropEditor"`
}

type Change struct {
	Table string      `json:"table"`
	Row   int         `json:"row"`
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

type UpdateRequest struct {
	ExpectedVersion    string                        `json:"expectedVersion"`
	Changes            []Change                      `json:"changes"`
	MissionRewards     *[]MissionRewardInput         `json:"missionRewards,omitempty"`
	ShopItemCellGroups *[]ShopItemCellGroupInput     `json:"shopItemCellGroups,omitempty"`
	ShopItemCells      *ShopItemCellStructuralUpdate `json:"shopItemCells,omitempty"`
	ShopItems          *ShopItemStructuralUpdate     `json:"shopItems,omitempty"`
}

type UpdateResult struct {
	Version      string `json:"version"`
	ChangedCells int    `json:"changedCells"`
	ChangedRows  int    `json:"changedRows"`
}

func Load(path string) (*Catalog, error) {
	file, err := memorydb.OpenFile(path)
	if err != nil {
		return nil, err
	}
	resolver := newTitleResolver(file, loadLocalizationIndex(path))
	return catalogFromFile(file, resolver)
}

// LoadMetadata returns table definitions and row counts without serializing
// any table rows or editor-specific supporting catalogs.
func LoadMetadata(path string) (*Catalog, error) {
	file, err := memorydb.OpenFile(path)
	if err != nil {
		return nil, err
	}
	catalog := newCatalog(file)
	for _, spec := range activityTableSpecs {
		table, exists, err := tableFromFile(file, nil, spec, false)
		if err != nil {
			return nil, err
		}
		if exists {
			appendCatalogTable(catalog, table)
		}
	}
	return catalog, nil
}

// AppendGachaScheduleMetadata exposes the JSON-backed limited Gacha inventory
// alongside binary-backed operational schedules. Rows are loaded separately
// from the Gacha configuration endpoint.
func AppendGachaScheduleMetadata(catalog *Catalog, rowCount int) {
	appendCatalogTable(catalog, Table{
		Name:       "gacha",
		EntityName: "Gacha",
		Primary:    true,
		Fields: []Field{
			{Name: "GachaId", Type: "int32", Kind: "int32", PrimaryKey: true},
			{Name: "BannerAssetName", Type: "string", Kind: "string", PrimaryKey: true},
			{Name: "GachaMedalId", Type: "int32", Kind: "int32"},
			{Name: "StartDatetime", Type: "int64", Kind: "int64", Datetime: true},
			{Name: "EndDatetime", Type: "int64", Kind: "int64", Datetime: true},
		},
		TimeFields: []string{"StartDatetime", "EndDatetime"},
		Pairs:      []timePair{{Start: "StartDatetime", End: "EndDatetime"}},
		RowCount:   rowCount,
	})
}

// LoadTable returns one table's rows and only the supporting data required by
// that table's specialized editor. A small number of display-only dependency
// tables are included for title and banner previews.
func LoadTable(path, tableName string) (*Catalog, error) {
	file, err := memorydb.OpenFile(path)
	if err != nil {
		return nil, err
	}
	resolver := newTitleResolver(file, loadLocalizationIndex(path))
	catalog := newCatalog(file)

	requested := []string{tableName}
	switch tableName {
	case "m_login_bonus_stamp":
		requested = append(requested, "m_login_bonus")
	case "m_mom_banner":
		requested = append(requested, "m_login_bonus", "m_event_quest_chapter")
	}

	seen := make(map[string]struct{}, len(requested))
	for _, name := range requested {
		if _, duplicate := seen[name]; duplicate {
			continue
		}
		seen[name] = struct{}{}
		spec, ok := tableSpecByName(name)
		if !ok {
			return nil, fmt.Errorf("table %q is not an activity configuration table", name)
		}
		table, exists, err := tableFromFile(file, resolver, spec, true)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("table %q is absent from the current master data", name)
		}
		appendCatalogTable(catalog, table)
	}

	switch tableName {
	case "m_mission_reward", "m_mission_term":
		catalog.MissionSources = loadMissionSources(file, resolver)
	case "m_shop_item_content_possession":
		catalog.ShopEditor = loadShopEditor(file, resolver)
	case "m_quest_pickup_reward_group":
		catalog.QuestDropEditor = loadQuestDropEditor(file, resolver)
	}
	return catalog, nil
}

func BuildUpdate(path string, request UpdateRequest) ([]byte, UpdateResult, error) {
	if err := validateUpdateEnvelope(request); err != nil {
		return nil, UpdateResult{}, err
	}
	file, err := memorydb.OpenFile(path)
	if err != nil {
		return nil, UpdateResult{}, err
	}
	if file.Version() != request.ExpectedVersion {
		return nil, UpdateResult{}, ErrVersionConflict
	}
	planned, _, _, err := expandLinkedChanges(file, request.Changes)
	if err != nil {
		return nil, UpdateResult{}, err
	}
	request.Changes = planned
	return buildUpdate(file, request)
}

func buildUpdate(file *memorydb.File, request UpdateRequest) ([]byte, UpdateResult, error) {
	if request.ExpectedVersion == "" {
		return nil, UpdateResult{}, fmt.Errorf("expectedVersion is required")
	}
	if len(request.Changes) > 10000 {
		return nil, UpdateResult{}, fmt.Errorf("too many changes")
	}

	if file.Version() != request.ExpectedVersion {
		return nil, UpdateResult{}, ErrVersionConflict
	}

	specByName := make(map[string]tableSpec, len(editableTableSpecs))
	rowsByTable := make(map[string][][]interface{})
	for _, spec := range editableTableSpecs {
		specByName[spec.Name] = spec
	}
	overrides := make(map[string]map[int]map[string]int64)
	seen := make(map[string]struct{}, len(request.Changes))
	var edits []memorydb.CellEdit
	changedRows := make(map[string]struct{})

	for _, change := range request.Changes {
		spec, ok := specByName[change.Table]
		if !ok {
			return nil, UpdateResult{}, fmt.Errorf("table %q is not an editable activity table", change.Table)
		}
		field, ok := findField(spec, change.Field)
		if !ok {
			return nil, UpdateResult{}, fmt.Errorf("field %q does not exist on table %q", change.Field, change.Table)
		}
		if field.PrimaryKey {
			return nil, UpdateResult{}, fmt.Errorf("primary key field %q is read-only on table %q", change.Field, change.Table)
		}
		value, err := parseChangeValue(field, change.Value)
		if err != nil {
			return nil, UpdateResult{}, fmt.Errorf("%s row %d field %s: %w", change.Table, change.Row, change.Field, err)
		}
		if change.Table == missionRewardTable && change.Field == "Count" && value.(int32) < 0 {
			return nil, UpdateResult{}, fmt.Errorf("%s row %d field Count: cannot be negative", change.Table, change.Row)
		}
		if change.Table == bigHuntRewardGroupTable && change.Field == "Count" && value.(int32) <= 0 {
			return nil, UpdateResult{}, fmt.Errorf("%s row %d field Count: must be greater than zero", change.Table, change.Row)
		}
		editKey := fmt.Sprintf("%s\x00%d\x00%s", change.Table, change.Row, change.Field)
		if _, duplicate := seen[editKey]; duplicate {
			return nil, UpdateResult{}, fmt.Errorf("duplicate change for %s row %d field %s", change.Table, change.Row, change.Field)
		}
		seen[editKey] = struct{}{}

		rows, loaded := rowsByTable[change.Table]
		if !loaded {
			var exists bool
			rows, exists, err = file.TableRows(change.Table)
			if err != nil {
				return nil, UpdateResult{}, err
			}
			if !exists {
				return nil, UpdateResult{}, fmt.Errorf("table %q is absent from the current master data", change.Table)
			}
			rowsByTable[change.Table] = rows
		}
		if change.Row < 0 || change.Row >= len(rows) {
			return nil, UpdateResult{}, fmt.Errorf("row %d is outside table %q", change.Row, change.Table)
		}
		if field.Index >= len(rows[change.Row]) {
			return nil, UpdateResult{}, fmt.Errorf("field %q is outside row %d in table %q", change.Field, change.Row, change.Table)
		}
		unchanged, err := scalarValuesEqual(field, rows[change.Row][field.Index], value)
		if err != nil {
			return nil, UpdateResult{}, fmt.Errorf("%s row %d field %s: %w", change.Table, change.Row, change.Field, err)
		}
		if unchanged {
			continue
		}
		if field.Datetime {
			if overrides[change.Table] == nil {
				overrides[change.Table] = make(map[int]map[string]int64)
			}
			if overrides[change.Table][change.Row] == nil {
				overrides[change.Table][change.Row] = make(map[string]int64)
			}
			overrides[change.Table][change.Row][change.Field] = value.(int64)
		}
		edits = append(edits, memorydb.CellEdit{Table: change.Table, Row: change.Row, Column: field.Index, Value: value})
		changedRows[fmt.Sprintf("%s\x00%d", change.Table, change.Row)] = struct{}{}
	}
	for tableName, rowOverrides := range overrides {
		spec := specByName[tableName]
		rows := rowsByTable[tableName]
		for rowIndex, values := range rowOverrides {
			for _, pair := range spec.pairs() {
				start, err := effectiveTime(rows[rowIndex], spec, pair.Start, values)
				if err != nil {
					return nil, UpdateResult{}, fmt.Errorf("%s row %d: %w", tableName, rowIndex, err)
				}
				end, err := effectiveTime(rows[rowIndex], spec, pair.End, values)
				if err != nil {
					return nil, UpdateResult{}, fmt.Errorf("%s row %d: %w", tableName, rowIndex, err)
				}
				// End=0 is used by this master data to explicitly disable a row.
				if end != 0 && start > end {
					return nil, UpdateResult{}, fmt.Errorf("%s must not be after %s", pair.Start, pair.End)
				}
			}
		}
	}

	replacements := make(map[string][][]interface{})
	missionRewardRows, missionRewardCells, missionRewardChangedRows, _, replaceMissionRewards, err :=
		prepareMissionRewardReplacement(file, request.MissionRewards, edits, true)
	if err != nil {
		return nil, UpdateResult{}, err
	}
	if replaceMissionRewards {
		replacements[missionRewardTable] = missionRewardRows
		filtered := edits[:0]
		for _, edit := range edits {
			if edit.Table != missionRewardTable {
				filtered = append(filtered, edit)
			}
		}
		edits = filtered
		for key := range changedRows {
			if strings.HasPrefix(key, missionRewardTable+"\x00") {
				delete(changedRows, key)
			}
		}
	}
	structuralCells, structuralRows, err := buildShopItemCellGroupReplacement(file, request.ShopItemCellGroups)
	if err != nil {
		return nil, UpdateResult{}, err
	}
	if structuralRows != 0 {
		replacements[shopItemCellGroupTable] = shopItemCellGroupRows(*request.ShopItemCellGroups)
	}
	structuralCells += missionRewardCells
	structuralRows += missionRewardChangedRows
	originalEditCount := len(edits)
	cellRows, cellCells, cellChangedRows, replaceCells, err := buildShopItemCellReplacement(file, request.ShopItemCells, edits)
	if err != nil {
		return nil, UpdateResult{}, err
	}
	if replaceCells {
		replacements[shopItemCellTable] = cellRows
	}
	structuralCells += cellCells
	structuralRows += cellChangedRows
	shopItemRows, shopItemCells, shopItemChangedRows, replaceShopItems, err := buildShopItemReplacement(file, request.ShopItems, edits)
	if err != nil {
		return nil, UpdateResult{}, err
	}
	if replaceShopItems {
		replacements[shopItemTable] = shopItemRows
	}
	structuralCells += shopItemCells
	structuralRows += shopItemChangedRows
	possessionRows, possessionCells, possessionChangedRows, replacePossessions, err := buildShopItemContentPossessionReplacement(file, request.ShopItems, edits)
	if err != nil {
		return nil, UpdateResult{}, err
	}
	if replacePossessions {
		replacements[shopItemContentPossessionTable] = possessionRows
	}
	if replaceCells || replaceShopItems || replacePossessions {
		filtered := edits[:0]
		for _, edit := range edits {
			if (replaceCells && edit.Table == shopItemCellTable) ||
				(replaceShopItems && edit.Table == shopItemTable) ||
				(replacePossessions && edit.Table == shopItemContentPossessionTable) {
				continue
			}
			filtered = append(filtered, edit)
		}
		edits = filtered
	}
	structuralCells += possessionCells
	structuralRows += possessionChangedRows
	if originalEditCount == 0 && structuralRows == 0 {
		return nil, UpdateResult{}, fmt.Errorf("the submitted values are unchanged")
	}

	candidate, err := file.RebuildTables(edits, replacements)
	if err != nil {
		return nil, UpdateResult{}, err
	}
	candidateFile, err := memorydb.OpenBytes(candidate)
	if err != nil {
		return nil, UpdateResult{}, fmt.Errorf("verify candidate: %w", err)
	}
	if changesBigHuntRewards(request.Changes) {
		if err := validateBigHuntRewardConfig(candidateFile); err != nil {
			return nil, UpdateResult{}, fmt.Errorf("validate big hunt rewards: %w", err)
		}
	}
	return candidate, UpdateResult{
		Version:      candidateFile.Version(),
		ChangedCells: originalEditCount + structuralCells,
		ChangedRows:  len(changedRows) + structuralRows,
	}, nil
}

func catalogFromFile(file *memorydb.File, resolver *titleResolver) (*Catalog, error) {
	catalog := newCatalog(file)
	catalog.MissionSources = loadMissionSources(file, resolver)
	catalog.ShopEditor = loadShopEditor(file, resolver)
	catalog.QuestDropEditor = loadQuestDropEditor(file, resolver)
	for _, spec := range activityTableSpecs {
		table, exists, err := tableFromFile(file, resolver, spec, true)
		if err != nil {
			return nil, err
		}
		if exists {
			appendCatalogTable(catalog, table)
		}
	}
	return catalog, nil
}

func newCatalog(file *memorydb.File) *Catalog {
	return &Catalog{
		Version:         file.Version(),
		DefaultLanguage: "en",
		Languages:       append([]string(nil), supportedLanguages...),
	}
}

func tableSpecByName(name string) (tableSpec, bool) {
	for _, spec := range activityTableSpecs {
		if spec.Name == name {
			return spec, true
		}
	}
	return tableSpec{}, false
}

func appendCatalogTable(catalog *Catalog, table Table) {
	catalog.RowCount += table.RowCount
	catalog.Tables = append(catalog.Tables, table)
	if table.Delivery {
		catalog.DeliveryCount++
	} else if table.Primary {
		catalog.PrimaryCount++
	} else {
		catalog.RelatedCount++
	}
	catalog.TableCount = len(catalog.Tables)
}

func tableFromFile(file *memorydb.File, resolver *titleResolver, spec tableSpec, includeRows bool) (Table, bool, error) {
	rows, exists, err := file.TableRows(spec.Name)
	if err != nil || !exists {
		return Table{}, exists, err
	}
	table := Table{
		Name:       spec.Name,
		EntityName: spec.EntityName,
		Primary:    spec.Primary,
		Delivery:   spec.Delivery,
		Pairs:      spec.pairs(),
		RowCount:   len(rows),
	}
	for _, field := range spec.Fields {
		table.Fields = append(table.Fields, Field{
			Name:       field.Name,
			Type:       field.SchemaType,
			Kind:       string(field.Kind),
			PrimaryKey: field.PrimaryKey,
			Datetime:   field.Datetime,
		})
		if field.Datetime {
			table.TimeFields = append(table.TimeFields, field.Name)
		}
	}
	if !includeRows {
		return table, true, nil
	}
	table.Rows = make([]Row, 0, len(rows))
	for rowIndex, values := range rows {
		row := Row{
			Index:  rowIndex,
			Values: make(map[string]string, len(spec.Fields)),
			Times:  make(map[string]int64, len(spec.Times)),
		}
		for _, field := range spec.Fields {
			if field.Index >= len(values) {
				return Table{}, true, fmt.Errorf("table %q row %d is missing column %s", spec.Name, rowIndex, field.Name)
			}
			row.Values[field.Name] = formatValue(values[field.Index])
			if field.PrimaryKey {
				row.Identity = append(row.Identity, FieldValue{Name: field.Name, Value: row.Values[field.Name]})
			}
			if !field.Datetime {
				continue
			}
			value, err := valueAsInt64(values[field.Index])
			if err != nil {
				return Table{}, true, fmt.Errorf("table %q row %d field %s: %w", spec.Name, rowIndex, field.Name, err)
			}
			row.Times[field.Name] = value
		}
		row.Titles = resolver.resolve(spec.Name, values)
		row.ContentBody = resolver.resolveContentBody(spec.Name, values)
		row.DokanImages = resolver.resolveDokanImages(spec.Name, values)
		row.ShopRelations = resolver.resolveShopRelations(spec.Name, values)
		row.ContentFootnotes = resolver.resolveContentFootnotes(spec.Name, values, row.ShopRelations)
		table.Rows = append(table.Rows, row)
	}
	return table, true, nil
}

func findField(spec tableSpec, name string) (columnSpec, bool) {
	for _, field := range spec.Fields {
		if field.Name == name {
			return field, true
		}
	}
	return columnSpec{}, false
}

func parseChangeValue(field columnSpec, raw interface{}) (interface{}, error) {
	switch field.Kind {
	case fieldKindInt32:
		value, err := parseInteger(raw, 32)
		if err != nil {
			return nil, err
		}
		return int32(value), nil
	case fieldKindInt64:
		value, err := parseInteger(raw, 64)
		if err != nil {
			return nil, err
		}
		if field.Datetime && (value < 0 || value > maxDatetimeMillis) {
			return nil, fmt.Errorf("value is outside the supported datetime range")
		}
		return value, nil
	case fieldKindBool:
		switch value := raw.(type) {
		case bool:
			return value, nil
		case string:
			if value == "true" {
				return true, nil
			}
			if value == "false" {
				return false, nil
			}
		}
		return nil, fmt.Errorf("expected true or false")
	case fieldKindString:
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", raw)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported field type %q", field.SchemaType)
	}
}

func parseInteger(raw interface{}, bitSize int) (int64, error) {
	if text, ok := raw.(string); ok {
		value, err := strconv.ParseInt(text, 10, bitSize)
		if err != nil {
			return 0, fmt.Errorf("expected signed %d-bit integer", bitSize)
		}
		return value, nil
	}
	if number, ok := raw.(float64); ok {
		if math.IsNaN(number) || math.IsInf(number, 0) || math.Trunc(number) != number {
			return 0, fmt.Errorf("expected integer")
		}
		if bitSize == 32 && (number < math.MinInt32 || number > math.MaxInt32) {
			return 0, fmt.Errorf("expected signed 32-bit integer")
		}
		if number < math.MinInt64 || number > math.MaxInt64 || math.Abs(number) > 1<<53 {
			return 0, fmt.Errorf("large 64-bit integers must be submitted as strings")
		}
		return int64(number), nil
	}
	value, err := valueAsInt64(raw)
	if err != nil {
		return 0, err
	}
	if bitSize == 32 && (value < math.MinInt32 || value > math.MaxInt32) {
		return 0, fmt.Errorf("expected signed 32-bit integer")
	}
	return value, nil
}

func scalarValuesEqual(field columnSpec, current, updated interface{}) (bool, error) {
	switch field.Kind {
	case fieldKindInt32, fieldKindInt64:
		currentInteger, err := valueAsInt64(current)
		if err != nil {
			return false, err
		}
		updatedInteger, err := valueAsInt64(updated)
		if err != nil {
			return false, err
		}
		return currentInteger == updatedInteger, nil
	case fieldKindBool:
		currentBool, ok := current.(bool)
		if !ok {
			return false, fmt.Errorf("expected bool, got %T", current)
		}
		return currentBool == updated.(bool), nil
	case fieldKindString:
		currentString, ok := current.(string)
		if !ok {
			return false, fmt.Errorf("expected string, got %T", current)
		}
		return currentString == updated.(string), nil
	default:
		return false, fmt.Errorf("unsupported field type %q", field.SchemaType)
	}
}

func findTimeField(spec tableSpec, name string) (columnSpec, bool) {
	for _, field := range spec.Times {
		if field.Name == name {
			return field, true
		}
	}
	return columnSpec{}, false
}

func effectiveTime(row []interface{}, spec tableSpec, fieldName string, overrides map[string]int64) (int64, error) {
	if value, ok := overrides[fieldName]; ok {
		return value, nil
	}
	field, ok := findTimeField(spec, fieldName)
	if !ok || field.Index >= len(row) {
		return 0, fmt.Errorf("missing field %s", fieldName)
	}
	return valueAsInt64(row[field.Index])
}

func valueAsInt64(value interface{}) (int64, error) {
	switch value := value.(type) {
	case int:
		return int64(value), nil
	case int8:
		return int64(value), nil
	case int16:
		return int64(value), nil
	case int32:
		return int64(value), nil
	case int64:
		return value, nil
	case uint:
		if uint64(value) > math.MaxInt64 {
			return 0, fmt.Errorf("integer overflows int64")
		}
		return int64(value), nil
	case uint8:
		return int64(value), nil
	case uint16:
		return int64(value), nil
	case uint32:
		return int64(value), nil
	case uint64:
		if value > math.MaxInt64 {
			return 0, fmt.Errorf("integer overflows int64")
		}
		return int64(value), nil
	default:
		return 0, fmt.Errorf("expected integer, got %T", value)
	}
}

func formatValue(value interface{}) string {
	switch value := value.(type) {
	case string:
		return value
	case []byte:
		return fmt.Sprintf("%x", value)
	case nil:
		return "null"
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
