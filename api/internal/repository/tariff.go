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

// scheduleCols is the shared column list for tariff_schedules (without entry_count, which is
// computed via a join/aggregate at each call site).
const scheduleCols = `ts.id, ts.eeg_id, ts.member_id, ts.name, ts.granularity, ts.is_active, ts.created_at,
	ts.free_kwh_override, ts.discount_pct_override, ts.meter_fee_eur_override,
	ts.participation_fee_eur_override, ts.zaehlpunkts_gebuehr_eur_override`

func scanSchedule(row pgx.Row, s *domain.TariffSchedule) error {
	return row.Scan(&s.ID, &s.EegID, &s.MemberID, &s.Name, &s.Granularity, &s.IsActive, &s.CreatedAt,
		&s.FreeKwhOverride, &s.DiscountPctOverride, &s.MeterFeeEurOverride,
		&s.ParticipationFeeEurOverride, &s.ZaehlpunktsGebuehrOverride)
}

// ListByEeg returns the EEG-wide (non-member-specific) tariff schedules.
func (r *TariffRepository) ListByEeg(ctx context.Context, eegID uuid.UUID) ([]domain.TariffSchedule, error) {
	q := `SELECT ` + scheduleCols + `, COUNT(te.id) AS entry_count
	      FROM tariff_schedules ts
	      LEFT JOIN tariff_entries te ON te.schedule_id = ts.id
	      WHERE ts.eeg_id = $1 AND ts.member_id IS NULL
	      GROUP BY ts.id
	      ORDER BY ts.is_active DESC, ts.created_at DESC`
	return r.listSchedules(ctx, q, eegID)
}

// ListAllByEeg returns every schedule for the EEG — default and member-specific alike.
// Used by backup export, which needs a complete snapshot regardless of scope.
func (r *TariffRepository) ListAllByEeg(ctx context.Context, eegID uuid.UUID) ([]domain.TariffSchedule, error) {
	q := `SELECT ` + scheduleCols + `, COUNT(te.id) AS entry_count
	      FROM tariff_schedules ts
	      LEFT JOIN tariff_entries te ON te.schedule_id = ts.id
	      WHERE ts.eeg_id = $1
	      GROUP BY ts.id
	      ORDER BY ts.is_active DESC, ts.created_at DESC`
	return r.listSchedules(ctx, q, eegID)
}

// ListByMember returns the tariff schedules (Individualtarife) specific to one member.
func (r *TariffRepository) ListByMember(ctx context.Context, eegID, memberID uuid.UUID) ([]domain.TariffSchedule, error) {
	q := `SELECT ` + scheduleCols + `, COUNT(te.id) AS entry_count
	      FROM tariff_schedules ts
	      LEFT JOIN tariff_entries te ON te.schedule_id = ts.id
	      WHERE ts.eeg_id = $1 AND ts.member_id = $2
	      GROUP BY ts.id
	      ORDER BY ts.is_active DESC, ts.created_at DESC`
	return r.listSchedules(ctx, q, eegID, memberID)
}

func (r *TariffRepository) listSchedules(ctx context.Context, q string, args ...any) ([]domain.TariffSchedule, error) {
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var result []domain.TariffSchedule
	for rows.Next() {
		var s domain.TariffSchedule
		if err := rows.Scan(&s.ID, &s.EegID, &s.MemberID, &s.Name, &s.Granularity, &s.IsActive, &s.CreatedAt,
			&s.FreeKwhOverride, &s.DiscountPctOverride, &s.MeterFeeEurOverride,
			&s.ParticipationFeeEurOverride, &s.ZaehlpunktsGebuehrOverride, &s.EntryCount); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *TariffRepository) GetWithEntries(ctx context.Context, id uuid.UUID) (*domain.TariffSchedule, error) {
	var s domain.TariffSchedule
	err := scanSchedule(r.db.QueryRow(ctx, `SELECT `+scheduleCols+` FROM tariff_schedules ts WHERE ts.id = $1`, id), &s)
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
		`INSERT INTO tariff_schedules
		   (eeg_id, member_id, name, granularity, is_active,
		    free_kwh_override, discount_pct_override, meter_fee_eur_override,
		    participation_fee_eur_override, zaehlpunkts_gebuehr_eur_override)
		 VALUES ($1, $2, $3, $4, false, $5, $6, $7, $8, $9)
		 RETURNING id, created_at`,
		s.EegID, s.MemberID, s.Name, s.Granularity,
		s.FreeKwhOverride, s.DiscountPctOverride, s.MeterFeeEurOverride,
		s.ParticipationFeeEurOverride, s.ZaehlpunktsGebuehrOverride).
		Scan(&s.ID, &s.CreatedAt)
}

func (r *TariffRepository) Update(ctx context.Context, s *domain.TariffSchedule) error {
	_, err := r.db.Exec(ctx,
		`UPDATE tariff_schedules SET
		   name = $2, granularity = $3,
		   free_kwh_override = $4, discount_pct_override = $5, meter_fee_eur_override = $6,
		   participation_fee_eur_override = $7, zaehlpunkts_gebuehr_eur_override = $8
		 WHERE id = $1`,
		s.ID, s.Name, s.Granularity,
		s.FreeKwhOverride, s.DiscountPctOverride, s.MeterFeeEurOverride,
		s.ParticipationFeeEurOverride, s.ZaehlpunktsGebuehrOverride)
	return err
}

func (r *TariffRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM tariff_schedules WHERE id = $1`, id)
	return err
}

// Activate sets one schedule as active, deactivating all other schedules in the same scope
// (the EEG-wide default scope when memberID is nil, or that member's own scope otherwise).
func (r *TariffRepository) Activate(ctx context.Context, scheduleID, eegID uuid.UUID, memberID *uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`UPDATE tariff_schedules SET is_active = false WHERE eeg_id = $1 AND member_id IS NOT DISTINCT FROM $2`,
		eegID, memberID); err != nil {
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

// GetAllActiveForEEG returns the active EEG-wide default schedule and a map of active
// member-specific schedules (Individualtarife), each with entries overlapping the billing period.
// Single query — avoids N+1 lookups per member during billing.
func (r *TariffRepository) GetAllActiveForEEG(ctx context.Context, eegID uuid.UUID, periodStart, periodEnd time.Time) (*domain.TariffSchedule, map[uuid.UUID]*domain.TariffSchedule, error) {
	q := `SELECT ` + scheduleCols + `,
	             te.id, te.schedule_id, te.valid_from, te.valid_until, te.energy_price, te.producer_price, te.created_at
	      FROM tariff_schedules ts
	      LEFT JOIN tariff_entries te ON te.schedule_id = ts.id
	        AND te.valid_from < $3 AND te.valid_until > $2
	      WHERE ts.eeg_id = $1 AND ts.is_active = true
	      ORDER BY ts.id, te.valid_from`
	rows, err := r.db.Query(ctx, q, eegID, periodStart, periodEnd)
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	schedules := make(map[uuid.UUID]*domain.TariffSchedule)
	var order []uuid.UUID
	for rows.Next() {
		var s domain.TariffSchedule
		var e domain.TariffEntry
		var entryID, entryScheduleID *uuid.UUID
		var validFrom, validUntil, entryCreatedAt *time.Time
		var energyPrice, producerPrice *float64
		if err := rows.Scan(&s.ID, &s.EegID, &s.MemberID, &s.Name, &s.Granularity, &s.IsActive, &s.CreatedAt,
			&s.FreeKwhOverride, &s.DiscountPctOverride, &s.MeterFeeEurOverride,
			&s.ParticipationFeeEurOverride, &s.ZaehlpunktsGebuehrOverride,
			&entryID, &entryScheduleID, &validFrom, &validUntil, &energyPrice, &producerPrice, &entryCreatedAt); err != nil {
			return nil, nil, fmt.Errorf("scan: %w", err)
		}
		existing, ok := schedules[s.ID]
		if !ok {
			schedules[s.ID] = &s
			existing = &s
			order = append(order, s.ID)
		}
		if entryID != nil {
			e.ID, e.ScheduleID, e.ValidFrom, e.ValidUntil, e.EnergyPrice, e.ProducerPrice, e.CreatedAt =
				*entryID, *entryScheduleID, *validFrom, *validUntil, *energyPrice, *producerPrice, *entryCreatedAt
			existing.Entries = append(existing.Entries, e)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var defaultSchedule *domain.TariffSchedule
	memberSchedules := make(map[uuid.UUID]*domain.TariffSchedule)
	for _, id := range order {
		s := schedules[id]
		s.EntryCount = len(s.Entries)
		if s.MemberID == nil {
			defaultSchedule = s
		} else {
			memberSchedules[*s.MemberID] = s
		}
	}
	return defaultSchedule, memberSchedules, nil
}
