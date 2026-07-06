"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

interface SaldenEintrag {
  konto_id: string;
  nummer: string;
  name: string;
  typ: string;
  einnahmen: number;
  ausgaben: number;
  saldo: number;
}

interface Jahresabschluss {
  jahr: number;
  total_einnahmen: number;
  total_ausgaben: number;
  ueberschuss: number;
  einnahmen: SaldenEintrag[] | null;
  ausgaben: SaldenEintrag[] | null;
}

function fmt(n: number): string {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(n);
}

export default function JahresabschlussPage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;
  const curYear = new Date().getFullYear();

  const [data, setData] = useState<Jahresabschluss | null>(null);
  const [loading, setLoading] = useState(true);
  const [year, setYear] = useState(curYear - 1);
  const [exporting, setExporting] = useState(false);

  useEffect(() => {
    setLoading(true);
    fetch(`/api/eegs/${eegId}/ea/jahresabschluss?jahr=${year}`)
      .then((r) => r.ok && r.json())
      .then((d) => { if (d) setData(d); setLoading(false); });
  }, [eegId, year]);

  async function handleExport() {
    setExporting(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/ea/jahresabschluss?jahr=${year}&format=xlsx`);
      if (!res.ok) { alert("Exportfehler"); return; }
      const blob = await res.blob();
      const cd = res.headers.get("Content-Disposition") || "";
      const m = cd.match(/filename="([^"]+)"/);
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = m ? m[1] : `jahresabschluss_${year}.xlsx`;
      a.click();
    } finally {
      setExporting(false);
    }
  }

  const years = Array.from({ length: 6 }, (_, i) => curYear - i);

  return (
    <div className="p-8 max-w-3xl">
      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">Übersicht</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea`} className="text-sm text-slate-500 hover:text-slate-700">E/A-Buchhaltung</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Jahresabschluss</span>
      </div>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">E/A-Jahresabschluss {year}</h1>
          <p className="text-slate-500 mt-1 text-sm">Einnahmen-Ausgaben-Rechnung nach IST-Prinzip</p>
        </div>
        <div className="flex items-center gap-3">
          <select value={year} onChange={(e) => setYear(Number(e.target.value))} className="px-3 py-2 border border-slate-300 rounded-lg text-slate-900 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
            {years.map((y) => <option key={y} value={y}>{y}</option>)}
          </select>
          <button onClick={handleExport} disabled={exporting} className="px-3 py-2 text-sm text-slate-600 border border-slate-300 rounded-lg hover:bg-slate-50 disabled:opacity-50">
            {exporting ? "Exportiert…" : "XLSX"}
          </button>
        </div>
      </div>

      {loading ? (
        <div className="p-8 text-center text-slate-500 text-sm">Wird geladen…</div>
      ) : !data ? (
        <div className="p-8 text-center text-slate-500 text-sm">Keine Daten für {year}.</div>
      ) : (
        <div className="space-y-4">
          {/* Einnahmen */}
          {data.einnahmen && data.einnahmen.length > 0 && (
            <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
              <div className="px-5 py-3 bg-green-50 border-b border-green-200 flex justify-between items-center">
                <h2 className="font-semibold text-green-900">Einnahmen</h2>
                <span className="font-bold text-green-900">{fmt(data.total_einnahmen)}</span>
              </div>
              <table className="w-full text-sm">
                <tbody className="divide-y divide-slate-50">
                  {data.einnahmen.map((z) => (
                    <tr key={z.konto_id} className="hover:bg-slate-50">
                      <td className="px-5 py-2.5 font-mono text-xs text-slate-400 w-16">{z.nummer}</td>
                      <td className="px-5 py-2.5 text-slate-700">{z.name}</td>
                      <td className="px-5 py-2.5 text-right text-slate-900">{fmt(z.einnahmen)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* Ausgaben */}
          {data.ausgaben && data.ausgaben.length > 0 && (
            <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
              <div className="px-5 py-3 bg-red-50 border-b border-red-200 flex justify-between items-center">
                <h2 className="font-semibold text-red-900">Ausgaben</h2>
                <span className="font-bold text-red-900">{fmt(data.total_ausgaben)}</span>
              </div>
              <table className="w-full text-sm">
                <tbody className="divide-y divide-slate-50">
                  {data.ausgaben.map((z) => (
                    <tr key={z.konto_id} className="hover:bg-slate-50">
                      <td className="px-5 py-2.5 font-mono text-xs text-slate-400 w-16">{z.nummer}</td>
                      <td className="px-5 py-2.5 text-slate-700">{z.name}</td>
                      <td className="px-5 py-2.5 text-right text-slate-900">{fmt(z.ausgaben)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* Kein Inhalt */}
          {(!data.einnahmen || data.einnahmen.length === 0) && (!data.ausgaben || data.ausgaben.length === 0) && (
            <div className="bg-white rounded-xl border border-slate-200 p-8 text-center text-slate-500 text-sm">
              Keine Buchungen für {year} vorhanden.
            </div>
          )}

          {/* Ergebnis */}
          <div className="bg-white rounded-xl border border-slate-200 p-5">
            <table className="w-full text-sm">
              <tbody className="divide-y divide-slate-100">
                <tr>
                  <td className="py-2 font-medium text-slate-700">Einnahmen gesamt</td>
                  <td className="py-2 text-right text-green-700 font-medium">{fmt(data.total_einnahmen)}</td>
                </tr>
                <tr>
                  <td className="py-2 font-medium text-slate-700">Ausgaben gesamt</td>
                  <td className="py-2 text-right text-red-700 font-medium">– {fmt(data.total_ausgaben)}</td>
                </tr>
                <tr className="border-t-2 border-slate-300">
                  <td className="py-2.5 font-bold text-slate-900">
                    {data.ueberschuss >= 0 ? "Überschuss (Gewinn)" : "Fehlbetrag (Verlust)"}
                  </td>
                  <td className={`py-2.5 text-right font-bold text-lg ${data.ueberschuss >= 0 ? "text-emerald-700" : "text-red-700"}`}>
                    {fmt(data.ueberschuss)}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div className="p-4 bg-blue-50 border border-blue-200 rounded-lg text-xs text-blue-800">
            <strong>Hinweis:</strong> E/A-Buchhaltung nach IST-Prinzip (§4 Abs. 3 EStG). Buchungen werden erst bei tatsächlichem Zahlungseingang/-ausgang erfasst (Zahlungsdatum gesetzt).
            Als EEG (wirtschaftlicher Geschäftsbetrieb) unterliegt ein allfälliger Überschuss der Körperschaftssteuer (KSt 23 %).
          </div>
        </div>
      )}
    </div>
  );
}
