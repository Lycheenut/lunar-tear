package service

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func TestFinishWaveCheckpointAndMissionDetailSurviveReloadAndNextWaveStart(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	repo := sqlite.New(db, nil)
	userID, err := repo.CreateUser("battle-resume", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	server := NewBattleServiceServer(repo, nil)
	checkpoint := []byte{0x10, 0x20, 0x30, 0x40}
	if _, err := server.FinishWave(context.Background(), &pb.FinishWaveRequest{BattleBinary: checkpoint, BattleDetail: &pb.BattleDetail{
		CharacterDeathCount: 1, MaxDamage: 500000,
		PlayerCostumeActiveSkillUsedCount: 3, PlayerWeaponActiveSkillUsedCount: 6, PlayerCompanionSkillUsedCount: 1,
		CriticalCount: 5, ComboCount: 8, ComboMaxDamage: 1000000, TotalRecoverPoint: 100,
		CostumeBattleInfo: []*pb.CostumeBattleInfo{{DeckCharacterNumber: 1, IsAlive: true, MaxHp: 100, RemainingHp: 75}},
	}}); err != nil {
		t.Fatal(err)
	}

	assertCheckpoint := func(stage string) {
		t.Helper()
		user, err := repo.LoadUser(userID)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(user.BattleBinary, checkpoint) {
			t.Fatalf("%s checkpoint = %x, want %x", stage, user.BattleBinary, checkpoint)
		}
		if user.Battle.MissionDetail.CostumeSkillUseCount != 3 {
			t.Fatalf("%s lost the previous wave's skill count: %+v", stage, user.Battle.MissionDetail)
		}
	}
	assertCheckpoint("finished wave")

	if _, err := server.StartWave(context.Background(), &pb.StartWaveRequest{}); err != nil {
		t.Fatal(err)
	}
	assertCheckpoint("next wave started")
	if _, err := server.FinishWave(context.Background(), &pb.FinishWaveRequest{BattleDetail: &pb.BattleDetail{
		MaxDamage: 100000, PlayerCostumeActiveSkillUsedCount: 2, PlayerWeaponActiveSkillUsedCount: 4, PlayerCompanionSkillUsedCount: 1,
		CriticalCount: 2, ComboCount: 3, ComboMaxDamage: 200000,
		CostumeBattleInfo: []*pb.CostumeBattleInfo{{DeckCharacterNumber: 1, IsAlive: true, MaxHp: 100, RemainingHp: 90}},
	}}); err != nil {
		t.Fatal(err)
	}
	user, err := repo.LoadUser(userID)
	if err != nil {
		t.Fatal(err)
	}
	want := store.BattleMissionDetailState{
		IsValid: true, CharacterDeathCount: 1, MaxDamage: 500000,
		CostumeSkillUseCount: 5, WeaponSkillUseCount: 10, CompanionSkillUseCount: 2,
		CriticalCount: 7, ComboCount: 8, ComboMaxDamage: 1000000, TotalRecoverPoint: 100,
		CostumeResultCount: 1, CostumeResults: [3]store.CostumeBattleResultState{{IsAlive: true, MaxHp: 100, RemainingHp: 90}},
	}
	if user.Battle.MissionDetail != want {
		t.Fatalf("accumulated mission detail = %+v, want %+v", user.Battle.MissionDetail, want)
	}
}

func TestQuestSkillLimitsCountEveryWave(t *testing.T) {
	for _, skill := range []struct {
		name      string
		condition model.QuestMissionConditionType
		setCount  func(*pb.BattleDetail, int32)
	}{
		{"costume", model.QuestMissionConditionTypeLessThanOrEqualXCostumeSkillUseCount, func(d *pb.BattleDetail, n int32) { d.PlayerCostumeActiveSkillUsedCount = n }},
		{"weapon", model.QuestMissionConditionTypeLessThanOrEqualXWeaponSkillUseCount, func(d *pb.BattleDetail, n int32) { d.PlayerWeaponActiveSkillUsedCount = n }},
		{"companion", model.QuestMissionConditionTypeLessThanOrEqualXCompanionSkillUseCount, func(d *pb.BattleDetail, n int32) { d.PlayerCompanionSkillUsedCount = n }},
	} {
		t.Run(skill.name, func(t *testing.T) {
			for _, tt := range []struct {
				name      string
				counts    []int32 // -1 means the wave's battle detail is missing.
				duplicate bool
				resume    bool
				active    bool
				wantClear bool
			}{
				{name: "single wave at limit", counts: []int32{10}, wantClear: true},
				{name: "single wave over limit", counts: []int32{11}},
				{name: "three waves at limit", counts: []int32{4, 4, 2}, wantClear: true},
				{name: "three waves over limit", counts: []int32{4, 4, 3}},
				{name: "over limit before final wave", counts: []int32{11, 0, 0}},
				{name: "missing first wave", counts: []int32{-1, 0, 0}},
				{name: "missing middle wave", counts: []int32{4, -1, 0}},
				{name: "missing final wave", counts: []int32{4, 4, -1}},
				{name: "duplicate finish", counts: []int32{4, 4, 2}, duplicate: true, wantClear: true},
				{name: "resume between waves", counts: []int32{4, 4, 3}, resume: true},
				{name: "next wave still active", counts: []int32{4}, active: true},
			} {
				t.Run(tt.name, func(t *testing.T) {
					db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
					if err != nil {
						t.Fatal(err)
					}
					defer db.Close()
					if err := migrations.Up(context.Background(), db); err != nil {
						t.Fatal(err)
					}
					repo := sqlite.New(db, nil)
					userID, err := repo.CreateUser("skill-limit", model.ClientPlatform{})
					if err != nil {
						t.Fatal(err)
					}
					server := NewBattleServiceServer(repo, nil)
					// Fate Board: Shadow Quest 46 (240096) requires at most 10
					// costume skills across three waves. Exercise all three skill types.
					const questID, missionID int32 = 240096, 240307
					h := &questflow.QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
						QuestById:           map[int32]masterdata.EntityMQuest{questID: {QuestId: questID}},
						MissionIdsByQuestId: map[int32][]int32{questID: {missionID}},
						MissionById: map[int32]masterdata.EntityMQuestMission{missionID: {
							QuestMissionId: missionID, QuestMissionConditionType: int32(skill.condition), ConditionValue: 10,
						}},
					}, Config: &masterdata.GameConfig{}, Granter: &store.PossessionGranter{}}
					if _, err := repo.UpdateUser(userID, func(user *store.UserState) {
						user.Quests[questID] = store.UserQuestState{QuestId: questID, QuestStateType: model.UserQuestStateTypeActive, LatestStartDatetime: 1}
					}); err != nil {
						t.Fatal(err)
					}
					for i, count := range tt.counts {
						if tt.resume && i > 0 {
							if _, err := repo.UpdateUser(userID, func(user *store.UserState) {
								h.HandleEventQuestRestart(user, 802, questID, 2)
							}); err != nil {
								t.Fatal(err)
							}
							server = NewBattleServiceServer(sqlite.New(db, nil), nil)
						}
						if _, err := server.StartWave(context.Background(), &pb.StartWaveRequest{}); err != nil {
							t.Fatal(err)
						}
						req := &pb.FinishWaveRequest{BattleBinary: []byte{byte(i + 1)}}
						if count >= 0 {
							req.BattleDetail = &pb.BattleDetail{}
							skill.setCount(req.BattleDetail, count)
						}
						if _, err := server.FinishWave(context.Background(), req); err != nil {
							t.Fatal(err)
						}
						if tt.duplicate {
							if _, err := server.FinishWave(context.Background(), req); err != nil {
								t.Fatal(err)
							}
						}
					}
					if tt.active {
						if _, err := server.StartWave(context.Background(), &pb.StartWaveRequest{}); err != nil {
							t.Fatal(err)
						}
					}
					user, err := repo.LoadUser(userID)
					if err != nil {
						t.Fatal(err)
					}
					detail := user.Battle.MissionDetail
					h.HandleEventQuestFinish(&user, 802, questID, false, false, user.Battle.LastFinishedAt+1)
					if got := user.QuestMissions[store.QuestMissionKey{QuestId: questID, QuestMissionId: missionID}].IsClear; got != tt.wantClear {
						t.Errorf("mission cleared = %v, want %v; wave counts = %v, stored detail = %+v", got, tt.wantClear, tt.counts, detail)
					}
				})
			}
		})
	}
}
