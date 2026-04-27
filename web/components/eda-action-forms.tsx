"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

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

interface Props {
  eegId: string;
  edaConfigured: boolean;
  netzbetreiberId: string;
}

export function EDAActionForms({ eegId, edaConfigured, netzbetreiberId }: Props) {
  const [activeTab, setActiveTab] = useState<Tab>("anmeldung-online");
  const router = useRouter();

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
          <AnmeldungOnlineForm eegId={eegId} disabled={!edaConfigured} netzbetreiberId={netzbetreiberId} onSuccess={() => router.refresh()} />
        )}
        {activeTab === "teilnahmefaktor" && (
          <TeilnahmefaktorForm eegId={eegId} disabled={!edaConfigured} netzbetreiberId={netzbetreiberId} onSuccess={() => router.refresh()} />
        )}
        {activeTab === "zaehlerstandsgang" && (
          <ZaehlerstandsgangForm eegId={eegId} disabled={!edaConfigured} netzbetreiberId={netzbetreiberId} onSuccess={() => router.refresh()} />
        )}
        {activeTab === "podlist" && (
          <PODListForm eegId={eegId} disabled={!edaConfigured} onSuccess={() => router.refresh()} />
        )}
        {activeTab === "widerruf" && (
          <WiderrufForm eegId={eegId} disabled={!edaConfigured} netzbetreiberId={netzbetreiberId} onSuccess={() => router.refresh()} />
        )}
      </div>
    </div>
  );
}

// ── Online-Anmeldung (EC_REQ_ONL) ─────────────────────────────────────────────

function AnmeldungOnlineForm({
  eegId,
  disabled,
  netzbetreiberId,
  onSuccess,
}: {
  eegId: string;
  disabled: boolean;
  netzbetreiberId: string;
  onSuccess: () => void;
}) {
  const [zaehlpunkt, setZaehlpunkt] = useState("");
  const [validFrom, setValidFrom] = useState("");
  const [factor, setFactor] = useState("");
  const [ecDisModel, setEcDisModel] = useState("S");
  const [energyDirection, setEnergyDirection] = useState("CONSUMPTION");
  const [ecShare, setEcShare] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(false);
    if (netzbetreiberId && zaehlpunkt.length >= 8 && zaehlpunkt.substring(0, 8) !== netzbetreiberId) {
      setError(`Zählpunkt-Präfix „${zaehlpunkt.substring(0, 8)}" passt nicht zum konfigurierten Netzbetreiber „${netzbetreiberId}"`);
      setLoading(false);
      return;
    }
    try {
      const body: Record<string, unknown> = {
        zaehlpunkt,
        valid_from: validFrom || undefined,
        ec_dis_model: ecDisModel,
        energy_direction: energyDirection,
        participation_factor: factor ? parseFloat(factor) : undefined,
      };
      if (ecShare) {
        body.ec_share = parseFloat(ecShare);
      }
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
      setFactor("");
      setEcShare("");
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

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">Verteilmodell</label>
          <select
            value={ecDisModel}
            onChange={(e) => setEcDisModel(e.target.value)}
            disabled={disabled || loading}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:bg-slate-50"
          >
            <option value="S">S — Statisch</option>
            <option value="D">D — Dynamisch</option>
          </select>
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
          <label className="block text-sm font-medium text-slate-700 mb-1.5">Teilnahmefaktor (%)</label>
          <input
            type="number"
            step="0.01"
            min="0"
            max="100"
            value={factor}
            onChange={(e) => setFactor(e.target.value)}
            placeholder="100"
            disabled={disabled || loading}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:bg-slate-50"
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-slate-700 mb-1.5">
          Anteil % (ECShare, optional)
        </label>
        <input
          type="number"
          step="0.0001"
          min="0"
          max="100"
          value={ecShare}
          onChange={(e) => setEcShare(e.target.value)}
          placeholder="33.3300"
          disabled={disabled || loading}
          className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:bg-slate-50"
        />
      </div>

      <button
        type="submit"
        disabled={disabled || loading || !zaehlpunkt}
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
  netzbetreiberId,
  onSuccess,
}: {
  eegId: string;
  disabled: boolean;
  netzbetreiberId: string;
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
    if (netzbetreiberId && zaehlpunkt.length >= 8 && zaehlpunkt.substring(0, 8) !== netzbetreiberId) {
      setError(`Zählpunkt-Präfix „${zaehlpunkt.substring(0, 8)}" passt nicht zum konfigurierten Netzbetreiber „${netzbetreiberId}"`);
      setLoading(false);
      return;
    }
    try {
      const res = await fetch(`/api/eegs/${eegId}/eda/teilnahmefaktor`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          zaehlpunkt,
          participation_factor: parseFloat(factor),
          share_type: "GC",
          ec_dis_model: "D",
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

// ── Zählpunktdaten nachfordern (EC_REQ_PT) ────────────────────────────────────

function ZaehlerstandsgangForm({
  eegId,
  disabled,
  netzbetreiberId,
  onSuccess,
}: {
  eegId: string;
  disabled: boolean;
  netzbetreiberId: string;
  onSuccess: () => void;
}) {
  const [zaehlpunkt, setZaehlpunkt] = useState("");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(false);
    if (netzbetreiberId && zaehlpunkt.length >= 8 && zaehlpunkt.substring(0, 8) !== netzbetreiberId) {
      setError(`Zählpunkt-Präfix „${zaehlpunkt.substring(0, 8)}" passt nicht zum konfigurierten Netzbetreiber „${netzbetreiberId}"`);
      setLoading(false);
      return;
    }

    try {
      const res = await fetch(`/api/eegs/${eegId}/eda/zaehlerstandsgang`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          zaehlpunkt,
          date_from: dateFrom,
          date_to: dateTo,
        }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      setSuccess(true);
      setZaehlpunkt("");
      setDateFrom("");
      setDateTo("");
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
        Zählpunktdaten (Messwerte) für einen Zeitraum beim Netzbetreiber nachfordern (EC_REQ_PT).
      </p>

      {success && (
        <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-green-800 text-sm">
          Anfrage wurde in die Warteschlange aufgenommen und wird übermittelt.
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
          className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-teal-500 disabled:bg-slate-50 disabled:text-slate-400"
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Von *
          </label>
          <input
            type="date"
            value={dateFrom}
            onChange={(e) => setDateFrom(e.target.value)}
            required
            disabled={disabled || loading}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-teal-500 disabled:bg-slate-50"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Bis *
          </label>
          <input
            type="date"
            value={dateTo}
            onChange={(e) => setDateTo(e.target.value)}
            required
            disabled={disabled || loading}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-teal-500 disabled:bg-slate-50"
          />
        </div>
      </div>

      <button
        type="submit"
        disabled={disabled || loading || !zaehlpunkt || !dateFrom || !dateTo}
        className="px-5 py-2.5 bg-teal-700 text-white text-sm font-medium rounded-lg hover:bg-teal-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        {loading ? "Wird gesendet…" : "Daten anfordern"}
      </button>
    </form>
  );
}

// ── Zählpunktliste anfordern (EC_PODLIST) ────────────────────────────────────

function PODListForm({
  eegId,
  disabled,
  onSuccess,
}: {
  eegId: string;
  disabled: boolean;
  onSuccess: () => void;
}) {
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
        body: JSON.stringify({}),
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
        ANFORDERUNG_ECP (CPRequest 01.12) — Aktuelle Zählpunktliste vom Netzbetreiber anfordern.
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
  netzbetreiberId,
  onSuccess,
}: {
  eegId: string;
  disabled: boolean;
  netzbetreiberId: string;
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
    if (netzbetreiberId && zaehlpunkt.length >= 8 && zaehlpunkt.substring(0, 8) !== netzbetreiberId) {
      setError(`Zählpunkt-Präfix „${zaehlpunkt.substring(0, 8)}" passt nicht zum konfigurierten Netzbetreiber „${netzbetreiberId}"`);
      setLoading(false);
      return;
    }

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
