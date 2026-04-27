"use client";

import { useState, useRef, DragEvent, ChangeEvent } from "react";
import { useSession } from "next-auth/react";

interface FileUploadSectionProps {
  eegId: string;
  type: "stammdaten" | "energiedaten";
  title: string;
  description: string;
  acceptedFormats: string;
  onImportComplete?: () => void;
}

interface PreviewResult {
  period_start?: string;
  period_end?: string;
  total_rows: number;
  new_rows: number;
  identical_rows: number;
  conflict_rows: number;
  skipped_meters: number;
}

interface ImportResult {
  rows_inserted?: number;
  rows_parsed?: number;
  members?: number;
  meter_points?: number;
  message?: string;
  errors?: string[];
  mode?: string;
}

export default function FileUploadSection({
  eegId,
  type,
  title,
  description,
  acceptedFormats,
  onImportComplete,
}: FileUploadSectionProps) {
  const { data: session } = useSession();
  const [isDragging, setIsDragging] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [preview, setPreview] = useState<PreviewResult | null>(null);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragging(true);
  };

  const handleDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragging(false);
  };

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragging(false);
    const dropped = e.dataTransfer.files[0];
    if (dropped) validateAndSetFile(dropped);
  };

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0];
    if (selected) validateAndSetFile(selected);
  };

  const validateAndSetFile = (f: File) => {
    const validExtensions = [".xlsx", ".xls"];
    const hasValidExt = validExtensions.some((ext) =>
      f.name.toLowerCase().endsWith(ext)
    );
    if (!hasValidExt) {
      setError("Nur XLSX- und XLS-Dateien werden akzeptiert.");
      return;
    }
    setFile(f);
    setError(null);
    setResult(null);
    setPreview(null);
  };

  // Stammdaten: simple single-step upload
  const handleUploadStammdaten = async () => {
    if (!file || !session?.accessToken) return;
    setLoading(true);
    setError(null);
    setResult(null);
    try {
      const formData = new FormData();
      formData.append("file", file);
      const res = await fetch(`/api/eegs/${eegId}/import/stammdaten`, {
        method: "POST",
        headers: { Authorization: `Bearer ${session.accessToken}` },
        body: formData,
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || body.message || `HTTP ${res.status}`);
      }
      const data: ImportResult = await res.json();
      setResult(data);
      setFile(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
    } catch (err: unknown) {
      setError((err as Error).message || "Fehler beim Importieren.");
    } finally {
      setLoading(false);
    }
  };

  // Energiedaten step 1: preview
  const handlePreview = async () => {
    if (!file || !session?.accessToken) return;
    setLoading(true);
    setError(null);
    setPreview(null);
    setResult(null);
    try {
      const formData = new FormData();
      formData.append("file", file);
      const res = await fetch(`/api/eegs/${eegId}/import/energiedaten/preview`, {
        method: "POST",
        headers: { Authorization: `Bearer ${session.accessToken}` },
        body: formData,
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || body.message || `HTTP ${res.status}`);
      }
      setPreview(await res.json());
    } catch (err: unknown) {
      setError((err as Error).message || "Fehler beim Analysieren.");
    } finally {
      setLoading(false);
    }
  };

  // Energiedaten step 2: actual import with chosen mode
  const handleImportEnergie = async (mode: "overwrite" | "skip") => {
    if (!file || !session?.accessToken) return;
    setLoading(true);
    setError(null);
    try {
      const formData = new FormData();
      formData.append("file", file);
      const res = await fetch(
        `/api/eegs/${eegId}/import/energiedaten?mode=${mode}`,
        {
          method: "POST",
          headers: { Authorization: `Bearer ${session.accessToken}` },
          body: formData,
        }
      );
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || body.message || `HTTP ${res.status}`);
      }
      const data: ImportResult = await res.json();
      setResult(data);
      setFile(null);
      setPreview(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
      onImportComplete?.();
    } catch (err: unknown) {
      setError((err as Error).message || "Fehler beim Importieren.");
    } finally {
      setLoading(false);
    }
  };

  const clearFile = () => {
    setFile(null);
    setError(null);
    setResult(null);
    setPreview(null);
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const fmtDate = (d?: string) =>
    d ? new Date(d).toLocaleDateString("de-AT") : "—";

  const hasOverlap =
    preview && (preview.identical_rows > 0 || preview.conflict_rows > 0);
  const noOverlap = preview && preview.identical_rows === 0 && preview.conflict_rows === 0;

  const iconColor = type === "stammdaten" ? "text-blue-700" : "text-green-700";
  const bgColor = type === "stammdaten" ? "bg-blue-50" : "bg-green-50";
  const borderActive =
    type === "stammdaten"
      ? "border-blue-400 bg-blue-50"
      : "border-green-400 bg-green-50";
  const buttonColor =
    type === "stammdaten"
      ? "bg-blue-700 hover:bg-blue-800"
      : "bg-green-700 hover:bg-green-800";

  return (
    <div className="bg-white rounded-xl border border-slate-200 p-6">
      <div className="flex items-start gap-4 mb-4">
        <div className={`w-10 h-10 rounded-lg ${bgColor} flex items-center justify-center flex-shrink-0`}>
          <svg className={`w-5 h-5 ${iconColor}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
              d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
          </svg>
        </div>
        <div>
          <h3 className="font-semibold text-slate-900">{title}</h3>
          <p className="text-sm text-slate-500 mt-0.5">{description}</p>
        </div>
      </div>

      {/* Success */}
      {result && (
        <div className="mb-4 p-4 bg-green-50 border border-green-200 rounded-lg">
          <div className="flex items-start gap-2">
            <svg className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div>
              <p className="font-medium text-green-800">Import erfolgreich</p>
              {result.message && <p className="text-sm text-green-700 mt-0.5">{result.message}</p>}
              <div className="flex flex-wrap gap-4 mt-2">
                {result.members !== undefined && (
                  <p className="text-sm text-green-700">
                    <span className="font-semibold">{result.members}</span> Mitglieder importiert
                  </p>
                )}
                {result.meter_points !== undefined && (
                  <p className="text-sm text-green-700">
                    <span className="font-semibold">{result.meter_points}</span> Zählpunkte importiert
                  </p>
                )}
                {result.rows_inserted !== undefined && (
                  <p className="text-sm text-green-700">
                    <span className="font-semibold">{result.rows_inserted}</span> von{" "}
                    {result.rows_parsed} Datensätzen importiert
                    {result.mode === "skip" && " (bestehende übersprungen)"}
                  </p>
                )}
              </div>
              {result.errors && result.errors.length > 0 && (
                <ul className="mt-2 space-y-0.5 list-disc list-inside">
                  {result.errors.map((e, i) => (
                    <li key={i} className="text-xs text-amber-700">{e}</li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <div className="flex items-start gap-2">
            <svg className="w-5 h-5 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div>
              <p className="font-medium">Fehler beim Import</p>
              <p className="text-sm mt-0.5">{error}</p>
            </div>
          </div>
        </div>
      )}

      {/* File selected */}
      {file ? (
        <div className="space-y-3">
          <div className="border border-slate-200 rounded-lg p-4 flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-slate-100 flex items-center justify-center flex-shrink-0">
              <svg className="w-5 h-5 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                  d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-slate-900 truncate">{file.name}</p>
              <p className="text-xs text-slate-500">{(file.size / 1024).toFixed(1)} KB</p>
            </div>
            <div className="flex gap-2 flex-shrink-0">
              {type === "stammdaten" ? (
                <button
                  onClick={handleUploadStammdaten}
                  disabled={loading}
                  className={`px-4 py-2 text-sm font-medium text-white rounded-lg ${buttonColor} disabled:opacity-50 disabled:cursor-not-allowed transition-colors`}
                >
                  {loading ? <Spinner /> : "Importieren"}
                </button>
              ) : (
                !preview && (
                  <button
                    onClick={handlePreview}
                    disabled={loading}
                    className={`px-4 py-2 text-sm font-medium text-white rounded-lg ${buttonColor} disabled:opacity-50 disabled:cursor-not-allowed transition-colors`}
                  >
                    {loading ? <Spinner label="Analysiere..." /> : "Analysieren"}
                  </button>
                )
              )}
              <button
                onClick={clearFile}
                disabled={loading}
                className="px-3 py-2 text-sm font-medium text-slate-600 border border-slate-300 rounded-lg hover:bg-slate-50 disabled:opacity-50 transition-colors"
              >
                Entfernen
              </button>
            </div>
          </div>

          {/* Preview result panel (energiedaten only) */}
          {preview && (
            <div className={`rounded-lg border p-4 ${
              hasOverlap
                ? "bg-amber-50 border-amber-200"
                : "bg-blue-50 border-blue-200"
            }`}>
              <p className={`font-medium mb-2 ${hasOverlap ? "text-amber-800" : "text-blue-800"}`}>
                {hasOverlap ? "Überschneidungen gefunden" : "Bereit zum Import"}
              </p>

              {preview.period_start && (
                <p className="text-xs text-slate-600 mb-3">
                  Zeitraum der Datei: <strong>{fmtDate(preview.period_start)}</strong> – <strong>{fmtDate(preview.period_end)}</strong>
                </p>
              )}

              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-3">
                <Stat label="Gesamt" value={preview.total_rows} />
                <Stat label="Neu" value={preview.new_rows} color="text-emerald-700" />
                <Stat label="Identisch" value={preview.identical_rows} color="text-slate-600" />
                <Stat label="Abweichend" value={preview.conflict_rows} color={preview.conflict_rows > 0 ? "text-red-600" : "text-slate-600"} />
              </div>
              {preview.skipped_meters > 0 && (
                <p className="text-xs text-amber-700 mb-3">
                  ⚠ {preview.skipped_meters} Zählpunkt(e) nicht gefunden und übersprungen.
                </p>
              )}

              {hasOverlap ? (
                <div className="space-y-2">
                  <p className="text-xs text-amber-700">
                    {preview.conflict_rows > 0
                      ? `${preview.conflict_rows} Datensätze haben abweichende Werte in der Datenbank.`
                      : `${preview.identical_rows} Datensätze sind bereits identisch vorhanden.`}
                    {" "}Wie soll verfahren werden?
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <button
                      onClick={() => handleImportEnergie("overwrite")}
                      disabled={loading}
                      className="px-3 py-1.5 text-sm font-medium text-white bg-amber-600 rounded hover:bg-amber-700 disabled:opacity-50 transition-colors"
                    >
                      {loading ? <Spinner /> : "Überschreiben"}
                    </button>
                    <button
                      onClick={() => handleImportEnergie("skip")}
                      disabled={loading}
                      className="px-3 py-1.5 text-sm font-medium text-amber-800 bg-white border border-amber-300 rounded hover:bg-amber-50 disabled:opacity-50 transition-colors"
                    >
                      {loading ? <Spinner /> : "Nur neue importieren"}
                    </button>
                    <button
                      onClick={clearFile}
                      disabled={loading}
                      className="px-3 py-1.5 text-sm font-medium text-slate-600 bg-white border border-slate-300 rounded hover:bg-slate-50 disabled:opacity-50 transition-colors"
                    >
                      Abbrechen
                    </button>
                  </div>
                </div>
              ) : (
                <button
                  onClick={() => handleImportEnergie("overwrite")}
                  disabled={loading}
                  className={`px-4 py-1.5 text-sm font-medium text-white rounded ${buttonColor} disabled:opacity-50 transition-colors`}
                >
                  {loading ? <Spinner /> : `${preview.new_rows} Datensätze importieren`}
                </button>
              )}
            </div>
          )}
        </div>
      ) : (
        /* Drop zone */
        <div
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          onClick={() => fileInputRef.current?.click()}
          className={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
            isDragging
              ? borderActive
              : "border-slate-300 hover:border-slate-400 hover:bg-slate-50"
          }`}
        >
          <svg className="mx-auto w-10 h-10 text-slate-400 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
              d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
          </svg>
          <p className="text-sm font-medium text-slate-700">
            Datei hierher ziehen oder{" "}
            <span className="text-blue-600 hover:text-blue-700">durchsuchen</span>
          </p>
          <p className="text-xs text-slate-400 mt-1">Akzeptierte Formate: {acceptedFormats}</p>
          <input
            ref={fileInputRef}
            type="file"
            accept={acceptedFormats}
            onChange={handleFileChange}
            className="hidden"
          />
        </div>
      )}
    </div>
  );
}

function Spinner({ label }: { label?: string }) {
  return (
    <span className="flex items-center gap-2">
      <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
      {label || "Importiere..."}
    </span>
  );
}

function Stat({ label, value, color = "text-slate-800" }: { label: string; value: number; color?: string }) {
  return (
    <div className="text-center">
      <p className={`text-xl font-bold ${color}`}>{value.toLocaleString("de-AT")}</p>
      <p className="text-xs text-slate-500 mt-0.5">{label}</p>
    </div>
  );
}
