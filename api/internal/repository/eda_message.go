package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type EDAMessageRepository struct {
	db *pgxpool.Pool
}

func NewEDAMessageRepository(db *pgxpool.Pool) *EDAMessageRepository {
	return &EDAMessageRepository{db: db}
}

// MarkProcessed sets processed_at = now() and status = 'processed' for the given message.
func (r *EDAMessageRepository) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE eda_messages SET processed_at = now(), status = 'processed' WHERE id = $1`,
		id,
	)
	return err
}

// MarkError sets status = 'error' and error_msg for the given message.
func (r *EDAMessageRepository) MarkError(ctx context.Context, id uuid.UUID, errorMsg string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE eda_messages SET status = 'error', error_msg = $1 WHERE id = $2`,
		errorMsg, id,
	)
	return err
}

// GetIDBySubjectOutbound returns the DB id of an outbound message matching the given subject.
func (r *EDAMessageRepository) GetIDBySubjectOutbound(ctx context.Context, subject string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT id FROM eda_messages WHERE subject = $1 AND direction = 'outbound' ORDER BY created_at DESC LIMIT 1`,
		subject,
	).Scan(&id)
	return id, err
}

// UpdateEegID sets eeg_id on an existing message (called after the EEG is determined
// from the message content, e.g. via Zählpunkt lookup or ConversationID match).
func (r *EDAMessageRepository) UpdateEegID(ctx context.Context, id, eegID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE eda_messages SET eeg_id = $1 WHERE id = $2`,
		eegID, id,
	)
	return err
}

// UpdateClassification overwrites process and message_type for a stored message
// after the inbound XML has been parsed into a more specific MaKo message code.
func (r *EDAMessageRepository) UpdateClassification(ctx context.Context, id uuid.UUID, code string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE eda_messages SET process = $1, message_type = $1 WHERE id = $2`,
		code, id,
	)
	return err
}

const edaMsgCols = `id, coalesce(message_id,''), direction, coalesce(process,''), message_type, subject, coalesce(body,''), coalesce(from_address,''), coalesce(to_address,''), coalesce(status,''), coalesce(error_msg,''), processed_at, created_at`

func scanEDAMessages(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}) ([]domain.EDAMessage, error) {
	defer rows.Close()
	var msgs []domain.EDAMessage
	for rows.Next() {
		var m domain.EDAMessage
		if err := rows.Scan(
			&m.ID, &m.MessageID, &m.Direction, &m.Process, &m.MessageType, &m.Subject,
			&m.Body, &m.FromAddress, &m.ToAddress,
			&m.Status, &m.ErrorMsg, &m.ProcessedAt, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// ExistsByMessageID returns true if a message with the given message_id is already stored.
// Used to deduplicate DUPL retransmissions.
func (r *EDAMessageRepository) ExistsByMessageID(ctx context.Context, messageID string) (bool, error) {
	if messageID == "" {
		return false, nil
	}
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM eda_messages WHERE message_id = $1)`,
		messageID,
	).Scan(&exists)
	return exists, err
}

// CountByEEG returns the total number of EDA messages for one EEG.
func (r *EDAMessageRepository) CountByEEG(ctx context.Context, eegID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM eda_messages WHERE eeg_id = $1`,
		eegID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return count, nil
}

// ListByEEG returns EDA messages for a specific EEG, newest first.
func (r *EDAMessageRepository) ListByEEG(ctx context.Context, eegID uuid.UUID, limit, offset int) ([]domain.EDAMessage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+edaMsgCols+` FROM eda_messages WHERE eeg_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		eegID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return scanEDAMessages(rows)
}

// GetXMLPayload returns the raw xml_payload for a single message, scoped to an EEG.
func (r *EDAMessageRepository) GetXMLPayload(ctx context.Context, id, eegID uuid.UUID) (string, error) {
	var payload string
	err := r.db.QueryRow(ctx,
		`SELECT xml_payload FROM eda_messages WHERE id = $1 AND eeg_id = $2`,
		id, eegID,
	).Scan(&payload)
	if err != nil {
		return "", fmt.Errorf("get xml_payload: %w", err)
	}
	return payload, nil
}

// List returns EDA messages ordered by created_at DESC (all EEGs, for backward compat).
func (r *EDAMessageRepository) List(ctx context.Context, limit int) ([]domain.EDAMessage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+edaMsgCols+` FROM eda_messages ORDER BY created_at DESC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return scanEDAMessages(rows)
}
