package userdata

import (
	"sort"
	"sync"
	"sync/atomic"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

var gimmickOrnamentRefs = sync.OnceValue(masterdata.LoadGimmickOrnamentRefs)
var gimmickSequenceChains = sync.OnceValue(masterdata.LoadGimmickSequenceChains)
var hiddenSequenceSet = sync.OnceValue(masterdata.LoadHiddenGimmickSequenceIDs)
var gimmickSequenceRanks = sync.OnceValue(masterdata.LoadGimmickSequenceRanks)
var birdGimmicks = sync.OnceValue(masterdata.LoadBirdGimmickIDs)
var gimmickCatalog atomic.Pointer[masterdata.GimmickCatalog]

func SetGimmickCatalog(catalog *masterdata.GimmickCatalog) {
	gimmickCatalog.Store(catalog)
}

const birdDefaultBaseDatetime int64 = 1577836800000 // 2020-01-01 00:00:00 UTC in ms

var gimmickRecordBuilders = map[string]func(store.UserState) []map[string]any{
	"IUserGimmick":                 sortedGimmickRecords,
	"IUserGimmickOrnamentProgress": sortedGimmickOrnamentProgressRecords,
	"IUserGimmickSequence":         sortedGimmickSequenceRecords,
	"IUserGimmickUnlock":           sortedGimmickUnlockRecords,
}

func init() {
	for table := range gimmickRecordBuilders {
		register(table, func(user store.UserState) string {
			s, _ := utils.EncodeJSONMaps(projectedGimmickRecords(table, user)...)
			return s
		})
	}
}

func projectedGimmickRecords(table string, user store.UserState) []map[string]any {
	records := gimmickRecordBuilders[table]
	if table == "IUserGimmickSequence" {
		records = currentGimmickSequenceRecords
	}
	return visibleGimmickRecords(user, records(user))
}

func visibleGimmickRecords(user store.UserState, records []map[string]any) []map[string]any {
	catalog := gimmickCatalog.Load()
	if catalog == nil {
		return records
	}
	now := gametime.NowMillis()
	visible := records[:0]
	for _, row := range records {
		scheduleId := row["gimmickSequenceScheduleId"].(int32)
		sequenceId := row["gimmickSequenceId"].(int32)
		gimmickId, hasGimmick := row["gimmickId"].(int32)
		if catalog.IsReportSequence(sequenceId) || catalog.GimmickType(gimmickId) == model.GimmickTypeReport {
			// Map visibility follows entry/unlock prerequisites, not the story's
			// own mission completion condition used when collecting its reward.
			if !catalog.SequenceAvailable(&user, scheduleId, sequenceId, now) ||
				(hasGimmick && !catalog.GimmickUnlockAvailable(&user, scheduleId, sequenceId, gimmickId, now)) {
				continue
			}
		}
		visible = append(visible, row)
	}
	return visible
}

// GimmickRefreshDiff also removes unavailable markers cached from older servers,
// even when the saved state has not changed since that unfiltered projection.
func GimmickRefreshDiff(before, after store.UserState) map[string]*pb.DiffData {
	diff := ComputeDelta(&before, &after, ChangedTables(&before, &after))
	for table, records := range gimmickRecordBuilders {
		visible := projectedGimmickRecords(table, after)
		updates, _ := utils.EncodeJSONMaps(visible...)
		diff[table] = &pb.DiffData{
			UpdateRecordsJson: updates,
			DeleteKeysJson:    ComputeDeleteKeys(records(before), visible, keyFieldsForTable(table)),
		}
	}
	return diff
}

func projectActiveChainOrnaments(
	user store.UserState,
	addKey func(seqKey store.GimmickSequenceKey, seqId int32, ref masterdata.GimmickOrnamentRef),
	sizeFn func() int,
	cap int,
) {
	refs := gimmickOrnamentRefs()
	chains := gimmickSequenceChains()
	hiddenSeq := hiddenSequenceSet()

	walkChain := func(seqKey store.GimmickSequenceKey) {
		chain := chains[seqKey.GimmickSequenceId]
		if len(chain) == 0 {
			chain = []int32{seqKey.GimmickSequenceId}
		}
		for _, seqId := range chain {
			for _, ref := range refs[seqId] {
				addKey(seqKey, seqId, ref)
			}
		}
	}

	var nonHidden []store.GimmickSequenceKey
	for seqKey := range user.Gimmick.Sequences {
		if hiddenSeq[seqKey.GimmickSequenceId] {
			walkChain(seqKey)
		} else {
			nonHidden = append(nonHidden, seqKey)
		}
	}
	for _, seqKey := range nonHidden {
		if sizeFn() >= cap {
			break
		}
		walkChain(seqKey)
	}
}

func sortedGimmickRecords(user store.UserState) []map[string]any {

	keySet := make(map[store.GimmickKey]struct{})
	// Real progress rows (genuine user data) — always kept.
	for key := range user.Gimmick.Progress {
		keySet[key] = struct{}{}
	}
	projectActiveChainOrnaments(user,
		func(seqKey store.GimmickSequenceKey, seqId int32, ref masterdata.GimmickOrnamentRef) {
			keySet[store.GimmickKey{
				GimmickSequenceScheduleId: seqKey.GimmickSequenceScheduleId,
				GimmickSequenceId:         seqId,
				GimmickId:                 ref.GimmickId,
			}] = struct{}{}
		},
		func() int { return len(keySet) },
		masterdata.MaxUserGimmickRows,
	)

	keys := make([]store.GimmickKey, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return compareGimmickKey(keys[i], keys[j]) < 0
	})

	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		isGimmickCleared := false
		startDatetime := user.GameStartDatetime
		latestVersion := user.GameStartDatetime
		if row, ok := user.Gimmick.Progress[key]; ok {
			isGimmickCleared = row.IsGimmickCleared
			startDatetime = row.StartDatetime
			latestVersion = row.LatestVersion
		}
		records = append(records, map[string]any{
			"userId":                    user.UserId,
			"gimmickSequenceScheduleId": key.GimmickSequenceScheduleId,
			"gimmickSequenceId":         key.GimmickSequenceId,
			"gimmickId":                 key.GimmickId,
			"isGimmickCleared":          isGimmickCleared,
			"startDatetime":             startDatetime,
			"latestVersion":             latestVersion,
		})
	}
	return records
}

func sortedGimmickOrnamentProgressRecords(user store.UserState) []map[string]any {

	keySet := make(map[store.GimmickOrnamentKey]struct{})
	// Real progress rows (genuine user data) — always kept.
	for key := range user.Gimmick.OrnamentProgress {
		keySet[key] = struct{}{}
	}
	projectActiveChainOrnaments(user,
		func(seqKey store.GimmickSequenceKey, seqId int32, ref masterdata.GimmickOrnamentRef) {
			keySet[store.GimmickOrnamentKey{
				GimmickSequenceScheduleId: seqKey.GimmickSequenceScheduleId,
				GimmickSequenceId:         seqId,
				GimmickId:                 ref.GimmickId,
				GimmickOrnamentIndex:      ref.OrnamentIndex,
			}] = struct{}{}
		},
		func() int { return len(keySet) },
		masterdata.MaxUserGimmickRows,
	)

	keys := make([]store.GimmickOrnamentKey, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return compareGimmickOrnamentKey(keys[i], keys[j]) < 0
	})

	birdG := birdGimmicks()
	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		progressValueBit := int32(0)
		baseDatetime := user.GameStartDatetime
		latestVersion := user.GameStartDatetime
		if row, ok := user.Gimmick.OrnamentProgress[key]; ok {
			progressValueBit = row.ProgressValueBit
			baseDatetime = row.BaseDatetime
			latestVersion = row.LatestVersion
		} else if birdG[key.GimmickId] {
			baseDatetime = birdDefaultBaseDatetime
		}
		records = append(records, map[string]any{
			"userId":                    user.UserId,
			"gimmickSequenceScheduleId": key.GimmickSequenceScheduleId,
			"gimmickSequenceId":         key.GimmickSequenceId,
			"gimmickId":                 key.GimmickId,
			"gimmickOrnamentIndex":      key.GimmickOrnamentIndex,
			"progressValueBit":          progressValueBit,
			"baseDatetime":              baseDatetime,
			"latestVersion":             latestVersion,
		})
	}
	return records
}

func sortedGimmickSequenceRecords(user store.UserState) []map[string]any {
	keys := make([]store.GimmickSequenceKey, 0, len(user.Gimmick.Sequences))
	for key := range user.Gimmick.Sequences {
		keys = append(keys, key)
	}
	return gimmickSequenceRecords(user, keys)
}

func currentGimmickSequenceRecords(user store.UserState) []map[string]any {
	// The client keys this table by (userId, scheduleId), not sequenceId.
	// Keep history in the save, but expose only one current cursor per schedule.
	bySchedule := make(map[int32]store.GimmickSequenceKey)
	for key, row := range user.Gimmick.Sequences {
		previous, exists := bySchedule[key.GimmickSequenceScheduleId]
		previousRow := user.Gimmick.Sequences[previous]
		if !exists || row.LatestVersion > previousRow.LatestVersion ||
			(row.LatestVersion == previousRow.LatestVersion && key.GimmickSequenceId > previous.GimmickSequenceId) {
			bySchedule[key.GimmickSequenceScheduleId] = key
		}
	}
	if catalog := gimmickCatalog.Load(); catalog != nil {
		// ActiveScheduleKeys follows master-data chain order. Advancing the
		// cursor also works before the next sequence has a persisted row.
		for _, key := range catalog.ActiveScheduleKeys(user, gametime.NowMillis()) {
			if _, initialized := bySchedule[key.GimmickSequenceScheduleId]; initialized {
				bySchedule[key.GimmickSequenceScheduleId] = key
			}
		}
	}
	keys := make([]store.GimmickSequenceKey, 0, len(bySchedule))
	for _, key := range bySchedule {
		keys = append(keys, key)
	}
	return gimmickSequenceRecords(user, keys)
}

func gimmickSequenceRecords(user store.UserState, keys []store.GimmickSequenceKey) []map[string]any {
	ranks := gimmickSequenceRanks()
	sort.Slice(keys, func(i, j int) bool {
		ri, rj := ranks[keys[i].GimmickSequenceId], ranks[keys[j].GimmickSequenceId]
		if ri != rj {
			return ri < rj
		}
		if keys[i].GimmickSequenceScheduleId != keys[j].GimmickSequenceScheduleId {
			return keys[i].GimmickSequenceScheduleId < keys[j].GimmickSequenceScheduleId
		}
		return keys[i].GimmickSequenceId < keys[j].GimmickSequenceId
	})
	if len(keys) > masterdata.MaxUserGimmickRows {
		keys = keys[:masterdata.MaxUserGimmickRows]
	}

	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.Gimmick.Sequences[key]
		records = append(records, map[string]any{
			"userId":                    user.UserId,
			"gimmickSequenceScheduleId": key.GimmickSequenceScheduleId,
			"gimmickSequenceId":         key.GimmickSequenceId,
			"isGimmickSequenceCleared":  row.IsGimmickSequenceCleared,
			"clearDatetime":             row.ClearDatetime,
			"latestVersion":             row.LatestVersion,
		})
	}
	return records
}

func sortedGimmickUnlockRecords(user store.UserState) []map[string]any {
	keys := make([]store.GimmickKey, 0, len(user.Gimmick.Unlocks))
	for key := range user.Gimmick.Unlocks {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return compareGimmickKey(keys[i], keys[j]) < 0
	})

	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.Gimmick.Unlocks[key]
		records = append(records, map[string]any{
			"userId":                    user.UserId,
			"gimmickSequenceScheduleId": row.Key.GimmickSequenceScheduleId,
			"gimmickSequenceId":         row.Key.GimmickSequenceId,
			"gimmickId":                 row.Key.GimmickId,
			"isUnlocked":                row.IsUnlocked,
			"latestVersion":             row.LatestVersion,
		})
	}
	return records
}
