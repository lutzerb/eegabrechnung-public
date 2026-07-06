"use client";

import { useState } from "react";

interface Props {
  eegId: string;
  billingRunId: string;
}

export function SepaDownloadButtons({ eegId, billingRunId }: Props) {
  const [error, setError] = useState<string | null>(null);

  const download = async (file: "pain001" | "pain008") => {
    setError(null);
    try {
      const res = await fetch(
        `/api/eegs/${eegId}/sepa/${file}?billing_run_id=${billingRunId}`
      );
      if (!res.ok) {
        const msg = await res.text();
        setError(msg || `Fehler ${res.status}`);
        return;
      }
      const blob = await res.blob();
      const cd = res.headers.get("Content-Disposition");
      const filename = cd?.match(/filename="([^"]+)"/)?.[1] ?? `${file}.xml`;
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      setError("Netzwerkfehler beim Download.");
    }
  };

  return (
    <div className="flex items-center gap-2 flex-wrap">
      <span className="text-xs text-slate-500 mr-1">SEPA:</span>
      <button
        onClick={() => download("pain001")}
        className="px-2.5 py-1 text-xs font-medium text-emerald-700 bg-emerald-50 border border-emerald-200 rounded hover:bg-emerald-100 transition-colors"
      >
        pain.001 (Gutschriften)
      </button>
      <button
        onClick={() => download("pain008")}
        className="px-2.5 py-1 text-xs font-medium text-blue-700 bg-blue-50 border border-blue-200 rounded hover:bg-blue-100 transition-colors"
      >
        pain.008 (Lastschriften)
      </button>
      {error && (
        <span className="text-xs text-red-600 ml-1">{error}</span>
      )}
    </div>
  );
}
