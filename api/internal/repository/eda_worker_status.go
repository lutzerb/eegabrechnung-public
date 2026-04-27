package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// EDAWorkerStatusRepository reads and updates the eda_worker_status singleton.
type EDAWorkerStatusRepository struct {
	db *pgxpool.Pool
}

func NewEDAWorkerStatusRepository(db *pgxpool.Pool) *EDAWorkerStatusRepository {
	return &EDAWorkerStatusRepository{db: db}
}

// Upsert updates the singleton row with the current worker state.
func (r *EDAWorkerStatusRepository) Upsert(ctx context.Context, s *domain.EDAWorkerStatus) error {
	q := `INSERT INTO eda_worker_status (id, transport_mode, last_poll_at, last_error, updated_at)
	      VALUES (1, $1, $2, $3, now())
	      ON CONFLICT (id) DO UPDATE
	        SET transport_mode = EXCLUDED.transport_mode,
	            last_poll_at   = EXCLUDED.last_poll_at,
	            last_error     = EXCLUDED.last_error,
	            updated_at     = now()`
	_, err := r.db.Exec(ctx, q, s.TransportMode, s.LastPollAt, s.LastError)
	return err
}

// Get returns the current worker status.
func (r *EDAWorkerStatusRepository) Get(ctx context.Context) (*domain.EDAWorkerStatus, error) {
	q := `SELECT transport_mode, last_poll_at, last_error, updated_at
	      FROM eda_worker_status WHERE id = 1`
	var s domain.EDAWorkerStatus
	if err := r.db.QueryRow(ctx, q).Scan(&s.TransportMode, &s.LastPollAt, &s.LastError, &s.UpdatedAt); err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return &s, nil
}
