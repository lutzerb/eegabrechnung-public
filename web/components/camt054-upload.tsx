"use client";

import { useState, useRef, useTransition } from "react";

interface Camt054UploadProps {
  eegId: string;
  onImported?: () => void;
}

interface ImportResult {
  matched: number;
  not_found: number;
  already_returned: number;
}

export default function Camt054Upload({ eegId, onImported }: Camt054UploadProps) {
  const [result, setResult] = useState<ImportResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();
  const fileRef = useRef<HTMLInputElement>(null);

  function handleUpload() {
    const file = fileRef.current?.files?.[0];
    if (!file) {
      setError("Bitte eine CAMT.054 XML-Datei auswählen.");
      return;
    }
    setError(null);
    setResult(null);
    startTransition(async () => {
      try {
        const formData = new FormData();
        formData.append("file", file);
        const res = await fetch(`/api/eegs/${eegId}/sepa/camt054`, {
          method: "POST",
          body: formData,
        });
        const data = await res.json();
        if (!res.ok) {
          setError(data.error || `Fehler ${res.status}`);
          return;
        }
        setResult(data as ImportResult);
        if (fileRef.current) fileRef.current.value = "";
        onImported?.();
      } catch (err: unknown) {
        setError((err as { message?: string }).message || "Upload fehlgeschlagen.");
      }
    });
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2 flex-wrap">
        <input
          ref={fileRef}
          type="file"
          accept=".xml,application/xml,text/xml"
          className="text-xs text-slate-600 file:mr-2 file:py-1 file:px-2.5 file:rounded file:border-0 file:text-xs file:font-medium file:bg-slate-100 file:text-slate-700 hover:file:bg-slate-200 cursor-pointer"
        />
        <button
          onClick={handleUpload}
          disabled={isPending}
          className="px-2.5 py-1 text-xs font-medium text-teal-700 bg-teal-50 border border-teal-200 rounded hover:bg-teal-100 transition-colors disabled:opacity-50 whitespace-nowrap"
        >
          {isPending ? "Importieren…" : "CAMT.054 importieren"}
        </button>
      </div>

      {error && (
        <p className="text-xs text-red-600">{error}</p>
      )}

      {result && (
        <p className="text-xs text-slate-600">
          <span className="font-medium text-green-700">{result.matched} erkannt</span>
          {result.already_returned > 0 && (
            <span className="ml-2 text-slate-500">{result.already_returned} bereits erfasst</span>
          )}
          {result.not_found > 0 && (
            <span className="ml-2 text-orange-600">{result.not_found} nicht gefunden</span>
          )}
        </p>
      )}
    </div>
  );
}
