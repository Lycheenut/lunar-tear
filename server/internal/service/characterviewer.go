package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"

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
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return nil, fmt.Errorf("load character viewer user %d: %w", userId, err)
	}

	released := s.holder.Get().CharacterViewer.ReleasedFieldIds(user)
	log.Printf("[CharacterViewerService] released %d fields for user %d", len(released), userId)

	return &pb.CharacterViewerTopResponse{
		ReleaseCharacterViewerFieldId: released,
	}, nil
}
