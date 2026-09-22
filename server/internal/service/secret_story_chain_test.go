package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func TestSecretStoryFiveSequenceGatesAndPersistence(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := migrations.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	userId, err := repo.CreateUser("story-chain", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	masterData, err := os.ReadFile(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e"))
	if err != nil {
		t.Fatal(err)
	}
	masterDataPath := filepath.Join(t.TempDir(), "master-data.bin.e")
	if err := os.WriteFile(masterDataPath, masterData, 0o600); err != nil {
		t.Fatal(err)
	}
	holder, err := runtime.NewHolder(masterDataPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
		for _, conditionId := range []int32{4104, 4204, 4304} {
			questId, ok := holder.Get().ConditionResolver.RequiredQuestId(conditionId)
			if !ok {
				t.Fatalf("missing chapter condition %d", conditionId)
			}
			user.Quests[questId] = store.UserQuestState{QuestId: questId, QuestStateType: model.UserQuestStateTypeCleared}
		}
		// Complete Dimos, Griff, Marie and Yurie's own mission requirements.
		// The latter two stories in each pair must still wait for their predecessor.
		for _, missionId := range []int32{500021, 500023, 500072, 500075} {
			user.Missions[missionId] = store.UserMissionState{MissionId: missionId, MissionProgressStatusType: int32(model.MissionProgressStatusTypeClear)}
		}
	}); err != nil {
		t.Fatal(err)
	}
	server := NewGimmickServiceServer(repo, repo, holder)
	progress := func(key *pb.GimmickKey) (*pb.UpdateGimmickProgressResponse, error) {
		return server.UpdateGimmickProgress(ctx, &pb.UpdateGimmickProgressRequest{
			GimmickSequenceScheduleId: key.GimmickSequenceScheduleId,
			GimmickSequenceId:         key.GimmickSequenceId,
			GimmickId:                 key.GimmickId,
			GimmickOrnamentIndex:      1,
			ProgressValueBit:          1,
		})
	}
	chains := []struct {
		name        string
		first, next *pb.GimmickKey
		itemId      int32
	}{
		{"Dimos to Griff", &pb.GimmickKey{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 900305, GimmickId: 91034005}, &pb.GimmickKey{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 901005, GimmickId: 91104005}, 510205},
		{"Marie to Yurie (Sun)", &pb.GimmickKey{GimmickSequenceScheduleId: 901505, GimmickSequenceId: 901505, GimmickId: 92034005}, &pb.GimmickKey{GimmickSequenceScheduleId: 901505, GimmickSequenceId: 901605, GimmickId: 92044005}, 520305},
		{"Marie to Yurie (Moon)", &pb.GimmickKey{GimmickSequenceScheduleId: 902105, GimmickSequenceId: 902105, GimmickId: 93034005}, &pb.GimmickKey{GimmickSequenceScheduleId: 902105, GimmickSequenceId: 902205, GimmickId: 93044005}, 520305},
	}
	for _, chain := range chains {
		t.Run(chain.name, func(t *testing.T) {
			if _, err := server.InitSequenceSchedule(ctx, &emptypb.Empty{}); err != nil {
				t.Fatal(err)
			}
			if _, err := server.UpdateSequence(ctx, &pb.UpdateSequenceRequest{GimmickSequenceScheduleId: chain.next.GimmickSequenceScheduleId, GimmickSequenceId: chain.next.GimmickSequenceId}); status.Code(err) != codes.FailedPrecondition {
				t.Fatalf("skipping to next sequence: %v", err)
			}
			if _, err := server.Unlock(ctx, &pb.UnlockRequest{GimmickKey: []*pb.GimmickKey{chain.next}}); status.Code(err) != codes.FailedPrecondition {
				t.Fatalf("unlocking next story early: %v", err)
			}
			if _, err := progress(chain.next); status.Code(err) != codes.FailedPrecondition {
				t.Fatalf("collecting next story early: %v", err)
			}
			if _, err := server.Unlock(ctx, &pb.UnlockRequest{GimmickKey: []*pb.GimmickKey{chain.first}}); err != nil {
				t.Fatal(err)
			}
			if _, err := progress(chain.next); status.Code(err) != codes.FailedPrecondition {
				t.Fatalf("unlocking a marker advanced the chain: %v", err)
			}
			response, err := progress(chain.first)
			if err != nil {
				t.Fatal(err)
			}
			if !response.IsSequenceCleared || len(response.GimmickSequenceClearReward) != 1 || response.GimmickSequenceClearReward[0].PossessionId != chain.itemId {
				t.Fatalf("first story clear response = %+v", response)
			}
			// Reopen the repository, as after a server restart, before initializing
			// the scene again. The newly available successor must survive re-entry.
			repo = sqlite.New(db, nil)
			server = NewGimmickServiceServer(repo, repo, holder)
			if _, err := server.InitSequenceSchedule(ctx, &emptypb.Empty{}); err != nil {
				t.Fatal(err)
			}
			user, err := repo.LoadUser(userId)
			if err != nil {
				t.Fatal(err)
			}
			nextKey := store.GimmickSequenceKey{GimmickSequenceScheduleId: chain.next.GimmickSequenceScheduleId, GimmickSequenceId: chain.next.GimmickSequenceId}
			if _, ok := user.Gimmick.Sequences[nextKey]; !ok {
				t.Fatal("initializing schedules did not retain the available successor")
			}
			if _, ok := user.ImportantItems[chain.itemId]; !ok {
				t.Fatal("collected story reward was not persisted")
			}
			response, err = progress(chain.first)
			if err != nil || response.IsSequenceCleared || len(response.GimmickSequenceClearReward) != 0 {
				t.Fatalf("repeated clear granted rewards: %+v, %v", response, err)
			}
			if _, err := server.UpdateSequence(ctx, &pb.UpdateSequenceRequest{GimmickSequenceScheduleId: chain.next.GimmickSequenceScheduleId, GimmickSequenceId: chain.next.GimmickSequenceId}); err != nil {
				t.Fatal(err)
			}
			if _, err := server.Unlock(ctx, &pb.UnlockRequest{GimmickKey: []*pb.GimmickKey{chain.next}}); err != nil {
				t.Fatal(err)
			}
			response, err = progress(chain.next)
			if err != nil || !response.IsSequenceCleared {
				t.Fatalf("collecting the available successor: %+v, %v", response, err)
			}
		})
	}
}
