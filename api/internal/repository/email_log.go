package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type EmailLogRepository struct {
	db *pgxpool.Pool
}

func NewEmailLogRepository(db *pgxpool.Pool) *EmailLogRepository {
	return &EmailLogRepository{db: db}
}

// Log writes one email attempt to the log table. Non-fatal: errors are silently ignored
// so that a logging failure never blocks the actual email send result.
func (r *EmailLogRepository) Log(ctx context.Context, e domain.EmailLogEntry) {
	_, _ = r.db.Exec(ctx,
		`INSERT INTO email_log (eeg_id, email_type, to_address, subject, status, error_msg, member_id, invoice_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		e.EegID, e.EmailType, e.ToAddress, e.Subject, e.Status, e.ErrorMsg, e.MemberID, e.InvoiceID,
	)
}

// ListByEEG returns email log entries for an EEG, newest first.
func (r *EmailLogRepository) ListByEEG(ctx context.Context, eegID uuid.UUID, limit, offset int) ([]domain.EmailLogEntry, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM email_log WHERE eeg_id = $1`, eegID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, eeg_id, email_type, to_address, subject, status, error_msg, member_id, invoice_id, created_at
		 FROM email_log
		 WHERE eeg_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		eegID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []domain.EmailLogEntry
	for rows.Next() {
		var e domain.EmailLogEntry
		if err := rows.Scan(&e.ID, &e.EegID, &e.EmailType, &e.ToAddress, &e.Subject,
			&e.Status, &e.ErrorMsg, &e.MemberID, &e.InvoiceID, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}
