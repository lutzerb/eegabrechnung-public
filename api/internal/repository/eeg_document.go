package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type EEGDocumentRepository struct {
	db *pgxpool.Pool
}

func NewEEGDocumentRepository(db *pgxpool.Pool) *EEGDocumentRepository {
	return &EEGDocumentRepository{db: db}
}

func (r *EEGDocumentRepository) Create(ctx context.Context, d *domain.EEGDocument) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO eeg_documents (eeg_id, title, description, filename, file_path, mime_type, file_size_bytes, sort_order, show_in_onboarding)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at`,
		d.EegID, d.Title, d.Description, d.Filename, d.FilePath, d.MimeType, d.FileSizeBytes, d.SortOrder, d.ShowInOnboarding,
	).Scan(&d.ID, &d.CreatedAt)
}

func (r *EEGDocumentRepository) List(ctx context.Context, eegID uuid.UUID) ([]domain.EEGDocument, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, eeg_id, title, description, filename, file_path, mime_type, file_size_bytes, sort_order, show_in_onboarding, created_at
		FROM eeg_documents WHERE eeg_id = $1 ORDER BY sort_order, created_at`, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var out []domain.EEGDocument
	for rows.Next() {
		var d domain.EEGDocument
		if err := rows.Scan(&d.ID, &d.EegID, &d.Title, &d.Description, &d.Filename, &d.FilePath, &d.MimeType, &d.FileSizeBytes, &d.SortOrder, &d.ShowInOnboarding, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if out == nil {
		out = []domain.EEGDocument{}
	}
	return out, rows.Err()
}

// ListForOnboarding returns only documents marked show_in_onboarding = true.
func (r *EEGDocumentRepository) ListForOnboarding(ctx context.Context, eegID uuid.UUID) ([]domain.EEGDocument, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, eeg_id, title, description, filename, file_path, mime_type, file_size_bytes, sort_order, show_in_onboarding, created_at
		FROM eeg_documents WHERE eeg_id = $1 AND show_in_onboarding = true ORDER BY sort_order, created_at`, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var out []domain.EEGDocument
	for rows.Next() {
		var d domain.EEGDocument
		if err := rows.Scan(&d.ID, &d.EegID, &d.Title, &d.Description, &d.Filename, &d.FilePath, &d.MimeType, &d.FileSizeBytes, &d.SortOrder, &d.ShowInOnboarding, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if out == nil {
		out = []domain.EEGDocument{}
	}
	return out, rows.Err()
}

func (r *EEGDocumentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.EEGDocument, error) {
	var d domain.EEGDocument
	err := r.db.QueryRow(ctx, `
		SELECT id, eeg_id, title, description, filename, file_path, mime_type, file_size_bytes, sort_order, show_in_onboarding, created_at
		FROM eeg_documents WHERE id = $1`, id,
	).Scan(&d.ID, &d.EegID, &d.Title, &d.Description, &d.Filename, &d.FilePath, &d.MimeType, &d.FileSizeBytes, &d.SortOrder, &d.ShowInOnboarding, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *EEGDocumentRepository) Delete(ctx context.Context, id, eegID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM eeg_documents WHERE id=$1 AND eeg_id=$2`, id, eegID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func (r *EEGDocumentRepository) UpdateSortOrder(ctx context.Context, id uuid.UUID, sortOrder int) error {
	_, err := r.db.Exec(ctx, `UPDATE eeg_documents SET sort_order=$1 WHERE id=$2`, sortOrder, id)
	return err
}

// SetShowInOnboarding toggles whether a document appears on the public onboarding page.
func (r *EEGDocumentRepository) SetShowInOnboarding(ctx context.Context, id uuid.UUID, show bool) error {
	tag, err := r.db.Exec(ctx, `UPDATE eeg_documents SET show_in_onboarding=$1 WHERE id=$2`, show, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}
