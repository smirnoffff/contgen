package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"contgen/internal/storage/dbgen"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	Pool *pgxpool.Pool
	*dbgen.Queries
}

func New(ctx context.Context, dsn string) (*Storage, error) {
	connCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Info("инициализация пула соединений с PostgreSQL")

	pool, err := pgxpool.New(connCtx, dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания пула: %w", err)
	}

	slog.Info("пул соединений с PostgreSQL успешно запущен")

	return &Storage{Pool: pool, Queries: dbgen.New(pool)}, nil
}

func (s *Storage) Close() {
	if s.Pool != nil {
		s.Pool.Close()
	}
}
