// Package netzbetreiber provides a lookup table for Austrian electricity
// grid operators (Netzbetreiber) identified by their EDA Marktpartner-ID.
package netzbetreiber

import (
	"sort"

	"github.com/lutzerb/eegabrechnung/internal/domain"
)

// Info holds the portal and contact details of a Netzbetreiber.
type Info struct {
	ID         string // EDA Marktpartner-ID, e.g. "AT001000"
	Name       string // e.g. "Wiener Netze"
	PortalName string // e.g. "Smart Meter Webportal"
	PortalURL  string // direct URL
	Hinweis    string // short note for members
}

// registry maps EDA Marktpartner-IDs to Netzbetreiber info.
// The ID format is "AT" + 6 decimal digits (e.g. "AT001000").
var registry = map[string]Info{
	"AT001000": {
		Name:       "Wiener Netze",
		PortalName: "Smart Meter Webportal",
		PortalURL:  "https://smartmeter-web.wienernetze.at",
		Hinweis:    "Zugangscode wird per Post zugestellt (~7 Tage)",
	},
	"AT002000": {
		Name:       "Netz NÖ",
		PortalName: "Smart Meter Web-Portal",
		PortalURL:  "https://smartmeter.netz-noe.at",
		Hinweis:    "Login mit Kundennummer + Vertragskonto",
	},
	"AT003000": {
		Name:       "Netz OÖ",
		PortalName: "eService Portal",
		PortalURL:  "https://eservice.netzooe.at/app/login",
		Hinweis:    "Login mit Kundennummer",
	},
	"AT003100": {
		Name:       "Linz Netz",
		PortalName: "Serviceportal",
		PortalURL:  "https://www.linznetz.at/portal/de/home/online_services/serviceportal/mein_serviceportal",
		Hinweis:    `Freigaben unter "Freigabeverwaltung"`,
	},
	"AT003300": {
		Name:       "eww ag",
		PortalName: "Kundenportal",
		PortalURL:  "https://mein.eww.at/de/login",
		Hinweis:    `Freigaben unter "Einstellungen → Freigaben"`,
	},
	"AT004000": {
		Name:       "Salzburg Netz",
		PortalName: "Serviceportal",
		PortalURL:  "https://portal.salzburgnetz.at",
		Hinweis:    "Login mit Kundennummer",
	},
	"AT005000": {
		Name:       "TINETZ",
		PortalName: "Kundenportal",
		PortalURL:  "https://kundenportal.tinetz.at",
		Hinweis:    `Freigaben unter "Datenfreigaben → Anfragen"`,
	},
	"AT005100": {
		Name:       "IKB Innsbruck",
		PortalName: "IKB Direkt",
		PortalURL:  "https://direkt.ikb.at",
		Hinweis:    "Nur aus AT/DE/CH/IT erreichbar",
	},
	"AT005120": {
		Name:       "HALLAG Hall i.T.",
		PortalName: "Kundenportal",
		PortalURL:  "https://kundenportal.hall.ag",
		Hinweis:    "Login mit Kundennummer + Anlagennummer",
	},
	"AT006000": {
		Name:       "Vorarlberg Netz",
		PortalName: "Webportal",
		PortalURL:  "https://webportal.vorarlbergnetz.at",
		Hinweis:    `Freigaben unter "Vollmachten → Datenfreigaben"`,
	},
	"AT007000": {
		Name:       "Kärntner Netze",
		PortalName: "Mein Portal",
		PortalURL:  "https://services.kaerntennetz.at/meinPortal",
		Hinweis:    `Freigaben unter "Meine Services → Datenfreigaben"`,
	},
	"AT007100": {
		Name:       "Stadtwerke Klagenfurt",
		PortalName: "Meine STW",
		PortalURL:  "https://www.stw.at/kundenportal",
		Hinweis:    "Login unter stw.at/login",
	},
	"AT008000": {
		Name:       "Energienetze Steiermark",
		PortalName: "Kundenportal",
		PortalURL:  "https://portal.e-netze.at",
		Hinweis:    "Login mit Kundennummer",
	},
	"AT008100": {
		Name:       "Stromnetz Graz",
		PortalName: "Serviceportal",
		PortalURL:  "https://www.stromnetz-graz.at/sgg/customer/account/login",
		Hinweis:    "Anlagencode von Jahresrechnung erforderlich",
	},
	"AT008130": {
		Name:       "Feistritzwerke",
		PortalName: "Kundenportal",
		PortalURL:  "https://kundenportal.feistritzwerke.at",
		Hinweis:    "Für Steiermark/Burgenland/NÖ",
	},
	"AT009000": {
		Name:       "Netz Burgenland",
		PortalName: "Online Kundencenter",
		PortalURL:  "https://kundencenter.netzburgenland.at/okc-netz/home.xhtml",
		Hinweis:    `Opt-In unter "Aktivierung" separat erforderlich`,
	},

	// Remaining Austrian grid operators (small local E-Werke) — name only, no
	// portal metadata (only relevant for the ~16 large operators above, whose
	// self-service portals members are guided to during onboarding).
	"AT002110": {Name: "Stadtwerke Amstetten GmbH"},
	"AT002120": {Name: "Licht- und Kraftvertrieb der Marktgemeinde Göstling an der Ybbs"},
	"AT002130": {Name: "Licht- und Kraftvertrieb der Gemeinde Hollenstein an der Ybbs"},
	"AT002210": {Name: "Elektrizitätswerke Eisenhuber GmbH & Co KG"},
	"AT002220": {Name: "Anton Kittel Mühle Plaika GmbH"},
	"AT002230": {Name: "wüsterstrom E-Werk GmbH"},
	"AT002250": {Name: "E-Werk Schwaighofer GmbH"},
	"AT002270": {Name: "Polsterer Kerres Ruttin Holding GmbH"},
	"AT002280": {Name: "Forstverwaltung Seehof GmbH"},
	"AT002290": {Name: "Forstverwaltung Neuhaus Alpl Kraftwerksbetrieb"},
	"AT002300": {Name: "Licht- und Kraftstromvertrieb der Gemeinde Opponitz"},
	"AT002400": {Name: "Heinrich Polsterer und Mitges GesnbR"},
	"AT002900": {Name: "E-Werk Sarmingstein Ing. H. Engelmann & Co KG"},
	"AT002910": {Name: "Elektrizitätswerk Clam, Carl Philip Clam Martinic e.U."},
	"AT003200": {Name: "ENERGIE RIED GmbH"},
	"AT003310": {Name: "Elektrizitätswerk Perg GmbH"},
	"AT003460": {Name: "EBNER STROM GmbH"},
	"AT003470": {Name: "KARLSTROM e.U."},
	"AT003510": {Name: "Kraftwerk Glatzing-Rüstorf eGen"},
	"AT003520": {Name: "K.u.F. Drack GmbH & Co KG"},
	"AT003540": {Name: "Revertera'sches Elektrizitätswerk"},
	"AT003570": {Name: "EVU Mathe Gerald e.U."},
	"AT003580": {Name: "Energieversorgungs GmbH"},
	"AT003590": {Name: "E-Werk Ranklleiten"},
	"AT003900": {Name: "E-Werk Altenfelden GmbH"},
	"AT003910": {Name: "E-Werk Redlmühle / Verteilnetzbetrieb"},
	"AT003920": {Name: "E-Werk Dietrichschlag eingetragene Genossenschaft"},
	"AT004110": {Name: "Elektrizitätswerk Bad Hofgastein"},
	"AT004120": {Name: "Lichtgenossenschaft Neukirchen eGen"},
	"AT004130": {Name: "Elektrizitätswerk Lechner August KG"},
	"AT005110": {Name: "Elektrizitätswerke Reutte AG"},
	"AT005130": {Name: "Stadtwerke Kitzbühel e.U."},
	"AT005140": {Name: "Stadtwerke Kufstein Gesellschaft m.b.H."},
	"AT005150": {Name: "Stadtwerke Schwaz GmbH"},
	"AT005160": {Name: "Stadtwerke Wörgl GmbH"},
	"AT005210": {Name: "Kraftwerk Haim KG"},
	"AT005320": {Name: "Energie und Wirtschaftsbetriebe der Gemeinde St. Anton GmbH"},
	"AT005330": {Name: "Kommunalbetriebe Hopfgarten GmbH"},
	"AT005340": {Name: "Elektrizitätswerk Schattwald e.U."},
	"AT005350": {Name: "Gemeindewerke Kematen"},
	"AT005370": {Name: "Stadtwerke Imst"},
	"AT005440": {Name: "Johann Dandler GmbH & CO KG"},
	"AT005460": {Name: "Elektrizitätswerk Prantl GmbH & CoKG"},
	"AT005470": {Name: "E-Werk Stadler GmbH"},
	"AT005480": {Name: "Elektrizitätswerk Winkler GmbH"},
	"AT005490": {Name: "Elektrowerk Assling reg. Gen. m. b. H."},
	"AT005540": {Name: "Elektritzitätswerk der Gemeinde Gries am Brenner"},
	"AT005600": {Name: "Elektrowerkgenossenschaft Hopfgarten i. D. reg. Gen.m.b.H."},
	"AT005610": {Name: "Wasserkraft Sölden eGen mbH"},
	"AT005630": {Name: "Elektrogenossenschaft Weerberg reg.genmbH"},
	"AT005650": {Name: "Kommunalbetriebe Rinn GmbH"},
	"AT006110": {Name: "Stadtwerke Feldkirch"},
	"AT006210": {Name: "Elektrizitätswerke Frastanz Gesellschaft mbH"},
	"AT006220": {Name: "Montafonerbahn Aktiengesellschaft"},
	"AT006230": {Name: "Energieversorgung Kleinwalsertal GesmbH"},
	"AT006240": {Name: "Getzner, Mutter & Cie. GmbH & Co KG"},
	"AT006250": {Name: "Alfenzwerke Elektrizitätserzeugung GmbH"},
	"AT007240": {Name: "AAE Wasserkraft GmbH"},
	"AT008110": {Name: "Elektrizitätswerk der Stadtgemeinde Kindberg"},
	"AT008120": {Name: "Stadtwerke Voitsberg GmbH"},
	"AT008140": {Name: "Stadtwerke Bruck an der Mur GmbH"},
	"AT008150": {Name: "Stadtwerke Fürstenfeld GmbH"},
	"AT008160": {Name: "Stadtwerke Judenburg AG"},
	"AT008170": {Name: "Stadtwerke Kapfenberg GmbH"},
	"AT008180": {Name: "Stadtwerke Köflach"},
	"AT008190": {Name: "Stadtwerke Mürzzuschlag GmbH"},
	"AT008210": {Name: "E-WERK GÖSTING STROMVERSORGUNGS GmbH"},
	"AT008310": {Name: "Bad Gleichenberger Energie GmbH"},
	"AT008350": {Name: "Städtische Betriebe Rottenmann GmbH"},
	"AT008360": {Name: "EVU der Stadtgemeinde Mureck"},
	"AT008370": {Name: "E-Werk Mürzsteg"},
	"AT008390": {Name: "Elektroversorgungsunternehmen der Marktgemeinde Eibiswald"},
	"AT008410": {Name: "EVU der Marktgemeinde Unzmarkt-Frauenburg"},
	"AT008420": {Name: "Marktgemeinde Neumarkt Versorgungsbetriebsgesellschaft m.b.H."},
	"AT008430": {Name: "Murauer Stadtwerke Gesellschaft m.b.H."},
	"AT008440": {Name: "Stadtbetriebe Mariazell GmbH"},
	"AT008470": {Name: "Stadtwerke Hartberg Energieversorgungs GmbH"},
	"AT008490": {Name: "Stadtwerke Trofaiach Ges.m.b.H."},
	"AT008510": {Name: "E-Genossenschaft Laintal eGen"},
	"AT008540": {Name: "E-Werk SIGL GmbH & CO KG"},
	"AT008560": {Name: "ENVESTA Energie- u. Dienstleistungs GmbH"},
	"AT008570": {Name: "Elektrizitätswerk Fernitz Ing. Franz Purkarthofer GmbH & Co KG"},
	"AT008580": {Name: "Kiendler Vulkanlandstrom GmbH"},
	"AT008620": {Name: "Elektrizitätswerk Gröbming KG"},
	"AT008650": {Name: "Elektrizitätswerk Mariahof GmbH"},
	"AT008690": {Name: "Gertraud Schafler GmbH."},
	"AT008720": {Name: "Elektrowerk Schöder GmbH"},
	"AT008740": {Name: "E-Werk Gleinstätten GmbH"},
	"AT008770": {Name: "Feistritzthaler Elektrizitätswerk eGen"},
	"AT008850": {Name: "Schwarz, Wagendorffer & Co, Elektrizitätswerk GmbH"},
	"AT008910": {Name: "E-Werk Stubenberg eGen"},
	"AT008930": {Name: "Joh. Pengg Holding Gesellschaft m.b.H."},
	"AT008950": {Name: "E-Werk Piwetz"},
	"AT008960": {Name: "E-Werk Neudau Kottulinsky KG"},
	"AT009220": {Name: "Energie Güssing GmbH"},
}

// ByMarktpartnerID looks up a Netzbetreiber by its EDA Marktpartner-ID (e.g. "AT001000").
// Returns the Info and true if found, or an empty Info and false if not found.
func ByMarktpartnerID(id string) (Info, bool) {
	info, ok := registry[id]
	return info, ok
}

// ByZaehlpunkt extracts the Marktpartner-ID from an Austrian Zählpunkt and looks it up.
//
// Austrian Zählpunkt format: "AT" (2 chars) + 6-char Marktpartner suffix = 8-char prefix.
// Example: "AT001000123456789012345678" → Marktpartner-ID "AT001000".
//
// Returns the Info and true if the extracted ID is known, or an empty Info and false otherwise.
func ByZaehlpunkt(zaehlpunkt string) (Info, bool) {
	if len(zaehlpunkt) < 8 {
		return Info{}, false
	}
	id := zaehlpunkt[:8]
	return ByMarktpartnerID(id)
}

// ActiveFromMeterPoints derives the distinct set of Netzbetreiber currently
// serving an EEG's active meter points (abgemeldet_am IS NULL), by grouping
// on the Zählpunkt's 8-char Marktpartner-ID prefix. Mirrors the routing logic
// used for BEG multi-Netzbetreiber EDA actions (handler/eda.go). Unknown IDs
// are still returned, with the raw ID standing in for the name.
// Result is sorted by name for a stable display order.
func ActiveFromMeterPoints(mps []domain.MeterPoint) []Info {
	seen := map[string]bool{}
	var result []Info
	for _, mp := range mps {
		if mp.AbgemeldetAm != nil || len(mp.Zaehlpunkt) < 8 {
			continue
		}
		id := mp.Zaehlpunkt[:8]
		if seen[id] {
			continue
		}
		seen[id] = true
		info, ok := ByMarktpartnerID(id)
		if !ok {
			info = Info{Name: id}
		}
		info.ID = id
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
