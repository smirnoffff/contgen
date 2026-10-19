package storage

import (
	"context"
	"errors"
	"sync"
	"time"

	"contgen/internal/storage/dbgen"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrWorkerRunning = errors.New("обработчик видео уже запущен")

// WorkerLease binds every worker query to the session holding the advisory lock.
// A lost connection therefore prevents both claiming and completing jobs. A new
// worker can recover PROCESSING jobs only after the previous session is gone.
// Queries must be used sequentially; a PostgreSQL connection is not concurrent.
type WorkerLease struct {
	*dbgen.Queries
	conn *pgxpool.Conn
	once sync.Once
}

func (s *Storage) AcquireWorker(ctx context.Context) (*WorkerLease, error) {
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	q := dbgen.New(conn)
	locked, err := q.TryWorkerLock(ctx)
	if err != nil || !locked {
		// If the result was interrupted, the server may have acquired the lock.
		// Closing that session instead of pooling it guarantees lock release.
		if err != nil {
			_ = conn.Conn().Close(context.Background())
		}
		conn.Release()
		if err != nil {
			return nil, err
		}
		return nil, ErrWorkerRunning
	}
	return &WorkerLease{Queries: q, conn: conn}, nil
}

func (w *WorkerLease) Close() {
	w.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := w.ReleaseWorkerLock(ctx); err != nil {
			_ = w.conn.Conn().Close(ctx)
		}
		w.conn.Release()
	})
}
