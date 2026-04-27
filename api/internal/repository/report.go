package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type ReportRepository struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{db: db}
}

// MonthlyEnergy aggregates invoice data by month for the given year.
func (r *ReportRepository) MonthlyEnergy(ctx context.Context, eegID uuid.UUID, year int) ([]domain.MonthlyEnergyRow, error) {
	q := `
		SELECT
			DATE_TRUNC('month', period_start) AS month,
			COALESCE(SUM(consumption_kwh), 0),
			COALESCE(SUM(generation_kwh), 0),
			COALESCE(SUM(CASE WHEN total_amount > 0 THEN total_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN total_amount < 0 THEN ABS(total_amount) ELSE 0 END), 0)
		FROM invoices
		WHERE eeg_id = $1
		  AND status != 'cancelled'
		  AND EXTRACT(YEAR FROM period_start) = $2
		GROUP BY DATE_TRUNC('month', period_start)
		ORDER BY month`

	rows, err := r.db.Query(ctx, q, eegID, year)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	result := []domain.MonthlyEnergyRow{}
	for rows.Next() {
		var row domain.MonthlyEnergyRow
		if err := rows.Scan(&row.Month, &row.ConsumptionKwh, &row.GenerationKwh, &row.Revenue, &row.Payouts); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// MemberStats aggregates invoice totals per member, optionally filtered by date range.
func (r *ReportRepository) MemberStats(ctx context.Context, eegID uuid.UUID, from, to *time.Time) ([]domain.MemberStat, error) {
	q := `
		SELECT
			member_id,
			COALESCE(SUM(consumption_kwh), 0),
			COALESCE(SUM(generation_kwh), 0),
			COALESCE(SUM(total_amount), 0),
			COUNT(id)
		FROM invoices
		WHERE eeg_id = $1
		  AND status != 'cancelled'
		  AND ($2::timestamptz IS NULL OR period_start >= $2)
		  AND ($3::timestamptz IS NULL OR period_end <= $3)
		GROUP BY member_id
		ORDER BY SUM(total_amount) DESC`

	rows, err := r.db.Query(ctx, q, eegID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	result := []domain.MemberStat{}
	for rows.Next() {
		var s domain.MemberStat
		if err := rows.Scan(&s.MemberID, &s.ConsumptionKwh, &s.GenerationKwh, &s.TotalAmount, &s.InvoiceCount); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// RawMemberEnergy aggregates raw energy_readings per member for a date range.
// Returns the same MemberStat shape as MemberStats, but TotalAmount is always 0
// (no billing calculation – use MemberStats for invoice-based amounts).
// Pass a non-nil memberID to restrict results to a single member.
func (r *ReportRepository) RawMemberEnergy(ctx context.Context, eegID uuid.UUID, from, to time.Time, memberID *uuid.UUID) ([]domain.MemberStat, error) {
	// Field convention (kWh, matches XLSX import):
	//   CONSUMPTION er.wh_self      = EEG share consumed ("Bezug EEG" / "Ausgetauscht") = G.03
	//   CONSUMPTION er.wh_community = allocated EEG energy (G.02, informational)
	//   GENERATION  er.wh_community = EEG share fed in ("Einspeisung EEG")
	//   GENERATION  er.wh_self      = Resteinspeisung ins öffentliche Netz
	q := `
		SELECT
			m.id AS member_id,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self      ELSE 0 END), 0) AS consumption_kwh,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_community ELSE 0 END), 0) AS generation_kwh,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_total     ELSE 0 END), 0) AS consumption_total_kwh,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_total     ELSE 0 END), 0) AS generation_total_kwh,
			0::float8 AS total_amount,
			0::int    AS invoice_count
		FROM members m
		JOIN meter_points mp ON mp.member_id = m.id
		JOIN energy_readings er ON er.meter_point_id = mp.id
		WHERE m.eeg_id = $1
		  AND er.ts >= $2
		  AND er.ts < $3
		  AND er.quality <> 'L3'
		  AND ($4::uuid IS NULL OR m.id = $4)
		GROUP BY m.id
		ORDER BY consumption_kwh DESC`

	rows, err := r.db.Query(ctx, q, eegID, from, to, memberID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	result := []domain.MemberStat{}
	for rows.Next() {
		var s domain.MemberStat
		if err := rows.Scan(&s.MemberID, &s.ConsumptionKwh, &s.GenerationKwh, &s.ConsumptionTotalKwh, &s.GenerationTotalKwh, &s.TotalAmount, &s.InvoiceCount); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// EnergySummary aggregates raw energy metrics from energy_readings for an EEG.
// granularity must be one of: "day", "month", "year", "15min".
// Pass a non-nil memberID to restrict results to a single member.
func (r *ReportRepository) EnergySummary(ctx context.Context, eegID uuid.UUID, from, to time.Time, granularity string, memberID *uuid.UUID) ([]domain.EnergySummaryRow, error) {
	var periodExpr string
	if granularity == "15min" {
		periodExpr = `date_trunc('hour', er.ts AT TIME ZONE 'Europe/Vienna') + (FLOOR(EXTRACT(MINUTE FROM (er.ts AT TIME ZONE 'Europe/Vienna')) / 15) * INTERVAL '15 minutes')`
	} else {
		// granularity is validated against a whitelist in the handler
		periodExpr = fmt.Sprintf("date_trunc('%s', er.ts AT TIME ZONE 'Europe/Vienna')", granularity)
	}

	q := fmt.Sprintf(`
		SELECT
			%s AS period,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self      ELSE 0 END), 0) AS wh_self,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_community ELSE 0 END), 0) AS wh_community,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_total     ELSE 0 END), 0) AS wh_total_consumption,
			COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_total     ELSE 0 END), 0) AS wh_total_generation
		FROM members m
		JOIN meter_points mp ON mp.member_id = m.id
		JOIN energy_readings er ON er.meter_point_id = mp.id
		WHERE m.eeg_id = $1
		  AND er.ts >= $2
		  AND er.ts < $3
		  AND er.quality <> 'L3'
		  AND ($4::uuid IS NULL OR m.id = $4)
		GROUP BY 1
		ORDER BY period ASC`, periodExpr)

	rows, err := r.db.Query(ctx, q, eegID, from, to, memberID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var result []domain.EnergySummaryRow
	for rows.Next() {
		var row domain.EnergySummaryRow
		if err := rows.Scan(&row.Period, &row.WhSelf, &row.WhCommunity, &row.WhTotalConsumption, &row.WhTotalGeneration); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		row.WhRestbedarf = row.WhTotalConsumption - row.WhSelf
		if row.WhRestbedarf < 0 {
			row.WhRestbedarf = 0
		}
		row.WhResteinspeisung = row.WhTotalGeneration - row.WhCommunity
		if row.WhResteinspeisung < 0 {
			row.WhResteinspeisung = 0
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// AnnualReport builds the data for the annual report.
//
// date     — reference date: members active on this day are included in the member list
// from, to — date range for energy readings and invoice totals
func (r *ReportRepository) AnnualReport(ctx context.Context, eegID uuid.UUID, date, from, to time.Time) ([]domain.AnnualReportMember, error) {
	// 1. Members active at the reference date.
	memberRows, err := r.db.Query(ctx, `
		SELECT id, mitglieds_nr, TRIM(name1 || ' ' || name2), email, strasse, plz, ort, member_type,
		       beitritt_datum, austritt_datum
		FROM members
		WHERE eeg_id = $1
		  AND (beitritt_datum IS NULL OR beitritt_datum <= $2)
		  AND (austritt_datum IS NULL OR austritt_datum > $2)
		ORDER BY mitglieds_nr`,
		eegID, date)
	if err != nil {
		return nil, fmt.Errorf("members query: %w", err)
	}
	defer memberRows.Close()

	var members []domain.AnnualReportMember
	memberIdx := map[uuid.UUID]int{}
	for memberRows.Next() {
		var m domain.AnnualReportMember
		if err := memberRows.Scan(
			&m.MemberID, &m.MitgliedsNr, &m.Name, &m.Email, &m.Strasse, &m.Plz, &m.Ort, &m.MemberType,
			&m.BeitrittsDatum, &m.AustrittsDatum,
		); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		memberIdx[m.MemberID] = len(members)
		members = append(members, m)
	}
	if err := memberRows.Err(); err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return members, nil
	}

	ids := make([]uuid.UUID, len(members))
	for i, m := range members {
		ids[i] = m.MemberID
	}

	// 2. Meter points for those members.
	mpRows, err := r.db.Query(ctx, `
		SELECT member_id, zaehlpunkt, energierichtung, generation_type, registriert_seit, abgemeldet_am
		FROM meter_points
		WHERE member_id = ANY($1)
		ORDER BY member_id, zaehlpunkt`,
		ids)
	if err != nil {
		return nil, fmt.Errorf("meter points query: %w", err)
	}
	defer mpRows.Close()
	for mpRows.Next() {
		var memberID uuid.UUID
		var mp domain.AnnualReportMeterPoint
		if err := mpRows.Scan(&memberID, &mp.Zaehlpunkt, &mp.Energierichtung, &mp.GenerationType, &mp.RegistriertSeit, &mp.AbgemeldetAm); err != nil {
			return nil, fmt.Errorf("scan mp: %w", err)
		}
		if idx, ok := memberIdx[memberID]; ok {
			members[idx].Zaehlpunkte = append(members[idx].Zaehlpunkte, mp)
		}
	}
	if err := mpRows.Err(); err != nil {
		return nil, err
	}

	// 3. Energy per member in [from, to).
	energyRows, err := r.db.Query(ctx, `
		SELECT mp.member_id,
		       COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_total     ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_total     ELSE 0 END), 0),
		       COALESCE(SUM(er.wh_community), 0)
		FROM meter_points mp
		JOIN energy_readings er ON er.meter_point_id = mp.id
		WHERE mp.member_id = ANY($1)
		  AND er.ts >= $2 AND er.ts < $3
		  AND er.quality <> 'L3'
		GROUP BY mp.member_id`,
		ids, from, to)
	if err != nil {
		return nil, fmt.Errorf("energy query: %w", err)
	}
	defer energyRows.Close()
	for energyRows.Next() {
		var memberID uuid.UUID
		var wc, wg, wcom float64
		if err := energyRows.Scan(&memberID, &wc, &wg, &wcom); err != nil {
			return nil, fmt.Errorf("scan energy: %w", err)
		}
		if idx, ok := memberIdx[memberID]; ok {
			members[idx].WhConsumption = wc
			members[idx].WhGeneration = wg
			members[idx].WhCommunity = wcom
		}
	}
	if err := energyRows.Err(); err != nil {
		return nil, err
	}

	// 4. Invoice totals per member in [from, to].
	invRows, err := r.db.Query(ctx, `
		SELECT member_id,
		       COALESCE(SUM(CASE WHEN total_amount > 0 THEN total_amount  ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN total_amount < 0 THEN -total_amount ELSE 0 END), 0),
		       COUNT(*)
		FROM invoices
		WHERE eeg_id    = $1
		  AND member_id = ANY($2)
		  AND period_start >= $3 AND period_start <= $4
		  AND status <> 'cancelled'
		GROUP BY member_id`,
		eegID, ids, from, to)
	if err != nil {
		return nil, fmt.Errorf("invoice query: %w", err)
	}
	defer invRows.Close()
	for invRows.Next() {
		var memberID uuid.UUID
		var invoiced, credited float64
		var count int
		if err := invRows.Scan(&memberID, &invoiced, &credited, &count); err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		if idx, ok := memberIdx[memberID]; ok {
			members[idx].Invoiced = invoiced
			members[idx].Credited = credited
			members[idx].InvoiceCount = count
		}
	}
	return members, invRows.Err()
}
