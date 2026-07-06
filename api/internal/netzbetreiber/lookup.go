// Package netzbetreiber provides a lookup table for Austrian electricity
// grid operators (Netzbetreiber) identified by their EDA Marktpartner-ID.
package netzbetreiber

// Info holds the portal and contact details of a Netzbetreiber.
type Info struct {
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
