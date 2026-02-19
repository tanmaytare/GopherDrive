package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

type ProcessingJob struct {
	Ctx      context.Context
	FileID   string
	FilePath string
	UpdateFn func(context.Context, string, string, string) error
}

type Pool struct {
	jobs chan ProcessingJob
	wg   sync.WaitGroup
}

func NewPool(workerCount int) *Pool {
	p := &Pool{
		jobs: make(chan ProcessingJob),
	}

	for i := 0; i < workerCount; i++ {
		p.wg.Add(1)
		go p.worker()
	}

	return p
}

func (p *Pool) Submit(job ProcessingJob) {
	p.jobs <- job
}

func (p *Pool) Shutdown() {
	close(p.jobs)
	p.wg.Wait()
}

func (p *Pool) worker() {
	defer p.wg.Done()

	for job := range p.jobs {
		ctx := job.Ctx
		if ctx == nil {
			ctx = context.Background()
		}
		start := time.Now()
		slog.Info("processing started", "fileID", job.FileID, "start_time", start)

		done := make(chan struct{})
		var hash string
		var err error
		go func() {
			hash, err = calculateSHA(job.FilePath)
			close(done)
		}()

		select {
		case <-ctx.Done():
			slog.Warn("processing cancelled", "fileID", job.FileID)
			return
		case <-done:
		}

		status := "COMPLETED"
		if err != nil {
			status = "FAILED"
		}

		_ = job.UpdateFn(ctx, job.FileID, hash, status)

		end := time.Now()
		latency := end.Sub(start)
		slog.Info("processing finished",
			"fileID", job.FileID,
			"end_time", end,
			"latency_ms", latency.Milliseconds(),
			"status", status,
			"error", err,
		)
	}
}

func calculateSHA(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
