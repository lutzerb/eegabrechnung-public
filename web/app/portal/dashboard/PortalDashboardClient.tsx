"use client";

import { useState, useEffect } from "react";
import dynamic from "next/dynamic";

const PortalEnergyChart = dynamic(() => import("./PortalEnergyChart"), { ssr: false });

const MONTHS = ["Jan","Feb","Mär","Apr","Mai","Jun","Jul","Aug","Sep","Okt","Nov","Dez"];

function fmtKwh(v: number) {
  if (v >= 100000) return new Intl.NumberFormat("de-AT", { maximumFractionDigits: 1 }).format(v / 1000) + " MWh";
  return new Intl.NumberFormat("de-AT", { maximumFractionDigits: 1 }).format(v) + " kWh";
}

function fmtEur(v: number) {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(v);
}

function fmtDate(s: string) {
  return new Date(s).toLocaleDateString("de-AT");
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function pad(n: number) { return String(n).padStart(2, "0"); }

type PeriodMode = "year" | "month" | "day" | "15min";

const MODE_LABELS: Record<PeriodMode, string> = {
  year: "Jahr", month: "Monat", day: "Tag", "15min": "15 min",
};

function chartLabel(period: string, mode: PeriodMode): string {
  if (mode === "year") return period;
  if (mode === "month") {
    const [y, m] = period.split("-");
    return `${MONTHS[parseInt(m) - 1]} ${y.slice(2)}`;
  }
  if (mode === "day") {
    const [, m, d] = period.split("-");
    return `${d}.${m}.`;
  }
  // 15min: "2025-01-15 08:15" → "08:15"
  return period.split(" ")[1] || period;
}

function formatPeriodLabel(period: string, mode: PeriodMode): string {
  if (mode === "year") return period;
  if (mode === "month") {
    const [y, m] = period.split("-");
    return `${MONTHS[parseInt(m) - 1]} ${y}`;
  }
  if (mode === "day") {
    const [y, m, d] = period.split("-");
    return `${d}.${m}.${y}`;
  }
  // 15min: "2025-01-15 08:15" → "08:15"
  return period.split(" ")[1] || period;
}

interface PortalEnergyRow {
  period: string;
  wh_total_consumption: number;
  wh_community: number;
  wh_total_generation: number;
  wh_community_gen: number;
}

interface Invoice {
  id: string;
  period_start: string;
  period_end: string;
  total_amount: number;
  total_kwh: number;
  consumption_kwh: number;
  generation_kwh: number;
  document_type: string;
  sent_at: string | null;
}

interface PortalDocument {
  id: string;
  title: string;
  description: string;
  filename: string;
  mime_type: string;
  file_size_bytes: number;
  created_at: string;
}

interface MeterPoint {
  id: string;
  zaehlpunkt: string;
  direction: string;
  status: string;
  participation_factor: number;
}

interface Props {
  member: { id: string; name1: string; name2: string; email: string; mitglieds_nr: string; status: string };
  eeg: { id: string; name: string };
  invoices: Invoice[];
  documents: PortalDocument[];
  meterPoints: MeterPoint[];
  showFullEnergy: boolean;
}

export default function PortalDashboardClient({ member, eeg, invoices, documents, meterPoints, showFullEnergy }: Props) {
  const [activeTab, setActiveTab] = useState<"energy" | "invoices" | "downloads" | "zaehlpunkte">("energy");
  const [changingFactor, setChangingFactor] = useState<string | null>(null);
  const [newFactor, setNewFactor] = useState<string>("");
  const [factorError, setFactorError] = useState<string | null>(null);
  const [factorSuccess, setFactorSuccess] = useState<string | null>(null);
  const [factorSubmitting, setFactorSubmitting] = useState(false);

  // Energy period state
  const [periodMode, setPeriodMode] = useState<PeriodMode>("month");
  const [navYear, setNavYear] = useState<number>(() => new Date().getFullYear());
  const [navMonth, setNavMonth] = useState<number>(() => new Date().getMonth() + 1);
  const [navDay, setNavDay] = useState<number>(() => new Date().getDate());
  const [energyRows, setEnergyRows] = useState<PortalEnergyRow[]>([]);
  const [energyLoading, setEnergyLoading] = useState(true);
  const [energyError, setEnergyError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setEnergyLoading(true);
    setEnergyError(null);

    let from: string, to: string, granularity: string;

    if (periodMode === "year") {
      from = "2000-01-01";
      to = "2100-01-01";
      granularity = "year";
    } else if (periodMode === "month") {
      from = `${navYear}-01-01`;
      to = `${navYear + 1}-01-01`;
      granularity = "month";
    } else if (periodMode === "day") {
      const nextMonth = navMonth === 12 ? 1 : navMonth + 1;
      const nextYear = navMonth === 12 ? navYear + 1 : navYear;
      from = `${navYear}-${pad(navMonth)}-01`;
      to = `${nextYear}-${pad(nextMonth)}-01`;
      granularity = "day";
    } else {
      // 15min
      const nextDate = new Date(navYear, navMonth - 1, navDay + 1);
      from = `${navYear}-${pad(navMonth)}-${pad(navDay)}`;
      to = `${nextDate.getFullYear()}-${pad(nextDate.getMonth() + 1)}-${pad(nextDate.getDate())}`;
      granularity = "15min";
    }

    fetch(`/api/portal/energy?from=${from}&to=${to}&granularity=${granularity}`)
      .then(r => r.json())
      .then(data => {
        if (!cancelled) {
          setEnergyRows(Array.isArray(data) ? data : []);
          setEnergyLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setEnergyError("Fehler beim Laden der Energiedaten.");
          setEnergyLoading(false);
        }
      });

    return () => { cancelled = true; };
  }, [periodMode, navYear, navMonth, navDay]);

  function navPrev() {
    if (periodMode === "month") {
      setNavYear(y => y - 1);
    } else if (periodMode === "day") {
      if (navMonth === 1) { setNavYear(y => y - 1); setNavMonth(12); }
      else setNavMonth(m => m - 1);
    } else if (periodMode === "15min") {
      const d = new Date(navYear, navMonth - 1, navDay - 1);
      setNavYear(d.getFullYear()); setNavMonth(d.getMonth() + 1); setNavDay(d.getDate());
    }
  }

  function navNext() {
    if (periodMode === "month") {
      setNavYear(y => y + 1);
    } else if (periodMode === "day") {
      if (navMonth === 12) { setNavYear(y => y + 1); setNavMonth(1); }
      else setNavMonth(m => m + 1);
    } else if (periodMode === "15min") {
      const d = new Date(navYear, navMonth - 1, navDay + 1);
      setNavYear(d.getFullYear()); setNavMonth(d.getMonth() + 1); setNavDay(d.getDate());
    }
  }

  function navLabel(): string {
    if (periodMode === "year") return "Alle Jahre";
    if (periodMode === "month") return String(navYear);
    if (periodMode === "day") return `${MONTHS[navMonth - 1]} ${navYear}`;
    return `${pad(navDay)}.${pad(navMonth)}.${navYear}`;
  }

  const memberName = [member.name1, member.name2].filter(Boolean).join(" ");
  const hasGeneration = energyRows.some(r => r.wh_total_generation > 0);

  const chartData = energyRows.map(row => ({
    label: chartLabel(row.period, periodMode),
    "Bezug EEG":       Math.round(row.wh_community),
    ...(showFullEnergy ? { "Restbezug": Math.round(Math.max(0, row.wh_total_consumption - row.wh_community)) } : {}),
    "Einspeisung EEG": Math.round(row.wh_community_gen),
    ...(showFullEnergy ? { "Resteinspeisung": Math.round(Math.max(0, row.wh_total_generation - row.wh_community_gen)) } : {}),
  }));

  async function downloadPDF(invoiceId: string) {
    const res = await fetch(`/api/portal/invoices/${invoiceId}/pdf`);
    if (!res.ok) return;
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    window.open(url, "_blank");
    setTimeout(() => URL.revokeObjectURL(url), 10000);
  }

  async function downloadDocument(docId: string, filename: string) {
    const res = await fetch(`/api/portal/documents/${docId}`);
    if (!res.ok) return;
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }

  async function handleLogout() {
    await fetch("/api/portal/logout", { method: "POST" });
    window.location.href = "/portal";
  }

  async function handleChangeFactor(zaehlpunkt: string) {
    const factor = parseFloat(newFactor);
    if (isNaN(factor) || factor <= 0 || factor > 100) {
      setFactorError("Bitte einen Wert zwischen 0,01 und 100 eingeben.");
      return;
    }
    setFactorSubmitting(true);
    setFactorError(null);
    setFactorSuccess(null);
    try {
      const res = await fetch("/api/portal/change-factor", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ zaehlpunkt, participation_factor: factor }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setFactorError((data as { error?: string }).error || "Fehler beim Senden der Anfrage.");
      } else {
        setFactorSuccess(`Änderungsantrag eingereicht. Der neue Teilnahmefaktor von ${factor}% wird ab morgen aktiv, sobald der Netzbetreiber bestätigt.`);
        setChangingFactor(null);
        setNewFactor("");
      }
    } catch {
      setFactorError("Netzwerkfehler. Bitte versuchen Sie es erneut.");
    } finally {
      setFactorSubmitting(false);
    }
  }

  return (
    <div className="min-h-screen bg-slate-50">
      {/* Header */}
      <header className="bg-white border-b border-slate-200 px-4 sm:px-8 py-4">
        <div className="max-w-4xl mx-auto flex items-center justify-between">
          <div>
            <p className="text-xs text-slate-500">{eeg.name}</p>
            <h1 className="text-lg font-bold text-slate-900">{memberName}</h1>
            {member.mitglieds_nr && (
              <p className="text-xs text-slate-400 font-mono">Mitgl.-Nr. {member.mitglieds_nr}</p>
            )}
          </div>
          <button
            onClick={handleLogout}
            className="text-sm text-slate-500 hover:text-slate-700 transition-colors"
          >
            Abmelden
          </button>
        </div>
      </header>

      <div className="max-w-4xl mx-auto px-4 sm:px-8 py-6">
        {/* Tabs */}
        <div className="flex gap-1 mb-6 bg-slate-100 rounded-lg p-1 w-fit">
          <button
            onClick={() => setActiveTab("energy")}
            className={`px-4 py-1.5 text-sm font-medium rounded-md transition-colors ${
              activeTab === "energy" ? "bg-white text-slate-900 shadow-sm" : "text-slate-600 hover:text-slate-900"
            }`}
          >
            Energiedaten
          </button>
          <button
            onClick={() => setActiveTab("invoices")}
            className={`px-4 py-1.5 text-sm font-medium rounded-md transition-colors ${
              activeTab === "invoices" ? "bg-white text-slate-900 shadow-sm" : "text-slate-600 hover:text-slate-900"
            }`}
          >
            Rechnungen {invoices.length > 0 && `(${invoices.length})`}
          </button>
          <button
            onClick={() => setActiveTab("downloads")}
            className={`px-4 py-1.5 text-sm font-medium rounded-md transition-colors ${
              activeTab === "downloads" ? "bg-white text-slate-900 shadow-sm" : "text-slate-600 hover:text-slate-900"
            }`}
          >
            Downloads {documents.length > 0 && `(${documents.length})`}
          </button>
          <button
            onClick={() => setActiveTab("zaehlpunkte")}
            className={`px-4 py-1.5 text-sm font-medium rounded-md transition-colors ${
              activeTab === "zaehlpunkte" ? "bg-white text-slate-900 shadow-sm" : "text-slate-600 hover:text-slate-900"
            }`}
          >
            Zählpunkte
          </button>
        </div>

        {/* Energy Tab */}
        {activeTab === "energy" && (
          <div className="space-y-4">
            {/* Period controls */}
            <div className="flex flex-wrap items-center justify-between gap-3">
              {/* Mode tabs */}
              <div className="flex gap-1 bg-slate-100 rounded-lg p-1">
                {(["year", "month", "day", "15min"] as const).map(mode => (
                  <button
                    key={mode}
                    onClick={() => setPeriodMode(mode)}
                    className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                      periodMode === mode
                        ? "bg-white text-slate-900 shadow-sm"
                        : "text-slate-600 hover:text-slate-900"
                    }`}
                  >
                    {MODE_LABELS[mode]}
                  </button>
                ))}
              </div>

              {/* Navigator (hidden for year mode) */}
              {periodMode !== "year" && (
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
                  <span className="text-sm font-medium text-slate-700 min-w-[110px] text-center">
                    {navLabel()}
                  </span>
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
              )}
            </div>

            {energyLoading ? (
              <div className="bg-white rounded-xl border border-slate-200 px-6 py-16 text-center">
                <p className="text-slate-400 text-sm">Wird geladen…</p>
              </div>
            ) : energyError ? (
              <div className="bg-white rounded-xl border border-slate-200 px-6 py-8 text-center">
                <p className="text-red-500 text-sm">{energyError}</p>
              </div>
            ) : energyRows.length === 0 ? (
              <div className="bg-white rounded-xl border border-slate-200 px-6 py-16 text-center">
                <p className="text-slate-500">Keine Energiedaten für diesen Zeitraum.</p>
              </div>
            ) : (
              <>
                {/* Chart */}
                <div className="bg-white rounded-xl border border-slate-200 p-6">
                  <PortalEnergyChart data={chartData} hasGeneration={hasGeneration} showFullEnergy={showFullEnergy} />
                </div>

                {/* Table */}
                <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
                  <div
                    className="overflow-x-auto"
                    style={energyRows.length > 20 ? { maxHeight: "400px", overflowY: "auto" } : undefined}
                  >
                    <table className="w-full text-sm min-w-[500px]">
                      <thead className={energyRows.length > 20 ? "sticky top-0 z-10 bg-slate-50" : ""}>
                        <tr className="bg-slate-50 border-b border-slate-200">
                          <th className="px-4 py-3 text-left font-medium text-slate-600">Zeitraum</th>
                          {showFullEnergy && <th className="px-3 py-3 text-right font-medium text-slate-600">Bezug gesamt</th>}
                          <th className="px-3 py-3 text-right font-medium text-blue-600">Bezug EEG</th>
                          {showFullEnergy && <th className="px-3 py-3 text-right font-medium text-slate-500">Restbezug</th>}
                          {hasGeneration && showFullEnergy && <th className="px-3 py-3 text-right font-medium text-emerald-600">Einsp. gesamt</th>}
                          {hasGeneration && <th className="px-3 py-3 text-right font-medium text-emerald-700">Einsp. EEG</th>}
                          {hasGeneration && showFullEnergy && <th className="px-3 py-3 text-right font-medium text-slate-400">Resteinsp.</th>}
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-100">
                        {energyRows.map(row => (
                          <tr key={row.period} className="hover:bg-slate-50">
                            <td className="px-4 py-3 font-medium text-slate-900">{formatPeriodLabel(row.period, periodMode)}</td>
                            {showFullEnergy && <td className="px-3 py-3 text-right text-slate-600 font-mono text-xs">{fmtKwh(row.wh_total_consumption)}</td>}
                            <td className="px-3 py-3 text-right text-blue-600 font-mono text-xs">{fmtKwh(row.wh_community)}</td>
                            {showFullEnergy && <td className="px-3 py-3 text-right text-slate-400 font-mono text-xs">{fmtKwh(Math.max(0, row.wh_total_consumption - row.wh_community))}</td>}
                            {hasGeneration && showFullEnergy && (
                              <td className="px-3 py-3 text-right text-emerald-500 font-mono text-xs">{fmtKwh(row.wh_total_generation)}</td>
                            )}
                            {hasGeneration && (
                              <td className="px-3 py-3 text-right text-emerald-700 font-mono text-xs">{fmtKwh(row.wh_community_gen)}</td>
                            )}
                            {hasGeneration && showFullEnergy && (
                              <td className="px-3 py-3 text-right text-slate-400 font-mono text-xs">{fmtKwh(Math.max(0, row.wh_total_generation - row.wh_community_gen))}</td>
                            )}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              </>
            )}
          </div>
        )}

        {/* Invoices Tab */}
        {activeTab === "invoices" && (
          <div className="space-y-3">
            {invoices.length === 0 ? (
              <div className="bg-white rounded-xl border border-slate-200 px-6 py-16 text-center">
                <p className="text-slate-500">Noch keine Rechnungen vorhanden.</p>
              </div>
            ) : (
              invoices.map(inv => {
                const label = `${fmtDate(inv.period_start)}\u2013${fmtDate(inv.period_end)}`;
                const isCredit = inv.document_type === "credit_note" || inv.total_amount < 0;
                return (
                  <div key={inv.id} className="bg-white rounded-xl border border-slate-200 px-5 py-4 flex items-center justify-between gap-4">
                    <div>
                      <p className="font-medium text-slate-900 text-sm">{label}</p>
                      <p className="text-xs text-slate-400 mt-0.5">
                        {isCredit ? "Gutschrift" : "Rechnung"}{inv.consumption_kwh > 0 ? ` \u00b7 ${inv.consumption_kwh.toLocaleString("de-AT")} kWh` : ""}
                      </p>
                    </div>
                    <div className="flex items-center gap-4">
                      <span className={`text-sm font-semibold ${isCredit ? "text-green-600" : "text-slate-900"}`}>
                        {fmtEur(inv.total_amount)}
                      </span>
                      <button
                        onClick={() => downloadPDF(inv.id)}
                        className="px-3 py-1.5 text-xs font-medium bg-slate-100 text-slate-700 rounded-md hover:bg-slate-200 transition-colors flex items-center gap-1"
                      >
                        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                        </svg>
                        PDF
                      </button>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        )}

        {/* Downloads Tab */}
        {activeTab === "downloads" && (
          <div className="space-y-3">
            {documents.length === 0 ? (
              <div className="bg-white rounded-xl border border-slate-200 px-6 py-16 text-center">
                <p className="text-slate-500">Keine Dokumente verfügbar.</p>
              </div>
            ) : (
              documents.map(doc => (
                <div
                  key={doc.id}
                  className="bg-white rounded-xl border border-slate-200 px-5 py-4 flex items-center justify-between gap-4"
                >
                  <div className="min-w-0">
                    <p className="font-medium text-slate-900 text-sm truncate">{doc.title}</p>
                    {doc.description && (
                      <p className="text-xs text-slate-500 mt-0.5 truncate">{doc.description}</p>
                    )}
                    <p className="text-xs text-slate-400 mt-0.5">
                      {formatFileSize(doc.file_size_bytes)} &middot; {fmtDate(doc.created_at)}
                    </p>
                  </div>
                  <button
                    onClick={() => downloadDocument(doc.id, doc.filename)}
                    className="flex-shrink-0 px-3 py-1.5 text-xs font-medium bg-slate-100 text-slate-700 rounded-md hover:bg-slate-200 transition-colors flex items-center gap-1"
                  >
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                    </svg>
                    Herunterladen
                  </button>
                </div>
              ))
            )}
          </div>
        )}

        {/* Zählpunkte Tab */}
        {activeTab === "zaehlpunkte" && (
          <div className="space-y-4">
            <div className="bg-blue-50 border border-blue-200 rounded-xl p-4 text-sm text-blue-800">
              <p className="font-medium mb-1">Was ist der Teilnahmefaktor?</p>
              <p>Der Teilnahmefaktor gibt an, welcher Anteil Ihrer Energie (in %) an der Energiegemeinschaft teilnimmt. Bei <strong>100%</strong> wird Ihr gesamter Verbrauch bzw. Ihre gesamte Einspeisung in der Gemeinschaft verrechnet – das ist der Standardfall. Ein niedrigerer Wert macht Sinn, wenn Sie gleichzeitig in mehreren Energiegemeinschaften Mitglied sind.</p>
            </div>

            {factorSuccess && (
              <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-800">
                {factorSuccess}
              </div>
            )}

            {meterPoints.length === 0 ? (
              <div className="bg-white rounded-xl border border-slate-200 px-6 py-16 text-center">
                <p className="text-slate-500">Keine Zählpunkte vorhanden.</p>
              </div>
            ) : (
              meterPoints.map(mp => (
                <div key={mp.id} className="bg-white rounded-xl border border-slate-200 p-5">
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <p className="font-mono text-sm font-medium text-slate-900">{mp.zaehlpunkt}</p>
                      <p className="text-xs text-slate-500 mt-0.5">
                        {mp.direction === "GENERATION" ? "Einspeisung" : "Verbrauch"} &middot; Status: {mp.status}
                      </p>
                      <p className="text-sm text-slate-700 mt-2">
                        Aktueller Teilnahmefaktor: <strong>{mp.participation_factor}%</strong>
                      </p>
                    </div>
                    {changingFactor !== mp.zaehlpunkt && (
                      <button
                        onClick={() => { setChangingFactor(mp.zaehlpunkt); setNewFactor(String(mp.participation_factor)); setFactorError(null); setFactorSuccess(null); }}
                        className="flex-shrink-0 px-3 py-1.5 text-xs font-medium bg-slate-100 text-slate-700 rounded-md hover:bg-slate-200 transition-colors"
                      >
                        Ändern
                      </button>
                    )}
                  </div>

                  {changingFactor === mp.zaehlpunkt && (
                    <div className="mt-4 pt-4 border-t border-slate-100">
                      <p className="text-xs text-slate-500 mb-2">Neuer Teilnahmefaktor (0,01–100%):</p>
                      <div className="flex items-center gap-2">
                        <input
                          type="number"
                          min="0.01"
                          max="100"
                          step="0.01"
                          value={newFactor}
                          onChange={e => setNewFactor(e.target.value)}
                          className="w-28 px-3 py-1.5 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                        <span className="text-sm text-slate-500">%</span>
                        <button
                          onClick={() => handleChangeFactor(mp.zaehlpunkt)}
                          disabled={factorSubmitting}
                          className="px-3 py-1.5 bg-blue-600 text-white text-xs font-medium rounded-md hover:bg-blue-700 disabled:opacity-50 transition-colors"
                        >
                          {factorSubmitting ? "Wird gesendet…" : "EDA-Anfrage senden"}
                        </button>
                        <button
                          onClick={() => { setChangingFactor(null); setFactorError(null); }}
                          className="px-3 py-1.5 text-xs text-slate-500 hover:text-slate-700"
                        >
                          Abbrechen
                        </button>
                      </div>
                      {factorError && (
                        <p className="mt-2 text-xs text-red-600">{factorError}</p>
                      )}
                      <p className="mt-2 text-xs text-slate-400">
                        Die Änderung wird als EDA-Prozess an den Netzbetreiber gesendet und tritt ab morgen in Kraft, sobald dieser bestätigt.
                      </p>
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        )}
      </div>
    </div>
  );
}
