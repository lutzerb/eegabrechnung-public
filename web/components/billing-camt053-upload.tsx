"use client";

import { useState, useRef, useTransition } from "react";

interface BillingCamt053UploadProps {
  eegId: string;
  onImported?: () => void;
}

interface FileResult {
  matched: number;
  not_found: number;
  already_paid: number;
}

interface AggregateResult {
  matched: number;
  already_paid: number;
  files: { name: string; matched: number; already_paid: number; not_found: number }[];
  errors: string[];
}

export default function BillingCamt053Upload({ eegId, onImported }: BillingCamt053UploadProps) {
  const [result, setResult] = useState<AggregateResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [progress, setProgress] = useState("");
  const [isPending, startTransition] = useTransition();
  const fileRef = useRef<HTMLInputElement>(null);

  function handleUpload() {
    const files = Array.from(fileRef.current?.files || []);
    if (files.length === 0) {
      setError("Bitte mindestens eine CAMT.053 XML-Datei auswählen.");
      return;
    }
    setError(null);
    setResult(null);
    startTransition(async () => {
      const agg: AggregateResult = { matched: 0, already_paid: 0, files: [], errors: [] };
      try {
        for (let i = 0; i < files.length; i++) {
          const file = files[i];
          if (files.length > 1) setProgress(`(${i + 1}/${files.length}) ${file.name}`);
          const formData = new FormData();
          formData.append("file", file);
          const res = await fetch(`/api/eegs/${eegId}/billing/camt053`, {
            method: "POST",
            body: formData,
          });
          const data = await res.json();
          if (!res.ok) {
            agg.errors.push(`${file.name}: ${data.error || `Fehler ${res.status}`}`);
          } else {
            const r = data as FileResult;
            agg.matched += r.matched;
            agg.already_paid += r.already_paid;
            agg.files.push({ name: file.name, matched: r.matched, already_paid: r.already_paid, not_found: r.not_found });
          }
        }
        setResult(agg);
        if (fileRef.current) fileRef.current.value = "";
        onImported?.();
      } finally {
        setProgress("");
      }
    });
  }

  const buttonLabel = isPending
    ? (progress || "Importieren…")
    : "Zahlungsdaten von Bank importieren";

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2 flex-wrap">
        <input
          ref={fileRef}
          type="file"
          accept=".xml,application/xml,text/xml"
          multiple
          className="text-xs text-slate-600 file:mr-2 file:py-1 file:px-2.5 file:rounded file:border-0 file:text-xs file:font-medium file:bg-slate-100 file:text-slate-700 hover:file:bg-slate-200 cursor-pointer"
        />
        <button
          onClick={handleUpload}
          disabled={isPending}
          className="px-2.5 py-1 text-xs font-medium text-green-700 bg-green-50 border border-green-200 rounded hover:bg-green-100 transition-colors disabled:opacity-50 whitespace-nowrap"
        >
          {buttonLabel}
        </button>
      </div>

      {error && <p className="text-xs text-red-600">{error}</p>}

      {result && (
        <div className="text-xs text-slate-600">
          <span className="font-medium text-green-700">{result.matched} als bezahlt markiert</span>
          {result.already_paid > 0 && (
            <span className="ml-2 text-slate-500">{result.already_paid} bereits bezahlt</span>
          )}
          {result.files.length > 1 && (
            <ul className="mt-1 space-y-0.5 pl-1">
              {result.files.map((f, i) => (
                <li key={i} className="text-slate-500">
                  <span className="font-mono">{f.name}</span>
                  {" — "}{f.matched} neu{f.already_paid > 0 ? `, ${f.already_paid} bereits bezahlt` : ""}
                  {f.not_found > 0 && <span className="text-orange-500"> ({f.not_found} nicht gefunden)</span>}
                </li>
              ))}
            </ul>
          )}
          {result.errors.map((e, i) => (
            <p key={i} className="text-red-600 mt-0.5">{e}</p>
          ))}
        </div>
      )}
    </div>
  );
}
