package sqlite

import (
	"context"
	"fmt"
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
		user.CharacterViewerFields[1] = store.CharacterViewerFieldState{CharacterViewerFieldId: 1, ReleaseDatetime: 11, LatestVersion: 12}
		user.BeginnerCampaign = store.UserBeginnerCampaignState{BeginnerCampaignId: 1, CampaignRegisterDatetime: 100, LatestVersion: 101}
		user.ComebackCampaign = store.UserComebackCampaignState{ComebackCampaignId: 2, ComebackDatetime: 200, LatestVersion: 201}
		user.LoginBonuses[1] = store.UserLoginBonusState{LoginBonusId: 1, CurrentPageNumber: 2, CurrentStampNumber: 3, LatestVersion: 301}
		user.LoginBonuses[91] = store.UserLoginBonusState{LoginBonusId: 91, CurrentPageNumber: 1, CurrentStampNumber: 1, LatestVersion: 302}
		user.CostumeLevelBonusReleaseStatuses[10] = store.CostumeLevelBonusReleaseStatusState{CostumeId: 10, LastReleasedBonusLevel: 60, ConfirmedBonusLevel: 60, LatestVersion: 1}
		user.CostumeLotteryEffectAbilities[store.CostumeLotteryEffectKey{UserCostumeUuid: "costume", SlotNumber: 1}] = store.CostumeLotteryEffectAbilityState{UserCostumeUuid: "costume", SlotNumber: 1, AbilityId: 20, AbilityLevel: 2, LatestVersion: 2}
		user.CostumeLotteryEffectStatusUps[store.CostumeLotteryEffectStatusKey{UserCostumeUuid: "costume", StatusCalculationType: model.StatusCalculationTypeAdd}] = store.CostumeLotteryEffectStatusUpState{UserCostumeUuid: "costume", StatusCalculationType: model.StatusCalculationTypeAdd, Attack: 30, LatestVersion: 3}
		user.DeckLimitContentRestricted["restricted"] = store.DeckLimitContentRestrictedState{DeckRestrictedUuid: "restricted", EventQuestChapterId: 30, QuestId: 31, PossessionType: 1, TargetUuid: "costume", LatestVersion: 3}
		user.CageOrnamentAccesses[40] = store.CageOrnamentAccessState{CageOrnamentId: 40, FirstAccessDatetime: 4, LatestAccessDatetime: 5, LatestVersion: 5}
		user.QuestReplayFlowRewards[50] = store.QuestReplayFlowRewardState{QuestReplayFlowRewardGroupId: 50, RewardReceiveDatetime: 6, LatestVersion: 6}
		choiceKey := store.QuestSceneChoiceKey{QuestSceneId: 51, QuestFlowType: 2}
		user.QuestSceneChoices[choiceKey] = store.QuestSceneChoiceState{QuestSceneId: 51, QuestFlowType: 2, ChoiceNumber: 3, ChoiceDatetime: 7, LatestVersion: 7}
		historyKey := store.QuestSceneChoiceHistoryKey{QuestSceneId: 51, QuestFlowType: 2, ChoiceNumber: 3}
		user.QuestSceneChoiceHistory[historyKey] = user.QuestSceneChoices[choiceKey]
		user.EventQuestDailyRewards[52] = store.EventQuestDailyRewardState{EventQuestDailyGroupId: 52, RewardReceiveDatetime: 8, LatestVersion: 8}
		user.MissionPassPoints[60] = store.MissionPassPointState{MissionPassId: 60, Point: 100, LatestVersion: 7}
		rewardKey := store.MissionPassRewardKey{MissionPassId: 60, Level: 2, IsPremium: true}
		user.MissionPassRewards[rewardKey] = store.MissionPassRewardState{MissionPassId: 60, Level: 2, IsPremium: true, RewardReceiveDatetime: 9, LatestVersion: 9}
		user.MissionPassRemaining[60] = store.MissionPassRemainingState{MissionPassId: 60, RewardReceived: true, RewardReceiveDatetime: 10, LatestVersion: 10}
		user.Battle.MissionDetail = store.BattleMissionDetailState{IsValid: true, CriticalCount: 8, CostumeResultCount: 1, CostumeResults: [3]store.CostumeBattleResultState{{IsAlive: true, MaxHp: 100, RemainingHp: 40}}}
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
	if user.CharacterViewerFields[1].ReleaseDatetime != 11 || user.CharacterViewerFields[1].LatestVersion != 12 {
		t.Fatal("character viewer field was not persisted")
	}
	if user.BeginnerCampaign.BeginnerCampaignId != 1 || user.ComebackCampaign.ComebackCampaignId != 2 ||
		len(user.LoginBonuses) != 2 || user.LoginBonuses[1].CurrentStampNumber != 3 || user.LoginBonuses[91].CurrentStampNumber != 1 {
		t.Fatal("campaign or multi-login-bonus state was not persisted")
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
	choiceKey := store.QuestSceneChoiceKey{QuestSceneId: 51, QuestFlowType: 2}
	historyKey := store.QuestSceneChoiceHistoryKey{QuestSceneId: 51, QuestFlowType: 2, ChoiceNumber: 3}
	rewardKey := store.MissionPassRewardKey{MissionPassId: 60, Level: 2, IsPremium: true}
	if user.QuestReplayFlowRewards[50].RewardReceiveDatetime != 6 ||
		user.QuestSceneChoices[choiceKey].ChoiceNumber != 3 || user.QuestSceneChoiceHistory[historyKey].ChoiceDatetime != 7 ||
		user.EventQuestDailyRewards[52].RewardReceiveDatetime != 8 || user.MissionPassPoints[60].Point != 100 ||
		user.MissionPassRewards[rewardKey].RewardReceiveDatetime != 9 || !user.MissionPassRemaining[60].RewardReceived ||
		user.Battle.MissionDetail.CriticalCount != 8 || !user.Battle.MissionDetail.CostumeResults[0].IsAlive ||
		user.WebviewPanelMissions[70].RewardReceiveDatetime != 8 {
		t.Fatal("mechanism state was not persisted")
	}
}

func TestMechanismTablesAreIndependent(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := New(db, nil)
	userId, err := repo.CreateUser("independent-mechanisms", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
		user.QuestReplayFlowRewards[1] = store.QuestReplayFlowRewardState{QuestReplayFlowRewardGroupId: 1, RewardReceiveDatetime: 10}
		user.MissionPassPoints[2] = store.MissionPassPointState{MissionPassId: 2, Point: 20}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM user_quest_replay_flow_rewards WHERE user_id=?`, userId); err != nil {
		t.Fatal(err)
	}
	user, err := repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if len(user.QuestReplayFlowRewards) != 0 || user.MissionPassPoints[2].Point != 20 {
		t.Fatal("one mechanism table affected an unrelated mechanism")
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

func TestImportUserClearsEveryOwnedTable(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := New(db, nil)
	userID, err := repo.CreateUser("import-cleanup", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := repo.LoadUser(userID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_triple_decks (user_id, deck_type, user_deck_number) VALUES (?,?,?)`, userID, 1, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_big_hunt_costume_battle_infos (user_id, wave_index, sort_order) VALUES (?,?,?)`, userID, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := repo.ImportUser(&snapshot); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"user_triple_decks", "user_big_hunt_costume_battle_infos"} {
		var count int
		if err := db.QueryRow(fmt.Sprintf(`SELECT count(*) FROM %s WHERE user_id=?`, table), userID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("stale rows remained in %s after import", table)
		}
	}
}
