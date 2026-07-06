package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// EDAErrorRepository persists failed inbound EDA messages for operator review.
type EDAErrorRepository struct {
	db *pgxpool.Pool
}

func NewEDAErrorRepository(db *pgxpool.Pool) *EDAErrorRepository {
	return &EDAErrorRepository{db: db}
}

// Create stores a dead-letter entry.
func (r *EDAErrorRepository) Create(ctx context.Context, e *domain.EDAError) error {
	q := `INSERT INTO eda_errors (eeg_id, direction, message_type, subject, raw_content, error_msg)
	      VALUES ($1, $2, $3, $4, $5, $6)
	      RETURNING id, created_at`
	return r.db.QueryRow(ctx, q,
		e.EegID, e.Direction, e.MessageType, e.Subject, e.RawContent, e.ErrorMsg,
	).Scan(&e.ID, &e.CreatedAt)
}

// ListByEEG returns the most recent errors for an EEG.
func (r *EDAErrorRepository) ListByEEG(ctx context.Context, eegID uuid.UUID, limit int) ([]domain.EDAError, error) {
	q := `SELECT id, eeg_id, direction, message_type, subject, raw_content, error_msg, created_at
	      FROM eda_errors
	      WHERE eeg_id = $1
	      ORDER BY created_at DESC
	      LIMIT $2`
	rows, err := r.db.Query(ctx, q, eegID, limit)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var result []domain.EDAError
	for rows.Next() {
		var e domain.EDAError
		if err := rows.Scan(&e.ID, &e.EegID, &e.Direction, &e.MessageType, &e.Subject, &e.RawContent, &e.ErrorMsg, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
