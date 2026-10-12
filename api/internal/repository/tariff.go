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
	ts.participation_fee_eur_override, ts.zaehlpunkts_gebuehr_eur_override,
	ts.servicegebuehr_bezug_ct_kwh_override, ts.servicegebuehr_einspeisung_ct_kwh_override, ts.is_default`

func scanSchedule(row pgx.Row, s *domain.TariffSchedule) error {
	return row.Scan(&s.ID, &s.EegID, &s.MemberID, &s.Name, &s.Granularity, &s.IsActive, &s.CreatedAt,
		&s.FreeKwhOverride, &s.DiscountPctOverride, &s.MeterFeeEurOverride,
		&s.ParticipationFeeEurOverride, &s.ZaehlpunktsGebuehrOverride,
		&s.ServicegebuehrBezugCtKwhOverride, &s.ServicegebuehrEinspeisungCtKwhOverride, &s.IsDefault)
}

// ListByEeg returns the EEG-wide (non-member-specific) tariff schedules.
func (r *TariffRepository) ListByEeg(ctx context.Context, eegID uuid.UUID) ([]domain.TariffSchedule, error) {
	q := `SELECT ` + scheduleCols + `, COUNT(te.id) AS entry_count, (SELECT COUNT(*) FROM members m WHERE m.tariff_schedule_id = ts.id) AS assigned_member_count
	      FROM tariff_schedules ts
	      LEFT JOIN tariff_entries te ON te.schedule_id = ts.id
	      WHERE ts.eeg_id = $1 AND ts.member_id IS NULL
	      GROUP BY ts.id
	      ORDER BY ts.is_default DESC, ts.is_active DESC, ts.created_at DESC`
	return r.listSchedules(ctx, q, eegID)
}

// ListAllByEeg returns every schedule for the EEG — default and member-specific alike.
// Used by backup export, which needs a complete snapshot regardless of scope.
func (r *TariffRepository) ListAllByEeg(ctx context.Context, eegID uuid.UUID) ([]domain.TariffSchedule, error) {
	q := `SELECT ` + scheduleCols + `, COUNT(te.id) AS entry_count, (SELECT COUNT(*) FROM members m WHERE m.tariff_schedule_id = ts.id) AS assigned_member_count
	      FROM tariff_schedules ts
	      LEFT JOIN tariff_entries te ON te.schedule_id = ts.id
	      WHERE ts.eeg_id = $1
	      GROUP BY ts.id
	      ORDER BY ts.is_default DESC, ts.is_active DESC, ts.created_at DESC`
	return r.listSchedules(ctx, q, eegID)
}

// ListByMember returns the tariff schedules (Individualtarife) specific to one member.
func (r *TariffRepository) ListByMember(ctx context.Context, eegID, memberID uuid.UUID) ([]domain.TariffSchedule, error) {
	q := `SELECT ` + scheduleCols + `, COUNT(te.id) AS entry_count, (SELECT COUNT(*) FROM members m WHERE m.tariff_schedule_id = ts.id) AS assigned_member_count
	      FROM tariff_schedules ts
	      LEFT JOIN tariff_entries te ON te.schedule_id = ts.id
	      WHERE ts.eeg_id = $1 AND ts.member_id = $2
	      GROUP BY ts.id
	      ORDER BY ts.is_default DESC, ts.is_active DESC, ts.created_at DESC`
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
			&s.ParticipationFeeEurOverride, &s.ZaehlpunktsGebuehrOverride,
			&s.ServicegebuehrBezugCtKwhOverride, &s.ServicegebuehrEinspeisungCtKwhOverride, &s.IsDefault,
			&s.EntryCount, &s.AssignedMemberCount); err != nil {
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
		    participation_fee_eur_override, zaehlpunkts_gebuehr_eur_override,
		    servicegebuehr_bezug_ct_kwh_override, servicegebuehr_einspeisung_ct_kwh_override)
		 VALUES ($1, $2, $3, $4, false, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id, created_at`,
		s.EegID, s.MemberID, s.Name, s.Granularity,
		s.FreeKwhOverride, s.DiscountPctOverride, s.MeterFeeEurOverride,
		s.ParticipationFeeEurOverride, s.ZaehlpunktsGebuehrOverride,
		s.ServicegebuehrBezugCtKwhOverride, s.ServicegebuehrEinspeisungCtKwhOverride).
		Scan(&s.ID, &s.CreatedAt)
}

func (r *TariffRepository) Update(ctx context.Context, s *domain.TariffSchedule) error {
	_, err := r.db.Exec(ctx,
		`UPDATE tariff_schedules SET
		   name = $2, granularity = $3,
		   free_kwh_override = $4, discount_pct_override = $5, meter_fee_eur_override = $6,
		   participation_fee_eur_override = $7, zaehlpunkts_gebuehr_eur_override = $8,
		   servicegebuehr_bezug_ct_kwh_override = $9, servicegebuehr_einspeisung_ct_kwh_override = $10
		 WHERE id = $1`,
		s.ID, s.Name, s.Granularity,
		s.FreeKwhOverride, s.DiscountPctOverride, s.MeterFeeEurOverride,
		s.ParticipationFeeEurOverride, s.ZaehlpunktsGebuehrOverride,
		s.ServicegebuehrBezugCtKwhOverride, s.ServicegebuehrEinspeisungCtKwhOverride)
	return err
}

func (r *TariffRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM tariff_schedules WHERE id = $1`, id)
	return err
}

// Activate marks a schedule as active. For a member-specific Individualtarif (memberID != nil)
// all other schedules of that member are deactivated first (one active per member). EEG-wide
// plans may be active in parallel, so nothing else is touched there.
func (r *TariffRepository) Activate(ctx context.Context, scheduleID, eegID uuid.UUID, memberID *uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if memberID != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE tariff_schedules SET is_active = false WHERE eeg_id = $1 AND member_id = $2`,
			eegID, *memberID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE tariff_schedules SET is_active = true WHERE id = $1`, scheduleID); err != nil {
		return err
	}
	if memberID == nil {
		// An EEG without a Standardtarif adopts the first plan that gets activated.
		if _, err := tx.Exec(ctx,
			`UPDATE tariff_schedules SET is_default = true
			 WHERE id = $1 AND NOT EXISTS (
			   SELECT 1 FROM tariff_schedules WHERE eeg_id = $2 AND member_id IS NULL AND is_default)`,
			scheduleID, eegID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// SetDefault makes an EEG-wide schedule the Standardtarif (and active), clearing the flag on all others.
func (r *TariffRepository) SetDefault(ctx context.Context, scheduleID, eegID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`UPDATE tariff_schedules SET is_default = false WHERE eeg_id = $1 AND member_id IS NULL AND is_default`,
		eegID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE tariff_schedules SET is_default = true, is_active = true,
		   free_kwh_override = NULL, discount_pct_override = NULL, meter_fee_eur_override = NULL,
		   participation_fee_eur_override = NULL, zaehlpunkts_gebuehr_eur_override = NULL,
		   servicegebuehr_bezug_ct_kwh_override = NULL, servicegebuehr_einspeisung_ct_kwh_override = NULL
		 WHERE id = $1`, scheduleID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *TariffRepository) Deactivate(ctx context.Context, scheduleID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE tariff_schedules SET is_active = false WHERE id = $1`, scheduleID)
	return err
}

// CountAssignedMembers returns how many members are explicitly assigned to an EEG-wide schedule.
func (r *TariffRepository) CountAssignedMembers(ctx context.Context, scheduleID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM members WHERE tariff_schedule_id = $1`, scheduleID).Scan(&n)
	return n, err
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

// LoadBillingTariffs returns every active schedule of the EEG (the Standardtarif, all other
// active EEG-wide plans and the active member Individualtarife) with the entries overlapping the
// billing period, plus the member → plan assignments. Single set of queries — avoids N+1 lookups
// per member during billing.
func (r *TariffRepository) LoadBillingTariffs(ctx context.Context, eegID uuid.UUID, periodStart, periodEnd time.Time) (*domain.BillingTariffs, error) {
	q := `SELECT ` + scheduleCols + `,
	             te.id, te.schedule_id, te.valid_from, te.valid_until, te.energy_price, te.producer_price, te.created_at
	      FROM tariff_schedules ts
	      LEFT JOIN tariff_entries te ON te.schedule_id = ts.id
	        AND te.valid_from < $3 AND te.valid_until > $2
	      WHERE ts.eeg_id = $1 AND ts.is_active = true
	      ORDER BY ts.id, te.valid_from`
	rows, err := r.db.Query(ctx, q, eegID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
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
			&s.ServicegebuehrBezugCtKwhOverride, &s.ServicegebuehrEinspeisungCtKwhOverride, &s.IsDefault,
			&entryID, &entryScheduleID, &validFrom, &validUntil, &energyPrice, &producerPrice, &entryCreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
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
		return nil, err
	}

	bt := &domain.BillingTariffs{
		Shared:     make(map[uuid.UUID]*domain.TariffSchedule),
		Individual: make(map[uuid.UUID]*domain.TariffSchedule),
		Assignment: make(map[uuid.UUID]uuid.UUID),
	}
	for _, id := range order {
		s := schedules[id]
		s.EntryCount = len(s.Entries)
		switch {
		case s.MemberID != nil:
			bt.Individual[*s.MemberID] = s
		case s.IsDefault:
			bt.Default = s
			bt.Shared[s.ID] = s
		default:
			bt.Shared[s.ID] = s
		}
	}

	arows, err := r.db.Query(ctx,
		`SELECT id, tariff_schedule_id FROM members WHERE eeg_id = $1 AND tariff_schedule_id IS NOT NULL`, eegID)
	if err != nil {
		return nil, fmt.Errorf("query assignments: %w", err)
	}
	defer arows.Close()
	for arows.Next() {
		var memberID, scheduleID uuid.UUID
		if err := arows.Scan(&memberID, &scheduleID); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		bt.Assignment[memberID] = scheduleID
	}
	return bt, arows.Err()
}
