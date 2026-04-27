package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
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

const memberCols = `id, eeg_id, mitglieds_nr, name1, name2, email, iban, strasse, plz, ort,
	business_role, uid_nummer, use_vat, vat_pct, status, beitritt_datum, austritt_datum,
	sepa_mandate_signed_at, sepa_mandate_signed_ip, sepa_mandate_text, created_at`

func scanMember(row interface{ Scan(...any) error }, m *domain.Member) error {
	var mandateIP *string
	var mandateText *string
	err := row.Scan(
		&m.ID, &m.EegID, &m.MitgliedsNr, &m.Name1, &m.Name2, &m.Email, &m.IBAN, &m.Strasse, &m.Plz, &m.Ort,
		&m.BusinessRole, &m.UidNummer, &m.UseVat, &m.VatPct, &m.Status, &m.BeitrittsDatum, &m.AustrittsDatum,
		&m.SepaMandateSignedAt, &mandateIP, &mandateText, &m.CreatedAt,
	)
	if mandateIP != nil {
		m.SepaMandateSignedIP = *mandateIP
	}
	if mandateText != nil {
		m.SepaMandateText = *mandateText
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
	q := `INSERT INTO members (eeg_id, mitglieds_nr, name1, name2, email, iban, strasse, plz, ort, business_role, uid_nummer, use_vat, vat_pct, status, beitritt_datum, austritt_datum)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	      RETURNING id, created_at`
	return r.db.QueryRow(ctx, q,
		m.EegID, m.MitgliedsNr, m.Name1, m.Name2, m.Email, m.IBAN, m.Strasse, m.Plz, m.Ort,
		m.BusinessRole, m.UidNummer, m.UseVat, m.VatPct, m.Status,
		m.BeitrittsDatum, m.AustrittsDatum,
	).Scan(&m.ID, &m.CreatedAt)
}

func (r *MemberRepository) Update(ctx context.Context, m *domain.Member) error {
	if m.Status == "" {
		m.Status = "ACTIVE"
	}
	q := `UPDATE members SET
	        name1=$1, name2=$2, email=$3, iban=$4, strasse=$5, plz=$6, ort=$7,
	        business_role=$8, mitglieds_nr=$9, uid_nummer=$10, use_vat=$11, vat_pct=$12, status=$13,
	        beitritt_datum=$14, austritt_datum=$15
	      WHERE id=$16`
	_, err := r.db.Exec(ctx, q,
		m.Name1, m.Name2, m.Email, m.IBAN, m.Strasse, m.Plz, m.Ort,
		m.BusinessRole, m.MitgliedsNr, m.UidNummer, m.UseVat, m.VatPct, m.Status,
		m.BeitrittsDatum, m.AustrittsDatum, m.ID,
	)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	return nil
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

// MeterPointInfo holds the meter point ID and direction for PDF/billing use.
type MeterPointInfo struct {
	Zaehlpunkt string
	Richtung   string
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
