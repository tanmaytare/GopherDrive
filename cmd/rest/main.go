package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	pb "github.com/tanmaytare/gopherdrive/proto"

	"github.com/google/uuid"
	"github.com/tanmaytare/gopherdrive/internal/worker"
	"google.golang.org/grpc"
)

func main() {
	conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
	client := pb.NewMetadataServiceClient(conn)

	pool := worker.NewPool(5)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		// Check DB
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		_, err := client.GetFile(ctx, &pb.GetRequest{Id: "nonexistent-healthz-check"})
		dbOK := false
		if err == nil {
			dbOK = true
		} else if err.Error() == "rpc error: code = NotFound desc = not found: sql: no rows in result set" {
			dbOK = true
		} else {
			log.Printf("gRPC healthz error: %v", err)
		}

		// Check disk
		f, ferr := os.CreateTemp("./data", "healthz-*")
		diskOK := ferr == nil
		if diskOK {
			f.Close()
			os.Remove(f.Name())
		}

		status := map[string]bool{"db": dbOK, "disk": diskOK}
		if dbOK && diskOK {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(status)
	})

	// POST /files handler
	http.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "file not provided or does not exist", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Check gRPC connectivity before proceeding
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, healthErr := client.GetFile(ctx, &pb.GetRequest{Id: "nonexistent-healthz-check"})
		if healthErr != nil && healthErr.Error() != "rpc error: code = NotFound desc = not found: sql: no rows in result set" {
			log.Printf("gRPC health check failed in POST /files: %v", healthErr)
			http.Error(w, "gRPC server unavailable", http.StatusServiceUnavailable)
			return
		}

		id := uuid.New().String()
		safeID := filepath.Base(id)
		ext := filepath.Ext(header.Filename)
		tempPath := "./data/.tmp-" + safeID
		finalPath := "./data/" + safeID

		tempFile, err := os.Create(tempPath)
		if err != nil {
			http.Error(w, "could not create file", http.StatusInternalServerError)
			return
		}
		bufWriter := bufio.NewWriter(tempFile)
		size, err := io.Copy(bufWriter, file)
		if err != nil {
			tempFile.Close()
			os.Remove(tempPath)
			http.Error(w, "could not write file", http.StatusInternalServerError)
			return
		}
		bufWriter.Flush()
		tempFile.Close()

		if err := os.Rename(tempPath, finalPath); err != nil {
			os.Remove(tempPath)
			http.Error(w, "could not save file", http.StatusInternalServerError)
			return
		}

		// Register file with extension
		_, regErr := client.RegisterFile(context.Background(), &pb.RegisterRequest{
			Id:        safeID,
			Path:      finalPath,
			Size:      size,
			Extension: ext,
		})
		if regErr != nil {
			http.Error(w, "failed to register file metadata", http.StatusInternalServerError)
			return
		}

		pool.Submit(worker.ProcessingJob{
			Ctx:      context.Background(),
			FileID:   safeID,
			FilePath: finalPath,
			UpdateFn: func(ctx context.Context, id, hash, status string) error {
				_, err := client.UpdateStatus(ctx, &pb.UpdateRequest{
					Id:     id,
					Hash:   hash,
					Status: status,
				})
				return err
			},
		})

		json.NewEncoder(w).Encode(map[string]string{"id": safeID})
	})

	// GET /files/{id} handler
	http.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Path[len("/files/"):]
		if id == "" {
			http.Error(w, "missing file id", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.GetFile(ctx, &pb.GetRequest{Id: id})
		if err != nil {
			// Always return 404 for not found or invalid id
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        resp.Id,
			"path":      resp.Path,
			"hash":      resp.Hash,
			"size":      resp.Size,
			"status":    resp.Status,
			"extension": resp.Extension,
		})
	})

	server := &http.Server{Addr: ":8080"}

	go func() {
		log.Println("REST running on :8080")
		server.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT)
	<-quit

	slog.Info("shutting down")

	server.Shutdown(context.Background())
	pool.Shutdown()
	conn.Close()
}
