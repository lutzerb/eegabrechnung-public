package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// ParticipationRepository manages EEG meter point participation records.
type ParticipationRepository struct {
	db *pgxpool.Pool
}

func NewParticipationRepository(db *pgxpool.Pool) *ParticipationRepository {
	return &ParticipationRepository{db: db}
}

// ListByEEG returns all participation records for a given EEG.
func (r *ParticipationRepository) ListByEEG(ctx context.Context, eegID uuid.UUID) ([]domain.EEGMeterParticipation, error) {
	q := `SELECT id, eeg_id, meter_point_id, participation_factor, share_type, valid_from, valid_until, notes, created_at
		  FROM eeg_meter_participations
		  WHERE eeg_id = $1
		  ORDER BY valid_from DESC`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var list []domain.EEGMeterParticipation
	for rows.Next() {
		var p domain.EEGMeterParticipation
		if err := rows.Scan(&p.ID, &p.EegID, &p.MeterPointID, &p.ParticipationFactor,
			&p.ShareType, &p.ValidFrom, &p.ValidUntil, &p.Notes, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// ListByMeterPoint returns all participation records for a given meter point.
func (r *ParticipationRepository) ListByMeterPoint(ctx context.Context, meterPointID uuid.UUID) ([]domain.EEGMeterParticipation, error) {
	q := `SELECT id, eeg_id, meter_point_id, participation_factor, share_type, valid_from, valid_until, notes, created_at
		  FROM eeg_meter_participations
		  WHERE meter_point_id = $1
		  ORDER BY valid_from DESC`
	rows, err := r.db.Query(ctx, q, meterPointID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var list []domain.EEGMeterParticipation
	for rows.Next() {
		var p domain.EEGMeterParticipation
		if err := rows.Scan(&p.ID, &p.EegID, &p.MeterPointID, &p.ParticipationFactor,
			&p.ShareType, &p.ValidFrom, &p.ValidUntil, &p.Notes, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// GetActiveForPeriod returns the active participation record for (eegID, meterPointID)
// at a given date (used during billing to look up the factor).
func (r *ParticipationRepository) GetActiveForPeriod(ctx context.Context, eegID, meterPointID uuid.UUID, at time.Time) (*domain.EEGMeterParticipation, error) {
	q := `SELECT id, eeg_id, meter_point_id, participation_factor, share_type, valid_from, valid_until, notes, created_at
		  FROM eeg_meter_participations
		  WHERE eeg_id = $1
		    AND meter_point_id = $2
		    AND valid_from <= $3
		    AND (valid_until IS NULL OR valid_until >= $3)
		  ORDER BY valid_from DESC
		  LIMIT 1`
	var p domain.EEGMeterParticipation
	err := r.db.QueryRow(ctx, q, eegID, meterPointID, at).Scan(
		&p.ID, &p.EegID, &p.MeterPointID, &p.ParticipationFactor,
		&p.ShareType, &p.ValidFrom, &p.ValidUntil, &p.Notes, &p.CreatedAt,
	)
	if err != nil {
		return nil, err // pgx.ErrNoRows if not found
	}
	return &p, nil
}

// Create inserts a new participation record.
func (r *ParticipationRepository) Create(ctx context.Context, p *domain.EEGMeterParticipation) error {
	q := `INSERT INTO eeg_meter_participations
		  (eeg_id, meter_point_id, participation_factor, share_type, valid_from, valid_until, notes)
		  VALUES ($1, $2, $3, $4, $5, $6, $7)
		  RETURNING id, created_at`
	return r.db.QueryRow(ctx, q,
		p.EegID, p.MeterPointID, p.ParticipationFactor, p.ShareType,
		p.ValidFrom, p.ValidUntil, p.Notes,
	).Scan(&p.ID, &p.CreatedAt)
}

// Update modifies an existing participation record.
func (r *ParticipationRepository) Update(ctx context.Context, p *domain.EEGMeterParticipation) error {
	q := `UPDATE eeg_meter_participations
		  SET participation_factor=$1, share_type=$2, valid_from=$3, valid_until=$4, notes=$5
		  WHERE id=$6`
	cmd, err := r.db.Exec(ctx, q,
		p.ParticipationFactor, p.ShareType, p.ValidFrom, p.ValidUntil, p.Notes, p.ID,
	)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("participation record not found")
	}
	return nil
}

// Delete removes a participation record by ID.
func (r *ParticipationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM eeg_meter_participations WHERE id=$1`, id)
	return err
}

// UpsertCurrentFactor sets the open-ended participation factor for a meter point in an EEG.
// If an open-ended record (valid_until IS NULL) exists, it is updated in place.
// Otherwise a new record is inserted with share_type GC and valid_from today.
func (r *ParticipationRepository) UpsertCurrentFactor(ctx context.Context, eegID, meterPointID uuid.UUID, factor float64) error {
	q := `
		WITH upd AS (
			UPDATE eeg_meter_participations
			SET participation_factor = $3, valid_from = CURRENT_DATE
			WHERE eeg_id = $1 AND meter_point_id = $2 AND valid_until IS NULL
			RETURNING id
		)
		INSERT INTO eeg_meter_participations (eeg_id, meter_point_id, participation_factor, share_type, valid_from, valid_until, notes)
		SELECT $1, $2, $3, 'GC', CURRENT_DATE, NULL, ''
		WHERE NOT EXISTS (SELECT 1 FROM upd)`
	_, err := r.db.Exec(ctx, q, eegID, meterPointID, factor)
	return err
}

// GetCurrentFactorsByEEG returns a map of meter_point_id → current participation_factor
// for all meter points in the given EEG that have an active participation record today.
func (r *ParticipationRepository) GetCurrentFactorsByEEG(ctx context.Context, eegID uuid.UUID) (map[uuid.UUID]float64, error) {
	q := `SELECT DISTINCT ON (meter_point_id) meter_point_id, participation_factor
	      FROM eeg_meter_participations
	      WHERE eeg_id = $1
	        AND valid_from <= CURRENT_DATE
	        AND (valid_until IS NULL OR valid_until >= CURRENT_DATE)
	      ORDER BY meter_point_id, valid_from DESC`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	out := make(map[uuid.UUID]float64)
	for rows.Next() {
		var mpID uuid.UUID
		var factor float64
		if err := rows.Scan(&mpID, &factor); err != nil {
			return nil, err
		}
		out[mpID] = factor
	}
	return out, rows.Err()
}
