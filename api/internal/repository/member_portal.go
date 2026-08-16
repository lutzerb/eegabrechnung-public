package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type MemberPortalRepository struct {
	db *pgxpool.Pool
}

func NewMemberPortalRepository(db *pgxpool.Pool) *MemberPortalRepository {
	return &MemberPortalRepository{db: db}
}

func generatePortalToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// PortalMemberChoice is a lightweight struct for the EEG-selection step.
type PortalMemberChoice struct {
	MemberID      uuid.UUID
	EegID         uuid.UUID
	EegName       string
	Name1         string
	Name2         string
	Email         string
	IsDemo        bool
	PortalBaseURL string
}

// FindMembersByEmail finds all active members with the given email, scoped to the
// given organization — the organization is resolved from the requesting domain
// (see OrganizationRepository.GetOrgIDByHost) and must never be omitted here, or
// members of unrelated organizations sharing the same email become discoverable
// across tenant boundaries.
func (r *MemberPortalRepository) FindMembersByEmail(ctx context.Context, email string, organizationID uuid.UUID) ([]PortalMemberChoice, error) {
	q := `SELECT m.id, m.eeg_id, COALESCE(NULLIF(e.display_name, ''), e.name), m.name1, m.name2, m.email, e.is_demo, o.portal_base_url
	      FROM members m
	      JOIN eegs e ON e.id = m.eeg_id
	      JOIN organizations o ON o.id = e.organization_id
	      WHERE LOWER(m.email) = LOWER($1) AND m.status != 'INACTIVE' AND e.organization_id = $2
	      ORDER BY e.name`
	rows, err := r.db.Query(ctx, q, email, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PortalMemberChoice
	for rows.Next() {
		var c PortalMemberChoice
		if err := rows.Scan(&c.MemberID, &c.EegID, &c.EegName, &c.Name1, &c.Name2, &c.Email, &c.IsDemo, &c.PortalBaseURL); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// FindMemberByEmailAndEEG finds a specific active member by email within a given EEG,
// scoped to the given organization (defense-in-depth alongside the explicit eegID —
// see FindMembersByEmail for why organization scoping matters here).
func (r *MemberPortalRepository) FindMemberByEmailAndEEG(ctx context.Context, email string, eegID, organizationID uuid.UUID) (*domain.Member, error) {
	q := `SELECT ` + memberCols + ` FROM members
	      WHERE LOWER(email) = LOWER($1) AND eeg_id = $2 AND status != 'INACTIVE'
	      AND EXISTS (SELECT 1 FROM eegs WHERE eegs.id = members.eeg_id AND eegs.organization_id = $3)
	      LIMIT 1`
	var m domain.Member
	if err := scanMember(r.db.QueryRow(ctx, q, email, eegID, organizationID), &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// CreateLinkSession creates a new one-time link token for a member (expires in 30 minutes).
func (r *MemberPortalRepository) CreateLinkSession(ctx context.Context, memberID, eegID uuid.UUID) (string, error) {
	token, err := generatePortalToken()
	if err != nil {
		return "", err
	}
	expires := time.Now().Add(30 * time.Minute)
	_, err = r.db.Exec(ctx, `
		INSERT INTO member_portal_sessions (member_id, eeg_id, link_token, link_expires_at)
		VALUES ($1, $2, $3, $4)
	`, memberID, eegID, token, expires)
	return token, err
}

// ExchangeLinkToken validates a one-time link token, marks it used, creates a session token, and returns the session token + member/eeg IDs.
func (r *MemberPortalRepository) ExchangeLinkToken(ctx context.Context, linkToken string) (sessionToken string, memberID, eegID uuid.UUID, err error) {
	// Find unused, unexpired link token
	var id uuid.UUID
	err = r.db.QueryRow(ctx, `
		SELECT id, member_id, eeg_id FROM member_portal_sessions
		WHERE link_token = $1
		  AND link_used_at IS NULL
		  AND link_expires_at > NOW()
	`, linkToken).Scan(&id, &memberID, &eegID)
	if err != nil {
		return "", uuid.Nil, uuid.Nil, err
	}

	// Generate session token
	sessionToken, err = generatePortalToken()
	if err != nil {
		return "", uuid.Nil, uuid.Nil, err
	}
	sessionExpires := time.Now().Add(24 * time.Hour)

	// Mark used + set session token
	_, err = r.db.Exec(ctx, `
		UPDATE member_portal_sessions
		SET link_used_at = NOW(), session_token = $1, session_expires_at = $2
		WHERE id = $3
	`, sessionToken, sessionExpires, id)
	if err != nil {
		return "", uuid.Nil, uuid.Nil, err
	}
	return sessionToken, memberID, eegID, nil
}

// FindBySessionToken validates a session token and returns member + eeg IDs.
func (r *MemberPortalRepository) FindBySessionToken(ctx context.Context, sessionToken string) (memberID, eegID uuid.UUID, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT member_id, eeg_id FROM member_portal_sessions
		WHERE session_token = $1
		  AND session_expires_at > NOW()
	`, sessionToken).Scan(&memberID, &eegID)
	return
}

// CreateSessionForMember creates a 24h portal session directly for a member/eeg pair,
// without a magic-link step (used by password login). member_portal_sessions.link_token
// is NOT NULL UNIQUE, so a throwaway already-used/expired placeholder is stored alongside it.
func (r *MemberPortalRepository) CreateSessionForMember(ctx context.Context, memberID, eegID uuid.UUID) (string, error) {
	placeholderToken, err := generatePortalToken()
	if err != nil {
		return "", err
	}
	sessionToken, err := generatePortalToken()
	if err != nil {
		return "", err
	}
	sessionExpires := time.Now().Add(24 * time.Hour)
	_, err = r.db.Exec(ctx, `
		INSERT INTO member_portal_sessions
			(member_id, eeg_id, link_token, link_used_at, link_expires_at, session_token, session_expires_at)
		VALUES ($1, $2, $3, NOW(), NOW(), $4, $5)
	`, memberID, eegID, placeholderToken, sessionToken, sessionExpires)
	if err != nil {
		return "", err
	}
	return sessionToken, nil
}

// SetPassword upserts the bcrypt hash for a member portal login, keyed by email
// (not member_id) — a password applies to every EEG membership sharing that email.
func (r *MemberPortalRepository) SetPassword(ctx context.Context, email, hash string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO member_portal_credentials (email, password_hash)
		VALUES (LOWER($1), $2)
		ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash, updated_at = NOW()
	`, email, hash)
	return err
}

// HasPassword reports whether a portal password has been set for the given email.
func (r *MemberPortalRepository) HasPassword(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM member_portal_credentials WHERE email = LOWER($1))
	`, email).Scan(&exists)
	return exists, err
}

// FindPasswordHash returns the stored bcrypt hash for an email, or an empty string if none is set.
func (r *MemberPortalRepository) FindPasswordHash(ctx context.Context, email string) (string, error) {
	var hash string
	err := r.db.QueryRow(ctx, `
		SELECT password_hash FROM member_portal_credentials WHERE email = LOWER($1)
	`, email).Scan(&hash)
	if err != nil {
		return "", err
	}
	return hash, nil
}

// EmailChangeVerification is a pending or completed self-service email change request.
type EmailChangeVerification struct {
	ID         uuid.UUID
	MemberID   uuid.UUID
	EegID      uuid.UUID
	NewEmail   string
	ExpiresAt  time.Time
	VerifiedAt *time.Time
}

// CreateEmailChangeVerification removes any pending (unverified) email change request for
// the member and creates a new one with a fresh token, valid for 30 minutes.
func (r *MemberPortalRepository) CreateEmailChangeVerification(ctx context.Context, memberID, eegID uuid.UUID, newEmail string) (string, error) {
	token, err := generatePortalToken()
	if err != nil {
		return "", err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM member_email_change_verifications WHERE member_id = $1 AND verified_at IS NULL`,
		memberID,
	); err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(30 * time.Minute)
	if _, err := tx.Exec(ctx, `
		INSERT INTO member_email_change_verifications (member_id, eeg_id, new_email, token, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, memberID, eegID, newEmail, token, expiresAt); err != nil {
		return "", err
	}

	return token, tx.Commit(ctx)
}

// FindEmailChangeVerificationByToken looks up a verification by token. Not scoped by EEG
// since confirmation may happen from a different device/session than the request.
func (r *MemberPortalRepository) FindEmailChangeVerificationByToken(ctx context.Context, token string) (*EmailChangeVerification, error) {
	var v EmailChangeVerification
	err := r.db.QueryRow(ctx, `
		SELECT id, member_id, eeg_id, new_email, expires_at, verified_at
		FROM member_email_change_verifications WHERE token = $1
	`, token).Scan(&v.ID, &v.MemberID, &v.EegID, &v.NewEmail, &v.ExpiresAt, &v.VerifiedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ConfirmEmailChange applies the verified new email to the member and marks the
// verification as confirmed, in one transaction. Returns the member's email as it was
// before the update (for the old-address security notice).
func (r *MemberPortalRepository) ConfirmEmailChange(ctx context.Context, v *EmailChangeVerification) (oldEmail string, err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	if err := tx.QueryRow(ctx, `SELECT email FROM members WHERE id = $1 FOR UPDATE`, v.MemberID).Scan(&oldEmail); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE members SET email = $1 WHERE id = $2`, v.NewEmail, v.MemberID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE member_email_change_verifications SET verified_at = NOW() WHERE id = $1`, v.ID); err != nil {
		return "", err
	}

	return oldEmail, tx.Commit(ctx)
}
