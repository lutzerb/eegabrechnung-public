package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// ReferralBonusRepository handles persistence of member_referral_bonuses
// (Migration 114) — flat "Mitglieder werben Mitglieder" credits, manually
// granted by an admin and applied automatically at the referrer's next
// billing run.
type ReferralBonusRepository struct {
	db *pgxpool.Pool
}

func NewReferralBonusRepository(db *pgxpool.Pool) *ReferralBonusRepository {
	return &ReferralBonusRepository{db: db}
}

// EligibleReferral is a successful referral (a converted member with
// referred_by_member_id set) that has not yet had a bonus granted for it.
type EligibleReferral struct {
	ReferrerMemberID uuid.UUID `json:"referrer_member_id"`
	ReferrerName     string    `json:"referrer_name"`
	ReferredMemberID uuid.UUID `json:"referred_member_id"`
	ReferredName     string    `json:"referred_name"`
	JoinedAt         *string   `json:"joined_at,omitempty"`
}

// ListEligible returns members referred by another member within this EEG that
// don't have a member_referral_bonuses row yet — candidates for the admin's
// manual "Prämie gutschreiben" action.
func (r *ReferralBonusRepository) ListEligible(ctx context.Context, eegID uuid.UUID) ([]EligibleReferral, error) {
	q := `SELECT ref.id, COALESCE(NULLIF(TRIM(ref.name1 || ' ' || ref.name2), ''), ref.email),
	             m.id, COALESCE(NULLIF(TRIM(m.name1 || ' ' || m.name2), ''), m.email),
	             TO_CHAR(m.created_at, 'YYYY-MM-DD')
	      FROM members m
	      JOIN members ref ON ref.id = m.referred_by_member_id
	      WHERE m.eeg_id = $1
	        AND NOT EXISTS (SELECT 1 FROM member_referral_bonuses b WHERE b.referred_member_id = m.id)
	      ORDER BY m.created_at DESC`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var out []EligibleReferral
	for rows.Next() {
		var e EligibleReferral
		if err := rows.Scan(&e.ReferrerMemberID, &e.ReferrerName, &e.ReferredMemberID, &e.ReferredName, &e.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// List returns all granted bonuses for an EEG (pending + applied + cancelled),
// newest first, with referrer/referred names joined in for display.
func (r *ReferralBonusRepository) List(ctx context.Context, eegID uuid.UUID) ([]domain.MemberReferralBonus, error) {
	q := `SELECT b.id, b.eeg_id, b.referrer_member_id, b.referred_member_id, b.amount_eur, b.status,
	             b.granted_at, b.granted_by, b.applied_invoice_id, b.applied_at,
	             COALESCE(NULLIF(TRIM(ref.name1 || ' ' || ref.name2), ''), ref.email),
	             COALESCE(NULLIF(TRIM(m.name1 || ' ' || m.name2), ''), m.email)
	      FROM member_referral_bonuses b
	      JOIN members ref ON ref.id = b.referrer_member_id
	      JOIN members m ON m.id = b.referred_member_id
	      WHERE b.eeg_id = $1
	      ORDER BY b.granted_at DESC`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var out []domain.MemberReferralBonus
	for rows.Next() {
		var b domain.MemberReferralBonus
		if err := rows.Scan(&b.ID, &b.EegID, &b.ReferrerMemberID, &b.ReferredMemberID, &b.AmountEur, &b.Status,
			&b.GrantedAt, &b.GrantedBy, &b.AppliedInvoiceID, &b.AppliedAt,
			&b.ReferrerName, &b.ReferredName); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Grant creates a pending bonus for a successful referral. The unique index on
// referred_member_id backstops double-granting (e.g. a double click).
func (r *ReferralBonusRepository) Grant(ctx context.Context, eegID, referrerMemberID, referredMemberID uuid.UUID, amountEur float64, grantedBy uuid.UUID) (*domain.MemberReferralBonus, error) {
	b := &domain.MemberReferralBonus{
		EegID:            eegID,
		ReferrerMemberID: referrerMemberID,
		ReferredMemberID: referredMemberID,
		AmountEur:        amountEur,
		Status:           "pending",
		GrantedBy:        &grantedBy,
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO member_referral_bonuses (eeg_id, referrer_member_id, referred_member_id, amount_eur, status, granted_by)
		VALUES ($1, $2, $3, $4, 'pending', $5)
		RETURNING id, granted_at
	`, eegID, referrerMemberID, referredMemberID, amountEur, grantedBy).Scan(&b.ID, &b.GrantedAt)
	if err != nil {
		return nil, fmt.Errorf("grant: %w", err)
	}
	return b, nil
}

// CancelPending deletes a bonus row, only while it is still pending (not yet
// applied to an invoice) — lets an admin undo a mis-click.
func (r *ReferralBonusRepository) CancelPending(ctx context.Context, eegID, referredMemberID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM member_referral_bonuses WHERE eeg_id = $1 AND referred_member_id = $2 AND status = 'pending'`,
		eegID, referredMemberID,
	)
	if err != nil {
		return fmt.Errorf("cancel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("not found or already applied")
	}
	return nil
}

// SumPendingForMember returns the total unclaimed pending bonus amount for a
// referrer, plus the IDs contributing to it. "Unclaimed" = applied_invoice_id
// IS NULL — once a draft invoice reserves a bonus (see ReserveForInvoice), it's
// no longer offered to a concurrent/later draft, preventing double-application.
func (r *ReferralBonusRepository) SumPendingForMember(ctx context.Context, eegID, memberID uuid.UUID) (float64, []uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, amount_eur FROM member_referral_bonuses
		WHERE eeg_id = $1 AND referrer_member_id = $2 AND status = 'pending' AND applied_invoice_id IS NULL
	`, eegID, memberID)
	if err != nil {
		return 0, nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var total float64
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		var amount float64
		if err := rows.Scan(&id, &amount); err != nil {
			return 0, nil, fmt.Errorf("scan: %w", err)
		}
		total += amount
		ids = append(ids, id)
	}
	return total, ids, rows.Err()
}

// ReserveForInvoice attaches the given pending bonuses to a draft invoice —
// removing them from SumPendingForMember's pool without yet marking them
// "applied" (that only happens at billing-run finalize, see MarkApplied). If
// the draft invoice is later deleted, invoices(id) ON DELETE SET NULL on
// applied_invoice_id automatically frees the bonus back to "pending" +
// unclaimed, ready to be picked up again — no extra cleanup code needed.
func (r *ReferralBonusRepository) ReserveForInvoice(ctx context.Context, bonusIDs []uuid.UUID, invoiceID uuid.UUID) error {
	if len(bonusIDs) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx,
		`UPDATE member_referral_bonuses SET applied_invoice_id = $1 WHERE id = ANY($2)`,
		invoiceID, bonusIDs,
	)
	if err != nil {
		return fmt.Errorf("reserve: %w", err)
	}
	return nil
}

// MarkAppliedForBillingRun flips every bonus reserved against an invoice of the
// given billing run from pending to applied — called when the run is finalized.
func (r *ReferralBonusRepository) MarkAppliedForBillingRun(ctx context.Context, billingRunID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE member_referral_bonuses SET status = 'applied', applied_at = NOW()
		WHERE status = 'pending'
		  AND applied_invoice_id IN (SELECT id FROM invoices WHERE billing_run_id = $1)
	`, billingRunID)
	if err != nil {
		return fmt.Errorf("mark applied: %w", err)
	}
	return nil
}
