"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

interface EAKonto {
  id: string;
  nummer: string;
  name: string;
  typ: string;
}

interface KontenblattEintrag {
  id: string;
  buchungsnr: string;
  beleg_datum?: string;
  zahlung_datum?: string;
  beschreibung: string;
  richtung: string;
  betrag_brutto: number;
  ust_betrag: number;
  laufender_saldo: number;
}

interface Kontenblatt {
  konto: EAKonto;
  eintraege: KontenblattEintrag[] | null;
  summe: number;
}

function fmt(n: number): string {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(n);
}

function fmtDate(s?: string): string {
  if (!s) return "—";
  try { return new Date(s).toLocaleDateString("de-AT", { day: "2-digit", month: "2-digit", year: "numeric" }); } catch { return s; }
}

export default function KontenblattPage() {
  const params = useParams<{ eegId: string; id: string }>();
  const { eegId, id } = params;
  const curYear = new Date().getFullYear();

  const [data, setData] = useState<Kontenblatt | null>(null);
  const [loading, setLoading] = useState(true);
  const [year, setYear] = useState(curYear);

  useEffect(() => {
    setLoading(true);
    setData(null);
    const von = `${year}-01-01`;
    const bis = `${year}-12-31`;
    fetch(`/api/eegs/${eegId}/ea/kontenblatt/${id}?von=${von}&bis=${bis}`)
      .then((r) => r.ok && r.json())
      .then((d) => { if (d) setData(d); setLoading(false); });
  }, [eegId, id, year]);

  const years = Array.from({ length: 6 }, (_, i) => curYear - i);

  const eintraege = data?.eintraege ?? [];
  const summeEin = eintraege.filter((b) => b.richtung === "EINNAHME").reduce((s, b) => s + b.betrag_brutto, 0);
  const summeAus = eintraege.filter((b) => b.richtung === "AUSGABE").reduce((s, b) => s + b.betrag_brutto, 0);

  return (
    <div className="p-8 max-w-4xl">
      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">Übersicht</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea`} className="text-sm text-slate-500 hover:text-slate-700">E/A-Buchhaltung</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea/saldenliste`} className="text-sm text-slate-500 hover:text-slate-700">Saldenliste</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Kontenblatt</span>
      </div>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">
            Kontenblatt {data ? `${data.konto.nummer} – ${data.konto.name}` : "…"}
          </h1>
          <p className="text-slate-500 mt-1 text-sm">Alle Buchungen auf diesem Konto, Geschäftsjahr {year}</p>
        </div>
        <select
          value={year}
          onChange={(e) => setYear(Number(e.target.value))}
          className="px-3 py-2 border border-slate-300 rounded-lg text-slate-900 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {years.map((y) => <option key={y} value={y}>{y}</option>)}
        </select>
      </div>

      {loading ? (
        <div className="p-8 text-center text-slate-500 text-sm">Wird geladen…</div>
      ) : !data ? (
        <div className="p-8 text-center text-slate-500 text-sm">Keine Daten gefunden.</div>
      ) : (
        <>
          <div className="grid grid-cols-3 gap-4 mb-4">
            <div className="bg-green-50 border border-green-200 rounded-xl p-4">
              <p className="text-xs font-medium text-green-700">Einnahmen</p>
              <p className="text-xl font-bold text-green-900 mt-1">{fmt(summeEin)}</p>
            </div>
            <div className="bg-red-50 border border-red-200 rounded-xl p-4">
              <p className="text-xs font-medium text-red-700">Ausgaben</p>
              <p className="text-xl font-bold text-red-900 mt-1">{fmt(summeAus)}</p>
            </div>
            <div className="bg-slate-50 border border-slate-200 rounded-xl p-4">
              <p className="text-xs font-medium text-slate-600">Saldo</p>
              <p className={`text-xl font-bold mt-1 ${data.summe >= 0 ? "text-slate-900" : "text-red-700"}`}>{fmt(data.summe)}</p>
            </div>
          </div>

          <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
            {eintraege.length === 0 ? (
              <div className="p-8 text-center text-slate-500 text-sm">Keine Buchungen in diesem Zeitraum.</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-slate-200 bg-slate-50">
                      <th className="text-left px-4 py-3 text-xs font-medium text-slate-500">Belegdatum</th>
                      <th className="text-left px-4 py-3 text-xs font-medium text-slate-500">Zahlung</th>
                      <th className="text-left px-4 py-3 text-xs font-medium text-slate-500">Beleg-Nr.</th>
                      <th className="text-left px-4 py-3 text-xs font-medium text-slate-500">Beschreibung</th>
                      <th className="text-right px-4 py-3 text-xs font-medium text-slate-500">Brutto</th>
                      <th className="text-right px-4 py-3 text-xs font-medium text-slate-500">lfd. Saldo</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {eintraege.map((b) => (
                      <tr key={b.id} className="hover:bg-slate-50">
                        <td className="px-4 py-3 text-slate-600">{fmtDate(b.beleg_datum)}</td>
                        <td className="px-4 py-3 text-slate-500">
                          {b.zahlung_datum ? fmtDate(b.zahlung_datum) : <span className="text-amber-500 text-xs">offen</span>}
                        </td>
                        <td className="px-4 py-3 font-mono text-xs text-slate-500">{b.buchungsnr}</td>
                        <td className="px-4 py-3 text-slate-900">{b.beschreibung}</td>
                        <td className={`px-4 py-3 text-right font-medium ${b.richtung === "EINNAHME" ? "text-green-700" : "text-red-700"}`}>
                          {b.richtung === "AUSGABE" ? "–" : ""}{fmt(b.betrag_brutto)}
                        </td>
                        <td className="px-4 py-3 text-right text-slate-700">{fmt(b.laufender_saldo)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
