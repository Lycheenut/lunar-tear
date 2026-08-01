package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/migrations"
)

func TestUserStateRoundTripForContentState(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := New(db, nil)
	userId, err := repo.CreateUser("mechanisms", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.UpdateUser(userId, func(user *store.UserState) {
		user.CostumeLevelBonusReleaseStatuses[10] = store.CostumeLevelBonusReleaseStatusState{CostumeId: 10, LastReleasedBonusLevel: 60, ConfirmedBonusLevel: 60, LatestVersion: 1}
		user.CostumeLotteryEffectAbilities[store.CostumeLotteryEffectKey{UserCostumeUuid: "costume", SlotNumber: 1}] = store.CostumeLotteryEffectAbilityState{UserCostumeUuid: "costume", SlotNumber: 1, AbilityId: 20, AbilityLevel: 2, LatestVersion: 2}
		user.CostumeLotteryEffectStatusUps[store.CostumeLotteryEffectStatusKey{UserCostumeUuid: "costume", StatusCalculationType: model.StatusCalculationTypeAdd}] = store.CostumeLotteryEffectStatusUpState{UserCostumeUuid: "costume", StatusCalculationType: model.StatusCalculationTypeAdd, Attack: 30, LatestVersion: 3}
		user.DeckLimitContentRestricted["restricted"] = store.DeckLimitContentRestrictedState{DeckRestrictedUuid: "restricted", EventQuestChapterId: 30, QuestId: 31, PossessionType: 1, TargetUuid: "costume", LatestVersion: 3}
		user.CageOrnamentAccesses[40] = store.CageOrnamentAccessState{CageOrnamentId: 40, FirstAccessDatetime: 4, LatestAccessDatetime: 5, LatestVersion: 5}
		user.QuestReplayFlowRewards[50] = store.QuestReplayFlowRewardState{QuestReplayFlowRewardGroupId: 50, RewardReceiveDatetime: 6, LatestVersion: 6}
		user.MissionPassPoints[60] = store.MissionPassPointState{MissionPassId: 60, Point: 100, LatestVersion: 7}
		user.Battle.MissionDetail = store.BattleMissionDetailState{IsValid: true, CriticalCount: 8}
		user.WebviewPanelMissions[70] = store.WebviewPanelMissionState{WebviewPanelMissionPageId: 70, RewardReceiveDatetime: 8, LatestVersion: 8}
	})
	if err != nil {
		t.Fatal(err)
	}
	user, err := repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if user.CostumeLevelBonusReleaseStatuses[10].ConfirmedBonusLevel != 60 {
		t.Fatal("costume level bonus was not persisted")
	}
	if user.CostumeLotteryEffectAbilities[store.CostumeLotteryEffectKey{UserCostumeUuid: "costume", SlotNumber: 1}].AbilityId != 20 {
		t.Fatal("costume lottery ability was not persisted")
	}
	if user.CostumeLotteryEffectStatusUps[store.CostumeLotteryEffectStatusKey{UserCostumeUuid: "costume", StatusCalculationType: model.StatusCalculationTypeAdd}].Attack != 30 {
		t.Fatal("costume lottery status was not persisted")
	}
	if user.DeckLimitContentRestricted["restricted"].TargetUuid != "costume" {
		t.Fatal("limit content restriction was not persisted")
	}
	if user.CageOrnamentAccesses[40].LatestAccessDatetime != 5 {
		t.Fatal("cage ornament access was not persisted")
	}
	if user.QuestReplayFlowRewards[50].RewardReceiveDatetime != 6 || user.MissionPassPoints[60].Point != 100 || user.Battle.MissionDetail.CriticalCount != 8 || user.WebviewPanelMissions[70].RewardReceiveDatetime != 8 {
		t.Fatal("mechanism state was not persisted")
	}
}

func TestLoadUserRejectsCorruptMechanismState(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := New(db, nil)
	userId, err := repo.CreateUser("corrupt-mechanism", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE user_mechanism_state SET state_json=? WHERE user_id=?`, `{`, userId); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.LoadUser(userId); err == nil {
		t.Fatal("corrupt mechanism state was silently ignored")
	}
}

func TestUserStatePersistenceErrorRollsBack(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := New(db, nil)
	userId, err := repo.CreateUser("mechanism-errors", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	before, err := repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TABLE user_costume_lottery_effect_abilities`); err != nil {
		t.Fatal(err)
	}
	_, err = repo.UpdateUser(userId, func(user *store.UserState) {
		user.Status.Level = before.Status.Level + 1
		key := store.CostumeLotteryEffectKey{UserCostumeUuid: "costume", SlotNumber: 1}
		user.CostumeLotteryEffectAbilities[key] = store.CostumeLotteryEffectAbilityState{UserCostumeUuid: "costume", SlotNumber: 1, AbilityId: 20}
	})
	if err == nil {
		t.Fatal("persistence error was ignored")
	}
	after, err := repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status.Level != before.Status.Level {
		t.Fatal("failed update was partially committed")
	}
}
