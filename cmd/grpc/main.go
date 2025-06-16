package main

import (
	"log"
	"net"
	"os"

	_ "github.com/lib/pq"
	pb "github.com/taiki-umetsu/ndc007-bookpicker/api/v1"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/database"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/env"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/grpcserver"
	"google.golang.org/grpc"
)

func main() {
	env.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("環境変数 DATABASE_URL が未設定です")
	}

	db, err := database.Setup(dsn)
	if err != nil {
		log.Fatal("DB接続エラー:", err)
	}
	defer db.Close()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterBookServiceServer(s, grpcserver.NewBookServiceServer(db))

	log.Println("gRPC server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
