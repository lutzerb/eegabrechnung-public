"use client";

import { useState, useTransition } from "react";

const RETURN_REASON_CODES: { code: string; label: string }[] = [
  { code: "AC01", label: "AC01 – Ungültige Kontonummer" },
  { code: "AM04", label: "AM04 – Kein Guthaben / Kontodeckung unzureichend" },
  { code: "MD01", label: "MD01 – Kein Mandat" },
  { code: "MD06", label: "MD06 – Widerruf durch Zahler" },
  { code: "MD07", label: "MD07 – Konto geschlossen" },
  { code: "MS02", label: "MS02 – Abgelehnt durch Kontoinhaber" },
  { code: "MS03", label: "MS03 – Abgelehnt durch Bank (kein Grund)" },
  { code: "SL01", label: "SL01 – Dienst nicht verfügbar" },
  { code: "ARDT", label: "ARDT – Lastschrift bereits zurückgebucht" },
  { code: "BE04", label: "BE04 – Fehlende oder ungültige Gläubiger-Adresse" },
  { code: "DNOR", label: "DNOR – Bank nicht registriert" },
  { code: "FOCR", label: "FOCR – Widerruf durch Zahlungsempfänger" },
  { code: "FR01", label: "FR01 – Verdacht auf Betrug" },
  { code: "NARR", label: "NARR – Sonstiger Grund" },
];

interface SepaReturnDialogProps {
  eegId: string;
  invoiceId: string;
  /** Display label shown in the dialog title */
  invoiceRef: string;
  /** Existing return data (if any) */
  existingReturn?: {
    sepa_return_at?: string;
    sepa_return_reason?: string;
    sepa_return_note?: string;
  };
  onClose: () => void;
  onSaved: () => void;
}

function formatDateInput(isoString?: string): string {
  if (!isoString) return new Date().toISOString().slice(0, 10);
  return isoString.slice(0, 10);
}

export default function SepaReturnDialog({
  eegId,
  invoiceId,
  invoiceRef,
  existingReturn,
  onClose,
  onSaved,
}: SepaReturnDialogProps) {
  const [returnAt, setReturnAt] = useState(
    formatDateInput(existingReturn?.sepa_return_at)
  );
  const [reason, setReason] = useState(existingReturn?.sepa_return_reason ?? "");
  const [note, setNote] = useState(existingReturn?.sepa_return_note ?? "");
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();

  async function handleSave() {
    setError(null);
    startTransition(async () => {
      try {
        const res = await fetch(
          `/api/eegs/${eegId}/invoices/${invoiceId}/sepa-return`,
          {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ return_at: returnAt, reason, note }),
          }
        );
        if (!res.ok) {
          const body = await res.json().catch(() => ({}));
          setError(body.error || `Fehler ${res.status}`);
          return;
        }
        onSaved();
      } catch (err: unknown) {
        setError((err as { message?: string }).message || "Unbekannter Fehler");
      }
    });
  }

  async function handleClear() {
    setError(null);
    startTransition(async () => {
      try {
        const res = await fetch(
          `/api/eegs/${eegId}/invoices/${invoiceId}/sepa-return`,
          {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({}),
          }
        );
        if (!res.ok) {
          const body = await res.json().catch(() => ({}));
          setError(body.error || `Fehler ${res.status}`);
          return;
        }
        onSaved();
      } catch (err: unknown) {
        setError((err as { message?: string }).message || "Unbekannter Fehler");
      }
    });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-md">
        <div className="px-6 py-4 border-b border-slate-100 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-900">
            Rücklastschrift {existingReturn?.sepa_return_at ? "bearbeiten" : "erfassen"}
          </h2>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600 transition-colors"
            aria-label="Schließen"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="px-6 py-4 space-y-4">
          <p className="text-sm text-slate-500">
            Rechnung: <span className="font-medium text-slate-700">{invoiceRef}</span>
          </p>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Rücklastschrift-Datum
            </label>
            <input
              type="date"
              value={returnAt}
              onChange={(e) => setReturnAt(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Grund (SEPA-Rückgabegrund)
            </label>
            <select
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
            >
              <option value="">— Kein Grund gewählt —</option>
              {RETURN_REASON_CODES.map((r) => (
                <option key={r.code} value={r.code}>
                  {r.label}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Notiz (optional)
            </label>
            <textarea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              rows={2}
              placeholder="Z.B. Kontakt mit Mitglied aufnehmen..."
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
            />
          </div>

          {error && (
            <p className="text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">
              {error}
            </p>
          )}
        </div>

        <div className="px-6 py-4 border-t border-slate-100 flex items-center justify-between gap-2">
          <div>
            {existingReturn?.sepa_return_at && (
              <button
                onClick={handleClear}
                disabled={isPending}
                className="px-3 py-1.5 text-sm font-medium text-red-700 bg-red-50 border border-red-200 rounded-lg hover:bg-red-100 transition-colors disabled:opacity-50"
              >
                Rücklastschrift löschen
              </button>
            )}
          </div>
          <div className="flex gap-2">
            <button
              onClick={onClose}
              disabled={isPending}
              className="px-4 py-1.5 text-sm font-medium text-slate-700 bg-slate-100 rounded-lg hover:bg-slate-200 transition-colors disabled:opacity-50"
            >
              Abbrechen
            </button>
            <button
              onClick={handleSave}
              disabled={isPending}
              className="px-4 py-1.5 text-sm font-medium text-white bg-orange-600 rounded-lg hover:bg-orange-700 transition-colors disabled:opacity-50"
            >
              {isPending ? "Speichern…" : "Speichern"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
