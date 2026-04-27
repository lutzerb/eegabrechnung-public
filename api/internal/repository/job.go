package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type JobRepository struct {
	db *pgxpool.Pool
}

func NewJobRepository(db *pgxpool.Pool) *JobRepository {
	return &JobRepository{db: db}
}

// edaJobPayload mirrors the JSON structure the EDA worker reads from jobs.payload.
type edaJobPayload struct {
	Process        string    `json:"process"`
	From           string    `json:"from"`
	To             string    `json:"to"`
	GemeinschaftID string    `json:"gemeinschaft_id"`
	ConversationID string    `json:"conversation_id"`
	XMLPayload     string    `json:"xml_payload"`
	EDAProcessID   uuid.UUID `json:"eda_process_id"`
	EegID          uuid.UUID `json:"eeg_id"`
}

// EnqueueEDA inserts an outbound EDA job into the jobs table.
func (r *JobRepository) EnqueueEDA(
	ctx context.Context,
	process, from, to, gemeinschaftID, conversationID, xmlPayload string,
	edaProcessID uuid.UUID,
	eegID uuid.UUID,
) error {
	payload := edaJobPayload{
		Process:        process,
		From:           from,
		To:             to,
		GemeinschaftID: gemeinschaftID,
		ConversationID: conversationID,
		XMLPayload:     xmlPayload,
		EDAProcessID:   edaProcessID,
		EegID:          eegID,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal job payload: %w", err)
	}
	_, err = r.db.Exec(ctx,
		`INSERT INTO jobs (type, payload) VALUES ('eda.'||$1, $2)`,
		process, raw,
	)
	return err
}
