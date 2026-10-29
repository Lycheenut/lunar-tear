package interceptor

import (
	"context"
	"log"
	"reflect"
	"sort"
	"strings"
	"sync"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/service"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type diffUserDataGetter interface {
	GetDiffUserData() map[string]*pb.DiffData
}

var (
	diffFieldCache sync.Map // map[reflect.Type]bool
	diffDataMapTyp = reflect.TypeFor[map[string]*pb.DiffData]()
)

func hasDiffField(resp any) bool {
	t := reflect.TypeOf(resp)
	if cached, ok := diffFieldCache.Load(t); ok {
		return cached.(bool)
	}
	elem := t
	if elem.Kind() == reflect.Pointer {
		elem = elem.Elem()
	}
	f, ok := elem.FieldByName("DiffUserData")
	result := ok && f.Type == diffDataMapTyp
	diffFieldCache.Store(t, result)
	return result
}

func NewDiffInterceptor(
	users store.UserRepository,
	sessions store.SessionRepository,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if skipDiffForMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		userId := service.CurrentUserId(ctx, users, sessions)
		if userId == 0 {
			return handler(ctx, req)
		}

		before, err := users.LoadUser(userId)
		if err != nil {
			return handler(ctx, req)
		}

		resp, handlerErr := handler(ctx, req)
		if handlerErr != nil || resp == nil {
			return resp, handlerErr
		}

		if !hasDiffField(resp) {
			return resp, nil
		}

		if getter, ok := resp.(diffUserDataGetter); ok {
			if existing := getter.GetDiffUserData(); len(existing) > 0 {
				if after, err := users.LoadUser(userId); err == nil {
					if conversion := userdata.MaterialConversionDiff(&before, &after); conversion != nil {
						existing["IUserPossessionAutoConvert"] = conversion
					}
				}
				setUpdateNamesTrailer(ctx, existing)
				return resp, nil
			}
		}

		after, err := users.LoadUser(userId)
		if err != nil {
			return resp, nil
		}

		changed := userdata.ChangedTables(&before, &after)
		if len(changed) == 0 {
			return resp, nil
		}

		diff := userdata.ComputeDelta(&before, &after, changed)
		reflect.ValueOf(resp).Elem().FieldByName("DiffUserData").Set(reflect.ValueOf(diff))
		setUpdateNamesTrailer(ctx, diff)

		return resp, nil
	}
}

func skipDiffForMethod(method string) bool {
	switch method {
	case "/apb.api.user.UserService/Auth",
		"/apb.api.user.UserService/RegisterUser",
		"/apb.api.user.UserService/TransferUser",
		"/apb.api.user.UserService/TransferUserByFacebook",
		"/apb.api.config.ConfigService/GetConfig",
		"/apb.api.data.DataService/GetLatestMasterDataVersion",
		"/apb.api.data.DataService/GetUserDataNameV2",
		"/apb.api.data.DataService/GetUserData":
		return true
	}
	return false
}

func setUpdateNamesTrailer(ctx context.Context, diff map[string]*pb.DiffData) {
	if len(diff) == 0 {
		return
	}
	keys := make([]string, 0, len(diff))
	for key := range diff {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	value := strings.Join(keys, ",")
	if err := grpc.SetTrailer(ctx, metadata.Pairs("x-apb-update-user-data-names", value)); err != nil {
		log.Printf("[DiffInterceptor] failed to set trailer: %v", err)
	}
}
