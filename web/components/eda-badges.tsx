"use client";

import { useState } from "react";
import { edaMessageStatusLabelAndStyle } from "@/lib/eda-status-labels";
import { formatXml } from "@/lib/format-xml";

// Shared by eda-messages-table.tsx and eda-processes-table.tsx so a message's direction/
// status badge looks identical whether it's shown in the flat Nachrichten tab or nested
// inside a process's message accordion.

export function DirectionBadge({ direction }: { direction: string }) {
  const isInbound = direction === "inbound";
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
      isInbound ? "bg-blue-50 text-blue-700" : "bg-orange-50 text-orange-700"
    }`}>
      {isInbound ? "Eingang" : "Ausgang"}
    </span>
  );
}

export function MessageStatusBadge({ status }: { status: string }) {
  const { label, cls } = edaMessageStatusLabelAndStyle(status);
  return <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${cls}`}>{label}</span>;
}

// "XML anzeigen" (lazy-fetched inline preview, toggled) + "XML herunterladen" (unchanged
// download link) for one EDA message. Shared by the flat Nachrichten tab and the per-process
// message accordion so both offer the same preview behavior.
export function XmlPreviewToggle({
  eegId,
  messageId,
  filenameHint,
  align = "start",
}: {
  eegId: string;
  messageId: string;
  filenameHint: string;
  align?: "start" | "end";
}) {
  const [xml, setXml] = useState<string | "loading" | "error" | null>(null);

  const toggleXml = async () => {
    if (xml !== null) {
      setXml(null);
      return;
    }
    setXml("loading");
    try {
      const res = await fetch(`/api/eegs/${eegId}/eda/messages/${messageId}/xml`);
      if (!res.ok) throw new Error(`status ${res.status}`);
      setXml(formatXml(await res.text()));
    } catch {
      setXml("error");
    }
  };

  return (
    <div>
      <div className={`flex items-center gap-3 ${align === "end" ? "justify-end" : ""}`}>
        <button
          onClick={toggleXml}
          className="text-xs text-blue-600 hover:text-blue-800 hover:underline"
        >
          {xml !== null ? "XML ausblenden" : "XML anzeigen"}
        </button>
        <a
          href={`/api/eegs/${eegId}/eda/messages/${messageId}/xml`}
          download={`eda-${filenameHint}-${messageId.slice(0, 8)}.xml`}
          className="inline-flex items-center gap-1.5 text-xs text-blue-600 hover:text-blue-800 hover:underline"
        >
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
              d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          XML herunterladen
        </a>
      </div>
      {xml === "loading" && (
        <p className="mt-2 text-xs text-slate-400">XML wird geladen…</p>
      )}
      {xml === "error" && (
        <p className="mt-2 text-xs text-red-600">XML konnte nicht geladen werden.</p>
      )}
      {xml && xml !== "loading" && xml !== "error" && (
        <pre className="mt-2 max-h-64 overflow-y-auto text-[11px] font-mono text-slate-700 bg-slate-50 border border-slate-200 rounded p-2 whitespace-pre-wrap break-all">
          {xml}
        </pre>
      )}
    </div>
  );
}
