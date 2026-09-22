package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/missionprogress"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestLegacySecretStoriesSurviveLoginAndResume(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e"))
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "master.bin.e")
	if err := os.WriteFile(bin, data, 0o600); err != nil {
		t.Fatal(err)
	}
	holder, err := runtime.NewHolder(bin)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name                                                string
		sequenceRow, unlocked, sequenceClear, progressClear bool
		retryBeforeInit                                     bool
	}{
		{name: "unlocked out of order", sequenceRow: true, unlocked: true},
		{name: "unlock without sequence", unlocked: true},
		{name: "completed out of order", sequenceRow: true, sequenceClear: true, progressClear: true},
		{name: "completed sequence without progress", sequenceRow: true, sequenceClear: true},
		{name: "completed progress without sequence", progressClear: true},
		{name: "completed progress with unfinished sequence", sequenceRow: true, progressClear: true},
		{name: "completed progress retried before init", progressClear: true, retryBeforeInit: true},
		{name: "placeholder rows do not unlock", sequenceRow: true},
		{name: "new story remains locked"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			ctx := context.Background()
			if err := migrations.Up(ctx, db); err != nil {
				t.Fatal(err)
			}
			raw := sqlite.New(db, nil)
			id, err := raw.CreateUser("legacy-story", model.ClientPlatform{})
			if err != nil {
				t.Fatal(err)
			}
			// Griff's No. 5 was unlocked/collected without Dimos's No. 5.
			key := store.GimmickKey{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 901005, GimmickId: 91104005}
			seqKey := store.GimmickSequenceKey{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 901005}
			completed := tt.sequenceClear || tt.progressClear
			legacy := completed || tt.unlocked
			if _, err := raw.UpdateUser(id, func(u *store.UserState) {
				questId, ok := holder.Get().ConditionResolver.RequiredQuestId(4104)
				if !ok {
					t.Fatal("missing story chapter condition")
				}
				u.Quests[questId] = store.UserQuestState{QuestId: questId, QuestStateType: model.UserQuestStateTypeCleared}
				if tt.sequenceRow {
					u.Gimmick.Sequences[seqKey] = store.GimmickSequenceState{Key: seqKey, IsGimmickSequenceCleared: tt.sequenceClear, ClearDatetime: 123, LatestVersion: 123}
				}
				if tt.unlocked || tt.sequenceRow {
					u.Gimmick.Unlocks[key] = store.GimmickUnlockState{Key: key, IsUnlocked: tt.unlocked, LatestVersion: 123}
				}
				if tt.progressClear || (tt.sequenceRow && !tt.sequenceClear) {
					u.Gimmick.Progress[key] = store.GimmickProgressState{Key: key, IsGimmickCleared: tt.progressClear, StartDatetime: 123, LatestVersion: 123}
				}
				if completed {
					u.ImportantItems[510905] = 1
				}
			}); err != nil {
				t.Fatal(err)
			}
			repo := missionprogress.NewRepository(sqlite.New(db, nil), holder)
			if _, err := NewUserServiceServer(repo, repo, holder, "", false).Auth(ctx, &pb.AuthUserRequest{Uuid: "legacy-story"}); err != nil {
				t.Fatalf("legacy login: %v", err)
			}
			server := NewGimmickServiceServer(repo, repo, holder)
			progress := func() (*pb.UpdateGimmickProgressResponse, error) {
				return server.UpdateGimmickProgress(ctx, &pb.UpdateGimmickProgressRequest{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 901005, GimmickId: 91104005, GimmickOrnamentIndex: 1, ProgressValueBit: 1})
			}
			if tt.retryBeforeInit {
				response, err := progress()
				if err != nil || response.IsSequenceCleared || len(response.GimmickSequenceClearReward) != 0 {
					t.Fatalf("replaying completed story before init: %+v, %v", response, err)
				}
			}
			if _, err := server.InitSequenceSchedule(ctx, &emptypb.Empty{}); err != nil {
				t.Fatal(err)
			}
			snapshot, err := NewDataServiceServer(repo, repo).GetUserData(ctx, &pb.UserDataGetRequest{TableName: []string{"IUserGimmick", "IUserGimmickSequence", "IUserGimmickUnlock", "IUserGimmickOrnamentProgress", "IUserImportantItem"}})
			if err != nil {
				t.Fatal(err)
			}
			for table, value := range snapshot.UserDataJson {
				if !json.Valid([]byte(value)) {
					t.Fatalf("invalid %s JSON", table)
				}
			}
			user, err := repo.LoadUser(id)
			if err != nil {
				t.Fatal(err)
			}
			sequence, present := user.Gimmick.Sequences[seqKey]
			if present != legacy || sequence.IsGimmickSequenceCleared != completed {
				t.Fatalf("legacy sequence after init: present=%v, state=%+v", present, sequence)
			}
			_, updateErr := server.UpdateSequence(ctx, &pb.UpdateSequenceRequest{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 901005})
			_, unlockErr := server.Unlock(ctx, &pb.UnlockRequest{GimmickKey: []*pb.GimmickKey{{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 901005, GimmickId: 91104005}}})
			if !legacy {
				if _, err := repo.UpdateUser(id, func(u *store.UserState) {
					u.Missions[500023] = store.UserMissionState{MissionId: 500023, MissionProgressStatusType: int32(model.MissionProgressStatusTypeClear)}
				}); err != nil {
					t.Fatal(err)
				}
				_, progressErr := progress()
				for _, err := range []error{updateErr, unlockErr, progressErr} {
					if status.Code(err) != codes.FailedPrecondition {
						t.Fatalf("new/placeholder story bypassed its predecessor: %v", err)
					}
				}
				return
			}
			if updateErr != nil || unlockErr != nil {
				t.Fatalf("legacy retry: sequence=%v unlock=%v", updateErr, unlockErr)
			}
			// An old unlock preserves access to this story, not its successor or
			// its reward before the story's own mission is complete.
			if !completed {
				if _, err := server.UpdateSequence(ctx, &pb.UpdateSequenceRequest{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 900505}); status.Code(err) != codes.FailedPrecondition {
					t.Fatalf("uncollected legacy story opened its successor: %v", err)
				}
				if _, err := progress(); status.Code(err) != codes.FailedPrecondition {
					t.Fatalf("legacy unlock bypassed its own mission: %v", err)
				}
				if _, err := repo.UpdateUser(id, func(u *store.UserState) {
					u.Missions[500023] = store.UserMissionState{MissionId: 500023, MissionProgressStatusType: int32(model.MissionProgressStatusTypeClear)}
				}); err != nil {
					t.Fatal(err)
				}
				response, err := progress()
				if err != nil || !response.IsSequenceCleared || len(response.GimmickSequenceClearReward) != 1 {
					t.Fatalf("collecting legacy story: %+v, %v", response, err)
				}
			}
			before, err := repo.LoadUser(id)
			if err != nil {
				t.Fatal(err)
			}
			// Completed stories accept retries even when their own mission record
			// is missing. Reopening the repository must not grant rewards again.
			repo = missionprogress.NewRepository(sqlite.New(db, nil), holder)
			server = NewGimmickServiceServer(repo, repo, holder)
			for range 2 {
				response, err := progress()
				if err != nil || response.IsSequenceCleared || len(response.GimmickSequenceClearReward) != 0 || len(response.GimmickOrnamentReward) != 0 {
					t.Fatalf("replaying completed story: %+v, %v", response, err)
				}
			}
			after, err := repo.LoadUser(id)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before.Gimmick, after.Gimmick) || !reflect.DeepEqual(before.ImportantItems, after.ImportantItems) || after.ImportantItems[510905] != 1 {
				t.Fatal("completed story retries changed progress or rewards")
			}
			if _, err := server.UpdateSequence(ctx, &pb.UpdateSequenceRequest{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 900505}); err != nil {
				t.Fatalf("continuing after legacy story: %v", err)
			}
			if _, err := server.UpdateSequence(ctx, &pb.UpdateSequenceRequest{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 900705}); status.Code(err) != codes.FailedPrecondition {
				t.Fatalf("legacy completion allowed another skip: %v", err)
			}
			for _, invalid := range []*pb.GimmickKey{
				{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 901005, GimmickId: 91034005},
				{GimmickSequenceScheduleId: 901505, GimmickSequenceId: 901005, GimmickId: 91104005},
				{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 999999, GimmickId: 91104005},
			} {
				if _, err := server.Unlock(ctx, &pb.UnlockRequest{GimmickKey: []*pb.GimmickKey{invalid}}); status.Code(err) != codes.FailedPrecondition {
					t.Fatalf("legacy state allowed an unrelated unlock: %v", err)
				}
				if _, err := server.UpdateGimmickProgress(ctx, &pb.UpdateGimmickProgressRequest{GimmickSequenceScheduleId: invalid.GimmickSequenceScheduleId, GimmickSequenceId: invalid.GimmickSequenceId, GimmickId: invalid.GimmickId}); status.Code(err) != codes.FailedPrecondition {
					t.Fatalf("legacy state allowed unrelated progress: %v", err)
				}
			}
		})
	}
}
