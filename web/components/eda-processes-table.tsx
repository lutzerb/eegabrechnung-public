"use client";

import React, { useState, useMemo } from "react";
import type { EDAProcess } from "@/lib/api";

interface Props {
  processes: EDAProcess[];
}

function formatDate(dateStr: string | undefined): string {
  if (!dateStr) return "—";
  try {
    return new Date(dateStr).toLocaleDateString("de-AT", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return dateStr;
  }
}

function formatDateShort(dateStr: string | undefined): string {
  if (!dateStr) return "—";
  try {
    return new Date(dateStr).toLocaleDateString("de-AT", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
    });
  } catch {
    return dateStr;
  }
}

function ProcessStatusBadge({ status }: { status: string }) {
  const styles: Record<string, string> = {
    pending:         "bg-yellow-50 text-yellow-700",
    sent:            "bg-blue-50 text-blue-700",
    first_confirmed: "bg-indigo-50 text-indigo-700",
    confirmed:       "bg-green-50 text-green-700",
    completed:       "bg-green-100 text-green-800",
    rejected:        "bg-red-50 text-red-700",
    error:           "bg-red-100 text-red-800",
  };
  const labels: Record<string, string> = {
    pending:         "Ausstehend",
    sent:            "Gesendet",
    first_confirmed: "Erst-Bestätigt",
    confirmed:       "Bestätigt",
    completed:       "Abgeschlossen",
    rejected:        "Abgelehnt",
    error:           "Fehler",
  };
  const cls = styles[status] ?? "bg-slate-50 text-slate-600";
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${cls}`}>
      {labels[status] ?? status}
    </span>
  );
}

function deadlineStyle(deadline_at: string | undefined): { dot: string; text: string; label: string } {
  if (!deadline_at) return { dot: "bg-slate-300", text: "text-slate-400", label: "—" };
  const now = new Date();
  const dl = new Date(deadline_at);
  const diffMs = dl.getTime() - now.getTime();
  const diffDays = diffMs / (1000 * 60 * 60 * 24);
  if (diffMs < 0 || diffDays < 3) {
    return { dot: "bg-red-500", text: "text-red-600 font-medium", label: formatDateShort(deadline_at) };
  }
  if (diffDays <= 7) {
    return { dot: "bg-yellow-400", text: "text-yellow-600 font-medium", label: formatDateShort(deadline_at) };
  }
  return { dot: "bg-green-500", text: "text-green-600", label: formatDateShort(deadline_at) };
}

// EDA response codes — Customer Processes category, scraped from ebutilities.at/responsecodes
const RESPONSE_CODES: Record<string, string> = {
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
};

interface ParsedError {
  code: string | null;
  label: string;
  detail: string;
}

function parseErrorMsg(errorMsg: string | undefined): ParsedError | null {
  if (!errorMsg) return null;

  // Pattern: "ABLEHNUNG_ECON response_code=56"
  const m = errorMsg.match(/response_code=(\d+)/);
  if (m) {
    const code = m[1];
    const description = RESPONSE_CODES[code] ?? "Unbekannter Ablehnungsgrund";
    return { code, label: description, detail: errorMsg };
  }

  // EDASendError / gateway validation error
  if (errorMsg.includes("Validation failed") || errorMsg.includes("XML errors")) {
    const firstLine = errorMsg.split("\n")[0].trim();
    return { code: null, label: "XML-Validierungsfehler", detail: firstLine };
  }

  // SMTP / network send error
  if (errorMsg.startsWith("smtp") || errorMsg.startsWith("transport") || errorMsg.startsWith("failed to")) {
    return { code: null, label: "Sendefehler", detail: errorMsg };
  }

  // Generic gateway reason text
  return { code: null, label: "Gateway-Fehler", detail: errorMsg };
}

const TYPE_LABELS: Record<string, string> = {
  EC_REQ_ONL:     "Anmeldung",
  EC_PRTFACT_CHG: "Teilnahmefaktor",
  CM_REV_SP:      "Widerruf",
  EC_REQ_PT:      "Zählerstandsgang",
};

const STATUS_OPTIONS = [
  { value: "", label: "Alle Status" },
  { value: "pending", label: "Ausstehend" },
  { value: "sent", label: "Gesendet" },
  { value: "first_confirmed", label: "Erst-Bestätigt" },
  { value: "confirmed", label: "Bestätigt" },
  { value: "completed", label: "Abgeschlossen" },
  { value: "rejected", label: "Abgelehnt" },
  { value: "error", label: "Fehler" },
];

const TYPE_OPTIONS = [
  { value: "", label: "Alle Typen" },
  { value: "EC_REQ_ONL", label: "Anmeldung" },
  { value: "EC_PRTFACT_CHG", label: "Teilnahmefaktor" },
  { value: "CM_REV_SP", label: "Widerruf" },
  { value: "EC_REQ_PT", label: "Zählerstandsgang" },
];

export function EDAProcessesTable({ processes }: Props) {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [typeFilter, setTypeFilter] = useState("");
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const filtered = useMemo(() => {
    const q = search.toLowerCase();
    return processes.filter((proc) => {
      if (statusFilter && proc.status !== statusFilter) return false;
      if (typeFilter && proc.process_type !== typeFilter) return false;
      if (q) {
        const haystack = [proc.zaehlpunkt, proc.process_type, proc.error_msg, proc.member_name]
          .filter(Boolean)
          .join(" ")
          .toLowerCase();
        if (!haystack.includes(q)) return false;
      }
      return true;
    });
  }, [processes, statusFilter, typeFilter, search]);

  if (processes.length === 0) {
    return (
      <div className="px-6 py-8 text-center text-slate-400 text-sm">
        Keine Prozesse vorhanden.
      </div>
    );
  }

  return (
    <>
      {/* Filters */}
      <div className="px-4 py-3 border-b border-slate-100 flex flex-wrap gap-3 items-center bg-slate-50/50">
        <input
          type="search"
          placeholder="Suche nach Zählpunkt oder Mitglied…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="flex-1 min-w-[180px] text-sm border border-slate-200 rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
        />
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="text-sm border border-slate-200 rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
        >
          {STATUS_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>
        <select
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value)}
          className="text-sm border border-slate-200 rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
        >
          {TYPE_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>
        {(search || statusFilter || typeFilter) && (
          <button
            onClick={() => { setSearch(""); setStatusFilter(""); setTypeFilter(""); }}
            className="text-xs text-slate-500 hover:text-slate-700 underline"
          >
            Zurücksetzen
          </button>
        )}
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full text-sm min-w-[800px]">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-50">
              <th className="text-left px-6 py-3 font-medium text-slate-600">Typ</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Mitglied</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Zählpunkt</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Status</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Faktor</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Gültig ab</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Frist</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Gestartet</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {filtered.length === 0 ? (
              <tr>
                <td colSpan={8} className="px-6 py-8 text-center text-slate-400 text-sm">
                  Keine Prozesse gefunden.
                </td>
              </tr>
            ) : (
              filtered.map((proc) => (
                <React.Fragment key={proc.id}>
                <tr
                  className={`transition-colors ${proc.error_msg ? "cursor-pointer" : ""} ${expandedId === proc.id ? "bg-red-50/40" : "hover:bg-slate-50"}`}
                  onClick={() => proc.error_msg ? setExpandedId(expandedId === proc.id ? null : proc.id) : undefined}
                >
                  <td className="px-6 py-3.5">
                    <span className="font-mono text-xs text-slate-600">
                      {TYPE_LABELS[proc.process_type] ?? proc.process_type}
                    </span>
                  </td>
                  <td className="px-6 py-3.5 text-xs text-slate-700 max-w-[160px] truncate" title={proc.member_name}>
                    {proc.member_name || <span className="text-slate-400">—</span>}
                  </td>
                  <td className="px-6 py-3.5 font-mono text-xs text-slate-600 max-w-[200px] truncate">
                    {proc.zaehlpunkt}
                  </td>
                  <td className="px-6 py-3.5">
                    <div className="flex flex-wrap items-center gap-1.5">
                      <ProcessStatusBadge status={proc.status} />
                      {proc.error_msg && (() => {
                        const parsed = parseErrorMsg(proc.error_msg);
                        if (!parsed) return null;
                        return parsed.code ? (
                          <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-mono font-semibold bg-red-100 text-red-800">
                            Code {parsed.code}
                          </span>
                        ) : null;
                      })()}
                    </div>
                    {proc.error_msg && (() => {
                      const parsed = parseErrorMsg(proc.error_msg);
                      if (!parsed) return null;
                      return (
                        <p className="text-xs text-red-700 mt-1 leading-snug" title={parsed.detail}>
                          {parsed.label}
                        </p>
                      );
                    })()}
                  </td>
                  <td className="px-6 py-3.5 text-slate-600 text-xs tabular-nums">
                    {proc.participation_factor != null
                      ? `${proc.participation_factor.toLocaleString("de-AT", { maximumFractionDigits: 2 })} %`
                      : "—"}
                  </td>
                  <td className="px-6 py-3.5 text-slate-600 text-xs whitespace-nowrap">
                    {proc.valid_from ? formatDateShort(proc.valid_from) : "—"}
                  </td>
                  <td className="px-6 py-3.5 text-xs whitespace-nowrap">
                    {(() => {
                      const ds = deadlineStyle(proc.deadline_at);
                      if (!proc.deadline_at) return <span className="text-slate-400">—</span>;
                      return (
                        <span className={`inline-flex items-center gap-1.5 ${ds.text}`}>
                          <span className={`w-2 h-2 rounded-full flex-shrink-0 ${ds.dot}`} />
                          {ds.label}
                        </span>
                      );
                    })()}
                  </td>
                  <td className="px-6 py-3.5 text-slate-400 text-xs whitespace-nowrap">
                    {formatDate(proc.initiated_at)}
                    {proc.error_msg && (
                      <span className="ml-1 text-slate-300 text-xs">{expandedId === proc.id ? "▲" : "▼"}</span>
                    )}
                  </td>
                </tr>
                {expandedId === proc.id && proc.error_msg && (() => {
                  const parsed = parseErrorMsg(proc.error_msg);
                  return (
                    <tr key={`${proc.id}-detail`} className="bg-red-50/60 border-t border-red-100">
                      <td colSpan={8} className="px-6 py-4">
                        <div className="flex items-start gap-6 text-xs">
                          {parsed?.code && (
                            <div className="shrink-0">
                              <p className="text-slate-500 font-medium mb-1">Ablehnungscode</p>
                              <span className="text-2xl font-mono font-bold text-red-700">{parsed.code}</span>
                            </div>
                          )}
                          <div className="min-w-0 flex-1">
                            <p className="text-slate-500 font-medium mb-1">Bedeutung</p>
                            <p className="text-red-800 font-medium text-sm">{parsed?.label ?? "Unbekannter Fehler"}</p>
                            {parsed?.code && (
                              <p className="text-slate-500 mt-1">
                                Code {parsed.code} laut ebutilities.at Responsecodes (Kategorie Customer Processes)
                              </p>
                            )}
                          </div>
                          <div className="min-w-0 max-w-sm">
                            <p className="text-slate-500 font-medium mb-1">Rohe Fehlermeldung</p>
                            <p className="text-slate-600 break-words leading-relaxed">{proc.error_msg}</p>
                          </div>
                        </div>
                      </td>
                    </tr>
                  );
                })()}
                </React.Fragment>
              ))
            )}
          </tbody>
        </table>
      </div>

      {(search || statusFilter || typeFilter) && (
        <div className="px-6 py-3 border-t border-slate-100 bg-slate-50">
          <p className="text-xs text-slate-500">
            {filtered.length} von {processes.length} Prozessen
          </p>
        </div>
      )}
    </>
  );
}
