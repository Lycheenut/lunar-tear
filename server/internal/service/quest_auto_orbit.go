package service

import (
	"log"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"
)

func startAutoOrbit(user *store.UserState, questType model.QuestType, chapterId, questId, maxCount int32, nowMillis int64) {
	if maxCount <= 0 {
		if user.QuestAutoOrbit.MaxAutoOrbitCount > 0 {
			log.Printf("[autoOrbit] clear (start without max): prev questType=%d chapter=%d quest=%d cleared=%d/%d",
				user.QuestAutoOrbit.QuestType, user.QuestAutoOrbit.ChapterId, user.QuestAutoOrbit.QuestId,
				user.QuestAutoOrbit.ClearedAutoOrbitCount, user.QuestAutoOrbit.MaxAutoOrbitCount)
		}
		user.QuestAutoOrbit = store.QuestAutoOrbitState{}
		return
	}
	s := user.QuestAutoOrbit
	if s.MaxAutoOrbitCount > 0 &&
		s.QuestType == int32(questType) && s.ChapterId == chapterId &&
		s.QuestId == questId && s.MaxAutoOrbitCount == maxCount {
		s.LatestVersion = nowMillis
		user.QuestAutoOrbit = s
		log.Printf("[autoOrbit] continue cleared=%d/%d", s.ClearedAutoOrbitCount, s.MaxAutoOrbitCount)
		return
	}
	log.Printf("[autoOrbit] start questType=%d chapter=%d quest=%d max=%d", questType, chapterId, questId, maxCount)
	user.QuestAutoOrbit = store.QuestAutoOrbitState{
		QuestType:         int32(questType),
		ChapterId:         chapterId,
		QuestId:           questId,
		MaxAutoOrbitCount: maxCount,
		LatestVersion:     nowMillis,
	}
}

func finishAutoOrbit(user *store.UserState, isAutoOrbit, isRetired, isAnnihilated bool, questType model.QuestType, chapterId, questId int32, nowMillis int64, drops []questflow.RewardGrant) (endedDrops []store.AutoOrbitDropEntry, loopEnded bool) {
	s := user.QuestAutoOrbit
	if s.MaxAutoOrbitCount <= 0 {
		return nil, false
	}
	if s.QuestType != int32(questType) || s.ChapterId != chapterId || s.QuestId != questId {
		log.Printf("[autoOrbit] finish for other quest, ignored: tracked={qt=%d ch=%d q=%d} got={qt=%d ch=%d q=%d}",
			s.QuestType, s.ChapterId, s.QuestId, int32(questType), chapterId, questId)
		return nil, false
	}
	if !isRetired && !isAnnihilated {
		added := 0
		for _, d := range drops {
			s.AccumulatedDrops = append(s.AccumulatedDrops, store.AutoOrbitDropEntry{
				PossessionType: int32(d.PossessionType),
				PossessionId:   d.PossessionId,
				Count:          d.Count,
				IsAutoSale:     d.IsAutoSale,
				EquipmentData:  d.EquipmentData,
			})
			added++
		}
		s.ClearedAutoOrbitCount++
		log.Printf("[autoOrbit] iter cleared=%d/%d +%d drops (total=%d)",
			s.ClearedAutoOrbitCount, s.MaxAutoOrbitCount, added, len(s.AccumulatedDrops))
	}
	s.LastClearDatetime = nowMillis
	s.LatestVersion = nowMillis
	if !isAutoOrbit || isRetired || isAnnihilated || s.ClearedAutoOrbitCount >= s.MaxAutoOrbitCount {
		log.Printf("[autoOrbit] loop end: cleared=%d/%d total drops=%d (returned in response, accumulator kept)",
			s.ClearedAutoOrbitCount, s.MaxAutoOrbitCount, len(s.AccumulatedDrops))
		user.QuestAutoOrbit = store.QuestAutoOrbitState{AccumulatedDrops: s.AccumulatedDrops}
		return s.AccumulatedDrops, true
	}
	user.QuestAutoOrbit = s
	return nil, false
}

func consumeAutoOrbitRewards(user *store.UserState) []store.AutoOrbitDropEntry {
	drops := user.QuestAutoOrbit.AccumulatedDrops
	log.Printf("[autoOrbit] consume on FinishAutoOrbit: returning %d drops (loop status max=%d cleared=%d)",
		len(drops), user.QuestAutoOrbit.MaxAutoOrbitCount, user.QuestAutoOrbit.ClearedAutoOrbitCount)
	user.QuestAutoOrbit = store.QuestAutoOrbitState{}
	return drops
}
