package main

import (
	"context"
	"database/sql"
	"log"
	"net"

	pb "github.com/tanmaytare/gopherdrive/proto"

	"github.com/tanmaytare/gopherdrive/internal/repository"

	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedMetadataServiceServer
	repo repository.MetadataRepo
}

func (s *server) RegisterFile(ctx context.Context, r *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return &pb.RegisterResponse{}, s.repo.RegisterFile(ctx, r.Id, r.Path, r.Size, r.Extension)
}

func (s *server) UpdateStatus(ctx context.Context, r *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	return &pb.UpdateResponse{}, s.repo.UpdateStatus(ctx, r.Id, r.Hash, r.Status)
}

func (s *server) GetFile(ctx context.Context, r *pb.GetRequest) (*pb.GetResponse, error) {
	path, hash, size, status, extension, err := s.repo.GetFile(ctx, r.Id)
	if err != nil {
		if err.Error() != "" && (err.Error() == "not found: sql: no rows in result set" || err.Error() == "sql: no rows in result set") {
			return nil, grpcstatus.Error(codes.NotFound, err.Error())
		}
		return nil, grpcstatus.Error(codes.Unknown, err.Error())
	}
	return &pb.GetResponse{
		Id:        r.Id,
		Path:      path,
		Hash:      hash,
		Size:      size,
		Status:    status,
		Extension: extension,
	}, nil
}

func main() {
	db, err := sql.Open("mysql", "root:rootuser@tcp(127.0.0.1:3306)/gopherdrive")
	if err != nil {
		log.Fatal(err)
	}
	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * 60) // 5 minutes

	repo := repository.NewMySQLRepo(db)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterMetadataServiceServer(s, &server{repo: repo})

	log.Println("gRPC running on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
