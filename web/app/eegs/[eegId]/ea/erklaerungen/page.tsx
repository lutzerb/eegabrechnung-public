"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

// Annual U1 response — kz_* fields are live-computed from all buchungen for the year.
interface U1Response {
  jahr: number;
  kz_000: number;
  kz_022: number;
  kz_029: number;
  kz_044: number;
  kz_056: number;
  kz_057: number;
  kz_060: number;
  kz_065: number;
  kz_066: number;
  kz_083: number;
  zahllast: number;
}

interface SaldenEintrag {
  konto_id: string;
  nummer: string;
  name: string;
  typ: string;
  k1_kz?: string;
  einnahmen: number;
  ausgaben: number;
}

// K1/K2 both return EAJahresabschluss with per-account k1_kz
interface KStData {
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

// Human-readable labels for Kennzahlen (per official BMF K1/K2 2025 forms — identical KZ numbers)
const KZ_LABELS: Record<string, string> = {
  // Erträge
  "9040": "KZ 9040  Umsatzerlöse (Waren-/Leistungserlöse)",
  "9060": "KZ 9060  Anlagenerlöse",
  "9070": "KZ 9070  Aktivierte Eigenleistungen",
  "9080": "KZ 9080  Bestandsveränderungen",
  "9090": "KZ 9090  Übrige Erträge (Saldo)",
  // Aufwendungen
  "9100": "KZ 9100  Waren, Rohstoffe, Hilfsstoffe",
  "9110": "KZ 9110  Fremdpersonal und Fremdleistungen",
  "9120": "KZ 9120  Personalaufwand (eigenes Personal)",
  "9130": "KZ 9130  Absetzung für Abnutzung (AfA) Anlagevermögen",
  "9140": "KZ 9140  Abschreibungen Umlaufvermögen",
  "9150": "KZ 9150  Instandhaltungen Gebäude",
  "9160": "KZ 9160  Reise- und Fahrtspesen, Diäten",
  "9170": "KZ 9170  Tatsächliche Kfz-Kosten",
  "9180": "KZ 9180  Miet- und Pachtaufwand, Leasing",
  "9190": "KZ 9190  Provisionen, Lizenzgebühren",
  "9200": "KZ 9200  Werbe- und Repräsentationsaufwand",
  "9210": "KZ 9210  Buchwert abgegangener Anlagen",
  "9220": "KZ 9220  Zinsen und ähnliche Aufwendungen",
  "9230": "KZ 9230  Übrige Aufwendungen, Kapitalveränderungen (Saldo)",
};

const EIN_ORDER = ["9040", "9060", "9070", "9080", "9090"];
const AUS_ORDER = ["9100", "9110", "9120", "9130", "9140", "9150", "9160", "9170", "9180", "9190", "9200", "9210", "9220", "9230"];

function aggregateByKZ(entries: SaldenEintrag[], field: "einnahmen" | "ausgaben"): Record<string, number> {
  const out: Record<string, number> = {};
  for (const e of entries) {
    const kz = e.k1_kz || (field === "einnahmen" ? "9040" : "9100");
    out[kz] = (out[kz] ?? 0) + e[field];
  }
  return out;
}

function KZRow({ label, value, bold }: { label: string; value: number; bold?: boolean }) {
  const [kz, ...rest] = label.split("  ");
  return (
    <tr className={bold ? "border-t-2 border-slate-300" : ""}>
      <td className={`py-2 font-mono text-xs w-20 ${bold ? "font-bold text-slate-900" : "text-slate-400"}`}>{kz}</td>
      <td className={`py-2 ${bold ? "font-bold text-slate-900" : "text-slate-600"}`}>{rest.join("  ")}</td>
      <td className={`py-2 text-right tabular-nums ${bold ? "font-bold text-slate-900" : "text-slate-800"}`}>{fmt(value)}</td>
    </tr>
  );
}

export default function ErklaerунgenPage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;
  const curYear = new Date().getFullYear();
  const [year, setYear] = useState(curYear - 1);

  const [u1, setU1] = useState<U1Response | null>(null);
  const [kst, setKst] = useState<KStData | null>(null);
  const [loading, setLoading] = useState(true);
  const [exportingU1, setExportingU1] = useState(false);
  const [exportingK1, setExportingK1] = useState(false);
  const [exportingK2, setExportingK2] = useState(false);

  useEffect(() => {
    setLoading(true);
    Promise.allSettled([
      fetch(`/api/eegs/${eegId}/ea/erklaerungen/u1?jahr=${year}`),
      fetch(`/api/eegs/${eegId}/ea/erklaerungen/k2?jahr=${year}`),
    ]).then(async ([u1Res, kstRes]) => {
      if (u1Res.status === "fulfilled" && u1Res.value.ok) setU1(await u1Res.value.json());
      if (kstRes.status === "fulfilled" && kstRes.value.ok) setKst(await kstRes.value.json());
      setLoading(false);
    });
  }, [eegId, year]);

  async function handleExport(type: "u1" | "k1" | "k2") {
    const setExp = type === "u1" ? setExportingU1 : type === "k1" ? setExportingK1 : setExportingK2;
    setExp(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/ea/erklaerungen/${type}?jahr=${year}&format=xml`);
      if (!res.ok) { alert("Exportfehler"); return; }
      const blob = await res.blob();
      const cd = res.headers.get("Content-Disposition") || "";
      const m = cd.match(/filename="([^"]+)"/);
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = m ? m[1] : `${type}_${year}.xml`;
      a.click();
    } finally {
      setExp(false);
    }
  }

  const years = Array.from({ length: 6 }, (_, i) => curYear - i);

  // KöSt aggregation (K1 and K2 use identical KZ numbers)
  const einByKZ = kst?.einnahmen ? aggregateByKZ(kst.einnahmen, "einnahmen") : {};
  const aussByKZ = kst?.ausgaben ? aggregateByKZ(kst.ausgaben, "ausgaben") : {};
  const ueberschuss = kst?.ueberschuss ?? 0;

  const einRows = EIN_ORDER.filter((kz) => (einByKZ[kz] ?? 0) > 0);
  const ausRows = AUS_ORDER.filter((kz) => (aussByKZ[kz] ?? 0) > 0);
  const extraEin = Object.keys(einByKZ).filter((kz) => !EIN_ORDER.includes(kz) && (einByKZ[kz] ?? 0) > 0);
  const extraAus = Object.keys(aussByKZ).filter((kz) => !AUS_ORDER.includes(kz) && (aussByKZ[kz] ?? 0) > 0);

  return (
    <div className="p-8 max-w-4xl">
      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">Übersicht</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea`} className="text-sm text-slate-500 hover:text-slate-700">E/A-Buchhaltung</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Jahreserklärungen</span>
      </div>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Jahreserklärungen {year}</h1>
          <p className="text-slate-500 mt-1 text-sm">Grundlagendaten für U1 (Umsatzsteuer) und K2 (Körperschaftsteuer)</p>
        </div>
        <select value={year} onChange={(e) => setYear(Number(e.target.value))} className="px-3 py-2 border border-slate-300 rounded-lg text-slate-900 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
          {years.map((y) => <option key={y} value={y}>{y}</option>)}
        </select>
      </div>

      {loading ? (
        <div className="p-8 text-center text-slate-500 text-sm">Wird geladen…</div>
      ) : (
        <div className="space-y-6">
          {/* ── U1 ── */}
          <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
            <div className="px-5 py-4 bg-purple-50 border-b border-purple-200 flex items-center justify-between">
              <div>
                <h2 className="font-semibold text-purple-900">U1 – Umsatzsteuer-Jahreserklärung</h2>
                <p className="text-xs text-purple-700 mt-0.5">Veranlagungsjahr {year}</p>
              </div>
              <button onClick={() => handleExport("u1")} disabled={exportingU1} className="px-3 py-1.5 text-xs font-medium text-purple-700 border border-purple-300 rounded-lg hover:bg-purple-100 disabled:opacity-50">
                {exportingU1 ? "…" : "FinanzOnline XML"}
              </button>
            </div>
            {!u1 ? (
              <p className="p-5 text-sm text-slate-500">Keine Daten für {year}.</p>
            ) : (
              <div className="p-5">
                <table className="w-full text-sm">
                  <tbody className="divide-y divide-slate-100">
                    <tr><td className="py-2 font-mono text-xs text-slate-400 w-20">KZ 000</td><td className="py-2 text-slate-600">Gesamtbetrag der Lieferungen, sonstigen Leistungen und Eigenverbrauch</td><td className="py-2 text-right tabular-nums">{fmt(u1.kz_000)}</td></tr>
                    <tr><td className="py-2 font-mono text-xs text-slate-400">KZ 022</td><td className="py-2 text-slate-600">Lieferungen und sonstige Leistungen zu 20 %</td><td className="py-2 text-right tabular-nums">{fmt(u1.kz_022)}</td></tr>
                    <tr><td className="py-2 font-mono text-xs text-slate-400">KZ 056</td><td className="py-2 text-slate-600">Umsatzsteuer 20 %</td><td className="py-2 text-right tabular-nums">{fmt(u1.kz_056)}</td></tr>
                    {u1.kz_029 !== 0 && <tr><td className="py-2 font-mono text-xs text-slate-400">KZ 029</td><td className="py-2 text-slate-600">Lieferungen und sonstige Leistungen zu 10 %</td><td className="py-2 text-right tabular-nums">{fmt(u1.kz_029)}</td></tr>}
                    {u1.kz_044 !== 0 && <tr><td className="py-2 font-mono text-xs text-slate-400">KZ 044</td><td className="py-2 text-slate-600">Umsatzsteuer 10 %</td><td className="py-2 text-right tabular-nums">{fmt(u1.kz_044)}</td></tr>}
                    {u1.kz_057 !== 0 && <tr><td className="py-2 font-mono text-xs text-slate-400">KZ 057</td><td className="py-2 text-slate-600">Steuerschuld gem. § 19 Abs. 1 (Reverse Charge)</td><td className="py-2 text-right tabular-nums">{fmt(u1.kz_057)}</td></tr>}
                    {u1.kz_060 !== 0 && <tr><td className="py-2 font-mono text-xs text-slate-400">KZ 060</td><td className="py-2 text-slate-600">Gesamtbetrag der abziehbaren Vorsteuern</td><td className="py-2 text-right tabular-nums">{fmt(u1.kz_060)}</td></tr>}
                    {u1.kz_065 !== 0 && <tr><td className="py-2 font-mono text-xs text-slate-400">KZ 065</td><td className="py-2 text-slate-600">Vorsteuern aus ig. Erwerben</td><td className="py-2 text-right tabular-nums">{fmt(u1.kz_065)}</td></tr>}
                    {u1.kz_066 !== 0 && <tr><td className="py-2 font-mono text-xs text-slate-400">KZ 066</td><td className="py-2 text-slate-600">Vorsteuern für Leistungen gem. § 19 Abs. 1</td><td className="py-2 text-right tabular-nums">{fmt(u1.kz_066)}</td></tr>}
                    {u1.kz_083 !== 0 && <tr><td className="py-2 font-mono text-xs text-slate-400">KZ 083</td><td className="py-2 text-slate-600">Vorsteuern aus ig. Dreiecksgeschäften</td><td className="py-2 text-right tabular-nums">{fmt(u1.kz_083)}</td></tr>}
                    <tr className="border-t-2 border-slate-300">
                      <td className="py-2.5 font-mono text-xs font-bold text-slate-900">KZ 090</td>
                      <td className="py-2.5 font-bold text-slate-900">Vorauszahlung / Überschuss</td>
                      <td className={`py-2.5 text-right font-bold tabular-nums ${u1.zahllast > 0 ? "text-red-700" : "text-green-700"}`}>{fmt(u1.zahllast)}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* ── K2 (+ K1 als Alternative) ── */}
          <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
            <div className="px-5 py-4 bg-indigo-50 border-b border-indigo-200 flex items-center justify-between">
              <div>
                <h2 className="font-semibold text-indigo-900">K2 – Körperschaftsteuererklärung</h2>
                <p className="text-xs text-indigo-700 mt-0.5">Veranlagungsjahr {year} · Verein §5 KStG (beschränkt steuerpflichtig) · E/A-Rechnung</p>
              </div>
              <div className="flex gap-2">
                <button onClick={() => handleExport("k2")} disabled={exportingK2} className="px-3 py-1.5 text-xs font-medium text-indigo-700 border border-indigo-300 rounded-lg hover:bg-indigo-100 disabled:opacity-50">
                  {exportingK2 ? "…" : "K2 FinanzOnline XML"}
                </button>
                <button onClick={() => handleExport("k1")} disabled={exportingK1} className="px-3 py-1.5 text-xs font-medium text-slate-500 border border-slate-300 rounded-lg hover:bg-slate-100 disabled:opacity-50" title="K1 für unbeschränkt steuerpflichtige Körperschaften (GmbH/AG)">
                  {exportingK1 ? "…" : "K1 XML"}
                </button>
              </div>
            </div>
            {!kst ? (
              <p className="p-5 text-sm text-slate-500">Keine Daten für {year}.</p>
            ) : (
              <div className="p-5 space-y-5">

                {/* Erträge / Einnahmen */}
                <div>
                  <p className="text-xs font-semibold uppercase tracking-wide text-slate-400 mb-2">Erträge / Einnahmen</p>
                  <table className="w-full text-sm">
                    <tbody className="divide-y divide-slate-100">
                      {einRows.map((kz) => (
                        <KZRow key={kz} label={KZ_LABELS[kz] ?? `KZ ${kz}  Sonstige Einnahmen`} value={einByKZ[kz]} />
                      ))}
                      {extraEin.map((kz) => (
                        <KZRow key={kz} label={`KZ ${kz}  Sonstige Einnahmen`} value={einByKZ[kz]} />
                      ))}
                      {einRows.length === 0 && extraEin.length === 0 && (
                        <tr><td colSpan={3} className="py-2 text-slate-400 text-xs">Keine Einnahmen</td></tr>
                      )}
                      <KZRow label={`         Summe Betriebseinnahmen`} value={kst.total_einnahmen} bold />
                    </tbody>
                  </table>
                </div>

                {/* Aufwendungen / Ausgaben */}
                <div>
                  <p className="text-xs font-semibold uppercase tracking-wide text-slate-400 mb-2">Aufwendungen / Ausgaben</p>
                  <table className="w-full text-sm">
                    <tbody className="divide-y divide-slate-100">
                      {ausRows.map((kz) => (
                        <KZRow key={kz} label={KZ_LABELS[kz] ?? `KZ ${kz}  Sonstige Ausgaben`} value={aussByKZ[kz]} />
                      ))}
                      {extraAus.map((kz) => (
                        <KZRow key={kz} label={`KZ ${kz}  Sonstige Ausgaben`} value={aussByKZ[kz]} />
                      ))}
                      {ausRows.length === 0 && extraAus.length === 0 && (
                        <tr><td colSpan={3} className="py-2 text-slate-400 text-xs">Keine Ausgaben</td></tr>
                      )}
                      <KZRow label="         Summe Betriebsausgaben" value={kst.total_ausgaben} bold />
                    </tbody>
                  </table>
                </div>

                {/* Ergebnis */}
                <div>
                  <p className="text-xs font-semibold uppercase tracking-wide text-slate-400 mb-2">Ergebnis (KZ 9280)</p>
                  <table className="w-full text-sm">
                    <tbody>
                      <tr className="border-t-2 border-slate-300">
                        <td className="py-2.5 font-mono text-xs font-bold text-slate-900 w-20">KZ 9280</td>
                        <td className="py-2.5 font-bold text-slate-900">Überschuss (+) / Fehlbetrag (–)</td>
                        <td className={`py-2.5 text-right font-bold tabular-nums ${ueberschuss >= 0 ? "text-green-700" : "text-red-700"}`}>{fmt(ueberschuss)}</td>
                      </tr>
                    </tbody>
                  </table>
                  <p className="mt-2 text-xs text-slate-400">K2 für Vereine (§5 KStG) · K2a = EINKUENFTE_GEWERBEBETRIEB_K2 im XML · K1 XML für GmbH/AG verfügbar.</p>
                </div>

              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
