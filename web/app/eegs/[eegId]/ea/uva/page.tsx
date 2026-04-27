"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

interface UVAKennzahlen {
  kz_000: number; // Gesamtbetrag Lieferungen/Leistungen
  kz_022: number; // Umsätze zu 20 % (Bemessungsgrundlage)
  kz_029: number; // Umsätze zu 10 % (Bemessungsgrundlage)
  kz_044: number; // Umsatzsteuer 10 %
  kz_056: number; // Umsatzsteuer 20 %
  kz_057: number; // Steuerschuld gem. § 19 Abs. 1 (Reverse Charge)
  kz_060: number; // Gesamtbetrag abziehbare Vorsteuern
  kz_065: number; // Vorsteuern aus ig. Erwerben
  kz_066: number; // Vorsteuern für Leistungen gem. § 19 Abs. 1
  kz_083: number; // Vorsteuern aus ig. Dreiecksgeschäften
  zahllast: number;
}

interface UVAPeriode {
  id: string;
  jahr: number;
  quartal?: number;
  monat?: number;
  periodentyp: string;
  eingereicht_am?: string;
  kennzahlen?: UVAKennzahlen;
}

function fmt(n: number): string {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(n);
}

function periodLabel(u: UVAPeriode): string {
  if (u.periodentyp === "QUARTAL" && u.quartal) return `Q${u.quartal}/${u.jahr}`;
  if (u.periodentyp === "MONAT" && u.monat) {
    const months = ["", "Jän", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"];
    return `${months[u.monat]}/${u.jahr}`;
  }
  return `${u.jahr}`;
}

export default function UVAPage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;
  const curYear = new Date().getFullYear();

  const [uvas, setUvas] = useState<UVAPeriode[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [kennzahlen, setKennzahlen] = useState<UVAKennzahlen | null>(null);
  const [knLoading, setKnLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [marking, setMarking] = useState(false);
  const [xmlLoading, setXmlLoading] = useState(false);

  const [newForm, setNewForm] = useState({ jahr: curYear, periodentyp: "QUARTAL", quartal: 1, monat: 1 });
  const [showCreate, setShowCreate] = useState(false);

  async function load() {
    const res = await fetch(`/api/eegs/${eegId}/ea/uva`);
    if (res.ok) setUvas(await res.json());
    setLoading(false);
  }

  useEffect(() => { load(); }, [eegId]);

  async function selectUVA(id: string) {
    setSelectedId(id);
    setKnLoading(true);
    const res = await fetch(`/api/eegs/${eegId}/ea/uva/${id}?action=kennzahlen`);
    if (res.ok) setKennzahlen(await res.json());
    setKnLoading(false);
  }

  async function handleCreate() {
    setCreating(true);
    const body: Record<string, unknown> = { jahr: newForm.jahr, periodentyp: newForm.periodentyp };
    if (newForm.periodentyp === "QUARTAL") body.quartal = newForm.quartal;
    if (newForm.periodentyp === "MONAT") body.monat = newForm.monat;
    const res = await fetch(`/api/eegs/${eegId}/ea/uva`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
    if (res.ok) { await load(); setShowCreate(false); }
    else alert((await res.json()).error || "Fehler");
    setCreating(false);
  }

  async function handleEinreichen(id: string) {
    if (!confirm("UVA als eingereicht markieren? Diese Aktion kann nicht rückgängig gemacht werden.")) return;
    setMarking(true);
    const res = await fetch(`/api/eegs/${eegId}/ea/uva/${id}`, { method: "PATCH" });
    if (res.ok || res.status === 204) await load();
    else alert("Fehler");
    setMarking(false);
  }

  async function handleExportXML(id: string) {
    setXmlLoading(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/ea/uva/${id}?action=export&format=xml`);
      if (!res.ok) { alert("Exportfehler"); return; }
      const blob = await res.blob();
      const cd = res.headers.get("Content-Disposition") || "";
      const m = cd.match(/filename="([^"]+)"/);
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = m ? m[1] : `uva_${id}.xml`;
      a.click();
    } finally {
      setXmlLoading(false);
    }
  }

  const sel = uvas.find((u) => u.id === selectedId);

  return (
    <div className="p-8">
      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">Übersicht</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea`} className="text-sm text-slate-500 hover:text-slate-700">E/A-Buchhaltung</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">UVA</span>
      </div>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Umsatzsteuervoranmeldung</h1>
          <p className="text-slate-500 mt-1 text-sm">UVA-Perioden verwalten und für FinanzOnline exportieren</p>
        </div>
        <button onClick={() => setShowCreate(!showCreate)} className="px-4 py-2 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800">
          + UVA-Periode
        </button>
      </div>

      {showCreate && (
        <div className="mb-4 bg-white rounded-xl border border-slate-200 p-5">
          <h3 className="font-semibold text-slate-900 mb-4 text-sm">Neue UVA-Periode</h3>
          <div className="flex flex-wrap gap-3 items-end">
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">Jahr</label>
              <select className="px-3 py-2 border border-slate-300 rounded-lg text-sm" value={newForm.jahr} onChange={(e) => setNewForm({ ...newForm, jahr: Number(e.target.value) })}>
                {Array.from({ length: 5 }, (_, i) => curYear - i).map((y) => <option key={y} value={y}>{y}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">Periodentyp</label>
              <select className="px-3 py-2 border border-slate-300 rounded-lg text-sm" value={newForm.periodentyp} onChange={(e) => setNewForm({ ...newForm, periodentyp: e.target.value })}>
                <option value="QUARTAL">Quartal</option>
                <option value="MONAT">Monat</option>
              </select>
            </div>
            {newForm.periodentyp === "QUARTAL" && (
              <div>
                <label className="block text-xs font-medium text-slate-700 mb-1">Quartal</label>
                <select className="px-3 py-2 border border-slate-300 rounded-lg text-sm" value={newForm.quartal} onChange={(e) => setNewForm({ ...newForm, quartal: Number(e.target.value) })}>
                  {[1, 2, 3, 4].map((q) => <option key={q} value={q}>Q{q}</option>)}
                </select>
              </div>
            )}
            {newForm.periodentyp === "MONAT" && (
              <div>
                <label className="block text-xs font-medium text-slate-700 mb-1">Monat</label>
                <select className="px-3 py-2 border border-slate-300 rounded-lg text-sm" value={newForm.monat} onChange={(e) => setNewForm({ ...newForm, monat: Number(e.target.value) })}>
                  {["Jänner","Februar","März","April","Mai","Juni","Juli","August","September","Oktober","November","Dezember"].map((m, i) => <option key={i+1} value={i+1}>{m}</option>)}
                </select>
              </div>
            )}
            <button onClick={handleCreate} disabled={creating} className="px-4 py-2 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800 disabled:opacity-50">
              {creating ? "Erstellt…" : "Erstellen"}
            </button>
            <button onClick={() => setShowCreate(false)} className="px-4 py-2 text-sm text-slate-600 border border-slate-300 rounded-lg hover:bg-slate-50">
              Abbrechen
            </button>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* UVA list */}
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          {loading ? (
            <div className="p-6 text-center text-slate-500 text-sm">Wird geladen…</div>
          ) : uvas.length === 0 ? (
            <div className="p-6 text-center text-slate-500 text-sm">Keine UVA-Perioden vorhanden.</div>
          ) : (
            <ul className="divide-y divide-slate-100">
              {uvas.map((u) => (
                <li key={u.id}>
                  <button
                    onClick={() => selectUVA(u.id)}
                    className={`w-full text-left px-4 py-3 hover:bg-slate-50 ${selectedId === u.id ? "bg-blue-50" : ""}`}
                  >
                    <div className="flex items-center justify-between">
                      <span className="font-medium text-slate-900 text-sm">{periodLabel(u)}</span>
                      {u.eingereicht_am ? (
                        <span className="text-xs bg-green-50 text-green-700 px-2 py-0.5 rounded">eingereicht</span>
                      ) : (
                        <span className="text-xs bg-amber-50 text-amber-700 px-2 py-0.5 rounded">offen</span>
                      )}
                    </div>
                    {u.kennzahlen && (
                      <p className="text-xs text-slate-500 mt-0.5">Zahllast: {fmt(u.kennzahlen.zahllast)}</p>
                    )}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Kennzahlen detail */}
        <div className="lg:col-span-2">
          {!selectedId ? (
            <div className="bg-white rounded-xl border border-slate-200 p-8 text-center text-slate-500 text-sm">
              Wählen Sie eine UVA-Periode aus der Liste.
            </div>
          ) : knLoading ? (
            <div className="bg-white rounded-xl border border-slate-200 p-8 text-center text-slate-500 text-sm">Wird geladen…</div>
          ) : (
            <div className="bg-white rounded-xl border border-slate-200 p-6">
              <div className="flex items-center justify-between mb-5">
                <h2 className="font-semibold text-slate-900">{sel ? periodLabel(sel) : ""} — Kennzahlen</h2>
                <div className="flex gap-2">
                  {sel && !sel.eingereicht_am && (
                    <button
                      onClick={() => handleEinreichen(sel.id)}
                      disabled={marking}
                      className="px-3 py-1.5 text-xs font-medium bg-green-700 text-white rounded-lg hover:bg-green-800 disabled:opacity-50"
                    >
                      Als eingereicht markieren
                    </button>
                  )}
                  <button
                    onClick={() => selectedId && handleExportXML(selectedId)}
                    disabled={xmlLoading}
                    className="px-3 py-1.5 text-xs font-medium text-slate-600 border border-slate-300 rounded-lg hover:bg-slate-50 disabled:opacity-50"
                  >
                    {xmlLoading ? "Exportiert…" : "XML (FinanzOnline)"}
                  </button>
                </div>
              </div>

              {kennzahlen && (
                <table className="w-full text-sm">
                  <tbody className="divide-y divide-slate-100">
                    <tr><td className="py-2 text-slate-600">KZ 000 – Gesamtbetrag der Lieferungen, sonstigen Leistungen und Eigenverbrauch</td><td className="py-2 text-right font-medium">{fmt(kennzahlen.kz_000)}</td></tr>
                    <tr><td className="py-2 text-slate-600">KZ 022 – Lieferungen und sonstige Leistungen zu 20 %</td><td className="py-2 text-right font-medium">{fmt(kennzahlen.kz_022)}</td></tr>
                    <tr><td className="py-2 text-slate-600">KZ 056 – Umsatzsteuer 20 %</td><td className="py-2 text-right font-medium">{fmt(kennzahlen.kz_056)}</td></tr>
                    {kennzahlen.kz_029 !== 0 && <tr><td className="py-2 text-slate-600">KZ 029 – Lieferungen und sonstige Leistungen zu 10 %</td><td className="py-2 text-right font-medium">{fmt(kennzahlen.kz_029)}</td></tr>}
                    {kennzahlen.kz_044 !== 0 && <tr><td className="py-2 text-slate-600">KZ 044 – Umsatzsteuer 10 %</td><td className="py-2 text-right font-medium">{fmt(kennzahlen.kz_044)}</td></tr>}
                    {kennzahlen.kz_057 !== 0 && <tr><td className="py-2 text-slate-600">KZ 057 – Steuerschuld gem. § 19 Abs. 1 (Reverse Charge)</td><td className="py-2 text-right font-medium">{fmt(kennzahlen.kz_057)}</td></tr>}
                    <tr><td className="py-2 text-slate-600">KZ 060 – Gesamtbetrag der abziehbaren Vorsteuern</td><td className="py-2 text-right font-medium">{fmt(kennzahlen.kz_060)}</td></tr>
                    {kennzahlen.kz_065 !== 0 && <tr><td className="py-2 text-slate-600">KZ 065 – Vorsteuern aus ig. Erwerben</td><td className="py-2 text-right font-medium">{fmt(kennzahlen.kz_065)}</td></tr>}
                    {kennzahlen.kz_066 !== 0 && <tr><td className="py-2 text-slate-600">KZ 066 – Vorsteuern für Leistungen gem. § 19 Abs. 1</td><td className="py-2 text-right font-medium">{fmt(kennzahlen.kz_066)}</td></tr>}
                    {kennzahlen.kz_083 !== 0 && <tr><td className="py-2 text-slate-600">KZ 083 – Vorsteuern aus ig. Dreiecksgeschäften</td><td className="py-2 text-right font-medium">{fmt(kennzahlen.kz_083)}</td></tr>}
                    <tr className="border-t-2 border-slate-300">
                      <td className="py-2.5 font-bold text-slate-900">KZ 090 – Vorauszahlung / Überschuss</td>
                      <td className={`py-2.5 text-right font-bold text-base ${kennzahlen.zahllast > 0 ? "text-red-700" : "text-green-700"}`}>{fmt(kennzahlen.zahllast)}</td>
                    </tr>
                  </tbody>
                </table>
              )}

              {sel?.eingereicht_am && (
                <p className="mt-4 text-xs text-slate-500">
                  Eingereicht am: {new Date(sel.eingereicht_am).toLocaleDateString("de-AT", { day: "2-digit", month: "2-digit", year: "numeric" })}
                </p>
              )}

              <div className="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg text-xs text-blue-800">
                Für Kleinunternehmer (§6 Abs. 1 Z 27 UStG) entfällt der Vorsteuerabzug (KZ 060, KZ 066 = 0) — die Steuerschuld aus Reverse Charge (KZ 057) ist trotzdem zu melden und abzuführen.
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
