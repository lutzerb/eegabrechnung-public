package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// CoverageDay holds the reading count for a single calendar day.
type CoverageDay struct {
	Date  time.Time `json:"date"`
	Count int       `json:"count"`
}

type ReadingRepository struct {
	db *pgxpool.Pool
}

func NewReadingRepository(db *pgxpool.Pool) *ReadingRepository {
	return &ReadingRepository{db: db}
}

// BulkUpsert inserts energy readings, updating on conflict.
func (r *ReadingRepository) BulkUpsert(ctx context.Context, readings []domain.EnergyReading) (int, error) {
	if len(readings) == 0 {
		return 0, nil
	}

	inserted := 0
	for _, rd := range readings {
		src := rd.Source
		if src == "" {
			src = "xlsx"
		}
		quality := rd.Quality
		if quality == "" {
			quality = "L0"
		}
		q := `INSERT INTO energy_readings (meter_point_id, ts, wh_total, wh_community, wh_self, source, quality)
		      VALUES ($1, $2, $3, $4, $5, $6, $7)
		      ON CONFLICT (meter_point_id, ts) DO UPDATE
		        SET wh_total     = EXCLUDED.wh_total,
		            wh_community = EXCLUDED.wh_community,
		            wh_self      = EXCLUDED.wh_self,
		            source       = EXCLUDED.source,
		            quality      = EXCLUDED.quality`
		if _, e := r.db.Exec(ctx, q, rd.MeterPointID, rd.Ts, rd.WhTotal, rd.WhCommunity, rd.WhSelf, src, quality); e != nil {
			return inserted, fmt.Errorf("upsert: %w", e)
		}
		inserted++
	}
	return inserted, nil
}

// GetInPeriod returns existing readings for the given meter points in the given time range,
// keyed by "meterPointID|ts_unix".
func (r *ReadingRepository) GetInPeriod(ctx context.Context, meterPointIDs []uuid.UUID, start, end time.Time) (map[string]domain.EnergyReading, error) {
	if len(meterPointIDs) == 0 {
		return nil, nil
	}
	q := `SELECT id, meter_point_id, ts, wh_total, wh_community, wh_self, source, quality
	      FROM energy_readings
	      WHERE meter_point_id = ANY($1) AND ts >= $2 AND ts <= $3`
	rows, err := r.db.Query(ctx, q, meterPointIDs, start, end)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	result := make(map[string]domain.EnergyReading)
	for rows.Next() {
		var rd domain.EnergyReading
		if err := rows.Scan(&rd.ID, &rd.MeterPointID, &rd.Ts, &rd.WhTotal, &rd.WhCommunity, &rd.WhSelf, &rd.Source, &rd.Quality); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		key := fmt.Sprintf("%s|%d", rd.MeterPointID, rd.Ts.Unix())
		result[key] = rd
	}
	return result, rows.Err()
}

// BulkInsertSkipExisting inserts readings, skipping rows that already exist (DO NOTHING on conflict).
func (r *ReadingRepository) BulkInsertSkipExisting(ctx context.Context, readings []domain.EnergyReading) (int, error) {
	inserted := 0
	for _, rd := range readings {
		src := rd.Source
		if src == "" {
			src = "xlsx"
		}
		quality := rd.Quality
		if quality == "" {
			quality = "L0"
		}
		q := `INSERT INTO energy_readings (meter_point_id, ts, wh_total, wh_community, wh_self, source, quality)
		      VALUES ($1, $2, $3, $4, $5, $6, $7)
		      ON CONFLICT (meter_point_id, ts) DO NOTHING`
		tag, err := r.db.Exec(ctx, q, rd.MeterPointID, rd.Ts, rd.WhTotal, rd.WhCommunity, rd.WhSelf, src, quality)
		if err != nil {
			return inserted, fmt.Errorf("insert: %w", err)
		}
		inserted += int(tag.RowsAffected())
	}
	return inserted, nil
}

// GetAllByEEG returns all energy readings for an EEG, joined through meter_points.
func (r *ReadingRepository) GetAllByEEG(ctx context.Context, eegID uuid.UUID) ([]domain.EnergyReading, error) {
	q := `SELECT er.id, er.meter_point_id, er.ts, er.wh_total, er.wh_community, er.wh_self, er.source, er.quality
	      FROM energy_readings er
	      JOIN meter_points mp ON mp.id = er.meter_point_id
	      JOIN members m ON m.id = mp.member_id
	      WHERE m.eeg_id = $1
	      ORDER BY er.ts, er.meter_point_id`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var result []domain.EnergyReading
	for rows.Next() {
		var rd domain.EnergyReading
		if err := rows.Scan(&rd.ID, &rd.MeterPointID, &rd.Ts, &rd.WhTotal, &rd.WhCommunity, &rd.WhSelf, &rd.Source, &rd.Quality); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, rd)
	}
	return result, rows.Err()
}

// GetCoverageByYear returns per-day reading counts for a given EEG and year.
func (r *ReadingRepository) GetCoverageByYear(ctx context.Context, eegID uuid.UUID, year int) ([]CoverageDay, error) {
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	q := `SELECT date_trunc('day', er.ts) AS day, COUNT(*) AS cnt
	      FROM energy_readings er
	      JOIN meter_points mp ON mp.id = er.meter_point_id
	      JOIN members m ON m.id = mp.member_id
	      WHERE m.eeg_id = $1 AND er.ts >= $2 AND er.ts < $3
	      GROUP BY day
	      ORDER BY day`
	rows, err := r.db.Query(ctx, q, eegID, start, end)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var result []CoverageDay
	for rows.Next() {
		var cd CoverageDay
		if err := rows.Scan(&cd.Date, &cd.Count); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, cd)
	}
	return result, rows.Err()
}

// MemberSum holds aggregated community energy split by direction for a member.
type MemberSum struct {
	MemberID       uuid.UUID
	ConsumptionKwh float64 // community kWh from CONSUMPTION meter points
	GenerationKwh  float64 // community kWh from GENERATION meter points
}

// SumByMemberAndPeriod returns consumption and generation community kWh per member
// for a billing period, split by meter point direction.
// Readings with quality = 'L3' (faulty values) are excluded from billing.
//
// For CONSUMPTION members: wh_self (Eigendeckung) is used — the actual energy drawn
// from the community pool per 15-min interval (capped at own consumption).
//
// For GENERATION members: their wh_community contribution is scaled by the ratio
// (total_consumer_wh_self / total_generator_wh_community) so that total credited
// generation equals total actual community consumption. Excess generation stays
// with the producers (they sell it themselves outside the EEG).
func (r *ReadingRepository) SumByMemberAndPeriod(ctx context.Context, eegID uuid.UUID, start, end time.Time) ([]MemberSum, error) {
	q := `
		WITH totals AS (
			SELECT
				COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self      ELSE 0 END), 0) AS total_consumer_self,
				COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_community ELSE 0 END), 0) AS total_generator_community
			FROM members m
			JOIN meter_points mp ON mp.member_id = m.id
			JOIN energy_readings er ON er.meter_point_id = mp.id
			WHERE m.eeg_id = $1
			  AND er.ts >= $2
			  AND er.ts <= $3
			  AND er.quality <> 'L3'
		)
		SELECT
			m.id AS member_id,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self ELSE 0 END), 0) AS consumption_kwh,
			CASE
				WHEN (SELECT total_generator_community FROM totals) > 0
				THEN COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION' THEN er.wh_community ELSE 0 END), 0)
				     * (SELECT total_consumer_self FROM totals)
				     / (SELECT total_generator_community FROM totals)
				ELSE 0
			END AS generation_kwh
		FROM members m
		JOIN meter_points mp ON mp.member_id = m.id
		JOIN energy_readings er ON er.meter_point_id = mp.id
		WHERE m.eeg_id = $1
		  AND er.ts >= $2
		  AND er.ts <= $3
		  AND er.quality <> 'L3'
		GROUP BY m.id
	`
	rows, err := r.db.Query(ctx, q, eegID, start, end)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var sums []MemberSum
	for rows.Next() {
		var s MemberSum
		if err := rows.Scan(&s.MemberID, &s.ConsumptionKwh, &s.GenerationKwh); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		sums = append(sums, s)
	}
	return sums, rows.Err()
}

// SumForMember returns the scaled consumption and generation community kWh for a single member
// within the given period. Uses the same EEG-wide scaling logic as SumByMemberAndPeriod
// (total_consumer_self / total_generator_community for the period), but scoped to one member.
// Used when a member's effective billing period differs from the billing run period due to
// beitritt_datum or austritt_datum falling within the period.
func (r *ReadingRepository) SumForMember(ctx context.Context, eegID, memberID uuid.UUID, start, end time.Time) (MemberSum, error) {
	q := `
		WITH totals AS (
			SELECT
				COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self      ELSE 0 END), 0) AS total_consumer_self,
				COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_community ELSE 0 END), 0) AS total_generator_community
			FROM members m
			JOIN meter_points mp ON mp.member_id = m.id
			JOIN energy_readings er ON er.meter_point_id = mp.id
			WHERE m.eeg_id = $1
			  AND er.ts >= $2
			  AND er.ts <= $3
			  AND er.quality <> 'L3'
		)
		SELECT
			$4::uuid AS member_id,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self ELSE 0 END), 0) AS consumption_kwh,
			CASE
				WHEN (SELECT total_generator_community FROM totals) > 0
				THEN COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION' THEN er.wh_community ELSE 0 END), 0)
				     * (SELECT total_consumer_self FROM totals)
				     / (SELECT total_generator_community FROM totals)
				ELSE 0
			END AS generation_kwh
		FROM members m
		JOIN meter_points mp ON mp.member_id = m.id
		JOIN energy_readings er ON er.meter_point_id = mp.id
		WHERE m.eeg_id = $1
		  AND m.id = $4
		  AND er.ts >= $2
		  AND er.ts <= $3
		  AND er.quality <> 'L3'
	`
	var s MemberSum
	if err := r.db.QueryRow(ctx, q, eegID, start, end, memberID).Scan(&s.MemberID, &s.ConsumptionKwh, &s.GenerationKwh); err != nil {
		return MemberSum{}, fmt.Errorf("query: %w", err)
	}
	return s, nil
}

// PortalMonthlyEnergy holds aggregated monthly energy for a member's portal view.
type PortalMonthlyEnergy struct {
	Month              string  `json:"month"`
	WhTotalConsumption float64 `json:"wh_total_consumption"`
	WhCommunity        float64 `json:"wh_community"`
	WhTotalGeneration  float64 `json:"wh_total_generation"`
	WhCommunityGen     float64 `json:"wh_community_gen"`
}

// GetMemberMonthlyEnergy returns monthly aggregated energy for a member's meter points.
func (r *ReadingRepository) GetMemberMonthlyEnergy(ctx context.Context, memberID uuid.UUID) ([]PortalMonthlyEnergy, error) {
	q := `
		SELECT
			TO_CHAR(DATE_TRUNC('month', er.ts), 'YYYY-MM') AS month,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_total     ELSE 0 END), 0) AS wh_total_consumption,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self      ELSE 0 END), 0) AS wh_community,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_total     ELSE 0 END), 0) AS wh_total_generation,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_community ELSE 0 END), 0) AS wh_community_gen
		FROM energy_readings er
		JOIN meter_points mp ON mp.id = er.meter_point_id
		WHERE mp.member_id = $1
		  AND er.quality != 'L3'
		GROUP BY DATE_TRUNC('month', er.ts)
		ORDER BY month DESC
		LIMIT 24
	`
	rows, err := r.db.Query(ctx, q, memberID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var result []PortalMonthlyEnergy
	for rows.Next() {
		var row PortalMonthlyEnergy
		if err := rows.Scan(&row.Month, &row.WhTotalConsumption, &row.WhCommunity, &row.WhTotalGeneration, &row.WhCommunityGen); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, row)
	}
	if result == nil {
		result = []PortalMonthlyEnergy{}
	}
	return result, rows.Err()
}

// PortalEnergyRow holds aggregated energy for a member's portal granular view.
// Period is a pre-formatted string in Vienna local time (e.g. "2025-01", "2025-01-15", "2025-01-15 08:15").
type PortalEnergyRow struct {
	Period             string  `json:"period"`
	WhTotalConsumption float64 `json:"wh_total_consumption"`
	WhCommunity        float64 `json:"wh_community"`
	WhTotalGeneration  float64 `json:"wh_total_generation"`
	WhCommunityGen     float64 `json:"wh_community_gen"`
}

// GetMemberEnergy returns aggregated energy for a member with configurable granularity and date range.
// granularity must be one of: "year", "month", "day", "15min".
// The period label is pre-formatted in Europe/Vienna time to avoid client-side timezone gymnastics.
func (r *ReadingRepository) GetMemberEnergy(ctx context.Context, memberID uuid.UUID, from, to time.Time, granularity string) ([]PortalEnergyRow, error) {
	var periodExpr, periodFmt string
	switch granularity {
	case "15min":
		periodExpr = `date_trunc('hour', er.ts AT TIME ZONE 'Europe/Vienna') + (FLOOR(EXTRACT(MINUTE FROM (er.ts AT TIME ZONE 'Europe/Vienna')) / 15) * INTERVAL '15 minutes')`
		periodFmt = "YYYY-MM-DD HH24:MI"
	case "day":
		periodExpr = `date_trunc('day', er.ts AT TIME ZONE 'Europe/Vienna')`
		periodFmt = "YYYY-MM-DD"
	case "month":
		periodExpr = `date_trunc('month', er.ts AT TIME ZONE 'Europe/Vienna')`
		periodFmt = "YYYY-MM"
	default: // year
		periodExpr = `date_trunc('year', er.ts AT TIME ZONE 'Europe/Vienna')`
		periodFmt = "YYYY"
	}

	q := fmt.Sprintf(`
		SELECT
			TO_CHAR(%s, '%s') AS period,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_total     ELSE 0 END), 0) AS wh_total_consumption,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self      ELSE 0 END), 0) AS wh_community,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_total     ELSE 0 END), 0) AS wh_total_generation,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_community ELSE 0 END), 0) AS wh_community_gen
		FROM meter_points mp
		JOIN energy_readings er ON er.meter_point_id = mp.id
		WHERE mp.member_id = $1
		  AND er.ts >= $2
		  AND er.ts < $3
		  AND er.quality != 'L3'
		GROUP BY %s
		ORDER BY %s ASC`, periodExpr, periodFmt, periodExpr, periodExpr)

	rows, err := r.db.Query(ctx, q, memberID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var result []PortalEnergyRow
	for rows.Next() {
		var row PortalEnergyRow
		if err := rows.Scan(&row.Period, &row.WhTotalConsumption, &row.WhCommunity, &row.WhTotalGeneration, &row.WhCommunityGen); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, row)
	}
	if result == nil {
		result = []PortalEnergyRow{}
	}
	return result, rows.Err()
}

// MonthlyKwh holds aggregated consumption and generation kWh for a single calendar month.
type MonthlyKwh struct {
	Month          time.Time
	ConsumptionKwh float64
	GenerationKwh  float64
}

// MonthlySummaryForMember returns monthly consumption and generation totals for a member
// within the given date range, ordered by month ascending.
// Used for the "Entwicklung" bar chart on invoices.
func (r *ReadingRepository) MonthlySummaryForMember(ctx context.Context, memberID uuid.UUID, from, to time.Time) ([]MonthlyKwh, error) {
	q := `
		SELECT
			date_trunc('month', er.ts) AS month,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self      ELSE 0 END), 0) AS consumption_kwh,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_community ELSE 0 END), 0) AS generation_kwh
		FROM meter_points mp
		JOIN energy_readings er ON er.meter_point_id = mp.id
		WHERE mp.member_id = $1
		  AND er.ts >= $2
		  AND er.ts <= $3
		  AND er.quality <> 'L3'
		GROUP BY date_trunc('month', er.ts)
		ORDER BY month ASC
	`
	rows, err := r.db.Query(ctx, q, memberID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var months []MonthlyKwh
	for rows.Next() {
		var m MonthlyKwh
		if err := rows.Scan(&m.Month, &m.ConsumptionKwh, &m.GenerationKwh); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		months = append(months, m)
	}
	return months, rows.Err()
}

// MissingReadingDays returns the Zählpunkt identifiers of active meter points
// that are missing at least one day of readings in [periodStart, periodEnd).
// A "missing day" means the EEG has readings for that day from other meter points
// but this specific meter point has none. If the EEG has no readings at all for
// some days, those are also reported.
//
// Active = registriert_seit IS NOT NULL AND (abgemeldet_am IS NULL OR abgemeldet_am > periodStart)
func (r *ReadingRepository) MissingReadingDays(ctx context.Context, eegID uuid.UUID, periodStart, periodEnd time.Time) ([]string, error) {
	// Find active meter points for this EEG
	q := `
		WITH active_mps AS (
		  SELECT mp.id, mp.zaehlpunkt
		  FROM meter_points mp
		  JOIN members m ON m.id = mp.member_id
		  WHERE m.eeg_id = $1
		    AND mp.registriert_seit IS NOT NULL
		    AND (mp.abgemeldet_am IS NULL OR mp.abgemeldet_am > $2)
		),
		expected_days AS (
		  SELECT generate_series($2::date, ($3::date - interval '1 day'), interval '1 day')::date AS day
		),
		mp_coverage AS (
		  SELECT mp.zaehlpunkt,
		         COUNT(DISTINCT er.ts::date) AS covered_days
		  FROM active_mps mp
		  LEFT JOIN energy_readings er
		         ON er.meter_point_id = mp.id
		         AND er.ts >= $2
		         AND er.ts < $3
		         AND er.quality <> 'L3'
		  GROUP BY mp.zaehlpunkt
		),
		expected_count AS (
		  SELECT COUNT(*) AS days FROM expected_days
		)
		SELECT mc.zaehlpunkt
		FROM mp_coverage mc, expected_count ec
		WHERE mc.covered_days < ec.days
		ORDER BY mc.zaehlpunkt
	`
	rows, err := r.db.Query(ctx, q, eegID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("missing reading days query: %w", err)
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var zp string
		if err := rows.Scan(&zp); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, zp)
	}
	return result, rows.Err()
}
