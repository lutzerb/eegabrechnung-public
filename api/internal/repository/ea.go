package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutzerb/eegabrechnung/internal/domain"
)

type EARepository struct {
	db *pgxpool.Pool
}

func NewEARepository(db *pgxpool.Pool) *EARepository {
	return &EARepository{db: db}
}

// ── USt calculation ──────────────────────────────────────────────────────────

// CalcUSt derives ust_pct, ust_betrag, and betrag_netto from betrag_brutto + ust_code.
// For RC codes: brutto == netto (payment amount), USt is self-assessed on top.
// For VST/UST codes: brutto includes the VAT; netto is derived by dividing out.
func CalcUSt(brutto float64, code string) (pct float64, ustBetrag float64, netto float64) {
	switch code {
	case "UST_20", "VST_20":
		pct = 20
		netto = brutto / 1.20
		ustBetrag = brutto - netto
	case "UST_10", "VST_10":
		pct = 10
		netto = brutto / 1.10
		ustBetrag = brutto - netto
	case "RC_20":
		pct = 20
		netto = brutto
		ustBetrag = brutto * 0.20
	case "RC_13":
		pct = 13
		netto = brutto
		ustBetrag = brutto * 0.13
	default: // KEINE
		pct = 0
		netto = brutto
		ustBetrag = 0
	}
	return
}

// ── Konten ────────────────────────────────────────────────────────────────────

const kontoCols = `id, eeg_id, nummer, name, typ, ust_relevanz, standard_ust_pct, uva_kz, k1_kz, sortierung, aktiv, created_at`

func scanKonto(row interface{ Scan(...any) error }, k *domain.EAKonto) error {
	return row.Scan(
		&k.ID, &k.EegID, &k.Nummer, &k.Name, &k.Typ, &k.UstRelevanz,
		&k.StandardUstPct, &k.UvaKZ, &k.K1KZ, &k.Sortierung, &k.Aktiv, &k.CreatedAt,
	)
}

func (r *EARepository) ListKonten(ctx context.Context, eegID uuid.UUID, nurAktiv bool) ([]domain.EAKonto, error) {
	q := `SELECT ` + kontoCols + ` FROM ea_konten WHERE eeg_id = $1`
	if nurAktiv {
		q += ` AND aktiv = true`
	}
	q += ` ORDER BY sortierung, nummer`
	rows, err := r.db.Query(ctx, q, eegID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var konten []domain.EAKonto
	for rows.Next() {
		var k domain.EAKonto
		if err := scanKonto(rows, &k); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		konten = append(konten, k)
	}
	return konten, rows.Err()
}

func (r *EARepository) GetKonto(ctx context.Context, id uuid.UUID) (*domain.EAKonto, error) {
	q := `SELECT ` + kontoCols + ` FROM ea_konten WHERE id = $1`
	var k domain.EAKonto
	if err := scanKonto(r.db.QueryRow(ctx, q, id), &k); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &k, nil
}

func (r *EARepository) CreateKonto(ctx context.Context, k *domain.EAKonto) error {
	q := `INSERT INTO ea_konten (eeg_id, nummer, name, typ, ust_relevanz, standard_ust_pct, uva_kz, k1_kz, sortierung, aktiv)
	      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	      RETURNING id, created_at`
	return r.db.QueryRow(ctx, q,
		k.EegID, k.Nummer, k.Name, k.Typ, k.UstRelevanz, k.StandardUstPct, k.UvaKZ, k.K1KZ, k.Sortierung, k.Aktiv,
	).Scan(&k.ID, &k.CreatedAt)
}

func (r *EARepository) UpdateKonto(ctx context.Context, k *domain.EAKonto) error {
	q := `UPDATE ea_konten SET nummer=$1, name=$2, typ=$3, ust_relevanz=$4, standard_ust_pct=$5, uva_kz=$6, k1_kz=$7, sortierung=$8, aktiv=$9
	      WHERE id=$10 AND eeg_id=$11`
	_, err := r.db.Exec(ctx, q, k.Nummer, k.Name, k.Typ, k.UstRelevanz, k.StandardUstPct, k.UvaKZ, k.K1KZ, k.Sortierung, k.Aktiv, k.ID, k.EegID)
	return err
}

func (r *EARepository) DeleteKonto(ctx context.Context, id, eegID uuid.UUID) error {
	// Only allow deletion if no non-deleted bookings reference this account
	var cnt int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM ea_buchungen WHERE konto_id=$1 AND deleted_at IS NULL`, id).Scan(&cnt); err != nil {
		return fmt.Errorf("check: %w", err)
	}
	if cnt > 0 {
		return fmt.Errorf("konto hat %d Buchungen und kann nicht gelöscht werden; bitte deaktivieren", cnt)
	}
	_, err := r.db.Exec(ctx, `DELETE FROM ea_konten WHERE id=$1 AND eeg_id=$2`, id, eegID)
	return err
}

// KontenExists checks whether any accounts exist for this EEG yet.
func (r *EARepository) KontenExists(ctx context.Context, eegID uuid.UUID) (bool, error) {
	var cnt int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM ea_konten WHERE eeg_id=$1`, eegID).Scan(&cnt)
	return cnt > 0, err
}

// SeedDefaultKonten inserts the standard EEG-Verein chart of accounts.
func (r *EARepository) SeedDefaultKonten(ctx context.Context, eegID uuid.UUID) error {
	defaults := []domain.EAKonto{
		// Einnahmen — K1 KZ9040 (Betriebseinnahmen / Umsatzerlöse)
		{Nummer: "4000", Name: "Bezugsgebühren Mitglieder", Typ: "EINNAHME", UstRelevanz: "KEINE", K1KZ: "9040", Sortierung: 10},
		{Nummer: "4010", Name: "Grundgebühren / Mitgliedsbeiträge", Typ: "EINNAHME", UstRelevanz: "KEINE", K1KZ: "9040", Sortierung: 20},
		{Nummer: "4900", Name: "Sonstige Einnahmen", Typ: "EINNAHME", UstRelevanz: "STEUERBAR", K1KZ: "9040", Sortierung: 90},
		// Ausgaben – Einspeisevergütungen — K1 KZ9100 (Wareneinsatz / Materialeinsatz)
		{Nummer: "6000", Name: "Einspeisevergütungen Privatpersonen", Typ: "AUSGABE", UstRelevanz: "KEINE", K1KZ: "9100", Sortierung: 100},
		{Nummer: "6010", Name: "Einspeisevergütungen Unternehmen (RC 20%)", Typ: "AUSGABE", UstRelevanz: "RC", StandardUstPct: ptr64(20), UvaKZ: "083", K1KZ: "9100", Sortierung: 110},
		{Nummer: "6015", Name: "Einspeisevergütungen LuF pauschaliert (RC 13%)", Typ: "AUSGABE", UstRelevanz: "RC", StandardUstPct: ptr64(13), UvaKZ: "083", K1KZ: "9100", Sortierung: 120},
		// Ausgaben – Betrieb
		{Nummer: "6100", Name: "Netzkosten / Netzbetreibergebühren", Typ: "AUSGABE", UstRelevanz: "VST", StandardUstPct: ptr64(20), K1KZ: "9230", Sortierung: 200}, // Übrige Aufwendungen (kein separater K1-KZ für Netzkosten)
		{Nummer: "6200", Name: "Bankspesen und -gebühren", Typ: "AUSGABE", UstRelevanz: "KEINE", K1KZ: "9220", Sortierung: 210},                                    // Zinsen und ähnliche Aufwendungen
		{Nummer: "6300", Name: "Vereinskosten", Typ: "AUSGABE", UstRelevanz: "VST", StandardUstPct: ptr64(20), K1KZ: "9230", Sortierung: 220},                      // Übrige Aufwendungen
		{Nummer: "6400", Name: "IT / Hosting / Software", Typ: "AUSGABE", UstRelevanz: "VST", StandardUstPct: ptr64(20), K1KZ: "9230", Sortierung: 230},            // Übrige Aufwendungen
		{Nummer: "6500", Name: "Steuerberatung / Buchhaltung", Typ: "AUSGABE", UstRelevanz: "VST", StandardUstPct: ptr64(20), K1KZ: "9230", Sortierung: 240},       // Übrige Aufwendungen
		{Nummer: "6900", Name: "Sonstige Ausgaben", Typ: "AUSGABE", UstRelevanz: "VST", StandardUstPct: ptr64(20), K1KZ: "9230", Sortierung: 290},                  // Übrige Aufwendungen
		// Geldkonten — kein K1 KZ (Bestandskonten)
		{Nummer: "1000", Name: "Bankkonto Hauptkonto", Typ: "SONSTIG", UstRelevanz: "KEINE", Sortierung: 500},
		{Nummer: "1010", Name: "Bankkonto SEPA-Einzug", Typ: "SONSTIG", UstRelevanz: "KEINE", Sortierung: 510},
	}
	for i := range defaults {
		defaults[i].EegID = eegID
		defaults[i].Aktiv = true
		if err := r.CreateKonto(ctx, &defaults[i]); err != nil {
			return fmt.Errorf("seed konto %s: %w", defaults[i].Nummer, err)
		}
	}
	return nil
}

func ptr64(v float64) *float64 { return &v }

// ── Buchungen ─────────────────────────────────────────────────────────────────

const buchungCols = `b.id, b.eeg_id, b.geschaeftsjahr, COALESCE(b.buchungsnr,''), b.zahlung_datum, b.beleg_datum,
	COALESCE(b.belegnr,''), b.beschreibung, b.konto_id, b.richtung, b.betrag_brutto, b.ust_code, b.ust_pct,
	b.ust_betrag, b.betrag_netto, COALESCE(b.gegenseite,''), b.quelle, b.quelle_id, b.beleg_id,
	COALESCE(b.notizen,''), b.erstellt_von, b.erstellt_am, b.aktualisiert_am,
	b.deleted_at, COALESCE(b.deleted_by,'')`

func scanBuchung(row interface{ Scan(...any) error }, b *domain.EABuchung) error {
	return row.Scan(
		&b.ID, &b.EegID, &b.Geschaeftsjahr, &b.Buchungsnr, &b.ZahlungDatum, &b.BelegDatum,
		&b.Belegnr, &b.Beschreibung, &b.KontoID, &b.Richtung, &b.BetragBrutto, &b.UstCode, &b.UstPct,
		&b.UstBetrag, &b.BetragNetto, &b.Gegenseite, &b.Quelle, &b.QuelleID, &b.BelegID,
		&b.Notizen, &b.ErstelltVon, &b.ErstelltAm, &b.AktualisiertAm,
		&b.DeletedAt, &b.DeletedBy,
	)
}

func (r *EARepository) ListBuchungen(ctx context.Context, eegID uuid.UUID, filters BuchungFilter) ([]domain.EABuchung, error) {
	args := []any{eegID}
	where := []string{"b.eeg_id = $1"}

	if !filters.InclDeleted {
		where = append(where, "b.deleted_at IS NULL")
	}
	if filters.Geschaeftsjahr > 0 {
		args = append(args, filters.Geschaeftsjahr)
		where = append(where, fmt.Sprintf("b.geschaeftsjahr = $%d", len(args)))
	}
	if !filters.Von.IsZero() {
		args = append(args, filters.Von)
		where = append(where, fmt.Sprintf("b.zahlung_datum >= $%d", len(args)))
	}
	if !filters.Bis.IsZero() {
		args = append(args, filters.Bis)
		where = append(where, fmt.Sprintf("b.zahlung_datum <= $%d", len(args)))
	}
	if filters.KontoID != uuid.Nil {
		args = append(args, filters.KontoID)
		where = append(where, fmt.Sprintf("b.konto_id = $%d", len(args)))
	}
	if filters.Richtung != "" {
		args = append(args, filters.Richtung)
		where = append(where, fmt.Sprintf("b.richtung = $%d", len(args)))
	}
	if filters.NurBezahlt {
		where = append(where, "b.zahlung_datum IS NOT NULL")
	}
	if filters.NurOffen {
		where = append(where, "b.zahlung_datum IS NULL")
	}

	q := `SELECT ` + buchungCols + `
	      FROM ea_buchungen b
	      WHERE ` + strings.Join(where, " AND ") + `
	      ORDER BY b.zahlung_datum DESC NULLS FIRST, b.erstellt_am DESC`

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var buchungen []domain.EABuchung
	for rows.Next() {
		var b domain.EABuchung
		if err := scanBuchung(rows, &b); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		buchungen = append(buchungen, b)
	}
	return buchungen, rows.Err()
}

type BuchungFilter struct {
	Geschaeftsjahr int
	Von, Bis       time.Time
	KontoID        uuid.UUID
	Richtung       string
	NurBezahlt     bool
	NurOffen       bool
	InclDeleted    bool // include soft-deleted entries (BAO §131 audit view)
}

func (r *EARepository) GetBuchung(ctx context.Context, id, eegID uuid.UUID) (*domain.EABuchung, error) {
	q := `SELECT ` + buchungCols + ` FROM ea_buchungen b WHERE b.id=$1 AND b.eeg_id=$2`
	var b domain.EABuchung
	if err := scanBuchung(r.db.QueryRow(ctx, q, id, eegID), &b); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &b, nil
}

// NextBuchungsnr generates the next sequential booking number for the year.
func (r *EARepository) NextBuchungsnr(ctx context.Context, eegID uuid.UUID, jahr int) (string, error) {
	var max *string
	err := r.db.QueryRow(ctx,
		`SELECT MAX(buchungsnr) FROM ea_buchungen WHERE eeg_id=$1 AND geschaeftsjahr=$2 AND buchungsnr IS NOT NULL`,
		eegID, jahr,
	).Scan(&max)
	if err != nil {
		return "", err
	}
	seq := 1
	if max != nil {
		// format: "2026-0042" → parse last part
		parts := strings.SplitN(*max, "-", 2)
		if len(parts) == 2 {
			fmt.Sscanf(parts[1], "%d", &seq)
			seq++
		}
	}
	return fmt.Sprintf("%d-%04d", jahr, seq), nil
}

// buchungSnapshot serialises the mutable fields of a booking for the changelog.
func buchungSnapshot(b *domain.EABuchung) json.RawMessage {
	m := map[string]any{
		"buchungsnr":    b.Buchungsnr,
		"beschreibung":  b.Beschreibung,
		"zahlung_datum": b.ZahlungDatum,
		"beleg_datum":   b.BelegDatum,
		"belegnr":       b.Belegnr,
		"konto_id":      b.KontoID,
		"richtung":      b.Richtung,
		"betrag_brutto": b.BetragBrutto,
		"ust_code":      b.UstCode,
		"gegenseite":    b.Gegenseite,
		"notizen":       b.Notizen,
	}
	data, _ := json.Marshal(m)
	return data
}

func nullableJSON(v json.RawMessage) any {
	if len(v) == 0 {
		return nil
	}
	return v
}

func (r *EARepository) CreateBuchung(ctx context.Context, b *domain.EABuchung, changedBy string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := `INSERT INTO ea_buchungen
	        (eeg_id, geschaeftsjahr, buchungsnr, zahlung_datum, beleg_datum, belegnr, beschreibung,
	         konto_id, richtung, betrag_brutto, ust_code, ust_pct, ust_betrag, betrag_netto,
	         gegenseite, quelle, quelle_id, notizen, erstellt_von)
	      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
	      RETURNING id, erstellt_am, aktualisiert_am`
	if err := tx.QueryRow(ctx, q,
		b.EegID, b.Geschaeftsjahr, nilStr(b.Buchungsnr), b.ZahlungDatum, b.BelegDatum, nilStr(b.Belegnr), b.Beschreibung,
		b.KontoID, b.Richtung, b.BetragBrutto, b.UstCode, b.UstPct, b.UstBetrag, b.BetragNetto,
		nilStr(b.Gegenseite), b.Quelle, b.QuelleID, nilStr(b.Notizen), b.ErstelltVon,
	).Scan(&b.ID, &b.ErstelltAm, &b.AktualisiertAm); err != nil {
		return err
	}

	if err := txWriteChangelog(ctx, tx, b.ID, "create", changedBy, "", nil, buchungSnapshot(b)); err != nil {
		return fmt.Errorf("changelog: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *EARepository) UpdateBuchung(ctx context.Context, b *domain.EABuchung, changedBy, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Capture old state before update
	var old domain.EABuchung
	if err := scanBuchung(
		tx.QueryRow(ctx, `SELECT `+buchungCols+` FROM ea_buchungen b WHERE b.id=$1 AND b.eeg_id=$2`, b.ID, b.EegID),
		&old,
	); err != nil {
		return fmt.Errorf("fetch old: %w", err)
	}

	q := `UPDATE ea_buchungen SET
	        zahlung_datum=$1, beleg_datum=$2, belegnr=$3, beschreibung=$4,
	        konto_id=$5, richtung=$6, betrag_brutto=$7, ust_code=$8, ust_pct=$9,
	        ust_betrag=$10, betrag_netto=$11, gegenseite=$12, notizen=$13, beleg_id=$14,
	        aktualisiert_am=NOW()
	      WHERE id=$15 AND eeg_id=$16
	      RETURNING aktualisiert_am`
	if err := tx.QueryRow(ctx, q,
		b.ZahlungDatum, b.BelegDatum, nilStr(b.Belegnr), b.Beschreibung,
		b.KontoID, b.Richtung, b.BetragBrutto, b.UstCode, b.UstPct,
		b.UstBetrag, b.BetragNetto, nilStr(b.Gegenseite), nilStr(b.Notizen), b.BelegID,
		b.ID, b.EegID,
	).Scan(&b.AktualisiertAm); err != nil {
		return err
	}

	if err := txWriteChangelog(ctx, tx, b.ID, "update", changedBy, reason, buchungSnapshot(&old), buchungSnapshot(b)); err != nil {
		return fmt.Errorf("changelog: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *EARepository) DeleteBuchung(ctx context.Context, id, eegID uuid.UUID, changedBy, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Capture state before deletion
	var old domain.EABuchung
	if err := scanBuchung(
		tx.QueryRow(ctx, `SELECT `+buchungCols+` FROM ea_buchungen b WHERE b.id=$1 AND b.eeg_id=$2 AND b.deleted_at IS NULL`, id, eegID),
		&old,
	); err != nil {
		return fmt.Errorf("not found: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE ea_buchungen SET deleted_at=NOW(), deleted_by=$1 WHERE id=$2 AND eeg_id=$3`,
		changedBy, id, eegID,
	); err != nil {
		return err
	}

	if err := txWriteChangelog(ctx, tx, id, "delete", changedBy, reason, buchungSnapshot(&old), nil); err != nil {
		return fmt.Errorf("changelog: %w", err)
	}
	return tx.Commit(ctx)
}

// txWriteChangelog inserts a changelog entry within an open transaction.
func txWriteChangelog(ctx context.Context, tx pgx.Tx, buchungID uuid.UUID, operation, changedBy, reason string, oldValues, newValues json.RawMessage) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO ea_buchungen_changelog (buchung_id, operation, changed_by, old_values, new_values, reason)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		buchungID, operation, changedBy,
		nullableJSON(oldValues), nullableJSON(newValues), nilStr(reason),
	)
	return err
}

// IsDuplicateImport checks if an invoice was already imported as a non-deleted Buchung.
func (r *EARepository) IsDuplicateImport(ctx context.Context, eegID, invoiceID uuid.UUID) (bool, error) {
	var cnt int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM ea_buchungen WHERE eeg_id=$1 AND quelle_id=$2 AND deleted_at IS NULL`,
		eegID, invoiceID,
	).Scan(&cnt)
	return cnt > 0, err
}

// ── Belege ────────────────────────────────────────────────────────────────────

func (r *EARepository) CreateBeleg(ctx context.Context, b *domain.EABeleg) error {
	q := `INSERT INTO ea_belege (eeg_id, buchung_id, dateiname, pfad, groesse, mime_typ, beschreibung, hochgeladen_von)
	      VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	      RETURNING id, hochgeladen_am`
	return r.db.QueryRow(ctx, q,
		b.EegID, b.BuchungID, b.Dateiname, b.Pfad, b.Groesse, b.MimeTyp, b.Beschreibung, b.HochgeladenVon,
	).Scan(&b.ID, &b.HochgeladenAm)
}

func (r *EARepository) GetBeleg(ctx context.Context, id, eegID uuid.UUID) (*domain.EABeleg, error) {
	q := `SELECT id, eeg_id, buchung_id, dateiname, pfad, groesse, mime_typ, beschreibung, hochgeladen_am, hochgeladen_von
	      FROM ea_belege WHERE id=$1 AND eeg_id=$2`
	var b domain.EABeleg
	err := r.db.QueryRow(ctx, q, id, eegID).Scan(
		&b.ID, &b.EegID, &b.BuchungID, &b.Dateiname, &b.Pfad, &b.Groesse, &b.MimeTyp, &b.Beschreibung, &b.HochgeladenAm, &b.HochgeladenVon,
	)
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &b, nil
}

func (r *EARepository) ListBelegeForBuchung(ctx context.Context, buchungID uuid.UUID) ([]domain.EABeleg, error) {
	q := `SELECT id, eeg_id, buchung_id, dateiname, pfad, groesse, mime_typ, beschreibung, hochgeladen_am, hochgeladen_von
	      FROM ea_belege WHERE buchung_id=$1 ORDER BY hochgeladen_am`
	rows, err := r.db.Query(ctx, q, buchungID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.EABeleg
	for rows.Next() {
		var b domain.EABeleg
		if err := rows.Scan(&b.ID, &b.EegID, &b.BuchungID, &b.Dateiname, &b.Pfad, &b.Groesse, &b.MimeTyp, &b.Beschreibung, &b.HochgeladenAm, &b.HochgeladenVon); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

func (r *EARepository) LinkBelegToBuchung(ctx context.Context, belegID, buchungID, eegID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE ea_belege SET buchung_id=$1 WHERE id=$2 AND eeg_id=$3`,
		buchungID, belegID, eegID,
	)
	if err != nil {
		return err
	}
	// also update buchung.beleg_id (first beleg wins)
	_, err = r.db.Exec(ctx,
		`UPDATE ea_buchungen SET beleg_id=$1 WHERE id=$2 AND eeg_id=$3 AND beleg_id IS NULL`,
		belegID, buchungID, eegID,
	)
	return err
}

func (r *EARepository) DeleteBeleg(ctx context.Context, id, eegID uuid.UUID) (string, error) {
	var pfad string
	err := r.db.QueryRow(ctx, `DELETE FROM ea_belege WHERE id=$1 AND eeg_id=$2 RETURNING pfad`, id, eegID).Scan(&pfad)
	return pfad, err
}

// ── Saldenliste / Reports ─────────────────────────────────────────────────────

func (r *EARepository) Saldenliste(ctx context.Context, eegID uuid.UUID, von, bis *time.Time) ([]domain.EASaldenlisteEintrag, error) {
	args := []any{eegID}
	dateFilter := ""
	if von != nil {
		args = append(args, *von)
		dateFilter += fmt.Sprintf(" AND b.zahlung_datum >= $%d", len(args))
	}
	if bis != nil {
		args = append(args, *bis)
		dateFilter += fmt.Sprintf(" AND b.zahlung_datum <= $%d", len(args))
	}

	q := fmt.Sprintf(`
		SELECT k.id, k.nummer, k.name, k.typ, k.k1_kz,
		       COALESCE(SUM(CASE WHEN b.richtung='EINNAHME' THEN ABS(b.betrag_netto) ELSE 0 END), 0) AS einnahmen,
		       COALESCE(SUM(CASE WHEN b.richtung='AUSGABE'  THEN ABS(b.betrag_netto) ELSE 0 END), 0) AS ausgaben,
		       COUNT(b.id) AS anzahl
		FROM ea_konten k
		LEFT JOIN ea_buchungen b ON b.konto_id = k.id AND b.zahlung_datum IS NOT NULL AND b.deleted_at IS NULL %s
		WHERE k.eeg_id = $1 AND k.aktiv = true
		GROUP BY k.id, k.nummer, k.name, k.typ, k.k1_kz, k.sortierung
		ORDER BY k.sortierung, k.nummer`, dateFilter)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var result []domain.EASaldenlisteEintrag
	for rows.Next() {
		var e domain.EASaldenlisteEintrag
		if err := rows.Scan(&e.KontoID, &e.Nummer, &e.Name, &e.Typ, &e.K1KZ, &e.Einnahmen, &e.Ausgaben, &e.AnzahlBuchungen); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		e.Saldo = e.Einnahmen - e.Ausgaben
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *EARepository) Jahresabschluss(ctx context.Context, eegID uuid.UUID, jahr int) (*domain.EAJahresabschluss, error) {
	von := time.Date(jahr, 1, 1, 0, 0, 0, 0, time.UTC)
	bis := time.Date(jahr, 12, 31, 23, 59, 59, 0, time.UTC)
	rows, err := r.Saldenliste(ctx, eegID, &von, &bis)
	if err != nil {
		return nil, err
	}
	ja := &domain.EAJahresabschluss{Jahr: jahr}
	for _, e := range rows {
		switch e.Typ {
		case "EINNAHME":
			if e.Einnahmen > 0 || e.AnzahlBuchungen > 0 {
				ja.Einnahmen = append(ja.Einnahmen, e)
				ja.TotalEinnahmen += e.Einnahmen
			}
		case "AUSGABE":
			if e.Ausgaben > 0 || e.AnzahlBuchungen > 0 {
				ja.Ausgaben = append(ja.Ausgaben, e)
				ja.TotalAusgaben += e.Ausgaben
			}
		}
	}
	ja.Ueberschuss = ja.TotalEinnahmen - ja.TotalAusgaben
	return ja, nil
}

func (r *EARepository) Kontenblatt(ctx context.Context, eegID, kontoID uuid.UUID, von, bis *time.Time) (*domain.EAKontenblatt, error) {
	konto, err := r.GetKonto(ctx, kontoID)
	if err != nil {
		return nil, fmt.Errorf("konto: %w", err)
	}
	if konto.EegID != eegID {
		return nil, fmt.Errorf("konto not found")
	}

	args := []any{kontoID}
	dateFilter := ""
	if von != nil {
		args = append(args, *von)
		dateFilter += fmt.Sprintf(" AND b.zahlung_datum >= $%d", len(args))
	}
	if bis != nil {
		args = append(args, *bis)
		dateFilter += fmt.Sprintf(" AND b.zahlung_datum <= $%d", len(args))
	}

	q := fmt.Sprintf(`SELECT `+buchungCols+`
		FROM ea_buchungen b
		WHERE b.konto_id = $1 AND b.zahlung_datum IS NOT NULL AND b.deleted_at IS NULL %s
		ORDER BY b.zahlung_datum, b.erstellt_am`, dateFilter)

	dbrows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer dbrows.Close()

	kb := &domain.EAKontenblatt{Konto: *konto}
	var running float64
	for dbrows.Next() {
		var b domain.EABuchung
		if err := scanBuchung(dbrows, &b); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if b.Richtung == "EINNAHME" {
			running += b.BetragNetto
		} else {
			running -= b.BetragNetto
		}
		kb.Eintraege = append(kb.Eintraege, domain.EAKontenblattEintrag{EABuchung: b, LaufenderSaldo: running})
	}
	kb.Summe = running
	return kb, dbrows.Err()
}

// ── UVA ───────────────────────────────────────────────────────────────────────

const uvaCols = `id, eeg_id, jahr, periodentyp, periode_nr, datum_von, datum_bis, status,
	kz_000, kz_022, kz_029, kz_044, kz_056, kz_057, kz_060, kz_065, kz_066, kz_083, zahllast,
	eingereicht_am, erstellt_am`

func scanUVA(row interface{ Scan(...any) error }, u *domain.EAUVAPeriode) error {
	return row.Scan(
		&u.ID, &u.EegID, &u.Jahr, &u.Periodentyp, &u.PeriodeNr, &u.DatumVon, &u.DatumBis, &u.Status,
		&u.KZ000, &u.KZ022, &u.KZ029, &u.KZ044, &u.KZ056, &u.KZ057, &u.KZ060, &u.KZ065, &u.KZ066, &u.KZ083, &u.Zahllast,
		&u.EingereichtAm, &u.ErstelltAm,
	)
}

func (r *EARepository) ListUVA(ctx context.Context, eegID uuid.UUID, jahr int) ([]domain.EAUVAPeriode, error) {
	args := []any{eegID}
	where := "eeg_id = $1"
	if jahr > 0 {
		args = append(args, jahr)
		where += fmt.Sprintf(" AND jahr = $%d", len(args))
	}
	rows, err := r.db.Query(ctx, `SELECT `+uvaCols+` FROM ea_uva_perioden WHERE `+where+` ORDER BY jahr DESC, periode_nr`, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var result []domain.EAUVAPeriode
	for rows.Next() {
		var u domain.EAUVAPeriode
		if err := scanUVA(rows, &u); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		u.PeriodeLabel = uvaPeriodeLabel(u)
		result = append(result, u)
	}
	return result, rows.Err()
}

func (r *EARepository) GetUVA(ctx context.Context, id, eegID uuid.UUID) (*domain.EAUVAPeriode, error) {
	q := `SELECT ` + uvaCols + ` FROM ea_uva_perioden WHERE id=$1 AND eeg_id=$2`
	var u domain.EAUVAPeriode
	if err := scanUVA(r.db.QueryRow(ctx, q, id, eegID), &u); err != nil {
		return nil, err
	}
	u.PeriodeLabel = uvaPeriodeLabel(u)
	return &u, nil
}

func (r *EARepository) UpsertUVA(ctx context.Context, u *domain.EAUVAPeriode) error {
	q := `INSERT INTO ea_uva_perioden
	        (eeg_id, jahr, periodentyp, periode_nr, datum_von, datum_bis, status,
	         kz_000, kz_022, kz_029, kz_044, kz_056, kz_057, kz_060, kz_065, kz_066, kz_083, zahllast)
	      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
	      ON CONFLICT (eeg_id, jahr, periodentyp, periode_nr) DO UPDATE SET
	        kz_000=$8, kz_022=$9, kz_029=$10, kz_044=$11, kz_056=$12, kz_057=$13,
	        kz_060=$14, kz_065=$15, kz_066=$16, kz_083=$17, zahllast=$18
	      RETURNING id, erstellt_am`
	return r.db.QueryRow(ctx, q,
		u.EegID, u.Jahr, u.Periodentyp, u.PeriodeNr, u.DatumVon, u.DatumBis, u.Status,
		u.KZ000, u.KZ022, u.KZ029, u.KZ044, u.KZ056, u.KZ057, u.KZ060, u.KZ065, u.KZ066, u.KZ083, u.Zahllast,
	).Scan(&u.ID, &u.ErstelltAm)
}

func (r *EARepository) SetUVAEingereicht(ctx context.Context, id, eegID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE ea_uva_perioden SET status='eingereicht', eingereicht_am=NOW() WHERE id=$1 AND eeg_id=$2`,
		id, eegID,
	)
	return err
}

// CalcUVAKennzahlen computes UVA Kennzahlen from bookings for the given period.
//
// KZ mapping (per Austrian UStG and UVA form U30):
//   UST_20  EINNAHME → KZ000+KZ022 (base), KZ056 (20% output tax)
//   UST_10  EINNAHME → KZ000+KZ029 (base), KZ044 (10% output tax)
//   KEINE   EINNAHME → KZ000 (exempt/Kleinunternehmer turnover, no output tax)
//   RC_20   AUSGABE  → KZ057 (Steuerschuld §19 Abs.1, self-assessed RC tax)
//                      KZ060 (deductible as VST if full deduction right)
//   RC_13   AUSGABE  → same as RC_20
//   VST_20  AUSGABE  → KZ060 (input VAT from supplier invoices)
//   VST_10  AUSGABE  → KZ060
func (r *EARepository) CalcUVAKennzahlen(ctx context.Context, eegID uuid.UUID, von, bis time.Time) (*domain.EAUVAPeriode, error) {
	q := `SELECT ust_code, richtung, SUM(ABS(betrag_netto)), SUM(ABS(ust_betrag))
	      FROM ea_buchungen
	      WHERE eeg_id=$1 AND zahlung_datum >= $2 AND zahlung_datum <= $3
	        AND zahlung_datum IS NOT NULL AND deleted_at IS NULL
	      GROUP BY ust_code, richtung`
	rows, err := r.db.Query(ctx, q, eegID, von, bis)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	u := &domain.EAUVAPeriode{EegID: eegID, DatumVon: von, DatumBis: bis}
	for rows.Next() {
		var code, richtung string
		var netto, ust float64
		if err := rows.Scan(&code, &richtung, &netto, &ust); err != nil {
			return nil, err
		}
		switch code {
		case "UST_20":
			if richtung == "EINNAHME" {
				u.KZ000 += netto
				u.KZ022 += netto
				u.KZ056 += ust
			}
		case "UST_10":
			if richtung == "EINNAHME" {
				u.KZ000 += netto
				u.KZ029 += netto
				u.KZ044 += ust
			}
		case "KEINE":
			if richtung == "EINNAHME" {
				u.KZ000 += netto // steuerfreie/Kleinunternehmer-Umsätze
			}
		case "RC_20", "RC_13":
			if richtung == "AUSGABE" {
				// Self-assessed RC tax under §19 Abs. 1 UStG
				u.KZ057 += ust // output tax (Steuerschuld RC)
				u.KZ060 += ust // deductible as VST (assuming full deduction right)
			}
		case "VST_20", "VST_10":
			if richtung == "AUSGABE" {
				u.KZ060 += ust // input VAT from supplier invoices
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Zahllast = all output taxes - all input taxes
	// KZ056 (20% USt) + KZ044 (10% USt) + KZ057 (RC §19) - KZ060 (Vorsteuer gesamt)
	u.Zahllast = u.KZ056 + u.KZ044 + u.KZ057 - u.KZ060

	return u, nil
}

// ── EA Settings ───────────────────────────────────────────────────────────────

func (r *EARepository) GetSettings(ctx context.Context, eegID uuid.UUID) (*domain.EASettings, error) {
	s := &domain.EASettings{EegID: eegID}
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(ea_uva_periodentyp,'QUARTAL'), COALESCE(ea_steuernummer,''), COALESCE(ea_finanzamt,'')
		 FROM eegs WHERE id=$1`, eegID,
	).Scan(&s.UvaPeriodentyp, &s.Steuernummer, &s.Finanzamt)
	return s, err
}

func (r *EARepository) UpdateSettings(ctx context.Context, s *domain.EASettings) error {
	_, err := r.db.Exec(ctx,
		`UPDATE eegs SET ea_uva_periodentyp=$1, ea_steuernummer=$2, ea_finanzamt=$3 WHERE id=$4`,
		s.UvaPeriodentyp, nilStr(s.Steuernummer), nilStr(s.Finanzamt), s.EegID,
	)
	return err
}

// ── Bank transactions ─────────────────────────────────────────────────────────

const bankCols = `id, eeg_id, import_am, import_format, konto_iban, buchungsdatum, valutadatum,
	betrag, waehrung, verwendungszweck, auftraggeber_empfaenger, referenz,
	matched_buchung_id, match_konfidenz, match_status`

func scanBank(row interface{ Scan(...any) error }, t *domain.EABankTransaktion) error {
	return row.Scan(
		&t.ID, &t.EegID, &t.ImportAm, &t.ImportFormat, &t.KontoIBAN, &t.Buchungsdatum, &t.Valutadatum,
		&t.Betrag, &t.Waehrung, &t.Verwendungszweck, &t.AuftraggeberEmpfaenger, &t.Referenz,
		&t.MatchedBuchungID, &t.MatchKonfidenz, &t.MatchStatus,
	)
}

func (r *EARepository) InsertBankTransaktion(ctx context.Context, t *domain.EABankTransaktion) error {
	q := `INSERT INTO ea_banktransaktionen
	        (eeg_id, import_format, konto_iban, buchungsdatum, valutadatum, betrag, waehrung,
	         verwendungszweck, auftraggeber_empfaenger, referenz, matched_buchung_id, match_konfidenz, match_status)
	      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	      RETURNING id, import_am`
	return r.db.QueryRow(ctx, q,
		t.EegID, t.ImportFormat, nilStr(t.KontoIBAN), t.Buchungsdatum, t.Valutadatum, t.Betrag, t.Waehrung,
		nilStr(t.Verwendungszweck), nilStr(t.AuftraggeberEmpfaenger), nilStr(t.Referenz),
		t.MatchedBuchungID, t.MatchKonfidenz, t.MatchStatus,
	).Scan(&t.ID, &t.ImportAm)
}

func (r *EARepository) ListBankTransaktionen(ctx context.Context, eegID uuid.UUID, status string) ([]domain.EABankTransaktion, error) {
	args := []any{eegID}
	where := "eeg_id = $1"
	if status != "" && status != "alle" {
		args = append(args, status)
		where += fmt.Sprintf(" AND match_status = $%d", len(args))
	}
	rows, err := r.db.Query(ctx, `SELECT `+bankCols+` FROM ea_banktransaktionen WHERE `+where+` ORDER BY buchungsdatum DESC`, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var result []domain.EABankTransaktion
	for rows.Next() {
		var t domain.EABankTransaktion
		if err := scanBank(rows, &t); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *EARepository) SetBankMatch(ctx context.Context, transaktionID, buchungID, eegID uuid.UUID, konfidenz float64, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE ea_banktransaktionen SET matched_buchung_id=$1, match_konfidenz=$2, match_status=$3
		 WHERE id=$4 AND eeg_id=$5`,
		buchungID, konfidenz, status, transaktionID, eegID,
	)
	if err != nil {
		return err
	}
	// Set zahlung_datum on the booking from the bank transaction
	_, err = r.db.Exec(ctx,
		`UPDATE ea_buchungen SET zahlung_datum = (
		   SELECT buchungsdatum FROM ea_banktransaktionen WHERE id=$1
		 ) WHERE id=$2 AND eeg_id=$3`,
		transaktionID, buchungID, eegID,
	)
	return err
}

func (r *EARepository) SetBankStatus(ctx context.Context, transaktionID, eegID uuid.UUID, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE ea_banktransaktionen SET match_status=$1, matched_buchung_id=NULL, match_konfidenz=NULL
		 WHERE id=$2 AND eeg_id=$3`,
		status, transaktionID, eegID,
	)
	return err
}

// FindMatchCandidates returns open buchungen that are plausible matches for a bank transaction.
func (r *EARepository) FindMatchCandidates(ctx context.Context, eegID uuid.UUID, betrag float64, buchDatum time.Time) ([]domain.EABuchung, error) {
	// Look for open bookings within ±14 days with matching amount sign
	richtung := "EINNAHME"
	if betrag < 0 {
		richtung = "AUSGABE"
		betrag = -betrag
	}
	q := `SELECT ` + buchungCols + ` FROM ea_buchungen b
	      WHERE b.eeg_id=$1 AND b.richtung=$2
	        AND b.betrag_brutto BETWEEN $3 AND $4
	        AND (b.zahlung_datum IS NULL OR ABS(b.zahlung_datum - $5) <= 14)
	        AND b.deleted_at IS NULL
	      ORDER BY ABS(b.betrag_brutto - $3), b.erstellt_am DESC
	      LIMIT 5`
	rows, err := r.db.Query(ctx, q, eegID, richtung, betrag*0.999, betrag*1.001, buchDatum)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.EABuchung
	for rows.Next() {
		var b domain.EABuchung
		if err := scanBuchung(rows, &b); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

// ── Invoice import ────────────────────────────────────────────────────────────

// QueryInvoicesForImport returns invoices in the given period, ready for EA import.
//
// Booking structure depends on the EEG's Kleinunternehmerregelung status (eeg.use_vat):
//
//   Kleinunternehmer (use_vat=false):
//     Each line is booked at the BRUTTO amount (= net since there's no VAT).
//     USt-Code = KEINE for consumption; RC_20/RC_13/KEINE for generation.
//
//   USt-pflichtig (use_vat=true):
//     Consumption line: USt-Code = UST_20; betrag_brutto = consumptionNet × 1.20;
//     betrag_netto = consumptionNet (Saldenliste shows net amount on Konto 4000).
//     Generation line: unchanged (RC bookings work the same regardless of EEG VAT status).
//
// Prosumer invoices produce TWO rows: Bezug (Konto 4000) + Einspeisung (Konto 6000/6010/6015).
// Pure consumers produce one row (Bezug). Pure producers / credit notes produce one row (Einspeisung).
// Datum is COALESCE(sent_at, created_at) — i.e. Rechnungsdatum, not period_end.
func (r *EARepository) QueryInvoicesForImport(ctx context.Context, eegID uuid.UUID, von, bis *time.Time) ([]domain.EAImportPreviewRow, error) {
	args := []any{eegID}
	// Only invoices from finalized billing runs — excludes orphaned drafts and cancelled runs.
	where := "i.eeg_id = $1"
	if von != nil {
		args = append(args, *von)
		where += fmt.Sprintf(" AND COALESCE(i.sent_at, i.created_at) >= $%d", len(args))
	}
	if bis != nil {
		args = append(args, *bis)
		where += fmt.Sprintf(" AND COALESCE(i.sent_at, i.created_at) <= $%d", len(args))
	}

	q := fmt.Sprintf(`
		SELECT i.id, i.invoice_number, COALESCE(i.sent_at, i.created_at) AS datum, i.document_type,
		       COALESCE(m.name1,'') || COALESCE(' ' || NULLIF(m.name2,''),'') AS mitglied_name,
		       COALESCE(m.business_role,'privat') AS business_role,
		       i.net_amount, i.consumption_net_amount, i.generation_net_amount,
		       i.consumption_vat_pct, i.consumption_vat_amount,
		       EXISTS(SELECT 1 FROM ea_buchungen b WHERE b.eeg_id=i.eeg_id AND b.quelle_id=i.id AND b.deleted_at IS NULL) AS already_imported,
		       e.invoice_number_prefix, COALESCE(e.invoice_number_digits, 5),
		       e.credit_note_number_prefix, COALESCE(e.credit_note_number_digits, 5),
		       e.use_vat,
		       COALESCE(i.pdf_path, '') AS pdf_path
		FROM invoices i
		JOIN billing_runs br ON br.id = i.billing_run_id AND br.status = 'finalized'
		LEFT JOIN members m ON m.id = i.member_id
		JOIN eegs e ON e.id = i.eeg_id
		WHERE %s
		ORDER BY COALESCE(i.sent_at, i.created_at), i.document_type, i.invoice_number`, where)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var result []domain.EAImportPreviewRow
	for rows.Next() {
		var invoiceID uuid.UUID
		var invoiceNr *int
		var datum time.Time
		var docType, businessRole string
		var netAmt, consumptionNet, generationNet float64
		var consumptionVatPct, consumptionVatAmount float64
		var alreadyImported, eegUseVat bool
		var invPrefix, creditPrefix string
		var invDigits, creditDigits int
		var mitgliedName, pdfPath string
		if err := rows.Scan(&invoiceID, &invoiceNr, &datum, &docType,
			&mitgliedName, &businessRole,
			&netAmt, &consumptionNet, &generationNet,
			&consumptionVatPct, &consumptionVatAmount,
			&alreadyImported, &invPrefix, &invDigits, &creditPrefix, &creditDigits,
			&eegUseVat, &pdfPath); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		// Format invoice number with EEG-configured prefix and digit count
		var invoiceNrStr string
		if invoiceNr != nil {
			if docType == "credit_note" {
				invoiceNrStr = fmt.Sprintf("%s%0*d", creditPrefix, creditDigits, *invoiceNr)
			} else {
				invoiceNrStr = fmt.Sprintf("%s%0*d", invPrefix, invDigits, *invoiceNr)
			}
		}

		// Einspeisung konto/ustcode depends on member business role (unchanged for both KU and non-KU)
		einsKontoNr, einsKontoName, einsUstCode := einspeisungKonto(businessRole)

		// Bezug USt-Code:
		//   Kleinunternehmer (use_vat=false): KEINE — no output VAT on consumption charges
		//   USt-pflichtig (use_vat=true):     UST_20 — EEG charges 20% USt; betrag_netto = consumptionNet
		// Use consumption_vat_pct from the invoice as the ground truth (handles historical data correctly).
		bezugUstCode := "KEINE"
		if eegUseVat && consumptionVatPct > 0 {
			bezugUstCode = "UST_20"
		}

		// isSplit is true when the invoice has exact per-direction amounts stored.
		// After migrations 063–065 + Go backfill, all prosumer invoices satisfy this.
		// The only remaining edge case is invoices where the backfill couldn't recover
		// exact values (tariff changed after billing) — those fall through to default.
		isSplit := consumptionNet > 0 && generationNet > 0 &&
			math.Abs((consumptionNet-generationNet)-netAmt) <= 0.005

		switch {
		case docType == "credit_note" || (consumptionNet == 0 && generationNet > 0):
			// Pure producer (credit_note or invoice with zero consumption): single Einspeisung booking
			_, ust, netto := CalcUSt(generationNet, einsUstCode)
			result = append(result, domain.EAImportPreviewRow{
				InvoiceID: invoiceID, InvoiceNr: invoiceNrStr, Datum: datum,
				DocumentType: docType, BusinessRole: businessRole, MitgliedName: mitgliedName,
				Beschreibung: "Gutschrift " + invoiceNrStr + " – " + mitgliedName,
				KontoNummer: einsKontoNr, KontoName: einsKontoName, UstCode: einsUstCode,
				BetragBrutto: generationNet, BetragNetto: netto, UstBetrag: ust,
				AlreadyImported: alreadyImported, PdfPath: pdfPath,
			})

		case isSplit:
			// Prosumer: two separate bookings — Bezug and Einspeisung.
			//
			// Kleinunternehmer (KEINE): betrag_brutto = betrag_netto = consumptionNet.
			// USt-pflichtig (UST_20):   betrag_brutto = consumptionNet + consumptionVatAmount
			//                            betrag_netto  = consumptionNet  (Saldenliste shows net on 4000)
			var bezugBrutto float64
			if bezugUstCode == "UST_20" {
				// Use stored vat_amount for exact brutto (avoids rounding from vat_pct multiplication)
				bezugBrutto = consumptionNet + consumptionVatAmount
			} else {
				bezugBrutto = consumptionNet
			}
			_, bezugUst, bezugNetto := CalcUSt(bezugBrutto, bezugUstCode)
			result = append(result, domain.EAImportPreviewRow{
				InvoiceID: invoiceID, InvoiceNr: invoiceNrStr, Datum: datum,
				DocumentType: docType, BusinessRole: businessRole, MitgliedName: mitgliedName,
				SplitPart:    "bezug",
				Beschreibung: "Rechnung " + invoiceNrStr + " – " + mitgliedName + " (Bezug)",
				KontoNummer: "4000", KontoName: "Bezugsgebühren Mitglieder",
				UstCode: bezugUstCode, BetragBrutto: bezugBrutto, BetragNetto: bezugNetto, UstBetrag: bezugUst,
				AlreadyImported: alreadyImported, PdfPath: pdfPath,
			})
			_, einsUst, einsNetto := CalcUSt(generationNet, einsUstCode)
			result = append(result, domain.EAImportPreviewRow{
				InvoiceID: invoiceID, InvoiceNr: invoiceNrStr, Datum: datum,
				DocumentType: docType, BusinessRole: businessRole, MitgliedName: mitgliedName,
				SplitPart:    "einspeisung",
				Beschreibung: "Rechnung " + invoiceNrStr + " – " + mitgliedName + " (Einspeisung)",
				KontoNummer: einsKontoNr, KontoName: einsKontoName,
				UstCode: einsUstCode, BetragBrutto: generationNet, BetragNetto: einsNetto, UstBetrag: einsUst,
				AlreadyImported: alreadyImported,
			})

		default:
			// Pure consumer, or prosumer whose backfill couldn't recover exact split amounts.
			// For pure consumers: single Bezug booking.
			// For un-backfillable prosumers: book net_amount as Bezug (conservative fallback).
			amount := consumptionNet
			if consumptionNet == 0 || (consumptionNet > 0 && generationNet > 0) {
				// prosumer fallback: use net_amount (may be negative → still book as Bezug with actual sign)
				amount = netAmt
			}
			var brutto float64
			if bezugUstCode == "UST_20" {
				brutto = amount + consumptionVatAmount
			} else {
				brutto = amount
			}
			if brutto < 0 {
				brutto = -brutto
			}
			_, ust, netto := CalcUSt(brutto, bezugUstCode)
			result = append(result, domain.EAImportPreviewRow{
				InvoiceID: invoiceID, InvoiceNr: invoiceNrStr, Datum: datum,
				DocumentType: docType, BusinessRole: businessRole, MitgliedName: mitgliedName,
				Beschreibung: "Rechnung " + invoiceNrStr + " – " + mitgliedName,
				KontoNummer: "4000", KontoName: "Bezugsgebühren Mitglieder",
				UstCode: bezugUstCode, BetragBrutto: brutto, BetragNetto: netto, UstBetrag: ust,
				AlreadyImported: alreadyImported, PdfPath: pdfPath,
			})
		}
	}
	return result, rows.Err()
}

// einspeisungKonto returns the EA account number, name, and USt code for a generation credit
// based on the member's business role.
func einspeisungKonto(businessRole string) (nummer, name, ustCode string) {
	switch businessRole {
	case "landwirt_pauschaliert":
		return "6015", "Einspeisevergütungen LuF pauschaliert (RC 13%)", "RC_13"
	case "unternehmen", "gemeinde_bga":
		return "6010", "Einspeisevergütungen Unternehmen (RC 20%)", "RC_20"
	default:
		return "6000", "Einspeisevergütungen Privatpersonen", "KEINE"
	}
}

// ── Changelog (BAO §131) ──────────────────────────────────────────────────────

type ChangelogFilter struct {
	Von       time.Time
	Bis       time.Time
	ChangedBy string
	Operation string
	Limit     int
	Offset    int
}

// GetBuchungChangelog returns the full audit trail for one booking.
func (r *EARepository) GetBuchungChangelog(ctx context.Context, buchungID, eegID uuid.UUID) ([]domain.EABuchungChangelog, error) {
	// Verify the booking belongs to this EEG (including soft-deleted)
	var exists bool
	if err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM ea_buchungen WHERE id=$1 AND eeg_id=$2)`, buchungID, eegID,
	).Scan(&exists); err != nil || !exists {
		return nil, fmt.Errorf("buchung not found")
	}

	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.buchung_id, c.operation, c.changed_at, c.changed_by,
		       c.old_values, c.new_values, COALESCE(c.reason,''),
		       COALESCE(b.buchungsnr,''), COALESCE(b.beschreibung,'')
		FROM ea_buchungen_changelog c
		JOIN ea_buchungen b ON b.id = c.buchung_id
		WHERE c.buchung_id = $1
		ORDER BY c.changed_at ASC`, buchungID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	return scanChangelogRows(rows)
}

// ListChangelog returns the EEG-wide audit trail with optional filters.
func (r *EARepository) ListChangelog(ctx context.Context, eegID uuid.UUID, f ChangelogFilter) ([]domain.EABuchungChangelog, error) {
	args := []any{eegID}
	where := []string{"b.eeg_id = $1"}

	if !f.Von.IsZero() {
		args = append(args, f.Von)
		where = append(where, fmt.Sprintf("c.changed_at >= $%d", len(args)))
	}
	if !f.Bis.IsZero() {
		args = append(args, f.Bis)
		where = append(where, fmt.Sprintf("c.changed_at <= $%d", len(args)))
	}
	if f.ChangedBy != "" {
		args = append(args, f.ChangedBy)
		where = append(where, fmt.Sprintf("c.changed_by = $%d", len(args)))
	}
	if f.Operation != "" {
		args = append(args, f.Operation)
		where = append(where, fmt.Sprintf("c.operation = $%d", len(args)))
	}

	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	q := fmt.Sprintf(`
		SELECT c.id, c.buchung_id, c.operation, c.changed_at, c.changed_by,
		       c.old_values, c.new_values, COALESCE(c.reason,''),
		       COALESCE(b.buchungsnr,''), COALESCE(b.beschreibung,'')
		FROM ea_buchungen_changelog c
		JOIN ea_buchungen b ON b.id = c.buchung_id
		WHERE %s
		ORDER BY c.changed_at DESC
		LIMIT %d OFFSET %d`, strings.Join(where, " AND "), limit, offset)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	return scanChangelogRows(rows)
}

func scanChangelogRows(rows pgx.Rows) ([]domain.EABuchungChangelog, error) {
	var result []domain.EABuchungChangelog
	for rows.Next() {
		var c domain.EABuchungChangelog
		if err := rows.Scan(
			&c.ID, &c.BuchungID, &c.Operation, &c.ChangedAt, &c.ChangedBy,
			&c.OldValues, &c.NewValues, &c.Reason,
			&c.Buchungsnr, &c.Beschreibung,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func nilStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func uvaPeriodeLabel(u domain.EAUVAPeriode) string {
	if u.Periodentyp == "QUARTAL" {
		quarters := []string{"", "Q1 (Jän–Mär)", "Q2 (Apr–Jun)", "Q3 (Jul–Sep)", "Q4 (Okt–Dez)"}
		if u.PeriodeNr >= 1 && u.PeriodeNr <= 4 {
			return fmt.Sprintf("%d / %s", u.Jahr, quarters[u.PeriodeNr])
		}
	}
	months := []string{"", "Jänner", "Februar", "März", "April", "Mai", "Juni",
		"Juli", "August", "September", "Oktober", "November", "Dezember"}
	if u.PeriodeNr >= 1 && u.PeriodeNr <= 12 {
		return fmt.Sprintf("%s %d", months[u.PeriodeNr], u.Jahr)
	}
	return fmt.Sprintf("%d/%d", u.Jahr, u.PeriodeNr)
}
