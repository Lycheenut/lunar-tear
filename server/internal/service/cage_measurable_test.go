package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func TestCageMeasurableValuesAreConsumedByEveryReportingRPC(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	userId, err := repo.CreateUser("cage-measurable", model.ClientPlatform{})
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

	ctx := context.Background()
	if _, err := NewMissionServiceServer(repo, repo, holder).UpdateMissionProgress(ctx, &pb.UpdateMissionProgressRequest{
		CageMeasurableValues: &pb.CageMeasurableValues{RunningDistanceMeters: 10},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewBannerServiceServer(repo, repo, holder).GetMamaBanner(ctx, &pb.GetMamaBannerRequest{
		CageMeasurableValues: &pb.CageMeasurableValues{RunningDistanceMeters: 20},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFriendServiceServer(repo, repo, holder).GetFriendList(ctx, &pb.GetFriendListRequest{
		CageMeasurableValues: &pb.CageMeasurableValues{RunningDistanceMeters: 30},
	}); err != nil {
		t.Fatal(err)
	}

	questIds := holder.Get().Quest.OrderedQuestIds
	if len(questIds) == 0 {
		t.Fatal("master data contains no main quests")
	}
	questServer := NewQuestServiceServer(repo, repo, holder)
	if _, err := questServer.StartMainQuest(ctx, &pb.StartMainQuestRequest{
		QuestId:              questIds[0],
		CageMeasurableValues: &pb.CageMeasurableValues{RunningDistanceMeters: 40},
	}); err != nil {
		t.Fatalf("start main quest %d: %v", questIds[0], err)
	}

	note, err := NewUserServiceServer(repo, repo, holder, "", false).GetUserGamePlayNote(ctx, &pb.GetUserGamePlayNoteRequest{
		GamePlayHistoryTypeId: playHistoryCageWalkDistance,
	})
	if err != nil {
		t.Fatal(err)
	}
	if note.ProgressValue != 100 {
		t.Fatalf("cage walk history = %d, want 100", note.ProgressValue)
	}
	user, err := repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if user.CageRunningDistanceMeters != 100 {
		t.Fatalf("persisted cage running distance = %d, want 100", user.CageRunningDistanceMeters)
	}

	if _, err := questServer.StartMainQuest(ctx, &pb.StartMainQuestRequest{
		QuestId:              -1,
		CageMeasurableValues: &pb.CageMeasurableValues{RunningDistanceMeters: 50},
	}); err == nil {
		t.Fatal("unknown quest was accepted")
	}
	user, err = repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if user.CageRunningDistanceMeters != 100 {
		t.Fatalf("failed quest persisted cage distance: got %d, want 100", user.CageRunningDistanceMeters)
	}
}
