package interceptor

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type conversionTestStream struct {
	grpc.ServerTransportStream
	trailer metadata.MD
}

func (s *conversionTestStream) SetTrailer(md metadata.MD) error {
	s.trailer = metadata.Join(s.trailer, md)
	return nil
}

func TestDiffInterceptorIncludesMaterialConversionNotification(t *testing.T) {
	for _, existingDiff := range []bool{false, true} {
		name := "automatic_diff"
		if existingDiff {
			name = "handler_supplied_diff"
		}
		t.Run(name, func(t *testing.T) {
			db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := migrations.Up(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			repo := sqlite.New(db, nil)
			userId, err := repo.CreateUser("conversion", model.ClientPlatform{})
			if err != nil {
				t.Fatal(err)
			}
			stream := &conversionTestStream{}
			ctx := grpc.NewContextWithServerTransportStream(context.Background(), stream)
			resp, err := NewDiffInterceptor(repo, nil)(ctx, &pb.ReceiveGiftRequest{},
				&grpc.UnaryServerInfo{FullMethod: "/apb.api.gift.GiftService/ReceiveGift"},
				func(context.Context, any) (any, error) {
					// One request can grant rewards in more than one repository update.
					for range 2 {
						if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
							store.GrantPossession(user, model.PossessionTypeMaterial, 501002, 10)
						}); err != nil {
							return nil, err
						}
					}
					response := &pb.ReceiveGiftResponse{ReceivedGiftUuid: []string{"gift"}}
					if existingDiff {
						response.DiffUserData = map[string]*pb.DiffData{
							"IUserMaterial": {UpdateRecordsJson: "[]", DeleteKeysJson: "[]"},
						}
					}
					return response, nil
				})
			if err != nil {
				t.Fatal(err)
			}
			wire, err := proto.Marshal(resp.(*pb.ReceiveGiftResponse))
			if err != nil {
				t.Fatal(err)
			}
			var decoded pb.ReceiveGiftResponse
			if err := proto.Unmarshal(wire, &decoded); err != nil {
				t.Fatal(err)
			}
			diff := decoded.DiffUserData["IUserPossessionAutoConvert"]
			if diff == nil || decoded.DiffUserData["IUserMaterial"] == nil || !slices.Equal(decoded.ReceivedGiftUuid, []string{"gift"}) {
				t.Fatalf("response lost conversion notification or handler data: %v", &decoded)
			}
			var rows []struct {
				PossessionId   int32 `json:"possessionId"`
				FromCount      int32 `json:"fromCount"`
				ToPossessionId int32 `json:"toPossessionId"`
				ToCount        int32 `json:"toCount"`
			}
			if err := json.Unmarshal([]byte(diff.UpdateRecordsJson), &rows); err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 || rows[0].PossessionId != 501002 || rows[0].FromCount != 20 || rows[0].ToPossessionId != 312003 || rows[0].ToCount != 2 {
				t.Fatalf("incorrect wire notification: %+v", rows)
			}
			names := strings.Split(strings.Join(stream.trailer.Get("x-apb-update-user-data-names"), ","), ",")
			if !slices.Contains(names, "IUserPossessionAutoConvert") {
				t.Fatalf("conversion table missing from update trailer: %v", names)
			}
		})
	}
}
