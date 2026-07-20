"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  Customized,
} from "recharts";
import type { MonthlyEnergyRow, MemberStat, EnergySummaryRow } from "@/lib/api";

function useIsMounted(): boolean {
  const [mounted, setMounted] = useState(false);
  useEffect(() => {
    setMounted(true);
  }, []);
  return mounted;
}

// Custom responsive container hook: measures the div width via a callback ref.
// Replaces recharts' ResponsiveContainer which has a broken state-update
// lifecycle in Next.js 16 / React 18.3.
// Uses a callback ref so measurement fires exactly when the div attaches to the DOM.
function useContainerWidth() {
  const [width, setWidth] = useState(0);
  const roRef = useRef<ResizeObserver | null>(null);

  const ref = useCallback((node: HTMLDivElement | null) => {
    // Disconnect previous observer when the node changes / unmounts
    if (roRef.current) {
      roRef.current.disconnect();
      roRef.current = null;
    }
    if (!node) return;
    // Measure immediately once mounted
    setWidth(Math.round(node.getBoundingClientRect().width));
    // Keep tracking resize
    const ro = new ResizeObserver(([entry]) => {
      setWidth(Math.round(entry.contentRect.width));
    });
    ro.observe(node);
    roRef.current = ro;
  }, []);

  return [ref, width] as const;
}

const MONTHS = ["Jan", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"];

function monthLabel(iso: string) {
  return MONTHS[new Date(iso).getUTCMonth()];
}

function dayLabel(iso: string) {
  const d = new Date(iso);
  return `${d.getUTCDate()}.${d.getUTCMonth() + 1}.`;
}

function intervalLabel(iso: string) {
  const d = new Date(iso);
  return `${String(d.getUTCHours()).padStart(2, "0")}:${String(d.getUTCMinutes()).padStart(2, "0")}`;
}

function fmtKwh(v: number) {
  if (v >= 100000)
    return new Intl.NumberFormat("de-AT", { maximumFractionDigits: 1 }).format(v / 1000) + " MWh";
  return new Intl.NumberFormat("de-AT", { maximumFractionDigits: 1 }).format(v) + " kWh";
}

function fmtEur(v: number) {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(v);
}

// ── Monthly energy flow chart ──────────────────────────────────────────────

interface EnergyChartProps {
  data: MonthlyEnergyRow[];
}

export function EnergyFlowChart({ data }: EnergyChartProps) {
  const mounted = useIsMounted();
  const [containerRef, width] = useContainerWidth();
  const chartData = data.map((r) => ({
    name: monthLabel(r.month),
    "Einspeisung (kWh)": r.generation_kwh,
    "Bezug (kWh)": r.consumption_kwh,
  }));

  if (!mounted || chartData.length === 0) {
    return <EmptyChart label="Keine Energiedaten für dieses Jahr." />;
  }

  return (
    <div ref={containerRef} style={{ width: "100%", height: 280 }}>
      {width > 0 && (
        <BarChart width={width} height={280} data={chartData} margin={{ top: 4, right: 8, left: 0, bottom: 4 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
          <XAxis orientation="bottom" type="category" scale="auto" height={30} mirror={false} dataKey="name" tick={{ fontSize: 12, fill: "#64748b" }} />
          <YAxis
            orientation="left" type="number" scale="auto" mirror={false}
            tick={{ fontSize: 11, fill: "#94a3b8" }}
            tickFormatter={(v) => `${(v / 1000).toFixed(0)} MWh`}
            width={64}
          />
          <Tooltip formatter={(v: number) => fmtKwh(v)} />
          <Legend iconType="circle" iconSize={8} />
          <Bar dataKey="Einspeisung (kWh)" fill="#10b981" radius={[3, 3, 0, 0]} maxBarSize={40} minPointSize={0} isAnimationActive={false} />
          <Bar dataKey="Bezug (kWh)" fill="#3b82f6" radius={[3, 3, 0, 0]} maxBarSize={40} minPointSize={0} isAnimationActive={false} />
        </BarChart>
      )}
    </div>
  );
}

// ── Monthly financial chart ────────────────────────────────────────────────

export function FinancialChart({ data }: EnergyChartProps) {
  const mounted = useIsMounted();
  const [containerRef, width] = useContainerWidth();
  const chartData = data.map((r) => ({
    name: monthLabel(r.month),
    Einnahmen: r.revenue,
    Gutschriften: r.payouts,
  }));

  if (!mounted || chartData.length === 0) {
    return <EmptyChart label="Keine Finanzdaten für dieses Jahr." />;
  }

  return (
    <div ref={containerRef} style={{ width: "100%", height: 240 }}>
      {width > 0 && (
        <BarChart width={width} height={240} data={chartData} margin={{ top: 4, right: 8, left: 0, bottom: 4 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
          <XAxis orientation="bottom" type="category" scale="auto" height={30} mirror={false} dataKey="name" tick={{ fontSize: 12, fill: "#64748b" }} />
          <YAxis
            orientation="left" type="number" scale="auto" mirror={false}
            tick={{ fontSize: 11, fill: "#94a3b8" }}
            tickFormatter={(v) => `€ ${v}`}
            width={64}
          />
          <Tooltip formatter={(v: number) => fmtEur(v)} />
          <Legend iconType="circle" iconSize={8} />
          <Bar dataKey="Einnahmen" fill="#3b82f6" radius={[3, 3, 0, 0]} maxBarSize={40} minPointSize={0} isAnimationActive={false} />
          <Bar dataKey="Gutschriften" fill="#f59e0b" radius={[3, 3, 0, 0]} maxBarSize={40} minPointSize={0} isAnimationActive={false} />
        </BarChart>
      )}
    </div>
  );
}

// ── Member horizontal bar chart ────────────────────────────────────────────

interface MemberChartProps {
  data: MemberStat[];
  memberNames: Record<string, string>;
}

export function MemberEnergyChart({ data, memberNames }: MemberChartProps) {
  const mounted = useIsMounted();
  const [containerRef, width] = useContainerWidth();
  const chartData = data
    .slice(0, 10)
    .map((s) => ({
      name: memberNames[s.member_id] || s.member_id.slice(0, 8),
      "Einspeisung (kWh)": s.generation_kwh,
      "Bezug (kWh)": s.consumption_kwh,
    }))
    .reverse();

  if (!mounted || chartData.length === 0) {
    return <EmptyChart label="Keine Mitgliederdaten vorhanden." />;
  }

  const chartHeight = Math.max(200, chartData.length * 52);
  return (
    <div ref={containerRef} style={{ width: "100%", height: chartHeight }}>
      {width > 0 && (
        <BarChart
          layout="vertical"
          width={width}
          height={chartHeight}
          data={chartData}
          margin={{ top: 4, right: 16, left: 8, bottom: 0 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" horizontal={false} />
          <XAxis
            orientation="bottom" type="number" scale="auto" height={30} mirror={false}
            tick={{ fontSize: 11, fill: "#94a3b8" }}
            tickFormatter={(v) => v >= 1000 ? `${(v / 1000).toFixed(1)} MWh` : `${v} kWh`}
          />
          <YAxis
            orientation="left" type="category" scale="auto" mirror={false}
            dataKey="name"
            tick={{ fontSize: 12, fill: "#475569" }}
            width={140}
          />
          <Tooltip formatter={(v: number) => fmtKwh(v)} />
          <Legend iconType="circle" iconSize={8} />
          <Bar dataKey="Einspeisung (kWh)" fill="#6366f1" radius={[0, 3, 3, 0]} maxBarSize={20} minPointSize={0} isAnimationActive={false} />
          <Bar dataKey="Bezug (kWh)" fill="#10b981" radius={[0, 3, 3, 0]} maxBarSize={20} minPointSize={0} isAnimationActive={false} />
        </BarChart>
      )}
    </div>
  );
}

// ── Year selector ──────────────────────────────────────────────────────────

interface YearSelectorProps {
  currentYear: number;
  availableYears: number[];
}

export function YearSelector({ currentYear, availableYears }: YearSelectorProps) {
  return (
    <select
      value={currentYear}
      onChange={(e) => {
        const url = new URL(window.location.href);
        url.searchParams.set("year", e.target.value);
        window.location.href = url.toString();
      }}
      className="text-sm border border-slate-200 rounded-lg px-3 py-1.5 bg-white text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
    >
      {availableYears.map((y) => (
        <option key={y} value={y}>
          {y}
        </option>
      ))}
    </select>
  );
}

// ── Energy Summary Chart ───────────────────────────────────────────────────
// Two grouped bar groups per time bucket:
//   "cons":  Ausgetauscht (green, EEG share) stacked with Restbedarf (amber, grid)
//            → total height = Gesamtverbrauch
//   "gen":   Eingespeist (indigo, fed into community pool)
// This makes the EEG coverage ratio immediately visible without double-counting.

export interface ForecastDayTotal {
  day: string;            // "YYYY-MM-DD"
  consumption_kwh: number;
  generation_kwh: number;
}

interface EnergySummaryChartProps {
  data: EnergySummaryRow[];
  granularity: "day" | "month" | "year" | "15min";
  forecastDailyTotals?: ForecastDayTotal[];
  // Requested range ("YYYY-MM-DD"), used to fill in gaps for periods with no
  // readings at all so missing days/months/etc. still show up (marked) on the
  // x-axis instead of silently disappearing.
  from?: string;
  to?: string;
}

// Truncates a period ISO string to the key granularity buckets align on.
function periodKeyFn(iso: string, granularity: "day" | "month" | "year" | "15min"): string {
  if (granularity === "15min") return iso.slice(0, 16); // "YYYY-MM-DDTHH:MM"
  if (granularity === "day") return iso.slice(0, 10);   // "YYYY-MM-DD"
  if (granularity === "year") return iso.slice(0, 4);   // "YYYY"
  return iso.slice(0, 7);                                // "YYYY-MM"
}

// Reconstructs a representative ISO instant for a period key, for reuse with the
// existing labelFn helpers (dayLabel/monthLabel/intervalLabel all read UTC getters).
function canonicalIso(key: string, granularity: "day" | "month" | "year" | "15min"): string {
  if (granularity === "15min") return `${key}:00Z`;
  if (granularity === "day") return `${key}T00:00:00Z`;
  if (granularity === "year") return `${key}-01-01T00:00:00Z`;
  return `${key}-01T00:00:00Z`;
}

// Enumerates every expected period key between from/to (inclusive) at the given
// granularity, so gaps in the underlying data can be detected and marked.
function enumeratePeriods(from: string, to: string, granularity: "day" | "month" | "year" | "15min"): string[] {
  const fromD = new Date(`${from}T00:00:00Z`);
  const toD = new Date(`${to}T00:00:00Z`);
  if (Number.isNaN(fromD.getTime()) || Number.isNaN(toD.getTime()) || fromD > toD) return [];
  const keys: string[] = [];

  if (granularity === "year") {
    for (let y = fromD.getUTCFullYear(); y <= toD.getUTCFullYear(); y++) keys.push(String(y));
    return keys;
  }
  if (granularity === "month") {
    let y = fromD.getUTCFullYear();
    let m = fromD.getUTCMonth();
    const endY = toD.getUTCFullYear();
    const endM = toD.getUTCMonth();
    while (y < endY || (y === endY && m <= endM)) {
      keys.push(`${y}-${String(m + 1).padStart(2, "0")}`);
      m++;
      if (m > 11) { m = 0; y++; }
    }
    return keys;
  }
  if (granularity === "day") {
    const cur = new Date(fromD);
    while (cur.getTime() <= toD.getTime()) {
      keys.push(cur.toISOString().slice(0, 10));
      cur.setUTCDate(cur.getUTCDate() + 1);
    }
    return keys;
  }
  // 15min — walk every calendar day in range, 96 slots each
  const cur = new Date(fromD);
  while (cur.getTime() <= toD.getTime()) {
    const dayStr = cur.toISOString().slice(0, 10);
    for (let h = 0; h < 24; h++) {
      for (let mm = 0; mm < 60; mm += 15) {
        keys.push(`${dayStr}T${String(h).padStart(2, "0")}:${String(mm).padStart(2, "0")}`);
      }
    }
    cur.setUTCDate(cur.getUTCDate() + 1);
  }
  return keys;
}

function EnergySummaryTooltip({ active, payload, label }: {
  active?: boolean;
  payload?: Array<{ name: string; value: number; fill: string }>;
  label?: string;
}) {
  if (!active || !payload?.length) return null;
  const val = (name: string) => payload.find((p) => p.name === name)?.value ?? 0;

  const ausgetauscht    = val("Ausgetauscht");
  const restbedarf      = val("Restbedarf");
  const eingespeist     = val("Einspeisung EEG");
  const resteinspeisung = val("Resteinspeisung");
  const gesamtVerbrauch = ausgetauscht + restbedarf;
  const gesamtEinspeis  = eingespeist + resteinspeisung;

  const verbrauchProg   = val("Verbrauch Prognose");
  const erzeugungProg   = val("Erzeugung Prognose");
  const isForecast      = verbrauchProg > 0 || erzeugungProg > 0;
  const isMissing       = val("Keine Daten") > 0;

  if (isMissing) {
    return (
      <div className="bg-white border border-slate-200 rounded-lg shadow-sm px-3 py-2.5 text-xs min-w-44">
        <p className="font-semibold text-slate-700 mb-1">{label}</p>
        <p className="text-slate-400 flex items-center gap-1.5">
          <span className="w-2 h-2 rounded-sm bg-slate-300 inline-block" />
          Keine Messdaten für diesen Zeitraum
        </p>
      </div>
    );
  }

  if (isForecast) {
    return (
      <div className="bg-white border border-indigo-100 rounded-lg shadow-sm px-3 py-2.5 text-xs min-w-44">
        <p className="font-semibold text-slate-700 mb-1">{label}</p>
        <p className="text-slate-400 mb-2 text-[10px]">Prognose</p>
        <div className="space-y-1">
          {verbrauchProg > 0 && (
            <div className="flex justify-between gap-4">
              <span className="flex items-center gap-1 text-emerald-600">
                <span className="w-2 h-2 rounded-sm bg-emerald-400 inline-block opacity-60" />
                Verbrauch
              </span>
              <span className="font-medium text-emerald-700">~{fmtKwh(verbrauchProg)}</span>
            </div>
          )}
          {erzeugungProg > 0 && (
            <div className="flex justify-between gap-4">
              <span className="flex items-center gap-1 text-indigo-600">
                <span className="w-2 h-2 rounded-sm bg-indigo-400 inline-block opacity-60" />
                Erzeugung
              </span>
              <span className="font-medium text-indigo-700">~{fmtKwh(erzeugungProg)}</span>
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white border border-slate-200 rounded-lg shadow-sm px-3 py-2.5 text-xs min-w-44">
      <p className="font-semibold text-slate-700 mb-2">{label}</p>
      <div className="space-y-1">
        <div className="flex justify-between gap-4">
          <span className="text-slate-500">Gesamtverbrauch</span>
          <span className="font-medium text-slate-800">{fmtKwh(gesamtVerbrauch)}</span>
        </div>
        <div className="flex justify-between gap-4 pl-2">
          <span className="flex items-center gap-1 text-emerald-600">
            <span className="w-2 h-2 rounded-sm bg-emerald-500 inline-block" />
            davon Ausgetauscht
          </span>
          <span className="font-medium text-emerald-700">{fmtKwh(ausgetauscht)}</span>
        </div>
        <div className="flex justify-between gap-4 pl-2">
          <span className="flex items-center gap-1 text-amber-600">
            <span className="w-2 h-2 rounded-sm bg-amber-400 inline-block" />
            davon Restbedarf
          </span>
          <span className="font-medium text-amber-700">{fmtKwh(restbedarf)}</span>
        </div>
        {gesamtEinspeis > 0 && (
          <>
            <div className="flex justify-between gap-4 border-t border-slate-100 pt-1 mt-1">
              <span className="text-slate-500">Gesamteinspeisung</span>
              <span className="font-medium text-slate-800">{fmtKwh(gesamtEinspeis)}</span>
            </div>
            <div className="flex justify-between gap-4 pl-2">
              <span className="flex items-center gap-1 text-indigo-600">
                <span className="w-2 h-2 rounded-sm bg-indigo-500 inline-block" />
                davon in EEG
              </span>
              <span className="font-medium text-indigo-700">{fmtKwh(eingespeist)}</span>
            </div>
            <div className="flex justify-between gap-4 pl-2">
              <span className="flex items-center gap-1 text-violet-400">
                <span className="w-2 h-2 rounded-sm bg-violet-300 inline-block" />
                davon ins Netz
              </span>
              <span className="font-medium text-violet-500">{fmtKwh(resteinspeisung)}</span>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

// Custom hatched bar shape for forecast entries.
// Defines an SVG pattern inline within the same <svg> so url(#id) resolves correctly.
function makeForecastShape(color: string, patternId: string) {
  return function ForecastBar(props: {
    x?: number; y?: number; width?: number; height?: number; [k: string]: unknown;
  }) {
    const { x = 0, y = 0, width = 0, height = 0 } = props;
    if ((height as number) <= 0 || (width as number) <= 0) return <g />;
    const h = Math.max(0, height as number);
    return (
      <g>
        <defs>
          <pattern id={patternId} patternUnits="userSpaceOnUse" width="6" height="6" patternTransform="rotate(45)">
            <rect width="6" height="6" fill="white" fillOpacity={0.55} />
            <line x1="0" y1="0" x2="0" y2="6" stroke={color} strokeWidth="2.5" strokeOpacity={0.55} />
          </pattern>
        </defs>
        <rect x={x as number} y={y as number} width={width as number} height={h} fill={`url(#${patternId})`} rx={2} />
      </g>
    );
  };
}

const forecastConsShape = makeForecastShape("#10b981", "eeg-fc-cons");
const forecastGenShape  = makeForecastShape("#6366f1", "eeg-fc-gen");

// Gray cross-hatch marker for periods with no readings at all — visually
// distinct from the (colored, single-direction) forecast hatch above.
// The <pattern> it references is defined once in <MissingHatchDef />, not here —
// one <Bar> per missing period would otherwise redefine the same id repeatedly,
// which is invalid SVG and caused a stray misplaced square during re-renders.
function MissingBarShape(props: { x?: number; y?: number; width?: number; height?: number; [k: string]: unknown }) {
  const { x, y, width, height } = props;
  // Recharts briefly reuses stale bar-rectangle nodes with a missing x (null,
  // not undefined — so a destructuring default won't catch it) while
  // reconciling a granularity switch (different category count). Bail out
  // rather than let it fall back to x=0, which drew a stray square at the
  // chart's left edge.
  if (x == null || y == null || !width || !height || (width as number) <= 0 || (height as number) <= 0) {
    return null;
  }
  return (
    <rect
      x={x} y={y} width={width} height={height}
      fill="url(#eeg-missing-hatch)" stroke="#cbd5e1" strokeWidth={1} strokeDasharray="2 2" rx={2}
    />
  );
}

function MissingHatchDef() {
  return (
    <defs>
      <pattern id="eeg-missing-hatch" patternUnits="userSpaceOnUse" width="6" height="6" patternTransform="rotate(-45)">
        <rect width="6" height="6" fill="#f8fafc" />
        <line x1="0" y1="0" x2="0" y2="6" stroke="#cbd5e1" strokeWidth="2" />
      </pattern>
    </defs>
  );
}

export function EnergySummaryChart({ data, granularity, forecastDailyTotals, from, to }: EnergySummaryChartProps) {
  const mounted = useIsMounted();
  const [containerRef, width] = useContainerWidth();
  const labelFn =
    granularity === "15min" ? intervalLabel :
    granularity === "day"   ? dayLabel :
    granularity === "year"  ? (s: string) => new Date(s).getUTCFullYear().toString() :
    monthLabel;
  const xAxisInterval = granularity === "15min" ? 7 : 0;

  const yTickFormatter =
    granularity === "day" || granularity === "15min"
      ? (v: number) => v >= 1000 ? `${(v / 1000).toFixed(2)} MWh` : `${v % 1 === 0 ? v : v.toFixed(2)}`
      : (v: number) => v >= 1000 ? `${(v / 1000).toFixed(0)} MWh` : `${v}`;

  // Merge historical and forecast data.
  // Forecast bars are only shown for "day" granularity (month view with daily bars).
  const hasForecast = granularity === "day" && !!forecastDailyTotals?.length;

  const historicalDays = new Set(data.map((r) => r.period.slice(0, 10)));

  const historicalItems = data.map((r) => ({
    name:                  labelFn(r.period),
    period:                periodKeyFn(r.period, granularity),
    "Ausgetauscht":        r.wh_self,
    "Restbedarf":          r.wh_restbedarf,
    "Einspeisung EEG":     r.wh_community,
    "Resteinspeisung":     r.wh_resteinspeisung,
    "Verbrauch Prognose":  0,
    "Erzeugung Prognose":  0,
    "Keine Daten":         0,
  }));

  const forecastItems = hasForecast
    ? (forecastDailyTotals ?? [])
        .filter((f) => !historicalDays.has(f.day))
        .map((f) => ({
          name:                  dayLabel(f.day + "T00:00:00Z"),
          period:                f.day,
          "Ausgetauscht":        0,
          "Restbedarf":          0,
          "Einspeisung EEG":     0,
          "Resteinspeisung":     0,
          "Verbrauch Prognose":  f.consumption_kwh,
          "Erzeugung Prognose":  f.generation_kwh,
          "Keine Daten":         0,
        }))
    : [];

  if (!mounted || (historicalItems.length === 0 && forecastItems.length === 0)) {
    return <EmptyChart label="Keine Messdaten für den gewählten Zeitraum." />;
  }

  // Fill gaps: any expected period in [from, to] with neither real nor forecast
  // data gets a marked placeholder bar so it still shows up on the x-axis.
  const coveredKeys = new Set([
    ...historicalItems.map((r) => r.period),
    ...forecastItems.map((r) => r.period),
  ]);
  const maxCons = Math.max(
    0,
    ...historicalItems.map((r) => r["Ausgetauscht"] + r["Restbedarf"]),
    ...forecastItems.map((r) => r["Verbrauch Prognose"])
  );
  const maxGen = Math.max(
    0,
    ...historicalItems.map((r) => r["Einspeisung EEG"] + r["Resteinspeisung"]),
    ...forecastItems.map((r) => r["Erzeugung Prognose"])
  );
  const placeholderHeight = Math.max(maxCons, maxGen) * 0.12 || 1;

  const missingItems = (from && to ? enumeratePeriods(from, to, granularity) : [])
    .filter((key) => !coveredKeys.has(key))
    .map((key) => ({
      name:                  labelFn(canonicalIso(key, granularity)),
      period:                key,
      "Ausgetauscht":        0,
      "Restbedarf":          0,
      "Einspeisung EEG":     0,
      "Resteinspeisung":     0,
      "Verbrauch Prognose":  0,
      "Erzeugung Prognose":  0,
      "Keine Daten":         placeholderHeight,
    }));

  const chartData = [...historicalItems, ...forecastItems, ...missingItems].sort((a, b) =>
    a.period.localeCompare(b.period)
  );
  const hasMissing = missingItems.length > 0;

  return (
    <div>
      {/* Legend */}
      <div className="flex flex-wrap gap-4 mb-4 text-xs text-slate-500">
        <div className="flex items-center gap-1.5">
          <span className="w-3 h-2.5 rounded-sm inline-block bg-emerald-500" />
          <span>Ausgetauscht <span className="text-slate-400">(EEG-Anteil am Verbrauch)</span></span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="w-3 h-2.5 rounded-sm inline-block bg-amber-400" />
          <span>Restbedarf <span className="text-slate-400">(aus dem Netz)</span></span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="w-3 h-2.5 rounded-sm inline-block bg-indigo-500" />
          <span>Einspeisung EEG <span className="text-slate-400">(in EEG-Pool)</span></span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="w-3 h-2.5 rounded-sm inline-block bg-violet-300" />
          <span>Resteinspeisung <span className="text-slate-400">(ins Netz)</span></span>
        </div>
        {hasForecast && (
          <>
            <div className="flex items-center gap-1.5">
              <svg width="12" height="10"><rect width="12" height="10" fill="url(#eeg-fc-cons-legend)" rx={2} /><defs><pattern id="eeg-fc-cons-legend" patternUnits="userSpaceOnUse" width="6" height="6" patternTransform="rotate(45)"><rect width="6" height="6" fill="white" fillOpacity={0.55} /><line x1="0" y1="0" x2="0" y2="6" stroke="#10b981" strokeWidth="2.5" strokeOpacity={0.55} /></pattern></defs></svg>
              <span>Verbrauch <span className="text-slate-400">(Prognose)</span></span>
            </div>
            <div className="flex items-center gap-1.5">
              <svg width="12" height="10"><rect width="12" height="10" fill="url(#eeg-fc-gen-legend)" rx={2} /><defs><pattern id="eeg-fc-gen-legend" patternUnits="userSpaceOnUse" width="6" height="6" patternTransform="rotate(45)"><rect width="6" height="6" fill="white" fillOpacity={0.55} /><line x1="0" y1="0" x2="0" y2="6" stroke="#6366f1" strokeWidth="2.5" strokeOpacity={0.55} /></pattern></defs></svg>
              <span>Erzeugung <span className="text-slate-400">(Prognose)</span></span>
            </div>
          </>
        )}
        {hasMissing && (
          <div className="flex items-center gap-1.5">
            <svg width="12" height="10"><rect width="12" height="10" fill="url(#eeg-missing-legend)" stroke="#cbd5e1" strokeWidth={1} strokeDasharray="2 2" rx={2} /><defs><pattern id="eeg-missing-legend" patternUnits="userSpaceOnUse" width="6" height="6" patternTransform="rotate(-45)"><rect width="6" height="6" fill="#f8fafc" /><line x1="0" y1="0" x2="0" y2="6" stroke="#cbd5e1" strokeWidth="2" /></pattern></defs></svg>
            <span>Keine Daten <span className="text-slate-400">(Lücke)</span></span>
          </div>
        )}
      </div>

      <div ref={containerRef} style={{ width: "100%", height: 300 }}>
        {width > 0 && (
          <BarChart key={granularity} width={width} height={300} data={chartData} margin={{ top: 4, right: 8, left: 0, bottom: 4 }}>
            <Customized component={MissingHatchDef} />
            <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
            <XAxis orientation="bottom" type="category" scale="auto" height={30} mirror={false} dataKey="name" tick={{ fontSize: 12, fill: "#64748b" }} interval={xAxisInterval} />
            <YAxis
              orientation="left" type="number" scale="auto" mirror={false}
              tick={{ fontSize: 11, fill: "#94a3b8" }}
              tickFormatter={yTickFormatter}
              width={64}
            />
            <Tooltip content={<EnergySummaryTooltip />} />
            {/* Consumption stack (historical + forecast in same stackId so width is consistent) */}
            <Bar dataKey="Restbedarf"          stackId="cons" fill="#fbbf24"  maxBarSize={40} minPointSize={0} isAnimationActive={false} />
            <Bar dataKey="Ausgetauscht"        stackId="cons" fill="#10b981"  maxBarSize={40} radius={[3, 3, 0, 0]} minPointSize={0} isAnimationActive={false} />
            <Bar dataKey="Verbrauch Prognose"  stackId="cons" shape={forecastConsShape as never} maxBarSize={40} minPointSize={0} isAnimationActive={false} />
            {/* Generation stack */}
            <Bar dataKey="Resteinspeisung"     stackId="gen"  fill="#c4b5fd"  maxBarSize={40} minPointSize={0} isAnimationActive={false} />
            <Bar dataKey="Einspeisung EEG"     stackId="gen"  fill="#6366f1"  maxBarSize={40} radius={[3, 3, 0, 0]} minPointSize={0} isAnimationActive={false} />
            <Bar dataKey="Erzeugung Prognose"  stackId="gen"  shape={forecastGenShape as never} maxBarSize={40} minPointSize={0} isAnimationActive={false} />
            {/* Gap markers — rendered in both stacks so a missing period shows a hatch on either side */}
            <Bar dataKey="Keine Daten"         stackId="cons" shape={MissingBarShape as never} maxBarSize={40} minPointSize={0} isAnimationActive={false} legendType="none" />
            <Bar dataKey="Keine Daten"         stackId="gen"  shape={MissingBarShape as never} maxBarSize={40} minPointSize={0} isAnimationActive={false} legendType="none" />
          </BarChart>
        )}
      </div>
    </div>
  );
}

// ── Shared empty state ─────────────────────────────────────────────────────

function EmptyChart({ label }: { label: string }) {
  return (
    <div className="flex items-center justify-center h-40 text-sm text-slate-400">{label}</div>
  );
}
