package main

import (
	"context"
	"database/sql"
	"log"
	"net"

	"github.com/tanmaytare/gopherdrive/internal/repository"
	pb "github.com/tanmaytare/gopherdrive/proto"

	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedMetadataServiceServer
	repo repository.MetadataRepo
}

func (s *server) RegisterFile(ctx context.Context, r *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return &pb.RegisterResponse{}, s.repo.RegisterFile(ctx, r.Id, r.Path, r.Size)
}

func (s *server) UpdateStatus(ctx context.Context, r *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	return &pb.UpdateResponse{}, s.repo.UpdateStatus(ctx, r.Id, r.Hash, r.Status)
}

func (s *server) GetFile(ctx context.Context, r *pb.GetRequest) (*pb.GetResponse, error) {
	path, hash, size, status, err := s.repo.GetFile(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return &pb.GetResponse{
		Id:     r.Id,
		Path:   path,
		Hash:   hash,
		Size:   size,
		Status: status,
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

	lis, _ := net.Listen("tcp", ":50051")
	s := grpc.NewServer()
	pb.RegisterMetadataServiceServer(s, &server{repo: repo})

	log.Println("gRPC running on :50051")
	s.Serve(lis)
}
