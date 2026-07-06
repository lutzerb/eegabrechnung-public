package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// MeterPointRegistrationPeriodRepository tracks the full Anmeldung/Abmeldung history
// for a Zählpunkt, keyed by (eeg_id, zaehlpunkt) so history survives Zählpunkt
// recycling (a new meter_points row for a new member) as well as re-registration of
// the same row.
type MeterPointRegistrationPeriodRepository struct {
	db *pgxpool.Pool
}

func NewMeterPointRegistrationPeriodRepository(db *pgxpool.Pool) *MeterPointRegistrationPeriodRepository {
	return &MeterPointRegistrationPeriodRepository{db: db}
}

// OpenPeriod inserts a new open registration period, unless one is already open for
// this Zählpunkt (idempotent — tolerates replayed/duplicate confirmations).
func (r *MeterPointRegistrationPeriodRepository) OpenPeriod(ctx context.Context, eegID uuid.UUID, zaehlpunkt string, meterPointID uuid.UUID, registriertSeit time.Time, consentID, source string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO meter_point_registration_periods (eeg_id, zaehlpunkt, meter_point_id, registriert_seit, consent_id, source)
		 SELECT $1, $2, $3, $4, $5, $6
		 WHERE NOT EXISTS (
		   SELECT 1 FROM meter_point_registration_periods
		   WHERE eeg_id = $1 AND zaehlpunkt = $2 AND abgemeldet_am IS NULL
		 )`,
		eegID, zaehlpunkt, meterPointID, registriertSeit.UTC().Truncate(24*time.Hour), consentID, source,
	)
	if err != nil {
		return fmt.Errorf("open registration period: %w", err)
	}
	return nil
}

// ClosePeriod closes whichever period is currently open for this Zählpunkt. No-op if
// none is open.
func (r *MeterPointRegistrationPeriodRepository) ClosePeriod(ctx context.Context, eegID uuid.UUID, zaehlpunkt string, abgemeldetAm time.Time, reason, source string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE meter_point_registration_periods
		 SET abgemeldet_am = $3, reason = $4, source = $5
		 WHERE eeg_id = $1 AND zaehlpunkt = $2 AND abgemeldet_am IS NULL`,
		eegID, zaehlpunkt, abgemeldetAm.UTC().Truncate(24*time.Hour), reason, source,
	)
	if err != nil {
		return fmt.Errorf("close registration period: %w", err)
	}
	return nil
}

// ReopenLatestPeriod clears abgemeldet_am on the most recent period for this
// Zählpunkt. Used by the manual "Datum löschen" admin action.
func (r *MeterPointRegistrationPeriodRepository) ReopenLatestPeriod(ctx context.Context, eegID uuid.UUID, zaehlpunkt string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE meter_point_registration_periods
		 SET abgemeldet_am = NULL, reason = ''
		 WHERE id = (
		   SELECT id FROM meter_point_registration_periods
		   WHERE eeg_id = $1 AND zaehlpunkt = $2
		   ORDER BY registriert_seit DESC, created_at DESC
		   LIMIT 1
		 )`,
		eegID, zaehlpunkt,
	)
	if err != nil {
		return fmt.Errorf("reopen registration period: %w", err)
	}
	return nil
}

// ListByZaehlpunkt returns all registration periods for a Zählpunkt, oldest first.
func (r *MeterPointRegistrationPeriodRepository) ListByZaehlpunkt(ctx context.Context, eegID uuid.UUID, zaehlpunkt string) ([]domain.MeterPointRegistrationPeriod, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, eeg_id, zaehlpunkt, meter_point_id, registriert_seit, abgemeldet_am, consent_id, source, reason, created_at
		 FROM meter_point_registration_periods
		 WHERE eeg_id = $1 AND zaehlpunkt = $2
		 ORDER BY registriert_seit ASC`,
		eegID, zaehlpunkt,
	)
	if err != nil {
		return nil, fmt.Errorf("query registration periods: %w", err)
	}
	defer rows.Close()

	var periods []domain.MeterPointRegistrationPeriod
	for rows.Next() {
		var p domain.MeterPointRegistrationPeriod
		if err := rows.Scan(&p.ID, &p.EegID, &p.Zaehlpunkt, &p.MeterPointID, &p.RegistriertSeit, &p.AbgemeldetAm, &p.ConsentID, &p.Source, &p.Reason, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan registration period: %w", err)
		}
		periods = append(periods, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return periods, nil
}
