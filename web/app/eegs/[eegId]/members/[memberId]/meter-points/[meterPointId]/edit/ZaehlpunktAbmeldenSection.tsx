"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

export function ZaehlpunktAbmeldenSection({
  eegId,
  zaehlpunkt,
}: {
  eegId: string;
  zaehlpunkt: string;
}) {
  const router = useRouter();
  const [consentEnd, setConsentEnd] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  // today in Vienna time (YYYY-MM-DD) for min on date picker
  const todayVienna = new Date()
    .toLocaleDateString("sv-SE", { timeZone: "Europe/Vienna" });

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!consentEnd) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/eegs/${eegId}/eda/widerruf`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ zaehlpunkt, consent_end: consentEnd }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      setSuccess(true);
      router.refresh();
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="p-6 border-t border-slate-200">
      <h2 className="text-base font-semibold text-slate-900 mb-1">Zählpunkt abmelden (EDA)</h2>
      <p className="text-sm text-slate-500 mb-4">
        Sendet eine Abmeldenachricht (CM_REV_SP) an den Netzbetreiber. Das Abmeldedatum wird
        automatisch gesetzt, sobald der Netzbetreiber bestätigt.
      </p>

      {success ? (
        <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-green-800 text-sm">
          Abmeldung wurde in die Warteschlange aufgenommen und wird übermittelt.
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-4 max-w-sm">
          {error && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-800 text-sm">
              {error}
            </div>
          )}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1.5">
              Wirksamkeitsdatum <span className="text-red-500">*</span>
            </label>
            <input
              type="date"
              required
              min={todayVienna}
              value={consentEnd}
              onChange={(e) => setConsentEnd(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
            <p className="text-xs text-slate-400 mt-1">
              Frühestens heute, höchstens 30 Arbeitstage in der Zukunft (EDA-Regelwerk).
            </p>
          </div>
          <button
            type="submit"
            disabled={loading || !consentEnd}
            className="px-5 py-2.5 bg-red-600 text-white text-sm font-medium rounded-lg hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {loading ? "Wird gesendet…" : "Zählpunkt abmelden"}
          </button>
        </form>
      )}
    </div>
  );
}
