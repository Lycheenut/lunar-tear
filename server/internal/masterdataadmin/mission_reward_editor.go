package masterdataadmin

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/masterdata/memorydb"
)

const missionRewardTable = "m_mission_reward"

type MissionRewardInput struct {
	MissionRewardID int32 `json:"missionRewardId"`
	PossessionType  int32 `json:"possessionType"`
	PossessionID    int32 `json:"possessionId"`
	Count           int32 `json:"count"`
}

func missionRewardInputAt(row []interface{}) (MissionRewardInput, bool) {
	values := make([]int64, 4)
	for column := range values {
		value, ok := integerAt(row, column)
		if !ok {
			return MissionRewardInput{}, false
		}
		values[column] = value
	}
	return MissionRewardInput{
		MissionRewardID: int32(values[0]), PossessionType: int32(values[1]),
		PossessionID: int32(values[2]), Count: int32(values[3]),
	}, true
}

func missionRewardRows(rows []MissionRewardInput) [][]interface{} {
	ordered := append([]MissionRewardInput(nil), rows...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].MissionRewardID < ordered[j].MissionRewardID
	})
	result := make([][]interface{}, 0, len(ordered))
	for _, row := range ordered {
		result = append(result, []interface{}{
			row.MissionRewardID, row.PossessionType, row.PossessionID, row.Count,
		})
	}
	return result
}

func prepareMissionRewardReplacement(
	file *memorydb.File,
	replacement *[]MissionRewardInput,
	edits []memorydb.CellEdit,
	validateReferences bool,
) ([][]interface{}, int, int, map[int64]bool, bool, error) {
	if replacement == nil {
		return nil, 0, 0, nil, false, nil
	}
	current, exists, err := file.TableRows(missionRewardTable)
	if err != nil {
		return nil, 0, 0, nil, false, err
	}
	if !exists {
		return nil, 0, 0, nil, false, fmt.Errorf("table %q is absent from the current master data", missionRewardTable)
	}

	currentByID := make(map[int64]MissionRewardInput, len(current))
	for _, row := range current {
		reward, ok := missionRewardInputAt(row)
		if !ok {
			return nil, 0, 0, nil, false, fmt.Errorf("table %q contains a malformed row", missionRewardTable)
		}
		id := int64(reward.MissionRewardID)
		if _, duplicate := currentByID[id]; duplicate {
			return nil, 0, 0, nil, false, fmt.Errorf("table %q contains duplicate RewardId %d", missionRewardTable, id)
		}
		currentByID[id] = reward
	}

	replacementByID := make(map[int64]MissionRewardInput, len(*replacement))
	for _, reward := range *replacement {
		id := int64(reward.MissionRewardID)
		if id <= 0 {
			return nil, 0, 0, nil, false, fmt.Errorf("RewardId must be positive")
		}
		if reward.Count < 0 {
			return nil, 0, 0, nil, false, fmt.Errorf("RewardId %d Count cannot be negative", id)
		}
		if _, duplicate := replacementByID[id]; duplicate {
			return nil, 0, 0, nil, false, fmt.Errorf("duplicate RewardId %d", id)
		}
		replacementByID[id] = reward
	}

	deletedIDs := make(map[int64]bool)
	changedCells, changedRows := 0, 0
	for id, before := range currentByID {
		after, retained := replacementByID[id]
		if !retained {
			deletedIDs[id] = true
			changedCells += 4
			changedRows++
			continue
		}
		rowCells := 0
		if before.PossessionType != after.PossessionType {
			rowCells++
		}
		if before.PossessionID != after.PossessionID {
			rowCells++
		}
		if before.Count != after.Count {
			rowCells++
		}
		if rowCells != 0 {
			changedCells += rowCells
			changedRows++
		}
	}
	for id := range replacementByID {
		if _, exists := currentByID[id]; !exists {
			changedCells += 4
			changedRows++
		}
	}
	if changedRows == 0 {
		return nil, 0, 0, deletedIDs, false, nil
	}
	if validateReferences {
		if err := validateMissionRewardDeletes(file, deletedIDs, edits); err != nil {
			return nil, 0, 0, nil, false, err
		}
	}
	return missionRewardRows(*replacement), changedCells, changedRows, deletedIDs, true, nil
}

func validateMissionRewardDeletes(file *memorydb.File, deletedIDs map[int64]bool, edits []memorydb.CellEdit) error {
	if len(deletedIDs) == 0 {
		return nil
	}
	missions, exists, err := file.TableRows("m_mission")
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("table %q is absent from the current master data", "m_mission")
	}
	effectiveRewardIDs := make([]int64, len(missions))
	for rowIndex, row := range missions {
		rewardID, ok := integerAt(row, 11)
		if !ok {
			return fmt.Errorf("table %q contains a malformed row", "m_mission")
		}
		effectiveRewardIDs[rowIndex] = rewardID
	}
	for _, edit := range edits {
		if edit.Table != "m_mission" || edit.Column != 11 {
			continue
		}
		if edit.Row < 0 || edit.Row >= len(effectiveRewardIDs) {
			return fmt.Errorf("row %d is outside table %q", edit.Row, "m_mission")
		}
		rewardID, err := valueAsInt64(edit.Value)
		if err != nil {
			return fmt.Errorf("m_mission row %d MissionRewardId: %w", edit.Row, err)
		}
		effectiveRewardIDs[edit.Row] = rewardID
	}
	for _, rewardID := range effectiveRewardIDs {
		if deletedIDs[rewardID] {
			return fmt.Errorf("RewardId %d is still referenced by m_mission", rewardID)
		}
	}
	return nil
}
