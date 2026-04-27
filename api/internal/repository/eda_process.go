package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type EDAProcessRepository struct {
	db *pgxpool.Pool
}

func NewEDAProcessRepository(db *pgxpool.Pool) *EDAProcessRepository {
	return &EDAProcessRepository{db: db}
}

const edaProcessCols = `id, eeg_id, meter_point_id, process_type, status,
	conversation_id, zaehlpunkt, valid_from, participation_factor, share_type,
	ec_dis_model, date_to, energy_direction, ec_share,
	initiated_at, deadline_at, completed_at, error_msg, error_notification_sent_at, created_at`

func scanEDAProcess(row interface{ Scan(...any) error }, p *domain.EDAProcess) error {
	return row.Scan(
		&p.ID, &p.EegID, &p.MeterPointID, &p.ProcessType, &p.Status,
		&p.ConversationID, &p.Zaehlpunkt, &p.ValidFrom, &p.ParticipationFactor, &p.ShareType,
		&p.ECDisModel, &p.DateTo, &p.EnergyDirection, &p.ECShare,
		&p.InitiatedAt, &p.DeadlineAt, &p.CompletedAt, &p.ErrorMsg, &p.ErrorNotificationSentAt, &p.CreatedAt,
	)
}

// Create inserts a new EDA process record and returns the generated ID.
func (r *EDAProcessRepository) Create(ctx context.Context, p *domain.EDAProcess) error {
	q := `INSERT INTO eda_processes
	        (eeg_id, meter_point_id, process_type, status, conversation_id,
	         zaehlpunkt, valid_from, participation_factor, share_type,
	         ec_dis_model, date_to, energy_direction, ec_share,
	         initiated_at, deadline_at)
	      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
	      RETURNING id, created_at`
	return r.db.QueryRow(ctx, q,
		p.EegID, p.MeterPointID, p.ProcessType, p.Status, p.ConversationID,
		p.Zaehlpunkt, p.ValidFrom, p.ParticipationFactor, p.ShareType,
		p.ECDisModel, p.DateTo, p.EnergyDirection, p.ECShare,
		p.InitiatedAt, p.DeadlineAt,
	).Scan(&p.ID, &p.CreatedAt)
}

// ListByEEG returns all EDA processes for an EEG, newest first.
// Joins meter_points and members to populate MemberName.
func (r *EDAProcessRepository) ListByEEG(ctx context.Context, eegID uuid.UUID) ([]domain.EDAProcess, error) {
	q := `SELECT ep.id, ep.eeg_id, ep.meter_point_id, ep.process_type, ep.status,
		ep.conversation_id, ep.zaehlpunkt, ep.valid_from, ep.participation_factor, ep.share_type,
		ep.ec_dis_model, ep.date_to, ep.energy_direction, ep.ec_share,
		ep.initiated_at, ep.deadline_at, ep.completed_at, ep.error_msg, ep.error_notification_sent_at, ep.created_at,
		TRIM(COALESCE(m.name1, '') || CASE WHEN COALESCE(m.name2, '') <> '' THEN ' ' || m.name2 ELSE '' END) AS member_name
	      FROM eda_processes ep
	      LEFT JOIN meter_points mp ON ep.meter_point_id = mp.id
	      LEFT JOIN members m ON mp.member_id = m.id
	      WHERE ep.eeg_id = $1
	      ORDER BY ep.created_at DESC`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var ps []domain.EDAProcess
	for rows.Next() {
		var p domain.EDAProcess
		if err := rows.Scan(
			&p.ID, &p.EegID, &p.MeterPointID, &p.ProcessType, &p.Status,
			&p.ConversationID, &p.Zaehlpunkt, &p.ValidFrom, &p.ParticipationFactor, &p.ShareType,
			&p.ECDisModel, &p.DateTo, &p.EnergyDirection, &p.ECShare,
			&p.InitiatedAt, &p.DeadlineAt, &p.CompletedAt, &p.ErrorMsg, &p.ErrorNotificationSentAt, &p.CreatedAt,
			&p.MemberName,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		ps = append(ps, p)
	}
	return ps, rows.Err()
}

// GetByID returns a single EDA process by its primary key.
func (r *EDAProcessRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.EDAProcess, error) {
	q := `SELECT ` + edaProcessCols + `
	      FROM eda_processes WHERE id = $1 LIMIT 1`
	var p domain.EDAProcess
	if err := scanEDAProcess(r.db.QueryRow(ctx, q, id), &p); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &p, nil
}

// GetByConversationID finds a process by its ConversationID for matching
// incoming CPDocument confirmations.
func (r *EDAProcessRepository) GetByConversationID(ctx context.Context, convID string) (*domain.EDAProcess, error) {
	q := `SELECT ` + edaProcessCols + `
	      FROM eda_processes WHERE conversation_id = $1 LIMIT 1`
	var p domain.EDAProcess
	if err := scanEDAProcess(r.db.QueryRow(ctx, q, convID), &p); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &p, nil
}

// UpdateStatus updates the status (and optionally completed_at) for a process.
func (r *EDAProcessRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, completedAt *time.Time, errMsg string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE eda_processes
		 SET status = $1, completed_at = $2, error_msg = $3, updated_at = now()
		 WHERE id = $4`,
		status, completedAt, errMsg, id,
	)
	return err
}

// FindSentReqPTByZaehlpunkt returns the most recent EC_REQ_PT process in "sent"
// status for a given Zählpunkt. Used to auto-complete the process when a
// DATEN_CRMSG (ConsumptionRecord) arrives.
func (r *EDAProcessRepository) FindSentReqPTByZaehlpunkt(ctx context.Context, zaehlpunkt string) (*domain.EDAProcess, error) {
	q := `SELECT ` + edaProcessCols + `
	      FROM eda_processes
	      WHERE zaehlpunkt = $1
	        AND process_type = 'EC_REQ_PT'
	        AND status = 'sent'
	      ORDER BY initiated_at DESC
	      LIMIT 1`
	var p domain.EDAProcess
	if err := scanEDAProcess(r.db.QueryRow(ctx, q, zaehlpunkt), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// SetErrorNotificationSent marks error_notification_sent_at = now() for a process.
// Used by the worker to ensure the error notification email is sent at most once.
func (r *EDAProcessRepository) SetErrorNotificationSent(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE eda_processes SET error_notification_sent_at = now() WHERE id = $1`,
		id,
	)
	return err
}

// HasPendingABM returns true if there is already a pending or sent CM_REV_SP
// process for the given Zählpunkt that has not yet been rejected, errored, or completed.
func (r *EDAProcessRepository) HasPendingABM(ctx context.Context, eegID uuid.UUID, zaehlpunkt string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM eda_processes
		WHERE eeg_id = $1
		  AND zaehlpunkt = $2
		  AND process_type = 'CM_REV_SP'
		  AND status NOT IN ('rejected', 'error', 'completed')
	`, eegID, zaehlpunkt).Scan(&count)
	return count > 0, err
}

// HasPendingFactorChangeToday returns true if there is already a PRTFACT_CHG
// process initiated today for the given Zählpunkt.
func (r *EDAProcessRepository) HasPendingFactorChangeToday(ctx context.Context, eegID uuid.UUID, zaehlpunkt string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM eda_processes
		WHERE eeg_id = $1
		  AND zaehlpunkt = $2
		  AND process_type = 'EC_PRTFACT_CHG'
		  AND initiated_at >= date_trunc('day', now())
		  AND status NOT IN ('rejected', 'error')
	`, eegID, zaehlpunkt).Scan(&count)
	return count > 0, err
}
