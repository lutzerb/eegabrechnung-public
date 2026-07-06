package billing

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/invoice"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// Config holds configuration for PDF generation.
type Config struct {
	InvoiceDir string // filesystem directory to store PDFs, default "/data/invoices"
}

// Service handles billing aggregation and invoice generation.
type Service struct {
	db             *pgxpool.Pool
	eegRepo        *repository.EEGRepository
	memberRepo     *repository.MemberRepository
	readingRepo    *repository.ReadingRepository
	invoiceRepo    *repository.InvoiceRepository
	billingRunRepo *repository.BillingRunRepository
	tariffRepo     *repository.TariffRepository
	emailLogRepo   *repository.EmailLogRepository
	meterPointRepo *repository.MeterPointRepository
	cfg            Config
}

func NewService(
	db *pgxpool.Pool,
	eegRepo *repository.EEGRepository,
	memberRepo *repository.MemberRepository,
	readingRepo *repository.ReadingRepository,
	invoiceRepo *repository.InvoiceRepository,
	billingRunRepo *repository.BillingRunRepository,
	tariffRepo *repository.TariffRepository,
	emailLogRepo *repository.EmailLogRepository,
	meterPointRepo *repository.MeterPointRepository,
	cfg ...Config,
) *Service {
	s := &Service{
		db:             db,
		eegRepo:        eegRepo,
		memberRepo:     memberRepo,
		readingRepo:    readingRepo,
		invoiceRepo:    invoiceRepo,
		billingRunRepo: billingRunRepo,
		tariffRepo:     tariffRepo,
		emailLogRepo:   emailLogRepo,
		meterPointRepo: meterPointRepo,
	}
	if len(cfg) > 0 {
		s.cfg = cfg[0]
	}
	if s.cfg.InvoiceDir == "" {
		s.cfg.InvoiceDir = "/data/invoices"
	}
	return s
}

// meterPointKwhBreakdown returns per-ZP billed kWh for a member. Consumption ZPs
// carry their raw wh_self; generation ZPs are scaled proportionally so they sum to totalGen.
func (s *Service) meterPointKwhBreakdown(ctx context.Context, memberID uuid.UUID, from, to time.Time, totalCons, totalGen float64) (consMPs, genMPs []invoice.MeterPointKwh) {
	mpSums, err := s.readingRepo.SumByMeterPointForMember(ctx, memberID, from, to)
	if err != nil {
		return nil, nil
	}
	var rawConsTotal, rawGenTotal float64
	for _, mp := range mpSums {
		if mp.Direction == "CONSUMPTION" {
			rawConsTotal += mp.RawKwh
		} else {
			rawGenTotal += mp.RawKwh
		}
	}
	for _, mp := range mpSums {
		if mp.Direction == "CONSUMPTION" {
			kwh := mp.RawKwh
			if rawConsTotal > 0 {
				kwh = mp.RawKwh / rawConsTotal * totalCons
			}
			consMPs = append(consMPs, invoice.MeterPointKwh{Zaehlpunkt: mp.Zaehlpunkt, Kwh: kwh})
		} else {
			kwh := 0.0
			if rawGenTotal > 0 {
				kwh = mp.RawKwh / rawGenTotal * totalGen
			}
			genMPs = append(genMPs, invoice.MeterPointKwh{Zaehlpunkt: mp.Zaehlpunkt, Kwh: kwh})
		}
	}
	return consMPs, genMPs
}

// SendAllResult holds the outcome of a bulk email send.
type SendAllResult struct {
	Sent    int      `json:"sent"`
	Skipped int      `json:"skipped"` // no email address or no PDF
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

// SendAll sends all unsent (draft/pending) invoices for the given EEG
// (optionally scoped to a billing run) to members who have an email address and a generated PDF.
func (s *Service) SendAll(ctx context.Context, eegID uuid.UUID, billingRunID *uuid.UUID) (*SendAllResult, error) {
	eeg, err := s.eegRepo.GetByIDInternal(ctx, eegID)
	if err != nil {
		return nil, fmt.Errorf("get eeg: %w", err)
	}

	var invoices []domain.Invoice
	if billingRunID != nil {
		invoices, err = s.invoiceRepo.ListByBillingRun(ctx, *billingRunID)
	} else {
		invoices, err = s.invoiceRepo.ListByEeg(ctx, eegID)
	}
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}

	result := &SendAllResult{}

	smtpCfg := invoice.SMTPConfig{
		Host:     eeg.SMTPHost,
		From:     eeg.SMTPFrom,
		Username: eeg.SMTPUser,
		Password: eeg.SMTPPassword,
	}

	// The SMTP connection is opened lazily on the first message that actually
	// needs sending, and reused for every subsequent one in this run — avoids
	// paying connect+TLS+auth per invoice (net/smtp.SendMail does that per call).
	var bulkSender *invoice.BulkSender
	defer func() {
		if bulkSender != nil {
			bulkSender.Close()
		}
	}()

	for i := range invoices {
		inv := &invoices[i]
		// Only send draft or pending invoices
		if inv.Status != "draft" && inv.Status != "pending" {
			continue
		}
		if inv.PdfPath == "" {
			result.Skipped++
			continue
		}

		member, err := s.memberRepo.GetByID(ctx, inv.MemberID)
		if err != nil {
			result.Skipped++
			continue
		}
		if member.Email == "" {
			result.Skipped++
			continue
		}

		pdfData, err := os.ReadFile(inv.PdfPath)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("invoice %s: PDF not readable: %v", inv.ID, err))
			continue
		}

		invID := inv.ID
		memID := member.ID
		msgBytes, buildErr := invoice.BuildInvoiceMessage(smtpCfg.From, member.Email, inv, eeg, member, pdfData)
		if buildErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("invoice %s (%s): build message: %v", inv.ID, member.Email, buildErr))
			continue
		}
		subject := invoice.InvoiceSubject(inv, eeg)
		entry := domain.EmailLogEntry{
			EegID:     eeg.ID,
			EmailType: "invoice",
			ToAddress: member.Email,
			Subject:   subject,
			MemberID:  &memID,
			InvoiceID: &invID,
		}

		if bulkSender == nil {
			bulkSender, err = invoice.NewBulkSender(smtpCfg)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("invoice %s (%s): smtp connect: %v", inv.ID, member.Email, err))
				slog.Warn("failed to open smtp connection", "invoice_id", inv.ID, "email", member.Email, "error", err)
				entry.Status = "failed"
				entry.ErrorMsg = err.Error()
				s.emailLogRepo.Log(ctx, entry)
				continue
			}
		}

		if bulkSender.Dead() {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("invoice %s (%s): smtp connection unavailable", inv.ID, member.Email))
			entry.Status = "failed"
			entry.ErrorMsg = "smtp connection unavailable"
			s.emailLogRepo.Log(ctx, entry)
			continue
		}

		if err := bulkSender.Send(member.Email, msgBytes); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("invoice %s (%s): %v", inv.ID, member.Email, err))
			slog.Warn("failed to send invoice", "invoice_id", inv.ID, "email", member.Email, "error", err)
			entry.Status = "failed"
			entry.ErrorMsg = err.Error()
			s.emailLogRepo.Log(ctx, entry)
			continue
		}
		entry.Status = "sent"
		s.emailLogRepo.Log(ctx, entry)

		if err := s.invoiceRepo.MarkSent(ctx, inv.ID); err != nil {
			slog.Warn("failed to mark invoice as sent", "invoice_id", inv.ID, "error", err)
		}
		result.Sent++
		slog.Info("invoice sent", "invoice_id", inv.ID, "member_id", member.ID, "email", member.Email)
	}

	return result, nil
}

// RunOptions configures a billing run.
type RunOptions struct {
	PeriodStart time.Time
	PeriodEnd   time.Time
	// MemberIDs limits billing to specific members (nil/empty = all members).
	MemberIDs []uuid.UUID
	// BillingType: "all" (default), "consumption_only", "production_only"
	BillingType string
	// Force skips the duplicate-period overlap check.
	Force bool
	// Preview: if true, compute amounts and generate PDFs but do not persist
	// any billing_run or invoice records — nothing is locked or committed.
	Preview bool
}

// OverlapError is returned when the requested period overlaps an existing billing run.
type OverlapError struct {
	Existing *domain.BillingRun
}

func (e *OverlapError) Error() string {
	return fmt.Sprintf("period overlaps with existing billing run %s (%s – %s)",
		e.Existing.ID,
		e.Existing.PeriodStart.Format("2006-01-02"),
		e.Existing.PeriodEnd.Format("2006-01-02"),
	)
}

// DataGapError is returned when one or more active meter points are missing
// 15-minute L1/L2 readings within the billing period.
type DataGapError struct {
	Details []repository.MeterPointGapDetail
}

func (e *DataGapError) Error() string {
	return fmt.Sprintf("incomplete data: %d Zählpunkt(e) haben fehlende L1/L2-Intervalle", len(e.Details))
}

// RunResult holds the outcome of a billing run.
type RunResult struct {
	BillingRun *domain.BillingRun `json:"billing_run"`
	Invoices   []domain.Invoice   `json:"invoices"`
}

// RunBilling aggregates energy readings for the period and creates invoices.
// Consumption (Bezug) and generation (Einspeisung) are calculated separately;
// the invoice total_amount is the net saldo (Bezug minus Einspeisung credit).
// When opts.Preview is true, all computation and PDF generation happens but
// nothing is persisted — billing_run and invoices are returned in-memory only.
func (s *Service) RunBilling(ctx context.Context, eegID uuid.UUID, opts RunOptions) (*RunResult, error) {
	periodStart, periodEnd := opts.PeriodStart, opts.PeriodEnd
	if periodEnd.Before(periodStart) {
		return nil, fmt.Errorf("period_end must be after period_start")
	}

	// Overlap check — skipped for preview runs and force-mode
	if !opts.Force && !opts.Preview {
		existing, err := s.billingRunRepo.FindOverlap(ctx, eegID, periodStart, periodEnd)
		if err != nil {
			return nil, fmt.Errorf("overlap check: %w", err)
		}
		if existing != nil {
			return nil, &OverlapError{Existing: existing}
		}

		// Data completeness check: every active ZP must have L1/L2 data for every 15-min slot
		details, err := s.readingRepo.MissingIntervalDetails(ctx, eegID, periodStart, periodEnd)
		if err != nil {
			return nil, fmt.Errorf("completeness check: %w", err)
		}
		if len(details) > 0 {
			return nil, &DataGapError{Details: details}
		}
	}

	eeg, err := s.eegRepo.GetByIDInternal(ctx, eegID)
	if err != nil {
		return nil, fmt.Errorf("get eeg: %w", err)
	}

	// Check for an active tariff schedule (EEG default) plus any active member-specific
	// Individualtarif schedules, to use time-varying prices and per-member fee overrides.
	defaultTariff, memberTariffs, err := s.tariffRepo.GetAllActiveForEEG(ctx, eegID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("get active tariff: %w", err)
	}
	if defaultTariff != nil {
		slog.Info("using active tariff for billing",
			"schedule_id", defaultTariff.ID,
			"schedule_name", defaultTariff.Name,
			"granularity", defaultTariff.Granularity,
			"entry_count", defaultTariff.EntryCount,
		)
	}
	if len(memberTariffs) > 0 {
		slog.Info("using member-specific Individualtarif schedules", "count", len(memberTariffs))
	}

	slog.Info("running billing",
		"eeg_id", eegID,
		"gemeinschaft_id", eeg.GemeinschaftID,
		"period_start", periodStart,
		"period_end", periodEnd,
		"energy_price", eeg.EnergyPrice,
		"producer_price", eeg.ProducerPrice,
		"use_vat", eeg.UseVat,
		"billing_type", opts.BillingType,
		"member_filter_count", len(opts.MemberIDs),
		"force", opts.Force,
	)

	// Get per-member energy sums split by direction (CONSUMPTION vs GENERATION)
	sums, err := s.readingRepo.SumByMemberAndPeriod(ctx, eegID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("sum by member: %w", err)
	}

	// Filter by member IDs if specified
	if len(opts.MemberIDs) > 0 {
		allowed := make(map[uuid.UUID]bool, len(opts.MemberIDs))
		for _, id := range opts.MemberIDs {
			allowed[id] = true
		}
		filtered := sums[:0]
		for _, s := range sums {
			if allowed[s.MemberID] {
				filtered = append(filtered, s)
			}
		}
		sums = filtered
	}

	// Create billing run record before processing invoices (skipped in preview mode)
	run := &domain.BillingRun{
		EegID:       eegID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Status:      "draft",
	}
	if !opts.Preview {
		if err := s.billingRunRepo.Create(ctx, run); err != nil {
			return nil, fmt.Errorf("create billing run: %w", err)
		}
		slog.Info("billing run created", "billing_run_id", run.ID)
	} else {
		slog.Info("preview billing run (not persisted)", "eeg_id", eegID, "period_start", periodStart, "period_end", periodEnd)
	}

	if len(sums) == 0 {
		slog.Warn("no readings found for billing period",
			"eeg_id", eegID,
			"period_start", periodStart,
			"period_end", periodEnd,
		)
		return &RunResult{
			BillingRun: run,
			Invoices:   []domain.Invoice{},
		}, nil
	}

	var invoices []domain.Invoice
	for _, sum := range sums {
		consumptionKwh := sum.ConsumptionKwh
		generationKwh := sum.GenerationKwh

		// Fetch member early — needed for effective period and credit-note VAT
		member, memberErr := s.memberRepo.GetByID(ctx, sum.MemberID)

		// ── Individualtarif resolution ───────────────────────────────────────────────
		// A member with an active member-specific tariff schedule uses its Entries for the
		// Arbeitspreis blend (see tariffWeightedPrice below) and its non-nil *Override fields
		// for fixed fees; anything left nil (or no member schedule at all) falls back to the
		// EEG-wide default schedule/settings.
		activeTariff := defaultTariff
		if mt, ok := memberTariffs[sum.MemberID]; ok {
			activeTariff = mt
		}
		freeKwh, discountPct := eeg.FreeKwh, eeg.DiscountPct
		meterFee, participationFee, zpGebuehr := eeg.MeterFeeEur, eeg.ParticipationFeeEur, eeg.ZaehlpunktsGebuehrEur
		if activeTariff != nil && activeTariff.MemberID != nil {
			if v := activeTariff.FreeKwhOverride; v != nil {
				freeKwh = *v
			}
			if v := activeTariff.DiscountPctOverride; v != nil {
				discountPct = *v
			}
			if v := activeTariff.MeterFeeEurOverride; v != nil {
				meterFee = *v
			}
			if v := activeTariff.ParticipationFeeEurOverride; v != nil {
				participationFee = *v
			}
			if v := activeTariff.ZaehlpunktsGebuehrOverride; v != nil {
				zpGebuehr = *v
			}
		}
		activeZPCount, _ := s.meterPointRepo.CountActiveByMember(ctx, sum.MemberID)
		zpGebuehrTotal := float64(activeZPCount) * zpGebuehr

		// ── Effective billing period per member ────────────────────────────────
		// Clamp to beitritt_datum / austritt_datum when they fall within the period.
		effectiveStart := periodStart
		effectiveEnd := periodEnd
		if memberErr == nil {
			// Clamp only when the date falls strictly within the billing period.
			// Dates before periodStart or after periodEnd mean the member was in the
			// EEG for the whole period (or the date is a future placeholder) — no trim.
			if member.BeitrittsDatum != nil {
				bd := member.BeitrittsDatum.UTC()
				if bd.After(periodStart) && !bd.After(periodEnd) {
					effectiveStart = bd
				}
			}
			if member.AustrittsDatum != nil {
				// Treat austritt_datum as end-of-day so the last day is fully billed.
				ad := member.AustrittsDatum.UTC()
				adEndOfDay := time.Date(ad.Year(), ad.Month(), ad.Day(), 23, 59, 59, 0, time.UTC)
				if adEndOfDay.After(periodStart) && adEndOfDay.Before(periodEnd) {
					effectiveEnd = adEndOfDay
				}
			}
			if !effectiveStart.Before(effectiveEnd) {
				slog.Info("skipping member — not in EEG during billing period",
					"member_id", sum.MemberID,
					"effective_start", effectiveStart,
					"effective_end", effectiveEnd,
				)
				continue
			}
			// If the period was clamped, re-query energy sums for the effective window.
			if effectiveStart != periodStart || effectiveEnd != periodEnd {
				if ms, err := s.readingRepo.SumForMember(ctx, eegID, sum.MemberID, effectiveStart, effectiveEnd); err == nil {
					consumptionKwh = ms.ConsumptionKwh
					generationKwh = ms.GenerationKwh
				}
			}
		}

		// Apply billing type filter
		switch opts.BillingType {
		case "consumption_only":
			generationKwh = 0
		case "production_only":
			consumptionKwh = 0
		}

		// ── Consumption charge (Bezug) ─────────────────────────────────────────
		// Apply free kWh and discount to consumption (Individualtarif-aware)
		effectiveConsumption := math.Max(0, consumptionKwh-freeKwh)
		effectiveConsumption = effectiveConsumption * (1 - discountPct/100)
		// Compute effective prices: use tariff weighted average if available, else EEG flat price
		energyPriceCt, producerPriceCt := eeg.EnergyPrice, eeg.ProducerPrice
		if activeTariff != nil && len(activeTariff.Entries) > 0 {
			energyPriceCt, producerPriceCt = tariffWeightedPrice(
				activeTariff.Entries, effectiveStart, effectiveEnd, eeg.EnergyPrice, eeg.ProducerPrice,
			)
		}
		// Work price: ct/kWh ÷ 100 = EUR/kWh — energy-only, excludes all fixed fees
		energyOnlyNet := effectiveConsumption * energyPriceCt / 100
		// Add fixed fees (meter/participation fee + Zählpunktsgebühr × active meter points)
		consumptionNet := energyOnlyNet + meterFee + participationFee + zpGebuehrTotal

		// ── Generation credit (Einspeisung) ────────────────────────────────────
		// Producer price: ct/kWh ÷ 100 = EUR/kWh
		generationCredit := generationKwh * producerPriceCt / 100

		// ── Net saldo ──────────────────────────────────────────────────────────
		// Net = consumption charge − generation credit
		// Negative → member is owed money (heavy producer)
		netAmount := consumptionNet - generationCredit

		// ── Monthly breakdown for multi-month invoices ─────────────────────────
		// ── Monthly breakdown for multi-month invoices ─────────────────────────
		// Build per-month line items with individual tariff prices.
		// kWh are scaled so months sum to the billed totals.
		// When a tariff schedule is active, each month gets its own weighted price
		// and the billing amounts (consumptionNet, generationCredit) are recomputed
		// from the per-month totals — more accurate than the period-level average.
		monthlyRaw, _ := s.readingRepo.MonthlySummaryForMember(ctx, sum.MemberID, effectiveStart, effectiveEnd)
		var monthlyItems []invoice.MonthlyKwh
		if len(monthlyRaw) > 0 {
			var totalMonthlyC, totalMonthlyG float64
			for _, m := range monthlyRaw {
				totalMonthlyC += m.ConsumptionKwh
				totalMonthlyG += m.GenerationKwh
			}
			for _, m := range monthlyRaw {
				// Per-month tariff price: call tariffWeightedPrice for each month's window.
				monthStart := time.Date(m.Month.Year(), m.Month.Month(), 1, 0, 0, 0, 0, time.UTC)
				monthEnd := monthStart.AddDate(0, 1, 0) // exclusive upper bound for this month
				// Clamp to effective period
				if monthStart.Before(effectiveStart) {
					monthStart = effectiveStart
				}
				if monthEnd.After(effectiveEnd) {
					monthEnd = effectiveEnd
				}
				mEnergyPriceCt, mProducerPriceCt := eeg.EnergyPrice, eeg.ProducerPrice
				if activeTariff != nil && len(activeTariff.Entries) > 0 {
					mEnergyPriceCt, mProducerPriceCt = tariffWeightedPrice(
						activeTariff.Entries, monthStart, monthEnd, eeg.EnergyPrice, eeg.ProducerPrice,
					)
				}

				scaled := invoice.MonthlyKwh{
					Month:           m.Month,
					EnergyPriceCt:   mEnergyPriceCt,
					ProducerPriceCt: mProducerPriceCt,
				}
				if totalMonthlyC > 0 {
					scaled.ConsumptionKwh = m.ConsumptionKwh * effectiveConsumption / totalMonthlyC
				}
				if totalMonthlyG > 0 {
					scaled.GenerationKwh = m.GenerationKwh * generationKwh / totalMonthlyG
				}
				monthlyItems = append(monthlyItems, scaled)
			}

			// For multi-month billing with a tariff schedule, recompute the billing
			// amounts from the per-month totals (each month × its own tariff price).
			if len(monthlyItems) > 1 && activeTariff != nil {
				var totalEnergyEur, totalGenEur float64
				for _, m := range monthlyItems {
					totalEnergyEur += m.ConsumptionKwh * m.EnergyPriceCt / 100
					totalGenEur += m.GenerationKwh * m.ProducerPriceCt / 100
				}
				energyOnlyNet = totalEnergyEur
				consumptionNet = totalEnergyEur + meterFee + participationFee + zpGebuehrTotal
				generationCredit = totalGenEur
				netAmount = consumptionNet - generationCredit
			}
		}

		// Determine document type: credit note for pure producers when EEG has credit notes enabled
		isPureProducer := consumptionKwh == 0 && generationKwh > 0
		isCreditNote := isPureProducer && eeg.GenerateCreditNotes

		// ── Meter points (for invoice imprint) ────────────────────────────────
		consumptionMPs, generationMPs := s.meterPointKwhBreakdown(ctx, sum.MemberID, effectiveStart, effectiveEnd, consumptionKwh, generationKwh)

		// ── VAT calculation — consumption and generation are independent ───────
		// Consumption (Bezug): EEG-level VAT (§ 6 exempt if Kleinunternehmer, 20% if VAT-registered)
		// Generation (Einspeisung): member-specific VAT per Austrian EEG law:
		//   - pauschalierter Landwirt (§ 22 UStG): 13 % on Gutschrift/Einspeisung credit
		//   - Unternehmen/Gemeinde (BgA) with UID: Reverse Charge § 19, 0% on document
		//   - all others: 0 % (exempt)
		consumptionVatPct := 0.0
		consumptionVatAmount := 0.0
		consumptionGross := consumptionNet
		if eeg.UseVat && consumptionNet > 0 {
			consumptionVatPct = eeg.VatPct
			consumptionVatAmount = consumptionNet * eeg.VatPct / 100
			consumptionGross = consumptionNet + consumptionVatAmount
		}

		generationVatPct := 0.0
		generationVatAmount := 0.0
		generationGross := generationCredit
		generationVatText := ""
		generationRC := false
		if generationKwh > 0 && memberErr == nil {
			generationVatText = invoice.GenerationVATText(member)
			generationVatPct = invoice.GenerationVATPct(member)
			generationRC = invoice.GenerationReverseCharge(member)
			if generationVatPct > 0 {
				generationVatAmount = generationCredit * generationVatPct / 100
				generationGross = generationCredit + generationVatAmount
			}
		}

		totalAmount := consumptionGross - generationGross
		vatAmount := consumptionVatAmount + generationVatAmount
		vatPctApplied := consumptionVatPct // EEG-level rate (for DATEV / summary)

		// § 19 Abs. 1 UStG (Reverse Charge on consumption): applies when a VAT-registered EEG
		// delivers electricity to a VAT-registered business (Unternehmen/Gemeinde BgA with UID).
		// The EEG still shows and remits 20% VAT, but the invoice must note the Steuerschuldübergang.
		consumptionReverseCharge := false
		if eeg.UseVat && memberErr == nil && member.UidNummer != "" {
			role := member.BusinessRole
			if role != "privat" && role != "gemeinde_hoheitlich" && role != "landwirt_pauschaliert" {
				consumptionReverseCharge = true
			}
		}

		vatOpts := invoice.VATOptions{
			UseVat:               eeg.UseVat,
			VatPct:               eeg.VatPct,
			ConsumptionKwh:       consumptionKwh,
			GenerationKwh:        generationKwh,
			ConsumptionNet:       consumptionNet,
			GenerationNet:        generationCredit,
			EnergyPrice:          energyPriceCt,
			ProducerPrice:        producerPriceCt,
			ConsumptionVatPct:    consumptionVatPct,
			ConsumptionVatAmount: consumptionVatAmount,
			ConsumptionGross:     consumptionGross,
			GenerationVatPct:     generationVatPct,
			GenerationVatAmount:  generationVatAmount,
			GenerationGross:      generationGross,
			GenerationVatText:    generationVatText,
			ConsumptionReverseCharge:  consumptionReverseCharge,
			GenerationReverseCharge:   generationRC,
			ConsumptionMeterPointKwh:  consumptionMPs,
			GenerationMeterPointKwh:   generationMPs,
			MonthlyLineItems:          monthlyItems,
			EnergyNet:                energyOnlyNet,
			MeterFeeEur:              meterFee,
			ParticipationFeeEur:      participationFee,
			ZaehlpunktsGebuehrEur:    zpGebuehr,
			ZaehlpunktsGebuehrCount:  activeZPCount,
			ZaehlpunktsGebuehrTotal:  zpGebuehrTotal,
		}

		inv := &domain.Invoice{
			MemberID:             sum.MemberID,
			EegID:                eegID,
			PeriodStart:          effectiveStart,
			PeriodEnd:            effectiveEnd,
			TotalKwh:             consumptionKwh,
			TotalAmount:          totalAmount,
			NetAmount:            consumptionNet - generationCredit,
			VatAmount:            vatAmount,
			VatPctApplied:        vatPctApplied,
			ConsumptionKwh:       consumptionKwh,
			GenerationKwh:        generationKwh,
			ConsumptionNetAmount: consumptionNet,
			GenerationNetAmount:  generationCredit,
			ConsumptionVatPct:    consumptionVatPct,
			ConsumptionVatAmount: consumptionVatAmount,
			GenerationVatPct:     generationVatPct,
			GenerationVatAmount:  generationVatAmount,
			PdfPath:              "",
			Status:               "draft",
			DocumentType:         "invoice",
		}
		if isCreditNote {
			inv.DocumentType = "credit_note"
		}
		if !opts.Preview {
			inv.BillingRunID = &run.ID
			if err := s.invoiceRepo.Create(ctx, inv); err != nil {
				return nil, fmt.Errorf("create invoice for member %s: %w", sum.MemberID, err)
			}
		}

		slog.Info("invoice created",
			"invoice_id", inv.ID,
			"member_id", sum.MemberID,
			"document_type", inv.DocumentType,
			"consumption_kwh", consumptionKwh,
			"generation_kwh", generationKwh,
			"consumption_net", consumptionNet,
			"generation_credit", generationCredit,
			"net_amount", netAmount,
			"total_amount", totalAmount,
			"vat_pct_applied", vatPctApplied,
		)

		if memberErr != nil {
			slog.Warn("failed to get member for PDF generation, skipping",
				"member_id", sum.MemberID,
				"invoice_id", inv.ID,
				"error", memberErr,
			)
			invoices = append(invoices, *inv)
			continue
		}

		// Fetch last 6 months of energy history for the bar chart.
		// Go back 5 full months before the billing period start so we end with the current period.
		historyFrom := time.Date(periodStart.Year(), periodStart.Month()-5, 1, 0, 0, 0, 0, time.UTC)
		rawHistory, histErr := s.readingRepo.MonthlySummaryForMember(ctx, sum.MemberID, historyFrom, periodEnd)
		var chartHistory []invoice.MonthlyKwh
		if histErr == nil {
			for _, h := range rawHistory {
				chartHistory = append(chartHistory, invoice.MonthlyKwh{
					Month:          h.Month,
					ConsumptionKwh: h.ConsumptionKwh,
					GenerationKwh:  h.GenerationKwh,
				})
			}
		}

		// Generate PDF — credit note or regular invoice
		var pdfData []byte
		if isCreditNote {
			pdfData, err = invoice.GenerateCreditNotePDF(inv, eeg, member, producerPriceCt, generationKwh, generationMPs, monthlyItems, chartHistory)
		} else {
			pdfData, err = invoice.GeneratePDF(inv, eeg, member, vatOpts, chartHistory)
		}
		if err != nil {
			slog.Warn("failed to generate PDF, skipping",
				"invoice_id", inv.ID,
				"error", err,
			)
			invoices = append(invoices, *inv)
			continue
		}

		if !opts.Preview {
			// Write PDF to disk
			pdfDir := filepath.Join(s.cfg.InvoiceDir, eegID.String())
			if err := os.MkdirAll(pdfDir, 0755); err != nil {
				slog.Warn("failed to create invoice directory, skipping PDF write",
					"dir", pdfDir,
					"error", err,
				)
				invoices = append(invoices, *inv)
				continue
			}

			pdfPath := filepath.Join(pdfDir, inv.ID.String()+".pdf")
			if err := os.WriteFile(pdfPath, pdfData, 0644); err != nil {
				slog.Warn("failed to write PDF file, skipping",
					"path", pdfPath,
					"error", err,
				)
				invoices = append(invoices, *inv)
				continue
			}

			// Update PDF path in DB
			inv.PdfPath = pdfPath
			if err := s.invoiceRepo.UpdatePdfPath(ctx, inv.ID, pdfPath); err != nil {
				slog.Warn("failed to update pdf_path in DB",
					"invoice_id", inv.ID,
					"error", err,
				)
			}

		}

		invoices = append(invoices, *inv)
	}

	run.InvoiceCount = len(invoices)
	for _, inv := range invoices {
		run.TotalAmount += inv.TotalAmount
	}

	return &RunResult{
		BillingRun: run,
		Invoices:   invoices,
	}, nil
}

// RegeneratePDF re-runs the PDF generation for an existing invoice using current code and stored data.
// It overwrites the PDF file on disk but does not change any invoice amounts in the DB.
func (s *Service) RegeneratePDF(ctx context.Context, invoiceID uuid.UUID) ([]byte, error) {
	inv, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("get invoice: %w", err)
	}
	eeg, err := s.eegRepo.GetByIDInternal(ctx, inv.EegID)
	if err != nil {
		return nil, fmt.Errorf("get eeg: %w", err)
	}
	member, err := s.memberRepo.GetByID(ctx, inv.MemberID)
	if err != nil {
		return nil, fmt.Errorf("get member: %w", err)
	}

	// Re-fetch energy sums for this member/period to recover per-direction data
	sums, err := s.readingRepo.SumByMemberAndPeriod(ctx, inv.EegID, inv.PeriodStart, inv.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("sum readings: %w", err)
	}
	consumptionKwh := inv.ConsumptionKwh
	generationKwh := inv.GenerationKwh
	for _, s := range sums {
		if s.MemberID == inv.MemberID {
			consumptionKwh = s.ConsumptionKwh
			generationKwh = s.GenerationKwh
			break
		}
	}

	// Meter points
	consumptionMPs, generationMPs := s.meterPointKwhBreakdown(ctx, inv.MemberID, inv.PeriodStart, inv.PeriodEnd, consumptionKwh, generationKwh)

	// Reconstruct net amounts from stored VAT data
	// NetAmount = ConsumptionNet - GenerationNet; VAT amounts are stored explicitly.
	consumptionVatPct := inv.ConsumptionVatPct
	consumptionVatAmount := inv.ConsumptionVatAmount
	generationVatPct := inv.GenerationVatPct
	generationVatAmount := inv.GenerationVatAmount

	var consumptionNet, generationNet float64
	if consumptionVatPct > 0 {
		consumptionNet = consumptionVatAmount * 100 / consumptionVatPct
		generationNet = consumptionNet - inv.NetAmount
	} else if generationVatPct > 0 {
		generationNet = generationVatAmount * 100 / generationVatPct
		consumptionNet = inv.NetAmount + generationNet
	} else {
		// Both 0%: fall back to stored net+kwh ratio to separate
		total := consumptionKwh + generationKwh
		if total > 0 {
			consumptionNet = inv.NetAmount * (consumptionKwh / total)
			generationNet = -inv.NetAmount * (generationKwh / total)
			if consumptionKwh == 0 {
				consumptionNet = 0
				generationNet = -inv.NetAmount
			} else if generationKwh == 0 {
				consumptionNet = inv.NetAmount
				generationNet = 0
			}
		} else {
			consumptionNet = inv.NetAmount
		}
	}
	consumptionGross := consumptionNet + consumptionVatAmount
	generationGross := generationNet + generationVatAmount

	// Re-detect reverse charge (current member/EEG state)
	consumptionReverseCharge := false
	if eeg.UseVat && member.UidNummer != "" {
		role := member.BusinessRole
		if role != "privat" && role != "gemeinde_hoheitlich" && role != "landwirt_pauschaliert" {
			consumptionReverseCharge = true
		}
	}

	// Reconstruct prices from stored amounts (best effort)
	energyPriceCt := eeg.EnergyPrice
	producerPriceCt := eeg.ProducerPrice
	if consumptionKwh > 0 {
		energyPriceCt = consumptionNet / consumptionKwh * 100
	}
	if generationKwh > 0 {
		producerPriceCt = generationNet / generationKwh * 100
	}

	generationVatText := ""
	generationRC := false
	if generationKwh > 0 {
		generationVatText = invoice.GenerationVATText(member)
		generationRC = invoice.GenerationReverseCharge(member)
	}

	// Best-effort fee split for display: RegeneratePDF only has the stored total ConsumptionNet,
	// not what fees actually applied at billing time (member Individualtarif overrides aren't
	// persisted per-invoice). Uses *current* EEG/meter-point state, same caveat as the price
	// reconstruction above.
	activeZPCount, _ := s.meterPointRepo.CountActiveByMember(ctx, inv.MemberID)
	zpGebuehrTotal := float64(activeZPCount) * eeg.ZaehlpunktsGebuehrEur
	feeTotal := eeg.MeterFeeEur + eeg.ParticipationFeeEur + zpGebuehrTotal
	energyOnlyNet := consumptionNet - feeTotal
	if energyOnlyNet < 0 {
		energyOnlyNet = consumptionNet
	}

	vatOpts := invoice.VATOptions{
		UseVat:                   eeg.UseVat,
		VatPct:                   eeg.VatPct,
		ConsumptionKwh:           consumptionKwh,
		GenerationKwh:            generationKwh,
		ConsumptionNet:           consumptionNet,
		GenerationNet:            generationNet,
		EnergyPrice:              energyPriceCt,
		ProducerPrice:            producerPriceCt,
		ConsumptionVatPct:        consumptionVatPct,
		ConsumptionVatAmount:     consumptionVatAmount,
		ConsumptionGross:         consumptionGross,
		GenerationVatPct:         generationVatPct,
		GenerationVatAmount:      generationVatAmount,
		GenerationGross:          generationGross,
		GenerationVatText:        generationVatText,
		ConsumptionReverseCharge:  consumptionReverseCharge,
		GenerationReverseCharge:   generationRC,
		ConsumptionMeterPointKwh:  consumptionMPs,
		GenerationMeterPointKwh:   generationMPs,
		EnergyNet:                energyOnlyNet,
		MeterFeeEur:              eeg.MeterFeeEur,
		ParticipationFeeEur:      eeg.ParticipationFeeEur,
		ZaehlpunktsGebuehrEur:    eeg.ZaehlpunktsGebuehrEur,
		ZaehlpunktsGebuehrCount:  activeZPCount,
		ZaehlpunktsGebuehrTotal:  zpGebuehrTotal,
	}

	// Energy history for chart
	historyFrom := time.Date(inv.PeriodStart.Year(), inv.PeriodStart.Month()-5, 1, 0, 0, 0, 0, time.UTC)
	rawHistory, _ := s.readingRepo.MonthlySummaryForMember(ctx, inv.MemberID, historyFrom, inv.PeriodEnd)
	var chartHistory []invoice.MonthlyKwh
	for _, h := range rawHistory {
		chartHistory = append(chartHistory, invoice.MonthlyKwh{
			Month:          h.Month,
			ConsumptionKwh: h.ConsumptionKwh,
			GenerationKwh:  h.GenerationKwh,
		})
	}

	// Generate PDF
	isCreditNote := inv.DocumentType == "credit_note"
	var pdfData []byte
	if isCreditNote {
		// monthlyItems not available during regeneration — falls back to single-month layout
		pdfData, err = invoice.GenerateCreditNotePDF(inv, eeg, member, producerPriceCt, generationKwh, generationMPs, nil, chartHistory)
	} else {
		pdfData, err = invoice.GeneratePDF(inv, eeg, member, vatOpts, chartHistory)
	}
	if err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}

	// Overwrite file on disk if path exists
	if inv.PdfPath != "" {
		if err := os.MkdirAll(filepath.Dir(inv.PdfPath), 0755); err == nil {
			_ = os.WriteFile(inv.PdfPath, pdfData, 0644)
		}
	}

	return pdfData, nil
}

// tariffWeightedPrice computes energy and producer prices from tariff entries weighted by time overlap
// with the billing period. Uncovered portions use the fallback flat prices.
func tariffWeightedPrice(entries []domain.TariffEntry, start, end time.Time, fallbackEnergy, fallbackProducer float64) (energyPrice, producerPrice float64) {
	totalSecs := end.Sub(start).Seconds()
	if totalSecs <= 0 || len(entries) == 0 {
		return fallbackEnergy, fallbackProducer
	}
	var coveredSecs float64
	for _, e := range entries {
		oStart, oEnd := start, end
		if e.ValidFrom.After(oStart) {
			oStart = e.ValidFrom
		}
		if e.ValidUntil.Before(oEnd) {
			oEnd = e.ValidUntil
		}
		if oEnd.After(oStart) {
			overlap := oEnd.Sub(oStart).Seconds()
			coveredSecs += overlap
			w := overlap / totalSecs
			energyPrice += e.EnergyPrice * w
			producerPrice += e.ProducerPrice * w
		}
	}
	// Blend in flat prices for any uncovered fraction of the period
	uncoveredFrac := (totalSecs - coveredSecs) / totalSecs
	if uncoveredFrac > 0 {
		energyPrice += fallbackEnergy * uncoveredFrac
		producerPrice += fallbackProducer * uncoveredFrac
	}
	return
}
