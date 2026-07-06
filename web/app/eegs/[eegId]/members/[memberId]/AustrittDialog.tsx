"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

interface Props {
  eegId: string;
  memberId: string;
  memberName: string;
}

export default function AustrittDialog({ eegId, memberId, memberName }: Props) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [austrittDatum, setAustrittDatum] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // min = tomorrow (local date)
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  const tomorrowStr = tomorrow.toISOString().slice(0, 10);

  function handleOpen() {
    setAustrittDatum(tomorrowStr);
    setError(null);
    setOpen(true);
  }

  function handleClose() {
    if (loading) return;
    setOpen(false);
    setError(null);
  }

  async function handleSubmit() {
    if (!austrittDatum) {
      setError("Bitte ein Austrittsdatum wählen.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(
        `/api/eegs/${eegId}/members/${memberId}/austritt`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ austritt_datum: austrittDatum }),
        }
      );
      if (!res.ok) {
        let msg = `Fehler (HTTP ${res.status})`;
        try {
          const body = await res.json();
          msg = body.error || body.message || msg;
        } catch {
          // ignore
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
        className="px-4 py-2 text-sm font-medium bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors"
      >
        Mitglied abmelden
      </button>

      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          {/* Backdrop */}
          <div
            className="absolute inset-0 bg-black/40"
            onClick={handleClose}
          />

          {/* Dialog */}
          <div className="relative bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6 space-y-4">
            <h2 className="text-lg font-semibold text-slate-900">
              Mitglied abmelden
            </h2>

            <p className="text-sm text-slate-600">
              <strong>{memberName}</strong> wird zum gewählten Austrittsdatum auf{" "}
              <span className="font-medium text-orange-700">INAKTIV</span> gesetzt.
              Für alle noch aktiven Zählpunkte wird ein EDA-Widerruf (CM_REV_SP)
              eingereicht.
            </p>

            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">
                Austrittsdatum
              </label>
              <input
                type="date"
                min={tomorrowStr}
                value={austrittDatum}
                onChange={(e) => setAustrittDatum(e.target.value)}
                className="block w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
              />
              <p className="text-xs text-slate-400 mt-1">
                Muss mindestens morgen sein (EDA-Anforderung).
              </p>
            </div>

            {error && (
              <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
                {error}
              </div>
            )}

            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={handleClose}
                disabled={loading}
                className="px-4 py-2 text-sm font-medium border border-slate-300 text-slate-700 rounded-lg hover:bg-slate-50 transition-colors disabled:opacity-50"
              >
                Abbrechen
              </button>
              <button
                onClick={handleSubmit}
                disabled={loading || !austrittDatum}
                className="px-4 py-2 text-sm font-medium bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors disabled:opacity-50"
              >
                {loading ? "Wird verarbeitet…" : "Abmelden"}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
