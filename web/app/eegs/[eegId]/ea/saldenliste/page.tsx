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
  anzahl_buchungen: number;
}

function fmt(n: number): string {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(n);
}

function fmtDate(iso: string): string {
  const [y, m, d] = iso.split("-");
  return `${d}.${m}.${y}`;
}

function toIso(d: Date): string {
  return d.toISOString().slice(0, 10);
}

export default function SaldenlistePage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;

  const today = toIso(new Date());
  const yearStart = `${new Date().getFullYear()}-01-01`;

  const [von, setVon] = useState(yearStart);
  const [bis, setBis] = useState(today);
  const [eintraege, setEintraege] = useState<SaldenEintrag[]>([]);
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);

  async function load() {
    setLoading(true);
    const res = await fetch(`/api/eegs/${eegId}/ea/saldenliste?von=${von}&bis=${bis}`);
    if (res.ok) setEintraege(await res.json());
    setLoading(false);
  }

  useEffect(() => { load(); }, [eegId, von, bis]);

  async function handleExport() {
    setExporting(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/ea/saldenliste?von=${von}&bis=${bis}&format=xlsx`);
      if (!res.ok) { alert("Exportfehler"); return; }
      const blob = await res.blob();
      const cd = res.headers.get("Content-Disposition") || "";
      const m = cd.match(/filename="([^"]+)"/);
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = m ? m[1] : `saldenliste_${bis}.xlsx`;
      a.click();
    } finally {
      setExporting(false);
    }
  }

  const inputClass = "px-3 py-2 border border-slate-300 rounded-lg text-slate-900 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500";

  const einnahmenKonten = eintraege.filter((k) => k.typ === "EINNAHME");
  const ausgabenKonten = eintraege.filter((k) => k.typ === "AUSGABE");
  const sonstigKonten = eintraege.filter((k) => k.typ !== "EINNAHME" && k.typ !== "AUSGABE");

  const summeEinnahmen = einnahmenKonten.reduce((s, k) => s + k.einnahmen, 0);
  const summeAusgaben = ausgabenKonten.reduce((s, k) => s + k.ausgaben, 0);
  const gewinnVerlust = summeEinnahmen - summeAusgaben;

  return (
    <div className="p-8 max-w-4xl">
      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">Übersicht</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea`} className="text-sm text-slate-500 hover:text-slate-700">E/A-Buchhaltung</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Saldenliste</span>
      </div>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Saldenliste</h1>
          <p className="text-slate-500 mt-1 text-sm">
            {fmtDate(von)} – {fmtDate(bis)}
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <label className="text-xs font-medium text-slate-600 whitespace-nowrap">Von</label>
            <input
              type="date"
              value={von}
              onChange={(e) => setVon(e.target.value)}
              className={inputClass}
            />
          </div>
          <div className="flex items-center gap-2">
            <label className="text-xs font-medium text-slate-600 whitespace-nowrap">Stichtag</label>
            <input
              type="date"
              value={bis}
              onChange={(e) => setBis(e.target.value)}
              className={inputClass}
            />
          </div>
          <button onClick={handleExport} disabled={exporting} className="px-3 py-2 text-sm text-slate-600 border border-slate-300 rounded-lg hover:bg-slate-50 disabled:opacity-50">
            {exporting ? "Exportiert…" : "XLSX"}
          </button>
        </div>
      </div>

      {loading ? (
        <div className="p-8 text-center text-slate-500 text-sm">Wird geladen…</div>
      ) : eintraege.length === 0 ? (
        <div className="p-8 text-center text-slate-500 text-sm">
          Keine Buchungen im Zeitraum {fmtDate(von)} – {fmtDate(bis)}.
        </div>
      ) : (
        <>
          {/* Summary */}
          <div className="grid grid-cols-3 gap-4 mb-6">
            <div className="bg-green-50 border border-green-200 rounded-xl p-4">
              <p className="text-xs font-medium text-green-700 uppercase tracking-wide">Einnahmen gesamt</p>
              <p className="text-2xl font-bold text-green-900 mt-1">{fmt(summeEinnahmen)}</p>
            </div>
            <div className="bg-red-50 border border-red-200 rounded-xl p-4">
              <p className="text-xs font-medium text-red-700 uppercase tracking-wide">Ausgaben gesamt</p>
              <p className="text-2xl font-bold text-red-900 mt-1">{fmt(summeAusgaben)}</p>
            </div>
            <div className={`rounded-xl p-4 border ${gewinnVerlust >= 0 ? "bg-emerald-50 border-emerald-200" : "bg-orange-50 border-orange-200"}`}>
              <p className={`text-xs font-medium uppercase tracking-wide ${gewinnVerlust >= 0 ? "text-emerald-700" : "text-orange-700"}`}>
                {gewinnVerlust >= 0 ? "Überschuss" : "Verlust"}
              </p>
              <p className={`text-2xl font-bold mt-1 ${gewinnVerlust >= 0 ? "text-emerald-900" : "text-orange-900"}`}>
                {fmt(Math.abs(gewinnVerlust))}
              </p>
            </div>
          </div>

          {einnahmenKonten.length > 0 && (
            <div className="bg-white rounded-xl border border-slate-200 overflow-hidden mb-4">
              <div className="px-4 py-3 bg-green-50 border-b border-green-200">
                <h3 className="font-semibold text-green-900 text-sm">Einnahmenkonten</h3>
              </div>
              <KontenTable konten={einnahmenKonten} eegId={eegId} />
            </div>
          )}

          {ausgabenKonten.length > 0 && (
            <div className="bg-white rounded-xl border border-slate-200 overflow-hidden mb-4">
              <div className="px-4 py-3 bg-red-50 border-b border-red-200">
                <h3 className="font-semibold text-red-900 text-sm">Ausgabenkonten</h3>
              </div>
              <KontenTable konten={ausgabenKonten} eegId={eegId} />
            </div>
          )}

          {sonstigKonten.length > 0 && (
            <div className="bg-white rounded-xl border border-slate-200 overflow-hidden mb-4">
              <div className="px-4 py-3 bg-slate-50 border-b border-slate-200">
                <h3 className="font-semibold text-slate-900 text-sm">Sonstige Konten</h3>
              </div>
              <KontenTable konten={sonstigKonten} eegId={eegId} />
            </div>
          )}
        </>
      )}
    </div>
  );
}

function KontenTable({ konten, eegId }: { konten: SaldenEintrag[]; eegId: string }) {
  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="border-b border-slate-100">
          <th className="text-left px-4 py-2.5 text-xs font-medium text-slate-500">Konto</th>
          <th className="text-left px-4 py-2.5 text-xs font-medium text-slate-500">Bezeichnung</th>
          <th className="text-right px-4 py-2.5 text-xs font-medium text-slate-500">Einnahmen</th>
          <th className="text-right px-4 py-2.5 text-xs font-medium text-slate-500">Ausgaben</th>
          <th className="text-right px-4 py-2.5 text-xs font-medium text-slate-500">Saldo</th>
          <th className="text-right px-4 py-2.5 text-xs font-medium text-slate-500">Buchungen</th>
          <th className="px-4 py-2.5"></th>
        </tr>
      </thead>
      <tbody className="divide-y divide-slate-50">
        {konten.map((k) => (
          <tr key={k.konto_id} className="hover:bg-slate-50">
            <td className="px-4 py-2.5 font-mono text-xs text-slate-600">{k.nummer}</td>
            <td className="px-4 py-2.5 text-slate-900">{k.name}</td>
            <td className="px-4 py-2.5 text-right text-slate-700">{k.einnahmen > 0 ? fmt(k.einnahmen) : "—"}</td>
            <td className="px-4 py-2.5 text-right text-slate-700">{k.ausgaben > 0 ? fmt(k.ausgaben) : "—"}</td>
            <td className={`px-4 py-2.5 text-right font-medium ${k.saldo >= 0 ? "text-slate-900" : "text-red-600"}`}>{fmt(k.saldo)}</td>
            <td className="px-4 py-2.5 text-right text-slate-400 text-xs">{k.anzahl_buchungen}</td>
            <td className="px-4 py-2.5">
              <Link href={`/eegs/${eegId}/ea/kontenblatt/${k.konto_id}`} className="text-xs text-blue-600 hover:underline">Blatt</Link>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
