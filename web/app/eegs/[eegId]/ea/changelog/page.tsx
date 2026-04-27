"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

interface ChangelogEntry {
  id: string;
  buchung_id: string;
  buchungsnr: string;
  beschreibung: string;
  operation: string;
  changed_at: string;
  changed_by: string;
  reason?: string;
}

const OP_LABELS: Record<string, string> = {
  create: "Erstellt",
  update: "Geändert",
  delete: "Gelöscht",
};

const OP_COLORS: Record<string, string> = {
  create: "bg-green-50 text-green-700 border-green-200",
  update: "bg-blue-50 text-blue-700 border-blue-200",
  delete: "bg-red-50 text-red-700 border-red-200",
};

function fmtDateTime(s: string): string {
  try {
    return new Date(s).toLocaleString("de-AT", {
      day: "2-digit", month: "2-digit", year: "numeric",
      hour: "2-digit", minute: "2-digit",
    });
  } catch { return s; }
}

export default function ChangelogPage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;

  const [entries, setEntries] = useState<ChangelogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [operation, setOperation] = useState("");
  const [von, setVon] = useState("");
  const [bis, setBis] = useState("");
  const [offset, setOffset] = useState(0);
  const limit = 100;

  async function load(newOffset = 0) {
    setLoading(true);
    setOffset(newOffset);
    const q = new URLSearchParams({ limit: String(limit), offset: String(newOffset) });
    if (operation) q.set("operation", operation);
    if (von) q.set("von", von);
    if (bis) q.set("bis", bis);
    const res = await fetch(`/api/eegs/${eegId}/ea/changelog?${q}`);
    if (res.ok) setEntries(await res.json());
    setLoading(false);
  }

  useEffect(() => { load(0); }, [eegId, operation, von, bis]);

  const selectClass = "px-3 py-2 border border-slate-300 rounded-lg text-slate-900 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500";
  const inputClass = "px-3 py-2 border border-slate-300 rounded-lg text-slate-900 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500";

  return (
    <div className="p-8">
      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">Übersicht</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea`} className="text-sm text-slate-500 hover:text-slate-700">E/A-Buchhaltung</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea/buchungen`} className="text-sm text-slate-500 hover:text-slate-700">Journal</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Audit-Trail</span>
      </div>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Audit-Trail</h1>
          <p className="text-slate-500 mt-1 text-sm">Alle Buchungsänderungen gemäß BAO §131</p>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-xl border border-slate-200 p-4 mb-4 flex flex-wrap gap-3 items-end">
        <div>
          <label className="block text-xs font-medium text-slate-700 mb-1">Vorgang</label>
          <select className={selectClass} value={operation} onChange={(e) => setOperation(e.target.value)}>
            <option value="">Alle</option>
            <option value="create">Erstellt</option>
            <option value="update">Geändert</option>
            <option value="delete">Gelöscht</option>
          </select>
        </div>
        <div>
          <label className="block text-xs font-medium text-slate-700 mb-1">Von</label>
          <input type="date" className={inputClass} value={von} onChange={(e) => setVon(e.target.value)} />
        </div>
        <div>
          <label className="block text-xs font-medium text-slate-700 mb-1">Bis</label>
          <input type="date" className={inputClass} value={bis} onChange={(e) => setBis(e.target.value)} />
        </div>
        <button onClick={() => { setVon(""); setBis(""); setOperation(""); }} className="px-3 py-2 text-sm text-slate-600 border border-slate-300 rounded-lg hover:bg-slate-50">
          Zurücksetzen
        </button>
      </div>

      {/* Table */}
      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-slate-500 text-sm">Wird geladen…</div>
        ) : entries.length === 0 ? (
          <div className="p-8 text-center text-slate-500 text-sm">Keine Einträge gefunden.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 bg-slate-50">
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase">Zeitpunkt</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase">Vorgang</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase">Buchung</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase">Benutzer</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase">Grund</th>
                  <th className="px-4 py-3"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {entries.map((e) => (
                  <tr key={e.id} className="hover:bg-slate-50">
                    <td className="px-4 py-3 text-slate-700 text-xs whitespace-nowrap">{fmtDateTime(e.changed_at)}</td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex px-2 py-0.5 rounded border text-xs font-medium ${OP_COLORS[e.operation] || "bg-slate-50 text-slate-600 border-slate-200"}`}>
                        {OP_LABELS[e.operation] || e.operation}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="font-mono text-xs text-slate-500">{e.buchungsnr}</span>
                      {e.beschreibung && (
                        <span className="ml-2 text-slate-700 truncate max-w-xs inline-block">{e.beschreibung}</span>
                      )}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-400">{e.changed_by.slice(0, 8)}…</td>
                    <td className="px-4 py-3 text-slate-500 text-xs italic max-w-xs truncate">{e.reason || "—"}</td>
                    <td className="px-4 py-3 text-right">
                      <Link
                        href={`/eegs/${eegId}/ea/buchungen/${e.buchung_id}`}
                        className="text-xs text-blue-600 hover:underline"
                      >
                        Buchung
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Pagination */}
      {entries.length === limit && (
        <div className="flex justify-end mt-3">
          <button
            onClick={() => load(offset + limit)}
            className="px-4 py-2 text-sm text-slate-600 border border-slate-300 rounded-lg hover:bg-slate-50"
          >
            Nächste {limit} →
          </button>
        </div>
      )}
      {offset > 0 && (
        <div className="flex justify-start mt-3">
          <button
            onClick={() => load(Math.max(0, offset - limit))}
            className="px-4 py-2 text-sm text-slate-600 border border-slate-300 rounded-lg hover:bg-slate-50"
          >
            ← Vorherige
          </button>
        </div>
      )}
    </div>
  );
}
