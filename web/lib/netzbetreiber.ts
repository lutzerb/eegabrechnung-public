export interface NetzbetreiberInfo {
  name: string;
  portalName: string;
  portalUrl: string;
  hinweis: string;
}

const registry: Record<string, NetzbetreiberInfo> = {
  "AT001000": { name: "Wiener Netze", portalName: "Smart Meter Webportal", portalUrl: "https://smartmeter-web.wienernetze.at", hinweis: "Zugangscode wird per Post zugestellt (~7 Tage)" },
  "AT002000": { name: "Netz N\u00D6", portalName: "Smart Meter Web-Portal", portalUrl: "https://smartmeter.netz-noe.at", hinweis: "Login mit Kundennummer + Vertragskonto" },
  "AT003000": { name: "Netz O\u00D6", portalName: "eService Portal", portalUrl: "https://eservice.netzooe.at/app/login", hinweis: "Login mit Kundennummer" },
  "AT003100": { name: "Linz Netz", portalName: "Serviceportal", portalUrl: "https://www.linznetz.at/portal/de/home/online_services/serviceportal/mein_serviceportal", hinweis: 'Freigaben unter \u201EFreigabeverwaltung\u201C' },
  "AT003300": { name: "eww ag", portalName: "Kundenportal", portalUrl: "https://mein.eww.at/de/login", hinweis: 'Freigaben unter \u201EEinstellungen \u2192 Freigaben\u201C' },
  "AT004000": { name: "Salzburg Netz", portalName: "Serviceportal", portalUrl: "https://portal.salzburgnetz.at", hinweis: "Login mit Kundennummer" },
  "AT005000": { name: "TINETZ", portalName: "Kundenportal", portalUrl: "https://kundenportal.tinetz.at", hinweis: 'Freigaben unter \u201EDatenfreigaben \u2192 Anfragen\u201C' },
  "AT005100": { name: "IKB Innsbruck", portalName: "IKB Direkt", portalUrl: "https://direkt.ikb.at", hinweis: "Nur aus AT/DE/CH/IT erreichbar" },
  "AT005120": { name: "HALLAG Hall i.T.", portalName: "Kundenportal", portalUrl: "https://kundenportal.hall.ag", hinweis: "Login mit Kundennummer + Anlagennummer" },
  "AT006000": { name: "Vorarlberg Netz", portalName: "Webportal", portalUrl: "https://webportal.vorarlbergnetz.at", hinweis: 'Freigaben unter \u201EVollmachten \u2192 Datenfreigaben\u201C' },
  "AT007000": { name: "K\u00E4rntner Netze", portalName: "Mein Portal", portalUrl: "https://services.kaerntennetz.at/meinPortal", hinweis: 'Freigaben unter \u201EMeine Services \u2192 Datenfreigaben\u201C' },
  "AT007100": { name: "Stadtwerke Klagenfurt", portalName: "Meine STW", portalUrl: "https://www.stw.at/kundenportal", hinweis: "Login unter stw.at/login" },
  "AT008000": { name: "Energienetze Steiermark", portalName: "Kundenportal", portalUrl: "https://portal.e-netze.at", hinweis: "Login mit Kundennummer" },
  "AT008100": { name: "Stromnetz Graz", portalName: "Serviceportal", portalUrl: "https://www.stromnetz-graz.at/sgg/customer/account/login", hinweis: "Anlagencode von Jahresrechnung erforderlich" },
  "AT008130": { name: "Feistritzwerke", portalName: "Kundenportal", portalUrl: "https://kundenportal.feistritzwerke.at", hinweis: "F\u00FCr Steiermark/Burgenland/N\u00D6" },
  "AT009000": { name: "Netz Burgenland", portalName: "Online Kundencenter", portalUrl: "https://kundencenter.netzburgenland.at/okc-netz/home.xhtml", hinweis: 'Opt-In unter \u201EAktivierung\u201C separat erforderlich' },
};

// byMarktpartnerID looks up a Netzbetreiber by its 8-char EDA Marktpartner-ID (e.g. "AT001000")
export function byMarktpartnerID(id: string): NetzbetreiberInfo | undefined {
  return registry[id.toUpperCase()];
}

// byZaehlpunkt extracts the 8-char Marktpartner-ID prefix from a Zaehlpunkt and looks it up.
// Austrian Zaehlpunkte: "AT" + 6-char ID + rest -> prefix = first 8 chars
export function byZaehlpunkt(zaehlpunkt: string): NetzbetreiberInfo | undefined {
  if (!zaehlpunkt || zaehlpunkt.length < 8) return undefined;
  return registry[zaehlpunkt.substring(0, 8).toUpperCase()];
}

// allNetzbetreiber returns all entries sorted by name, for dropdowns
export function allNetzbetreiber(): Array<{ id: string } & NetzbetreiberInfo> {
  return Object.entries(registry)
    .map(([id, info]) => ({ id, ...info }))
    .sort((a, b) => a.name.localeCompare(b.name, "de"));
}
