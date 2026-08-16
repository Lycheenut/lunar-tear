package main

import (
	"log"
	"net"
	"strconv"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/interceptor"
	"lunar-tear/server/internal/missionprogress"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/service"
	"lunar-tear/server/internal/store"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type loggingListener struct {
	net.Listener
}

func (l loggingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		log.Printf("[gRPC] Accept error: %v", err)
		return nil, err
	}
	log.Printf("[gRPC] New connection from %v", conn.RemoteAddr())
	return conn, nil
}

func startGRPC(
	listenAddr string,
	publicAddr string,
	octoURL string,
	authURL string,
	userStore interface {
		store.UserRepository
		store.SessionRepository
	},
	holder *runtime.Holder,
	noRegister bool,
) *grpc.Server {
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", listenAddr, err)
	}
	lis = loggingListener{Listener: lis}

	diffInterceptor := interceptor.NewDiffInterceptor(userStore, userStore)
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptor.Platform, interceptor.Logging, diffInterceptor, interceptor.TimeSync),
		grpc.UnknownServiceHandler(interceptor.UnknownService),
	)

	registerServices(grpcServer, publicAddr, octoURL, authURL, userStore, holder, noRegister)

	reflection.Register(grpcServer)

	log.Printf("gRPC server listening on %s", lis.Addr())
	log.Printf("public address: %s", publicAddr)

	if noRegister {
		log.Print("[!!WARNING!!] The gRPC server is running in NO-REGISTER mode. All new user registrations are denied, only existing accounts and auth-server logins are permitted.")
	}

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()
	return grpcServer
}

func registerServices(
	srv *grpc.Server,
	publicAddr string,
	octoURL string,
	authURL string,
	userStore interface {
		store.UserRepository
		store.SessionRepository
	},
	holder *runtime.Holder,
	noRegister bool,
) {
	pubHost, pubPortStr, _ := net.SplitHostPort(publicAddr)
	pubPort, _ := strconv.Atoi(pubPortStr)
	userStore = missionprogress.NewRepository(userStore, holder)

	pb.RegisterBannerServiceServer(srv, service.NewBannerServiceServer(userStore, userStore, holder))
	pb.RegisterUserServiceServer(srv, service.NewUserServiceServer(userStore, userStore, holder, authURL, noRegister))
	pb.RegisterBattleServiceServer(srv, service.NewBattleServiceServer(userStore, userStore))
	pb.RegisterConfigServiceServer(srv, service.NewConfigServiceServer(pubHost, int32(pubPort), octoURL))
	pb.RegisterDataServiceServer(srv, service.NewDataServiceServer(userStore, userStore))
	pb.RegisterTutorialServiceServer(srv, service.NewTutorialServiceServer(userStore, userStore, holder))
	pb.RegisterGachaServiceServer(srv, service.NewGachaServiceServer(userStore, userStore, holder))
	pb.RegisterGiftServiceServer(srv, service.NewGiftServiceServer(userStore, userStore, holder))
	pb.RegisterGamePlayServiceServer(srv, service.NewGameplayServiceServer())
	pb.RegisterGimmickServiceServer(srv, service.NewGimmickServiceServer(userStore, userStore, holder))
	pb.RegisterQuestServiceServer(srv, service.NewQuestServiceServer(userStore, userStore, holder))
	pb.RegisterNotificationServiceServer(srv, service.NewNotificationServiceServer(userStore, userStore))
	pb.RegisterCageOrnamentServiceServer(srv, service.NewCageOrnamentServiceServer(userStore, userStore, holder))
	pb.RegisterDeckServiceServer(srv, service.NewDeckServiceServer(userStore, userStore))
	pb.RegisterFriendServiceServer(srv, service.NewFriendServiceServer(userStore, userStore, holder))
	pb.RegisterLoginBonusServiceServer(srv, service.NewLoginBonusServiceServer(userStore, userStore, holder))
	pb.RegisterNaviCutInServiceServer(srv, service.NewNaviCutInServiceServer(userStore, userStore))
	pb.RegisterContentsStoryServiceServer(srv, service.NewContentsStoryServiceServer(userStore, userStore))
	pb.RegisterDokanServiceServer(srv, service.NewDokanServiceServer(userStore, userStore))
	pb.RegisterPortalCageServiceServer(srv, service.NewPortalCageServiceServer(userStore, userStore))
	pb.RegisterCharacterViewerServiceServer(srv, service.NewCharacterViewerServiceServer(userStore, userStore, holder))
	pb.RegisterMissionServiceServer(srv, service.NewMissionServiceServer(userStore, userStore, holder))
	pb.RegisterShopServiceServer(srv, service.NewShopServiceServer(userStore, userStore, holder))
	pb.RegisterCostumeServiceServer(srv, service.NewCostumeServiceServer(userStore, userStore, holder))
	pb.RegisterMovieServiceServer(srv, service.NewMovieServiceServer(userStore, userStore))
	pb.RegisterOmikujiServiceServer(srv, service.NewOmikujiServiceServer(userStore, userStore, holder))
	pb.RegisterWeaponServiceServer(srv, service.NewWeaponServiceServer(userStore, userStore, holder))
	pb.RegisterExploreServiceServer(srv, service.NewExploreServiceServer(userStore, userStore, holder))
	pb.RegisterCharacterBoardServiceServer(srv, service.NewCharacterBoardServiceServer(userStore, userStore, holder))
	pb.RegisterPartsServiceServer(srv, service.NewPartsServiceServer(userStore, userStore, holder))
	pb.RegisterCharacterServiceServer(srv, service.NewCharacterServiceServer(userStore, userStore, holder))
	pb.RegisterCompanionServiceServer(srv, service.NewCompanionServiceServer(userStore, userStore, holder))
	pb.RegisterMaterialServiceServer(srv, service.NewMaterialServiceServer(userStore, userStore, holder))
	pb.RegisterConsumableItemServiceServer(srv, service.NewConsumableItemServiceServer(userStore, userStore, holder))
	pb.RegisterSideStoryQuestServiceServer(srv, service.NewSideStoryQuestServiceServer(userStore, userStore, holder))
	pb.RegisterBigHuntServiceServer(srv, service.NewBigHuntServiceServer(userStore, userStore, holder))
	pb.RegisterRewardServiceServer(srv, service.NewRewardServiceServer(userStore, userStore, holder))
	pb.RegisterPvpServiceServer(srv, service.NewPvpServiceServer(userStore, userStore))
	pb.RegisterLabyrinthServiceServer(srv, service.NewLabyrinthServiceServer(userStore, userStore, holder))
}
