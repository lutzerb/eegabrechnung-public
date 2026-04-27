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

interface EnergySummaryChartProps {
  data: EnergySummaryRow[];
  granularity: "day" | "month" | "year" | "15min";
}

function EnergySummaryTooltip({ active, payload, label }: {
  active?: boolean;
  payload?: Array<{ name: string; value: number; fill: string }>;
  label?: string;
}) {
  if (!active || !payload?.length) return null;
  const ausgetauscht     = payload.find((p) => p.name === "Ausgetauscht")?.value ?? 0;
  const restbedarf       = payload.find((p) => p.name === "Restbedarf")?.value ?? 0;
  const eingespeist      = payload.find((p) => p.name === "Einspeisung EEG")?.value ?? 0;
  const resteinspeisung  = payload.find((p) => p.name === "Resteinspeisung")?.value ?? 0;
  const gesamtVerbrauch  = ausgetauscht + restbedarf;
  const gesamtEinspeis   = eingespeist + resteinspeisung;
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

export function EnergySummaryChart({ data, granularity }: EnergySummaryChartProps) {
  const mounted = useIsMounted();
  const [containerRef, width] = useContainerWidth();
  const labelFn =
    granularity === "15min" ? intervalLabel :
    granularity === "day"   ? dayLabel :
    granularity === "year"  ? (s: string) => new Date(s).getUTCFullYear().toString() :
    monthLabel;
  const xAxisInterval = granularity === "15min" ? 7 : 0;

  // For day/15min views values are typically < 10 kWh — show decimals.
  // For month/year views round to whole numbers (MWh scale).
  const yTickFormatter =
    granularity === "day" || granularity === "15min"
      ? (v: number) => v >= 1000 ? `${(v / 1000).toFixed(2)} MWh` : `${v % 1 === 0 ? v : v.toFixed(2)}`
      : (v: number) => v >= 1000 ? `${(v / 1000).toFixed(0)} MWh` : `${v}`;

  const chartData = data.map((r) => ({
    name:                labelFn(r.period),
    "Ausgetauscht":      r.wh_self,
    "Restbedarf":        r.wh_restbedarf,
    "Einspeisung EEG":   r.wh_community,
    "Resteinspeisung":   r.wh_resteinspeisung,
  }));

  if (!mounted || chartData.length === 0) {
    return <EmptyChart label="Keine Messdaten für den gewählten Zeitraum." />;
  }

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
      </div>

      <div ref={containerRef} style={{ width: "100%", height: 300 }}>
        {width > 0 && (
          <BarChart width={width} height={300} data={chartData} margin={{ top: 4, right: 8, left: 0, bottom: 4 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
            <XAxis orientation="bottom" type="category" scale="auto" height={30} mirror={false} dataKey="name" tick={{ fontSize: 12, fill: "#64748b" }} interval={xAxisInterval} />
            <YAxis
              orientation="left" type="number" scale="auto" mirror={false}
              tick={{ fontSize: 11, fill: "#94a3b8" }}
              tickFormatter={yTickFormatter}
              width={64}
            />
            <Tooltip content={<EnergySummaryTooltip />} />
            {/* Consumption stack: Restbedarf (Netz, bottom) + Ausgetauscht (EEG, top) = Gesamtverbrauch */}
            <Bar dataKey="Restbedarf"   stackId="cons" fill="#fbbf24" maxBarSize={40} minPointSize={0} isAnimationActive={false} />
            <Bar dataKey="Ausgetauscht" stackId="cons" fill="#10b981" maxBarSize={40} radius={[3, 3, 0, 0]} minPointSize={0} isAnimationActive={false} />
            {/* Generation stack: Resteinspeisung (Netz, bottom) + Einspeisung EEG (top) = Gesamteinspeisung */}
            <Bar dataKey="Resteinspeisung"  stackId="gen" fill="#c4b5fd" maxBarSize={40} minPointSize={0} isAnimationActive={false} />
            <Bar dataKey="Einspeisung EEG"  stackId="gen" fill="#6366f1" maxBarSize={40} radius={[3, 3, 0, 0]} minPointSize={0} isAnimationActive={false} />
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
