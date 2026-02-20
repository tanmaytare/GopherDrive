package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type MetadataRepo interface {
	RegisterFile(ctx context.Context, id, path string, size int64, extension string) error
	UpdateStatus(ctx context.Context, id, hash, status string) error
	GetFile(ctx context.Context, id string) (string, string, int64, string, string, error)
}

type MySQLRepo struct {
	db *sql.DB
}

func NewMySQLRepo(db *sql.DB) *MySQLRepo {
	return &MySQLRepo{db: db}
}

func (m *MySQLRepo) RegisterFile(ctx context.Context, id, path string, size int64, extension string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	stmt, err := m.db.PrepareContext(ctx,
		"INSERT INTO metadata (id, file_path, extension, file_size, status) VALUES (?, ?, ?, ?, 'PENDING')")
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, id, path, extension, size)
	if err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}
	return nil
}

func (m *MySQLRepo) UpdateStatus(ctx context.Context, id, hash, status string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	stmt, err := m.db.PrepareContext(ctx,
		"UPDATE metadata SET sha256=?, status=? WHERE id=?")
	if err != nil {
		return fmt.Errorf("prepare update: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, hash, status, id)
	if err != nil {
		return fmt.Errorf("exec update: %w", err)
	}
	return nil
}

func (m *MySQLRepo) GetFile(ctx context.Context, id string) (string, string, int64, string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	row := m.db.QueryRowContext(ctx,
		"SELECT file_path, sha256, file_size, status, extension FROM metadata WHERE id=?", id)

	var path, hash, status, extension string
	var size int64

	err := row.Scan(&path, &hash, &size, &status, &extension)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", 0, "", "", fmt.Errorf("not found: %w", err)
		}
		return "", "", 0, "", "", fmt.Errorf("scan: %w", err)
	}
	return path, hash, size, status, extension, nil
}
