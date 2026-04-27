"use client";

import { useState, useEffect, useCallback } from "react";
import { useSession } from "next-auth/react";
import { useParams } from "next/navigation";
import Link from "next/link";
import {
  EnergySummaryChart,
  EnergyFlowChart,
  FinancialChart,
  MemberEnergyChart,
} from "@/components/energy-charts";
import type {
  EnergySummaryRow,
  MonthlyEnergyRow,
  MemberStat,
  Member,
} from "@/lib/api";

interface Props {
  params: { eegId: string };
}

type PeriodMode = "year" | "month" | "day" | "custom";
type DataSource = "raw" | "billed";
type Granularity = "day" | "month" | "year" | "15min";

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(amount);
}

function formatKwh(kwh: number, decimals = 1): string {
  if (kwh >= 100000) {
    return (
      new Intl.NumberFormat("de-AT", { maximumFractionDigits: decimals }).format(kwh / 1000) + " MWh"
    );
  }
  return (
    new Intl.NumberFormat("de-AT", { maximumFractionDigits: decimals }).format(kwh) + " kWh"
  );
}

function KpiCard({ label, value, sub, accent }: { label: string; value: string; sub?: string; accent?: string }) {
  return (
    <div className="bg-white rounded-xl border border-slate-200 p-5">
      <p className="text-xs font-medium text-slate-500 uppercase tracking-wide">{label}</p>
      <p className={`text-2xl font-bold mt-1 ${accent ?? "text-slate-900"}`}>{value}</p>
      {sub && <p className="text-xs text-slate-400 mt-0.5">{sub}</p>}
    </div>
  );
}

function memberRole(s: MemberStat): "Produzent" | "Konsument" | "Prosument" {
  if (s.generation_kwh > 0 && s.consumption_kwh === 0) return "Produzent";
  if (s.generation_kwh > 0 && s.consumption_kwh > 0) return "Prosument";
  return "Konsument";
}

function isoDate(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

// ── Download helpers ───────────────────────────────────────────────────────

function triggerCSVDownload(rows: (string | number)[][], filename: string) {
  const content = rows
    .map((r) => r.map((c) => `"${String(c).replace(/"/g, '""')}"`).join(";"))
    .join("\r\n");
  const blob = new Blob(["\ufeff" + content], { type: "text/csv;charset=utf-8;" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

async function triggerXLSXDownload(sheets: { name: string; rows: (string | number)[][] }[], filename: string) {
  // Dynamic import to keep initial bundle lean
  const ExcelJS = await import("exceljs");
  const wb = new ExcelJS.Workbook();
  for (const { name, rows } of sheets) {
    const ws = wb.addWorksheet(name);
    for (const row of rows) {
      ws.addRow(row);
    }
  }
  const data = await wb.xlsx.writeBuffer();
  const blob = new Blob([data], {
    type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

function formatPeriodLabel(iso: string, granularity: Granularity): string {
  if (granularity === "15min") {
    const d = new Date(iso);
    return `${d.toISOString().slice(0, 10)} ${String(d.getUTCHours()).padStart(2, "0")}:${String(d.getUTCMinutes()).padStart(2, "0")}`;
  }
  if (granularity === "day") return iso.slice(0, 10);
  if (granularity === "month") return iso.slice(0, 7);
  return iso.slice(0, 4);
}

function DownloadButtons({
  onCSV,
  onXLSX,
}: {
  onCSV: () => void;
  onXLSX: () => void;
}) {
  return (
    <div className="flex items-center gap-1.5">
      <span className="text-xs text-slate-400 mr-0.5">Export:</span>
      <button
        onClick={onCSV}
        className="px-2.5 py-1 text-xs font-medium border border-slate-200 rounded text-slate-600 hover:bg-slate-50 transition-colors"
      >
        CSV
      </button>
      <button
        onClick={onXLSX}
        className="px-2.5 py-1 text-xs font-medium border border-slate-200 rounded text-slate-600 hover:bg-slate-50 transition-colors"
      >
        XLSX
      </button>
    </div>
  );
}

// ── Main page ──────────────────────────────────────────────────────────────

export default function ReportsPage() {
  const { eegId } = useParams<{ eegId: string }>();
  const { data: session } = useSession();

  // Period state
  const currentYear = new Date().getFullYear();
  const [periodMode, setPeriodMode] = useState<PeriodMode>("year");
  const [selectedYear, setSelectedYear] = useState(currentYear);
  const [selectedMonth, setSelectedMonth] = useState(() => {
    const now = new Date();
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
  });
  const [selectedDay, setSelectedDay] = useState(isoDate(new Date()));
  const [customFrom, setCustomFrom] = useState(isoDate(new Date(currentYear, 0, 1)));
  const [customTo, setCustomTo] = useState(isoDate(new Date(currentYear, 11, 31)));

  // Source toggle
  const [dataSource, setDataSource] = useState<DataSource>("raw");

  // Member filter
  const [selectedMemberId, setSelectedMemberId] = useState<string>("");
  const [members, setMembers] = useState<Member[]>([]);

  // Data
  const [summaryData, setSummaryData] = useState<EnergySummaryRow[]>([]);
  const [monthlyBilled, setMonthlyBilled] = useState<MonthlyEnergyRow[]>([]);
  const [memberStats, setMemberStats] = useState<MemberStat[]>([]);
  const [memberNames, setMemberNames] = useState<Record<string, string>>({});
  const [eegName, setEegName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const availableYears = Array.from({ length: 5 }, (_, i) => currentYear - i);

  // Compute date range and granularity from period state
  const getRange = useCallback((): { from: string; to: string; granularity: Granularity } => {
    if (periodMode === "year") {
      return {
        from: `${selectedYear}-01-01`,
        to: `${selectedYear}-12-31`,
        granularity: "month",
      };
    }
    if (periodMode === "month") {
      const [y, m] = selectedMonth.split("-").map(Number);
      const lastDay = new Date(y, m, 0).getDate();
      return {
        from: `${selectedMonth}-01`,
        to: `${selectedMonth}-${String(lastDay).padStart(2, "0")}`,
        granularity: "day",
      };
    }
    if (periodMode === "day") {
      return {
        from: selectedDay,
        to: selectedDay,
        granularity: "15min",
      };
    }
    // custom
    const fromDate = new Date(customFrom);
    const toDate = new Date(customTo);
    const diffDays = (toDate.getTime() - fromDate.getTime()) / (1000 * 60 * 60 * 24);
    const granularity: Granularity = diffDays <= 62 ? "day" : "month";
    return { from: customFrom, to: customTo, granularity };
  }, [periodMode, selectedYear, selectedMonth, selectedDay, customFrom, customTo]);

  const token = session?.accessToken ?? "";
  const authHeaders: Record<string, string> = token
    ? { Authorization: `Bearer ${token}` }
    : {};

  const load = useCallback(async () => {
    if (!session?.accessToken) return;
    setLoading(true);
    setError(null);

    const { from, to, granularity } = getRange();

    try {
      // Always load EEG name + member names (once)
      const [eegRes, membersRes] = await Promise.all([
        fetch(`/api/eegs/${eegId}`, { headers: authHeaders }),
        fetch(`/api/eegs/${eegId}/members`, { headers: authHeaders }),
      ]);
      if (eegRes.ok) {
        const eeg = await eegRes.json();
        setEegName(eeg.name || "");
      }
      if (membersRes.ok) {
        const memberList: Member[] = await membersRes.json();
        setMembers(memberList);
        setMemberNames(
          Object.fromEntries(
            memberList.map((m) => [m.id, [m.name1, m.name2].filter(Boolean).join(" ")])
          )
        );
      }

      if (dataSource === "raw") {
        const memberParam = selectedMemberId ? `&member_id=${selectedMemberId}` : "";
        const [summaryRes, memberStatsRes] = await Promise.all([
          fetch(
            `/api/eegs/${eegId}/energy/summary?from=${from}&to=${to}&granularity=${granularity}${memberParam}`,
            { headers: authHeaders }
          ),
          // Raw mode: use actual readings per member (works for any date range incl. single day)
          fetch(`/api/eegs/${eegId}/energy/members?from=${from}&to=${to}${memberParam}`, {
            headers: authHeaders,
          }),
        ]);
        setSummaryData(summaryRes.ok ? await summaryRes.json() : []);
        setMemberStats(memberStatsRes.ok ? await memberStatsRes.json() : []);
        setMonthlyBilled([]);
      } else {
        // billed — use year view only (invoices are monthly)
        const year = periodMode === "year" ? selectedYear : new Date(from).getFullYear();
        const [energyRes, memberStatsRes] = await Promise.all([
          fetch(`/api/eegs/${eegId}/reports/energy?year=${year}`, { headers: authHeaders }),
          fetch(`/api/eegs/${eegId}/reports/members?from=${from}&to=${to}`, {
            headers: authHeaders,
          }),
        ]);
        setMonthlyBilled(energyRes.ok ? await energyRes.json() : []);
        setMemberStats(memberStatsRes.ok ? await memberStatsRes.json() : []);
        setSummaryData([]);
      }
    } catch (e: unknown) {
      setError((e as Error).message || "Fehler beim Laden der Daten.");
    } finally {
      setLoading(false);
    }
  }, [session, eegId, dataSource, getRange, periodMode, selectedYear, selectedMemberId]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    load();
  }, [load]);

  const { from, to, granularity } = getRange();

  // KPIs from raw summary
  const totalSelf = summaryData.reduce((s, r) => s + r.wh_self, 0);
  const totalCommunity = summaryData.reduce((s, r) => s + r.wh_community, 0);
  const totalConsumption = summaryData.reduce((s, r) => s + r.wh_total_consumption, 0);
  const totalRestbedarf = summaryData.reduce((s, r) => s + r.wh_restbedarf, 0);
  const totalGeneration = summaryData.reduce((s, r) => s + r.wh_total_generation, 0);
  const totalResteinspeisung = summaryData.reduce((s, r) => s + r.wh_resteinspeisung, 0);
  const deckung = totalConsumption > 0 ? (totalSelf / totalConsumption) * 100 : null;

  // KPIs from billed monthly
  const billedConsumption = monthlyBilled.reduce((s, r) => s + r.consumption_kwh, 0);
  const billedGeneration = monthlyBilled.reduce((s, r) => s + r.generation_kwh, 0);
  const billedRevenue = monthlyBilled.reduce((s, r) => s + r.revenue, 0);
  const billedPayouts = monthlyBilled.reduce((s, r) => s + r.payouts, 0);
  const billedSaldo = billedRevenue - billedPayouts;

  // ── Download handlers ──────────────────────────────────────────────────

  function buildEnergyRows(): (string | number)[][] {
    const header = ["Zeitpunkt", "Ausgetauscht (kWh)", "Eingespeist (kWh)", "Gesamtverbrauch (kWh)", "Restbedarf (kWh)"];
    const rows = summaryData.map((r) => [
      formatPeriodLabel(r.period, granularity),
      +r.wh_self.toFixed(3),
      +r.wh_community.toFixed(3),
      +r.wh_total_consumption.toFixed(3),
      +r.wh_restbedarf.toFixed(3),
    ]);
    return [header, ...rows];
  }

  function buildBilledRows(): (string | number)[][] {
    const header = ["Monat", "Bezug (kWh)", "Einspeisung (kWh)", "Einnahmen (€)", "Gutschriften (€)"];
    const rows = monthlyBilled.map((r) => [
      r.month.slice(0, 7),
      +r.consumption_kwh.toFixed(3),
      +r.generation_kwh.toFixed(3),
      +r.revenue.toFixed(2),
      +r.payouts.toFixed(2),
    ]);
    return [header, ...rows];
  }

  function buildMemberRows(): (string | number)[][] {
    const header = dataSource === "billed"
      ? ["Mitglied", "Typ", "Einspeisung (kWh)", "Bezug (kWh)", "Saldo (€)"]
      : ["Mitglied", "Typ", "Bezug EEG (kWh)", "Restbezug (kWh)", "Einsp. EEG (kWh)", "Resteinsp. (kWh)"];
    const rows = displayedMemberStats.map((s) => [
      memberNames[s.member_id] || s.member_id.slice(0, 8),
      memberRole(s),
      ...(dataSource === "billed"
        ? [+s.generation_kwh.toFixed(3), +s.consumption_kwh.toFixed(3), +s.total_amount.toFixed(2)]
        : [
            +s.consumption_kwh.toFixed(3),
            +(s.consumption_total_kwh - s.consumption_kwh).toFixed(3),
            +s.generation_kwh.toFixed(3),
            +(s.generation_total_kwh - s.generation_kwh).toFixed(3),
          ]),
    ]);
    return [header, ...rows];
  }

  const periodLabel = periodMode === "year" ? String(selectedYear)
    : periodMode === "month" ? selectedMonth
    : periodMode === "day" ? selectedDay
    : `${from}_${to}`;

  function handleChartCSV() {
    const rows = dataSource === "raw" ? buildEnergyRows() : buildBilledRows();
    triggerCSVDownload(rows, `energie_${periodLabel}.csv`);
  }

  async function handleChartXLSX() {
    const energyRows = dataSource === "raw" ? buildEnergyRows() : buildBilledRows();
    const sheetName = dataSource === "raw" ? "Energiedaten" : "Abrechnung";
    await triggerXLSXDownload(
      [{ name: sheetName, rows: energyRows }, { name: "Mitglieder", rows: buildMemberRows() }],
      `energie_${periodLabel}.xlsx`
    );
  }

  function handleMemberCSV() {
    triggerCSVDownload(buildMemberRows(), `mitglieder_${periodLabel}.csv`);
  }

  async function handleMemberXLSX() {
    await triggerXLSXDownload(
      [{ name: "Mitglieder", rows: buildMemberRows() }],
      `mitglieder_${periodLabel}.xlsx`
    );
  }

  // In billed mode, filter the member table client-side (raw mode filters via API).
  const displayedMemberStats = dataSource === "billed" && selectedMemberId
    ? memberStats.filter((s) => s.member_id === selectedMemberId)
    : memberStats;

  return (
    <div className="p-8">
      {/* Header */}
      <div className="mb-6">
        <Link href="/eegs" className="text-sm text-slate-500 hover:text-slate-700">
          Energiegemeinschaften
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">
          {eegName || eegId}
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Auswertungen</span>
      </div>

      <div className="flex items-start justify-between mb-6 flex-wrap gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Auswertungen</h1>
          <p className="text-slate-500 mt-1">
            Energieflüsse für{" "}
            {selectedMemberId && memberNames[selectedMemberId]
              ? memberNames[selectedMemberId]
              : eegName || "diese Energiegemeinschaft"}
          </p>
        </div>

        {/* Source toggle */}
        <div className="flex items-center rounded-lg border border-slate-200 bg-white overflow-hidden text-sm">
          {(["raw", "billed"] as DataSource[]).map((src) => (
            <button
              key={src}
              onClick={() => setDataSource(src)}
              className={`px-4 py-2 font-medium transition-colors ${
                dataSource === src
                  ? "bg-slate-900 text-white"
                  : "text-slate-600 hover:bg-slate-50"
              }`}
            >
              {src === "raw" ? "Ausgetauscht" : "Abgerechnet"}
            </button>
          ))}
        </div>
      </div>

      {/* Period selector */}
      <div className="bg-white border border-slate-200 rounded-xl p-4 mb-6 flex flex-wrap gap-4 items-end">
        {/* Mode tabs */}
        <div className="flex rounded-lg border border-slate-200 overflow-hidden text-sm">
          {(["year", "month", "day", "custom"] as PeriodMode[]).map((mode) => (
            <button
              key={mode}
              onClick={() => setPeriodMode(mode)}
              className={`px-3 py-1.5 font-medium transition-colors ${
                periodMode === mode ? "bg-slate-800 text-white" : "text-slate-600 hover:bg-slate-50"
              }`}
            >
              {mode === "year" ? "Jahr" : mode === "month" ? "Monat" : mode === "day" ? "Tag" : "Zeitraum"}
            </button>
          ))}
        </div>

        {periodMode === "year" && (
          <select
            value={selectedYear}
            onChange={(e) => setSelectedYear(Number(e.target.value))}
            className="px-3 py-1.5 text-sm border border-slate-200 rounded-lg bg-white text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {availableYears.map((y) => (
              <option key={y} value={y}>{y}</option>
            ))}
          </select>
        )}

        {periodMode === "month" && (
          <input
            type="month"
            value={selectedMonth}
            onChange={(e) => setSelectedMonth(e.target.value)}
            className="w-40 px-3 py-1.5 text-sm border border-slate-200 rounded-lg text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        )}

        {periodMode === "day" && (
          <div className="flex items-center gap-2">
            <input
              type="date"
              value={selectedDay}
              onChange={(e) => setSelectedDay(e.target.value)}
              className="px-3 py-1.5 text-sm border border-slate-200 rounded-lg text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <span className="text-xs text-slate-400">15-Minuten-Intervalle</span>
          </div>
        )}

        {periodMode === "custom" && (
          <div className="flex items-center gap-2">
            <input
              type="date"
              value={customFrom}
              onChange={(e) => setCustomFrom(e.target.value)}
              className="px-3 py-1.5 text-sm border border-slate-200 rounded-lg text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <span className="text-slate-400 text-sm">bis</span>
            <input
              type="date"
              value={customTo}
              min={customFrom}
              onChange={(e) => setCustomTo(e.target.value)}
              className="px-3 py-1.5 text-sm border border-slate-200 rounded-lg text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        )}

        {/* Member filter */}
        {members.length > 0 && (
          <div className="flex items-center gap-2">
            <label className="text-xs text-slate-500 whitespace-nowrap">Mitglied:</label>
            <select
              value={selectedMemberId}
              onChange={(e) => setSelectedMemberId(e.target.value)}
              className="px-3 py-1.5 text-sm border border-slate-200 rounded-lg bg-white text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">Alle Mitglieder</option>
              {members.map((m) => (
                <option key={m.id} value={m.id}>
                  {[m.name1, m.name2].filter(Boolean).join(" ")}
                </option>
              ))}
            </select>
            {selectedMemberId && (
              <button
                onClick={() => setSelectedMemberId("")}
                className="text-xs text-slate-400 hover:text-slate-600"
                title="Filter zurücksetzen"
              >
                ✕
              </button>
            )}
          </div>
        )}

        {loading && (
          <svg className="animate-spin h-4 w-4 text-slate-400 ml-2" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
        )}
      </div>

      {error && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
          {error}
        </div>
      )}

      {/* KPI cards */}
      {dataSource === "raw" ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7 gap-4 mb-6">
          <KpiCard label="Gesamtverbrauch" value={formatKwh(totalConsumption)} accent="text-blue-700" sub="inkl. Netzanteil" />
          <KpiCard label="Bezug EEG" value={formatKwh(totalSelf)} accent="text-emerald-700" sub="aus EEG-Pool" />
          <KpiCard label="Restbezug" value={formatKwh(totalRestbedarf)} accent="text-amber-700" sub="aus dem Netz" />
          <KpiCard label="EEG-Deckung" value={deckung != null ? `${deckung.toFixed(0)} %` : "—"} sub="Bezug EEG / Gesamt" />
          <KpiCard label="Gesamteinspeisung" value={formatKwh(totalGeneration)} accent="text-indigo-700" sub="physikalisch erzeugt" />
          <KpiCard label="Einspeisung EEG" value={formatKwh(totalCommunity)} accent="text-indigo-600" sub="in EEG-Pool" />
          <KpiCard label="Resteinspeisung" value={formatKwh(totalResteinspeisung)} accent="text-violet-600" sub="ins Netz" />
        </div>
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4 mb-6">
          <KpiCard label="Bezug abgr." value={formatKwh(billedConsumption)} accent="text-blue-700" />
          <KpiCard label="Einspeisung abgr." value={formatKwh(billedGeneration)} accent="text-emerald-700" />
          <KpiCard label="Einnahmen" value={formatCurrency(billedRevenue)} accent="text-blue-700" sub="Konsumenten" />
          <KpiCard label="Gutschriften" value={formatCurrency(billedPayouts)} accent="text-amber-700" sub="Produzenten" />
          <KpiCard label="Saldo" value={formatCurrency(billedSaldo)} accent={billedSaldo >= 0 ? "text-emerald-700" : "text-red-600"} sub="Einnahmen − Gutschriften" />
        </div>
      )}

      {/* Main chart */}
      <div className="bg-white rounded-xl border border-slate-200 p-5 mb-6">
        <div className="flex items-center justify-between mb-4 flex-wrap gap-2">
          <h2 className="text-sm font-semibold text-slate-700">
            Energiefluss {periodMode === "year" ? selectedYear : periodMode === "day" ? selectedDay : `${from} – ${to}`}
            {selectedMemberId && memberNames[selectedMemberId] ? ` — ${memberNames[selectedMemberId]}` : ""}
          </h2>
          <DownloadButtons onCSV={handleChartCSV} onXLSX={handleChartXLSX} />
        </div>
        {dataSource === "raw" ? (
          <EnergySummaryChart
            data={summaryData}
            granularity={granularity}
          />
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div>
              <p className="text-xs text-slate-500 mb-2">Bezug / Einspeisung kWh</p>
              <EnergyFlowChart data={monthlyBilled} />
            </div>
            <div>
              <p className="text-xs text-slate-500 mb-2">Finanzen</p>
              <FinancialChart data={monthlyBilled} />
            </div>
          </div>
        )}
      </div>

      {/* Member breakdown */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <div className="bg-white rounded-xl border border-slate-200 p-5 overflow-y-auto" style={{ maxHeight: "400px" }}>
          <h2 className="text-sm font-semibold text-slate-700 mb-4">Mitglieder — Energiemengen</h2>
          <MemberEnergyChart data={displayedMemberStats} memberNames={memberNames} />
        </div>

        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          <div className="px-5 py-4 border-b border-slate-100 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-slate-700">Mitglieder — Details</h2>
            {displayedMemberStats.length > 0 && (
              <DownloadButtons onCSV={handleMemberCSV} onXLSX={handleMemberXLSX} />
            )}
          </div>
          {displayedMemberStats.length === 0 ? (
            <div className="px-5 py-10 text-center text-sm text-slate-400">
              Keine Messdaten für diesen Zeitraum vorhanden.
            </div>
          ) : (
            <div className="overflow-x-auto overflow-y-auto" style={{ maxHeight: "370px" }}>
              <table className="w-full text-sm min-w-[560px]">
                <thead className="sticky top-0 z-10">
                  <tr className="bg-slate-50 border-b border-slate-100">
                    <th className="text-left px-4 py-2.5 text-xs font-medium text-slate-500">Mitglied</th>
                    <th className="text-left px-4 py-2.5 text-xs font-medium text-slate-500">Typ</th>
                    {dataSource === "raw" ? (
                      <>
                        <th className="text-right px-3 py-2.5 text-xs font-medium text-emerald-600">Bezug EEG</th>
                        <th className="text-right px-3 py-2.5 text-xs font-medium text-amber-600">Restbezug</th>
                        <th className="text-right px-3 py-2.5 text-xs font-medium text-indigo-600">Einsp. EEG</th>
                        <th className="text-right px-3 py-2.5 text-xs font-medium text-violet-500">Resteinsp.</th>
                      </>
                    ) : (
                      <>
                        <th className="text-right px-4 py-2.5 text-xs font-medium text-emerald-600">Einspeisung</th>
                        <th className="text-right px-4 py-2.5 text-xs font-medium text-blue-600">Bezug</th>
                        <th className="text-right px-4 py-2.5 text-xs font-medium text-slate-500">Saldo</th>
                      </>
                    )}
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-50">
                  {displayedMemberStats.map((s) => (
                    <tr key={s.member_id} className="hover:bg-slate-50 transition-colors">
                      <td className="px-4 py-2.5 font-medium text-slate-800">
                        {memberNames[s.member_id] || s.member_id.slice(0, 8)}
                      </td>
                      <td className="px-4 py-2.5">
                        <span className={`text-xs px-1.5 py-0.5 rounded font-medium ${
                          memberRole(s) === "Produzent"
                            ? "bg-emerald-50 text-emerald-700"
                            : memberRole(s) === "Prosument"
                            ? "bg-violet-50 text-violet-700"
                            : "bg-blue-50 text-blue-700"
                        }`}>
                          {memberRole(s)}
                        </span>
                      </td>
                      {dataSource === "raw" ? (
                        <>
                          <td className="px-3 py-2.5 text-right text-emerald-700 font-medium font-mono text-xs">
                            {s.consumption_kwh > 0 ? formatKwh(s.consumption_kwh) : "—"}
                          </td>
                          <td className="px-3 py-2.5 text-right text-amber-600 font-mono text-xs">
                            {s.consumption_total_kwh > 0 ? formatKwh(s.consumption_total_kwh - s.consumption_kwh) : "—"}
                          </td>
                          <td className="px-3 py-2.5 text-right text-indigo-700 font-medium font-mono text-xs">
                            {s.generation_kwh > 0 ? formatKwh(s.generation_kwh) : "—"}
                          </td>
                          <td className="px-3 py-2.5 text-right text-violet-500 font-mono text-xs">
                            {s.generation_total_kwh > 0 ? formatKwh(s.generation_total_kwh - s.generation_kwh) : "—"}
                          </td>
                        </>
                      ) : (
                        <>
                          <td className="px-4 py-2.5 text-right text-emerald-700 font-medium">
                            {s.generation_kwh > 0 ? formatKwh(s.generation_kwh) : "—"}
                          </td>
                          <td className="px-4 py-2.5 text-right text-blue-700 font-medium">
                            {s.consumption_kwh > 0 ? formatKwh(s.consumption_kwh) : "—"}
                          </td>
                          <td className={`px-4 py-2.5 text-right font-medium ${s.total_amount >= 0 ? "text-slate-700" : "text-emerald-700"}`}>
                            {formatCurrency(s.total_amount)}
                          </td>
                        </>
                      )}
                    </tr>
                  ))}
                </tbody>
                <tfoot className="sticky bottom-0 z-10">
                  <tr className="bg-slate-50 border-t border-slate-200">
                    <td colSpan={2} className="px-4 py-2.5 text-xs font-medium text-slate-500">
                      {displayedMemberStats.length} Mitglied{displayedMemberStats.length !== 1 ? "er" : ""}
                    </td>
                    {dataSource === "raw" ? (
                      <>
                        <td className="px-3 py-2.5 text-right text-xs font-bold text-emerald-700">
                          {formatKwh(displayedMemberStats.reduce((s, r) => s + r.consumption_kwh, 0))}
                        </td>
                        <td className="px-3 py-2.5 text-right text-xs font-bold text-amber-600">
                          {formatKwh(displayedMemberStats.reduce((s, r) => s + r.consumption_total_kwh - r.consumption_kwh, 0))}
                        </td>
                        <td className="px-3 py-2.5 text-right text-xs font-bold text-indigo-700">
                          {formatKwh(displayedMemberStats.reduce((s, r) => s + r.generation_kwh, 0))}
                        </td>
                        <td className="px-3 py-2.5 text-right text-xs font-bold text-violet-500">
                          {formatKwh(displayedMemberStats.reduce((s, r) => s + r.generation_total_kwh - r.generation_kwh, 0))}
                        </td>
                      </>
                    ) : (
                      <>
                        <td className="px-4 py-2.5 text-right text-xs font-bold text-emerald-700">
                          {formatKwh(displayedMemberStats.reduce((s, r) => s + r.generation_kwh, 0))}
                        </td>
                        <td className="px-4 py-2.5 text-right text-xs font-bold text-blue-700">
                          {formatKwh(displayedMemberStats.reduce((s, r) => s + r.consumption_kwh, 0))}
                        </td>
                        <td className="px-4 py-2.5 text-right text-xs font-bold text-slate-700">
                          {formatCurrency(displayedMemberStats.reduce((s, r) => s + r.total_amount, 0))}
                        </td>
                      </>
                    )}
                  </tr>
                </tfoot>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
