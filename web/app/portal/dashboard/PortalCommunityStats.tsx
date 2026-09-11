"use client";

import { useState, useEffect, useMemo } from "react";
import { ProductionConsumptionChart } from "@/components/energy-charts";
import type { ComboBarLineRow } from "@/components/energy-charts";
import type { EnergySummaryRow } from "@/lib/api";

const MONTHS = ["Jan", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"];

// UI tab keys — each shows exactly one period of that size, broken down into
// the next-finer bucket (same "one level down" convention as the admin
// reports page: Jahr→12 Monate, Monat→Tage, Tag→15-Min-Intervalle).
type CommunityGranularity = "day" | "month" | "quarter" | "year";
// Backend bucket size actually requested for each tab.
type BackendGranularity = "15min" | "day" | "month";

const GRANULARITY_LABELS: Record<CommunityGranularity, string> = {
  day: "Tag", month: "Monat", quarter: "Quartal", year: "Jahr",
};

const BACKEND_GRANULARITY: Record<CommunityGranularity, BackendGranularity> = {
  day: "15min", month: "day", quarter: "month", year: "month",
};

// GET /api/portal/energy (member-scoped, existing endpoint) — pre-formatted
// Vienna-local period strings ("2026", "2026-08", "2026-08-15", "2026-08-15 14:00"),
// unlike the community endpoint's RFC3339 timestamps.
interface MemberEnergyRow {
  period: string;
  wh_total_consumption: number;
  wh_community: number;
  wh_total_generation: number;
  wh_community_gen: number;
}

function pad(n: number) {
  return String(n).padStart(2, "0");
}

function fmtKwh(v: number) {
  if (v >= 100000) return new Intl.NumberFormat("de-AT", { maximumFractionDigits: 1 }).format(v / 1000) + " MWh";
  return new Intl.NumberFormat("de-AT", { maximumFractionDigits: 1 }).format(v) + " kWh";
}

function KpiCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-white rounded-xl border border-slate-200 p-5">
      <p className="text-xs font-medium text-slate-500 uppercase tracking-wide">{label}</p>
      <p className="text-2xl font-bold mt-1 text-slate-900">{value}</p>
    </div>
  );
}

// Normalizes a period string from EITHER endpoint into Vienna-local calendar
// parts. The community endpoint returns RFC3339 (Go time.Time JSON, UTC-labeled
// but the fields already represent Vienna wall-clock time — see energy-charts.tsx);
// the member endpoint returns a pre-formatted Vienna-local string per bucket
// ("2026", "2026-08", "2026-08-15", "2026-08-15 14:00"). Both must resolve to the
// same key so rows from the two sources can be merged by period.
function parsePeriod(period: string): { y: number; mo: number; da: number; hh: number; mi: number } {
  if (period.includes("T")) {
    const d = new Date(period);
    return { y: d.getUTCFullYear(), mo: d.getUTCMonth() + 1, da: d.getUTCDate(), hh: d.getUTCHours(), mi: d.getUTCMinutes() };
  }
  const [datePart, timePart] = period.split(" ");
  const dateSegs = datePart.split("-").map(Number);
  const [y, mo, da] = [dateSegs[0], dateSegs[1] || 1, dateSegs[2] || 1];
  const [hh, mi] = (timePart || "00:00").split(":").map(Number);
  return { y, mo, da, hh, mi };
}

function keyFor(period: string, bucket: BackendGranularity): string {
  const p = parsePeriod(period);
  if (bucket === "month") return `${p.y}-${pad(p.mo)}`;
  if (bucket === "day") return `${p.y}-${pad(p.mo)}-${pad(p.da)}`;
  return `${p.y}-${pad(p.mo)}-${pad(p.da)} ${pad(p.hh)}:${pad(p.mi)}`;
}

function periodLabel(key: string, bucket: BackendGranularity): string {
  const p = parsePeriod(key);
  if (bucket === "month") return MONTHS[p.mo - 1];
  if (bucket === "day") return `${p.da}.${p.mo}.`;
  return `${pad(p.hh)}:${pad(p.mi)}`;
}

export default function PortalCommunityStats() {
  const [granularity, setGranularity] = useState<CommunityGranularity>("month");
  const [navYear, setNavYear] = useState<number>(() => new Date().getFullYear());
  const [navMonth, setNavMonth] = useState<number>(() => new Date().getMonth() + 1);
  const [navDay, setNavDay] = useState<number>(() => new Date().getDate());
  const [navQuarter, setNavQuarter] = useState<number>(() => Math.floor(new Date().getMonth() / 3) + 1);
  const [cumulative, setCumulative] = useState(false);
  const [memberRows, setMemberRows] = useState<MemberEnergyRow[]>([]);
  const [communityRows, setCommunityRows] = useState<EnergySummaryRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const bucket = BACKEND_GRANULARITY[granularity];

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    let from: string, to: string;
    if (granularity === "day") {
      // One calendar day, 15-min buckets (24h intraday view).
      const next = new Date(navYear, navMonth - 1, navDay + 1);
      from = `${navYear}-${pad(navMonth)}-${pad(navDay)}`;
      to = `${next.getFullYear()}-${pad(next.getMonth() + 1)}-${pad(next.getDate())}`;
    } else if (granularity === "month") {
      // One calendar month, daily buckets.
      const nextMonth = navMonth === 12 ? 1 : navMonth + 1;
      const nextYear = navMonth === 12 ? navYear + 1 : navYear;
      from = `${navYear}-${pad(navMonth)}-01`;
      to = `${nextYear}-${pad(nextMonth)}-01`;
    } else if (granularity === "quarter") {
      // One calendar quarter (3 months), monthly buckets.
      const startMonth = (navQuarter - 1) * 3 + 1;
      const endMonth = startMonth + 3;
      from = `${navYear}-${pad(startMonth)}-01`;
      to = endMonth > 12 ? `${navYear + 1}-${pad(endMonth - 12)}-01` : `${navYear}-${pad(endMonth)}-01`;
    } else {
      // One calendar year, monthly buckets.
      from = `${navYear}-01-01`;
      to = `${navYear + 1}-01-01`;
    }

    Promise.all([
      // Member's own consumption split (Bezug EEG / Bezug Netz) — same endpoint
      // the personal energy tab below already uses.
      fetch(`/api/portal/energy?from=${from}&to=${to}&granularity=${bucket}`).then(r => {
        if (!r.ok) throw new Error(String(r.status));
        return r.json();
      }),
      // EEG-wide totals — only Gesamterzeugung (and the Eigenverbrauchsanteil KPI) come from here.
      fetch(`/api/portal/community-energy?from=${from}&to=${to}&granularity=${bucket}`).then(r => {
        if (!r.ok) throw new Error(String(r.status));
        return r.json();
      }),
    ])
      .then(([memberData, communityData]) => {
        if (!cancelled) {
          setMemberRows(Array.isArray(memberData) ? memberData : []);
          setCommunityRows(Array.isArray(communityData) ? communityData : []);
          setLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setError("Fehler beim Laden der Energiedaten.");
          setLoading(false);
        }
      });

    return () => { cancelled = true; };
  }, [granularity, bucket, navYear, navMonth, navDay, navQuarter]);

  // Panel 1 "Produktion & Verbrauch": bars = member's OWN consumption split
  // (Bezug EEG / Bezug Netz), line = EEG-wide total generation.
  const { chartData, totalGeneration, totalSelf, totalRestbedarf, totalCommunityGen } = useMemo(() => {
    const byKey = new Map<string, { bezugEEG: number; bezugNetz: number; gesamterzeugung: number }>();
    for (const r of memberRows) {
      const k = keyFor(r.period, bucket);
      const bezugEEG = r.wh_community;
      const bezugNetz = Math.max(0, r.wh_total_consumption - r.wh_community);
      const entry = byKey.get(k) ?? { bezugEEG: 0, bezugNetz: 0, gesamterzeugung: 0 };
      entry.bezugEEG += bezugEEG;
      entry.bezugNetz += bezugNetz;
      byKey.set(k, entry);
    }
    for (const r of communityRows) {
      const k = keyFor(r.period, bucket);
      const entry = byKey.get(k) ?? { bezugEEG: 0, bezugNetz: 0, gesamterzeugung: 0 };
      entry.gesamterzeugung += r.wh_total_generation;
      byKey.set(k, entry);
    }

    const sortedKeys = [...byKey.keys()].sort();
    let accGen = 0, accEEG = 0, accNetz = 0;
    const chartData: ComboBarLineRow[] = sortedKeys.map(k => {
      const v = byKey.get(k)!;
      if (cumulative) {
        accGen += v.gesamterzeugung;
        accEEG += v.bezugEEG;
        accNetz += v.bezugNetz;
        return { label: periodLabel(k, bucket), period: k, barA: accEEG, barB: accNetz, line: accGen };
      }
      return { label: periodLabel(k, bucket), period: k, barA: v.bezugEEG, barB: v.bezugNetz, line: v.gesamterzeugung };
    });

    return {
      chartData,
      // KPI totals are always the raw (non-cumulative) sums over the loaded range —
      // the Absolut/Kumuliert toggle only changes how the chart presents the series.
      totalGeneration: communityRows.reduce((s, r) => s + r.wh_total_generation, 0),
      totalSelf: memberRows.reduce((s, r) => s + r.wh_community, 0),
      totalRestbedarf: memberRows.reduce((s, r) => s + Math.max(0, r.wh_total_consumption - r.wh_community), 0),
      totalCommunityGen: communityRows.reduce((s, r) => s + r.wh_community, 0),
    };
  }, [memberRows, communityRows, cumulative, bucket]);

  // Panel 2 "Einspeisung & Abnahme" — the mirrored pairing: bars = member's OWN
  // generation split (Einspeisung EEG / Resteinspeisung Netz), line = EEG-wide
  // total consumption (Gesamtabnahme).
  const { chartData2, totalGesamtabnahme, totalEinspeisungEEG, totalResteinspeisung, totalEEGSelf, totalMemberGeneration } = useMemo(() => {
    const byKey = new Map<string, { einspeisungEEG: number; resteinspeisung: number; gesamtabnahme: number }>();
    for (const r of memberRows) {
      const k = keyFor(r.period, bucket);
      const einspeisungEEG = r.wh_community_gen;
      const resteinspeisung = Math.max(0, r.wh_total_generation - r.wh_community_gen);
      const entry = byKey.get(k) ?? { einspeisungEEG: 0, resteinspeisung: 0, gesamtabnahme: 0 };
      entry.einspeisungEEG += einspeisungEEG;
      entry.resteinspeisung += resteinspeisung;
      byKey.set(k, entry);
    }
    for (const r of communityRows) {
      const k = keyFor(r.period, bucket);
      const entry = byKey.get(k) ?? { einspeisungEEG: 0, resteinspeisung: 0, gesamtabnahme: 0 };
      entry.gesamtabnahme += r.wh_total_consumption;
      byKey.set(k, entry);
    }

    const sortedKeys = [...byKey.keys()].sort();
    let accEin = 0, accRest = 0, accAbn = 0;
    const chartData2: ComboBarLineRow[] = sortedKeys.map(k => {
      const v = byKey.get(k)!;
      if (cumulative) {
        accEin += v.einspeisungEEG;
        accRest += v.resteinspeisung;
        accAbn += v.gesamtabnahme;
        return { label: periodLabel(k, bucket), period: k, barA: accEin, barB: accRest, line: accAbn };
      }
      return { label: periodLabel(k, bucket), period: k, barA: v.einspeisungEEG, barB: v.resteinspeisung, line: v.gesamtabnahme };
    });

    return {
      chartData2,
      totalGesamtabnahme: communityRows.reduce((s, r) => s + r.wh_total_consumption, 0),
      totalEinspeisungEEG: memberRows.reduce((s, r) => s + r.wh_community_gen, 0),
      totalResteinspeisung: memberRows.reduce((s, r) => s + Math.max(0, r.wh_total_generation - r.wh_community_gen), 0),
      totalEEGSelf: communityRows.reduce((s, r) => s + r.wh_self, 0),
      totalMemberGeneration: memberRows.reduce((s, r) => s + r.wh_total_generation, 0),
    };
  }, [memberRows, communityRows, cumulative, bucket]);

  // Autarkiegrad = MEMBER's own self-sufficiency (their EEG-covered share of their
  // own consumption). Eigenverbrauchsanteil = EEG-wide (how much of the community's
  // total generation is matched into community consumption).
  const totalConsumption = totalSelf + totalRestbedarf;
  const autarkiegrad = totalConsumption > 0 ? (totalSelf / totalConsumption) * 100 : null;
  const eigenverbrauchsanteil = totalGeneration > 0 ? (totalCommunityGen / totalGeneration) * 100 : null;

  // Mirrored ratios for panel 2: Autarkiegrad EEG = the community's own
  // self-sufficiency (EEG-wide self-consumption ÷ EEG-wide total consumption).
  // Eigenverbrauchsanteil (Mitglied) = how much of THIS member's own generation
  // stays within the community vs. gets sold to the grid.
  const autarkiegradEEG = totalGesamtabnahme > 0 ? (totalEEGSelf / totalGesamtabnahme) * 100 : null;
  const eigenverbrauchsanteilMitglied = totalMemberGeneration > 0 ? (totalEinspeisungEEG / totalMemberGeneration) * 100 : null;

  const navLabel =
    granularity === "day" ? `${pad(navDay)}.${pad(navMonth)}.${navYear}` :
    granularity === "month" ? `${MONTHS[navMonth - 1]} ${navYear}` :
    granularity === "quarter" ? `Q${navQuarter} ${navYear}` :
    String(navYear);

  function navPrev() {
    if (granularity === "day") {
      const d = new Date(navYear, navMonth - 1, navDay - 1);
      setNavYear(d.getFullYear()); setNavMonth(d.getMonth() + 1); setNavDay(d.getDate());
    } else if (granularity === "month") {
      if (navMonth === 1) { setNavYear(y => y - 1); setNavMonth(12); }
      else setNavMonth(m => m - 1);
    } else if (granularity === "quarter") {
      if (navQuarter === 1) { setNavYear(y => y - 1); setNavQuarter(4); }
      else setNavQuarter(q => q - 1);
    } else {
      setNavYear(y => y - 1);
    }
  }

  function navNext() {
    if (granularity === "day") {
      const d = new Date(navYear, navMonth - 1, navDay + 1);
      setNavYear(d.getFullYear()); setNavMonth(d.getMonth() + 1); setNavDay(d.getDate());
    } else if (granularity === "month") {
      if (navMonth === 12) { setNavYear(y => y + 1); setNavMonth(1); }
      else setNavMonth(m => m + 1);
    } else if (granularity === "quarter") {
      if (navQuarter === 4) { setNavYear(y => y + 1); setNavQuarter(1); }
      else setNavQuarter(q => q + 1);
    } else {
      setNavYear(y => y + 1);
    }
  }

  return (
    <div className="space-y-4 mb-6">
      <h2 className="text-base font-semibold text-slate-900">Übersicht Gemeinschaft</h2>

      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
        <KpiCard label="Gesamterzeugung" value={fmtKwh(totalGeneration)} />
        <KpiCard label="Strombezug von EEG" value={fmtKwh(totalSelf)} />
        <KpiCard label="Strombezug aus Netz" value={fmtKwh(totalRestbedarf)} />
        <KpiCard label="Autarkiegrad" value={autarkiegrad != null ? `${autarkiegrad.toFixed(0)} %` : "—"} />
        <KpiCard label="Eigenverbrauchsanteil" value={eigenverbrauchsanteil != null ? `${eigenverbrauchsanteil.toFixed(0)} %` : "—"} />
      </div>

      <div className="bg-white rounded-xl border border-slate-200 p-6">
        <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
          <div className="flex gap-1 bg-slate-100 rounded-lg p-1">
            {(["day", "month", "quarter", "year"] as const).map(g => (
              <button
                key={g}
                onClick={() => setGranularity(g)}
                className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                  granularity === g ? "bg-white text-slate-900 shadow-sm" : "text-slate-600 hover:text-slate-900"
                }`}
              >
                {GRANULARITY_LABELS[g]}
              </button>
            ))}
          </div>

          <div className="flex items-center gap-1">
            <button
              onClick={navPrev}
              className="p-1.5 rounded-md hover:bg-slate-200 text-slate-600 transition-colors"
              aria-label="Vorheriger Zeitraum"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
            </button>
            <span className="text-sm font-medium text-slate-700 min-w-[110px] text-center">{navLabel}</span>
            <button
              onClick={navNext}
              className="p-1.5 rounded-md hover:bg-slate-200 text-slate-600 transition-colors"
              aria-label="Nächster Zeitraum"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </button>
          </div>

          <div className="flex gap-1 bg-slate-100 rounded-lg p-1">
            <button
              onClick={() => setCumulative(false)}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                !cumulative ? "bg-white text-slate-900 shadow-sm" : "text-slate-600 hover:text-slate-900"
              }`}
            >
              Absolut
            </button>
            <button
              onClick={() => setCumulative(true)}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                cumulative ? "bg-white text-slate-900 shadow-sm" : "text-slate-600 hover:text-slate-900"
              }`}
            >
              Kumuliert
            </button>
          </div>
        </div>

        {loading ? (
          <div className="px-6 py-16 text-center">
            <p className="text-slate-400 text-sm">Wird geladen…</p>
          </div>
        ) : error ? (
          <div className="px-6 py-8 text-center">
            <p className="text-red-500 text-sm">{error}</p>
          </div>
        ) : (
          <>
            <h3 className="text-sm font-medium text-slate-700 mb-2">Produktion &amp; Verbrauch</h3>
            <ProductionConsumptionChart
              data={chartData}
              barALabel="Strombezug von EEG"
              barBLabel="Strombezug aus Netz"
              lineLabel="Gesamterzeugung"
              lineHint="(EEG gesamt, rechte Achse)"
              barAColor="#3b82f6"
              barBColor="#f59e0b"
              lineColor="#059669"
            />
          </>
        )}
      </div>

      {/* Mirrored pairing: this member's OWN feed-in vs. the whole EEG's total off-take. */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
        <KpiCard label="Gesamtabnahme EEG" value={fmtKwh(totalGesamtabnahme)} />
        <KpiCard label="Einspeisung EEG" value={fmtKwh(totalEinspeisungEEG)} />
        <KpiCard label="Resteinspeisung Netz" value={fmtKwh(totalResteinspeisung)} />
        <KpiCard label="Autarkiegrad EEG" value={autarkiegradEEG != null ? `${autarkiegradEEG.toFixed(0)} %` : "—"} />
        <KpiCard label="Eigenverbrauchsanteil (Mitglied)" value={eigenverbrauchsanteilMitglied != null ? `${eigenverbrauchsanteilMitglied.toFixed(0)} %` : "—"} />
      </div>

      <div className="bg-white rounded-xl border border-slate-200 p-6">
        {loading ? (
          <div className="px-6 py-16 text-center">
            <p className="text-slate-400 text-sm">Wird geladen…</p>
          </div>
        ) : error ? (
          <div className="px-6 py-8 text-center">
            <p className="text-red-500 text-sm">{error}</p>
          </div>
        ) : (
          <>
            <h3 className="text-sm font-medium text-slate-700 mb-2">Einspeisung &amp; Abnahme</h3>
            <ProductionConsumptionChart
              data={chartData2}
              barALabel="Einspeisung EEG"
              barBLabel="Resteinspeisung Netz"
              lineLabel="Gesamtabnahme EEG"
              lineHint="(EEG gesamt, rechte Achse)"
              barAColor="#6366f1"
              barBColor="#a78bfa"
              lineColor="#0ea5e9"
            />
          </>
        )}
      </div>
    </div>
  );
}
