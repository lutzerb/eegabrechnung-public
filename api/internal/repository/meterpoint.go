package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type MeterPointRepository struct {
	db *pgxpool.Pool
}

type MeterPointSearchResult struct {
	ID              uuid.UUID
	Zaehlpunkt      string
	Energierichtung string
	MemberID        uuid.UUID
	MemberName      string
}

func NewMeterPointRepository(db *pgxpool.Pool) *MeterPointRepository {
	return &MeterPointRepository{db: db}
}

func (r *MeterPointRepository) Upsert(ctx context.Context, mp *domain.MeterPoint) error {
	q := `INSERT INTO meter_points (member_id, eeg_id, zaehlpunkt, energierichtung, verteilungsmodell, zugeteilte_menge_pct, status, registriert_seit)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	      ON CONFLICT (zaehlpunkt) DO UPDATE
	        SET member_id           = EXCLUDED.member_id,
	            eeg_id              = EXCLUDED.eeg_id,
	            energierichtung     = EXCLUDED.energierichtung,
	            verteilungsmodell   = EXCLUDED.verteilungsmodell,
	            zugeteilte_menge_pct = EXCLUDED.zugeteilte_menge_pct,
	            status              = EXCLUDED.status,
	            registriert_seit    = EXCLUDED.registriert_seit
	      RETURNING id, created_at`
	return r.db.QueryRow(ctx, q,
		mp.MemberID, mp.EegID, mp.Zaehlpunkt, mp.Energierichtung,
		mp.Verteilungsmodell, mp.ZugeteilteMenugePct, mp.Status, mp.RegistriertSeit,
	).Scan(&mp.ID, &mp.CreatedAt)
}

const selectMeterPoint = `id, member_id, eeg_id, zaehlpunkt, energierichtung, verteilungsmodell, zugeteilte_menge_pct, status, registriert_seit, abgemeldet_am, generation_type, gap_alert_sent_at, notes, consent_id, created_at`

func scanMeterPoint(row interface{ Scan(...any) error }, mp *domain.MeterPoint) error {
	return row.Scan(
		&mp.ID, &mp.MemberID, &mp.EegID, &mp.Zaehlpunkt,
		&mp.Energierichtung, &mp.Verteilungsmodell, &mp.ZugeteilteMenugePct,
		&mp.Status, &mp.RegistriertSeit, &mp.AbgemeldetAm, &mp.GenerationType, &mp.GapAlertSentAt, &mp.Notes, &mp.ConsentID, &mp.CreatedAt,
	)
}

// MeterPointGap holds a meter point with additional gap-detection context.
type MeterPointGap struct {
	domain.MeterPoint
	MemberName    string     `json:"member_name"`
	LastReadingAt *time.Time `json:"last_reading_at,omitempty"` // nil = never had readings
}

const selectMeterPointMP = `mp.id, mp.member_id, mp.eeg_id, mp.zaehlpunkt, mp.energierichtung, mp.verteilungsmodell, mp.zugeteilte_menge_pct, mp.status, mp.registriert_seit, mp.abgemeldet_am, mp.generation_type, mp.gap_alert_sent_at, mp.notes, mp.consent_id, mp.created_at`

// GetMeterPointsWithGap returns active, registered meter points that have no readings
// within the last thresholdDays days and for which no gap alert has been sent yet.
func (r *MeterPointRepository) GetMeterPointsWithGap(ctx context.Context, eegID uuid.UUID, thresholdDays int) ([]MeterPointGap, error) {
	q := `
		SELECT ` + selectMeterPointMP + `, TRIM(m.name1 || ' ' || m.name2) AS member_name,
		       (SELECT MAX(er.ts) FROM energy_readings er WHERE er.meter_point_id = mp.id) AS last_reading_at
		FROM meter_points mp
		JOIN members m ON m.id = mp.member_id
		WHERE m.eeg_id = $1
		  AND m.status = 'ACTIVE'
		  AND mp.registriert_seit IS NOT NULL
		  AND mp.abgemeldet_am IS NULL
		  AND mp.gap_alert_sent_at IS NULL
		  AND mp.registriert_seit < NOW() - ($2::int * INTERVAL '1 day')
		  AND (
		    NOT EXISTS (SELECT 1 FROM energy_readings er WHERE er.meter_point_id = mp.id)
		    OR (SELECT MAX(er.ts) FROM energy_readings er WHERE er.meter_point_id = mp.id)
		       < NOW() - ($2::int * INTERVAL '1 day')
		  )`
	rows, err := r.db.Query(ctx, q, eegID, thresholdDays)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var result []MeterPointGap
	for rows.Next() {
		var g MeterPointGap
		if err := rows.Scan(
			&g.ID, &g.MemberID, &g.EegID, &g.Zaehlpunkt,
			&g.Energierichtung, &g.Verteilungsmodell, &g.ZugeteilteMenugePct,
			&g.Status, &g.RegistriertSeit, &g.AbgemeldetAm, &g.GenerationType, &g.GapAlertSentAt, &g.Notes, &g.ConsentID, &g.CreatedAt,
			&g.MemberName, &g.LastReadingAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

// ListCurrentGapAlerts returns meter points where gap_alert_sent_at IS NOT NULL for an EEG.
func (r *MeterPointRepository) ListCurrentGapAlerts(ctx context.Context, eegID uuid.UUID) ([]MeterPointGap, error) {
	q := `
		SELECT ` + selectMeterPointMP + `, TRIM(m.name1 || ' ' || m.name2) AS member_name,
		       (SELECT MAX(er.ts) FROM energy_readings er WHERE er.meter_point_id = mp.id) AS last_reading_at
		FROM meter_points mp
		JOIN members m ON m.id = mp.member_id
		WHERE m.eeg_id = $1
		  AND m.status = 'ACTIVE'
		  AND mp.registriert_seit IS NOT NULL
		  AND mp.abgemeldet_am IS NULL
		  AND mp.gap_alert_sent_at IS NOT NULL
		ORDER BY mp.gap_alert_sent_at DESC`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var result []MeterPointGap
	for rows.Next() {
		var g MeterPointGap
		if err := rows.Scan(
			&g.ID, &g.MemberID, &g.EegID, &g.Zaehlpunkt,
			&g.Energierichtung, &g.Verteilungsmodell, &g.ZugeteilteMenugePct,
			&g.Status, &g.RegistriertSeit, &g.AbgemeldetAm, &g.GenerationType, &g.GapAlertSentAt, &g.Notes, &g.ConsentID, &g.CreatedAt,
			&g.MemberName, &g.LastReadingAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

// SetGapAlertSent marks a meter point as having had a gap alert sent (sets gap_alert_sent_at = NOW()).
func (r *MeterPointRepository) SetGapAlertSent(ctx context.Context, meterPointID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE meter_points SET gap_alert_sent_at = NOW() WHERE id = $1`,
		meterPointID,
	)
	return err
}

// ResetGapAlertSent clears the gap_alert_sent_at on a single meter point.
func (r *MeterPointRepository) ResetGapAlertSent(ctx context.Context, meterPointID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE meter_points SET gap_alert_sent_at = NULL WHERE id = $1`,
		meterPointID,
	)
	return err
}

// ResetGapAlertIfReadingsRecent clears gap_alert_sent_at for meter points in an EEG
// that now have a reading within the last thresholdDays days.
func (r *MeterPointRepository) ResetGapAlertIfReadingsRecent(ctx context.Context, eegID uuid.UUID, thresholdDays int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE meter_points mp
		SET gap_alert_sent_at = NULL
		FROM members m
		WHERE mp.member_id = m.id
		  AND m.eeg_id = $1
		  AND mp.gap_alert_sent_at IS NOT NULL
		  AND EXISTS (
		    SELECT 1 FROM energy_readings er
		    WHERE er.meter_point_id = mp.id
		      AND er.ts >= NOW() - ($2::int * INTERVAL '1 day')
		  )`,
		eegID, thresholdDays,
	)
	return err
}

// ClearGapAlertForInactivePoints clears gap_alert_sent_at for meter points that are no longer
// eligible for alerts: member is INACTIVE, meter point is abgemeldet, or registriert_seit IS NULL.
func (r *MeterPointRepository) ClearGapAlertForInactivePoints(ctx context.Context, eegID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE meter_points mp
		SET gap_alert_sent_at = NULL
		FROM members m
		WHERE mp.member_id = m.id
		  AND m.eeg_id = $1
		  AND mp.gap_alert_sent_at IS NOT NULL
		  AND (m.status != 'ACTIVE' OR mp.abgemeldet_am IS NOT NULL OR mp.registriert_seit IS NULL)`,
		eegID,
	)
	return err
}

// UpdateRegistriertSeit sets the NB-confirmed activation date on a meter point.
func (r *MeterPointRepository) UpdateRegistriertSeit(ctx context.Context, meterPointID uuid.UUID, date time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE meter_points SET registriert_seit = $1 WHERE id = $2`,
		date.UTC().Truncate(24*time.Hour), meterPointID,
	)
	return err
}

// UpdateConsentID stores the NB-assigned ConsentId (from ZUSTIMMUNG_ECON) on a meter point.
func (r *MeterPointRepository) UpdateConsentID(ctx context.Context, meterPointID uuid.UUID, consentID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE meter_points SET consent_id = $1 WHERE id = $2`,
		consentID, meterPointID,
	)
	return err
}

// BackfillConsentIDs updates consent_id for meter points whose consent_id is currently
// empty, matching by zaehlpunkt. Used when a SENDEN_ECP (EC_PODLIST response) arrives
// containing the NB-assigned ConsentIds for all active meter points in the community.
// Only updates rows where consent_id = '' to avoid overwriting existing values.
// Returns the number of rows updated.
func (r *MeterPointRepository) BackfillConsentIDs(ctx context.Context, entries map[string]string) (int64, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	// Build CASE … END expression: UPDATE meter_points SET consent_id = CASE zaehlpunkt WHEN $1 THEN $2 … END
	// Simpler: loop individually — SENDEN_ECP responses are typically ≤ a few hundred entries.
	var total int64
	for zp, cid := range entries {
		if cid == "" {
			continue
		}
		tag, err := r.db.Exec(ctx,
			`UPDATE meter_points SET consent_id = $1 WHERE zaehlpunkt = $2 AND consent_id = ''`,
			cid, zp,
		)
		if err != nil {
			return total, fmt.Errorf("backfill consent_id for %s: %w", zp, err)
		}
		total += tag.RowsAffected()
	}
	return total, nil
}

// UpdateAbgemeldetAm sets the NB-confirmed deregistration date on a meter point.
func (r *MeterPointRepository) UpdateAbgemeldetAm(ctx context.Context, meterPointID uuid.UUID, date time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE meter_points SET abgemeldet_am = $1 WHERE id = $2`,
		date.UTC().Truncate(24*time.Hour), meterPointID,
	)
	return err
}

// ExistsByZaehlpunktInEEG returns true if a meter point with that Zählpunkt ID already exists in the EEG.
func (r *MeterPointRepository) ExistsByZaehlpunktInEEG(ctx context.Context, eegID uuid.UUID, zaehlpunkt string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM meter_points WHERE eeg_id=$1 AND zaehlpunkt=$2)`,
		eegID, zaehlpunkt,
	).Scan(&exists)
	return exists, err
}

func (r *MeterPointRepository) GetByZaehlpunkt(ctx context.Context, zaehlpunkt string) (*domain.MeterPoint, error) {
	q := `SELECT ` + selectMeterPoint + ` FROM meter_points WHERE zaehlpunkt = $1`
	var mp domain.MeterPoint
	if err := scanMeterPoint(r.db.QueryRow(ctx, q, zaehlpunkt), &mp); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &mp, nil
}

func (r *MeterPointRepository) ListByEeg(ctx context.Context, eegID uuid.UUID) ([]domain.MeterPoint, error) {
	q := `SELECT ` + selectMeterPoint + ` FROM meter_points WHERE eeg_id = $1`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var mps []domain.MeterPoint
	for rows.Next() {
		var mp domain.MeterPoint
		if err := scanMeterPoint(rows, &mp); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		mps = append(mps, mp)
	}
	return mps, rows.Err()
}

func (r *MeterPointRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.MeterPoint, error) {
	q := `SELECT ` + selectMeterPoint + ` FROM meter_points WHERE id = $1`
	var mp domain.MeterPoint
	if err := scanMeterPoint(r.db.QueryRow(ctx, q, id), &mp); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &mp, nil
}

func (r *MeterPointRepository) Create(ctx context.Context, mp *domain.MeterPoint) error {
	q := `INSERT INTO meter_points (member_id, eeg_id, zaehlpunkt, energierichtung, verteilungsmodell, zugeteilte_menge_pct, status, registriert_seit, generation_type)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	      RETURNING id, created_at`
	return r.db.QueryRow(ctx, q,
		mp.MemberID, mp.EegID, mp.Zaehlpunkt, mp.Energierichtung,
		mp.Verteilungsmodell, mp.ZugeteilteMenugePct, mp.Status, mp.RegistriertSeit, mp.GenerationType,
	).Scan(&mp.ID, &mp.CreatedAt)
}

func (r *MeterPointRepository) Update(ctx context.Context, mp *domain.MeterPoint) error {
	q := `UPDATE meter_points SET
	        energierichtung=$1, verteilungsmodell=$2, zugeteilte_menge_pct=$3, status=$4, generation_type=$5, notes=$6
	      WHERE id=$7`
	_, err := r.db.Exec(ctx, q,
		mp.Energierichtung, mp.Verteilungsmodell, mp.ZugeteilteMenugePct, mp.Status, mp.GenerationType, mp.Notes, mp.ID,
	)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	return nil
}

func (r *MeterPointRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM meter_points WHERE id=$1`
	_, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (r *MeterPointRepository) ListByMember(ctx context.Context, memberID uuid.UUID) ([]domain.MeterPoint, error) {
	q := `SELECT ` + selectMeterPoint + ` FROM meter_points WHERE member_id = $1`
	rows, err := r.db.Query(ctx, q, memberID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var mps []domain.MeterPoint
	for rows.Next() {
		var mp domain.MeterPoint
		if err := scanMeterPoint(rows, &mp); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		mps = append(mps, mp)
	}
	return mps, rows.Err()
}

func (r *MeterPointRepository) SearchByEeg(ctx context.Context, eegID uuid.UUID, query string, limit int) ([]MeterPointSearchResult, error) {
	q := `SELECT mp.id, mp.zaehlpunkt, mp.energierichtung, m.id, TRIM(m.name1 || ' ' || m.name2)
	      FROM meter_points mp
	      JOIN members m ON m.id = mp.member_id
	      WHERE mp.eeg_id = $1 AND mp.zaehlpunkt ILIKE $2
	      ORDER BY mp.zaehlpunkt
	      LIMIT $3`
	rows, err := r.db.Query(ctx, q, eegID, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var results []MeterPointSearchResult
	for rows.Next() {
		var res MeterPointSearchResult
		if err := rows.Scan(&res.ID, &res.Zaehlpunkt, &res.Energierichtung, &res.MemberID, &res.MemberName); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, res)
	}
	return results, rows.Err()
}

// MeterPointEDAStatus holds the latest EDA Anmeldung/Abmeldung process status for a meter point.
type MeterPointEDAStatus struct {
	AnmeldungStatus     *string    // latest EC_REQ_ONL status; nil = no process
	AbmeldungStatus     *string    // latest CM_REV_SP status; nil = no process
	ParticipationFactor *float64   // from confirmed/latest EC_REQ_ONL; nil = unknown
	FactorValidFrom     *time.Time // valid_from of the process that set ParticipationFactor
}

// GetEDAStatusByMeterPoints returns the latest EC_REQ_ONL and CM_REV_SP
// status for each of the provided meter point IDs. Keys missing from the result
// map have no EDA processes on record.
// For EC_REQ_ONL the confirmed process is preferred over rejected/pending ones
// so that the participation factor reflects the active registration.
func (r *MeterPointRepository) GetEDAStatusByMeterPoints(ctx context.Context, mpIDs []uuid.UUID) (map[uuid.UUID]MeterPointEDAStatus, error) {
	if len(mpIDs) == 0 {
		return map[uuid.UUID]MeterPointEDAStatus{}, nil
	}
	q := `
		SELECT DISTINCT ON (meter_point_id, process_type)
		  meter_point_id, process_type, status, participation_factor, valid_from
		FROM eda_processes
		WHERE meter_point_id = ANY($1)
		  AND process_type IN ('EC_REQ_ONL', 'CM_REV_SP')
		ORDER BY meter_point_id, process_type,
		  CASE status WHEN 'confirmed' THEN 0 WHEN 'first_confirmed' THEN 1 ELSE 2 END,
		  initiated_at DESC`
	rows, err := r.db.Query(ctx, q, mpIDs)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]MeterPointEDAStatus)
	for rows.Next() {
		var mpID uuid.UUID
		var processType, status string
		var factor *float64
		var validFrom *time.Time
		if err := rows.Scan(&mpID, &processType, &status, &factor, &validFrom); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		s := result[mpID]
		switch processType {
		case "EC_REQ_ONL":
			s.AnmeldungStatus = &status
			s.ParticipationFactor = factor
			s.FactorValidFrom = validFrom
		case "CM_REV_SP":
			s.AbmeldungStatus = &status
		}
		result[mpID] = s
	}
	return result, rows.Err()
}
