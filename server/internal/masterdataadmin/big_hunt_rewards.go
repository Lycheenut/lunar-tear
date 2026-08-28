package masterdataadmin

import (
	"fmt"

	"lunar-tear/server/internal/masterdata/memorydb"
)

const (
	bigHuntBossQuestTable            = "m_big_hunt_boss_quest"
	bigHuntRewardGroupTable          = "m_big_hunt_reward_group"
	bigHuntScoreRewardGroupTable     = "m_big_hunt_score_reward_group"
	bigHuntScoreRewardScheduleTable  = "m_big_hunt_score_reward_group_schedule"
	bigHuntWeeklyRewardScheduleTable = "m_big_hunt_weekly_attribute_score_reward_group_schedule"
)

var bigHuntRewardTables = map[string]bool{
	bigHuntBossQuestTable:            true,
	bigHuntRewardGroupTable:          true,
	bigHuntScoreRewardGroupTable:     true,
	bigHuntScoreRewardScheduleTable:  true,
	bigHuntWeeklyRewardScheduleTable: true,
}

func changesBigHuntRewards(changes []Change) bool {
	for _, change := range changes {
		if bigHuntRewardTables[change.Table] {
			return true
		}
	}
	return false
}

func validateBigHuntRewardConfig(file *memorydb.File) error {
	bossQuests, err := requiredBigHuntRows(file, bigHuntBossQuestTable)
	if err != nil {
		return err
	}
	rewardGroups, err := requiredBigHuntRows(file, bigHuntRewardGroupTable)
	if err != nil {
		return err
	}
	scoreGroups, err := requiredBigHuntRows(file, bigHuntScoreRewardGroupTable)
	if err != nil {
		return err
	}
	scoreSchedules, err := requiredBigHuntRows(file, bigHuntScoreRewardScheduleTable)
	if err != nil {
		return err
	}
	weeklySchedules, err := requiredBigHuntRows(file, bigHuntWeeklyRewardScheduleTable)
	if err != nil {
		return err
	}

	rewardGroupIDs := make(map[int64]bool)
	for rowIndex, row := range rewardGroups {
		rewardGroupID, idOK := integerAt(row, 0)
		count, countOK := integerAt(row, 4)
		if !idOK || !countOK {
			return fmt.Errorf("%s row %d is malformed", bigHuntRewardGroupTable, rowIndex)
		}
		if count <= 0 {
			return fmt.Errorf("%s row %d field Count: must be greater than zero", bigHuntRewardGroupTable, rowIndex)
		}
		rewardGroupIDs[rewardGroupID] = true
	}

	scoreGroupIDs := make(map[int64]bool)
	for rowIndex, row := range scoreGroups {
		scoreGroupID, idOK := integerAt(row, 0)
		necessaryScore, scoreOK := integerAt(row, 1)
		rewardGroupID, rewardOK := integerAt(row, 2)
		if !idOK || !scoreOK || !rewardOK {
			return fmt.Errorf("%s row %d is malformed", bigHuntScoreRewardGroupTable, rowIndex)
		}
		if necessaryScore < 0 {
			return fmt.Errorf("%s row %d field NecessaryScore: cannot be negative", bigHuntScoreRewardGroupTable, rowIndex)
		}
		if !rewardGroupIDs[rewardGroupID] {
			return fmt.Errorf("%s row %d references unknown BigHuntRewardGroupId %d", bigHuntScoreRewardGroupTable, rowIndex, rewardGroupID)
		}
		scoreGroupIDs[scoreGroupID] = true
	}

	scoreScheduleIDs := make(map[int64]bool)
	dailyStarts := make(map[[2]int64]int)
	for rowIndex, row := range scoreSchedules {
		scheduleID, scheduleOK := integerAt(row, 0)
		scoreGroupID, groupOK := integerAt(row, 2)
		start, startOK := integerAt(row, 3)
		if !scheduleOK || !groupOK || !startOK {
			return fmt.Errorf("%s row %d is malformed", bigHuntScoreRewardScheduleTable, rowIndex)
		}
		if !scoreGroupIDs[scoreGroupID] {
			return fmt.Errorf("%s row %d references unknown BigHuntScoreRewardGroupId %d", bigHuntScoreRewardScheduleTable, rowIndex, scoreGroupID)
		}
		key := [2]int64{scheduleID, start}
		if previous, duplicate := dailyStarts[key]; duplicate {
			return fmt.Errorf("%s rows %d and %d have the same schedule and StartDatetime", bigHuntScoreRewardScheduleTable, previous, rowIndex)
		}
		dailyStarts[key] = rowIndex
		scoreScheduleIDs[scheduleID] = true
	}

	weeklyStarts := make(map[[2]int64]int)
	for rowIndex, row := range weeklySchedules {
		attributeType, attributeOK := integerAt(row, 1)
		scoreGroupID, groupOK := integerAt(row, 3)
		start, startOK := integerAt(row, 4)
		if !attributeOK || !groupOK || !startOK {
			return fmt.Errorf("%s row %d is malformed", bigHuntWeeklyRewardScheduleTable, rowIndex)
		}
		if !scoreGroupIDs[scoreGroupID] {
			return fmt.Errorf("%s row %d references unknown BigHuntScoreRewardGroupId %d", bigHuntWeeklyRewardScheduleTable, rowIndex, scoreGroupID)
		}
		key := [2]int64{attributeType, start}
		if previous, duplicate := weeklyStarts[key]; duplicate {
			return fmt.Errorf("%s rows %d and %d have the same attribute and StartDatetime", bigHuntWeeklyRewardScheduleTable, previous, rowIndex)
		}
		weeklyStarts[key] = rowIndex
	}

	for rowIndex, row := range bossQuests {
		scheduleID, ok := integerAt(row, 4)
		if !ok {
			return fmt.Errorf("%s row %d is malformed", bigHuntBossQuestTable, rowIndex)
		}
		if !scoreScheduleIDs[scheduleID] {
			return fmt.Errorf("%s row %d references unknown BigHuntScoreRewardGroupScheduleId %d", bigHuntBossQuestTable, rowIndex, scheduleID)
		}
	}
	return nil
}

func requiredBigHuntRows(file *memorydb.File, table string) ([][]interface{}, error) {
	rows, exists, err := file.TableRows(table)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("required big hunt reward table %q is absent", table)
	}
	return rows, nil
}
