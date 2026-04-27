"use client";

import { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

export default function AnnualReportPage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;

  const currentYear = new Date().getFullYear();
  const lastYear = currentYear - 1;

  const pad = (n: number) => String(n).padStart(2, "0");
  const lastDayOf = (y: number, m: number) => new Date(y, m, 0).getDate();

  // Member reference date — default: last day of previous year
  const [dateYear, setDateYear] = useState(lastYear);
  const [dateMonth, setDateMonth] = useState(12);
  const [dateDay, setDateDay] = useState(31);

  // Period for energy & billing — default: full previous year
  const [fromYear, setFromYear] = useState(lastYear);
  const [fromMonth, setFromMonth] = useState(1);
  const [toYear, setToYear] = useState(lastYear);
  const [toMonth, setToMonth] = useState(12);

  const [loading, setLoading] = useState(false);

  const yearRange = Array.from({ length: 6 }, (_, i) => currentYear - i);
  const monthNames = [
    "", "Jänner", "Februar", "März", "April", "Mai", "Juni",
    "Juli", "August", "September", "Oktober", "November", "Dezember",
  ];

  const dateStr = `${dateYear}-${pad(dateMonth)}-${pad(Math.min(dateDay, lastDayOf(dateYear, dateMonth)))}`;
  const fromStr = `${fromYear}-${pad(fromMonth)}-01`;
  const toStr   = `${toYear}-${pad(toMonth)}-${pad(lastDayOf(toYear, toMonth))}`;

  async function handleDownload() {
    setLoading(true);
    try {
      const url = `/api/eegs/${eegId}/reports/annual?date=${dateStr}&from=${fromStr}&to=${toStr}`;
      const res = await fetch(url);
      if (!res.ok) {
        const txt = await res.text();
        alert("Fehler beim Export: " + txt);
        return;
      }
      const blob = await res.blob();
      const disposition = res.headers.get("Content-Disposition") || "";
      const match = disposition.match(/filename="([^"]+)"/);
      const filename = match ? match[1] : `jahresbericht_${fromStr}_${toStr}.xlsx`;
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = filename;
      a.click();
      URL.revokeObjectURL(a.href);
    } finally {
      setLoading(false);
    }
  }

  const inputCls = "px-3 py-2 border border-slate-300 rounded-lg text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500";
  const labelCls = "block text-xs font-medium text-slate-500 mb-1";

  return (
    <div className="p-8 max-w-2xl">
      <div className="mb-6 text-sm text-slate-500">
        <Link href={`/eegs/${eegId}/reports`} className="hover:text-slate-700">Berichte</Link>
        <span className="mx-2 text-slate-300">/</span>
        <span className="text-slate-900 font-medium">Jahresbericht</span>
      </div>

      <h1 className="text-2xl font-bold text-slate-900 mb-1">Jahresbericht</h1>
      <p className="text-slate-500 text-sm mb-8">
        XLSX-Export mit Mitgliederliste (Stichtag) und Energie-/Abrechnungsdaten (Zeitraum).
      </p>

      <div className="space-y-6">
        {/* Stichtag Mitglieder */}
        <div className="bg-white rounded-xl border border-slate-200 p-5">
          <h2 className="text-sm font-semibold text-slate-700 mb-1">Mitglieder-Stichtag</h2>
          <p className="text-xs text-slate-400 mb-4">
            Welche Mitglieder waren zu diesem Datum aktiv (Beitritt ≤ Datum &lt; Austritt)?
          </p>
          <div className="flex flex-wrap gap-3 items-end">
            <div>
              <label className={labelCls}>Jahr</label>
              <select value={dateYear} onChange={e => setDateYear(+e.target.value)} className={inputCls}>
                {yearRange.map(y => <option key={y} value={y}>{y}</option>)}
              </select>
            </div>
            <div>
              <label className={labelCls}>Monat</label>
              <select value={dateMonth} onChange={e => { setDateMonth(+e.target.value); setDateDay(d => Math.min(d, lastDayOf(dateYear, +e.target.value))); }} className={inputCls}>
                {monthNames.slice(1).map((n, i) => <option key={i+1} value={i+1}>{n}</option>)}
              </select>
            </div>
            <div>
              <label className={labelCls}>Tag</label>
              <select value={dateDay} onChange={e => setDateDay(+e.target.value)} className={inputCls}>
                {Array.from({ length: lastDayOf(dateYear, dateMonth) }, (_, i) => i + 1).map(d => (
                  <option key={d} value={d}>{d}</option>
                ))}
              </select>
            </div>
            <div className="text-sm text-slate-500 pb-2 font-mono">{dateStr}</div>
          </div>
        </div>

        {/* Zeitraum Energie & Abrechnung */}
        <div className="bg-white rounded-xl border border-slate-200 p-5">
          <h2 className="text-sm font-semibold text-slate-700 mb-1">Zeitraum Energie & Abrechnung</h2>
          <p className="text-xs text-slate-400 mb-4">
            Energiemengen aus Messdaten und Rechnungssummen werden für diesen Zeitraum aggregiert.
          </p>
          <div className="grid grid-cols-2 gap-6">
            <div>
              <p className="text-xs font-medium text-slate-500 mb-2">Von</p>
              <div className="flex gap-2">
                <div>
                  <label className={labelCls}>Jahr</label>
                  <select value={fromYear} onChange={e => setFromYear(+e.target.value)} className={inputCls}>
                    {yearRange.map(y => <option key={y} value={y}>{y}</option>)}
                  </select>
                </div>
                <div>
                  <label className={labelCls}>Monat</label>
                  <select value={fromMonth} onChange={e => setFromMonth(+e.target.value)} className={inputCls}>
                    {monthNames.slice(1).map((n, i) => <option key={i+1} value={i+1}>{n}</option>)}
                  </select>
                </div>
              </div>
              <div className="text-xs text-slate-400 mt-1 font-mono">{fromStr}</div>
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 mb-2">Bis</p>
              <div className="flex gap-2">
                <div>
                  <label className={labelCls}>Jahr</label>
                  <select value={toYear} onChange={e => setToYear(+e.target.value)} className={inputCls}>
                    {yearRange.map(y => <option key={y} value={y}>{y}</option>)}
                  </select>
                </div>
                <div>
                  <label className={labelCls}>Monat</label>
                  <select value={toMonth} onChange={e => setToMonth(+e.target.value)} className={inputCls}>
                    {monthNames.slice(1).map((n, i) => <option key={i+1} value={i+1}>{n}</option>)}
                  </select>
                </div>
              </div>
              <div className="text-xs text-slate-400 mt-1 font-mono">{toStr}</div>
            </div>
          </div>
        </div>

        {/* Vorschau & Download */}
        <div className="bg-slate-50 rounded-xl border border-slate-200 p-5">
          <h2 className="text-sm font-semibold text-slate-700 mb-3">Bericht</h2>
          <div className="text-sm text-slate-600 space-y-1 mb-5">
            <div className="flex gap-2">
              <span className="text-slate-400 w-40">Blatt 1 — Mitglieder</span>
              <span>aktive Mitglieder am <span className="font-mono font-medium">{dateStr}</span></span>
            </div>
            <div className="flex gap-2">
              <span className="text-slate-400 w-40">Blatt 2 — Energie</span>
              <span>Mengen & Rechnungssummen <span className="font-mono font-medium">{fromStr} – {toStr}</span></span>
            </div>
            <div className="flex gap-2">
              <span className="text-slate-400 w-40">Format</span>
              <span>XLSX (Microsoft Excel)</span>
            </div>
          </div>
          <button
            onClick={handleDownload}
            disabled={loading}
            className="px-5 py-2.5 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors disabled:opacity-50"
          >
            {loading ? "Wird erstellt…" : "XLSX herunterladen"}
          </button>
        </div>
      </div>
    </div>
  );
}
