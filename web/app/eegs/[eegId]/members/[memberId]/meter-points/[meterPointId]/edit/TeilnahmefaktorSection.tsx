"use client";

import { useState } from "react";

interface Props {
  eegId: string;
  zaehlpunkt: string;
  energyDirection: string; // CONSUMPTION | GENERATION
  currentFactor?: number;
  currentValidFrom?: string; // ISO date
}

export function TeilnahmefaktorSection({
  eegId,
  zaehlpunkt,
  energyDirection,
  currentFactor,
  currentValidFrom,
}: Props) {
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  const tomorrowStr = tomorrow.toISOString().slice(0, 10);

  const [factor, setFactor] = useState(currentFactor != null ? String(currentFactor) : "");
  const [validFrom, setValidFrom] = useState(tomorrowStr);
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
          ec_dis_model: "D",
          energy_direction: energyDirection,
          valid_from: validFrom,
        }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      setSuccess(true);
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  const inputCls = "w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-slate-50 disabled:text-slate-400";
  const labelCls = "block text-sm font-medium text-slate-700 mb-1.5";

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-900">Teilnahmefaktor (EDA)</h3>
          <p className="text-xs text-slate-400 mt-0.5">
            Nur 09:00–17:00 Uhr (Wien), einmal pro Tag pro Zählpunkt.
          </p>
        </div>
        <div className="flex items-center gap-4 text-sm">
          <div className="text-right">
            <span className="text-xs text-slate-500 block">Aktuell</span>
            <span className="font-semibold text-slate-900">
              {currentFactor != null ? `${currentFactor}%` : "—"}
            </span>
          </div>
          {currentValidFrom && (
            <div className="text-right">
              <span className="text-xs text-slate-500 block">Gültig ab</span>
              <span className="text-slate-700">
                {new Date(currentValidFrom).toLocaleDateString("de-AT")}
              </span>
            </div>
          )}
        </div>
      </div>

      {success && (
        <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-green-800 text-sm">
          Änderung in Warteschlange aufgenommen und wird an den Netzbetreiber übermittelt.
        </div>
      )}
      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-800 text-sm">
          {error}
        </div>
      )}

      <form onSubmit={submit} className="space-y-3">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className={labelCls}>Neuer Faktor (%) *</label>
            <div className="relative">
              <input
                type="number"
                min="0.001"
                max="100"
                step="0.001"
                required
                value={factor}
                onChange={(e) => setFactor(e.target.value)}
                disabled={loading}
                className={`${inputCls} pr-8`}
              />
              <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 text-sm">%</span>
            </div>
          </div>
          <div>
            <label className={labelCls}>Gültig ab *</label>
            <input
              type="date"
              required
              min={tomorrowStr}
              value={validFrom}
              onChange={(e) => setValidFrom(e.target.value)}
              disabled={loading}
              className={inputCls}
            />
          </div>
        </div>

        <button
          type="submit"
          disabled={loading || !factor}
          className="px-4 py-2 bg-amber-600 text-white text-sm font-medium rounded-lg hover:bg-amber-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? "Wird übermittelt…" : "Teilnahmefaktor ändern (EDA)"}
        </button>
      </form>
    </div>
  );
}
