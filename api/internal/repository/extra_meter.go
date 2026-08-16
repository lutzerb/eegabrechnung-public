package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// ExtraMeterRepository manages manually-read submeters ("Zusatzzähler") and their readings.
type ExtraMeterRepository struct {
	db *pgxpool.Pool
}

func NewExtraMeterRepository(db *pgxpool.Pool) *ExtraMeterRepository {
	return &ExtraMeterRepository{db: db}
}

func (r *ExtraMeterRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ExtraMeter, error) {
	q := `SELECT id, eeg_id, member_id, label, status, notes, created_at
		  FROM extra_meters WHERE id = $1`
	var m domain.ExtraMeter
	err := r.db.QueryRow(ctx, q, id).Scan(&m.ID, &m.EegID, &m.MemberID, &m.Label, &m.Status, &m.Notes, &m.CreatedAt)
	if err != nil {
		return nil, err // pgx.ErrNoRows if not found
	}
	return &m, nil
}

// ListByMember returns all extra meters (any status) for a member.
func (r *ExtraMeterRepository) ListByMember(ctx context.Context, memberID uuid.UUID) ([]domain.ExtraMeter, error) {
	q := `SELECT id, eeg_id, member_id, label, status, notes, created_at
		  FROM extra_meters WHERE member_id = $1 ORDER BY created_at`
	rows, err := r.db.Query(ctx, q, memberID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var list []domain.ExtraMeter
	for rows.Next() {
		var m domain.ExtraMeter
		if err := rows.Scan(&m.ID, &m.EegID, &m.MemberID, &m.Label, &m.Status, &m.Notes, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// ListActiveForMember returns only ACTIVE extra meters for a member — used during billing.
func (r *ExtraMeterRepository) ListActiveForMember(ctx context.Context, memberID uuid.UUID) ([]domain.ExtraMeter, error) {
	q := `SELECT id, eeg_id, member_id, label, status, notes, created_at
		  FROM extra_meters WHERE member_id = $1 AND status = 'ACTIVE' ORDER BY created_at`
	rows, err := r.db.Query(ctx, q, memberID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var list []domain.ExtraMeter
	for rows.Next() {
		var m domain.ExtraMeter
		if err := rows.Scan(&m.ID, &m.EegID, &m.MemberID, &m.Label, &m.Status, &m.Notes, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (r *ExtraMeterRepository) Create(ctx context.Context, m *domain.ExtraMeter) error {
	q := `INSERT INTO extra_meters (eeg_id, member_id, label, status, notes)
		  VALUES ($1, $2, $3, $4, $5)
		  RETURNING id, created_at`
	if m.Status == "" {
		m.Status = "ACTIVE"
	}
	return r.db.QueryRow(ctx, q, m.EegID, m.MemberID, m.Label, m.Status, m.Notes).Scan(&m.ID, &m.CreatedAt)
}

func (r *ExtraMeterRepository) Update(ctx context.Context, m *domain.ExtraMeter) error {
	q := `UPDATE extra_meters SET label=$1, status=$2, notes=$3 WHERE id=$4`
	cmd, err := r.db.Exec(ctx, q, m.Label, m.Status, m.Notes, m.ID)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("extra meter not found")
	}
	return nil
}

func (r *ExtraMeterRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM extra_meters WHERE id=$1`, id)
	return err
}

// ListReadings returns all readings for an extra meter, newest first.
func (r *ExtraMeterRepository) ListReadings(ctx context.Context, extraMeterID uuid.UUID) ([]domain.ExtraMeterReading, error) {
	q := `SELECT id, extra_meter_id, reading_date, counter_value, notes, created_by, created_at
		  FROM extra_meter_readings WHERE extra_meter_id = $1 ORDER BY reading_date DESC`
	rows, err := r.db.Query(ctx, q, extraMeterID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var list []domain.ExtraMeterReading
	for rows.Next() {
		var rd domain.ExtraMeterReading
		if err := rows.Scan(&rd.ID, &rd.ExtraMeterID, &rd.ReadingDate, &rd.CounterValue, &rd.Notes, &rd.CreatedBy, &rd.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		list = append(list, rd)
	}
	return list, rows.Err()
}

// LatestReadingOnOrBefore returns the most recent reading with reading_date <= at,
// used during billing to find the baseline/endpoint counter values bracketing a period.
// Returns nil (no error) when no such reading exists.
func (r *ExtraMeterRepository) LatestReadingOnOrBefore(ctx context.Context, extraMeterID uuid.UUID, at time.Time) (*domain.ExtraMeterReading, error) {
	q := `SELECT id, extra_meter_id, reading_date, counter_value, notes, created_by, created_at
		  FROM extra_meter_readings
		  WHERE extra_meter_id = $1 AND reading_date <= $2
		  ORDER BY reading_date DESC LIMIT 1`
	var rd domain.ExtraMeterReading
	err := r.db.QueryRow(ctx, q, extraMeterID, at).Scan(
		&rd.ID, &rd.ExtraMeterID, &rd.ReadingDate, &rd.CounterValue, &rd.Notes, &rd.CreatedBy, &rd.CreatedAt,
	)
	if err != nil {
		return nil, err // pgx.ErrNoRows if not found
	}
	return &rd, nil
}

func (r *ExtraMeterRepository) CreateReading(ctx context.Context, rd *domain.ExtraMeterReading) error {
	q := `INSERT INTO extra_meter_readings (extra_meter_id, reading_date, counter_value, notes, created_by)
		  VALUES ($1, $2, $3, $4, $5)
		  RETURNING id, created_at`
	return r.db.QueryRow(ctx, q, rd.ExtraMeterID, rd.ReadingDate, rd.CounterValue, rd.Notes, rd.CreatedBy).
		Scan(&rd.ID, &rd.CreatedAt)
}

func (r *ExtraMeterRepository) DeleteReading(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM extra_meter_readings WHERE id=$1`, id)
	return err
}
