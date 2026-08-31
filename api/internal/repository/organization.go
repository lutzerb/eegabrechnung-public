package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrganizationRepository struct {
	db *pgxpool.Pool
}

func NewOrganizationRepository(db *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// GetOrgIDByHost resolves the organization registered for the given Host header
// (port stripped, case-insensitive) via organization_domains — an org can have
// more than one domain (e.g. a customer domain living in a separate Cloudflare
// account, or a second public domain for the same org). Returns pgx.ErrNoRows if
// no organization is registered for that host — callers must treat that as a hard
// failure, not fall back to searching across all organizations.
func (r *OrganizationRepository) GetOrgIDByHost(ctx context.Context, host string) (uuid.UUID, error) {
	host = strings.ToLower(strings.SplitN(strings.TrimSpace(host), ":", 2)[0])
	if host == "" {
		return uuid.Nil, pgx.ErrNoRows
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT organization_id FROM organization_domains WHERE domain = $1`, host,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
