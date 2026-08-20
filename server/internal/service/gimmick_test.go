package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func TestUnlockReportDoesNotRequireClearCondition(t *testing.T) {
	const (
		scheduleId = int32(900101)
		sequenceId = int32(900101)
		gimmickId  = int32(91011001)
	)

	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	repo := sqlite.New(db, nil)
	userId, err := repo.CreateUser("report-unlock", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
		user.Quests[11] = store.UserQuestState{QuestId: 11, QuestStateType: model.UserQuestStateTypeCleared}
	}); err != nil {
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

	user, err := repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if holder.Get().Gimmick.GimmickAvailable(&user, scheduleId, sequenceId, gimmickId, gametime.NowMillis()) {
		t.Fatal("report clear condition was unexpectedly satisfied by the test user")
	}

	server := NewGimmickServiceServer(repo, repo, holder)
	_, err = server.Unlock(context.Background(), &pb.UnlockRequest{GimmickKey: []*pb.GimmickKey{{
		GimmickSequenceScheduleId: scheduleId,
		GimmickSequenceId:         sequenceId,
		GimmickId:                 gimmickId,
	}}})
	if err != nil {
		t.Fatalf("unlock report before its clear condition: %v", err)
	}

	user, err = repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	key := store.GimmickKey{
		GimmickSequenceScheduleId: scheduleId,
		GimmickSequenceId:         sequenceId,
		GimmickId:                 gimmickId,
	}
	if !user.Gimmick.Unlocks[key].IsUnlocked {
		t.Fatal("report unlock was not persisted")
	}

	_, err = server.UpdateGimmickProgress(context.Background(), &pb.UpdateGimmickProgressRequest{
		GimmickSequenceScheduleId: scheduleId,
		GimmickSequenceId:         sequenceId,
		GimmickId:                 gimmickId,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("progress before report clear condition error = %v, want FailedPrecondition", err)
	}
}
