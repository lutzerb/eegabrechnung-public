package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/crypto"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type EEGRepository struct {
	db     *pgxpool.Pool
	encKey []byte // AES-256 key for credential encryption; nil = credentials not stored/read
}

func NewEEGRepository(db *pgxpool.Pool, encKey []byte) *EEGRepository {
	return &EEGRepository{db: db, encKey: encKey}
}

// decrypt decrypts a credential ciphertext. Returns "" if ciphertext or key is empty.
func (r *EEGRepository) decrypt(enc string) string {
	if enc == "" || len(r.encKey) == 0 {
		return ""
	}
	pt, err := crypto.Decrypt(r.encKey, enc)
	if err != nil {
		return "" // bad ciphertext — treat as missing
	}
	return pt
}

// encrypt encrypts a plaintext credential. Returns "" if plaintext or key is empty.
func (r *EEGRepository) encrypt(pt string) (string, error) {
	if pt == "" || len(r.encKey) == 0 {
		return "", nil
	}
	return crypto.Encrypt(r.encKey, pt)
}

const eegCols = `id, organization_id, gemeinschaft_id, netzbetreiber, name,
	energy_price, producer_price, use_vat, vat_pct,
	meter_fee_eur, free_kwh, discount_pct, participation_fee_eur,
	billing_period,
	invoice_number_prefix, invoice_number_digits, invoice_number_start,
	invoice_pre_text, invoice_post_text, invoice_footer_text,
	logo_path,
	generate_credit_notes, credit_note_number_prefix, credit_note_number_digits,
	iban, bic, sepa_creditor_id,
	eda_transition_date, eda_marktpartner_id, eda_netzbetreiber_id,
	accounting_revenue_account, accounting_expense_account, accounting_debitor_prefix,
	datev_consultant_nr, datev_client_nr,
	strasse, plz, ort, uid_nummer,
	gruendungsdatum,
	onboarding_contract_text,
	sepa_pre_notification_days,
	is_demo,
	auto_billing_enabled, auto_billing_day_of_month, auto_billing_period, auto_billing_last_run_at,
	gap_alert_enabled, gap_alert_threshold_days,
	portal_show_full_energy,
	eda_imap_host, eda_imap_user, eda_imap_password_enc,
	eda_smtp_host, eda_smtp_user, eda_smtp_password_enc, eda_smtp_from,
	smtp_host, smtp_user, smtp_password_enc, smtp_from,
	created_at`

func (r *EEGRepository) scanEEG(row interface{ Scan(...any) error }, e *domain.EEG) error {
	// Use *string for all nullable credential columns.
	var edaImapHost, edaImapUser, edaImapPwEnc *string
	var edaSmtpHost, edaSmtpUser, edaSmtpPwEnc, edaSmtpFrom *string
	var smtpHost, smtpUser, smtpPwEnc, smtpFrom *string
	// Auto-billing nullable columns
	var autoBillingDayOfMonth *int
	var autoBillingPeriod *string
	err := row.Scan(
		&e.ID, &e.OrganizationID, &e.GemeinschaftID, &e.Netzbetreiber, &e.Name,
		&e.EnergyPrice, &e.ProducerPrice, &e.UseVat, &e.VatPct,
		&e.MeterFeeEur, &e.FreeKwh, &e.DiscountPct, &e.ParticipationFeeEur,
		&e.BillingPeriod,
		&e.InvoiceNumberPrefix, &e.InvoiceNumberDigits, &e.InvoiceNumberStart,
		&e.InvoicePreText, &e.InvoicePostText, &e.InvoiceFooterText,
		&e.LogoPath,
		&e.GenerateCreditNotes, &e.CreditNoteNumberPrefix, &e.CreditNoteNumberDigits,
		&e.IBAN, &e.BIC, &e.SepaCreditorID,
		&e.EdaTransitionDate, &e.EdaMarktpartnerID, &e.EdaNetzbetreiberID,
		&e.AccountingRevenueAccount, &e.AccountingExpenseAccount, &e.AccountingDebitorPrefix,
		&e.DatevConsultantNr, &e.DatevClientNr,
		&e.Strasse, &e.Plz, &e.Ort, &e.UidNummer,
		&e.Gruendungsdatum,
		&e.OnboardingContractText,
		&e.SepaPreNotificationDays,
		&e.IsDemo,
		&e.AutoBillingEnabled, &autoBillingDayOfMonth, &autoBillingPeriod, &e.AutoBillingLastRunAt,
		&e.GapAlertEnabled, &e.GapAlertThresholdDays,
		&e.PortalShowFullEnergy,
		&edaImapHost, &edaImapUser, &edaImapPwEnc,
		&edaSmtpHost, &edaSmtpUser, &edaSmtpPwEnc, &edaSmtpFrom,
		&smtpHost, &smtpUser, &smtpPwEnc, &smtpFrom,
		&e.CreatedAt,
	)
	if err != nil {
		return err
	}
	derefStr := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	if autoBillingDayOfMonth != nil {
		e.AutoBillingDayOfMonth = *autoBillingDayOfMonth
	}
	if autoBillingPeriod != nil {
		e.AutoBillingPeriod = *autoBillingPeriod
	}
	e.EDAIMAPHost = derefStr(edaImapHost)
	e.EDAIMAPUser = derefStr(edaImapUser)
	e.EDASmtpHost = derefStr(edaSmtpHost)
	e.EDASmtpUser = derefStr(edaSmtpUser)
	e.EDASmtpFrom = derefStr(edaSmtpFrom)
	e.SMTPHost = derefStr(smtpHost)
	e.SMTPUser = derefStr(smtpUser)
	e.SMTPFrom = derefStr(smtpFrom)
	if edaImapPwEnc != nil {
		e.EDAIMAPPassword = r.decrypt(*edaImapPwEnc)
	}
	if edaSmtpPwEnc != nil {
		e.EDASmtpPassword = r.decrypt(*edaSmtpPwEnc)
	}
	if smtpPwEnc != nil {
		e.SMTPPassword = r.decrypt(*smtpPwEnc)
	}
	return nil
}

func (r *EEGRepository) Create(ctx context.Context, eeg *domain.EEG) error {
	q := `INSERT INTO eegs
	        (organization_id, gemeinschaft_id, netzbetreiber, name, energy_price, producer_price,
	         use_vat, vat_pct, meter_fee_eur, free_kwh, discount_pct,
	         participation_fee_eur, billing_period,
	         invoice_number_prefix, invoice_number_digits, invoice_number_start,
	         invoice_pre_text, invoice_post_text, invoice_footer_text,
	         logo_path,
	         generate_credit_notes, credit_note_number_prefix, credit_note_number_digits,
	         iban, bic, sepa_creditor_id)
	      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)
	      RETURNING id, created_at`
	if eeg.BillingPeriod == "" {
		eeg.BillingPeriod = "monthly"
	}
	if eeg.VatPct == 0 {
		eeg.VatPct = 20
	}
	if eeg.InvoiceNumberPrefix == "" {
		eeg.InvoiceNumberPrefix = "INV"
	}
	if eeg.InvoiceNumberDigits == 0 {
		eeg.InvoiceNumberDigits = 5
	}
	if eeg.CreditNoteNumberPrefix == "" {
		eeg.CreditNoteNumberPrefix = "GS"
	}
	if eeg.CreditNoteNumberDigits == 0 {
		eeg.CreditNoteNumberDigits = 5
	}
	return r.db.QueryRow(ctx, q,
		eeg.OrganizationID, eeg.GemeinschaftID, eeg.Netzbetreiber, eeg.Name, eeg.EnergyPrice, eeg.ProducerPrice,
		eeg.UseVat, eeg.VatPct, eeg.MeterFeeEur, eeg.FreeKwh, eeg.DiscountPct,
		eeg.ParticipationFeeEur, eeg.BillingPeriod,
		eeg.InvoiceNumberPrefix, eeg.InvoiceNumberDigits, eeg.InvoiceNumberStart,
		eeg.InvoicePreText, eeg.InvoicePostText, eeg.InvoiceFooterText,
		eeg.LogoPath,
		eeg.GenerateCreditNotes, eeg.CreditNoteNumberPrefix, eeg.CreditNoteNumberDigits,
		eeg.IBAN, eeg.BIC, eeg.SepaCreditorID,
	).Scan(&eeg.ID, &eeg.CreatedAt)
}

func (r *EEGRepository) Update(ctx context.Context, eeg *domain.EEG) error {
	// Encrypt credential passwords before storing.
	edaImapPwEnc, err := r.encrypt(eeg.EDAIMAPPassword)
	if err != nil {
		return fmt.Errorf("encrypt eda_imap_password: %w", err)
	}
	edaSmtpPwEnc, err := r.encrypt(eeg.EDASmtpPassword)
	if err != nil {
		return fmt.Errorf("encrypt eda_smtp_password: %w", err)
	}
	smtpPwEnc, err := r.encrypt(eeg.SMTPPassword)
	if err != nil {
		return fmt.Errorf("encrypt smtp_password: %w", err)
	}

	q := `UPDATE eegs SET
	        gemeinschaft_id=$1, netzbetreiber=$2, name=$3,
	        energy_price=$4, producer_price=$5, use_vat=$6, vat_pct=$7,
	        meter_fee_eur=$8, free_kwh=$9, discount_pct=$10,
	        participation_fee_eur=$11, billing_period=$12,
	        invoice_number_prefix=$13, invoice_number_digits=$14, invoice_number_start=$15,
	        invoice_pre_text=$16, invoice_post_text=$17, invoice_footer_text=$18,
	        generate_credit_notes=$19, credit_note_number_prefix=$20, credit_note_number_digits=$21,
	        iban=$22, bic=$23, sepa_creditor_id=$24,
	        eda_transition_date=$25, eda_marktpartner_id=$26, eda_netzbetreiber_id=$27,
	        accounting_revenue_account=$28, accounting_expense_account=$29, accounting_debitor_prefix=$30,
	        datev_consultant_nr=$31, datev_client_nr=$32,
	        strasse=$33, plz=$34, ort=$35, uid_nummer=$36,
	        gruendungsdatum=$37,
	        onboarding_contract_text=$38,
	        eda_imap_host=NULLIF($39,''), eda_imap_user=NULLIF($40,''), eda_imap_password_enc=NULLIF($41,''),
	        eda_smtp_host=NULLIF($42,''), eda_smtp_user=NULLIF($43,''), eda_smtp_password_enc=NULLIF($44,''), eda_smtp_from=NULLIF($45,''),
	        smtp_host=NULLIF($46,''), smtp_user=NULLIF($47,''), smtp_password_enc=NULLIF($48,''), smtp_from=NULLIF($49,''),
	        auto_billing_enabled=$52, auto_billing_day_of_month=NULLIF($53,0), auto_billing_period=NULLIF($54,''),
	        gap_alert_enabled=$55, gap_alert_threshold_days=$56,
	        sepa_pre_notification_days=$57,
	        portal_show_full_energy=$58
	      WHERE id=$50 AND organization_id=$51`
	// Note: logo_path and auto_billing_last_run_at are not updated via this method.
	days := eeg.SepaPreNotificationDays
	if days <= 0 {
		days = 14
	}
	_, err = r.db.Exec(ctx, q,
		eeg.GemeinschaftID, eeg.Netzbetreiber, eeg.Name,
		eeg.EnergyPrice, eeg.ProducerPrice, eeg.UseVat, eeg.VatPct,
		eeg.MeterFeeEur, eeg.FreeKwh, eeg.DiscountPct,
		eeg.ParticipationFeeEur, eeg.BillingPeriod,
		eeg.InvoiceNumberPrefix, eeg.InvoiceNumberDigits, eeg.InvoiceNumberStart,
		eeg.InvoicePreText, eeg.InvoicePostText, eeg.InvoiceFooterText,
		eeg.GenerateCreditNotes, eeg.CreditNoteNumberPrefix, eeg.CreditNoteNumberDigits,
		eeg.IBAN, eeg.BIC, eeg.SepaCreditorID,
		eeg.EdaTransitionDate, eeg.EdaMarktpartnerID, eeg.EdaNetzbetreiberID,
		eeg.AccountingRevenueAccount, eeg.AccountingExpenseAccount, eeg.AccountingDebitorPrefix,
		eeg.DatevConsultantNr, eeg.DatevClientNr,
		eeg.Strasse, eeg.Plz, eeg.Ort, eeg.UidNummer,
		eeg.Gruendungsdatum,
		eeg.OnboardingContractText,
		eeg.EDAIMAPHost, eeg.EDAIMAPUser, edaImapPwEnc,
		eeg.EDASmtpHost, eeg.EDASmtpUser, edaSmtpPwEnc, eeg.EDASmtpFrom,
		eeg.SMTPHost, eeg.SMTPUser, smtpPwEnc, eeg.SMTPFrom,
		eeg.ID, eeg.OrganizationID,
		eeg.AutoBillingEnabled, eeg.AutoBillingDayOfMonth, eeg.AutoBillingPeriod,
		eeg.GapAlertEnabled, eeg.GapAlertThresholdDays,
		days,
		eeg.PortalShowFullEnergy,
	)
	return err
}

// ListGapAlertEEGs returns all EEGs with gap_alert_enabled = true.
func (r *EEGRepository) ListGapAlertEEGs(ctx context.Context) ([]*domain.EEG, error) {
	q := `SELECT ` + eegCols + ` FROM eegs WHERE gap_alert_enabled = true`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.EEG
	for rows.Next() {
		var e domain.EEG
		if err := r.scanEEG(rows, &e); err != nil {
			return nil, err
		}
		result = append(result, &e)
	}
	return result, rows.Err()
}

// ListAutoBillingEEGs returns all EEGs with auto_billing_enabled = true.
func (r *EEGRepository) ListAutoBillingEEGs(ctx context.Context) ([]*domain.EEG, error) {
	q := `SELECT ` + eegCols + ` FROM eegs WHERE auto_billing_enabled = true`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.EEG
	for rows.Next() {
		var e domain.EEG
		if err := r.scanEEG(rows, &e); err != nil {
			return nil, err
		}
		result = append(result, &e)
	}
	return result, rows.Err()
}

// UpdateAutoBillingLastRun sets auto_billing_last_run_at to now for the given EEG.
func (r *EEGRepository) UpdateAutoBillingLastRun(ctx context.Context, eegID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE eegs SET auto_billing_last_run_at = now() WHERE id = $1`,
		eegID,
	)
	return err
}

// ListEEGsWithIMAPCredentials returns all EEGs that have IMAP credentials configured.
// Used by the EDA worker to poll per-EEG mailboxes.
func (r *EEGRepository) ListEEGsWithIMAPCredentials(ctx context.Context) ([]*domain.EEG, error) {
	q := `SELECT ` + eegCols + `
	      FROM eegs
	      WHERE eda_imap_host IS NOT NULL AND eda_imap_user IS NOT NULL AND eda_imap_password_enc IS NOT NULL
	      ORDER BY name`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var result []*domain.EEG
	for rows.Next() {
		var e domain.EEG
		if err := r.scanEEG(rows, &e); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, &e)
	}
	return result, rows.Err()
}

// UpdateLogo sets the logo_path for the given EEG.
func (r *EEGRepository) UpdateLogo(ctx context.Context, id uuid.UUID, logoPath string) error {
	_, err := r.db.Exec(ctx, `UPDATE eegs SET logo_path=$1 WHERE id=$2`, logoPath, id)
	return err
}

func (r *EEGRepository) List(ctx context.Context, orgID uuid.UUID) ([]domain.EEG, error) {
	q := `SELECT ` + eegCols + ` FROM eegs WHERE organization_id=$1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var eegs []domain.EEG
	for rows.Next() {
		var e domain.EEG
		if err := r.scanEEG(rows, &e); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		eegs = append(eegs, e)
	}
	return eegs, rows.Err()
}

// ListForUser returns only EEGs that are assigned to the given user.
func (r *EEGRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]domain.EEG, error) {
	q := `SELECT ` + eegCols + `
	      FROM eegs e
	      INNER JOIN user_eeg_assignments a ON a.eeg_id = e.id
	      WHERE a.user_id = $1
	      ORDER BY e.name`
	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var eegs []domain.EEG
	for rows.Next() {
		var e domain.EEG
		if err := r.scanEEG(rows, &e); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		eegs = append(eegs, e)
	}
	return eegs, rows.Err()
}

func (r *EEGRepository) GetByID(ctx context.Context, id, orgID uuid.UUID) (*domain.EEG, error) {
	q := `SELECT ` + eegCols + ` FROM eegs WHERE id=$1 AND organization_id=$2`
	var e domain.EEG
	if err := r.scanEEG(r.db.QueryRow(ctx, q, id, orgID), &e); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &e, nil
}

// GetByIDForUser fetches an EEG only when it is explicitly assigned to the user.
func (r *EEGRepository) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*domain.EEG, error) {
	q := `SELECT ` + eegCols + `
	      FROM eegs e
	      INNER JOIN user_eeg_assignments a ON a.eeg_id = e.id
	      WHERE e.id = $1 AND a.user_id = $2`
	var e domain.EEG
	if err := r.scanEEG(r.db.QueryRow(ctx, q, id, userID), &e); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &e, nil
}

// GetByIDInternal fetches an EEG without org scoping — for internal service use only.
func (r *EEGRepository) GetByIDInternal(ctx context.Context, id uuid.UUID) (*domain.EEG, error) {
	q := `SELECT ` + eegCols + ` FROM eegs WHERE id=$1`
	var e domain.EEG
	if err := r.scanEEG(r.db.QueryRow(ctx, q, id), &e); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &e, nil
}

func (r *EEGRepository) GetByGemeinschaftID(ctx context.Context, gemeinschaftID string) (*domain.EEG, error) {
	q := `SELECT ` + eegCols + ` FROM eegs WHERE gemeinschaft_id=$1`
	var e domain.EEG
	if err := r.scanEEG(r.db.QueryRow(ctx, q, gemeinschaftID), &e); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &e, nil
}

// GetStats returns aggregate statistics for a given EEG.
func (r *EEGRepository) GetStats(ctx context.Context, eegID uuid.UUID) (*domain.EEGStats, error) {
	q := `SELECT
		(SELECT COUNT(*) FROM members WHERE eeg_id=$1) AS member_count,
		(SELECT COUNT(*) FROM meter_points mp JOIN members m ON mp.member_id=m.id WHERE m.eeg_id=$1) AS meter_point_count,
		(SELECT COUNT(*) FROM invoices WHERE eeg_id=$1 AND status != 'cancelled') AS invoice_count,
		(SELECT COUNT(*) FROM billing_runs WHERE eeg_id=$1 AND status != 'cancelled') AS billing_run_count,
		(SELECT COALESCE(SUM(consumption_kwh),0) FROM invoices WHERE eeg_id=$1 AND status != 'cancelled') AS total_kwh,
		(SELECT COALESCE(SUM(total_amount),0) FROM invoices WHERE eeg_id=$1 AND status != 'cancelled' AND total_amount > 0) AS total_revenue,
		(SELECT MAX(created_at) FROM billing_runs WHERE eeg_id=$1 AND status != 'cancelled') AS last_billing_run`
	var s domain.EEGStats
	err := r.db.QueryRow(ctx, q, eegID).Scan(
		&s.MemberCount, &s.MeterPointCount, &s.InvoiceCount, &s.BillingRunCount,
		&s.TotalKwh, &s.TotalRevenue, &s.LastBillingRun,
	)
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &s, nil
}

// Delete permanently removes an EEG and all its data (cascaded by the DB).
// Verifies org ownership before deleting.
func (r *EEGRepository) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM eegs WHERE id = $1 AND organization_id = $2`, id, orgID)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}
