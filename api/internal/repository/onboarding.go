package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// OnboardingRepository handles persistence of onboarding_requests.
type OnboardingRepository struct {
	db *pgxpool.Pool
}

// NewOnboardingRepository creates a new OnboardingRepository.
func NewOnboardingRepository(db *pgxpool.Pool) *OnboardingRepository {
	return &OnboardingRepository{db: db}
}

// generateToken creates a 32-byte random hex token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Create inserts a new onboarding request, generating a magic token.
func (r *OnboardingRepository) Create(ctx context.Context, req *domain.OnboardingRequest) error {
	token, err := generateToken()
	if err != nil {
		return err
	}
	req.MagicToken = token
	req.MagicTokenExpiresAt = time.Now().Add(30 * 24 * time.Hour)

	meterPointsJSON, err := json.Marshal(req.MeterPoints)
	if err != nil {
		return fmt.Errorf("marshal meter_points: %w", err)
	}

	q := `INSERT INTO onboarding_requests
	        (eeg_id, status, name1, name2, email, phone, strasse, plz, ort,
	         iban, bic, member_type, business_role, uid_nummer, use_vat,
	         meter_points, beitritts_datum, contract_accepted_at, contract_ip,
	         magic_token, magic_token_expires_at, admin_notes)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	      RETURNING id, created_at, updated_at`

	status := req.Status
	if status == "" {
		status = "pending"
	}
	businessRole := req.BusinessRole
	if businessRole == "" {
		businessRole = "privat"
	}

	return r.db.QueryRow(ctx, q,
		req.EegID, status, req.Name1, req.Name2, req.Email, req.Phone,
		req.Strasse, req.PLZ, req.Ort, req.IBAN, req.BIC, req.MemberType,
		businessRole, req.UidNummer, req.UseVat,
		meterPointsJSON, req.BeitrittsDatum, req.ContractAcceptedAt, req.ContractIP,
		req.MagicToken, req.MagicTokenExpiresAt, req.AdminNotes,
	).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)
}

// GetByToken looks up a request by its magic token.
// Returns nil, nil if no row exists. Returns an error if the token has expired.
func (r *OnboardingRepository) GetByToken(ctx context.Context, token string) (*domain.OnboardingRequest, error) {
	q := `SELECT id, eeg_id, status, name1, name2, email, phone, strasse, plz, ort,
	             iban, bic, member_type, business_role, uid_nummer, use_vat,
	             meter_points, beitritts_datum, contract_accepted_at, contract_ip,
	             magic_token, magic_token_expires_at, admin_notes, converted_member_id,
	             reminder_sent_at, created_at, updated_at
	      FROM onboarding_requests
	      WHERE magic_token = $1`

	req, err := scanOnboarding(r.db.QueryRow(ctx, q, token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan: %w", err)
	}

	if time.Now().After(req.MagicTokenExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	return req, nil
}

// GetByID looks up a request by its UUID.
func (r *OnboardingRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.OnboardingRequest, error) {
	q := `SELECT id, eeg_id, status, name1, name2, email, phone, strasse, plz, ort,
	             iban, bic, member_type, business_role, uid_nummer, use_vat,
	             meter_points, beitritts_datum, contract_accepted_at, contract_ip,
	             magic_token, magic_token_expires_at, admin_notes, converted_member_id,
	             reminder_sent_at, created_at, updated_at
	      FROM onboarding_requests
	      WHERE id = $1`

	req, err := scanOnboarding(r.db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan: %w", err)
	}
	return req, nil
}

// ListByEEG returns all onboarding requests for a given EEG, newest first.
func (r *OnboardingRepository) ListByEEG(ctx context.Context, eegID uuid.UUID) ([]domain.OnboardingRequest, error) {
	q := `SELECT id, eeg_id, status, name1, name2, email, phone, strasse, plz, ort,
	             iban, bic, member_type, business_role, uid_nummer, use_vat,
	             meter_points, beitritts_datum, contract_accepted_at, contract_ip,
	             magic_token, magic_token_expires_at, admin_notes, converted_member_id,
	             reminder_sent_at, created_at, updated_at
	      FROM onboarding_requests
	      WHERE eeg_id = $1
	      ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var result []domain.OnboardingRequest
	for rows.Next() {
		req, err := scanOnboarding(rows)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, *req)
	}
	return result, rows.Err()
}

// FindPendingByEmailAndEEG finds the most recent pending/approved request by email for a given EEG.
func (r *OnboardingRepository) FindPendingByEmailAndEEG(ctx context.Context, email string, eegID uuid.UUID) (*domain.OnboardingRequest, error) {
	q := `SELECT id, eeg_id, status, name1, name2, email, phone, strasse, plz, ort,
	             iban, bic, member_type, business_role, uid_nummer, use_vat,
	             meter_points, beitritts_datum, contract_accepted_at, contract_ip,
	             magic_token, magic_token_expires_at, admin_notes, converted_member_id,
	             reminder_sent_at, created_at, updated_at
	      FROM onboarding_requests
	      WHERE email = $1 AND eeg_id = $2 AND status IN ('pending', 'approved')
	      ORDER BY created_at DESC
	      LIMIT 1`

	req, err := scanOnboarding(r.db.QueryRow(ctx, q, email, eegID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan: %w", err)
	}
	return req, nil
}

// UpdateToken regenerates the magic token and extends the expiry.
func (r *OnboardingRepository) UpdateToken(ctx context.Context, id uuid.UUID) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	expires := time.Now().Add(30 * 24 * time.Hour)
	q := `UPDATE onboarding_requests
	      SET magic_token = $1, magic_token_expires_at = $2, updated_at = now()
	      WHERE id = $3`
	_, err = r.db.Exec(ctx, q, token, expires, id)
	if err != nil {
		return "", fmt.Errorf("update token: %w", err)
	}
	return token, nil
}

// UpdateStatus sets the status and admin_notes, updating updated_at.
func (r *OnboardingRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status, notes string) error {
	q := `UPDATE onboarding_requests
	      SET status = $1, admin_notes = $2, updated_at = now()
	      WHERE id = $3`
	_, err := r.db.Exec(ctx, q, status, notes, id)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// UpdateFields updates all editable data fields of an onboarding request.
// Only allowed for non-converted requests; does not change status.
func (r *OnboardingRepository) UpdateFields(ctx context.Context, req *domain.OnboardingRequest) error {
	meterPointsJSON, err := json.Marshal(req.MeterPoints)
	if err != nil {
		return fmt.Errorf("marshal meter_points: %w", err)
	}
	q := `UPDATE onboarding_requests
	      SET name1 = $1, name2 = $2, email = $3, phone = $4,
	          strasse = $5, plz = $6, ort = $7, iban = $8, bic = $9,
	          member_type = $10, business_role = $11, uid_nummer = $12, use_vat = $13,
	          meter_points = $14, beitritts_datum = $15, admin_notes = $16,
	          updated_at = now()
	      WHERE id = $17`
	_, err = r.db.Exec(ctx, q,
		req.Name1, req.Name2, req.Email, req.Phone,
		req.Strasse, req.PLZ, req.Ort, req.IBAN, req.BIC,
		req.MemberType, req.BusinessRole, req.UidNummer, req.UseVat,
		meterPointsJSON, req.BeitrittsDatum, req.AdminNotes,
		req.ID,
	)
	if err != nil {
		return fmt.Errorf("update fields: %w", err)
	}
	return nil
}

// SetConverted marks a request as converted and links the created member.
// Delete permanently removes an onboarding request.
func (r *OnboardingRepository) Delete(ctx context.Context, id, eegID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM onboarding_requests WHERE id=$1 AND eeg_id=$2`, id, eegID)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func (r *OnboardingRepository) SetConverted(ctx context.Context, id, memberID uuid.UUID) error {
	q := `UPDATE onboarding_requests
	      SET status = 'converted', converted_member_id = $1, updated_at = now()
	      WHERE id = $2`
	_, err := r.db.Exec(ctx, q, memberID, id)
	if err != nil {
		return fmt.Errorf("set converted: %w", err)
	}
	return nil
}

// SetActiveByMeterPoint sets onboarding_request status to 'active' for the request
// whose converted_member_id matches the member_id of the given meter point.
// Only updates if current status is 'eda_sent' or 'converted' (backward compat).
// Returns nil if no matching request exists (non-fatal).
func (r *OnboardingRepository) SetActiveByMeterPoint(ctx context.Context, meterPointID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE onboarding_requests
		SET status = 'active', updated_at = now()
		WHERE converted_member_id = (
			SELECT member_id FROM meter_points WHERE id = $1
		)
		AND status IN ('eda_sent', 'converted')
	`, meterPointID)
	return err
}

// scanOnboarding scans a single row into domain.OnboardingRequest.
func scanOnboarding(row interface {
	Scan(dest ...any) error
}) (*domain.OnboardingRequest, error) {
	var req domain.OnboardingRequest
	var meterPointsRaw []byte

	err := row.Scan(
		&req.ID, &req.EegID, &req.Status,
		&req.Name1, &req.Name2, &req.Email, &req.Phone,
		&req.Strasse, &req.PLZ, &req.Ort,
		&req.IBAN, &req.BIC, &req.MemberType,
		&req.BusinessRole, &req.UidNummer, &req.UseVat,
		&meterPointsRaw,
		&req.BeitrittsDatum,
		&req.ContractAcceptedAt, &req.ContractIP,
		&req.MagicToken, &req.MagicTokenExpiresAt,
		&req.AdminNotes, &req.ConvertedMemberID,
		&req.ReminderSentAt,
		&req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(meterPointsRaw) > 0 {
		if err := json.Unmarshal(meterPointsRaw, &req.MeterPoints); err != nil {
			return nil, fmt.Errorf("unmarshal meter_points: %w", err)
		}
	}
	if req.MeterPoints == nil {
		req.MeterPoints = []domain.OnboardingMeterPoint{}
	}

	return &req, nil
}

// FindNeedingReminder returns eda_sent requests where updated_at is older than
// delay and no reminder has been sent yet. Skips demo EEGs.
func (r *OnboardingRepository) FindNeedingReminder(ctx context.Context, delay time.Duration) ([]domain.OnboardingRequest, error) {
	q := `SELECT o.id, o.eeg_id, o.status, o.name1, o.name2, o.email, o.phone,
	             o.strasse, o.plz, o.ort, o.iban, o.bic, o.member_type,
	             o.business_role, o.uid_nummer, o.use_vat,
	             o.meter_points, o.beitritts_datum, o.contract_accepted_at, o.contract_ip,
	             o.magic_token, o.magic_token_expires_at, o.admin_notes, o.converted_member_id,
	             o.reminder_sent_at, o.created_at, o.updated_at
	      FROM onboarding_requests o
	      JOIN eegs e ON e.id = o.eeg_id
	      WHERE o.status = 'eda_sent'
	        AND o.updated_at < now() - $1::interval
	        AND o.reminder_sent_at IS NULL
	        AND e.is_demo = false`

	rows, err := r.db.Query(ctx, q, fmt.Sprintf("%.0f seconds", delay.Seconds()))
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var result []domain.OnboardingRequest
	for rows.Next() {
		req, err := scanOnboarding(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, *req)
	}
	return result, rows.Err()
}

// SetReminderSent marks reminder_sent_at = now() for the given request.
func (r *OnboardingRepository) SetReminderSent(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE onboarding_requests SET reminder_sent_at = now(), updated_at = now() WHERE id = $1`,
		id,
	)
	return err
}
