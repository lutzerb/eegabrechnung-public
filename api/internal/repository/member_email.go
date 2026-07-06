package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type MemberEmailRepository struct {
	db *pgxpool.Pool
}

func NewMemberEmailRepository(db *pgxpool.Pool) *MemberEmailRepository {
	return &MemberEmailRepository{db: db}
}

func (r *MemberEmailRepository) Create(ctx context.Context, c *domain.MemberEmailCampaign) error {
	attJSON, _ := json.Marshal(c.Attachments)
	return r.db.QueryRow(ctx, `
		INSERT INTO member_email_campaigns (eeg_id, subject, html_body, recipient_count, attachments_json)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		c.EegID, c.Subject, c.HtmlBody, c.RecipientCount, attJSON,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *MemberEmailRepository) UpdateRecipientCount(ctx context.Context, id uuid.UUID, count int) error {
	_, err := r.db.Exec(ctx, `UPDATE member_email_campaigns SET recipient_count=$1 WHERE id=$2`, count, id)
	return err
}

func (r *MemberEmailRepository) List(ctx context.Context, eegID uuid.UUID) ([]domain.MemberEmailCampaign, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, eeg_id, subject, html_body, recipient_count, attachments_json, created_at
		FROM member_email_campaigns
		WHERE eeg_id = $1
		ORDER BY created_at DESC`, eegID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MemberEmailCampaign
	for rows.Next() {
		var c domain.MemberEmailCampaign
		var attJSON []byte
		if err := rows.Scan(&c.ID, &c.EegID, &c.Subject, &c.HtmlBody, &c.RecipientCount, &attJSON, &c.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(attJSON, &c.Attachments) //nolint:errcheck
		if c.Attachments == nil {
			c.Attachments = []domain.CampaignAttachment{}
		}
		out = append(out, c)
	}
	if out == nil {
		out = []domain.MemberEmailCampaign{}
	}
	return out, rows.Err()
}

func (r *MemberEmailRepository) GetByID(ctx context.Context, id, eegID uuid.UUID) (*domain.MemberEmailCampaign, error) {
	var c domain.MemberEmailCampaign
	var attJSON []byte
	err := r.db.QueryRow(ctx, `
		SELECT id, eeg_id, subject, html_body, recipient_count, attachments_json, created_at
		FROM member_email_campaigns
		WHERE id = $1 AND eeg_id = $2`, id, eegID,
	).Scan(&c.ID, &c.EegID, &c.Subject, &c.HtmlBody, &c.RecipientCount, &attJSON, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(attJSON, &c.Attachments) //nolint:errcheck
	if c.Attachments == nil {
		c.Attachments = []domain.CampaignAttachment{}
	}
	return &c, nil
}
