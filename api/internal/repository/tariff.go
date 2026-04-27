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

type TariffRepository struct {
	db *pgxpool.Pool
}

func NewTariffRepository(db *pgxpool.Pool) *TariffRepository {
	return &TariffRepository{db: db}
}

func (r *TariffRepository) ListByEeg(ctx context.Context, eegID uuid.UUID) ([]domain.TariffSchedule, error) {
	q := `SELECT ts.id, ts.eeg_id, ts.name, ts.granularity, ts.is_active, ts.created_at,
	             COUNT(te.id) AS entry_count
	      FROM tariff_schedules ts
	      LEFT JOIN tariff_entries te ON te.schedule_id = ts.id
	      WHERE ts.eeg_id = $1
	      GROUP BY ts.id
	      ORDER BY ts.is_active DESC, ts.created_at DESC`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var result []domain.TariffSchedule
	for rows.Next() {
		var s domain.TariffSchedule
		if err := rows.Scan(&s.ID, &s.EegID, &s.Name, &s.Granularity, &s.IsActive, &s.CreatedAt, &s.EntryCount); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *TariffRepository) GetWithEntries(ctx context.Context, id uuid.UUID) (*domain.TariffSchedule, error) {
	var s domain.TariffSchedule
	err := r.db.QueryRow(ctx,
		`SELECT id, eeg_id, name, granularity, is_active, created_at FROM tariff_schedules WHERE id = $1`, id).
		Scan(&s.ID, &s.EegID, &s.Name, &s.Granularity, &s.IsActive, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan schedule: %w", err)
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, schedule_id, valid_from, valid_until, energy_price, producer_price, created_at
		 FROM tariff_entries WHERE schedule_id = $1 ORDER BY valid_from`, id)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var e domain.TariffEntry
		if err := rows.Scan(&e.ID, &e.ScheduleID, &e.ValidFrom, &e.ValidUntil, &e.EnergyPrice, &e.ProducerPrice, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		s.Entries = append(s.Entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	s.EntryCount = len(s.Entries)
	return &s, nil
}

func (r *TariffRepository) Create(ctx context.Context, s *domain.TariffSchedule) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO tariff_schedules (eeg_id, name, granularity, is_active)
		 VALUES ($1, $2, $3, false) RETURNING id, created_at`,
		s.EegID, s.Name, s.Granularity).
		Scan(&s.ID, &s.CreatedAt)
}

func (r *TariffRepository) Update(ctx context.Context, s *domain.TariffSchedule) error {
	_, err := r.db.Exec(ctx,
		`UPDATE tariff_schedules SET name = $2, granularity = $3 WHERE id = $1`,
		s.ID, s.Name, s.Granularity)
	return err
}

func (r *TariffRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM tariff_schedules WHERE id = $1`, id)
	return err
}

// Activate sets one schedule as active, deactivating all others for the same EEG.
func (r *TariffRepository) Activate(ctx context.Context, scheduleID, eegID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE tariff_schedules SET is_active = false WHERE eeg_id = $1`, eegID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE tariff_schedules SET is_active = true WHERE id = $1`, scheduleID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *TariffRepository) Deactivate(ctx context.Context, scheduleID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE tariff_schedules SET is_active = false WHERE id = $1`, scheduleID)
	return err
}

// ReplaceEntries replaces all entries for a schedule atomically.
func (r *TariffRepository) ReplaceEntries(ctx context.Context, scheduleID uuid.UUID, entries []domain.TariffEntry) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM tariff_entries WHERE schedule_id = $1`, scheduleID); err != nil {
		return err
	}
	for _, e := range entries {
		if _, err := tx.Exec(ctx,
			`INSERT INTO tariff_entries (schedule_id, valid_from, valid_until, energy_price, producer_price)
			 VALUES ($1, $2, $3, $4, $5)`,
			scheduleID, e.ValidFrom, e.ValidUntil, e.EnergyPrice, e.ProducerPrice); err != nil {
			return fmt.Errorf("insert entry %v: %w", e.ValidFrom, err)
		}
	}
	return tx.Commit(ctx)
}

// GetActiveForBilling returns the active tariff schedule with entries overlapping the billing period, or nil.
func (r *TariffRepository) GetActiveForBilling(ctx context.Context, eegID uuid.UUID, periodStart, periodEnd time.Time) (*domain.TariffSchedule, error) {
	var s domain.TariffSchedule
	err := r.db.QueryRow(ctx,
		`SELECT id, eeg_id, name, granularity, is_active, created_at
		 FROM tariff_schedules WHERE eeg_id = $1 AND is_active = true`, eegID).
		Scan(&s.ID, &s.EegID, &s.Name, &s.Granularity, &s.IsActive, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query active schedule: %w", err)
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, schedule_id, valid_from, valid_until, energy_price, producer_price, created_at
		 FROM tariff_entries
		 WHERE schedule_id = $1 AND valid_from < $3 AND valid_until > $2
		 ORDER BY valid_from`,
		s.ID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var e domain.TariffEntry
		if err := rows.Scan(&e.ID, &e.ScheduleID, &e.ValidFrom, &e.ValidUntil, &e.EnergyPrice, &e.ProducerPrice, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		s.Entries = append(s.Entries, e)
	}
	s.EntryCount = len(s.Entries)
	return &s, rows.Err()
}
