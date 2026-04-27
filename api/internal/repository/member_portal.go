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
	MemberID uuid.UUID
	EegID    uuid.UUID
	EegName  string
	Name1    string
	Name2    string
	Email    string
	IsDemo   bool
}

// FindMembersByEmail finds all active members with the given email across all EEGs.
func (r *MemberPortalRepository) FindMembersByEmail(ctx context.Context, email string) ([]PortalMemberChoice, error) {
	q := `SELECT m.id, m.eeg_id, e.name, m.name1, m.name2, m.email, e.is_demo
	      FROM members m
	      JOIN eegs e ON e.id = m.eeg_id
	      WHERE LOWER(m.email) = LOWER($1) AND m.status != 'INACTIVE'
	      ORDER BY e.name`
	rows, err := r.db.Query(ctx, q, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PortalMemberChoice
	for rows.Next() {
		var c PortalMemberChoice
		if err := rows.Scan(&c.MemberID, &c.EegID, &c.EegName, &c.Name1, &c.Name2, &c.Email, &c.IsDemo); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// FindMemberByEmailAndEEG finds a specific active member by email within a given EEG.
func (r *MemberPortalRepository) FindMemberByEmailAndEEG(ctx context.Context, email string, eegID uuid.UUID) (*domain.Member, error) {
	q := `SELECT ` + memberCols + ` FROM members WHERE LOWER(email) = LOWER($1) AND eeg_id = $2 AND status != 'INACTIVE' LIMIT 1`
	var m domain.Member
	if err := scanMember(r.db.QueryRow(ctx, q, email, eegID), &m); err != nil {
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
