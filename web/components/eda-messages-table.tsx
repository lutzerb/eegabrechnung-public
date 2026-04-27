"use client";

import { useState, useMemo } from "react";
import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import type { EDAMessage } from "@/lib/api";
import { byMarktpartnerID } from "@/lib/netzbetreiber";

interface Props {
  messages: EDAMessage[];
  eegId: string;
  totalCount: number;
  page: number;
  pageSize: number;
}

function formatDate(dateStr: string | undefined): string {
  if (!dateStr) return "—";
  try {
    return new Date(dateStr).toLocaleDateString("de-AT", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return dateStr;
  }
}

function labelAddress(code: string | undefined): string {
  if (!code) return "—";
  const nb = byMarktpartnerID(code);
  return nb ? nb.name : code;
}

function DirectionBadge({ direction }: { direction: string }) {
  const isInbound = direction === "inbound";
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
      isInbound ? "bg-blue-50 text-blue-700" : "bg-orange-50 text-orange-700"
    }`}>
      {isInbound ? "Eingang" : "Ausgang"}
    </span>
  );
}

function MessageStatusBadge({ status }: { status: string }) {
  if (status === "error") {
    return <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-50 text-red-700">Fehler</span>;
  }
  if (status === "ack") {
    return <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-50 text-green-700">Quittiert</span>;
  }
  if (status === "sent") {
    return <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-700">Gesendet</span>;
  }
  if (status === "processed") {
    return <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-50 text-green-700">Verarbeitet</span>;
  }
  return <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-50 text-yellow-700">Ausstehend</span>;
}

const PROCESS_LABELS: Record<string, string> = {
  DATEN_CRMSG:      "Energiedaten (Antwort)",
  ANTWORT_PT:       "Edanet-Eingangsbestätigung",
  EC_REQ_ONL:       "Anmeldung",
  CM_REV_SP:        "Widerruf (EEG)",
  CM_REV_CUS:       "Widerruf durch Kunde",
  CM_REV_IMP:       "Widerruf durch NB (Unmöglichkeit)",
  EC_PRTFACT_CHG:   "Teilnahmefaktor",
  EC_REQ_PT:        "Zählerstandsgang",
  ZUSTIMMUNG_ECON:  "Zustimmung",
  ABLEHNUNG_ECON:   "Ablehnung",
  ANTWORT_ECON:     "Zwischenbestätigung",
  ABSCHLUSS_ECON:   "Abschluss",
  SENDEN_ECP:       "Zählpunktliste",
  ERSTE_ANM:        "Erst-Bestätigung",
  FINALE_ANM:       "Final-Bestätigung",
  ABLEHNUNG_ANM:    "Ablehnung",
  ANFORDERUNG_ECON: "Zustimmungsanfrage",
  ANFORDERUNG_ECP:  "Listanforderung",
  ECMPList:         "Zählpunktliste",
};

function typeLabel(process: string, messageType: string): string {
  const code = process || messageType;
  return PROCESS_LABELS[code] ?? code;
}

const DIRECTION_OPTIONS = [
  { value: "", label: "Alle Richtungen" },
  { value: "inbound", label: "Eingang" },
  { value: "outbound", label: "Ausgang" },
];

const PROCESS_OPTIONS = [
  { value: "", label: "Alle Typen" },
  { value: "DATEN_CRMSG", label: "Energiedaten (Antwort)" },
  { value: "ANTWORT_PT", label: "Edanet-Eingangsbestätigung" },
  { value: "EC_REQ_ONL", label: "Anmeldung" },
  { value: "CM_REV_SP", label: "Widerruf (EEG)" },
  { value: "CM_REV_CUS", label: "Widerruf durch Kunde" },
  { value: "CM_REV_IMP", label: "Widerruf durch NB (Unmöglichkeit)" },
  { value: "EC_PRTFACT_CHG", label: "Teilnahmefaktor" },
  { value: "EC_REQ_PT", label: "Zählerstandsgang" },
  { value: "ZUSTIMMUNG_ECON", label: "Zustimmung" },
  { value: "ABLEHNUNG_ECON", label: "Ablehnung" },
  { value: "ANTWORT_ECON", label: "Zwischenbestätigung" },
  { value: "ABSCHLUSS_ECON", label: "Abschluss" },
  { value: "SENDEN_ECP", label: "Zählpunktliste" },
  { value: "ERSTE_ANM", label: "Erst-Bestätigung" },
  { value: "FINALE_ANM", label: "Final-Bestätigung" },
];

function ExpandedRow({ msg, eegId }: { msg: EDAMessage; eegId: string }) {
  return (
    <tr className="bg-slate-50/80 border-b border-slate-100">
      <td colSpan={7} className="px-6 py-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
          {/* Addresses */}
          <div className="space-y-2">
            {msg.from_address && (
              <div>
                <span className="font-medium text-slate-500">Von:</span>{" "}
                <span className="text-slate-800" title={msg.from_address}>{labelAddress(msg.from_address)}</span>
                {byMarktpartnerID(msg.from_address) && (
                  <span className="font-mono text-slate-400 ml-1">({msg.from_address})</span>
                )}
              </div>
            )}
            {msg.to_address && (
              <div>
                <span className="font-medium text-slate-500">An:</span>{" "}
                <span className="text-slate-800" title={msg.to_address}>{labelAddress(msg.to_address)}</span>
                {byMarktpartnerID(msg.to_address) && (
                  <span className="font-mono text-slate-400 ml-1">({msg.to_address})</span>
                )}
              </div>
            )}
            {msg.message_id && (
              <div>
                <span className="font-medium text-slate-500">Message-ID:</span>{" "}
                <span className="font-mono text-slate-600 break-all">{msg.message_id}</span>
              </div>
            )}
            {msg.processed_at && (
              <div>
                <span className="font-medium text-slate-500">Verarbeitet:</span>{" "}
                <span className="text-slate-600">{formatDate(msg.processed_at)}</span>
              </div>
            )}
          </div>

          {/* Email body (only if it's actual text, not XML) */}
          {msg.body && !msg.body.startsWith("<?xml") && (
            <div>
              <span className="font-medium text-slate-500 block mb-1">E-Mail Text:</span>
              <pre className="whitespace-pre-wrap text-slate-700 bg-white border border-slate-200 rounded p-2 max-h-40 overflow-y-auto text-xs leading-relaxed">
                {msg.body}
              </pre>
            </div>
          )}
        </div>

        {/* Error message */}
        {msg.error_msg && (
          <div className="mt-3 p-2 bg-red-50 border border-red-200 rounded text-xs text-red-700">
            <span className="font-medium">Fehler:</span> {msg.error_msg}
          </div>
        )}

        {/* XML download */}
        <div className="mt-3">
          <a
            href={`/api/eegs/${eegId}/eda/messages/${msg.id}/xml`}
            download={`eda-${msg.process || msg.message_type}-${msg.id.slice(0, 8)}.xml`}
            className="inline-flex items-center gap-1.5 text-xs text-blue-600 hover:text-blue-800 hover:underline"
          >
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
                d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            XML herunterladen
          </a>
        </div>
      </td>
    </tr>
  );
}

const PAGE_SIZE_OPTIONS = [50, 100, 200, 500];

export function EDAMessagesTable({ messages, eegId, totalCount, page, pageSize }: Props) {
  const [search, setSearch] = useState("");
  const [direction, setDirection] = useState("");
  const [processType, setProcessType] = useState("");
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const filtered = useMemo(() => {
    const q = search.toLowerCase();
    return messages.filter((msg) => {
      if (direction && msg.direction !== direction) return false;
      if (processType && msg.process !== processType && msg.message_type !== processType) return false;
      if (q) {
        const haystack = [msg.subject, msg.message_id, msg.process, msg.message_type, msg.error_msg, msg.from_address, msg.to_address, msg.body]
          .filter(Boolean)
          .join(" ")
          .toLowerCase();
        if (!haystack.includes(q)) return false;
      }
      return true;
    });
  }, [messages, direction, processType, search]);

  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));
  const currentPage = Math.min(page, totalPages);
  const pageStart = totalCount === 0 ? 0 : (currentPage - 1) * pageSize + 1;
  const pageEnd = totalCount === 0 ? 0 : Math.min(currentPage * pageSize, totalCount);

  const buildHref = (nextPage: number, nextPageSize = pageSize) => {
    const params = new URLSearchParams(searchParams.toString());
    params.set("tab", "nachrichten");
    params.set("page", String(nextPage));
    params.set("pageSize", String(nextPageSize));
    return `${pathname}?${params.toString()}`;
  };

  const pageNumbers = (() => {
    const pages: number[] = [];
    const start = Math.max(1, currentPage - 2);
    const end = Math.min(totalPages, currentPage + 2);
    for (let i = start; i <= end; i++) {
      pages.push(i);
    }
    return pages;
  })();

  if (messages.length === 0) {
    return (
      <div className="px-6 py-10 text-center">
        <svg className="mx-auto w-10 h-10 text-slate-300 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
            d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
        </svg>
        <p className="text-sm text-slate-500">Keine EDA-Nachrichten vorhanden.</p>
      </div>
    );
  }

  return (
    <>
      {/* Filters */}
      <div className="px-4 py-3 border-b border-slate-100 flex flex-wrap gap-3 items-center bg-slate-50/50">
        <input
          type="search"
          placeholder="Suche (Betreff, Adresse, Zählpunkt…)"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="flex-1 min-w-[200px] text-sm border border-slate-200 rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
        />
        <select
          value={direction}
          onChange={(e) => setDirection(e.target.value)}
          className="text-sm border border-slate-200 rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
        >
          {DIRECTION_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>
        <select
          value={processType}
          onChange={(e) => setProcessType(e.target.value)}
          className="text-sm border border-slate-200 rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
        >
          {PROCESS_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>
        {(search || direction || processType) && (
          <button
            onClick={() => { setSearch(""); setDirection(""); setProcessType(""); }}
            className="text-xs text-slate-500 hover:text-slate-700 underline"
          >
            Zurücksetzen
          </button>
        )}
        <div className="ml-auto flex items-center gap-2">
          <label className="text-xs text-slate-500" htmlFor="eda-messages-page-size">
            Pro Seite
          </label>
          <select
            id="eda-messages-page-size"
            value={pageSize}
            onChange={(e) => {
              const nextPageSize = Number(e.target.value) || 100;
              window.location.href = buildHref(1, nextPageSize);
            }}
            className="text-sm border border-slate-200 rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
          >
            {PAGE_SIZE_OPTIONS.map((size) => (
              <option key={size} value={size}>{size}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full text-sm min-w-[700px]">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-50">
              <th className="text-left px-4 py-3 font-medium text-slate-600 w-36">Datum / Uhrzeit</th>
              <th className="text-left px-4 py-3 font-medium text-slate-600 w-24">Richtung</th>
              <th className="text-left px-4 py-3 font-medium text-slate-600 w-44">Typ</th>
              <th className="text-left px-4 py-3 font-medium text-slate-600">Betreff / Zählpunkt</th>
              <th className="text-left px-4 py-3 font-medium text-slate-600 w-36">Von / An</th>
              <th className="text-left px-4 py-3 font-medium text-slate-600 w-28">Status</th>
              <th className="px-4 py-3 w-8"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {filtered.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-slate-400 text-sm">
                  Keine Nachrichten gefunden.
                </td>
              </tr>
            ) : (
              filtered.flatMap((msg) => {
                const isExpanded = expandedId === msg.id;
                const hasDetails = !!(msg.from_address || msg.to_address || msg.body || msg.error_msg || msg.message_id);
                const rows = [
                  <tr
                    key={msg.id}
                    className={`transition-colors ${msg.status === "error" ? "bg-red-50/40" : ""} ${hasDetails ? "cursor-pointer hover:bg-slate-50" : ""}`}
                    onClick={() => hasDetails && setExpandedId(isExpanded ? null : msg.id)}
                  >
                    <td className="px-4 py-3 text-slate-500 whitespace-nowrap text-xs">
                      {formatDate(msg.created_at)}
                    </td>
                    <td className="px-4 py-3">
                      <DirectionBadge direction={msg.direction} />
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs text-slate-800">{typeLabel(msg.process, msg.message_type)}</span>
                        {msg.process && msg.process !== msg.message_type && (
                          <span className="font-mono text-xs text-slate-400">{msg.process}</span>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-xs max-w-xs">
                      {msg.error_msg && !isExpanded ? (
                        <span className="text-red-700" title={msg.error_msg}>
                          {msg.error_msg.slice(0, 80)}{msg.error_msg.length > 80 ? "…" : ""}
                        </span>
                      ) : (
                        <span className="text-slate-700 truncate block" title={msg.subject}>
                          {msg.subject || "—"}
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-xs">
                      {msg.direction === "inbound" ? (
                        <span className="text-slate-500 truncate block max-w-[120px]" title={msg.from_address}>
                          {labelAddress(msg.from_address)}
                        </span>
                      ) : (
                        <span className="text-slate-500 truncate block max-w-[120px]" title={msg.to_address}>
                          {labelAddress(msg.to_address)}
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <MessageStatusBadge status={msg.status} />
                    </td>
                    <td className="px-4 py-3 text-slate-400">
                      {hasDetails && (
                        <svg
                          className={`w-4 h-4 transition-transform ${isExpanded ? "rotate-180" : ""}`}
                          fill="none" viewBox="0 0 24 24" stroke="currentColor"
                        >
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 9l-7 7-7-7" />
                        </svg>
                      )}
                    </td>
                  </tr>,
                ];
                if (isExpanded) {
                  rows.push(<ExpandedRow key={`${msg.id}-exp`} msg={msg} eegId={eegId} />);
                }
                return rows;
              })
            )}
          </tbody>
        </table>
      </div>

      <div className="px-6 py-3 border-t border-slate-100 bg-slate-50">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-xs text-slate-500">
            {filtered.length !== messages.length
              ? `${filtered.length} gefilterte Einträge auf dieser Seite`
              : `${pageStart} bis ${pageEnd} von ${totalCount} Nachrichten`}
          </p>
          <div className="flex items-center gap-1.5">
            <Link
              href={buildHref(Math.max(1, currentPage - 1))}
              aria-disabled={currentPage <= 1}
              className={`inline-flex items-center rounded-lg px-3 py-1.5 text-sm ${
                currentPage <= 1
                  ? "pointer-events-none bg-slate-100 text-slate-400"
                  : "bg-white border border-slate-200 text-slate-700 hover:bg-slate-50"
              }`}
            >
              Zurück
            </Link>
            {pageNumbers[0] > 1 && (
              <>
                <Link
                  href={buildHref(1)}
                  className="inline-flex items-center rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-50"
                >
                  1
                </Link>
                {pageNumbers[0] > 2 && <span className="px-1 text-sm text-slate-400">…</span>}
              </>
            )}
            {pageNumbers.map((n) => (
              <Link
                key={n}
                href={buildHref(n)}
                className={`inline-flex min-w-[38px] items-center justify-center rounded-lg px-3 py-1.5 text-sm ${
                  n === currentPage
                    ? "bg-blue-600 text-white"
                    : "border border-slate-200 bg-white text-slate-700 hover:bg-slate-50"
                }`}
              >
                {n}
              </Link>
            ))}
            {pageNumbers[pageNumbers.length - 1] < totalPages && (
              <>
                {pageNumbers[pageNumbers.length - 1] < totalPages - 1 && <span className="px-1 text-sm text-slate-400">…</span>}
                <Link
                  href={buildHref(totalPages)}
                  className="inline-flex items-center rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-50"
                >
                  {totalPages}
                </Link>
              </>
            )}
            <Link
              href={buildHref(Math.min(totalPages, currentPage + 1))}
              aria-disabled={currentPage >= totalPages}
              className={`inline-flex items-center rounded-lg px-3 py-1.5 text-sm ${
                currentPage >= totalPages
                  ? "pointer-events-none bg-slate-100 text-slate-400"
                  : "bg-white border border-slate-200 text-slate-700 hover:bg-slate-50"
              }`}
            >
              Weiter
            </Link>
          </div>
        </div>
      </div>
    </>
  );
}
