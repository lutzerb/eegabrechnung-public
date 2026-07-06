"use client";

import { useState, useRef } from "react";

interface Props {
  eegId: string;
  eegName: string;
}

export function BackupRestoreSection({ eegId, eegName }: Props) {
  const [confirmText, setConfirmText] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [restoring, setRestoring] = useState(false);
  const [result, setResult] = useState<{ ok: boolean; message: string } | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const confirmRequired = eegName;
  const confirmed = confirmText === confirmRequired;

  async function handleRestore(e: React.FormEvent) {
    e.preventDefault();
    if (!file || !confirmed) return;

    setRestoring(true);
    setResult(null);
    try {
      const fd = new FormData();
      fd.append("file", file);
      const res = await fetch(`/api/eegs/${eegId}/restore`, { method: "POST", body: fd });
      const data = await res.json();
      if (res.ok) {
        setResult({
          ok: true,
          message: `Wiederherstellung abgeschlossen: ${data.members} Mitglieder, ${data.meter_points} Zählpunkte, ${data.readings} Messwerte, ${data.invoices} Rechnungen.`,
        });
        setConfirmText("");
        setFile(null);
        if (fileInputRef.current) fileInputRef.current.value = "";
      } else {
        setResult({ ok: false, message: data.error || "Fehler bei der Wiederherstellung." });
      }
    } catch {
      setResult({ ok: false, message: "Netzwerkfehler." });
    } finally {
      setRestoring(false);
    }
  }

  return (
    <div className="bg-white rounded-xl border border-slate-200 p-6">
      <h2 className="text-base font-semibold text-slate-900 mb-1">Backup & Wiederherstellung</h2>
      <p className="text-xs text-slate-500 mb-6">
        Exportiere alle EEG-Daten (Mitglieder, Zählpunkte, Messwerte, Rechnungen, Tarifpläne) als JSON-Datei oder stelle einen früheren Stand vollständig wieder her.
        Eine Wiederherstellung <strong>überschreibt alle bestehenden Daten</strong> dieser Energiegemeinschaft unwiderruflich.
      </p>

      {/* Download */}
      <div className="mb-6">
        <h3 className="text-sm font-medium text-slate-700 mb-2">Backup herunterladen</h3>
        <a
          href={`/api/eegs/${eegId}/backup`}
          download
          className="inline-flex items-center gap-2 px-4 py-2 text-sm bg-slate-100 border border-slate-300 text-slate-700 rounded-lg hover:bg-slate-200 transition-colors font-medium"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          Backup herunterladen
        </a>
      </div>

      {/* Restore */}
      <div className="border-t border-slate-200 pt-6">
        <h3 className="text-sm font-medium text-slate-700 mb-1">Aus Backup wiederherstellen</h3>
        <div className="mb-4 p-3 bg-amber-50 border border-amber-200 rounded-lg">
          <p className="text-xs text-amber-800 font-medium">Achtung: Alle aktuellen Daten dieser EEG werden gelöscht und durch den Backup-Stand ersetzt.</p>
        </div>

        <form onSubmit={handleRestore} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Backup-Datei (.json)
            </label>
            <input
              ref={fileInputRef}
              type="file"
              accept=".json,application/json"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              className="block text-sm text-slate-700 file:mr-4 file:py-1.5 file:px-3 file:rounded file:border-0 file:text-xs file:font-medium file:bg-blue-50 file:text-blue-700 hover:file:bg-blue-100"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Zur Bestätigung den EEG-Namen eingeben:{" "}
              <span className="font-mono text-slate-900">{confirmRequired}</span>
            </label>
            <input
              type="text"
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              placeholder={confirmRequired}
              className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none"
            />
            {confirmText.length > 0 && !confirmed && (
              <p className="text-xs text-red-600 mt-1">Name stimmt nicht überein.</p>
            )}
          </div>

          <button
            type="submit"
            disabled={!file || !confirmed || restoring}
            className="px-4 py-2 text-sm bg-red-600 text-white font-medium rounded-lg hover:bg-red-700 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {restoring ? "Wird wiederhergestellt…" : "Jetzt wiederherstellen"}
          </button>
        </form>

        {result && (
          <div className={`mt-4 p-3 rounded-lg text-sm ${result.ok ? "bg-green-50 border border-green-200 text-green-800" : "bg-red-50 border border-red-200 text-red-700"}`}>
            {result.message}
          </div>
        )}
      </div>
    </div>
  );
}
