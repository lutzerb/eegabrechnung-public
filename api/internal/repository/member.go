package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type MemberRepository struct {
	db *pgxpool.Pool
}

func NewMemberRepository(db *pgxpool.Pool) *MemberRepository {
	return &MemberRepository{db: db}
}

func (r *MemberRepository) Upsert(ctx context.Context, m *domain.Member) error {
	q := `INSERT INTO members (eeg_id, mitglieds_nr, name1, name2, email, iban, strasse, plz, ort, business_role, beitritt_datum, austritt_datum)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	      ON CONFLICT (eeg_id, mitglieds_nr) DO UPDATE
	        SET name1 = EXCLUDED.name1,
	            name2 = EXCLUDED.name2,
	            email = EXCLUDED.email,
	            iban  = EXCLUDED.iban,
	            strasse = EXCLUDED.strasse,
	            plz = EXCLUDED.plz,
	            ort = EXCLUDED.ort,
	            business_role = EXCLUDED.business_role,
	            beitritt_datum = EXCLUDED.beitritt_datum,
	            austritt_datum = EXCLUDED.austritt_datum
	      RETURNING id, uid_nummer, use_vat, vat_pct, status, created_at`
	return r.db.QueryRow(ctx, q,
		m.EegID, m.MitgliedsNr, m.Name1, m.Name2, m.Email, m.IBAN, m.Strasse, m.Plz, m.Ort, m.BusinessRole,
		m.BeitrittsDatum, m.AustrittsDatum,
	).Scan(&m.ID, &m.UidNummer, &m.UseVat, &m.VatPct, &m.Status, &m.CreatedAt)
}

const memberCols = `id, eeg_id, mitglieds_nr, kundenkonto, name1, name2, email, iban, strasse, plz, ort,
	business_role, uid_nummer, use_vat, vat_pct, status, beitritt_datum, austritt_datum,
	sepa_mandate_signed_at, sepa_mandate_signed_ip, sepa_mandate_text, created_at,
	referral_code, referred_by_member_id, tariff_schedule_id`

func scanMember(row interface{ Scan(...any) error }, m *domain.Member) error {
	var mandateIP *string
	var mandateText *string
	var referralCode *string
	err := row.Scan(
		&m.ID, &m.EegID, &m.MitgliedsNr, &m.Kundenkonto, &m.Name1, &m.Name2, &m.Email, &m.IBAN, &m.Strasse, &m.Plz, &m.Ort,
		&m.BusinessRole, &m.UidNummer, &m.UseVat, &m.VatPct, &m.Status, &m.BeitrittsDatum, &m.AustrittsDatum,
		&m.SepaMandateSignedAt, &mandateIP, &mandateText, &m.CreatedAt,
		&referralCode, &m.ReferredByMemberID, &m.TariffScheduleID,
	)
	if mandateIP != nil {
		m.SepaMandateSignedIP = *mandateIP
	}
	if mandateText != nil {
		m.SepaMandateText = *mandateText
	}
	if referralCode != nil {
		m.ReferralCode = *referralCode
	}
	return err
}

func (r *MemberRepository) GetByEegAndMitgliedsNr(ctx context.Context, eegID uuid.UUID, mitgliedsNr string) (*domain.Member, error) {
	q := `SELECT ` + memberCols + ` FROM members WHERE eeg_id = $1 AND mitglieds_nr = $2`
	var m domain.Member
	if err := scanMember(r.db.QueryRow(ctx, q, eegID, mitgliedsNr), &m); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &m, nil
}

// ExistsByEmailInEEG returns true if any active (non-INACTIVE) member with that email exists in the EEG.
func (r *MemberRepository) ExistsByEmailInEEG(ctx context.Context, eegID uuid.UUID, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM members WHERE eeg_id=$1 AND lower(email)=lower($2) AND status != 'INACTIVE')`,
		eegID, email,
	).Scan(&exists)
	return exists, err
}

func (r *MemberRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Member, error) {
	q := `SELECT ` + memberCols + ` FROM members WHERE id = $1`
	var m domain.Member
	if err := scanMember(r.db.QueryRow(ctx, q, id), &m); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &m, nil
}

func (r *MemberRepository) ListByEeg(ctx context.Context, eegID uuid.UUID) ([]domain.Member, error) {
	q := `SELECT ` + memberCols + ` FROM members WHERE eeg_id = $1 ORDER BY mitglieds_nr`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var members []domain.Member
	for rows.Next() {
		var m domain.Member
		if err := scanMember(rows, &m); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// SearchByEeg filters members by optional full-text search (name/email/mitglieds_nr), status, and Stichtag.
// When stichtag is set, only members active at that date are returned:
// (beitritt_datum IS NULL OR beitritt_datum <= stichtag) AND (austritt_datum IS NULL OR austritt_datum > stichtag)
func (r *MemberRepository) SearchByEeg(ctx context.Context, eegID uuid.UUID, query, status string, stichtag *time.Time) ([]domain.Member, error) {
	args := []any{eegID}
	where := "eeg_id = $1"
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if query != "" {
		args = append(args, "%"+query+"%")
		n := len(args)
		where += fmt.Sprintf(" AND (name1 ILIKE $%d OR name2 ILIKE $%d OR email ILIKE $%d OR mitglieds_nr ILIKE $%d)", n, n, n, n)
	}
	if stichtag != nil {
		args = append(args, *stichtag)
		n := len(args)
		where += fmt.Sprintf(" AND (beitritt_datum IS NULL OR beitritt_datum <= $%d) AND (austritt_datum IS NULL OR austritt_datum > $%d)", n, n)
	}
	q := `SELECT ` + memberCols + ` FROM members WHERE ` + where + ` ORDER BY mitglieds_nr`
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var members []domain.Member
	for rows.Next() {
		var m domain.Member
		if err := scanMember(rows, &m); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *MemberRepository) SearchPreviewByEeg(ctx context.Context, eegID uuid.UUID, query string, limit int) ([]domain.Member, error) {
	q := `SELECT ` + memberCols + `
	      FROM members
	      WHERE eeg_id = $1
	        AND (name1 ILIKE $2 OR name2 ILIKE $2 OR email ILIKE $2 OR mitglieds_nr ILIKE $2)
	      ORDER BY mitglieds_nr
	      LIMIT $3`
	rows, err := r.db.Query(ctx, q, eegID, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var members []domain.Member
	for rows.Next() {
		var m domain.Member
		if err := scanMember(rows, &m); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// NextMemberNumber returns the next available member number for an EEG,
// derived from the highest existing numeric suffix, formatted as %04d.
func (r *MemberRepository) NextMemberNumber(ctx context.Context, eegID uuid.UUID) (string, error) {
	var maxNr int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(MAX(CAST(NULLIF(REGEXP_REPLACE(mitglieds_nr, '[^0-9]', '', 'g'), '') AS int)), 0) + 1
		 FROM members WHERE eeg_id = $1`,
		eegID,
	).Scan(&maxNr)
	if err != nil {
		return "", fmt.Errorf("next member number: %w", err)
	}
	return fmt.Sprintf("%04d", maxNr), nil
}

func (r *MemberRepository) Create(ctx context.Context, m *domain.Member) error {
	if m.Status == "" {
		m.Status = "ACTIVE"
	}
	q := `INSERT INTO members (eeg_id, mitglieds_nr, kundenkonto, name1, name2, email, iban, strasse, plz, ort, business_role, uid_nummer, use_vat, vat_pct, status, beitritt_datum, austritt_datum, tariff_schedule_id)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	      RETURNING id, created_at`
	return r.db.QueryRow(ctx, q,
		m.EegID, m.MitgliedsNr, m.Kundenkonto, m.Name1, m.Name2, m.Email, m.IBAN, m.Strasse, m.Plz, m.Ort,
		m.BusinessRole, m.UidNummer, m.UseVat, m.VatPct, m.Status,
		m.BeitrittsDatum, m.AustrittsDatum, m.TariffScheduleID,
	).Scan(&m.ID, &m.CreatedAt)
}

func (r *MemberRepository) Update(ctx context.Context, m *domain.Member) error {
	if m.Status == "" {
		m.Status = "ACTIVE"
	}
	q := `UPDATE members SET
	        name1=$1, name2=$2, email=$3, iban=$4, strasse=$5, plz=$6, ort=$7,
	        business_role=$8, mitglieds_nr=$9, uid_nummer=$10, use_vat=$11, vat_pct=$12, status=$13,
	        beitritt_datum=$14, austritt_datum=$15, tariff_schedule_id=$16, kundenkonto=$17
	      WHERE id=$18`
	_, err := r.db.Exec(ctx, q,
		m.Name1, m.Name2, m.Email, m.IBAN, m.Strasse, m.Plz, m.Ort,
		m.BusinessRole, m.MitgliedsNr, m.UidNummer, m.UseVat, m.VatPct, m.Status,
		m.BeitrittsDatum, m.AustrittsDatum, m.TariffScheduleID, m.Kundenkonto, m.ID,
	)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	return nil
}

// NextKundenkontoSuggestion returns a suggested next Kundenkonto value for an EEG:
// one more than the highest existing numeric Kundenkonto in that EEG, floored at
// eegs.kundenkonto_start. Purely advisory (used to prefill the create-member form) —
// the field stays freely editable, so no row locking is needed here. There is
// deliberately no cross-EEG or org-wide sequence; each EEG is numbered independently.
func (r *MemberRepository) NextKundenkontoSuggestion(ctx context.Context, eegID uuid.UUID) (string, error) {
	var start int
	if err := r.db.QueryRow(ctx, `SELECT kundenkonto_start FROM eegs WHERE id = $1`, eegID).Scan(&start); err != nil {
		return "", fmt.Errorf("load kundenkonto_start: %w", err)
	}
	var currentMax int
	q := `SELECT COALESCE(MAX(CAST(NULLIF(REGEXP_REPLACE(kundenkonto, '[^0-9]', '', 'g'), '') AS int)), 0)
	      FROM members WHERE eeg_id = $1`
	if err := r.db.QueryRow(ctx, q, eegID).Scan(&currentMax); err != nil {
		return "", fmt.Errorf("next kundenkonto suggestion: %w", err)
	}
	next := currentMax + 1
	if next < start {
		next = start
	}
	return strconv.Itoa(next), nil
}

// BulkAssignKundenkonto assigns sequential Kundenkonto numbers to every member of
// eegID that doesn't have one yet (optionally restricted to memberIDs), starting
// after the current max and floored at the EEG's own kundenkonto_start — never a
// cross-EEG or org-wide sequence. Locks the eegs row for the duration of the
// transaction, mirroring InvoiceRepository.nextNumberTx's locking pattern.
func (r *MemberRepository) BulkAssignKundenkonto(ctx context.Context, eegID uuid.UUID, memberIDs []uuid.UUID) ([]domain.Member, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin kundenkonto tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var start int
	if err := tx.QueryRow(ctx, `SELECT kundenkonto_start FROM eegs WHERE id=$1 FOR UPDATE`, eegID).Scan(&start); err != nil {
		return nil, fmt.Errorf("lock eeg for kundenkonto: %w", err)
	}

	var currentMax int
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(CAST(NULLIF(REGEXP_REPLACE(kundenkonto, '[^0-9]', '', 'g'), '') AS int)), 0)
		 FROM members WHERE eeg_id = $1`, eegID).Scan(&currentMax); err != nil {
		return nil, fmt.Errorf("query current max kundenkonto: %w", err)
	}
	next := currentMax + 1
	if next < start {
		next = start
	}

	args := []any{eegID}
	where := "eeg_id = $1 AND kundenkonto = ''"
	if len(memberIDs) > 0 {
		args = append(args, memberIDs)
		where += fmt.Sprintf(" AND id = ANY($%d)", len(args))
	}
	rows, err := tx.Query(ctx, `SELECT `+memberCols+` FROM members WHERE `+where+` ORDER BY mitglieds_nr FOR UPDATE`, args...)
	if err != nil {
		return nil, fmt.Errorf("select members for kundenkonto: %w", err)
	}
	var targets []domain.Member
	for rows.Next() {
		var m domain.Member
		if err := scanMember(rows, &m); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan: %w", err)
		}
		targets = append(targets, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	assigned := make([]domain.Member, 0, len(targets))
	for _, m := range targets {
		m.Kundenkonto = strconv.Itoa(next)
		next++
		if _, err := tx.Exec(ctx, `UPDATE members SET kundenkonto=$1 WHERE id=$2`, m.Kundenkonto, m.ID); err != nil {
			return nil, fmt.Errorf("assign kundenkonto to %s: %w", m.ID, err)
		}
		assigned = append(assigned, m)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit kundenkonto tx: %w", err)
	}
	return assigned, nil
}

// IsAssignableTariff reports whether scheduleID is an active EEG-wide (non-Individualtarif)
// plan of the given EEG, i.e. a valid target for members.tariff_schedule_id.
func (r *MemberRepository) IsAssignableTariff(ctx context.Context, eegID, scheduleID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM tariff_schedules WHERE id = $1 AND eeg_id = $2 AND member_id IS NULL AND is_active)`,
		scheduleID, eegID).Scan(&ok)
	return ok, err
}

// EffectiveTariffs returns, per member of the EEG, the tariff plan that is valid right now:
// an active Individualtarif, else the assigned EEG-wide plan, else the Standardtarif.
// Source "none" means no plan applies (flat EEG prices).
func (r *MemberRepository) EffectiveTariffs(ctx context.Context, eegID uuid.UUID) (map[uuid.UUID]domain.EffectiveTariff, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id,
		       CASE WHEN ind.id IS NOT NULL THEN 'individual'
		            WHEN asg.id IS NOT NULL THEN 'assigned'
		            WHEN def.id IS NOT NULL THEN 'default'
		            ELSE 'none' END,
		       COALESCE(ind.id, asg.id, def.id),
		       COALESCE(ind.name, asg.name, def.name, '')
		FROM members m
		LEFT JOIN tariff_schedules ind ON ind.member_id = m.id AND ind.is_active
		LEFT JOIN tariff_schedules asg ON asg.id = m.tariff_schedule_id AND asg.is_active AND asg.member_id IS NULL
		LEFT JOIN tariff_schedules def ON def.eeg_id = m.eeg_id AND def.is_default AND def.member_id IS NULL
		WHERE m.eeg_id = $1`, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	result := make(map[uuid.UUID]domain.EffectiveTariff)
	for rows.Next() {
		var memberID uuid.UUID
		var et domain.EffectiveTariff
		if err := rows.Scan(&memberID, &et.Source, &et.ScheduleID, &et.Name); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result[memberID] = et
	}
	return result, rows.Err()
}

// ActivateByMeterPoint sets the member's status to ACTIVE when an EDA Anmeldung
// is confirmed. Only transitions from REGISTERED → ACTIVE (idempotent on re-confirm).
func (r *MemberRepository) ActivateByMeterPoint(ctx context.Context, meterPointID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE members SET status = 'ACTIVE'
		 WHERE id = (SELECT member_id FROM meter_points WHERE id = $1)
		   AND status = 'REGISTERED'`,
		meterPointID,
	)
	return err
}

func (r *MemberRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Remove onboarding link (keep the request, just unlink the converted member)
	if _, err := tx.Exec(ctx,
		`UPDATE onboarding_requests SET converted_member_id = NULL WHERE converted_member_id = $1`, id,
	); err != nil {
		return fmt.Errorf("unlink onboarding: %w", err)
	}

	// Delete EDA processes for this member's meter points
	if _, err := tx.Exec(ctx,
		`DELETE FROM eda_processes WHERE meter_point_id IN (SELECT id FROM meter_points WHERE member_id = $1)`, id,
	); err != nil {
		return fmt.Errorf("delete eda_processes: %w", err)
	}

	// Delete energy readings for this member's meter points
	if _, err := tx.Exec(ctx,
		`DELETE FROM energy_readings WHERE meter_point_id IN (SELECT id FROM meter_points WHERE member_id = $1)`, id,
	); err != nil {
		return fmt.Errorf("delete energy_readings: %w", err)
	}

	// Delete invoices for this member
	if _, err := tx.Exec(ctx,
		`DELETE FROM invoices WHERE member_id = $1`, id,
	); err != nil {
		return fmt.Errorf("delete invoices: %w", err)
	}

	// Delete meter points
	if _, err := tx.Exec(ctx,
		`DELETE FROM meter_points WHERE member_id = $1`, id,
	); err != nil {
		return fmt.Errorf("delete meter_points: %w", err)
	}

	// Delete the member
	if _, err := tx.Exec(ctx, `DELETE FROM members WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete member: %w", err)
	}

	return tx.Commit(ctx)
}

// ArchiveMandate copies the member's currently-stored mandate snapshot (IBAN,
// signature date/IP/text) into member_sepa_mandate_history, tagged with reason.
// Call this BEFORE overwriting the in-memory member's mandate fields, since it
// archives whatever is passed in m.
func (r *MemberRepository) ArchiveMandate(ctx context.Context, m *domain.Member, reason string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO member_sepa_mandate_history (member_id, eeg_id, iban, signed_at, signed_ip, signed_text, reason)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		m.ID, m.EegID, m.IBAN, m.SepaMandateSignedAt, m.SepaMandateSignedIP, m.SepaMandateText, reason,
	)
	if err != nil {
		return fmt.Errorf("archive mandate: %w", err)
	}
	return nil
}

// ListMandateHistory returns archived mandate snapshots for a member, newest first.
func (r *MemberRepository) ListMandateHistory(ctx context.Context, memberID uuid.UUID) ([]domain.SepaMandateHistoryEntry, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, member_id, eeg_id, iban, signed_at, signed_ip, signed_text, archived_at, reason
		 FROM member_sepa_mandate_history WHERE member_id = $1 ORDER BY archived_at DESC`,
		memberID,
	)
	if err != nil {
		return nil, fmt.Errorf("query mandate history: %w", err)
	}
	defer rows.Close()

	var entries []domain.SepaMandateHistoryEntry
	for rows.Next() {
		var e domain.SepaMandateHistoryEntry
		if err := rows.Scan(&e.ID, &e.MemberID, &e.EegID, &e.IBAN, &e.SignedAt, &e.SignedIP, &e.SignedText, &e.ArchivedAt, &e.Reason); err != nil {
			return nil, fmt.Errorf("scan mandate history: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ClearMandateSignature nulls out the signature fields on a member (IBAN itself is left
// untouched — caller updates that separately). Used when an admin edits the IBAN directly:
// the old mandate is archived, but no new signature exists until the member re-confirms
// via the self-service portal.
func (r *MemberRepository) ClearMandateSignature(ctx context.Context, memberID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE members SET sepa_mandate_signed_at = NULL, sepa_mandate_signed_ip = '', sepa_mandate_text = '' WHERE id = $1`,
		memberID,
	)
	if err != nil {
		return fmt.Errorf("clear mandate signature: %w", err)
	}
	return nil
}

// SetMandate archives the member's current mandate snapshot, then atomically stores a newly
// signed one (new IBAN + signature fields). Used by the member portal self-service IBAN
// change flow, where the member themselves confirms the new mandate.
func (r *MemberRepository) SetMandate(ctx context.Context, m *domain.Member, newIBAN string, signedAt time.Time, signedIP, signedText string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`INSERT INTO member_sepa_mandate_history (member_id, eeg_id, iban, signed_at, signed_ip, signed_text, reason)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		m.ID, m.EegID, m.IBAN, m.SepaMandateSignedAt, m.SepaMandateSignedIP, m.SepaMandateText, "iban_change_portal",
	); err != nil {
		return fmt.Errorf("archive mandate: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE members SET iban = $1, sepa_mandate_signed_at = $2, sepa_mandate_signed_ip = $3, sepa_mandate_text = $4 WHERE id = $5`,
		newIBAN, signedAt, signedIP, signedText, m.ID,
	); err != nil {
		return fmt.Errorf("update mandate: %w", err)
	}

	return tx.Commit(ctx)
}

// MeterPointInfo holds the meter point ID and direction for PDF/billing use.
type MeterPointInfo struct {
	Zaehlpunkt string
	Richtung   string
}

// generateReferralCode creates a random referral code (8 bytes hex = 16 chars) —
// long enough to resist enumeration (there is no public code-lookup endpoint;
// codes are only ever resolved server-side, scoped to one EEG, at onboarding
// submit time), short enough to share as a link.
func generateReferralCode() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GetOrCreateReferralCode returns the member's personal "Mitglieder werben
// Mitglieder" referral code, generating and persisting one on first use.
func (r *MemberRepository) GetOrCreateReferralCode(ctx context.Context, memberID uuid.UUID) (string, error) {
	var code *string
	if err := r.db.QueryRow(ctx, `SELECT referral_code FROM members WHERE id = $1`, memberID).Scan(&code); err != nil {
		return "", fmt.Errorf("get referral code: %w", err)
	}
	if code != nil && *code != "" {
		return *code, nil
	}
	newCode, err := generateReferralCode()
	if err != nil {
		return "", fmt.Errorf("generate referral code: %w", err)
	}
	if _, err := r.db.Exec(ctx, `UPDATE members SET referral_code = $1 WHERE id = $2`, newCode, memberID); err != nil {
		return "", fmt.Errorf("set referral code: %w", err)
	}
	return newCode, nil
}

// GetByReferralCodeInEEG resolves a referral code to the referring member, scoped
// to the given EEG. An unknown code (or one from a different EEG) resolves to
// nil, nil — never an error — so callers can silently ignore an invalid ref
// param instead of leaking whether a code exists.
func (r *MemberRepository) GetByReferralCodeInEEG(ctx context.Context, eegID uuid.UUID, code string) (*domain.Member, error) {
	if code == "" {
		return nil, nil
	}
	q := `SELECT ` + memberCols + ` FROM members WHERE eeg_id = $1 AND referral_code = $2`
	var m domain.Member
	if err := scanMember(r.db.QueryRow(ctx, q, eegID, code), &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &m, nil
}

// GetMeterPoints returns all meter points for a member, ordered by direction and ID.
func (r *MemberRepository) GetMeterPoints(ctx context.Context, memberID uuid.UUID) ([]MeterPointInfo, error) {
	q := `SELECT zaehlpunkt, energierichtung FROM meter_points WHERE member_id = $1 ORDER BY energierichtung, zaehlpunkt`
	rows, err := r.db.Query(ctx, q, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []MeterPointInfo
	for rows.Next() {
		var mp MeterPointInfo
		if err := rows.Scan(&mp.Zaehlpunkt, &mp.Richtung); err != nil {
			return nil, err
		}
		result = append(result, mp)
	}
	return result, rows.Err()
}
