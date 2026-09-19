package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGameListenerServesGRPCAndWebView(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterConfigServiceServer(grpcServer, service.NewConfigServiceServer("localhost", 8003, "http://cdn"))
	srv := newGameHTTPServer(grpcServer, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("Gacha details")) }))
	go srv.Serve(lis)
	t.Cleanup(func() { srv.Close(); grpcServer.Stop() })
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for range 2 {
		res, err := pb.NewConfigServiceClient(conn).GetReviewServerConfig(ctx, &emptypb.Empty{})
		if err != nil || res.GetWebView().GetBaseUrl() != "http://cdn" {
			t.Fatalf("gRPC: %v %v", res, err)
		}
		response, err := http.Get("http://" + lis.Addr().String() + "/web/en/gacha-rate")
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil || string(body) != "Gacha details" {
			t.Fatalf("HTTP: %q %v", body, err)
		}
	}
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
}
