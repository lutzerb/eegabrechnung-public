"use client";

import React, { useState, useMemo } from "react";
import type { EDAProcess } from "@/lib/api";
import { parseEdaErrorCodes } from "@/lib/eda-error-codes";
import { EDA_PROCESS_TYPE_LABELS, EDA_PROCESS_STATUS_LABELS, EDA_PROCESS_STATUS_STYLES } from "@/lib/eda-status-labels";

interface Props {
  processes: EDAProcess[];
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

function formatDateShort(dateStr: string | undefined): string {
  if (!dateStr) return "—";
  try {
    return new Date(dateStr).toLocaleDateString("de-AT", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
    });
  } catch {
    return dateStr;
  }
}

function ProcessStatusBadge({ status }: { status: string }) {
  const cls = EDA_PROCESS_STATUS_STYLES[status] ?? "bg-slate-50 text-slate-600";
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${cls}`}>
      {EDA_PROCESS_STATUS_LABELS[status] ?? status}
    </span>
  );
}

function deadlineStyle(deadline_at: string | undefined): { dot: string; text: string; label: string } {
  if (!deadline_at) return { dot: "bg-slate-300", text: "text-slate-400", label: "—" };
  const now = new Date();
  const dl = new Date(deadline_at);
  const diffMs = dl.getTime() - now.getTime();
  const diffDays = diffMs / (1000 * 60 * 60 * 24);
  if (diffMs < 0 || diffDays < 3) {
    return { dot: "bg-red-500", text: "text-red-600 font-medium", label: formatDateShort(deadline_at) };
  }
  if (diffDays <= 7) {
    return { dot: "bg-yellow-400", text: "text-yellow-600 font-medium", label: formatDateShort(deadline_at) };
  }
  return { dot: "bg-green-500", text: "text-green-600", label: formatDateShort(deadline_at) };
}

const TYPE_LABELS = EDA_PROCESS_TYPE_LABELS;

const STATUS_OPTIONS = [
  { value: "", label: "Alle Status" },
  ...Object.entries(EDA_PROCESS_STATUS_LABELS).map(([value, label]) => ({ value, label })),
];

const TYPE_OPTIONS = [
  { value: "", label: "Alle Typen" },
  ...Object.entries(EDA_PROCESS_TYPE_LABELS)
    .filter(([value]) => value !== "EC_PODLIST") // not used as a process_type in this table's data
    .map(([value, label]) => ({ value, label })),
];

// Mirrors the dashboard "offene EDA-Prozesse mit ablaufender Frist" alert (web/app/eegs/[eegId]/page.tsx)
const OPEN_STATUSES = ["pending", "sent", "first_confirmed", "confirmed"];
const URGENT_DAYS_THRESHOLD = 7;

function isUrgent(proc: EDAProcess): boolean {
  if (!OPEN_STATUSES.includes(proc.status) || !proc.deadline_at) return false;
  const diffDays = (new Date(proc.deadline_at).getTime() - Date.now()) / (1000 * 60 * 60 * 24);
  return diffDays < URGENT_DAYS_THRESHOLD;
}

export function EDAProcessesTable({ processes }: Props) {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [typeFilter, setTypeFilter] = useState("");
  const [urgentOnly, setUrgentOnly] = useState(false);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const urgentCount = useMemo(() => processes.filter(isUrgent).length, [processes]);

  const filtered = useMemo(() => {
    const q = search.toLowerCase();
    const result = processes.filter((proc) => {
      if (urgentOnly && !isUrgent(proc)) return false;
      if (statusFilter && proc.status !== statusFilter) return false;
      if (typeFilter && proc.process_type !== typeFilter) return false;
      if (q) {
        const haystack = [proc.zaehlpunkt, proc.process_type, proc.error_msg, proc.member_name]
          .filter(Boolean)
          .join(" ")
          .toLowerCase();
        if (!haystack.includes(q)) return false;
      }
      return true;
    });
    if (urgentOnly) {
      result.sort((a, b) => new Date(a.deadline_at!).getTime() - new Date(b.deadline_at!).getTime());
    }
    return result;
  }, [processes, statusFilter, typeFilter, search, urgentOnly]);

  if (processes.length === 0) {
    return (
      <div className="px-6 py-8 text-center text-slate-400 text-sm">
        Keine Prozesse vorhanden.
      </div>
    );
  }

  return (
    <>
      {/* Filters */}
      <div className="px-4 py-3 border-b border-slate-100 flex flex-wrap gap-3 items-center bg-slate-50/50">
        <input
          type="search"
          placeholder="Suche nach Zählpunkt oder Mitglied…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="flex-1 min-w-[180px] text-sm border border-slate-200 rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
        />
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="text-sm border border-slate-200 rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
        >
          {STATUS_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>
        <select
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value)}
          className="text-sm border border-slate-200 rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
        >
          {TYPE_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>
        <button
          onClick={() => setUrgentOnly((v) => !v)}
          title={`Offene Prozesse mit Frist < ${URGENT_DAYS_THRESHOLD} Tagen bzw. überfällig`}
          className={`flex items-center gap-1.5 text-sm rounded-lg px-3 py-1.5 border transition-colors ${
            urgentOnly
              ? "bg-red-50 border-red-300 text-red-700"
              : "bg-white border-slate-200 text-slate-600 hover:bg-slate-50"
          }`}
        >
          <span className={`w-2 h-2 rounded-full flex-shrink-0 ${urgentOnly ? "bg-red-500" : "bg-slate-300"}`} />
          Nur dringend
          {urgentCount > 0 && (
            <span className={`ml-0.5 px-1.5 rounded-full text-xs font-medium ${
              urgentOnly ? "bg-red-200 text-red-800" : "bg-slate-100 text-slate-500"
            }`}>
              {urgentCount}
            </span>
          )}
        </button>
        {(search || statusFilter || typeFilter || urgentOnly) && (
          <button
            onClick={() => { setSearch(""); setStatusFilter(""); setTypeFilter(""); setUrgentOnly(false); }}
            className="text-xs text-slate-500 hover:text-slate-700 underline"
          >
            Zurücksetzen
          </button>
        )}
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full text-sm min-w-[800px]">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-50">
              <th className="text-left px-6 py-3 font-medium text-slate-600">Typ</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Mitglied</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Zählpunkt</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Status</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Faktor</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Gültig ab</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Frist</th>
              <th className="text-left px-6 py-3 font-medium text-slate-600">Gestartet</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {filtered.length === 0 ? (
              <tr>
                <td colSpan={8} className="px-6 py-8 text-center text-slate-400 text-sm">
                  Keine Prozesse gefunden.
                </td>
              </tr>
            ) : (
              filtered.map((proc) => (
                <React.Fragment key={proc.id}>
                <tr
                  className={`transition-colors ${proc.error_msg ? "cursor-pointer" : ""} ${expandedId === proc.id ? "bg-red-50/40" : "hover:bg-slate-50"}`}
                  onClick={() => proc.error_msg ? setExpandedId(expandedId === proc.id ? null : proc.id) : undefined}
                >
                  <td className="px-6 py-3.5">
                    <span className="font-mono text-xs text-slate-600">
                      {TYPE_LABELS[proc.process_type] ?? proc.process_type}
                    </span>
                  </td>
                  <td className="px-6 py-3.5 text-xs text-slate-700 max-w-[160px] truncate" title={proc.member_name}>
                    {proc.member_name || <span className="text-slate-400">—</span>}
                  </td>
                  <td className="px-6 py-3.5 font-mono text-xs text-slate-600 whitespace-nowrap">
                    {proc.zaehlpunkt}
                  </td>
                  <td className="px-6 py-3.5">
                    <div className="flex flex-wrap items-center gap-1.5">
                      <ProcessStatusBadge status={proc.status} />
                      {proc.error_msg && parseEdaErrorCodes(proc.error_msg).map((parsed) => (
                        <span
                          key={parsed.code}
                          className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-mono font-semibold bg-red-100 text-red-800"
                        >
                          Code {parsed.code}
                        </span>
                      ))}
                    </div>
                    {proc.error_msg && (() => {
                      const codes = parseEdaErrorCodes(proc.error_msg);
                      if (codes.length > 0) {
                        return codes.map((parsed) => (
                          <p key={parsed.code} className="text-xs text-red-700 mt-1 leading-snug" title={parsed.detail}>
                            {parsed.label}
                          </p>
                        ));
                      }
                      // No structured response code found — fall back to the raw gateway text.
                      return (
                        <p className="text-xs text-red-700 mt-1 leading-snug" title={proc.error_msg}>
                          Gateway-Fehler
                        </p>
                      );
                    })()}
                  </td>
                  <td className="px-6 py-3.5 text-slate-600 text-xs tabular-nums">
                    {proc.participation_factor != null
                      ? `${proc.participation_factor.toLocaleString("de-AT", { maximumFractionDigits: 2 })} %`
                      : "—"}
                  </td>
                  <td className="px-6 py-3.5 text-slate-600 text-xs whitespace-nowrap">
                    {proc.valid_from ? formatDateShort(proc.valid_from) : "—"}
                  </td>
                  <td className="px-6 py-3.5 text-xs whitespace-nowrap">
                    {(() => {
                      const ds = deadlineStyle(proc.deadline_at);
                      if (!proc.deadline_at) return <span className="text-slate-400">—</span>;
                      return (
                        <span className={`inline-flex items-center gap-1.5 ${ds.text}`}>
                          <span className={`w-2 h-2 rounded-full flex-shrink-0 ${ds.dot}`} />
                          {ds.label}
                        </span>
                      );
                    })()}
                  </td>
                  <td className="px-6 py-3.5 text-slate-400 text-xs whitespace-nowrap">
                    {formatDate(proc.initiated_at)}
                    {proc.error_msg && (
                      <span className="ml-1 text-slate-300 text-xs">{expandedId === proc.id ? "▲" : "▼"}</span>
                    )}
                  </td>
                </tr>
                {expandedId === proc.id && proc.error_msg && (() => {
                  const codes = parseEdaErrorCodes(proc.error_msg);
                  return (
                    <tr key={`${proc.id}-detail`} className="bg-red-50/60 border-t border-red-100">
                      <td colSpan={8} className="px-6 py-4">
                        <div className="flex items-start gap-6 text-xs">
                          <div className="min-w-0 flex-1">
                            <p className="text-slate-500 font-medium mb-1">
                              {codes.length > 1 ? "Bedeutung (mehrere Codes)" : "Bedeutung"}
                            </p>
                            {codes.length > 0 ? (
                              <ul className="space-y-1.5">
                                {codes.map((parsed) => (
                                  <li key={parsed.code}>
                                    <span className="text-2xl font-mono font-bold text-red-700 mr-2 align-middle">
                                      {parsed.code}
                                    </span>
                                    <span className="text-red-800 font-medium text-sm align-middle">{parsed.label}</span>
                                  </li>
                                ))}
                              </ul>
                            ) : (
                              <p className="text-red-800 font-medium text-sm">Unbekannter Fehler</p>
                            )}
                            {codes.length > 0 && (
                              <p className="text-slate-500 mt-2">
                                Laut ebutilities.at Responsecodes (Kategorie Customer Processes)
                              </p>
                            )}
                          </div>
                          <div className="min-w-0 max-w-sm">
                            <p className="text-slate-500 font-medium mb-1">Rohe Fehlermeldung</p>
                            <p className="text-slate-600 break-words leading-relaxed">{proc.error_msg}</p>
                          </div>
                        </div>
                      </td>
                    </tr>
                  );
                })()}
                </React.Fragment>
              ))
            )}
          </tbody>
        </table>
      </div>

      {(search || statusFilter || typeFilter) && (
        <div className="px-6 py-3 border-t border-slate-100 bg-slate-50">
          <p className="text-xs text-slate-500">
            {filtered.length} von {processes.length} Prozessen
          </p>
        </div>
      )}
    </>
  );
}
