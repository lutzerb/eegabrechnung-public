package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/auth"
	"github.com/lutzerb/eegabrechnung/internal/domain"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// EEGBackup is the JSON envelope for a full EEG data export.
type EEGBackup struct {
	Version         string                         `json:"version"`
	CreatedAt       time.Time                      `json:"created_at"`
	EEG             domain.EEG                     `json:"eeg"`
	Members         []domain.Member                `json:"members"`
	MeterPoints     []domain.MeterPoint            `json:"meter_points"`
	TariffSchedules []domain.TariffSchedule        `json:"tariff_schedules"`
	BillingRuns     []domain.BillingRun            `json:"billing_runs"`
	Invoices        []domain.Invoice               `json:"invoices"`
	Participations  []domain.EEGMeterParticipation `json:"participations"`
	Readings        []domain.EnergyReading         `json:"readings"`
}

// BackupHandler handles EEG data export and restore.
type BackupHandler struct {
	pool           *pgxpool.Pool
	eegRepo        *repository.EEGRepository
	memberRepo     *repository.MemberRepository
	meterPointRepo *repository.MeterPointRepository
	readingRepo    *repository.ReadingRepository
	invoiceRepo    *repository.InvoiceRepository
	billingRunRepo *repository.BillingRunRepository
	tariffRepo     *repository.TariffRepository
	participRepo   *repository.ParticipationRepository
}

func NewBackupHandler(
	pool *pgxpool.Pool,
	eegRepo *repository.EEGRepository,
	memberRepo *repository.MemberRepository,
	meterPointRepo *repository.MeterPointRepository,
	readingRepo *repository.ReadingRepository,
	invoiceRepo *repository.InvoiceRepository,
	billingRunRepo *repository.BillingRunRepository,
	tariffRepo *repository.TariffRepository,
	participRepo *repository.ParticipationRepository,
) *BackupHandler {
	return &BackupHandler{
		pool:           pool,
		eegRepo:        eegRepo,
		memberRepo:     memberRepo,
		meterPointRepo: meterPointRepo,
		readingRepo:    readingRepo,
		invoiceRepo:    invoiceRepo,
		billingRunRepo: billingRunRepo,
		tariffRepo:     tariffRepo,
		participRepo:   participRepo,
	}
}

// Export handles GET /api/v1/eegs/{eegID}/backup
//
//	@Summary		Export full EEG data snapshot
//	@Description	Exports a complete JSON snapshot of all EEG data including members, meter points, tariff schedules, billing runs, invoices, participations, and energy readings. The file is returned as a downloadable attachment.
//	@Tags			System
//	@Produce		application/json
//	@Param			eegID	path		string	true	"EEG UUID"
//	@Success		200		{object}	EEGBackup
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/backup [get]
func (h *BackupHandler) Export(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	eeg, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID)
	if err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	ctx := r.Context()

	members, err := h.memberRepo.ListByEeg(ctx, eegID)
	if err != nil {
		jsonError(w, "failed to load members: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if members == nil {
		members = []domain.Member{}
	}

	meterPoints, err := h.meterPointRepo.ListByEeg(ctx, eegID)
	if err != nil {
		jsonError(w, "failed to load meter points: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if meterPoints == nil {
		meterPoints = []domain.MeterPoint{}
	}

	// Load tariff schedules with their entries.
	scheduleList, err := h.tariffRepo.ListByEeg(ctx, eegID)
	if err != nil {
		jsonError(w, "failed to load tariff schedules: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tariffSchedules := make([]domain.TariffSchedule, 0, len(scheduleList))
	for _, s := range scheduleList {
		full, ferr := h.tariffRepo.GetWithEntries(ctx, s.ID)
		if ferr != nil || full == nil {
			tariffSchedules = append(tariffSchedules, s)
			continue
		}
		tariffSchedules = append(tariffSchedules, *full)
	}

	billingRuns, err := h.billingRunRepo.ListByEeg(ctx, eegID)
	if err != nil {
		jsonError(w, "failed to load billing runs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if billingRuns == nil {
		billingRuns = []domain.BillingRun{}
	}

	invoices, err := h.invoiceRepo.ListByEeg(ctx, eegID)
	if err != nil {
		jsonError(w, "failed to load invoices: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if invoices == nil {
		invoices = []domain.Invoice{}
	}

	participations, err := h.participRepo.ListByEEG(ctx, eegID)
	if err != nil {
		jsonError(w, "failed to load participations: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if participations == nil {
		participations = []domain.EEGMeterParticipation{}
	}

	readings, err := h.readingRepo.GetAllByEEG(ctx, eegID)
	if err != nil {
		jsonError(w, "failed to load readings: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if readings == nil {
		readings = []domain.EnergyReading{}
	}

	backup := EEGBackup{
		Version:         "1",
		CreatedAt:       time.Now().UTC(),
		EEG:             *eeg,
		Members:         members,
		MeterPoints:     meterPoints,
		TariffSchedules: tariffSchedules,
		BillingRuns:     billingRuns,
		Invoices:        invoices,
		Participations:  participations,
		Readings:        readings,
	}

	filename := fmt.Sprintf("backup_%s_%s.json", eeg.Name, time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(backup); err != nil {
		slog.Error("failed to encode backup", "error", err)
	}
	slog.Info("backup exported",
		"eeg_id", eegID,
		"members", len(members),
		"meter_points", len(meterPoints),
		"readings", len(readings),
	)
}

// Restore handles POST /api/v1/eegs/{eegID}/restore
//
//	@Summary		Restore EEG data from a backup snapshot
//	@Description	Restores all EEG data from a previously exported JSON backup file. The operation runs inside a single database transaction: existing data for the EEG is deleted in FK-safe order, then all records from the backup are re-inserted. The backup file must be uploaded as multipart/form-data with field name 'file'. Only backup version '1' is supported.
//	@Tags			System
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			eegID	path		string	true	"EEG UUID"
//	@Param			file	formData	file	true	"JSON backup file (produced by the Export endpoint)"
//	@Success		200		{object}	map[string]int	"counts of restored records"
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/eegs/{eegID}/restore [post]
func (h *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	eegID, err := uuid.Parse(chi.URLParam(r, "eegID"))
	if err != nil {
		jsonError(w, "invalid EEG ID", http.StatusBadRequest)
		return
	}
	if _, err := h.eegRepo.GetByID(r.Context(), eegID, claims.OrganizationID); err != nil {
		jsonError(w, "EEG not found", http.StatusNotFound)
		return
	}

	if err := r.ParseMultipartForm(256 << 20); err != nil {
		jsonError(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "file field required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, "failed to read backup file", http.StatusBadRequest)
		return
	}

	var backup EEGBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		jsonError(w, "invalid backup file: "+err.Error(), http.StatusBadRequest)
		return
	}
	if backup.Version != "1" {
		jsonError(w, fmt.Sprintf("unsupported backup version: %s", backup.Version), http.StatusBadRequest)
		return
	}

	backupEEGID := backup.EEG.ID
	if err := h.restoreInTx(r.Context(), eegID, backupEEGID, &backup); err != nil {
		slog.Error("restore failed", "eeg_id", eegID, "error", err)
		jsonError(w, "restore failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("restore complete",
		"eeg_id", eegID,
		"members", len(backup.Members),
		"meter_points", len(backup.MeterPoints),
		"readings", len(backup.Readings),
		"invoices", len(backup.Invoices),
	)
	jsonOK(w, map[string]any{
		"members":      len(backup.Members),
		"meter_points": len(backup.MeterPoints),
		"readings":     len(backup.Readings),
		"invoices":     len(backup.Invoices),
	})
}

func (h *BackupHandler) restoreInTx(ctx context.Context, targetEEGID, backupEEGID uuid.UUID, backup *EEGBackup) error {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete existing EEG data in FK-safe order.
	stmts := []string{
		`DELETE FROM energy_readings WHERE meter_point_id IN (SELECT id FROM meter_points WHERE eeg_id = $1)`,
		`DELETE FROM invoices WHERE eeg_id = $1`,
		`DELETE FROM billing_runs WHERE eeg_id = $1`,
		`DELETE FROM tariff_entries WHERE schedule_id IN (SELECT id FROM tariff_schedules WHERE eeg_id = $1)`,
		`DELETE FROM tariff_schedules WHERE eeg_id = $1`,
		`DELETE FROM eeg_meter_participations WHERE eeg_id = $1`,
		`DELETE FROM meter_points WHERE eeg_id = $1`,
		`DELETE FROM members WHERE eeg_id = $1`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(ctx, stmt, targetEEGID); err != nil {
			return fmt.Errorf("delete step failed: %w", err)
		}
	}

	// Restore EEG settings (keep the target EEG ID).
	eeg := backup.EEG
	_, err = tx.Exec(ctx, `UPDATE eegs SET
		name = $1, netzbetreiber = $2, energy_price = $3, producer_price = $4,
		use_vat = $5, vat_pct = $6, meter_fee_eur = $7, free_kwh = $8,
		discount_pct = $9, participation_fee_eur = $10, billing_period = $11,
		invoice_number_prefix = $12, invoice_number_digits = $13, invoice_number_start = $14,
		invoice_pre_text = $15, invoice_post_text = $16, invoice_footer_text = $17,
		generate_credit_notes = $18, credit_note_number_prefix = $19, credit_note_number_digits = $20,
		iban = $21, bic = $22, sepa_creditor_id = $23,
		eda_transition_date = $24, eda_marktpartner_id = $25, eda_netzbetreiber_id = $26,
		accounting_revenue_account = $27, accounting_expense_account = $28,
		accounting_debitor_prefix = $29, datev_consultant_nr = $30, datev_client_nr = $31,
		strasse = $32, plz = $33, ort = $34, uid_nummer = $35
		WHERE id = $36`,
		eeg.Name, eeg.Netzbetreiber, eeg.EnergyPrice, eeg.ProducerPrice,
		eeg.UseVat, eeg.VatPct, eeg.MeterFeeEur, eeg.FreeKwh,
		eeg.DiscountPct, eeg.ParticipationFeeEur, eeg.BillingPeriod,
		eeg.InvoiceNumberPrefix, eeg.InvoiceNumberDigits, eeg.InvoiceNumberStart,
		eeg.InvoicePreText, eeg.InvoicePostText, eeg.InvoiceFooterText,
		eeg.GenerateCreditNotes, eeg.CreditNoteNumberPrefix, eeg.CreditNoteNumberDigits,
		eeg.IBAN, eeg.BIC, eeg.SepaCreditorID,
		eeg.EdaTransitionDate, eeg.EdaMarktpartnerID, eeg.EdaNetzbetreiberID,
		eeg.AccountingRevenueAccount, eeg.AccountingExpenseAccount,
		eeg.AccountingDebitorPrefix, eeg.DatevConsultantNr, eeg.DatevClientNr,
		eeg.Strasse, eeg.Plz, eeg.Ort, eeg.UidNummer,
		targetEEGID,
	)
	if err != nil {
		return fmt.Errorf("update EEG settings: %w", err)
	}

	// Remap helper: replace backupEEGID with targetEEGID.
	remap := func(id uuid.UUID) uuid.UUID {
		if id == backupEEGID {
			return targetEEGID
		}
		return id
	}

	// Restore members.
	for _, m := range backup.Members {
		_, err = tx.Exec(ctx, `INSERT INTO members
			(id, eeg_id, mitglieds_nr, name1, name2, email, iban, strasse, plz, ort,
			 business_role, uid_nummer, use_vat, vat_pct, status, beitritt_datum, austritt_datum, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
			m.ID, remap(m.EegID), m.MitgliedsNr, m.Name1, m.Name2, m.Email, m.IBAN,
			m.Strasse, m.Plz, m.Ort, m.BusinessRole, m.UidNummer, m.UseVat, m.VatPct,
			m.Status, m.BeitrittsDatum, m.AustrittsDatum, m.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert member %s: %w", m.ID, err)
		}
	}

	// Restore meter points.
	for _, mp := range backup.MeterPoints {
		_, err = tx.Exec(ctx, `INSERT INTO meter_points
			(id, member_id, eeg_id, zaehlpunkt, energierichtung, verteilungsmodell,
			 zugeteilte_menge_pct, status, registriert_seit, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			mp.ID, mp.MemberID, remap(mp.EegID), mp.Zaehlpunkt, mp.Energierichtung,
			mp.Verteilungsmodell, mp.ZugeteilteMenugePct, mp.Status, mp.RegistriertSeit, mp.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert meter_point %s: %w", mp.ID, err)
		}
	}

	// Restore tariff schedules and entries.
	for _, s := range backup.TariffSchedules {
		_, err = tx.Exec(ctx, `INSERT INTO tariff_schedules
			(id, eeg_id, name, granularity, is_active, created_at)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			s.ID, remap(s.EegID), s.Name, s.Granularity, s.IsActive, s.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert tariff_schedule %s: %w", s.ID, err)
		}
		for _, e := range s.Entries {
			_, err = tx.Exec(ctx, `INSERT INTO tariff_entries
				(id, schedule_id, valid_from, valid_until, energy_price, producer_price, created_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				e.ID, s.ID, e.ValidFrom, e.ValidUntil, e.EnergyPrice, e.ProducerPrice, e.CreatedAt,
			)
			if err != nil {
				return fmt.Errorf("insert tariff_entry %s: %w", e.ID, err)
			}
		}
	}

	// Restore billing runs.
	for _, br := range backup.BillingRuns {
		_, err = tx.Exec(ctx, `INSERT INTO billing_runs
			(id, eeg_id, period_start, period_end, status, created_at)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			br.ID, remap(br.EegID), br.PeriodStart, br.PeriodEnd, br.Status, br.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert billing_run %s: %w", br.ID, err)
		}
	}

	// Restore invoices.
	for _, inv := range backup.Invoices {
		_, err = tx.Exec(ctx, `INSERT INTO invoices
			(id, member_id, eeg_id, period_start, period_end, total_kwh, total_amount,
			 net_amount, vat_amount, vat_pct_applied, consumption_kwh, generation_kwh,
			 pdf_path, storno_pdf_path, sent_at, invoice_number, status, billing_run_id,
			 document_type, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
			inv.ID, inv.MemberID, remap(inv.EegID),
			inv.PeriodStart, inv.PeriodEnd, inv.TotalKwh, inv.TotalAmount,
			inv.NetAmount, inv.VatAmount, inv.VatPctApplied, inv.ConsumptionKwh, inv.GenerationKwh,
			inv.PdfPath, inv.StornoPdfPath, inv.SentAt, inv.InvoiceNumber, inv.Status,
			inv.BillingRunID, inv.DocumentType, inv.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert invoice %s: %w", inv.ID, err)
		}
	}

	// Restore participation records.
	for _, p := range backup.Participations {
		_, err = tx.Exec(ctx, `INSERT INTO eeg_meter_participations
			(id, eeg_id, meter_point_id, participation_factor, share_type,
			 valid_from, valid_until, notes, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			p.ID, remap(p.EegID), p.MeterPointID, p.ParticipationFactor, p.ShareType,
			p.ValidFrom, p.ValidUntil, p.Notes, p.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert participation %s: %w", p.ID, err)
		}
	}

	// Restore energy readings.
	for _, rd := range backup.Readings {
		src := rd.Source
		if src == "" {
			src = "xlsx"
		}
		qual := rd.Quality
		if qual == "" {
			qual = "L0"
		}
		_, err = tx.Exec(ctx, `INSERT INTO energy_readings
			(meter_point_id, ts, wh_total, wh_community, wh_self, source, quality)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (meter_point_id, ts) DO NOTHING`,
			rd.MeterPointID, rd.Ts, rd.WhTotal, rd.WhCommunity, rd.WhSelf, src, qual,
		)
		if err != nil {
			return fmt.Errorf("insert reading: %w", err)
		}
	}

	return tx.Commit(ctx)
}
