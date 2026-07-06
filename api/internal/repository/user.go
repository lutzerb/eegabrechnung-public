package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

const userCols = `id, organization_id, email, password_hash, name, role, created_at`

func scanUser(row interface{ Scan(...any) error }, u *domain.User) error {
	return row.Scan(&u.ID, &u.OrganizationID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	if err := scanUser(r.db.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE email=$1`, email), &u); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User
	if err := scanUser(r.db.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id=$1`, id), &u); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) List(ctx context.Context, orgID uuid.UUID) ([]domain.User, error) {
	rows, err := r.db.Query(ctx, `SELECT `+userCols+` FROM users WHERE organization_id=$1 ORDER BY name`, orgID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	result := []domain.User{}
	for rows.Next() {
		var u domain.User
		if err := scanUser(rows, &u); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

func (r *UserRepository) Create(ctx context.Context, orgID uuid.UUID, email, passwordHash, name, role string) (*domain.User, error) {
	var u domain.User
	q := `INSERT INTO users (organization_id, email, password_hash, name, role)
	      VALUES ($1,$2,$3,$4,$5)
	      RETURNING ` + userCols
	if err := scanUser(r.db.QueryRow(ctx, q, orgID, email, passwordHash, name, role), &u); err != nil {
		return nil, fmt.Errorf("insert: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, name, email, role string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET name=$2, email=$3, role=$4 WHERE id=$1`, id, name, email, role)
	return err
}

func (r *UserRepository) SetPassword(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET password_hash=$2 WHERE id=$1`, id, hash)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	return err
}

func (r *UserRepository) GetEEGAssignments(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `SELECT eeg_id FROM user_eeg_assignments WHERE user_id=$1`, userID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *UserRepository) SetEEGAssignments(ctx context.Context, userID uuid.UUID, eegIDs []uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM user_eeg_assignments WHERE user_id=$1`, userID); err != nil {
		return err
	}
	for _, eegID := range eegIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO user_eeg_assignments (user_id, eeg_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, userID, eegID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
