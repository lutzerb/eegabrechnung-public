"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import type { FehlendeDatenPreviewItem, FehlendeDatenPreviewResponse, FehlendeDatenCategory } from "@/lib/api";

export function PollNowButton({ eegId }: { eegId: string }) {
  const [state, setState] = useState<"idle" | "polling" | "done">("idle");
  const router = useRouter();

  async function handleClick() {
    setState("polling");
    try {
      await fetch(`/api/eegs/${eegId}/eda/poll-now`, { method: "POST" });
    } finally {
      setState("done");
      setTimeout(() => { router.refresh(); setState("idle"); }, 1500);
    }
  }

  return (
    <button
      type="button"
      disabled={state !== "idle"}
      onClick={handleClick}
      className="px-3 py-1.5 text-xs font-medium bg-slate-800 text-white rounded-lg hover:bg-slate-700 disabled:opacity-50 transition-colors"
    >
      {state === "polling" ? "Polling…" : state === "done" ? "Gestartet ✓" : "Jetzt pollen"}
    </button>
  );
}

type Tab = "anmeldung-online" | "teilnahmefaktor" | "zaehlerstandsgang" | "podlist" | "widerruf";

export interface MemberMeterPointOption {
  memberName: string;
  meterId: string;
  direction: string;
  abgemeldetAm?: string;
}

interface Props {
  eegId: string;
  edaConfigured: boolean;
  members?: {
    name: string;
    name1?: string;
    name2?: string;
    meter_points: { meter_id: string; direction: string; abgemeldet_am?: string }[];
  }[];
}

export function EDAActionForms({ eegId, edaConfigured, members = [] }: Props) {
  const [activeTab, setActiveTab] = useState<Tab>("anmeldung-online");
  const router = useRouter();

  const meterPointOptions: MemberMeterPointOption[] = members.flatMap((m) =>
    m.meter_points
      .filter((mp) => mp.meter_id)
      .map((mp) => ({
        memberName: m.name || m.name1 || m.name2 || "—",
        meterId: mp.meter_id,
        direction: mp.direction,
        abgemeldetAm: mp.abgemeldet_am,
      }))
  );

  const tabs: { id: Tab; label: string }[] = [
    { id: "anmeldung-online", label: "Online-Anmeldung" },
    { id: "teilnahmefaktor", label: "Teilnahmefaktor" },
    { id: "zaehlerstandsgang", label: "Zählpunktdaten" },
    { id: "podlist", label: "Zählpunktliste" },
    { id: "widerruf", label: "Widerruf" },
  ];

  return (
    <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
      <div className="px-6 py-4 border-b border-slate-200">
        <h2 className="text-base font-semibold text-slate-900">EDA Prozesse starten</h2>
        <p className="text-xs text-slate-500 mt-0.5">
          Anmeldung, Widerruf und Teilnahmefaktoränderung per Marktkommunikation senden.
        </p>
      </div>

      {!edaConfigured && (
        <div className="px-6 py-4 bg-amber-50 border-b border-amber-100 text-sm text-amber-800">
          Bitte konfigurieren Sie zuerst die{" "}
          <a href={`/eegs/${eegId}/settings`} className="font-medium underline">
            EDA-Kommunikationseinstellungen
          </a>{" "}
          (Marktpartner-ID und Netzbetreiber-ID).
        </div>
      )}

      {/* Tabs */}
      <div className="flex border-b border-slate-200">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`px-5 py-3 text-sm font-medium transition-colors border-b-2 -mb-px ${
              activeTab === tab.id
                ? "border-blue-600 text-blue-700"
                : "border-transparent text-slate-500 hover:text-slate-700"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="p-6">
        {activeTab === "anmeldung-online" && (
          <AnmeldungOnlineForm eegId={eegId} disabled={!edaConfigured} onSuccess={() => router.refresh()} />
        )}
        {activeTab === "teilnahmefaktor" && (
          <TeilnahmefaktorForm eegId={eegId} disabled={!edaConfigured} onSuccess={() => router.refresh()} />
        )}
        {activeTab === "zaehlerstandsgang" && (
          <div className="space-y-8">
            <ZaehlerstandsgangForm eegId={eegId} disabled={!edaConfigured} meterPointOptions={meterPointOptions} onSuccess={() => router.refresh()} />
            <div className="border-t border-slate-200 pt-6">
              <FehlendeDatenSection eegId={eegId} disabled={!edaConfigured} onSuccess={() => router.refresh()} />
            </div>
          </div>
        )}
        {activeTab === "podlist" && (
          <PODListForm eegId={eegId} disabled={!edaConfigured} onSuccess={() => router.refresh()} />
        )}
        {activeTab === "widerruf" && (
          <WiderrufForm eegId={eegId} disabled={!edaConfigured} onSuccess={() => router.refresh()} />
        )}
      </div>
    </div>
  );
}

// ── Online-Anmeldung (EC_REQ_ONL) ─────────────────────────────────────────────

function AnmeldungOnlineForm({
  eegId,
  disabled,
  onSuccess,
}: {
  eegId: string;
  disabled: boolean;
  onSuccess: () => void;
}) {
  const [zaehlpunkt, setZaehlpunkt] = useState("");
  const [validFrom, setValidFrom] = useState("");
  const [factor, setFactor] = useState("100");
  const [energyDirection, setEnergyDirection] = useState("CONSUMPTION");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(false);
    try {
      const body: Record<string, unknown> = {
        zaehlpunkt,
        valid_from: validFrom || undefined,
        energy_direction: energyDirection,
        participation_factor: parseFloat(factor),
      };
      const res = await fetch(`/api/eegs/${eegId}/eda/anmeldung-online`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      setSuccess(true);
      setZaehlpunkt("");
      setValidFrom("");
      setFactor("100");
      onSuccess();
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-4 max-w-lg">
      {/* CCM flow explanation */}
      <div className="p-3 bg-blue-50 border border-blue-200 rounded-lg text-sm text-blue-800">
        <p className="font-medium mb-1">Ablauf Online-Anmeldung (ECM-Prozess):</p>
        <ol className="list-decimal list-inside space-y-0.5 text-blue-700">
          <li>EEG &rarr; NB: Anmeldeanforderung wird jetzt gesendet</li>
          <li>NB &rarr; Mitglied: Zustimmungslink (am NB-Portal)</li>
          <li>Mitglied: Bestätigt am NB-Portal</li>
          <li>NB &rarr; EEG: Bestätigung (ABSCHLUSS_ECON) &mdash; wird automatisch verarbeitet</li>
        </ol>
      </div>

      {success && (
        <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-green-800 text-sm">
          Online-Anmeldung wurde in die Warteschlange aufgenommen und wird übermittelt.
        </div>
      )}
      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-800 text-sm">
          {error}
        </div>
      )}

      <div>
        <label className="block text-sm font-medium text-slate-700 mb-1.5">Zählpunkt-ID *</label>
        <input
          type="text"
          value={zaehlpunkt}
          onChange={(e) => setZaehlpunkt(e.target.value)}
          placeholder="AT..."
          required
          disabled={disabled || loading}
          className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:bg-slate-50 disabled:text-slate-400"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-slate-700 mb-1.5">Energierichtung</label>
        <select
          value={energyDirection}
          onChange={(e) => setEnergyDirection(e.target.value)}
          disabled={disabled || loading}
          className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:bg-slate-50"
        >
          <option value="CONSUMPTION">CONSUMPTION — Verbrauch</option>
          <option value="GENERATION">GENERATION — Einspeisung</option>
        </select>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">Gültig ab</label>
          <input
            type="date"
            value={validFrom}
            onChange={(e) => setValidFrom(e.target.value)}
            disabled={disabled || loading}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:bg-slate-50"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">Teilnahmefaktor (%) *</label>
          <input
            type="number"
            step="0.01"
            min="0.01"
            max="100"
            value={factor}
            onChange={(e) => setFactor(e.target.value)}
            placeholder="100"
            required
            disabled={disabled || loading}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:bg-slate-50"
          />
        </div>
      </div>

      <button
        type="submit"
        disabled={disabled || loading || !zaehlpunkt || !factor}
        className="px-5 py-2.5 bg-indigo-700 text-white text-sm font-medium rounded-lg hover:bg-indigo-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        {loading ? "Wird gesendet…" : "Online-Anmeldung senden"}
      </button>
    </form>
  );
}

function TeilnahmefaktorForm({
  eegId,
  disabled,
  onSuccess,
}: {
  eegId: string;
  disabled: boolean;
  onSuccess: () => void;
}) {
  const [zaehlpunkt, setZaehlpunkt] = useState("");
  const [factor, setFactor] = useState("");
  const [validFrom, setValidFrom] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(false);
    try {
      const res = await fetch(`/api/eegs/${eegId}/eda/teilnahmefaktor`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          zaehlpunkt,
          participation_factor: parseFloat(factor),
          share_type: "GC",
          valid_from: validFrom || undefined,
        }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      setSuccess(true);
      setZaehlpunkt("");
      setFactor("");
      setValidFrom("");
      onSuccess();
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-4 max-w-md">
      <p className="text-sm text-slate-600">
        Teilnahmefaktor für einen Zählpunkt ändern (EC_PRTFACT_CHG).{" "}
        <span className="text-slate-400">Nur 09:00–17:00 Uhr (Wien), einmal pro Tag pro Zählpunkt.</span>
      </p>

      {success && (
        <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-green-800 text-sm">
          Teilnahmefaktoränderung wurde in die Warteschlange aufgenommen und wird übermittelt.
        </div>
      )}
      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-800 text-sm">
          {error}
        </div>
      )}

      <div>
        <label className="block text-sm font-medium text-slate-700 mb-1.5">Zählpunkt-ID *</label>
        <input
          type="text"
          value={zaehlpunkt}
          onChange={(e) => setZaehlpunkt(e.target.value)}
          placeholder="AT..."
          required
          disabled={disabled || loading}
          className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-slate-50 disabled:text-slate-400"
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">Neuer Faktor (%) *</label>
          <input
            type="number"
            step="0.001"
            min="0.001"
            max="100"
            value={factor}
            onChange={(e) => setFactor(e.target.value)}
            placeholder="100"
            required
            disabled={disabled || loading}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-slate-50"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">Gültig ab (Standard: morgen)</label>
          <input
            type="date"
            value={validFrom}
            onChange={(e) => setValidFrom(e.target.value)}
            disabled={disabled || loading}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-slate-50"
          />
        </div>
      </div>

      <button
        type="submit"
        disabled={disabled || loading || !zaehlpunkt || !factor}
        className="px-5 py-2.5 bg-purple-700 text-white text-sm font-medium rounded-lg hover:bg-purple-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        {loading ? "Wird gesendet…" : "Faktoränderung senden"}
      </button>
    </form>
  );
}

// ── Zählpunktdaten nachfordern (CR_REQ_PT) ────────────────────────────────────

// Sends a single CR_REQ_PT request; returns a formatted failure string, or null
// on success. Shared between ZaehlerstandsgangForm's own submit loop and
// FehlendeDatenSection's confirm step, since both fan out multiple (Zählpunkt,
// Zeitraum) requests against the same single-ZP backend endpoint.
async function postZaehlerstandsgang(
  eegId: string,
  zaehlpunkt: string,
  dateFrom: string,
  dateTo: string
): Promise<string | null> {
  try {
    const res = await fetch(`/api/eegs/${eegId}/eda/zaehlerstandsgang`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ zaehlpunkt, date_from: dateFrom, date_to: dateTo }),
    });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      return `${zaehlpunkt} (${dateFrom}–${dateTo}): ${data.error || `Fehler ${res.status}`}`;
    }
    return null;
  } catch (err: unknown) {
    return `${zaehlpunkt} (${dateFrom}–${dateTo}): ${(err as Error).message}`;
  }
}

interface PendingZaehlerstandsgangRequest {
  zaehlpunkt: string;
  date_from: string;
  date_to: string;
}

// For each selected Zählpunkt, fetches its registration periods and expands
// them into one request per period. Works for any Zählpunkt string (periods
// are keyed by eeg_id+zaehlpunkt, not a local meter_points row), so it also
// covers free-typed ZPs not in the picker. Open periods (no abgemeldet_am) run
// through "yesterday" — the backend never has data for today yet.
async function buildActivePeriodRequests(
  eegId: string,
  zaehlpunkte: string[]
): Promise<{ requests: PendingZaehlerstandsgangRequest[]; failures: string[] }> {
  const requests: PendingZaehlerstandsgangRequest[] = [];
  const failures: string[] = [];
  for (const zp of zaehlpunkte) {
    try {
      const res = await fetch(`/api/eegs/${eegId}/eda/zaehlerstandsgang/perioden?zaehlpunkt=${encodeURIComponent(zp)}`);
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        failures.push(`${zp}: ${data.error || `Fehler ${res.status}`}`);
        continue;
      }
      const periods: { registriert_seit: string; abgemeldet_am?: string }[] = await res.json();
      if (periods.length === 0) {
        failures.push(`${zp}: keine Registrierungsperioden gefunden`);
        continue;
      }
      for (const p of periods) {
        requests.push({
          zaehlpunkt: zp,
          date_from: p.registriert_seit.slice(0, 10),
          date_to: p.abgemeldet_am ? p.abgemeldet_am.slice(0, 10) : yesterday(),
        });
      }
    } catch (err: unknown) {
      failures.push(`${zp}: ${(err as Error).message}`);
    }
  }
  return { requests, failures };
}

function ZaehlerstandsgangForm({
  eegId,
  disabled,
  meterPointOptions,
  onSuccess,
}: {
  eegId: string;
  disabled: boolean;
  meterPointOptions: MemberMeterPointOption[];
  onSuccess: () => void;
}) {
  const [zaehlpunkteText, setZaehlpunkteText] = useState("");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [useActivePeriods, setUseActivePeriods] = useState(false);
  const [pickerFilter, setPickerFilter] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  function parseZaehlpunkte(): string[] {
    return Array.from(
      new Set(
        zaehlpunkteText
          .split(/[\n,;]+/)
          .map((z) => z.trim())
          .filter((z) => z.length > 0)
      )
    );
  }

  function toggleMeterPoint(meterId: string) {
    const current = parseZaehlpunkte();
    const next = current.includes(meterId)
      ? current.filter((z) => z !== meterId)
      : [...current, meterId];
    setZaehlpunkteText(next.join("\n"));
  }

  const filteredOptions = meterPointOptions.filter((opt) => {
    if (!pickerFilter.trim()) return true;
    const needle = pickerFilter.trim().toLowerCase();
    return (
      opt.memberName.toLowerCase().includes(needle) ||
      opt.meterId.toLowerCase().includes(needle)
    );
  });

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(null);

    const zaehlpunkte = parseZaehlpunkte();
    if (zaehlpunkte.length === 0) {
      setError("Mindestens ein Zählpunkt ist erforderlich");
      setLoading(false);
      return;
    }
    if (!useActivePeriods && (!dateFrom || !dateTo)) {
      setError("Von und Bis sind erforderlich");
      setLoading(false);
      return;
    }

    let requests: PendingZaehlerstandsgangRequest[];
    const preFailures: string[] = [];
    if (useActivePeriods) {
      const built = await buildActivePeriodRequests(eegId, zaehlpunkte);
      requests = built.requests;
      preFailures.push(...built.failures);
    } else {
      requests = zaehlpunkte.map((zaehlpunkt) => ({ zaehlpunkt, date_from: dateFrom, date_to: dateTo }));
    }

    const failures: string[] = [...preFailures];
    for (const req of requests) {
      const err = await postZaehlerstandsgang(eegId, req.zaehlpunkt, req.date_from, req.date_to);
      if (err) failures.push(err);
    }
    const totalAttempted = requests.length + preFailures.length;

    if (failures.length === 0) {
      setSuccess(
        requests.length === 1
          ? "Anfrage wurde in die Warteschlange aufgenommen und wird übermittelt."
          : `${requests.length} Anfragen wurden in die Warteschlange aufgenommen und werden übermittelt.`
      );
      setZaehlpunkteText("");
      setDateFrom("");
      setDateTo("");
      onSuccess();
    } else {
      setError(
        `${failures.length} von ${totalAttempted} Anfragen fehlgeschlagen:\n${failures.join("\n")}`
      );
      if (failures.length < totalAttempted) onSuccess();
    }
    setLoading(false);
  }

  const zaehlpunkte = parseZaehlpunkte();
  const selectedSet = new Set(zaehlpunkte);

  return (
    <form onSubmit={submit} className="space-y-4 max-w-2xl">
      <p className="text-sm text-slate-600">
        Zählpunktdaten (Messwerte) für einen Zeitraum beim Netzbetreiber nachfordern (CR_REQ_PT). Mehrere Zählpunkte können gleichzeitig abgefragt werden.
      </p>

      {success && (
        <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-green-800 text-sm">
          {success}
        </div>
      )}
      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-800 text-sm whitespace-pre-line">
          {error}
        </div>
      )}

      {meterPointOptions.length > 0 && (
        <div className="border border-slate-200 rounded-lg overflow-hidden">
          <div className="px-3 py-2 bg-slate-50 border-b border-slate-200 flex items-center gap-2">
            <input
              type="text"
              value={pickerFilter}
              onChange={(e) => setPickerFilter(e.target.value)}
              placeholder="Mitglied oder Zählpunkt suchen…"
              disabled={disabled}
              className="flex-1 px-2.5 py-1.5 text-sm border border-slate-300 rounded-md text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-teal-500 disabled:bg-slate-50"
            />
            <button
              type="button"
              disabled={disabled || filteredOptions.length === 0}
              onClick={() => {
                const current = new Set(parseZaehlpunkte());
                filteredOptions.forEach((opt) => current.add(opt.meterId));
                setZaehlpunkteText(Array.from(current).join("\n"));
              }}
              className="text-xs font-medium text-teal-700 hover:underline whitespace-nowrap disabled:opacity-40 disabled:no-underline"
            >
              alle auswählen
            </button>
            <button
              type="button"
              disabled={disabled || zaehlpunkte.length === 0}
              onClick={() => setZaehlpunkteText("")}
              className="text-xs font-medium text-slate-500 hover:underline whitespace-nowrap disabled:opacity-40 disabled:no-underline"
            >
              Auswahl leeren
            </button>
          </div>
          <div className="max-h-56 overflow-y-auto divide-y divide-slate-100">
            {filteredOptions.length === 0 ? (
              <p className="px-3 py-3 text-sm text-slate-400">Keine Zählpunkte gefunden.</p>
            ) : (
              filteredOptions.map((opt) => (
                <label
                  key={opt.meterId}
                  className="flex items-center gap-2.5 px-3 py-2 text-sm hover:bg-slate-50 cursor-pointer"
                >
                  <input
                    type="checkbox"
                    checked={selectedSet.has(opt.meterId)}
                    onChange={() => toggleMeterPoint(opt.meterId)}
                    disabled={disabled}
                    className="rounded border-slate-300 text-teal-700 focus:ring-teal-500"
                  />
                  <span className="text-slate-700 truncate">{opt.memberName}</span>
                  <span className="font-mono text-xs text-slate-500 truncate">{opt.meterId}</span>
                  <span className="ml-auto text-xs text-slate-400 whitespace-nowrap">
                    {opt.direction === "CONSUMPTION" ? "Bezug" : "Einspeisung"}
                    {opt.abgemeldetAm && " · abgemeldet"}
                  </span>
                </label>
              ))
            )}
          </div>
        </div>
      )}

      <div>
        <label className="block text-sm font-medium text-slate-700 mb-1.5">
          Zählpunkt-ID(s) *
        </label>
        <textarea
          value={zaehlpunkteText}
          onChange={(e) => setZaehlpunkteText(e.target.value)}
          placeholder={"AT...\nAT...\nAT..."}
          required
          rows={4}
          disabled={disabled || loading}
          className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-teal-500 disabled:bg-slate-50 disabled:text-slate-400 font-mono text-sm"
        />
        <p className="text-xs text-slate-500 mt-1">
          Mehrere Zählpunkte: je eine ID pro Zeile (oder durch Komma getrennt).
          {zaehlpunkte.length > 1 && ` ${zaehlpunkte.length} Zählpunkte erkannt.`}
        </p>
      </div>

      <label className="flex items-center gap-2 text-sm text-slate-700">
        <input
          type="checkbox"
          checked={useActivePeriods}
          onChange={(e) => setUseActivePeriods(e.target.checked)}
          disabled={disabled || loading}
          className="rounded border-slate-300 text-teal-700 focus:ring-teal-500"
        />
        Nur aktive Perioden anfordern (je eine Anfrage pro Registrierungsperiode statt manuellem Zeitraum)
      </label>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Von {!useActivePeriods && "*"}
          </label>
          <input
            type="date"
            value={dateFrom}
            onChange={(e) => setDateFrom(e.target.value)}
            required={!useActivePeriods}
            disabled={disabled || loading || useActivePeriods}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-teal-500 disabled:bg-slate-50"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Bis {!useActivePeriods && "*"}
          </label>
          <input
            type="date"
            value={dateTo}
            onChange={(e) => setDateTo(e.target.value)}
            required={!useActivePeriods}
            disabled={disabled || loading || useActivePeriods}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-teal-500 disabled:bg-slate-50"
          />
        </div>
      </div>

      <button
        type="submit"
        disabled={disabled || loading || zaehlpunkte.length === 0 || (!useActivePeriods && (!dateFrom || !dateTo))}
        className="px-5 py-2.5 bg-teal-700 text-white text-sm font-medium rounded-lg hover:bg-teal-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        {loading
          ? "Wird gesendet…"
          : zaehlpunkte.length > 1
          ? `Daten für ${zaehlpunkte.length} Zählpunkte anfordern`
          : "Daten anfordern"}
      </button>
    </form>
  );
}

// ── Fehlende Daten nachfordern (EEG-weite CR_REQ_PT-Lückenprüfung) ───────────

interface FehlendeDatenRow {
  key: string;
  zaehlpunkt: string;
  memberName: string;
  from: string;
  to: string;
  category: FehlendeDatenCategory;
  inFlight: boolean;
}

function flattenFehlendeDatenRows(items: FehlendeDatenPreviewItem[]): FehlendeDatenRow[] {
  return items.flatMap((it) =>
    it.missing_ranges.map((r) => ({
      key: `${it.zaehlpunkt}__${it.period_id}__${r.from}__${r.to}`,
      zaehlpunkt: it.zaehlpunkt,
      memberName: it.member_name,
      from: r.from,
      to: r.to,
      category: r.category,
      inFlight: it.in_flight,
    }))
  );
}

function CategoryBadge({ category }: { category: FehlendeDatenCategory }) {
  const styles: Record<FehlendeDatenCategory, string> = {
    no_data: "bg-red-100 text-red-700",
    l3_only: "bg-orange-100 text-orange-700",
    partial: "bg-slate-100 text-slate-600",
  };
  const labels: Record<FehlendeDatenCategory, string> = {
    no_data: "keine Daten",
    l3_only: "nur L3",
    partial: "teilweise",
  };
  return (
    <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium ${styles[category]}`}>
      {labels[category]}
    </span>
  );
}

function FehlendeDatenSection({
  eegId,
  disabled,
  onSuccess,
}: {
  eegId: string;
  disabled: boolean;
  onSuccess: () => void;
}) {
  const [state, setState] = useState<"idle" | "loading" | "loaded" | "error">("idle");
  const [rows, setRows] = useState<FehlendeDatenRow[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [includeNoData, setIncludeNoData] = useState(true);
  const [includeL3Only, setIncludeL3Only] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [result, setResult] = useState<{ ok: number; failures: string[] } | null>(null);

  async function loadPreview() {
    setState("loading");
    setLoadError(null);
    setResult(null);
    try {
      const res = await fetch(`/api/eegs/${eegId}/eda/zaehlerstandsgang/fehlende-daten`);
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setLoadError(data.error || `Fehler ${res.status}`);
        setState("error");
        return;
      }
      const data: FehlendeDatenPreviewResponse = await res.json();
      const flat = flattenFehlendeDatenRows(data.items);
      setRows(flat);
      setSelected(new Set(flat.map((r) => r.key))); // default: all selected
      setState("loaded");
    } catch (err: unknown) {
      setLoadError((err as Error).message);
      setState("error");
    }
  }

  function toggleRow(key: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }

  // "Zeiträume komplett ohne Daten" / "mit nur L3-Messwerten" sind abwählbar;
  // echte Teil-Lücken (partial, einzelne L1/L2-Werte vorhanden) sind nicht
  // gesondert filterbar und immer sichtbar.
  const visibleRows = rows.filter(
    (r) => r.category === "partial" || (r.category === "no_data" && includeNoData) || (r.category === "l3_only" && includeL3Only)
  );
  const visibleSelected = visibleRows.filter((r) => selected.has(r.key));

  async function confirmSelected() {
    setConfirming(true);
    const failures: string[] = [];
    for (const row of visibleSelected) {
      const err = await postZaehlerstandsgang(eegId, row.zaehlpunkt, row.from, row.to);
      if (err) failures.push(err);
    }
    setResult({ ok: visibleSelected.length - failures.length, failures });
    setConfirming(false);
    setState("idle");
    setRows([]);
    setSelected(new Set());
    if (failures.length < visibleSelected.length) onSuccess();
  }

  const groupedByZP = visibleRows.reduce<Map<string, FehlendeDatenRow[]>>((map, row) => {
    const list = map.get(row.zaehlpunkt) ?? [];
    list.push(row);
    map.set(row.zaehlpunkt, list);
    return map;
  }, new Map());

  return (
    <div className="border border-amber-200 bg-amber-50/40 rounded-lg p-4 space-y-4">
      <div>
        <h3 className="text-sm font-semibold text-amber-900">Fehlende Daten nachfordern</h3>
        <p className="text-xs text-amber-800 mt-0.5">
          Prüft alle Registrierungsperioden aller Zählpunkte dieser EEG (auch ehemalige Mitglieder) auf Zeiträume ohne Messwerte (L1/L2) und fordert die fehlenden Zeiträume gesammelt nach.
        </p>
      </div>

      {result && (
        <div
          className={`p-3 rounded-lg text-sm whitespace-pre-line ${
            result.failures.length === 0
              ? "bg-green-50 border border-green-200 text-green-800"
              : "bg-red-50 border border-red-200 text-red-800"
          }`}
        >
          {result.ok} Anfrage{result.ok === 1 ? "" : "n"} in die Warteschlange aufgenommen.
          {result.failures.length > 0 && `\n${result.failures.length} fehlgeschlagen:\n${result.failures.join("\n")}`}
        </div>
      )}
      {loadError && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-800 text-sm">{loadError}</div>
      )}

      {state === "idle" || state === "error" ? (
        <button
          type="button"
          disabled={disabled}
          onClick={loadPreview}
          className="px-4 py-2 bg-amber-700 text-white text-sm font-medium rounded-lg hover:bg-amber-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          Fehlende Daten prüfen
        </button>
      ) : state === "loading" ? (
        <p className="text-sm text-amber-800">Wird geprüft…</p>
      ) : rows.length === 0 ? (
        <p className="text-sm text-amber-800">Keine fehlenden Zeiträume gefunden.</p>
      ) : (
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-4 text-xs text-amber-900">
            <label className="flex items-center gap-1.5 cursor-pointer">
              <input
                type="checkbox"
                checked={includeNoData}
                onChange={(e) => setIncludeNoData(e.target.checked)}
                className="rounded border-slate-300 text-amber-700 focus:ring-amber-500"
              />
              Zeiträume komplett ohne Daten
            </label>
            <label className="flex items-center gap-1.5 cursor-pointer">
              <input
                type="checkbox"
                checked={includeL3Only}
                onChange={(e) => setIncludeL3Only(e.target.checked)}
                className="rounded border-slate-300 text-amber-700 focus:ring-amber-500"
              />
              Zeiträume mit nur L3-Messwerten
            </label>
          </div>

          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => setSelected(new Set([...selected, ...visibleRows.map((r) => r.key)]))}
              className="text-xs font-medium text-amber-800 hover:underline"
            >
              alle auswählen
            </button>
            <button
              type="button"
              onClick={() => setSelected(new Set([...selected].filter((k) => !visibleRows.some((r) => r.key === k))))}
              className="text-xs font-medium text-slate-500 hover:underline"
            >
              Auswahl leeren
            </button>
            <span className="text-xs text-amber-700 ml-auto">{visibleSelected.length} von {visibleRows.length} ausgewählt</span>
          </div>

          <div className="border border-amber-200 rounded-lg bg-white max-h-72 overflow-y-auto divide-y divide-slate-100">
            {visibleRows.length === 0 ? (
              <p className="px-3 py-3 text-sm text-slate-400">Keine Zeiträume für die aktuelle Filterauswahl.</p>
            ) : (
              Array.from(groupedByZP.entries()).map(([zp, zpRows]) => (
                <div key={zp} className="px-3 py-2">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-slate-800 font-medium truncate">{zpRows[0].memberName || "—"}</span>
                    <span className="font-mono text-xs text-slate-500 truncate">{zp}</span>
                    {zpRows[0].inFlight && (
                      <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-yellow-100 text-yellow-800">
                        bereits angefragt
                      </span>
                    )}
                  </div>
                  <div className="space-y-1">
                    {zpRows.map((row) => (
                      <label key={row.key} className="flex items-center gap-2 text-xs text-slate-600 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={selected.has(row.key)}
                          onChange={() => toggleRow(row.key)}
                          className="rounded border-slate-300 text-amber-700 focus:ring-amber-500"
                        />
                        {row.from} – {row.to}
                        <CategoryBadge category={row.category} />
                      </label>
                    ))}
                  </div>
                </div>
              ))
            )}
          </div>

          <div className="flex items-center gap-3">
            <button
              type="button"
              disabled={disabled || confirming || visibleSelected.length === 0}
              onClick={confirmSelected}
              className="px-4 py-2 bg-amber-700 text-white text-sm font-medium rounded-lg hover:bg-amber-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {confirming ? "Wird gesendet…" : `Anfordern (${visibleSelected.length})`}
            </button>
            <button
              type="button"
              disabled={confirming}
              onClick={() => { setState("idle"); setRows([]); setSelected(new Set()); }}
              className="text-xs text-slate-500 hover:underline disabled:opacity-50"
            >
              Abbrechen
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

// ── Zählpunktliste anfordern (EC_PODLIST) ────────────────────────────────────

// Yesterday's date as YYYY-MM-DD — the default DateTimeFrom/DateTimeTo range.
// Some Netzbetreiber (e.g. Energienetze Steiermark) reject ANFORDERUNG_ECP with
// response code 181 ("Gemeinschafts-ID nicht vorhanden") when no period is given,
// even though the schema marks it optional; others (e.g. Netz NÖ) accept it either
// way and return the "current" list. Pre-filling a period avoids the rejection.
function yesterday(): string {
  const d = new Date();
  d.setDate(d.getDate() - 1);
  return d.toISOString().slice(0, 10);
}

function PODListForm({
  eegId,
  disabled,
  onSuccess,
}: {
  eegId: string;
  disabled: boolean;
  onSuccess: () => void;
}) {
  const [dateFrom, setDateFrom] = useState(yesterday);
  const [dateTo, setDateTo] = useState(yesterday);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(false);

    try {
      const res = await fetch(`/api/eegs/${eegId}/eda/podlist`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ date_from: dateFrom, date_to: dateTo }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      setSuccess(true);
      onSuccess();
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-4 max-w-md">
      <p className="text-sm text-slate-600">
        ANFORDERUNG_ECP (CPRequest 01.12) — Zählpunktliste für den angegebenen Zeitraum vom Netzbetreiber anfordern.
      </p>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Von
          </label>
          <input
            type="date"
            value={dateFrom}
            onChange={(e) => setDateFrom(e.target.value)}
            disabled={disabled || loading}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-teal-500 disabled:bg-slate-50"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Bis
          </label>
          <input
            type="date"
            value={dateTo}
            onChange={(e) => setDateTo(e.target.value)}
            disabled={disabled || loading}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-teal-500 disabled:bg-slate-50"
          />
        </div>
      </div>
      <p className="text-xs text-slate-500">
        Beide Felder leer lassen, um ohne Zeitraum anzufragen (manche Netzbetreiber liefern dann den aktuellen Stand).
      </p>

      {success && (
        <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-green-800 text-sm">
          Zählpunktliste wurde in die Warteschlange aufgenommen und wird übermittelt.
        </div>
      )}
      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-800 text-sm">
          {error}
        </div>
      )}

      <button
        type="submit"
        disabled={disabled || loading}
        className="px-5 py-2.5 bg-slate-700 text-white text-sm font-medium rounded-lg hover:bg-slate-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        {loading ? "Wird gesendet…" : "Zählpunktliste anfordern"}
      </button>
    </form>
  );
}

// ── Widerruf (CM_REV_SP) ─────────────────────────────────────────────────────

function WiderrufForm({
  eegId,
  disabled,
  onSuccess,
}: {
  eegId: string;
  disabled: boolean;
  onSuccess: () => void;
}) {
  const [zaehlpunkt, setZaehlpunkt] = useState("");
  const [consentEnd, setConsentEnd] = useState("");
  const [reasonKey, setReasonKey] = useState("");
  const [reason, setReason] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(false);

    try {
      const body: Record<string, unknown> = {
        zaehlpunkt,
        consent_end: consentEnd,
      };
      if (reasonKey) {
        body.reason_key = parseInt(reasonKey, 10);
      }
      if (reason) {
        body.reason = reason;
      }
      const res = await fetch(`/api/eegs/${eegId}/eda/widerruf`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      setSuccess(true);
      setZaehlpunkt("");
      setConsentEnd("");
      setReasonKey("");
      setReason("");
      onSuccess();
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-4 max-w-lg">
      <p className="text-sm text-slate-600">
        AUFHEBUNG_CCMS (CMRevoke 01.10) — Zustimmung eines Zählpunkts widerrufen.{" "}
        <span className="text-slate-400">Verwendet wenn ein Mitglied die EEG verlässt.</span>
      </p>

      {success && (
        <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-green-800 text-sm">
          Widerruf wurde in die Warteschlange aufgenommen und wird übermittelt.
        </div>
      )}
      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-800 text-sm">
          {error}
        </div>
      )}

      <div>
        <label className="block text-sm font-medium text-slate-700 mb-1.5">
          Zählpunkt-ID *
        </label>
        <input
          type="text"
          value={zaehlpunkt}
          onChange={(e) => setZaehlpunkt(e.target.value)}
          placeholder="AT..."
          required
          disabled={disabled || loading}
          className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-red-500 disabled:bg-slate-50 disabled:text-slate-400"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-slate-700 mb-1.5">
          Zustimmungsende *
        </label>
        <input
          type="date"
          value={consentEnd}
          onChange={(e) => setConsentEnd(e.target.value)}
          required
          disabled={disabled || loading}
          className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-red-500 disabled:bg-slate-50"
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Reason Key (optional, 1–9)
          </label>
          <input
            type="number"
            min="1"
            max="9"
            step="1"
            value={reasonKey}
            onChange={(e) => setReasonKey(e.target.value)}
            placeholder="—"
            disabled={disabled || loading}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-red-500 disabled:bg-slate-50"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Begründung (optional, max. 50 Zeichen)
          </label>
          <input
            type="text"
            maxLength={50}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="z.B. Mitglied ausgetreten"
            disabled={disabled || loading}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-red-500 disabled:bg-slate-50"
          />
        </div>
      </div>

      <button
        type="submit"
        disabled={disabled || loading || !zaehlpunkt || !consentEnd}
        className="px-5 py-2.5 bg-red-700 text-white text-sm font-medium rounded-lg hover:bg-red-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        {loading ? "Wird gesendet…" : "Widerruf senden"}
      </button>
    </form>
  );
}
