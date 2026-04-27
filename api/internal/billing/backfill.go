package billing

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// BackfillSplitAmounts fixes historical prosumer invoices where consumption_net_amount
// and generation_net_amount are inconsistent with net_amount (i.e. migration 064 produced
// approximate values via a period-level tariff average rather than billing.go's per-month logic).
//
// It re-runs the exact billing computation — monthly energy readings × monthly tariff price —
// for each affected invoice, verifies the result against the stored net_amount (exact match,
// no tolerance), and writes the corrected split amounts to the DB.
//
// Invoices where the re-computed result does not match net_amount are skipped with a warning
// (this may happen if EEG settings were changed after the invoice was issued).
//
// This function is idempotent: once all affected invoices are fixed, the WHERE clause matches
// nothing and the function exits immediately.
func BackfillSplitAmounts(ctx context.Context, db *pgxpool.Pool) error {
	// Find all prosumer invoices whose stored split amounts don't add up to net_amount.
	// We tolerate a rounding gap of 0.001 EUR to avoid re-processing already-exact rows.
	const gapThreshold = 0.001
	q := `
		SELECT i.id, i.eeg_id, i.member_id, i.period_start, i.period_end,
		       i.consumption_kwh, i.generation_kwh, i.net_amount,
		       e.free_kwh, e.discount_pct, e.meter_fee_eur, e.participation_fee_eur,
		       e.energy_price AS flat_energy_price, e.producer_price AS flat_producer_price
		FROM invoices i
		JOIN eegs e ON e.id = i.eeg_id
		WHERE i.consumption_kwh > 0
		  AND i.generation_kwh  > 0
		  AND ABS((i.consumption_net_amount - i.generation_net_amount) - i.net_amount) > $1`

	rows, err := db.Query(ctx, q, gapThreshold)
	if err != nil {
		return fmt.Errorf("backfill query: %w", err)
	}
	defer rows.Close()

	type invoiceRow struct {
		id, eegID, memberID          uuid.UUID
		periodStart, periodEnd       time.Time
		consumptionKwh, generationKwh float64
		netAmount                    float64
		freeKwh, discountPct         float64
		meterFeeEur, participationFeeEur float64
		flatEnergyPrice, flatProducerPrice float64
	}

	var affected []invoiceRow
	for rows.Next() {
		var r invoiceRow
		if err := rows.Scan(
			&r.id, &r.eegID, &r.memberID, &r.periodStart, &r.periodEnd,
			&r.consumptionKwh, &r.generationKwh, &r.netAmount,
			&r.freeKwh, &r.discountPct, &r.meterFeeEur, &r.participationFeeEur,
			&r.flatEnergyPrice, &r.flatProducerPrice,
		); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		affected = append(affected, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows err: %w", err)
	}

	if len(affected) == 0 {
		return nil
	}
	slog.Info("backfill split amounts: found inconsistent prosumer invoices", "count", len(affected))

	fixed, skipped := 0, 0
	for _, inv := range affected {
		consumptionNet, generationNet, ok := recomputeSplitAmounts(ctx, db, inv.id, inv.eegID, inv.memberID,
			inv.periodStart, inv.periodEnd,
			inv.consumptionKwh, inv.generationKwh,
			inv.freeKwh, inv.discountPct, inv.meterFeeEur, inv.participationFeeEur,
			inv.flatEnergyPrice, inv.flatProducerPrice,
			inv.netAmount,
		)
		if !ok {
			skipped++
			continue
		}

		if _, err := db.Exec(ctx,
			`UPDATE invoices SET consumption_net_amount=$2, generation_net_amount=$3 WHERE id=$1`,
			inv.id, consumptionNet, generationNet,
		); err != nil {
			slog.Warn("backfill: failed to update invoice", "id", inv.id, "err", err)
			skipped++
			continue
		}
		fixed++
	}

	slog.Info("backfill split amounts: done", "fixed", fixed, "skipped", skipped)
	return nil
}

// recomputeSplitAmounts replicates billing.go's per-month tariff logic for a single invoice.
// Returns (consumptionNet, generationNet, ok). ok=false means the result doesn't match
// the stored net_amount and the invoice should be left unchanged.
func recomputeSplitAmounts(
	ctx context.Context,
	db *pgxpool.Pool,
	invoiceID, eegID, memberID uuid.UUID,
	periodStart, periodEnd time.Time,
	consumptionKwh, generationKwh float64,
	freeKwh, discountPct, meterFeeEur, participationFeeEur float64,
	flatEnergyPrice, flatProducerPrice float64,
	storedNetAmount float64,
) (consumptionNet, generationNet float64, ok bool) {

	// Effective consumption after free kWh and discount (mirrors billing.go)
	effectiveConsumption := math.Max(0, consumptionKwh-freeKwh) * (1 - discountPct/100)

	// Load active tariff schedule entries for this EEG and period
	domainEntries, err := loadTariffEntries(ctx, db, eegID, periodStart, periodEnd)
	if err != nil {
		slog.Warn("backfill: failed to load tariff entries", "invoice_id", invoiceID, "err", err)
		return 0, 0, false
	}

	// Load monthly energy kWh for this member over the invoice period
	monthlyRows, err := loadMonthlyKwh(ctx, db, memberID, periodStart, periodEnd)
	if err != nil {
		slog.Warn("backfill: failed to load monthly kWh", "invoice_id", invoiceID, "err", err)
		return 0, 0, false
	}

	if len(monthlyRows) == 0 {
		slog.Warn("backfill: no monthly readings found", "invoice_id", invoiceID)
		return 0, 0, false
	}

	// Sum total monthly raw kWh to compute scaling ratios (same as billing.go)
	var totalMonthlyC, totalMonthlyG float64
	for _, m := range monthlyRows {
		totalMonthlyC += m.consumptionKwh
		totalMonthlyG += m.generationKwh
	}

	// Multi-month per-tariff computation (billing.go lines 407-418)
	var totalEnergyEur, totalGenEur float64
	for _, m := range monthlyRows {
		monthStart := time.Date(m.month.Year(), m.month.Month(), 1, 0, 0, 0, 0, time.UTC)
		monthEnd := monthStart.AddDate(0, 1, 0)
		// Clamp to invoice period
		if monthStart.Before(periodStart) {
			monthStart = periodStart
		}
		if monthEnd.After(periodEnd) {
			monthEnd = periodEnd
		}

		mEnergyPrice, mProducerPrice := tariffWeightedPrice(domainEntries, monthStart, monthEnd, flatEnergyPrice, flatProducerPrice)

		// Scale monthly kWh by the same ratio as billing.go
		var scaledC, scaledG float64
		if totalMonthlyC > 0 {
			scaledC = m.consumptionKwh * effectiveConsumption / totalMonthlyC
		}
		if totalMonthlyG > 0 {
			scaledG = m.generationKwh * generationKwh / totalMonthlyG
		}

		totalEnergyEur += scaledC * mEnergyPrice / 100
		totalGenEur += scaledG * mProducerPrice / 100
	}

	consumptionNet = totalEnergyEur + meterFeeEur + participationFeeEur
	generationNet = totalGenEur

	if consumptionNet <= 0 {
		slog.Warn("backfill: computed consumptionNet <= 0, skipping", "invoice_id", invoiceID, "value", consumptionNet)
		return 0, 0, false
	}

	// Always derive generationNet from the constraint: consumptionNet − generationNet = net_amount.
	// This ensures the stored split is always consistent with net_amount, even when the active
	// tariff differs from what was used at billing time (e.g. tariff was modified after billing).
	generationNet = consumptionNet - storedNetAmount
	if generationNet <= 0 {
		slog.Warn("backfill: derived generationNet <= 0, skipping", "invoice_id", invoiceID, "value", generationNet)
		return 0, 0, false
	}

	diff := math.Abs((consumptionNet - generationNet) - storedNetAmount)
	if diff > 1e-6 {
		// Arithmetic sanity check — should never fail
		slog.Warn("backfill: constraint check failed", "invoice_id", invoiceID, "diff", diff)
		return 0, 0, false
	}

	// Round to 4 decimal places (matches billing.go's invoice storage)
	consumptionNet = math.Round(consumptionNet*10000) / 10000
	generationNet = math.Round(generationNet*10000) / 10000
	return consumptionNet, generationNet, true
}

// monthlyKwhRow holds raw kWh totals for one calendar month.
type monthlyKwhRow struct {
	month          time.Time
	consumptionKwh float64
	generationKwh  float64
}

// loadMonthlyKwh queries monthly wh_self (consumption) and wh_community (generation) sums
// for a member over a period. Mirrors MonthlySummaryForMember in repository/reading.go.
func loadMonthlyKwh(ctx context.Context, db *pgxpool.Pool, memberID uuid.UUID, from, to time.Time) ([]monthlyKwhRow, error) {
	q := `
		SELECT date_trunc('month', er.ts) AS month,
		       COALESCE(SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN er.wh_self      ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN mp.energierichtung = 'GENERATION'  THEN er.wh_community ELSE 0 END), 0)
		FROM meter_points mp
		JOIN energy_readings er ON er.meter_point_id = mp.id
		WHERE mp.member_id = $1
		  AND er.ts >= $2
		  AND er.ts <= $3
		  AND er.quality <> 'L3'
		GROUP BY date_trunc('month', er.ts)
		ORDER BY month ASC`
	rows, err := db.Query(ctx, q, memberID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []monthlyKwhRow
	for rows.Next() {
		var r monthlyKwhRow
		if err := rows.Scan(&r.month, &r.consumptionKwh, &r.generationKwh); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// loadTariffEntries loads the active tariff schedule's entries overlapping a period.
// Returns an empty slice (not an error) when no active schedule exists.
func loadTariffEntries(ctx context.Context, db *pgxpool.Pool, eegID uuid.UUID, periodStart, periodEnd time.Time) ([]domain.TariffEntry, error) {
	q := `
		SELECT te.id, te.schedule_id, te.valid_from, te.valid_until, te.energy_price, te.producer_price, te.created_at
		FROM tariff_schedules ts
		JOIN tariff_entries te ON te.schedule_id = ts.id
		WHERE ts.eeg_id = $1
		  AND ts.is_active = true
		  AND te.valid_from  < $3
		  AND te.valid_until > $2
		ORDER BY te.valid_from`
	rows, err := db.Query(ctx, q, eegID, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []domain.TariffEntry
	for rows.Next() {
		var e domain.TariffEntry
		if err := rows.Scan(&e.ID, &e.ScheduleID, &e.ValidFrom, &e.ValidUntil, &e.EnergyPrice, &e.ProducerPrice, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
