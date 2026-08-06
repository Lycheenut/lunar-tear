package service

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func TestFinishWaveCheckpointSurvivesReloadAndNextWaveStart(t *testing.T) {
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
	if _, err := server.FinishWave(context.Background(), &pb.FinishWaveRequest{BattleBinary: checkpoint}); err != nil {
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
	}
	assertCheckpoint("finished wave")

	if _, err := server.StartWave(context.Background(), &pb.StartWaveRequest{}); err != nil {
		t.Fatal(err)
	}
	assertCheckpoint("next wave started")
}
