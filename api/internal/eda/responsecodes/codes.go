// Package responsecodes maps EDA ResponseCodes (Customer Processes category,
// ebutilities.at/responsecodes) to their German-language meaning, and extracts
// them out of the error strings worker.go stores on eda_processes.error_msg.
//
// This is the Go-side counterpart of web/lib/eda-error-codes.ts — keep both in
// sync when new codes are added. The frontend uses its copy to annotate the
// EDA/Prozesse table; this copy is for the "[EDA Fehler]" operator email,
// which has no access to the frontend bundle.
package responsecodes

import (
	"regexp"
	"strings"
)

// Codes maps a ResponseCode string (e.g. "182") to its German meaning.
var Codes = map[string]string{
	"5":   "Datei kann nicht geöffnet werden",
	"10":  "Falsches Dateiformat",
	"12":  "Frist nicht eingehalten",
	"37":  "Stornierung nicht möglich",
	"55":  "Zählpunkt nicht dem Lieferanten zugeordnet",
	"56":  "Zählpunkt nicht gefunden",
	"57":  "Zählpunkt nicht versorgt",
	"67":  "Storno aus anderem Grund",
	"69":  "Teilnehmer E-Gemeinschaft/Anforderung/Änderung vorgemerkt",
	"70":  "Änderung/Anforderung akzeptiert",
	"71":  "Nachweisdokument fehlt",
	"72":  "Keine DA - AN/ABM notwendig",
	"73":  "Nachrichtendaten fehlen",
	"74":  "Nachweisdokument nicht akzeptiert",
	"75":  "Verbrauchsverhalten entspricht nicht dem angeforderten Profil",
	"76":  "Ungültige Anforderungsdaten",
	"77":  "Ohne Kostenübernahme Prozess nicht möglich",
	"78":  "Ablesung nicht möglich",
	"79":  "Ablehnung aus anderem Grund",
	"80":  "Kein Insolvenzverfahren bekannt",
	"81":  "Rückforderung akzeptiert",
	"82":  "Prozessdatum falsch",
	"83":  "kein VZ Prozess",
	"84":  "kein Vorleistungsmodell",
	"85":  "Betrag Null",
	"86":  "konkurrierende Prozesse",
	"87":  "Lieferant ist nicht Netzrechnungsempfänger",
	"88":  "Keine Vereinbarung vorhanden",
	"89":  "Zähler nicht erreichbar",
	"90":  "Kein Smart Meter",
	"91":  "Angefordertes Ablese-/Übertragungsintervall nicht möglich",
	"92":  "Lieferant erhält bereits angefordertes Ablese-/Übertragungsintervall",
	"93":  "Lieferant erhält bereits angefordertes Abrechnungsintervall",
	"94":  "Keine Daten im angeforderten Zeitraum vorhanden",
	"95":  "Versorgungssicherheitsgrund liegt vor",
	"96":  "Prepaymentfunktion für Lieferant bereits aktiviert",
	"97":  "Prepaymentfunktion für Lieferant bereits deaktiviert",
	"98":  "Anforderung zu weit in der Zukunft",
	"99":  "Meldung erhalten",
	"100": "Keine Ausschaltung durch Anforderer vorhanden",
	"101": "Authentifizierungsverfahren nicht zulässig",
	"102": "Keine Prepaymentanforderung vorhanden",
	"103": "Aktiver Prepaymentprozess vorhanden",
	"104": "Falsche Energierichtung",
	"105": "Änderung mit abweichendem Datum akzeptiert",
	"106": "Kombination Mess- und Übertragungsintervall nicht zulässig",
	"107": "Meldung erhalten, Änderung nicht übernommen",
	"108": "Zähler aus der Ferne nicht schaltbar",
	"109": "Keine präqualifizierte Technische Einheit",
	"150": "Verteilmodell fehlt bzw. nicht korrekt",
	"151": "NB ist zur Gemeinschafts-ID der BEG beim VEZ bereits zugeordnet",
	"152": "NB ist zur Gemeinschafts-ID der BEG beim VEZ nicht zugeordnet",
	"153": "Keine Vereinbarung vorhanden",
	"154": "Teilnehmender Berechtigter nicht identifiziert",
	"155": "Viertelstundenauslesung nicht möglich",
	"156": "ZP bereits zugeordnet",
	"157": "ZP bereits einem Betreiber zugeordnet",
	"158": "ZP ist nicht teilnahmeberechtigt",
	"159": "Zu Prozessdatum ZP inaktiv bzw. noch kein Gerät eingebaut",
	"160": "Verteilmodell entspricht nicht der Vereinbarung",
	"161": "Einspeisezählpunkt nicht vorhanden",
	"162": "ZP keinem Betreiber zugeordnet",
	"163": "Überschussbehandlung nicht erlaubt",
	"164": "Liste zur Aktivierung ungültig",
	"165": "Summe der Zuordnungen muss kleiner oder gleich 100 sein",
	"166": "Aktuellere Aktivierungsliste zum Prozessdatum vorhanden",
	"168": "Registrierung durchgeführt",
	"169": "Deregistrierung durchgeführt",
	"170": "Liste gültig",
	"171": "Liste teilweise gültig",
	"172": "Kunde hat Datenfreigabe abgelehnt",
	"173": "Kunde hat auf Datenfreigabe nicht reagiert (Timeout)",
	"174": "Angefragte Daten nicht lieferbar",
	"175": "Zustimmung erteilt",
	"176": "Zustimmung erfolgreich entzogen",
	"177": "Keine Datenfreigabe vorhanden",
	"178": "Consent existiert bereits",
	"179": "ConsentRequestID existiert bereits",
	"180": "ConsentID abgelaufen",
	"181": "Gemeinschafts-ID nicht vorhanden",
	"182": "Noch kein fernauslesbarer Zähler eingebaut",
	"183": "Summe der gemeldeten Aufteilungsschlüssel übersteigt 100%",
	"184": "Kunde hat optiert",
	"185": "Zählpunkt befindet sich nicht im Bereich der Energiegemeinschaft",
	"186": "Änderung Aufteilungsschlüssel nicht möglich",
	"187": "ConsentID und Zählpunkt passen nicht zusammen",
	"188": "Teilnahmefaktor von 100 % würde überschritten werden",
	"189": "Zählpunkt ist der Gemeinschafts-ID nicht zugeordnet",
	"190": "Referenz nicht zuordenbar",
	"191": "Marktprozess verwenden",
	"192": "Inhalt passt nicht zu Kategorie",
	"193": "Anfrage unklar",
	"194": "Anfrage nicht zulässig",
	"195": "Marktteilnehmer nicht vorhanden",
	"196": "Teilnahme-Limit wird überschritten",
	"197": "ZP nimmt nicht an einer Energiegemeinschaft teil",
	"198": "Unzureichende Granularität für Förderabwicklungsstelle (FAS)",
	"199": "Falsche Sparte",
	"200": "Ungültige anfragende Teilnehmerkennung",
	"201": "Ungültige abgefragte Teilnehmerkennung",
	"202": "Adresse nicht gefunden",
	"203": "Zustimmung wurde entzogen",
	"204": "Für ZP ist derzeit keine ausreichend stabile Kommunikation möglich",
	"205": "Adresse nicht eindeutig identifizierbar",
	"206": "Abweichung Anspruchsberechtigter zu Vertragspartner",
	"207": "Keine passende Anmeldung gefunden",
	"208": "Zählpunkt nimmt bereits an EG teil",
	"209": "Keine Viertelstundenauslesung möglich",
	"249": "Netzrechnung noch nicht bezahlt",
	"250": "Rechnung bereits ausgeglichen",
	"251": "nicht nachvollziehbar",
	"252": "Rechnung bereits storniert",
	"253": "Rechnung ist korrekt",
	"254": "Rechnung zu Zählpunkt nicht vorhanden",
	"255": "Mahnung bereits storniert",
	"256": "Mahnung bereits reklamiert",
	"257": "Mahnung zu ConversationID nicht gefunden",
	"258": "Postleitzahl passt nicht zu Zählpunkt",
	"259": "Netztopologie nicht ermittelbar",
	"260": "Mahnreklamation inkonsistent",
	"261": "Lieferant ist Netzrechnungsempfänger",
	"262": "Zählpunkt nimmt bereits an einer Flexibilitätsdienstleistung teil",
	"501": "Nachrichtendaten fehlen oder falsch",
	"502": "Zählpunkt nicht gefunden",
	"503": "Zählpunkt nicht versorgt",
	"504": "Es handelt sich um keinen Zählpunkt der Sparte Strom",
	"505": "Es handelt sich um keinen Verbrauchszählpunkt",
	"506": "Ein Rückabwicklungsprozess ist aktiv",
	"511": "Zählpunkt war versorgt",
	"512": "Ergänzungszuschuss-ID wurde bereits übermittelt",
	"513": "Für Grund und Zeitraum wurde bereits ein Ergänzungszuschuss verbucht",
	"514": "Für diesen ZP ist kein Grundkontingent vorhanden",
	"521": "Zählpunkt war versorgt",
	"522": "Grundkontingent-Antragsnummer wurde bereits übermittelt",
	"523": "Für diesen ZP ist Grundkontingent bereits vorhanden",
	"531": "Es handelt sich um keinen Einspeisezählpunkt",
	"532": "Zählpunkt zur Lieferperiode nicht versorgt",
	"533": "ID (eindeutige Nummer beim FAS) wurde bereits verbucht",
	"534": "Für den Zeitraum wurde bereits eine EAG-Marktprämie verbucht",
	"850": "keine Vorleistung vereinbart",
	"851": "OP-Liste bereits im Kalendermonat versendet",
	"852": "ungültiges Prozessdatum",
	"853": "OP-Liste bereits angefordert",
}

// unknownLabel is used for a code not (yet) in the Codes map above.
const unknownLabel = "Unbekannter Ablehnungsgrund"

var (
	multiCodeRe  = regexp.MustCompile(`response_codes=\[([^\]]*)\]`)
	singleCodeRe = regexp.MustCompile(`response_code=(\d+)`)
)

// Code is one ResponseCode found in an error string, with its decoded meaning.
type Code struct {
	Code  string
	Label string
}

// Parse extracts every ResponseCode from an eda_processes.error_msg string and
// returns each one with its German-language meaning (Codes[code], or
// unknownLabel if not in the table). Mirrors parseEdaErrorCodes() in
// web/lib/eda-error-codes.ts — see that file for the two error_msg shapes
// handled here ("response_codes=[182 99]" vs "response_code=56").
func Parse(errorMsg string) []Code {
	if errorMsg == "" {
		return nil
	}

	if m := multiCodeRe.FindStringSubmatch(errorMsg); m != nil {
		var codes []Code
		for _, c := range strings.Fields(m[1]) {
			codes = append(codes, Code{Code: c, Label: labelFor(c)})
		}
		if len(codes) > 0 {
			return codes
		}
	}

	if m := singleCodeRe.FindStringSubmatch(errorMsg); m != nil {
		return []Code{{Code: m[1], Label: labelFor(m[1])}}
	}

	return nil
}

func labelFor(code string) string {
	if label, ok := Codes[code]; ok {
		return label
	}
	return unknownLabel
}
