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

// MeterPointSum holds raw energy for one meter point within a billing period.
// Direction is "CONSUMPTION" or "GENERATION".
// RawKwh is wh_self for CONSUMPTION ZPs and wh_community (unscaled) for GENERATION ZPs.
type MeterPointSum struct {
	Zaehlpunkt string
	Direction  string
	RawKwh     float64
}

// SumByMeterPointForMember returns raw energy per meter point for a member.
// Used to build per-ZP kWh breakdown on invoices.
func (r *ReadingRepository) SumByMeterPointForMember(ctx context.Context, memberID uuid.UUID, from, to time.Time) ([]MeterPointSum, error) {
	q := `
		SELECT mp.zaehlpunkt, mp.energierichtung,
		       CASE WHEN mp.energierichtung = 'CONSUMPTION'
		            THEN COALESCE(SUM(er.wh_self), 0)
		            ELSE COALESCE(SUM(er.wh_community), 0)
		       END AS raw_kwh
		FROM meter_points mp
		JOIN energy_readings er ON er.meter_point_id = mp.id
		WHERE mp.member_id = $1
		  AND er.ts >= $2
		  AND er.ts <= $3
		  AND er.quality <> 'L3'
		GROUP BY mp.zaehlpunkt, mp.energierichtung
		ORDER BY mp.energierichtung, mp.zaehlpunkt
	`
	rows, err := r.db.Query(ctx, q, memberID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var sums []MeterPointSum
	for rows.Next() {
		var s MeterPointSum
		if err := rows.Scan(&s.Zaehlpunkt, &s.Direction, &s.RawKwh); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		sums = append(sums, s)
	}
	return sums, rows.Err()
}

// SumByMemberAndPeriod returns consumption and generation community kWh per member
// for a billing period, split by meter point direction.
// Readings with quality = 'L3' (faulty values) are excluded from billing.
//
// Both directions use the Netzbetreiber's own values verbatim, with no cross-member
// normalization: CONSUMPTION members are billed their raw wh_self (Eigendeckung —
// the energy the NB reports as actually drawn from the community pool), and
// GENERATION members are credited their raw wh_community (Einspeisung in die EEG)
// exactly as reported. Any EEG-wide imbalance between the two totals reflects the
// Netzbetreiber's own allocation data (or its current settlement state) and is
// intentionally left visible rather than smoothed over — see the Reports page for
// the same raw totals.
func (r *ReadingRepository) SumByMemberAndPeriod(ctx context.Context, eegID uuid.UUID, start, end time.Time) ([]MemberSum, error) {
	q := `
		SELECT
			m.id AS member_id,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self      ELSE 0 END), 0) AS consumption_kwh,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_community ELSE 0 END), 0) AS generation_kwh
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

// SumForMember returns the raw consumption and generation community kWh for a single member
// within the given period — same field semantics as SumByMemberAndPeriod (no cross-member
// normalization), just scoped to one member. Used when a member's effective billing period
// differs from the billing run period due to beitritt_datum or austritt_datum falling within
// the period.
func (r *ReadingRepository) SumForMember(ctx context.Context, eegID, memberID uuid.UUID, start, end time.Time) (MemberSum, error) {
	q := `
		SELECT
			$4::uuid AS member_id,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self      ELSE 0 END), 0) AS consumption_kwh,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_community ELSE 0 END), 0) AS generation_kwh
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

// TOUSum holds time-of-use priced energy for one member within a window:
// every 15-min reading is multiplied with the tariff entry covering its own
// timestamp. Covered* report how many kWh fell inside any tariff entry, so the
// caller can price the uncovered remainder with the flat fallback prices.
type TOUSum struct {
	ConsumptionEur        float64 // Σ wh_self × entry.energy_price / 100 (CONSUMPTION ZPs)
	GenerationEur         float64 // Σ wh_community × entry.producer_price / 100 (GENERATION ZPs)
	CoveredConsumptionKwh float64
	CoveredGenerationKwh  float64
}

// SumTOUForMember prices a member's readings per tariff entry (true time-of-use):
// each reading in [start, end) is matched to the tariff_entries row of the given
// schedule whose [valid_from, valid_until) contains its timestamp. Readings outside
// every entry are not included — the caller compares Covered* against the raw
// period totals and prices the remainder with the flat EEG fallback.
// Overlapping entries would price a reading once per overlap (the schema does not
// prevent overlaps yet — same caveat as the previous time-weighted blend).
func (r *ReadingRepository) SumTOUForMember(ctx context.Context, scheduleID, memberID uuid.UUID, start, end time.Time) (TOUSum, error) {
	q := `
		SELECT
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self      * te.energy_price   ELSE 0 END), 0) / 100 AS consumption_eur,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_community * te.producer_price ELSE 0 END), 0) / 100 AS generation_eur,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self      ELSE 0 END), 0) AS covered_consumption_kwh,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_community ELSE 0 END), 0) AS covered_generation_kwh
		FROM tariff_entries te
		JOIN meter_points mp ON mp.member_id = $2
		JOIN energy_readings er ON er.meter_point_id = mp.id
			AND er.ts >= te.valid_from AND er.ts < te.valid_until
			AND er.ts >= $3 AND er.ts < $4
			AND er.quality <> 'L3'
		WHERE te.schedule_id = $1
	`
	var s TOUSum
	if err := r.db.QueryRow(ctx, q, scheduleID, memberID, start, end).Scan(
		&s.ConsumptionEur, &s.GenerationEur, &s.CoveredConsumptionKwh, &s.CoveredGenerationKwh,
	); err != nil {
		return TOUSum{}, fmt.Errorf("tou sum query: %w", err)
	}
	return s, nil
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
			(date_trunc('month', er.ts AT TIME ZONE 'Europe/Vienna'))::timestamptz AS month,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self      ELSE 0 END), 0) AS consumption_kwh,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_community ELSE 0 END), 0) AS generation_kwh
		FROM meter_points mp
		JOIN energy_readings er ON er.meter_point_id = mp.id
		WHERE mp.member_id = $1
		  AND er.ts >= $2
		  AND er.ts <= $3
		  AND er.quality <> 'L3'
		GROUP BY (date_trunc('month', er.ts AT TIME ZONE 'Europe/Vienna'))::timestamptz
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

// MonthlyZPKwh holds one Zählpunkt's monthly consumption breakdown: WhTotal is
// the meter's total Gesamtverbrauch (Netzbezug + community-covered), WhSelf is
// the community-covered share already billed as "Bezug Strom" — the caller
// derives Netzbezug = WhTotal - WhSelf. CONSUMPTION meter points only.
type MonthlyZPKwh struct {
	Zaehlpunkt string
	Month      time.Time
	WhSelf     float64
	WhTotal    float64
}

// MonthlyEnergyByMeterPointForMember returns, for every CONSUMPTION meter point
// of a member, the monthly WhSelf (community-covered consumption, matches
// MonthlySummaryForMember's ConsumptionKwh) alongside WhTotal (Gesamtverbrauch) —
// used to render the Netzbezug/Community-Verbrauch breakdown table on the
// "individuell" invoice design (see invoice.EnergyPeriodRow).
func (r *ReadingRepository) MonthlyEnergyByMeterPointForMember(ctx context.Context, memberID uuid.UUID, from, to time.Time) ([]MonthlyZPKwh, error) {
	q := `
		SELECT
			mp.zaehlpunkt,
			(date_trunc('month', er.ts AT TIME ZONE 'Europe/Vienna'))::timestamptz AS month,
			COALESCE(SUM(er.wh_self), 0)  AS wh_self,
			COALESCE(SUM(er.wh_total), 0) AS wh_total
		FROM meter_points mp
		JOIN energy_readings er ON er.meter_point_id = mp.id
		WHERE mp.member_id = $1
		  AND mp.energierichtung = 'CONSUMPTION'
		  AND er.ts >= $2
		  AND er.ts <= $3
		  AND er.quality <> 'L3'
		GROUP BY mp.zaehlpunkt, (date_trunc('month', er.ts AT TIME ZONE 'Europe/Vienna'))::timestamptz
		ORDER BY mp.zaehlpunkt, month ASC
	`
	rows, err := r.db.Query(ctx, q, memberID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var result []MonthlyZPKwh
	for rows.Next() {
		var m MonthlyZPKwh
		if err := rows.Scan(&m.Zaehlpunkt, &m.Month, &m.WhSelf, &m.WhTotal); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// MonthlyGenerationZPKwh holds one Zählpunkt's monthly generation breakdown:
// WhTotal is the meter's total Gesamteinspeisung (feed-in), WhCommunity is the
// share absorbed/purchased by the energy community and already billed as
// "Einspeisung Strom" — the caller derives Resteinspeisung = WhTotal -
// WhCommunity. GENERATION meter points only.
type MonthlyGenerationZPKwh struct {
	Zaehlpunkt  string
	Month       time.Time
	WhCommunity float64
	WhTotal     float64
}

// MonthlyGenerationByMeterPointForMember returns, for every GENERATION meter
// point of a member, the monthly WhCommunity (community-absorbed generation,
// matches MonthlySummaryForMember's GenerationKwh) alongside WhTotal
// (Gesamteinspeisung) — used to render the Gesamteinspeisung/Abnahme durch
// Energiegemeinschaft/Resteinspeisung breakdown table on the "individuell"
// invoice design (see invoice.GenerationPeriodRow).
func (r *ReadingRepository) MonthlyGenerationByMeterPointForMember(ctx context.Context, memberID uuid.UUID, from, to time.Time) ([]MonthlyGenerationZPKwh, error) {
	q := `
		SELECT
			mp.zaehlpunkt,
			(date_trunc('month', er.ts AT TIME ZONE 'Europe/Vienna'))::timestamptz AS month,
			COALESCE(SUM(er.wh_community), 0) AS wh_community,
			COALESCE(SUM(er.wh_total), 0)      AS wh_total
		FROM meter_points mp
		JOIN energy_readings er ON er.meter_point_id = mp.id
		WHERE mp.member_id = $1
		  AND mp.energierichtung = 'GENERATION'
		  AND er.ts >= $2
		  AND er.ts <= $3
		  AND er.quality <> 'L3'
		GROUP BY mp.zaehlpunkt, (date_trunc('month', er.ts AT TIME ZONE 'Europe/Vienna'))::timestamptz
		ORDER BY mp.zaehlpunkt, month ASC
	`
	rows, err := r.db.Query(ctx, q, memberID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var result []MonthlyGenerationZPKwh
	for rows.Next() {
		var m MonthlyGenerationZPKwh
		if err := rows.Scan(&m.Zaehlpunkt, &m.Month, &m.WhCommunity, &m.WhTotal); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// MeterPointGapDetail holds the incomplete-days list for one meter point.
type MeterPointGapDetail struct {
	Zaehlpunkt  string   `json:"zaehlpunkt"`
	MissingDays []string `json:"missing_days"` // YYYY-MM-DD (Vienna date), days with <96 L1/L2 slots
}

// MissingIntervalDetails returns per-ZP gap details: for every active meter point
// that has at least one incomplete day, it returns the list of dates (Vienna time,
// YYYY-MM-DD) where fewer than 96 L1/L2 slots were found.
// Days with exactly 92 slots (DST spring-forward) or 100 slots (fall-back) will
// still appear as incomplete — the caller may display a note about DST.
//
// Meter points with a pending/sent/first_confirmed EC_REQ_ONL EDA process are
// excluded: their registriert_seit is set at onboarding-convert time but energy
// data won't arrive until the Netzbetreiber confirms the Anmeldung.
func (r *ReadingRepository) MissingIntervalDetails(ctx context.Context, eegID uuid.UUID, periodStart, periodEnd time.Time) ([]MeterPointGapDetail, error) {
	q := `
		WITH active_mps AS (
		  SELECT mp.id, mp.zaehlpunkt,
		         GREATEST(timezone('Europe/Vienna', mp.registriert_seit::timestamp), $2::timestamptz) AS effective_start,
		         LEAST(
		           COALESCE(timezone('Europe/Vienna', mp.abgemeldet_am::timestamp), 'infinity'::timestamptz),
		           $3::timestamptz + interval '1 second'
		         ) AS effective_end
		  FROM meter_points mp
		  JOIN members m ON m.id = mp.member_id
		  WHERE m.eeg_id = $1
		    AND m.status = 'ACTIVE'
		    AND mp.registriert_seit IS NOT NULL
		    AND (mp.abgemeldet_am IS NULL OR timezone('Europe/Vienna', mp.abgemeldet_am::timestamp) > $2)
		    AND NOT EXISTS (
		      SELECT 1 FROM eda_processes ep
		      WHERE ep.meter_point_id = mp.id
		        AND ep.process_type = 'EC_REQ_ONL'
		        AND ep.status IN ('pending', 'sent', 'first_confirmed')
		    )
		),
		expected_days AS (
		  SELECT amp.id AS mp_id, amp.zaehlpunkt,
		         generate_series(
		           (amp.effective_start AT TIME ZONE 'Europe/Vienna')::date,
		           ((amp.effective_end - interval '1 second') AT TIME ZONE 'Europe/Vienna')::date,
		           '1 day'::interval
		         )::date AS day
		  FROM active_mps amp
		  WHERE amp.effective_start < amp.effective_end
		),
		actual_per_day AS (
		  SELECT amp.id AS mp_id, amp.zaehlpunkt,
		         (er.ts AT TIME ZONE 'Europe/Vienna')::date AS day,
		         COUNT(DISTINCT er.ts) AS slot_count
		  FROM active_mps amp
		  JOIN energy_readings er ON er.meter_point_id = amp.id
		    AND er.ts >= amp.effective_start
		    AND er.ts < amp.effective_end
		    AND er.quality IN ('L1', 'L2')
		  GROUP BY amp.id, amp.zaehlpunkt, (er.ts AT TIME ZONE 'Europe/Vienna')::date
		)
		SELECT ed.zaehlpunkt, ed.day::text
		FROM expected_days ed
		LEFT JOIN actual_per_day ap ON ap.mp_id = ed.mp_id AND ap.day = ed.day
		WHERE COALESCE(ap.slot_count, 0) < 96
		ORDER BY ed.zaehlpunkt, ed.day
	`
	rows, err := r.db.Query(ctx, q, eegID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("missing interval details query: %w", err)
	}
	defer rows.Close()

	byZP := map[string]*MeterPointGapDetail{}
	var order []string
	for rows.Next() {
		var zp, day string
		if err := rows.Scan(&zp, &day); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if _, ok := byZP[zp]; !ok {
			byZP[zp] = &MeterPointGapDetail{Zaehlpunkt: zp}
			order = append(order, zp)
		}
		byZP[zp].MissingDays = append(byZP[zp].MissingDays, day)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]MeterPointGapDetail, len(order))
	for i, zp := range order {
		result[i] = *byZP[zp]
	}
	return result, nil
}

// MissingIntervals returns the Zählpunkt identifiers of active meter points that
// are missing at least one expected 15-minute slot with quality L1 or L2 in
// [periodStart, periodEnd].
//
// The expected slot count per ZP is derived from its effective active window:
//   - effective_start = max(registriert_seit, periodStart)
//   - effective_end   = min(abgemeldet_am (exclusive), periodEnd+1s)
//
// Active = registriert_seit IS NOT NULL AND active window overlaps the period.
// Meter points with a pending/sent/first_confirmed EC_REQ_ONL are excluded (same
// reasoning as MissingIntervalDetails).
func (r *ReadingRepository) MissingIntervals(ctx context.Context, eegID uuid.UUID, periodStart, periodEnd time.Time) ([]string, error) {
	q := `
		WITH active_mps AS (
		  SELECT mp.id, mp.zaehlpunkt,
		         GREATEST(timezone('Europe/Vienna', mp.registriert_seit::timestamp), $2::timestamptz) AS effective_start,
		         LEAST(
		           COALESCE(timezone('Europe/Vienna', mp.abgemeldet_am::timestamp), 'infinity'::timestamptz),
		           $3::timestamptz + interval '1 second'
		         ) AS effective_end
		  FROM meter_points mp
		  JOIN members m ON m.id = mp.member_id
		  WHERE m.eeg_id = $1
		    AND m.status = 'ACTIVE'
		    AND mp.registriert_seit IS NOT NULL
		    AND (mp.abgemeldet_am IS NULL OR timezone('Europe/Vienna', mp.abgemeldet_am::timestamp) > $2)
		    AND NOT EXISTS (
		      SELECT 1 FROM eda_processes ep
		      WHERE ep.meter_point_id = mp.id
		        AND ep.process_type = 'EC_REQ_ONL'
		        AND ep.status IN ('pending', 'sent', 'first_confirmed')
		    )
		),
		expected AS (
		  SELECT id AS mp_id, zaehlpunkt,
		         FLOOR(EXTRACT(EPOCH FROM (effective_end - effective_start)) / 900)::bigint AS expected_count
		  FROM active_mps
		  WHERE effective_start < effective_end
		),
		actual AS (
		  SELECT amp.id AS mp_id, amp.zaehlpunkt,
		         COUNT(DISTINCT er.ts) AS actual_count
		  FROM active_mps amp
		  JOIN energy_readings er ON er.meter_point_id = amp.id
		    AND er.ts >= amp.effective_start
		    AND er.ts < amp.effective_end
		    AND er.quality IN ('L1', 'L2')
		  GROUP BY amp.id, amp.zaehlpunkt
		)
		SELECT DISTINCT e.zaehlpunkt
		FROM expected e
		LEFT JOIN actual a ON a.mp_id = e.mp_id
		WHERE COALESCE(a.actual_count, 0) < e.expected_count
		ORDER BY e.zaehlpunkt
	`
	rows, err := r.db.Query(ctx, q, eegID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("missing intervals query: %w", err)
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
		         AND er.quality IN ('L1', 'L2')
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
