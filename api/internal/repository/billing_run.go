package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type BillingRunRepository struct {
	db *pgxpool.Pool
}

func NewBillingRunRepository(db *pgxpool.Pool) *BillingRunRepository {
	return &BillingRunRepository{db: db}
}

func (r *BillingRunRepository) Create(ctx context.Context, br *domain.BillingRun) error {
	q := `INSERT INTO billing_runs (eeg_id, period_start, period_end, status)
	      VALUES ($1, $2, $3, $4)
	      RETURNING id, created_at`
	return r.db.QueryRow(ctx, q, br.EegID, br.PeriodStart, br.PeriodEnd, br.Status).
		Scan(&br.ID, &br.CreatedAt)
}

// ListByEeg returns billing runs for an EEG with invoice count and total amount.
func (r *BillingRunRepository) ListByEeg(ctx context.Context, eegID uuid.UUID) ([]domain.BillingRun, error) {
	q := `SELECT br.id, br.eeg_id, br.period_start, br.period_end, br.status, br.created_at,
	             COUNT(i.id)                     AS invoice_count,
	             COALESCE(SUM(i.total_amount), 0) AS total_amount
	      FROM billing_runs br
	      LEFT JOIN invoices i ON i.billing_run_id = br.id
	      WHERE br.eeg_id = $1
	      GROUP BY br.id
	      ORDER BY br.created_at DESC`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var runs []domain.BillingRun
	for rows.Next() {
		var br domain.BillingRun
		if err := rows.Scan(
			&br.ID, &br.EegID, &br.PeriodStart, &br.PeriodEnd, &br.Status, &br.CreatedAt,
			&br.InvoiceCount, &br.TotalAmount,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		runs = append(runs, br)
	}
	return runs, rows.Err()
}

// FindOverlap returns an existing billing run whose period overlaps with [start, end],
// or nil if there is no overlap.
func (r *BillingRunRepository) FindOverlap(ctx context.Context, eegID uuid.UUID, start, end time.Time) (*domain.BillingRun, error) {
	q := `SELECT id, eeg_id, period_start, period_end, status, created_at
	      FROM billing_runs
	      WHERE eeg_id = $1
	        AND status != 'cancelled'
	        AND period_start < $3
	        AND period_end   > $2
	      LIMIT 1`
	var br domain.BillingRun
	err := r.db.QueryRow(ctx, q, eegID, start, end).Scan(
		&br.ID, &br.EegID, &br.PeriodStart, &br.PeriodEnd, &br.Status, &br.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query: %w", err)
	}
	return &br, nil
}

// Finalize promotes a draft billing run to finalized status.
// Only draft runs can be finalized. Returns an error if the run is not in draft status.
func (r *BillingRunRepository) Finalize(ctx context.Context, id uuid.UUID) error {
	var status string
	if err := r.db.QueryRow(ctx, `SELECT status FROM billing_runs WHERE id=$1`, id).Scan(&status); err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("billing run not found")
		}
		return fmt.Errorf("get status: %w", err)
	}
	if status != "draft" {
		return fmt.Errorf("only draft billing runs can be finalized (current status: %s)", status)
	}
	_, err := r.db.Exec(ctx, `UPDATE billing_runs SET status='finalized' WHERE id=$1`, id)
	return err
}

// DeleteDraft hard-deletes a draft billing run and all its invoices.
// Returns an error if the run is not in draft status.
func (r *BillingRunRepository) DeleteDraft(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM billing_runs WHERE id=$1`, id).Scan(&status); err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("billing run not found")
		}
		return fmt.Errorf("get status: %w", err)
	}
	if status != "draft" {
		return fmt.Errorf("only draft billing runs can be deleted (current status: %s)", status)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM invoices WHERE billing_run_id=$1`, id); err != nil {
		return fmt.Errorf("delete invoices: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM billing_runs WHERE id=$1`, id); err != nil {
		return fmt.Errorf("delete billing run: %w", err)
	}
	return tx.Commit(ctx)
}

// Cancel sets a finalized billing run's status to 'cancelled' and marks all non-paid
// invoices as cancelled. Draft runs must be deleted via DeleteDraft instead.
func (r *BillingRunRepository) Cancel(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check current status
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM billing_runs WHERE id=$1`, id).Scan(&status); err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("billing run not found")
		}
		return fmt.Errorf("get status: %w", err)
	}
	if status == "cancelled" {
		return fmt.Errorf("billing run is already cancelled")
	}
	if status == "draft" {
		return fmt.Errorf("draft billing runs cannot be cancelled — use delete instead")
	}

	// Cancel the run
	if _, err := tx.Exec(ctx, `UPDATE billing_runs SET status='cancelled' WHERE id=$1`, id); err != nil {
		return fmt.Errorf("cancel run: %w", err)
	}

	// Cancel all non-paid invoices in the run
	if _, err := tx.Exec(ctx,
		`UPDATE invoices SET status='cancelled' WHERE billing_run_id=$1 AND status != 'paid'`, id,
	); err != nil {
		return fmt.Errorf("cancel invoices: %w", err)
	}

	return tx.Commit(ctx)
}

// ListNonPaidInvoiceIDs returns IDs of all non-paid, non-cancelled invoices in a billing run.
func (r *BillingRunRepository) ListNonPaidInvoiceIDs(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id FROM invoices WHERE billing_run_id=$1 AND status != 'paid' AND status != 'cancelled'`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *BillingRunRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.BillingRun, error) {
	q := `SELECT br.id, br.eeg_id, br.period_start, br.period_end, br.status, br.created_at,
	             COUNT(i.id)                     AS invoice_count,
	             COALESCE(SUM(i.total_amount), 0) AS total_amount
	      FROM billing_runs br
	      LEFT JOIN invoices i ON i.billing_run_id = br.id
	      WHERE br.id = $1
	      GROUP BY br.id`
	var br domain.BillingRun
	err := r.db.QueryRow(ctx, q, id).Scan(
		&br.ID, &br.EegID, &br.PeriodStart, &br.PeriodEnd, &br.Status, &br.CreatedAt,
		&br.InvoiceCount, &br.TotalAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &br, nil
}
