package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"

	"google.golang.org/protobuf/types/known/emptypb"
)

func TestCharacterViewerTopOnlyReturnsNewlyReleasedFields(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	repo := sqlite.New(db, nil)
	userID, err := repo.CreateUser("character-viewer", model.ClientPlatform{})
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
	requiredQuestID, ok := holder.Get().ConditionResolver.RequiredQuestId(4103)
	if !ok {
		t.Fatal("character viewer field 1 release condition did not resolve to a quest")
	}
	if _, err := repo.UpdateUser(userID, func(user *store.UserState) {
		user.Quests[requiredQuestID] = store.UserQuestState{
			QuestId:        requiredQuestID,
			QuestStateType: model.UserQuestStateTypeCleared,
		}
	}); err != nil {
		t.Fatal(err)
	}
	server := NewCharacterViewerServiceServer(repo, nil, holder)

	first, err := server.CharacterViewerTop(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.ReleaseCharacterViewerFieldId) == 0 {
		t.Fatal("first entrance did not release any character viewer fields")
	}
	diff, ok := first.DiffUserData["IUserCharacterViewerField"]
	if !ok {
		t.Fatal("first entrance did not return IUserCharacterViewerField diff")
	}
	var records []struct {
		CharacterViewerFieldId int32 `json:"characterViewerFieldId"`
	}
	if err := json.Unmarshal([]byte(diff.UpdateRecordsJson), &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != len(first.ReleaseCharacterViewerFieldId) {
		t.Fatalf("diff records = %d, released fields = %d", len(records), len(first.ReleaseCharacterViewerFieldId))
	}

	second, err := server.CharacterViewerTop(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.ReleaseCharacterViewerFieldId) != 0 {
		t.Fatalf("second entrance released fields again: %v", second.ReleaseCharacterViewerFieldId)
	}
	if len(second.DiffUserData) != 0 {
		t.Fatalf("second entrance returned diff: %v", second.DiffUserData)
	}

	reloadedRepo := sqlite.New(db, nil)
	reloadedUser, err := reloadedRepo.LoadUser(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloadedUser.CharacterViewerFields) != len(first.ReleaseCharacterViewerFieldId) {
		t.Fatalf("persisted fields = %d, released fields = %d", len(reloadedUser.CharacterViewerFields), len(first.ReleaseCharacterViewerFieldId))
	}
	reloadedServer := NewCharacterViewerServiceServer(reloadedRepo, nil, holder)
	afterReload, err := reloadedServer.CharacterViewerTop(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(afterReload.ReleaseCharacterViewerFieldId) != 0 {
		t.Fatalf("fields released again after reload: %v", afterReload.ReleaseCharacterViewerFieldId)
	}
}
