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
	initiated_at, deadline_at, completed_at, error_msg, error_notification_sent_at,
	response_codes, meter_owner_name, portal_approval_url, customer_notified_at, created_at`

func scanEDAProcess(row interface{ Scan(...any) error }, p *domain.EDAProcess) error {
	return row.Scan(
		&p.ID, &p.EegID, &p.MeterPointID, &p.ProcessType, &p.Status,
		&p.ConversationID, &p.Zaehlpunkt, &p.ValidFrom, &p.ParticipationFactor, &p.ShareType,
		&p.ECDisModel, &p.DateTo, &p.EnergyDirection, &p.ECShare,
		&p.InitiatedAt, &p.DeadlineAt, &p.CompletedAt, &p.ErrorMsg, &p.ErrorNotificationSentAt,
		&p.ResponseCodes, &p.MeterOwnerName, &p.PortalApprovalURL, &p.CustomerNotifiedAt, &p.CreatedAt,
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
		ep.initiated_at, ep.deadline_at, ep.completed_at, ep.error_msg, ep.error_notification_sent_at,
		ep.response_codes, ep.meter_owner_name, ep.portal_approval_url, ep.customer_notified_at, ep.created_at,
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
			&p.InitiatedAt, &p.DeadlineAt, &p.CompletedAt, &p.ErrorMsg, &p.ErrorNotificationSentAt,
			&p.ResponseCodes, &p.MeterOwnerName, &p.PortalApprovalURL, &p.CustomerNotifiedAt, &p.CreatedAt,
			&p.MemberName,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		ps = append(ps, p)
	}
	return ps, rows.Err()
}

// ListByZaehlpunkt returns EC_REQ_ONL and CM_REV_SP processes for a Zählpunkt within
// an EEG, oldest first — the individual Anmeldung/Abmeldung attempts (sent, confirmed,
// rejected, error) that make up the Zählpunkt's registration history.
func (r *EDAProcessRepository) ListByZaehlpunkt(ctx context.Context, eegID uuid.UUID, zaehlpunkt string) ([]domain.EDAProcess, error) {
	q := `SELECT ` + edaProcessCols + `
	      FROM eda_processes
	      WHERE eeg_id = $1 AND zaehlpunkt = $2 AND process_type IN ('EC_REQ_ONL', 'CM_REV_SP')
	      ORDER BY initiated_at ASC`
	rows, err := r.db.Query(ctx, q, eegID, zaehlpunkt)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var ps []domain.EDAProcess
	for rows.Next() {
		var p domain.EDAProcess
		if err := scanEDAProcess(rows, &p); err != nil {
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

// FindSentReqPTByZaehlpunkt returns the most recent CR_REQ_PT process in "sent"
// status for a given Zählpunkt within one EEG. Used to auto-complete the process
// when a DATEN_CRMSG (ConsumptionRecord) arrives. The eeg_id scope matters for
// Mehrfachteilnahme: the same Zählpunkt can have open data requests in two EEGs,
// and an unscoped match could complete the wrong one.
func (r *EDAProcessRepository) FindSentReqPTByZaehlpunkt(ctx context.Context, eegID uuid.UUID, zaehlpunkt string) (*domain.EDAProcess, error) {
	q := `SELECT ` + edaProcessCols + `
	      FROM eda_processes
	      WHERE eeg_id = $1
	        AND zaehlpunkt = $2
	        AND process_type = 'CR_REQ_PT'
	        AND status = 'sent'
	      ORDER BY initiated_at DESC
	      LIMIT 1`
	var p domain.EDAProcess
	if err := scanEDAProcess(r.db.QueryRow(ctx, q, eegID, zaehlpunkt), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ListZaehlpunkteWithInFlightCRReqPT returns the set of Zählpunkte in this EEG
// that currently have a CR_REQ_PT process in 'pending' or 'sent' status. Used
// by the "Fehlende Daten nachfordern" preview to flag Zählpunkte that already
// have a request in flight (which the Netzbetreiber may simply never have
// answered — the preview still allows re-requesting them, this is only a
// display hint). A single EEG-wide batch query, unlike the per-ZP
// FindSentReqPTByZaehlpunkt used by the worker for message matching.
func (r *EDAProcessRepository) ListZaehlpunkteWithInFlightCRReqPT(ctx context.Context, eegID uuid.UUID) (map[string]bool, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT zaehlpunkt FROM eda_processes
		WHERE eeg_id = $1 AND process_type = 'CR_REQ_PT' AND status IN ('pending', 'sent')
	`, eegID)
	if err != nil {
		return nil, fmt.Errorf("list in-flight CR_REQ_PT zaehlpunkte: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var zp string
		if err := rows.Scan(&zp); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out[zp] = true
	}
	return out, rows.Err()
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

// UpdateResponseInfo stores response codes, meter owner name and portal URL received
// from an inbound CMNotification (ANTWORT_ECON or ZUSTIMMUNG_ECON).
func (r *EDAProcessRepository) UpdateResponseInfo(ctx context.Context, id uuid.UUID, responseCodes []string, meterOwnerName, portalApprovalURL string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE eda_processes
		 SET response_codes = $1, meter_owner_name = $2, portal_approval_url = $3, updated_at = now()
		 WHERE id = $4`,
		responseCodes, meterOwnerName, portalApprovalURL, id,
	)
	return err
}

// SetCustomerNotified marks customer_notified_at = now() for a process.
// Used to ensure the §16e SmartMeter info email is sent at most once.
func (r *EDAProcessRepository) SetCustomerNotified(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE eda_processes SET customer_notified_at = now() WHERE id = $1`,
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

// HasActiveAnmeldung returns true if there is already an EC_REQ_ONL process for
// this Zählpunkt that is pending, sent, first_confirmed, or confirmed — i.e. not
// yet known to have failed. Used to block a second Anmeldung while one is still
// in flight or already succeeded, since the Netzbetreiber rejects duplicates and
// deleting/recreating the meter point in between would otherwise be the only way
// a user could "retry", which orphans the original process's eventual confirmation.
func (r *EDAProcessRepository) HasActiveAnmeldung(ctx context.Context, eegID uuid.UUID, zaehlpunkt string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM eda_processes
		WHERE eeg_id = $1
		  AND zaehlpunkt = $2
		  AND process_type = 'EC_REQ_ONL'
		  AND status NOT IN ('rejected', 'error')
	`, eegID, zaehlpunkt).Scan(&count)
	return count > 0, err
}

// HasActiveProcess returns true if there is an in-flight (pending/sent/
// first_confirmed) EDA process still attached to this meter point. Used to
// block meter-point deletion while a process is awaiting Netzbetreiber
// confirmation: deleting the meter point sets eda_processes.meter_point_id to
// NULL (ON DELETE SET NULL), and the confirmation handler only applies its
// effects (registriert_seit, consent_id, member activation) when meter_point_id
// is set — an in-flight process orphaned this way would confirm successfully on
// the wire but silently vanish from the app.
func (r *EDAProcessRepository) HasActiveProcess(ctx context.Context, meterPointID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM eda_processes
		WHERE meter_point_id = $1
		  AND status IN ('pending', 'sent', 'first_confirmed')
	`, meterPointID).Scan(&count)
	return count > 0, err
}

// SetMeterPointID re-links an eda_processes row to a meter point. Used to
// recover a process that was orphaned (meter_point_id set to NULL by
// ON DELETE SET NULL) while still in flight, once its confirmation arrives —
// see worker.go's ABSCHLUSS_ECON handling.
func (r *EDAProcessRepository) SetMeterPointID(ctx context.Context, id uuid.UUID, meterPointID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE eda_processes SET meter_point_id = $2, updated_at = now() WHERE id = $1`,
		id, meterPointID,
	)
	return err
}
