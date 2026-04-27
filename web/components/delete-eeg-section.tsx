"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

interface Props {
  eegId: string;
  eegName: string;
}

export function DeleteEEGSection({ eegId, eegName }: Props) {
  const router = useRouter();
  const [confirmName, setConfirmName] = useState("");
  const [confirmPhrase, setConfirmPhrase] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const PHRASE = "unwiderruflich löschen";
  const nameOk = confirmName === eegName;
  const phraseOk = confirmPhrase.trim().toLowerCase() === PHRASE;
  const canDelete = nameOk && phraseOk && !deleting;

  async function handleDelete(e: React.FormEvent) {
    e.preventDefault();
    if (!canDelete) return;
    setDeleting(true);
    setError(null);
    try {
      const res = await fetch(`/api/eegs/${eegId}`, { method: "DELETE" });
      if (res.ok || res.status === 204) {
        router.push("/eegs?deleted=1");
      } else {
        const data = await res.json().catch(() => ({}));
        setError(data.error || "Löschen fehlgeschlagen.");
        setDeleting(false);
      }
    } catch {
      setError("Netzwerkfehler.");
      setDeleting(false);
    }
  }

  return (
    <div className="bg-white rounded-xl border border-red-200 p-6">
      <h2 className="text-base font-semibold text-red-700 mb-1">Energiegemeinschaft löschen</h2>
      <p className="text-xs text-slate-500 mb-4">
        Löscht diese Energiegemeinschaft samt <strong>allen Mitgliedern, Zählpunkten, Messdaten, Rechnungen und Tarifplänen</strong> dauerhaft.
        Diese Aktion kann nicht rückgängig gemacht werden.
      </p>

      <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg">
        <p className="text-xs text-red-800 font-medium">
          Erstelle zuerst ein Backup, falls du die Daten noch benötigst.
        </p>
      </div>

      <form onSubmit={handleDelete} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">
            1. EEG-Namen zur Bestätigung eingeben:{" "}
            <span className="font-mono text-slate-900">{eegName}</span>
          </label>
          <input
            type="text"
            value={confirmName}
            onChange={(e) => setConfirmName(e.target.value)}
            placeholder={eegName}
            autoComplete="off"
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-red-400 focus:ring-1 focus:ring-red-400 outline-none"
          />
          {confirmName.length > 0 && !nameOk && (
            <p className="text-xs text-red-600 mt-1">Name stimmt nicht überein.</p>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">
            2. Zur Bestätigung genau folgendes eingeben:{" "}
            <span className="font-mono text-slate-900">{PHRASE}</span>
          </label>
          <input
            type="text"
            value={confirmPhrase}
            onChange={(e) => setConfirmPhrase(e.target.value)}
            placeholder={PHRASE}
            autoComplete="off"
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-red-400 focus:ring-1 focus:ring-red-400 outline-none"
          />
          {confirmPhrase.length > 0 && !phraseOk && (
            <p className="text-xs text-red-600 mt-1">Phrase stimmt nicht überein.</p>
          )}
        </div>

        <button
          type="submit"
          disabled={!canDelete}
          className="px-4 py-2 text-sm bg-red-700 text-white font-medium rounded-lg hover:bg-red-800 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {deleting ? "Wird gelöscht…" : "Energiegemeinschaft dauerhaft löschen"}
        </button>
      </form>

      {error && (
        <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
          {error}
        </div>
      )}
    </div>
  );
}
