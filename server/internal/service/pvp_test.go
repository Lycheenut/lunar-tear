package service

import (
	"testing"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestRecordPvpMissionEventsTracksAndResetsStreaks(t *testing.T) {
	user := &store.UserState{}
	recordPvpMissionEvents(user, true)
	wantVictory := []model.MissionClearConditionType{
		model.MissionClearConditionTypePvpFinishByCount,
		model.MissionClearConditionTypePvpFinishByWinCount,
		model.MissionClearConditionTypePvpFinishByWinStreakCount,
		model.MissionClearConditionTypePvpFinishByWinStreakCountFromUnlock,
	}
	if len(user.PendingMissionEvents) != len(wantVictory) {
		t.Fatalf("victory events = %+v", user.PendingMissionEvents)
	}
	for i, conditionType := range wantVictory {
		if user.PendingMissionEvents[i].ConditionType != int32(conditionType) || user.PendingMissionEvents[i].Reset {
			t.Fatalf("victory event %d = %+v", i, user.PendingMissionEvents[i])
		}
	}

	user.PendingMissionEvents = nil
	recordPvpMissionEvents(user, false)
	if len(user.PendingMissionEvents) != 3 || !user.PendingMissionEvents[1].Reset || !user.PendingMissionEvents[2].Reset {
		t.Fatalf("loss events = %+v", user.PendingMissionEvents)
	}
}
