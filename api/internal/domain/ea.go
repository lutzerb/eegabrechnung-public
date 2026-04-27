package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EAKonto is an account in the E/A Kontenplan.
// Each EEG has its own chart of accounts, seeded with defaults on first access.
type EAKonto struct {
	ID             uuid.UUID `json:"id"`
	EegID          uuid.UUID `json:"eeg_id"`
	Nummer         string    `json:"nummer"`         // e.g. "6010"
	Name           string    `json:"name"`
	Typ            string    `json:"typ"`            // EINNAHME | AUSGABE | SONSTIG
	UstRelevanz    string    `json:"ust_relevanz"`   // KEINE | STEUERBAR | VST | RC
	StandardUstPct *float64  `json:"standard_ust_pct,omitempty"`
	UvaKZ          string    `json:"uva_kz,omitempty"`
	K1KZ           string    `json:"k1_kz,omitempty"` // FinanzOnline K1 Kennzahl, e.g. "9040", "9100"
	Sortierung     int       `json:"sortierung"`
	Aktiv          bool      `json:"aktiv"`
	CreatedAt      time.Time `json:"created_at"`
}

// EABuchung is a single cash-basis journal entry.
// zahlung_datum is the IST-Datum (actual payment date); nil = pending.
type EABuchung struct {
	ID             uuid.UUID  `json:"id"`
	EegID          uuid.UUID  `json:"eeg_id"`
	Geschaeftsjahr int        `json:"geschaeftsjahr"`
	Buchungsnr     string     `json:"buchungsnr,omitempty"`
	ZahlungDatum   *time.Time `json:"zahlung_datum,omitempty"`
	BelegDatum     *time.Time `json:"beleg_datum,omitempty"`
	Belegnr        string     `json:"belegnr,omitempty"`
	Beschreibung   string     `json:"beschreibung"`
	KontoID        uuid.UUID  `json:"konto_id"`
	Konto          *EAKonto   `json:"konto,omitempty"`
	Richtung       string     `json:"richtung"`      // EINNAHME | AUSGABE
	BetragBrutto   float64    `json:"betrag_brutto"` // amount actually paid/received
	// USt codes:
	//   KEINE      - not taxable or exempt
	//   UST_20/10  - outgoing VAT (EEG charges USt)
	//   VST_20/10  - input VAT (paid to supplier, embedded in price)
	//   RC_20      - Reverse Charge 20% (§2 Z 2 UStBBKV, unternehmen)
	//   RC_13      - Reverse Charge 13% (§22 UStG, landwirt_pauschaliert)
	UstCode      string     `json:"ust_code"`
	UstPct       *float64   `json:"ust_pct,omitempty"`
	UstBetrag    float64    `json:"ust_betrag"`
	BetragNetto  float64    `json:"betrag_netto"`
	Gegenseite   string     `json:"gegenseite,omitempty"`
	Quelle       string     `json:"quelle"`   // manual | eeg_rechnung | eeg_gutschrift | bankimport
	QuelleID     *uuid.UUID `json:"quelle_id,omitempty"`
	BelegID      *uuid.UUID `json:"beleg_id,omitempty"`
	Notizen      string     `json:"notizen,omitempty"`
	ErstelltVon    *uuid.UUID `json:"erstellt_von,omitempty"`
	ErstelltAm     time.Time  `json:"erstellt_am"`
	AktualisiertAm time.Time  `json:"aktualisiert_am"`
	// Soft-delete (BAO §131)
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	DeletedBy string     `json:"deleted_by,omitempty"`
	// Populated on detail view
	Belege []EABeleg `json:"belege,omitempty"`
}

// EABuchungChangelog is one audit-trail entry for a booking mutation.
// BAO §131: all changes (create/update/delete) are recorded here.
type EABuchungChangelog struct {
	ID        uuid.UUID       `json:"id"`
	BuchungID uuid.UUID       `json:"buchung_id"`
	Operation string          `json:"operation"` // create | update | delete
	ChangedAt time.Time       `json:"changed_at"`
	ChangedBy string          `json:"changed_by"` // user UUID from JWT
	OldValues json.RawMessage `json:"old_values,omitempty"`
	NewValues json.RawMessage `json:"new_values,omitempty"`
	Reason    string          `json:"reason,omitempty"`
	// Denormalized for list view
	Buchungsnr   string `json:"buchungsnr,omitempty"`
	Beschreibung string `json:"beschreibung,omitempty"`
}

// EABeleg is an uploaded receipt/document linked to a booking.
type EABeleg struct {
	ID             uuid.UUID  `json:"id"`
	EegID          uuid.UUID  `json:"eeg_id"`
	BuchungID      *uuid.UUID `json:"buchung_id,omitempty"`
	Dateiname      string     `json:"dateiname"`
	Pfad           string     `json:"pfad"`
	Groesse        *int       `json:"groesse,omitempty"`
	MimeTyp        string     `json:"mime_typ,omitempty"`
	Beschreibung   string     `json:"beschreibung,omitempty"`
	HochgeladenAm  time.Time  `json:"hochgeladen_am"`
	HochgeladenVon *uuid.UUID `json:"hochgeladen_von,omitempty"`
}

// EAUVAPeriode tracks one UVA reporting period with its Kennzahlen.
type EAUVAPeriode struct {
	ID          uuid.UUID  `json:"id"`
	EegID       uuid.UUID  `json:"eeg_id"`
	Jahr        int        `json:"jahr"`
	Periodentyp string     `json:"periodentyp"` // MONAT | QUARTAL
	PeriodeNr   int        `json:"periode_nr"`
	DatumVon    time.Time  `json:"datum_von"`
	DatumBis    time.Time  `json:"datum_bis"`
	Status      string     `json:"status"` // entwurf | eingereicht
	KZ000       float64    `json:"kz_000"`
	KZ022       float64    `json:"kz_022"`   // 20% Umsätze (Bemessungsgrundlage)
	KZ029       float64    `json:"kz_029"`   // 10% Umsätze (Bemessungsgrundlage)
	KZ044       float64    `json:"kz_044"`   // Steuer für KZ029 (10% output tax)
	KZ056       float64    `json:"kz_056"`   // Steuer für KZ022 (20% output tax)
	KZ057       float64    `json:"kz_057"`   // Steuerschuld §19 Abs. 1 (Reverse Charge domestic)
	KZ060       float64    `json:"kz_060"`   // Gesamtbetrag abziehbare Vorsteuer §12
	KZ065       float64    `json:"kz_065"`   // Vorsteuer aus igE
	KZ066       float64    `json:"kz_066"`   // Einfuhrumsatzsteuer (import VAT — rarely used)
	KZ083       float64    `json:"kz_083"`   // Vorsteuer aus ig. Dreiecksgeschäften (rarely used)
	Zahllast    float64    `json:"zahllast"`
	EingereichtAm *time.Time `json:"eingereicht_am,omitempty"`
	ErstelltAm  time.Time  `json:"erstellt_am"`
	// Computed display label
	PeriodeLabel string `json:"periode_label,omitempty"`
}

// EASaldenlisteEintrag is one row in the balance list (Saldenliste).
type EASaldenlisteEintrag struct {
	KontoID         uuid.UUID `json:"konto_id"`
	Nummer          string    `json:"nummer"`
	Name            string    `json:"name"`
	Typ             string    `json:"typ"`
	K1KZ            string    `json:"k1_kz,omitempty"` // FinanzOnline K1 Kennzahl for this account
	Einnahmen       float64   `json:"einnahmen"`
	Ausgaben        float64   `json:"ausgaben"`
	Saldo           float64   `json:"saldo"`
	AnzahlBuchungen int       `json:"anzahl_buchungen"`
}

// EAJahresabschluss is the annual income/expense statement.
type EAJahresabschluss struct {
	Jahr           int                    `json:"jahr"`
	TotalEinnahmen float64                `json:"total_einnahmen"`
	TotalAusgaben  float64                `json:"total_ausgaben"`
	Ueberschuss    float64                `json:"ueberschuss"`
	Einnahmen      []EASaldenlisteEintrag `json:"einnahmen"`
	Ausgaben       []EASaldenlisteEintrag `json:"ausgaben"`
}

// EAKontenblattEintrag is one row in an account sheet (Kontenblatt).
type EAKontenblattEintrag struct {
	EABuchung
	LaufenderSaldo float64 `json:"laufender_saldo"`
}

// EAKontenblatt is the full account transaction list for one account.
type EAKontenblatt struct {
	Konto     EAKonto                `json:"konto"`
	Eintraege []EAKontenblattEintrag `json:"eintraege"`
	Summe     float64                `json:"summe"`
}

// EASettings holds EA-specific configuration stored on the eegs table.
type EASettings struct {
	EegID           uuid.UUID `json:"eeg_id"`
	UvaPeriodentyp  string    `json:"uva_periodentyp"` // MONAT | QUARTAL
	Steuernummer    string    `json:"steuernummer,omitempty"`
	Finanzamt       string    `json:"finanzamt,omitempty"`
}

// EAImportPreviewRow is one row in the import preview.
// Prosumer invoices produce two rows — one for Bezug (consumption) and one for Einspeisung (generation).
type EAImportPreviewRow struct {
	InvoiceID       uuid.UUID `json:"invoice_id"`
	InvoiceNr       string    `json:"invoice_nr"`
	Datum           time.Time `json:"datum"`
	DocumentType    string    `json:"document_type"`
	BusinessRole    string    `json:"business_role"`
	MitgliedName    string    `json:"mitglied_name"`
	Beschreibung    string    `json:"beschreibung"`
	SplitPart       string    `json:"split_part"` // "" | "bezug" | "einspeisung"
	KontoNummer     string    `json:"konto_nummer"`
	KontoName       string    `json:"konto_name"`
	UstCode         string    `json:"ust_code"`
	BetragBrutto    float64   `json:"betrag_brutto"`
	BetragNetto     float64   `json:"betrag_netto"`
	UstBetrag       float64   `json:"ust_betrag"`
	AlreadyImported bool      `json:"already_imported"`
	PdfPath         string    `json:"-"` // internal only — not sent to frontend
}

// EAImportResult summarises the result of a batch import.
type EAImportResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Errors   []string `json:"errors,omitempty"`
}

// EABankTransaktion is an imported bank statement line.
type EABankTransaktion struct {
	ID                    uuid.UUID  `json:"id"`
	EegID                 uuid.UUID  `json:"eeg_id"`
	ImportAm              time.Time  `json:"import_am"`
	ImportFormat          string     `json:"import_format"`
	KontoIBAN             string     `json:"konto_iban,omitempty"`
	Buchungsdatum         time.Time  `json:"buchungsdatum"`
	Valutadatum           *time.Time `json:"valutadatum,omitempty"`
	Betrag                float64    `json:"betrag"` // positive = incoming
	Waehrung              string     `json:"waehrung"`
	Verwendungszweck      string     `json:"verwendungszweck,omitempty"`
	AuftraggeberEmpfaenger string    `json:"auftraggeber_empfaenger,omitempty"`
	Referenz              string     `json:"referenz,omitempty"`
	MatchedBuchungID      *uuid.UUID `json:"matched_buchung_id,omitempty"`
	MatchKonfidenz        *float64   `json:"match_konfidenz,omitempty"`
	MatchStatus           string     `json:"match_status"` // offen | auto | bestaetigt | ignoriert
	// Populated on list
	MatchedBuchung *EABuchung `json:"matched_buchung,omitempty"`
}
