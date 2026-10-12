"use client";

import { useState } from "react";
import type { PodlistReconcileResponse, PodlistApplyResultItem, PodlistNBFreshness } from "@/lib/api";
import { XmlPreviewToggle } from "@/components/eda-badges";

// Formats a last-received timestamp and flags it when it's old enough that the comparison
// may be based on a stale Zählpunktliste (we've seen this produce misleading "corrections"
// in practice — a factor change the Netzbetreiber hadn't caught up on yet looked like a real
// mismatch until a fresh request resolved it).
const STALE_DAYS = 30;
function formatTimestamp(iso: string): { text: string; stale: boolean } | null {
  const date = new Date(iso);
  if (isNaN(date.getTime())) return null;
  const text = date.toLocaleString("de-AT", {
    day: "2-digit", month: "2-digit", year: "numeric",
    hour: "2-digit", minute: "2-digit",
    timeZone: "Europe/Vienna",
  });
  const ageDays = (Date.now() - date.getTime()) / (1000 * 60 * 60 * 24);
  return { text, stale: ageDays > STALE_DAYS };
}

// Shows when the Zählpunktliste last arrived from each Netzbetreiber currently serving this
// EEG — one row for EEG/GEA (single NB), one per distinct NB for a BEG. A BEG sends one
// request per NB, and they don't all answer equally reliably in practice (seen one NB never
// respond across several requests over months while its siblings answered every time) — a
// single overall "last received" would hide that one NB's Zählpunkte are being compared
// against stale or nonexistent data, so each is shown individually.
function PodlistFreshnessList({ freshness }: { freshness: PodlistNBFreshness[] }) {
  if (freshness.length === 0) return null;
  return (
    <div className="mt-1.5 space-y-0.5">
      {freshness.map((nb) => {
        const info = nb.last_received_at ? formatTimestamp(nb.last_received_at) : null;
        return (
          <p
            key={nb.netzbetreiber_id}
            className={`text-xs ${!info || info.stale ? "text-amber-700 font-medium" : "text-slate-400"}`}
          >
            {nb.netzbetreiber_name}: {info ? info.text : "noch keine Antwort erhalten"}
            {info?.stale && " — älter als 30 Tage, Vergleich ggf. veraltet"}
          </p>
        );
      })}
    </div>
  );
}

function Stat({ label, value, color = "text-slate-800" }: { label: string; value: number; color?: string }) {
  return (
    <div className="text-center">
      <p className={`text-xl font-bold ${color}`}>{value.toLocaleString("de-AT")}</p>
      <p className="text-xs text-slate-500 mt-0.5">{label}</p>
    </div>
  );
}

// Compares our stored meter point data (Teilnahmefaktor, Energierichtung, Consent-ID,
// Anlagentyp, Registrierungsdaten) against the latest Netzbetreiber-reported state from
// all stored EC_PODLIST (Zählpunktliste) responses, and lets the admin review/approve the
// proposed changes — individually or in bulk. Mirrors the FehlendeDatenSection pattern
// (eda-action-forms.tsx): load → select → confirm → result banner → reload.
export function PodlistReconcileSection({ eegId, edaConfigured }: { eegId: string; edaConfigured: boolean }) {
  const [state, setState] = useState<"idle" | "loading" | "loaded" | "error">("idle");
  const [data, setData] = useState<PodlistReconcileResponse | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [loadError, setLoadError] = useState<string | null>(null);
  const [applying, setApplying] = useState(false);
  const [result, setResult] = useState<{ ok: number; failed: number; failures: string[] } | null>(null);
  const [requestingPodlist, setRequestingPodlist] = useState(false);
  const [podlistRequestMsg, setPodlistRequestMsg] = useState<{ ok: boolean; text: string } | null>(null);

  async function requestFreshPodlist() {
    setRequestingPodlist(true);
    setPodlistRequestMsg(null);
    try {
      const res = await fetch(`/api/eegs/${eegId}/eda/podlist`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      const d = await res.json().catch(() => ({}));
      if (!res.ok) {
        setPodlistRequestMsg({ ok: false, text: d.error || `Fehler ${res.status}` });
      } else {
        setPodlistRequestMsg({
          ok: true,
          text: "Zählpunktliste wurde beim Netzbetreiber angefordert. Die Antwort trifft per Mail ein (kann einige Minuten bis Stunden dauern) — danach hier erneut prüfen.",
        });
      }
    } catch (err: unknown) {
      setPodlistRequestMsg({ ok: false, text: (err as Error).message });
    } finally {
      setRequestingPodlist(false);
    }
  }

  function RequestPodlistButton() {
    return (
      <div className="space-y-2">
        <button
          type="button"
          onClick={requestFreshPodlist}
          disabled={requestingPodlist || !edaConfigured}
          title={!edaConfigured ? "EDA muss in den EEG-Einstellungen konfiguriert sein" : undefined}
          className="px-3 py-1.5 bg-slate-100 text-slate-700 text-xs font-medium rounded-lg hover:bg-slate-200 disabled:opacity-50 disabled:cursor-not-allowed transition-colors whitespace-nowrap"
        >
          {requestingPodlist ? "Wird angefordert…" : "Frische Zählpunktliste anfordern"}
        </button>
        {podlistRequestMsg && (
          <p className={`text-xs ${podlistRequestMsg.ok ? "text-green-700" : "text-red-600"}`}>
            {podlistRequestMsg.text}
          </p>
        )}
      </div>
    );
  }

  async function loadDiff() {
    setState("loading");
    setLoadError(null);
    setResult(null);
    try {
      const res = await fetch(`/api/eegs/${eegId}/eda/podlist/reconcile`);
      if (!res.ok) {
        const d = await res.json().catch(() => ({}));
        setLoadError(d.error || `Fehler ${res.status}`);
        setState("error");
        return;
      }
      const d: PodlistReconcileResponse = await res.json();
      setData(d);
      // Standard: alle Zeilen mit mindestens einem automatisch übernehmbaren Diff auswählen.
      setSelected(new Set(d.mismatches.filter((r) => r.diffs.some((x) => x.applicable)).map((r) => r.meter_point_id)));
      setState("loaded");
    } catch (err: unknown) {
      setLoadError((err as Error).message);
      setState("error");
    }
  }

  function toggleRow(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  async function applySelected() {
    if (!data || selected.size === 0) return;
    setApplying(true);
    const currentData = data;
    try {
      const res = await fetch(`/api/eegs/${eegId}/eda/podlist/apply`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ meter_point_ids: Array.from(selected) }),
      });
      const items: PodlistApplyResultItem[] = await res.json();
      const failures = items.filter((i) => !i.success);
      setResult({
        ok: items.length - failures.length,
        failed: failures.length,
        failures: failures.map((f) => {
          const row = currentData.mismatches.find((r) => r.meter_point_id === f.meter_point_id);
          return `${row?.zaehlpunkt || f.meter_point_id}: ${f.error}`;
        }),
      });
    } catch (err: unknown) {
      setResult({ ok: 0, failed: selected.size, failures: [(err as Error).message] });
    } finally {
      setApplying(false);
      await loadDiff(); // Diff neu laden, damit übernommene Zeilen sofort verschwinden.
    }
  }

  if (state === "idle") {
    return (
      <div className="bg-white rounded-xl border border-slate-200 p-6">
        <h2 className="text-base font-semibold text-slate-900">Zählpunktdaten-Abgleich</h2>
        <p className="text-sm text-slate-500 mt-1 mb-4">
          Vergleicht unsere gespeicherten Zählpunktdaten (Teilnahmefaktor, Energierichtung, Consent-ID, Anlagentyp,
          Registrierungsdaten) gegen die zuletzt vom Netzbetreiber gemeldete Zählpunktliste (EC_PODLIST).
        </p>
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={loadDiff}
            className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors"
          >
            Abgleich starten
          </button>
          <RequestPodlistButton />
        </div>
      </div>
    );
  }

  if (state === "loading") {
    return (
      <div className="bg-white rounded-xl border border-slate-200 p-6">
        <p className="text-sm text-slate-500">Wird geprüft…</p>
      </div>
    );
  }

  if (state === "error") {
    return (
      <div className="bg-white rounded-xl border border-slate-200 p-6 space-y-3">
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-800 text-sm">{loadError}</div>
        <button type="button" onClick={loadDiff} className="text-sm text-blue-600 hover:underline">
          Erneut versuchen
        </button>
      </div>
    );
  }

  if (!data) return null;

  if (!data.has_any_data) {
    return (
      <div className="bg-white rounded-xl border border-slate-200 p-6">
        <h2 className="text-base font-semibold text-slate-900">Zählpunktdaten-Abgleich</h2>
        <p className="text-sm text-slate-600 mt-2 mb-4">
          Für diese Energiegemeinschaft liegt noch keine Netzbetreiber-Antwort zur Zählpunktliste (EC_PODLIST) vor.
        </p>
        <div className="flex items-center gap-3">
          <RequestPodlistButton />
          <button type="button" onClick={loadDiff} className="text-sm text-blue-600 hover:underline whitespace-nowrap">
            Erneut prüfen
          </button>
        </div>
      </div>
    );
  }

  const applicableRows = data.mismatches.filter((r) => r.diffs.some((d) => d.applicable));

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-xl border border-slate-200 p-6">
        <div className="flex items-start justify-between mb-4">
          <div>
            <h2 className="text-base font-semibold text-slate-900">Zählpunktdaten-Abgleich</h2>
            <p className="text-xs text-slate-500 mt-0.5">
              Vergleich mit der zuletzt vom Netzbetreiber gemeldeten Zählpunktliste (EC_PODLIST).
            </p>
            <PodlistFreshnessList freshness={data.podlist_freshness} />
          </div>
          <div className="flex items-center gap-3">
            <RequestPodlistButton />
            <button type="button" onClick={loadDiff} className="text-xs text-blue-600 hover:underline whitespace-nowrap">
              Erneut prüfen
            </button>
          </div>
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <Stat
            label="Abweichungen gesamt"
            value={data.mismatches.length}
            color={data.mismatches.length > 0 ? "text-amber-700" : "text-slate-800"}
          />
          <Stat label="Automatisch übernehmbar" value={applicableRows.length} color="text-emerald-700" />
          <Stat label="Nur manuelle Prüfung" value={data.mismatches.length - applicableRows.length} />
          <Stat
            label="Unbekannte Zählpunkte (NB)"
            value={data.unknown_zaehlpunkte.length}
            color={data.unknown_zaehlpunkte.length > 0 ? "text-red-600" : "text-slate-800"}
          />
        </div>
      </div>

      {result && (
        <div
          className={`p-3 rounded-lg text-sm whitespace-pre-line ${
            result.failed === 0
              ? "bg-green-50 border border-green-200 text-green-800"
              : "bg-red-50 border border-red-200 text-red-800"
          }`}
        >
          {result.ok} Zählpunkt{result.ok === 1 ? "" : "e"} übernommen.
          {result.failed > 0 && `\n${result.failed} fehlgeschlagen:\n${result.failures.join("\n")}`}
        </div>
      )}

      {data.mismatches.length === 0 ? (
        <div className="bg-white rounded-xl border border-slate-200 p-6">
          <p className="text-sm text-slate-500">
            Keine Abweichungen gefunden — unsere Daten stimmen mit der zuletzt gemeldeten Zählpunktliste überein.
          </p>
        </div>
      ) : (
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          <div className="px-6 py-4 border-b border-slate-200 flex items-center gap-3">
            <button
              type="button"
              onClick={() => setSelected(new Set(applicableRows.map((r) => r.meter_point_id)))}
              className="text-xs font-medium text-blue-700 hover:underline"
            >
              Alle auswählen
            </button>
            <button
              type="button"
              onClick={() => setSelected(new Set())}
              className="text-xs font-medium text-slate-500 hover:underline"
            >
              Auswahl leeren
            </button>
            <span className="text-xs text-slate-500 ml-auto">
              {selected.size} von {applicableRows.length} ausgewählt
            </span>
          </div>
          <div className="divide-y divide-slate-100">
            {data.mismatches.map((row) => {
              const hasApplicable = row.diffs.some((d) => d.applicable);
              return (
                <div key={row.meter_point_id} className="px-6 py-4 flex gap-4">
                  {hasApplicable ? (
                    <input
                      type="checkbox"
                      checked={selected.has(row.meter_point_id)}
                      onChange={() => toggleRow(row.meter_point_id)}
                      className="mt-1 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                    />
                  ) : (
                    <span className="w-4 mt-1" />
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-2">
                      <span className="text-sm font-medium text-slate-800 truncate">{row.member_name || "—"}</span>
                      <span className="font-mono text-xs text-slate-500 truncate">{row.zaehlpunkt}</span>
                    </div>
                    <div className="space-y-1">
                      {row.diffs.map((d, i) => (
                        <div key={i} className="flex items-center gap-2 text-xs">
                          <span
                            className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium ${
                              d.applicable ? "bg-emerald-100 text-emerald-700" : "bg-slate-100 text-slate-600"
                            }`}
                          >
                            {d.applicable ? "wird übernommen" : "informativ — manuelle Prüfung"}
                          </span>
                          <span className="text-slate-600">{d.label}:</span>
                          <span className="text-slate-500">{d.our_value || "—"}</span>
                          <span className="text-slate-400">→</span>
                          <span className="font-medium text-slate-800">{d.nb_value || "—"}</span>
                        </div>
                      ))}
                    </div>
                    <div className="mt-2">
                      <XmlPreviewToggle
                        eegId={eegId}
                        messageId={row.source_message_id}
                        filenameHint={`podlist-${row.zaehlpunkt}`}
                        excerptForZaehlpunkt={row.zaehlpunkt}
                      />
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
          <div className="px-6 py-4 border-t border-slate-100 bg-slate-50">
            <button
              type="button"
              disabled={applying || selected.size === 0}
              onClick={applySelected}
              className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {applying ? "Wird übernommen…" : `Ausgewählte übernehmen (${selected.size})`}
            </button>
          </div>
        </div>
      )}

      {(data.unknown_zaehlpunkte.length > 0 || data.no_nb_data.length > 0) && (
        <div className="bg-slate-50 rounded-xl border border-slate-200 p-5 space-y-4">
          <h3 className="text-sm font-semibold text-slate-700">Weitere Hinweise (manuelle Prüfung)</h3>
          {data.unknown_zaehlpunkte.length > 0 && (
            <div>
              <p className="text-xs text-slate-500 mb-2">
                Zählpunkte, die der Netzbetreiber meldet, die wir aber nicht kennen:
              </p>
              <ul className="space-y-1">
                {data.unknown_zaehlpunkte.map((z) => (
                  <li key={z.zaehlpunkt} className="text-xs font-mono text-slate-700">
                    {z.zaehlpunkt} — Teilnahmefaktor laut NB: {z.reported_factor}%
                  </li>
                ))}
              </ul>
            </div>
          )}
          {data.no_nb_data.length > 0 && (
            <div>
              <p className="text-xs text-slate-500 mb-2">
                Aktive Zählpunkte, zu denen der Netzbetreiber bisher nicht geantwortet hat:
              </p>
              <ul className="space-y-1">
                {data.no_nb_data.map((z) => (
                  <li key={z.meter_point_id} className="text-xs text-slate-700">
                    <span className="font-medium">{z.member_name || "—"}</span>{" "}
                    <span className="font-mono text-slate-500">{z.zaehlpunkt}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
