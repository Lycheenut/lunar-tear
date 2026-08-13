package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"

	"google.golang.org/protobuf/types/known/emptypb"
)

type CharacterViewerServiceServer struct {
	pb.UnimplementedCharacterViewerServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewCharacterViewerServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *CharacterViewerServiceServer {
	return &CharacterViewerServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *CharacterViewerServiceServer) CharacterViewerTop(ctx context.Context, _ *emptypb.Empty) (*pb.CharacterViewerTopResponse, error) {
	log.Printf("[CharacterViewerService] CharacterViewerTop")

	userId := CurrentUserId(ctx, s.users, s.sessions)
	catalog := s.holder.Get().CharacterViewer
	nowMillis := gametime.NowMillis()
	var released []int32
	user, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		released = catalog.NewlyReleasedFieldIds(*user)
		for _, fieldId := range released {
			user.CharacterViewerFields[fieldId] = store.CharacterViewerFieldState{
				CharacterViewerFieldId: fieldId,
				ReleaseDatetime:        nowMillis,
				LatestVersion:          nowMillis,
			}
		}
	})
	if err != nil {
		return nil, fmt.Errorf("update character viewer user %d: %w", userId, err)
	}

	log.Printf("[CharacterViewerService] newly released %d fields for user %d", len(released), userId)

	diff := userdata.EmptyDiff()
	if len(released) > 0 {
		diff = userdata.BuildDiffFromTables(map[string]string{
			"IUserCharacterViewerField": userdata.CharacterViewerFieldRecordsForIds(user, released),
		})
	}

	return &pb.CharacterViewerTopResponse{
		ReleaseCharacterViewerFieldId: released,
		DiffUserData:                  diff,
	}, nil
}
