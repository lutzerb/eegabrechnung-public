package handler

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// ── Common FinanzOnline XML types ──────────────────────────────────────────────

// kzField is a FinanzOnline Kennzahl element: <KZxxx type="kz">decimal</KZxxx>
type kzField struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// kzf creates a kzField for non-zero values.
func kzf(v float64) kzField { return kzField{Type: "kz", Value: fmt.Sprintf("%.2f", v)} }

// kzPtr returns *kzField for non-zero v, nil for zero (so omitempty works).
func kzPtr(v float64) *kzField {
	if v == 0 {
		return nil
	}
	f := kzf(v)
	return &f
}

type datumField struct {
	Type  string `xml:"type,attr"` // always "datum"
	Value string `xml:",chardata"` // YYYY-MM-DD
}

type uhrzeitField struct {
	Type  string `xml:"type,attr"` // always "uhrzeit"
	Value string `xml:",chardata"` // HH:MM:SS
}

type jahrmonatField struct {
	Type  string `xml:"type,attr"` // always "jahrmonat"
	Value string `xml:",chardata"` // YYYY-MM
}

// infoDaten is the INFO_DATEN envelope header block.
type infoDaten struct {
	ArtIdentBegriff   string       `xml:"ART_IDENTIFIKATIONSBEGRIFF"` // always "FASTNR"
	IdentBegriff      string       `xml:"IDENTIFIKATIONSBEGRIFF"`     // 9-digit Steuernummer
	PaketNr           int          `xml:"PAKET_NR"`
	DatumErstellung   datumField   `xml:"DATUM_ERSTELLUNG"`
	UhrzeitErstellung uhrzeitField `xml:"UHRZEIT_ERSTELLUNG"`
	AnzahlErkl        int          `xml:"ANZAHL_ERKLAERUNGEN"`
}

func makeInfoDaten(fastnr string) infoDaten {
	now := time.Now()
	return infoDaten{
		ArtIdentBegriff:   "FASTNR",
		IdentBegriff:      fastnr,
		PaketNr:           1,
		DatumErstellung:   datumField{Type: "datum", Value: now.Format("2006-01-02")},
		UhrzeitErstellung: uhrzeitField{Type: "uhrzeit", Value: now.Format("15:04:05")},
		AnzahlErkl:        1,
	}
}

// normFastNr strips non-digits and ensures exactly 9 digits.
// Returns empty string if the result is invalid (< 9 digits after stripping).
func normFastNr(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := b.String()
	if len(d) < 9 {
		// Pad with leading zeros (best effort — user should configure valid Steuernummer)
		d = fmt.Sprintf("%09s", d)
	} else if len(d) > 9 {
		d = d[:9]
	}
	return d
}

// marshalFON marshals v to indented XML with the ISO-8859-1 declaration required by BMF.
// All content we generate is ASCII-safe so the declaration is cosmetically correct.
func marshalFON(v any) ([]byte, error) {
	b, err := xml.MarshalIndent(v, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	hdr := []byte(`<?xml version="1.0" encoding="iso-8859-1"?>` + "\n")
	return append(hdr, b...), nil
}

// ── U30 — Umsatzsteuervoranmeldung ────────────────────────────────────────────
// Schema: BMF_ERKLAERUNGS_UEBERMITTLUNG_U30_01_2022.xsd (no namespace)
//
// Structure:
//   ERKLAERUNGS_UEBERMITTLUNG
//     INFO_DATEN
//     ERKLAERUNG art="U30"
//       SATZNR
//       ALLGEMEINE_DATEN (ANBRINGEN, ZRVON, ZRBIS, FASTNR)
//       LIEFERUNGEN_LEISTUNGEN_EIGENVERBRAUCH
//         KZ000 (required, kznull)
//         VERSTEUERT? (KZ022, KZ029, KZ044, KZ056, KZ057, ...)
//       VORSTEUER? (KZ060, KZ090, ...)

type u30Root struct {
	XMLName    xml.Name        `xml:"ERKLAERUNGS_UEBERMITTLUNG"`
	InfoDaten  infoDaten       `xml:"INFO_DATEN"`
	Erklaerung u30Erklaerung   `xml:"ERKLAERUNG"`
}

type u30AllgDaten struct {
	Anbringen string         `xml:"ANBRINGEN"` // "U30"
	ZrVon     jahrmonatField `xml:"ZRVON"`
	ZrBis     jahrmonatField `xml:"ZRBIS"`
	FastNr    string         `xml:"FASTNR"`
}

// u30Versteuert — all KZ in schema sequence order (xs:sequence is strict).
type u30Versteuert struct {
	KZ022 *kzField `xml:"KZ022,omitempty"`
	KZ029 *kzField `xml:"KZ029,omitempty"`
	KZ056 *kzField `xml:"KZ056,omitempty"`
	KZ057 *kzField `xml:"KZ057,omitempty"`
	KZ044 *kzField `xml:"KZ044,omitempty"`
}

// u30Lieferungen — KZ000 is required (kznull allows 0), VERSTEUERT optional.
type u30Lieferungen struct {
	KZ000      kzField        `xml:"KZ000"`
	Versteuert *u30Versteuert `xml:"VERSTEUERT,omitempty"`
}

// u30Vorsteuer — sequence per BMF XSD VORSTEUER element.
// KZ060 (pos 1), KZ083 (pos 9), KZ065 (pos 10), KZ066 (pos 11), KZ090 (pos 19).
type u30Vorsteuer struct {
	KZ060 *kzField `xml:"KZ060,omitempty"` // Gesamtbetrag abziehbare Vorsteuern
	KZ083 *kzField `xml:"KZ083,omitempty"` // Vorsteuern aus ig. Dreiecksgeschäften
	KZ065 *kzField `xml:"KZ065,omitempty"` // Vorsteuern aus ig. Erwerben
	KZ066 *kzField `xml:"KZ066,omitempty"` // Vorsteuern für Leistungen gem. § 19 Abs. 1
	KZ090 *kzField `xml:"KZ090,omitempty"` // Vorauszahlung (pos) / Überschuss (neg)
}

type u30Erklaerung struct {
	Art         string         `xml:"art,attr"` // "U30"
	SatzNr      int            `xml:"SATZNR"`
	AlgDaten    u30AllgDaten   `xml:"ALLGEMEINE_DATEN"`
	Lieferungen u30Lieferungen `xml:"LIEFERUNGEN_LEISTUNGEN_EIGENVERBRAUCH"`
	Vorsteuer   *u30Vorsteuer  `xml:"VORSTEUER,omitempty"`
}

// EAUVAFinanzOnlineXML generates a FinanzOnline-compliant U30 (UVA) XML file.
// The schema has no namespace; encoding is ISO-8859-1.
func EAUVAFinanzOnlineXML(u *domain.EAUVAPeriode, settings *domain.EASettings) ([]byte, error) {
	fastnr := ""
	if settings != nil {
		fastnr = normFastNr(settings.Steuernummer)
	}

	// Period range as YYYY-MM
	von := u.DatumVon.Format("2006-01")
	bis := u.DatumBis.Format("2006-01")

	// Build VERSTEUERT section (only if there are taxable supplies or RC)
	var versteuert *u30Versteuert
	if u.KZ022 != 0 || u.KZ029 != 0 || u.KZ056 != 0 || u.KZ057 != 0 || u.KZ044 != 0 {
		versteuert = &u30Versteuert{
			KZ022: kzPtr(roundCent(u.KZ022)),
			KZ029: kzPtr(roundCent(u.KZ029)),
			KZ056: kzPtr(roundCent(u.KZ056)),
			KZ057: kzPtr(roundCent(u.KZ057)),
			KZ044: kzPtr(roundCent(u.KZ044)),
		}
	}

	// Build VORSTEUER section
	var vorsteuer *u30Vorsteuer
	zahllast := roundCent(u.Zahllast)
	if u.KZ060 != 0 || u.KZ083 != 0 || u.KZ065 != 0 || u.KZ066 != 0 || zahllast != 0 {
		vorsteuer = &u30Vorsteuer{
			KZ060: kzPtr(roundCent(u.KZ060)),
			KZ083: kzPtr(roundCent(u.KZ083)),
			KZ065: kzPtr(roundCent(u.KZ065)),
			KZ066: kzPtr(roundCent(u.KZ066)),
			KZ090: kzPtr(zahllast),
		}
	}

	doc := u30Root{
		InfoDaten: makeInfoDaten(fastnr),
		Erklaerung: u30Erklaerung{
			Art:    "U30",
			SatzNr: 1,
			AlgDaten: u30AllgDaten{
				Anbringen: "U30",
				ZrVon:     jahrmonatField{Type: "jahrmonat", Value: von},
				ZrBis:     jahrmonatField{Type: "jahrmonat", Value: bis},
				FastNr:    fastnr,
			},
			Lieferungen: u30Lieferungen{
				KZ000:      kzf(roundCent(u.KZ000)),
				Versteuert: versteuert,
			},
			Vorsteuer: vorsteuer,
		},
	}
	return marshalFON(doc)
}

// ── U1 — Umsatzsteuerjahreserklärung ──────────────────────────────────────────
// Schema: BMF_XSD_Jahreserklaerungen_2025.xsd (no namespace)
//
// Structure:
//   ERKLAERUNGS_UEBERMITTLUNG
//     INFO_DATEN
//     JAHRESERKLAERUNG art="JAHR_ERKL"
//       ERKLAERUNG art="U1"
//         SATZNR
//         ALLGEMEINE_DATEN (ANBRINGEN="U1", ZR=year, FASTNR)
//         LIEFERUNGEN_LEISTUNGEN_EIGENVERBRAUCH (KZ000, VERSTEUERT?)
//         VORSTEUER? (KZ060, KZ090)

type jahrRoot struct {
	XMLName          xml.Name          `xml:"ERKLAERUNGS_UEBERMITTLUNG"`
	InfoDaten        infoDaten         `xml:"INFO_DATEN"`
	Jahreserklaerung jahreserklaerung  `xml:"JAHRESERKLAERUNG"`
}

type jahreserklaerung struct {
	Art        string       `xml:"art,attr"` // always "JAHR_ERKL"
	Erklaerung jahrErklaerung `xml:"ERKLAERUNG"`
}

type jahrAllgDaten struct {
	Anbringen string `xml:"ANBRINGEN"` // "U1" or "K1"
	ZR        int    `xml:"ZR"`        // xs:gYear — just the 4-digit year
	FastNr    string `xml:"FASTNR"`
}

// u1Erklaerung extends the base declaration with USt-specific sections.
type u1Erklaerung struct {
	Art         string         `xml:"art,attr"` // "U1"
	SatzNr      int            `xml:"SATZNR"`
	AlgDaten    jahrAllgDaten  `xml:"ALLGEMEINE_DATEN"`
	Lieferungen u30Lieferungen `xml:"LIEFERUNGEN_LEISTUNGEN_EIGENVERBRAUCH"`
	Vorsteuer   *u30Vorsteuer  `xml:"VORSTEUER,omitempty"`
}

// jahrErklaerung is a union type (U1 or K1) — we embed the right one via interface wrapping.
// Since Go xml doesn't support xs:choice natively, we embed all optional sections as pointers.
type jahrErklaerung struct {
	Art         string          `xml:"art,attr"`
	SatzNr      int             `xml:"SATZNR"`
	AlgDaten    *jahrAllgDaten  `xml:"ALLGEMEINE_DATEN,omitempty"`
	// U1 sections
	Lieferungen *u30Lieferungen `xml:"LIEFERUNGEN_LEISTUNGEN_EIGENVERBRAUCH,omitempty"`
	Vorsteuer   *u30Vorsteuer   `xml:"VORSTEUER,omitempty"`
	// K1 sections
	GuV         *k1GuV          `xml:"GEWINN_VERLUSTRECHNUNG,omitempty"`
	Aufwendungen *k1Aufwendungen `xml:"AUFWENDUNGEN,omitempty"`
	Einkuenfte  *k1Einkuenfte   `xml:"EINKUENFTE_GEWERBEBETRIEB,omitempty"`
}

// EAU1FinanzOnlineXML generates a FinanzOnline-compliant U1 (annual VAT) XML.
// It uses the live annual Kennzahlen computed from all buchungen for the year
// (passed in as a computed EAUVAPeriode, not the stored UVA period values).
func EAU1FinanzOnlineXML(annual *domain.EAUVAPeriode, settings *domain.EASettings, jahr int) ([]byte, error) {
	fastnr := ""
	if settings != nil {
		fastnr = normFastNr(settings.Steuernummer)
	}

	var versteuert *u30Versteuert
	if annual.KZ022 != 0 || annual.KZ029 != 0 || annual.KZ056 != 0 || annual.KZ057 != 0 || annual.KZ044 != 0 {
		versteuert = &u30Versteuert{
			KZ022: kzPtr(roundCent(annual.KZ022)),
			KZ029: kzPtr(roundCent(annual.KZ029)),
			KZ056: kzPtr(roundCent(annual.KZ056)),
			KZ057: kzPtr(roundCent(annual.KZ057)),
			KZ044: kzPtr(roundCent(annual.KZ044)),
		}
	}

	zahllast := roundCent(annual.Zahllast)
	var vorsteuer *u30Vorsteuer
	if annual.KZ060 != 0 || annual.KZ083 != 0 || annual.KZ065 != 0 || annual.KZ066 != 0 || zahllast != 0 {
		vorsteuer = &u30Vorsteuer{
			KZ060: kzPtr(roundCent(annual.KZ060)),
			KZ083: kzPtr(roundCent(annual.KZ083)),
			KZ065: kzPtr(roundCent(annual.KZ065)),
			KZ066: kzPtr(roundCent(annual.KZ066)),
			KZ090: kzPtr(zahllast),
		}
	}

	lieferungen := &u30Lieferungen{
		KZ000:      kzf(roundCent(annual.KZ000)),
		Versteuert: versteuert,
	}

	doc := jahrRoot{
		InfoDaten: makeInfoDaten(fastnr),
		Jahreserklaerung: jahreserklaerung{
			Art: "JAHR_ERKL",
			Erklaerung: jahrErklaerung{
				Art:    "U1",
				SatzNr: 1,
				AlgDaten: &jahrAllgDaten{
					Anbringen: "U1",
					ZR:        jahr,
					FastNr:    fastnr,
				},
				Lieferungen: lieferungen,
				Vorsteuer:   vorsteuer,
			},
		},
	}
	return marshalFON(doc)
}

// ── K1 — Körperschaftsteuererklärung ──────────────────────────────────────────
// Schema: BMF_XSD_Jahreserklaerungen_2025.xsd (no namespace)
//
// Structure per official BMF K1 form (2025):
//   ERKLAERUNGS_UEBERMITTLUNG
//     INFO_DATEN
//     JAHRESERKLAERUNG art="JAHR_ERKL"
//       ERKLAERUNG art="K1"
//         SATZNR
//         ALLGEMEINE_DATEN (ANBRINGEN="K1", ZR=year, FASTNR)
//         GEWINN_VERLUSTRECHNUNG  — Punkt 2 Erträge
//           KZ9040  Umsatzerlöse (Waren-/Leistungserlöse)
//           KZ9060  Anlagenerlöse
//           KZ9070  Aktivierte Eigenleistungen
//           KZ9080  Bestandsveränderungen
//           KZ9090  Übrige Erträge (Saldo)
//         AUFWENDUNGEN  — Punkt 2 Aufwendungen
//           KZ9100  Waren, Rohstoffe, Hilfsstoffe
//           KZ9110  Fremdpersonal und Fremdleistungen
//           KZ9120  Personalaufwand (eigenes Personal)
//           KZ9130  AfA Anlagevermögen
//           KZ9140  Abschreibungen Umlaufvermögen / Forderungswertbericht.
//           KZ9150  Instandhaltungen Gebäude
//           KZ9160  Reise- und Fahrtspesen
//           KZ9170  Tatsächliche Kfz-Kosten
//           KZ9180  Miet- und Pachtaufwand, Leasing
//           KZ9190  Provisionen, Lizenzgebühren
//           KZ9200  Werbe- und Repräsentationsaufwand
//           KZ9210  Buchwert abgegangener Anlagen
//           KZ9220  Zinsen und ähnliche Aufwendungen
//           KZ9230  Übrige Aufwendungen, Kapitalveränderungen (Saldo)
//         EINKUENFTE_GEWERBEBETRIEB  — Punkt 8 (anzurechnende Steuern, leer für typische EEG)

// k1GuV — GEWINN_VERLUSTRECHNUNG section (xs:sequence order per BMF XSD, Punkt 2 Erträge).
type k1GuV struct {
	KZ9040 *kzField `xml:"KZ9040,omitempty"` // Umsatzerlöse (Waren-/Leistungserlöse)
	KZ9060 *kzField `xml:"KZ9060,omitempty"` // Anlagenerlöse
	KZ9070 *kzField `xml:"KZ9070,omitempty"` // Aktivierte Eigenleistungen
	KZ9080 *kzField `xml:"KZ9080,omitempty"` // Bestandsveränderungen
	KZ9090 *kzField `xml:"KZ9090,omitempty"` // Übrige Erträge (Saldo)
}

// k1Aufwendungen — AUFWENDUNGEN section (xs:sequence order per BMF XSD, Punkt 2 Aufwendungen).
type k1Aufwendungen struct {
	KZ9100 *kzField `xml:"KZ9100,omitempty"` // Waren, Rohstoffe, Hilfsstoffe (Einspeisevergütungen)
	KZ9110 *kzField `xml:"KZ9110,omitempty"` // Fremdpersonal und Fremdleistungen
	KZ9120 *kzField `xml:"KZ9120,omitempty"` // Personalaufwand (eigenes Personal)
	KZ9130 *kzField `xml:"KZ9130,omitempty"` // AfA Anlagevermögen
	KZ9140 *kzField `xml:"KZ9140,omitempty"` // Abschreibungen Umlaufvermögen / Forderungswertberichtigungen
	KZ9150 *kzField `xml:"KZ9150,omitempty"` // Instandhaltungen Gebäude
	KZ9160 *kzField `xml:"KZ9160,omitempty"` // Reise- und Fahrtspesen, Kilometergeld, Diäten
	KZ9170 *kzField `xml:"KZ9170,omitempty"` // Tatsächliche Kfz-Kosten (ohne AfA, Leasing)
	KZ9180 *kzField `xml:"KZ9180,omitempty"` // Miet- und Pachtaufwand, Leasing
	KZ9190 *kzField `xml:"KZ9190,omitempty"` // Provisionen an Dritte, Lizenzgebühren
	KZ9200 *kzField `xml:"KZ9200,omitempty"` // Werbe- und Repräsentationsaufwand
	KZ9210 *kzField `xml:"KZ9210,omitempty"` // Buchwert abgegangener Anlagen
	KZ9220 *kzField `xml:"KZ9220,omitempty"` // Zinsen und ähnliche Aufwendungen (Bankspesen)
	KZ9230 *kzField `xml:"KZ9230,omitempty"` // Übrige Aufwendungen, Kapitalveränderungen (Saldo)
}

// k1Einkuenfte — EINKUENFTE_GEWERBEBETRIEB section (Punkt 8: anzurechnende Steuern).
// KZ645 = anrechenbare inländische KESt; KZ292 = anrechenbare Abzugsteuer §107 EStG.
// Leave empty for typical EEG associations without capital income tax credits.
type k1Einkuenfte struct {
	KZ645 *kzField `xml:"KZ645,omitempty"` // Anrechenbare inländische Kapitalertragsteuer
	KZ292 *kzField `xml:"KZ292,omitempty"` // Anrechenbare Abzugsteuer (§107 EStG)
}

// EAK1Summary generates a FinanzOnline-compliant K1 (corporate tax) XML.
// KZ assignments follow the official BMF K1 2025 form exactly.
func EAK1Summary(ja *domain.EAJahresabschluss, settings *domain.EASettings) ([]byte, error) {
	fastnr := ""
	if settings != nil {
		fastnr = normFastNr(settings.Steuernummer)
	}

	// Aggregate Einnahmen by k1_kz.
	einByKZ := map[string]float64{}
	for _, e := range ja.Einnahmen {
		kz := e.K1KZ
		if kz == "" {
			kz = "9040" // fallback: Umsatzerlöse
		}
		einByKZ[kz] += e.Einnahmen
	}
	// Aggregate Ausgaben by k1_kz.
	aussByKZ := map[string]float64{}
	for _, e := range ja.Ausgaben {
		kz := e.K1KZ
		if kz == "" {
			kz = "9230" // fallback: Übrige Aufwendungen
		}
		aussByKZ[kz] += e.Ausgaben
	}

	rc := func(kz string, m map[string]float64) *kzField { return kzPtr(roundCent(m[kz])) }

	guv := &k1GuV{
		KZ9040: rc("9040", einByKZ),
		KZ9060: rc("9060", einByKZ),
		KZ9070: rc("9070", einByKZ),
		KZ9080: rc("9080", einByKZ),
		KZ9090: rc("9090", einByKZ),
	}

	aufwendungen := &k1Aufwendungen{
		KZ9100: rc("9100", aussByKZ),
		KZ9110: rc("9110", aussByKZ),
		KZ9120: rc("9120", aussByKZ),
		KZ9130: rc("9130", aussByKZ),
		KZ9140: rc("9140", aussByKZ),
		KZ9150: rc("9150", aussByKZ),
		KZ9160: rc("9160", aussByKZ),
		KZ9170: rc("9170", aussByKZ),
		KZ9180: rc("9180", aussByKZ),
		KZ9190: rc("9190", aussByKZ),
		KZ9200: rc("9200", aussByKZ),
		KZ9210: rc("9210", aussByKZ),
		KZ9220: rc("9220", aussByKZ),
		KZ9230: rc("9230", aussByKZ),
	}

	// EINKUENFTE_GEWERBEBETRIEB (Punkt 8) is for anzurechnende Steuern —
	// not the KöSt itself. Leave nil for typical EEG associations.
	var einkuenfte *k1Einkuenfte

	doc := jahrRoot{
		InfoDaten: makeInfoDaten(fastnr),
		Jahreserklaerung: jahreserklaerung{
			Art: "JAHR_ERKL",
			Erklaerung: jahrErklaerung{
				Art:    "K1",
				SatzNr: 1,
				AlgDaten: &jahrAllgDaten{
					Anbringen: "K1",
					ZR:        ja.Jahr,
					FastNr:    fastnr,
				},
				GuV:          guv,
				Aufwendungen: aufwendungen,
				Einkuenfte:   einkuenfte,
			},
		},
	}
	return marshalFON(doc)
}

// ── K2 — Körperschaftsteuererklärung für beschränkt steuerpflichtige Körperschaften ────────
// Schema: BMF_XSD_Jahreserklaerungen_2025.xsd (no namespace)
//
// Used by EEGs organised as Vereine (§5 KStG, beschränkt steuerpflichtig).
// The "K2a" beilage is not a separate FinanzOnline form — it maps to
// BETRIEBLICHE_EINKUNFTSARTEN_K2 > EINKUENFTE_GEWERBEBETRIEB_K2 within K2.
//
// Structure:
//   ERKLAERUNGS_UEBERMITTLUNG
//     INFO_DATEN
//     JAHRESERKLAERUNG art="JAHR_ERKL"
//       ERKLAERUNG art="K2"
//         SATZNR
//         ALLGEMEINE_DATEN_K2 (ANBRINGEN="K2", ZR=year, FASTNR)
//         BETRIEBLICHE_EINKUNFTSARTEN_K2
//           EINKUENFTE_GEWERBEBETRIEB_K2
//             EINZELUNTERNEHMER_K2
//               ALLGEMEIN_K2 (WIBETR="J" — wirtschaftlicher Geschäftsbetrieb)
//               ERTRAEGE_EINNAHMEN   — same KZ9040–KZ9090 as K1 GEWINN_VERLUSTRECHNUNG
//               AUFWENDUNGEN_AUSGABEN — same KZ9100–KZ9230 as K1 AUFWENDUNGEN
//               GEWINN_VERLUST
//                 KZ9280  Überschuss (+) / Fehlbetrag (-)
//
// KZ assignments reuse the k1_kz column — K2 uses identical KZ numbers for the same positions.

// k2AllgDaten — ALLGEMEINE_DATEN_K2 header (K2-specific element name, cf. ALLGEMEINE_DATEN for K1).
type k2AllgDaten struct {
	Anbringen string `xml:"ANBRINGEN"` // "K2"
	ZR        int    `xml:"ZR"`
	FastNr    string `xml:"FASTNR"`
}

// k2AllgemeinInner — ALLGEMEIN_K2 block inside EINZELUNTERNEHMER_K2.
// WIBETR=J marks a wirtschaftlicher Geschäftsbetrieb (Verein EEG).
type k2AllgemeinInner struct {
	WiBetр string `xml:"WIBETR,omitempty"` // "J"
}

// k2ErtraegeEinnahmen — ERTRAEGE_EINNAHMEN (xs:sequence per XSD).
type k2ErtraegeEinnahmen struct {
	KZ9040 *kzField `xml:"KZ9040,omitempty"` // Umsatzerlöse (Bezugsgebühren, Grundgebühren)
	KZ9060 *kzField `xml:"KZ9060,omitempty"` // Anlagenerlöse
	KZ9070 *kzField `xml:"KZ9070,omitempty"` // Aktivierte Eigenleistungen
	KZ9080 *kzField `xml:"KZ9080,omitempty"` // Bestandsveränderungen
	KZ9090 *kzField `xml:"KZ9090,omitempty"` // Übrige Erträge (Saldo)
}

// k2AufwendungenAusgaben — AUFWENDUNGEN_AUSGABEN (xs:sequence per XSD).
type k2AufwendungenAusgaben struct {
	KZ9100 *kzField `xml:"KZ9100,omitempty"` // Waren, Rohstoffe (Einspeisevergütungen)
	KZ9110 *kzField `xml:"KZ9110,omitempty"` // Fremdpersonal und Fremdleistungen
	KZ9120 *kzField `xml:"KZ9120,omitempty"` // Personalaufwand (eigenes Personal)
	KZ9130 *kzField `xml:"KZ9130,omitempty"` // AfA Anlagevermögen
	KZ9140 *kzField `xml:"KZ9140,omitempty"` // Abschreibungen Umlaufvermögen
	KZ9150 *kzField `xml:"KZ9150,omitempty"` // Instandhaltungen Gebäude
	KZ9160 *kzField `xml:"KZ9160,omitempty"` // Reise- und Fahrtspesen
	KZ9170 *kzField `xml:"KZ9170,omitempty"` // Tatsächliche Kfz-Kosten
	KZ9180 *kzField `xml:"KZ9180,omitempty"` // Miet- und Pachtaufwand, Leasing
	KZ9190 *kzField `xml:"KZ9190,omitempty"` // Provisionen, Lizenzgebühren
	KZ9200 *kzField `xml:"KZ9200,omitempty"` // Werbe- und Repräsentationsaufwand
	KZ9210 *kzField `xml:"KZ9210,omitempty"` // Buchwert abgegangener Anlagen
	KZ9220 *kzField `xml:"KZ9220,omitempty"` // Zinsen und ähnliche Aufwendungen (Bankspesen)
	KZ9230 *kzField `xml:"KZ9230,omitempty"` // Übrige Aufwendungen (Netzkosten, IT, Steuerberatung)
}

// k2GewinnVerlust — GEWINN_VERLUST (Punkt 8 — Überschuss/Fehlbetrag des Betriebs).
type k2GewinnVerlust struct {
	KZ9280 *kzField `xml:"KZ9280,omitempty"` // Gesamtbetrag Einkünfte aus wirtsch. Geschäftsbetrieb
}

type k2EinzelunternehmerK2 struct {
	AllgemeinK2          k2AllgemeinInner        `xml:"ALLGEMEIN_K2"`
	ErtraegeEinnahmen    *k2ErtraegeEinnahmen    `xml:"ERTRAEGE_EINNAHMEN,omitempty"`
	AufwendungenAusgaben *k2AufwendungenAusgaben `xml:"AUFWENDUNGEN_AUSGABEN,omitempty"`
	GewinnVerlust        *k2GewinnVerlust        `xml:"GEWINN_VERLUST,omitempty"`
}

type k2EinkuenfteGewerbe struct {
	Einzelunternehmer k2EinzelunternehmerK2 `xml:"EINZELUNTERNEHMER_K2"`
}

type k2BetrieblicheEinkuenfte struct {
	EinkuenfteGewerbebetrieb k2EinkuenfteGewerbe `xml:"EINKUENFTE_GEWERBEBETRIEB_K2"`
}

type k2Erklaerung struct {
	Art                      string                   `xml:"art,attr"` // "K2"
	SatzNr                   int                      `xml:"SATZNR"`
	AllgemeineDatenK2        k2AllgDaten              `xml:"ALLGEMEINE_DATEN_K2"`
	BetrieblicheEinkuenfte   k2BetrieblicheEinkuenfte `xml:"BETRIEBLICHE_EINKUNFTSARTEN_K2"`
}

type k2Jahreserklaerung struct {
	Art        string       `xml:"art,attr"` // "JAHR_ERKL"
	Erklaerung k2Erklaerung `xml:"ERKLAERUNG"`
}

type k2JahrRoot struct {
	XMLName          xml.Name           `xml:"ERKLAERUNGS_UEBERMITTLUNG"`
	InfoDaten        infoDaten          `xml:"INFO_DATEN"`
	Jahreserklaerung k2Jahreserklaerung `xml:"JAHRESERKLAERUNG"`
}

// EAK2Summary generates a FinanzOnline-compliant K2 XML for Verein EEGs (§5 KStG).
// KZ assignments reuse the k1_kz column — K2 uses identical KZ9040–KZ9230 numbering.
func EAK2Summary(ja *domain.EAJahresabschluss, settings *domain.EASettings) ([]byte, error) {
	fastnr := ""
	if settings != nil {
		fastnr = normFastNr(settings.Steuernummer)
	}

	// Aggregate by k1_kz (K2 reuses the identical KZ numbers).
	einByKZ := map[string]float64{}
	for _, e := range ja.Einnahmen {
		kz := e.K1KZ
		if kz == "" {
			kz = "9040" // fallback: Umsatzerlöse
		}
		einByKZ[kz] += e.Einnahmen
	}
	aussByKZ := map[string]float64{}
	for _, e := range ja.Ausgaben {
		kz := e.K1KZ
		if kz == "" {
			kz = "9230" // fallback: Übrige Aufwendungen
		}
		aussByKZ[kz] += e.Ausgaben
	}

	rc := func(kz string, m map[string]float64) *kzField { return kzPtr(roundCent(m[kz])) }

	ertraege := &k2ErtraegeEinnahmen{
		KZ9040: rc("9040", einByKZ),
		KZ9060: rc("9060", einByKZ),
		KZ9070: rc("9070", einByKZ),
		KZ9080: rc("9080", einByKZ),
		KZ9090: rc("9090", einByKZ),
	}
	if ertraege.KZ9040 == nil && ertraege.KZ9060 == nil && ertraege.KZ9070 == nil &&
		ertraege.KZ9080 == nil && ertraege.KZ9090 == nil {
		ertraege = nil
	}

	aufwendungen := &k2AufwendungenAusgaben{
		KZ9100: rc("9100", aussByKZ),
		KZ9110: rc("9110", aussByKZ),
		KZ9120: rc("9120", aussByKZ),
		KZ9130: rc("9130", aussByKZ),
		KZ9140: rc("9140", aussByKZ),
		KZ9150: rc("9150", aussByKZ),
		KZ9160: rc("9160", aussByKZ),
		KZ9170: rc("9170", aussByKZ),
		KZ9180: rc("9180", aussByKZ),
		KZ9190: rc("9190", aussByKZ),
		KZ9200: rc("9200", aussByKZ),
		KZ9210: rc("9210", aussByKZ),
		KZ9220: rc("9220", aussByKZ),
		KZ9230: rc("9230", aussByKZ),
	}
	if aufwendungen.KZ9100 == nil && aufwendungen.KZ9110 == nil && aufwendungen.KZ9120 == nil &&
		aufwendungen.KZ9130 == nil && aufwendungen.KZ9140 == nil && aufwendungen.KZ9150 == nil &&
		aufwendungen.KZ9160 == nil && aufwendungen.KZ9170 == nil && aufwendungen.KZ9180 == nil &&
		aufwendungen.KZ9190 == nil && aufwendungen.KZ9200 == nil && aufwendungen.KZ9210 == nil &&
		aufwendungen.KZ9220 == nil && aufwendungen.KZ9230 == nil {
		aufwendungen = nil
	}

	ueberschuss := roundCent(ja.Ueberschuss)
	var gewinnVerlust *k2GewinnVerlust
	if kz9280 := kzPtr(ueberschuss); kz9280 != nil {
		gewinnVerlust = &k2GewinnVerlust{KZ9280: kz9280}
	}

	doc := k2JahrRoot{
		InfoDaten: makeInfoDaten(fastnr),
		Jahreserklaerung: k2Jahreserklaerung{
			Art: "JAHR_ERKL",
			Erklaerung: k2Erklaerung{
				Art:    "K2",
				SatzNr: 1,
				AllgemeineDatenK2: k2AllgDaten{
					Anbringen: "K2",
					ZR:        ja.Jahr,
					FastNr:    fastnr,
				},
				BetrieblicheEinkuenfte: k2BetrieblicheEinkuenfte{
					EinkuenfteGewerbebetrieb: k2EinkuenfteGewerbe{
						Einzelunternehmer: k2EinzelunternehmerK2{
							AllgemeinK2:          k2AllgemeinInner{WiBetр: "J"},
							ErtraegeEinnahmen:    ertraege,
							AufwendungenAusgaben: aufwendungen,
							GewinnVerlust:        gewinnVerlust,
						},
					},
				},
			},
		},
	}
	return marshalFON(doc)
}

func roundCent(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
