"use client";

import { useMemo, useRef, useState } from "react";

interface Props {
  eegId: string;
  eegName: string;
  displayName?: string;
}

const SECTION_KEYS = [
  "members",
  "meter_points",
  "tariff_schedules",
  "billing_runs",
  "invoices",
  "participations",
  "readings",
] as const;

type SectionKey = (typeof SECTION_KEYS)[number];

const SECTION_LABELS: Record<SectionKey, string> = {
  members: "Mitglieder",
  meter_points: "Zählpunkte",
  tariff_schedules: "Tarife",
  billing_runs: "Abrechnungsläufe",
  invoices: "Rechnungen",
  participations: "Teilnahmen",
  readings: "Messwerte",
};

const STAMMDATEN_PRESET: SectionKey[] = ["members", "meter_points", "tariff_schedules", "participations"];

function allSectionsChecked(): Record<SectionKey, boolean> {
  return Object.fromEntries(SECTION_KEYS.map((k) => [k, true])) as Record<SectionKey, boolean>;
}

export function BackupRestoreSection({ eegId, eegName, displayName }: Props) {
  const hasAlias = !!displayName && displayName !== eegName;
  const [confirmText, setConfirmText] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [restoring, setRestoring] = useState(false);
  const [result, setResult] = useState<{ ok: boolean; message: string } | null>(null);
  const [missingSections, setMissingSections] = useState<SectionKey[]>([]);
  const [sections, setSections] = useState<Record<SectionKey, boolean>>(allSectionsChecked());
  const fileInputRef = useRef<HTMLInputElement>(null);

  const confirmRequired = eegName;
  const confirmed = confirmText === confirmRequired;

  const checkedKeys = SECTION_KEYS.filter((k) => sections[k]);
  const backupHref = useMemo(() => {
    if (checkedKeys.length === SECTION_KEYS.length) {
      return `/api/eegs/${eegId}/backup`;
    }
    return `/api/eegs/${eegId}/backup?sections=${checkedKeys.join(",")}`;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [eegId, sections]);

  function toggleSection(key: SectionKey) {
    setSections((prev) => ({ ...prev, [key]: !prev[key] }));
  }

  function applyStammdatenPreset() {
    const next = { ...allSectionsChecked() };
    for (const key of SECTION_KEYS) {
      next[key] = STAMMDATEN_PRESET.includes(key);
    }
    setSections(next);
  }

  function selectAllSections() {
    setSections(allSectionsChecked());
  }

  async function handleFileChange(selectedFile: File | null) {
    setFile(selectedFile);
    setMissingSections([]);
    if (!selectedFile) return;
    try {
      const text = await selectedFile.text();
      const parsed = JSON.parse(text);
      const includedInFile: string[] | undefined = parsed.included;
      if (includedInFile) {
        setMissingSections(SECTION_KEYS.filter((k) => !includedInFile.includes(k)));
      }
    } catch {
      // not valid JSON - let server-side validation report the real error on submit
    }
  }

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
        const restored = data.restored ?? {};
        const skipped: string[] = data.skipped ?? [];
        const restoredParts = SECTION_KEYS.filter((k) => k in restored).map(
          (k) => `${restored[k]} ${SECTION_LABELS[k]}`
        );
        let message = `Wiederherstellung abgeschlossen: ${restoredParts.join(", ")}.`;
        if (skipped.length > 0) {
          message += ` Nicht verändert: ${skipped.map((k) => SECTION_LABELS[k as SectionKey] ?? k).join(", ")}.`;
        }
        setResult({ ok: true, message });
        setConfirmText("");
        setFile(null);
        setMissingSections([]);
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
        Exportiere EEG-Daten (Mitglieder, Zählpunkte, Messwerte, Rechnungen, Tarifpläne) als JSON-Datei oder stelle einen früheren Stand wieder her.
        Eine Wiederherstellung <strong>überschreibt die im Backup enthaltenen Datenbereiche dieser Energiegemeinschaft unwiderruflich</strong>; nicht im Backup enthaltene Bereiche bleiben unverändert.
      </p>

      {/* Download */}
      <div className="mb-6">
        <h3 className="text-sm font-medium text-slate-700 mb-2">Backup herunterladen</h3>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-x-4 gap-y-1.5 mb-3">
          {SECTION_KEYS.map((key) => (
            <label key={key} className="flex items-center gap-2 text-sm text-slate-700">
              <input
                type="checkbox"
                checked={sections[key]}
                onChange={() => toggleSection(key)}
                className="rounded border-slate-300 text-blue-600 focus:ring-blue-500"
              />
              {SECTION_LABELS[key]}
            </label>
          ))}
        </div>
        <div className="flex items-center gap-2 mb-3">
          <button
            type="button"
            onClick={applyStammdatenPreset}
            className="text-xs px-2.5 py-1 bg-slate-100 border border-slate-300 text-slate-700 rounded-md hover:bg-slate-200 transition-colors font-medium"
          >
            Nur Stammdaten
          </button>
          <button
            type="button"
            onClick={selectAllSections}
            className="text-xs px-2.5 py-1 bg-slate-100 border border-slate-300 text-slate-700 rounded-md hover:bg-slate-200 transition-colors font-medium"
          >
            Alle auswählen
          </button>
        </div>
        <a
          href={backupHref}
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
          <p className="text-xs text-amber-800 font-medium">Achtung: Alle im Backup enthaltenen Datenbereiche dieser EEG werden gelöscht und durch den Backup-Stand ersetzt.</p>
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
              onChange={(e) => handleFileChange(e.target.files?.[0] ?? null)}
              className="block text-sm text-slate-700 file:mr-4 file:py-1.5 file:px-3 file:rounded file:border-0 file:text-xs file:font-medium file:bg-blue-50 file:text-blue-700 hover:file:bg-blue-100"
            />
          </div>

          {missingSections.length > 0 && (
            <div className="p-3 bg-blue-50 border border-blue-200 rounded-lg">
              <p className="text-xs text-blue-800">
                Dieses Backup enthält nicht: {missingSections.map((k) => SECTION_LABELS[k]).join(", ")}. Diese Bereiche bleiben bei dieser Wiederherstellung unverändert.
              </p>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Zur Bestätigung den EEG-Namen eingeben:{" "}
              <span className="font-mono text-slate-900">{confirmRequired}</span>
            </label>
            {hasAlias && (
              <p className="text-xs text-slate-500 mb-1">
                Hinweis: Angezeigter Name ist „{displayName}", zur Bestätigung ist aber der rechtliche Name „{eegName}" einzugeben.
              </p>
            )}
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
