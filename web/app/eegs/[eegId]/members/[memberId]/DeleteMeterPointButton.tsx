"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

interface Props {
  eegId: string;
  meterPointId: string;
  meterId: string;           // Zählpunkt-String (AT0...) — needed for Widerruf
  anmeldungStatus?: string | null;
}

// Widerruf (CM_REV_SP) is required when the meter point is fully or partially confirmed.
function needsWiderruf(status?: string | null): boolean {
  return status === "confirmed" || status === "first_confirmed";
}

export default function DeleteMeterPointButton({ eegId, meterPointId, meterId, anmeldungStatus }: Props) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const today = new Date().toISOString().slice(0, 10);
  const [abmeldedatum, setAbmeldedatum] = useState(today);

  const withWiderruf = needsWiderruf(anmeldungStatus);

  function handleOpen() {
    setError(null);
    setAbmeldedatum(today);
    setOpen(true);
  }

  async function handleDelete() {
    setLoading(true);
    setError(null);
    try {
      // Step 1: if the meter point is confirmed/partially confirmed, send a Widerruf (CM_REV_SP) first.
      if (withWiderruf) {
        const wRes = await fetch(`/api/eegs/${eegId}/eda/widerruf`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ zaehlpunkt: meterId, consent_end: abmeldedatum }),
        });
        if (!wRes.ok) {
          let msg = `Widerruf fehlgeschlagen (HTTP ${wRes.status})`;
          try {
            const body = await wRes.json();
            msg = body.error || body.message || msg;
          } catch {
            // ignore parse errors
          }
          setError(msg);
          return;
        }
      }

      // Step 2: delete the meter point from the database.
      const res = await fetch(`/api/eegs/${eegId}/meter-points/${meterPointId}`, {
        method: "DELETE",
      });
      if (!res.ok) {
        let msg = `Fehler beim Löschen (HTTP ${res.status})`;
        try {
          const body = await res.json();
          msg = body.error || body.message || msg;
        } catch {
          // ignore parse errors
        }
        setError(msg);
        return;
      }

      setOpen(false);
      router.refresh();
    } catch (e: unknown) {
      const err = e as Error;
      setError(err.message || "Unbekannter Fehler");
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <button
        onClick={handleOpen}
        className="px-2 py-1 text-xs text-red-600 hover:bg-red-50 rounded transition-colors"
      >
        Löschen
      </button>

      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/40"
            onClick={() => { if (!loading) setOpen(false); }}
          />
          <div className="relative bg-white rounded-xl shadow-xl w-full max-w-sm mx-4 p-6 space-y-4">
            <h2 className="text-lg font-semibold text-slate-900">Zählpunkt löschen</h2>

            <p className="text-sm text-slate-600">
              Zählpunkt{" "}
              <span className="font-mono font-medium text-slate-800">{meterId}</span>
              {withWiderruf
                ? " ist beim Netzbetreiber angemeldet. Vor dem Löschen wird ein EDA-Widerruf (CM_REV_SP) mit dem gewählten Abmeldedatum gesendet."
                : " wird unwiderruflich gelöscht."}
            </p>

            {withWiderruf && (
              <div>
                <label className="block text-xs font-medium text-slate-700 mb-1">
                  Abmeldedatum
                </label>
                <input
                  type="date"
                  min={today}
                  value={abmeldedatum}
                  onChange={(e) => setAbmeldedatum(e.target.value)}
                  className="block w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-red-500 focus:border-transparent"
                />
                <p className="text-xs text-slate-400 mt-1">
                  Frühestens heute, höchstens 30 Arbeitstage in der Zukunft (EDA-Regelwerk).
                </p>
              </div>
            )}

            {error && (
              <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
                {error}
              </div>
            )}

            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setOpen(false)}
                disabled={loading}
                className="px-4 py-2 text-sm font-medium border border-slate-300 text-slate-700 rounded-lg hover:bg-slate-50 transition-colors disabled:opacity-50"
              >
                Abbrechen
              </button>
              <button
                onClick={handleDelete}
                disabled={loading || (withWiderruf && !abmeldedatum)}
                className="px-4 py-2 text-sm font-medium bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
              >
                {loading
                  ? (withWiderruf ? "Wird abgemeldet und gelöscht…" : "Wird gelöscht…")
                  : (withWiderruf ? "Widerruf senden und löschen" : "Löschen")}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
