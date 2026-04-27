"use client";

import { useState, useEffect, useCallback } from "react";
import { useSession } from "next-auth/react";

interface OemagMonthPrice { month: number; pv_price: number; wind_price: number; }
interface OemagYearPrices { year: number; prices: OemagMonthPrice[]; static: boolean; }
interface OemagData { years: OemagYearPrices[]; scraped_at: string; }

interface TariffEntry {
  id?: string;
  valid_from: string;
  valid_until: string;
  energy_price: number;
  producer_price: number;
}

interface TariffSchedule {
  id: string;
  name: string;
  granularity: "annual" | "monthly" | "daily" | "quarter_hour";
  is_active: boolean;
  entry_count: number;
  entries?: TariffEntry[];
}

const MONTH_NAMES = ["Jänner","Februar","März","April","Mai","Juni","Juli","August","September","Oktober","November","Dezember"];
const GRANULARITY_LABELS: Record<string, string> = {
  annual: "Jährlich", monthly: "Monatlich", daily: "Täglich", quarter_hour: "15-Minuten",
};

function pad2(n: number) { return String(n).padStart(2, "0"); }

// Build the 12 monthly entries for a given year
function buildMonthlyEntries(year: number, prices: Array<{ energy_price: number; producer_price: number }>): TariffEntry[] {
  return Array.from({ length: 12 }, (_, m) => ({
    valid_from: `${year}-${pad2(m + 1)}-01T00:00:00Z`,
    valid_until: m === 11
      ? `${year + 1}-01-01T00:00:00Z`
      : `${year}-${pad2(m + 2)}-01T00:00:00Z`,
    energy_price: prices[m]?.energy_price ?? 0,
    producer_price: prices[m]?.producer_price ?? 0,
  }));
}

// Build annual entries for a range of years
function buildAnnualEntries(startYear: number, years: number, prices: Array<{ energy_price: number; producer_price: number }>): TariffEntry[] {
  return Array.from({ length: years }, (_, i) => {
    const y = startYear + i;
    return {
      valid_from: `${y}-01-01T00:00:00Z`,
      valid_until: `${y + 1}-01-01T00:00:00Z`,
      energy_price: prices[i]?.energy_price ?? 0,
      producer_price: prices[i]?.producer_price ?? 0,
    };
  });
}

interface EntryEditorProps {
  schedule: TariffSchedule;
  eegEnergyPrice: number;
  eegProducerPrice: number;
  onSave: (entries: TariffEntry[]) => Promise<void>;
  saving: boolean;
  oemagData: OemagData | null;
}

function MonthlyEditor({ schedule, eegEnergyPrice, eegProducerPrice, onSave, saving, oemagData }: EntryEditorProps) {
  const currentYear = new Date().getFullYear();
  const [year, setYear] = useState(() => {
    const entries = schedule.entries ?? [];
    if (entries.length > 0) return parseInt(entries[0].valid_from.substring(0, 4));
    return currentYear;
  });
  // prices[month] = {energy_price, producer_price}
  const [prices, setPrices] = useState<Array<{ energy_price: number; producer_price: number }>>(
    () => Array.from({ length: 12 }, () => ({ energy_price: eegEnergyPrice, producer_price: eegProducerPrice }))
  );

  // Load existing entries for selected year when schedule or year changes
  useEffect(() => {
    const entries = schedule.entries ?? [];
    const newPrices = Array.from({ length: 12 }, (_, m) => {
      const fromStr = `${year}-${pad2(m + 1)}-01T00:00:00Z`;
      const e = entries.find((e) => e.valid_from === fromStr);
      return e ? { energy_price: e.energy_price, producer_price: e.producer_price }
               : { energy_price: eegEnergyPrice, producer_price: eegProducerPrice };
    });
    setPrices(newPrices);
  }, [schedule.id, year, eegEnergyPrice, eegProducerPrice]); // eslint-disable-line react-hooks/exhaustive-deps

  const [diff, setDiff] = useState<string>("");

  const setPrice = (month: number, field: "energy_price" | "producer_price", val: string) => {
    setPrices((prev) => {
      const next = [...prev];
      next[month] = { ...next[month], [field]: parseFloat(val) || 0 };
      return next;
    });
  };

  // Fill all months with value from month 0
  const fillDown = (field: "energy_price" | "producer_price") => {
    const val = prices[0][field];
    setPrices((prev) => prev.map((p) => ({ ...p, [field]: val })));
  };

  const applyDiff = () => {
    const d = parseFloat(diff);
    if (isNaN(d)) return;
    setPrices((prev) => prev.map((p) => ({ ...p, energy_price: parseFloat((p.producer_price + d).toFixed(3)) })));
  };

  const fillFromOemag = (priceType: "pv" | "wind") => {
    const yearData = oemagData?.years.find((y) => y.year === year);
    if (!yearData) return;
    const d = parseFloat(diff);
    const hasDiff = !isNaN(d);
    setPrices((prev) => prev.map((p, m) => {
      const mp = yearData.prices.find((x) => x.month === m + 1);
      if (!mp) return p;
      const producer = priceType === "pv" ? mp.pv_price : mp.wind_price;
      return {
        ...p,
        producer_price: producer,
        energy_price: hasDiff ? parseFloat((producer + d).toFixed(3)) : p.energy_price,
      };
    }));
  };

  const handleSave = () => {
    const entries = buildMonthlyEntries(year, prices);
    onSave(entries);
  };

  const yearOptions = Array.from({ length: 8 }, (_, i) => currentYear - 2 + i);

  return (
    <div>
      <div className="flex items-center gap-3 mb-4">
        <label className="text-sm font-medium text-slate-700">Jahr:</label>
        <div className="flex items-center gap-1">
          <button onClick={() => setYear((y) => y - 1)} className="p-1 text-slate-400 hover:text-slate-700">&#8249;</button>
          <select value={year} onChange={(e) => setYear(Number(e.target.value))}
            className="px-2 py-1 border border-slate-300 rounded text-sm bg-white">
            {yearOptions.map((y) => <option key={y} value={y}>{y}</option>)}
          </select>
          <button onClick={() => setYear((y) => y + 1)} className="p-1 text-slate-400 hover:text-slate-700">&#8250;</button>
        </div>
      </div>

      <div className="flex items-center gap-2 mb-3 p-3 bg-slate-50 rounded-lg border border-slate-200">
        <span className="text-xs font-medium text-slate-600 whitespace-nowrap">Bezug = Einspeisung +</span>
        <input
          type="number" step="0.01" min="0" value={diff}
          onChange={(e) => setDiff(e.target.value)}
          placeholder="z.B. 3.00"
          className="w-24 px-2 py-1 border border-slate-300 rounded text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 text-right"
        />
        <span className="text-xs text-slate-500">ct/kWh</span>
        <button onClick={applyDiff} disabled={diff === "" || isNaN(parseFloat(diff))}
          className="px-3 py-1 text-xs font-medium text-white bg-blue-700 rounded hover:bg-blue-800 disabled:opacity-40 transition-colors">
          Alle Monate setzen
        </button>
      </div>

      {oemagData?.years.find((y) => y.year === year) && (
        <div className="flex items-center gap-2 mb-3 p-3 bg-amber-50 rounded-lg border border-amber-200 flex-wrap">
          <span className="text-xs font-medium text-amber-800 whitespace-nowrap">OeMAG {year} → Einspeisung:</span>
          <button onClick={() => fillFromOemag("pv")}
            className="px-3 py-1 text-xs font-medium text-green-800 bg-green-50 border border-green-200 rounded hover:bg-green-100 transition-colors">
            PV
          </button>
          <button onClick={() => fillFromOemag("wind")}
            className="px-3 py-1 text-xs font-medium text-green-800 bg-green-50 border border-green-200 rounded hover:bg-green-100 transition-colors">
            Wind
          </button>
          {!isNaN(parseFloat(diff)) && (
            <span className="text-xs text-amber-700">+ Bezug wird auto-berechnet</span>
          )}
        </div>
      )}

      <table className="w-full text-sm border border-slate-200 rounded-lg overflow-hidden">
        <thead>
          <tr className="bg-slate-50">
            <th className="text-left px-3 py-2 font-medium text-slate-600 w-32">Monat</th>
            <th className="px-3 py-2 font-medium text-slate-600">
              <div className="flex items-center justify-between">
                <span>Bezug (ct/kWh)</span>
                <button onClick={() => fillDown("energy_price")} title="Alle auf ersten Wert setzen"
                  className="text-xs text-blue-600 hover:text-blue-800 font-normal">&#8595; alle</button>
              </div>
            </th>
            <th className="px-3 py-2 font-medium text-slate-600">
              <div className="flex items-center justify-between">
                <span>Einspeisung (ct/kWh)</span>
                <button onClick={() => fillDown("producer_price")} title="Alle auf ersten Wert setzen"
                  className="text-xs text-blue-600 hover:text-blue-800 font-normal">&#8595; alle</button>
              </div>
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {MONTH_NAMES.map((name, m) => (
            <tr key={m} className="hover:bg-slate-50">
              <td className="px-3 py-2 text-slate-700 font-medium">{name} {year}</td>
              <td className="px-3 py-1.5">
                <input type="number" step="0.01" min="0" value={prices[m].energy_price}
                  onChange={(e) => setPrice(m, "energy_price", e.target.value)}
                  className="w-full px-2 py-1 border border-slate-300 rounded text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 text-right" />
              </td>
              <td className="px-3 py-1.5">
                <input type="number" step="0.01" min="0" value={prices[m].producer_price}
                  onChange={(e) => setPrice(m, "producer_price", e.target.value)}
                  className="w-full px-2 py-1 border border-slate-300 rounded text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 text-right" />
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <div className="mt-4 flex items-center justify-between">
        <p className="text-xs text-slate-400">Einträge werden für {year} gespeichert. Andere Jahre bleiben unverändert.</p>
        <button onClick={handleSave} disabled={saving}
          className="px-5 py-2 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800 disabled:opacity-50 transition-colors">
          {saving ? "Speichern..." : `${year} speichern`}
        </button>
      </div>
    </div>
  );
}

function AnnualEditor({ schedule, eegEnergyPrice, eegProducerPrice, onSave, saving, oemagData }: EntryEditorProps) {
  const currentYear = new Date().getFullYear();
  const START_YEAR = currentYear - 3;
  const NUM_YEARS = 8;

  const [prices, setPrices] = useState<Array<{ energy_price: number; producer_price: number }>>(
    () => Array.from({ length: NUM_YEARS }, () => ({ energy_price: eegEnergyPrice, producer_price: eegProducerPrice }))
  );

  useEffect(() => {
    const entries = schedule.entries ?? [];
    const newPrices = Array.from({ length: NUM_YEARS }, (_, i) => {
      const y = START_YEAR + i;
      const e = entries.find((e) => e.valid_from === `${y}-01-01T00:00:00Z`);
      return e ? { energy_price: e.energy_price, producer_price: e.producer_price }
               : { energy_price: eegEnergyPrice, producer_price: eegProducerPrice };
    });
    setPrices(newPrices);
  }, [schedule.id, eegEnergyPrice, eegProducerPrice]); // eslint-disable-line react-hooks/exhaustive-deps

  const [diff, setDiff] = useState<string>("");

  const setPrice = (i: number, field: "energy_price" | "producer_price", val: string) => {
    setPrices((prev) => {
      const next = [...prev];
      next[i] = { ...next[i], [field]: parseFloat(val) || 0 };
      return next;
    });
  };

  const applyDiff = () => {
    const d = parseFloat(diff);
    if (isNaN(d)) return;
    setPrices((prev) => prev.map((p) => ({ ...p, energy_price: parseFloat((p.producer_price + d).toFixed(3)) })));
  };

  const fillFromOemag = (priceType: "pv" | "wind") => {
    const d = parseFloat(diff);
    const hasDiff = !isNaN(d);
    setPrices((prev) => prev.map((p, i) => {
      const y = START_YEAR + i;
      const yearData = oemagData?.years.find((yd) => yd.year === y);
      if (!yearData || yearData.prices.length === 0) return p;
      const vals = yearData.prices.map((mp) => priceType === "pv" ? mp.pv_price : mp.wind_price);
      const producer = parseFloat((vals.reduce((a, b) => a + b, 0) / vals.length).toFixed(3));
      return {
        ...p,
        producer_price: producer,
        energy_price: hasDiff ? parseFloat((producer + d).toFixed(3)) : p.energy_price,
      };
    }));
  };

  const handleSave = () => onSave(buildAnnualEntries(START_YEAR, NUM_YEARS, prices));

  return (
    <div>
      {oemagData && (
        <div className="flex items-center gap-2 mb-3 p-3 bg-amber-50 rounded-lg border border-amber-200 flex-wrap">
          <span className="text-xs font-medium text-amber-800 whitespace-nowrap">OeMAG Jahresdurchschnitt → Einspeisung:</span>
          <button onClick={() => fillFromOemag("pv")}
            className="px-3 py-1 text-xs font-medium text-green-800 bg-green-50 border border-green-200 rounded hover:bg-green-100 transition-colors">
            PV
          </button>
          <button onClick={() => fillFromOemag("wind")}
            className="px-3 py-1 text-xs font-medium text-green-800 bg-green-50 border border-green-200 rounded hover:bg-green-100 transition-colors">
            Wind
          </button>
          {!isNaN(parseFloat(diff)) && (
            <span className="text-xs text-amber-700">+ Bezug wird auto-berechnet</span>
          )}
        </div>
      )}
      <div className="flex items-center gap-2 mb-3 p-3 bg-slate-50 rounded-lg border border-slate-200">
        <span className="text-xs font-medium text-slate-600 whitespace-nowrap">Bezug = Einspeisung +</span>
        <input
          type="number" step="0.01" min="0" value={diff}
          onChange={(e) => setDiff(e.target.value)}
          placeholder="z.B. 3.00"
          className="w-24 px-2 py-1 border border-slate-300 rounded text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 text-right"
        />
        <span className="text-xs text-slate-500">ct/kWh</span>
        <button onClick={applyDiff} disabled={diff === "" || isNaN(parseFloat(diff))}
          className="px-3 py-1 text-xs font-medium text-white bg-blue-700 rounded hover:bg-blue-800 disabled:opacity-40 transition-colors">
          Alle Jahre setzen
        </button>
      </div>

      <table className="w-full text-sm border border-slate-200 rounded-lg overflow-hidden">
        <thead>
          <tr className="bg-slate-50">
            <th className="text-left px-3 py-2 font-medium text-slate-600 w-24">Jahr</th>
            <th className="px-3 py-2 font-medium text-slate-600">Bezug (ct/kWh)</th>
            <th className="px-3 py-2 font-medium text-slate-600">Einspeisung (ct/kWh)</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {Array.from({ length: NUM_YEARS }, (_, i) => {
            const y = START_YEAR + i;
            const isCurrent = y === currentYear;
            return (
              <tr key={y} className={isCurrent ? "bg-blue-50" : "hover:bg-slate-50"}>
                <td className={`px-3 py-2 font-medium ${isCurrent ? "text-blue-700" : "text-slate-700"}`}>
                  {y}{isCurrent && <span className="ml-1 text-xs font-normal text-blue-500">aktuell</span>}
                </td>
                <td className="px-3 py-1.5">
                  <input type="number" step="0.01" min="0" value={prices[i].energy_price}
                    onChange={(e) => setPrice(i, "energy_price", e.target.value)}
                    className="w-full px-2 py-1 border border-slate-300 rounded text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 text-right" />
                </td>
                <td className="px-3 py-1.5">
                  <input type="number" step="0.01" min="0" value={prices[i].producer_price}
                    onChange={(e) => setPrice(i, "producer_price", e.target.value)}
                    className="w-full px-2 py-1 border border-slate-300 rounded text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 text-right" />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
      <div className="mt-4 flex items-center justify-between">
        <p className="text-xs text-slate-400">Zeigt {START_YEAR}&#8211;{START_YEAR + NUM_YEARS - 1}. Alle Jahre werden zusammen gespeichert.</p>
        <button onClick={handleSave} disabled={saving}
          className="px-5 py-2 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800 disabled:opacity-50 transition-colors">
          {saving ? "Speichern..." : "Alle Jahre speichern"}
        </button>
      </div>
    </div>
  );
}

interface TariffManagerProps {
  eegId: string;
  eegEnergyPrice: number;
  eegProducerPrice: number;
}

export default function TariffManager({ eegId, eegEnergyPrice, eegProducerPrice }: TariffManagerProps) {
  const { data: session } = useSession();
  const [schedules, setSchedules] = useState<TariffSchedule[]>([]);
  const [selected, setSelected] = useState<TariffSchedule | null>(null);
  const [loadingList, setLoadingList] = useState(true);
  const [loadingEntries, setLoadingEntries] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newGranularity, setNewGranularity] = useState<string>("monthly");
  const [creating, setCreating] = useState(false);
  const [oemagData, setOemagData] = useState<OemagData | null>(null);
  const [oemagRefreshing, setOemagRefreshing] = useState(false);
  const [showOemag, setShowOemag] = useState(false);

  const authHeaders = { Authorization: `Bearer ${session?.accessToken ?? ""}`, "Content-Type": "application/json" };

  const fetchOemag = useCallback(async () => {
    if (!session?.accessToken) return;
    try {
      const res = await fetch("/api/oemag/marktpreis", { headers: authHeaders });
      if (res.ok) setOemagData(await res.json());
    } catch { /* ignore */ }
  }, [session?.accessToken]); // eslint-disable-line react-hooks/exhaustive-deps

  const refreshOemag = async () => {
    setOemagRefreshing(true);
    try {
      await fetch("/api/oemag/refresh", { method: "POST", headers: authHeaders });
      await fetchOemag();
    } catch { /* ignore */ }
    finally { setOemagRefreshing(false); }
  };

  useEffect(() => { fetchOemag(); }, [fetchOemag]);

  const fetchSchedules = useCallback(async () => {
    if (!session?.accessToken) return;
    setLoadingList(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/tariffs`, { headers: authHeaders });
      const data = await res.json();
      setSchedules(Array.isArray(data) ? data : []);
    } catch { setError("Fehler beim Laden der Tarifpläne."); }
    finally { setLoadingList(false); }
  }, [eegId, session?.accessToken]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => { fetchSchedules(); }, [fetchSchedules]);

  const selectSchedule = async (s: TariffSchedule) => {
    setLoadingEntries(true);
    setSelected(null);
    setSuccess(null);
    setError(null);
    try {
      const res = await fetch(`/api/eegs/${eegId}/tariffs/${s.id}`, { headers: authHeaders });
      const data = await res.json();
      setSelected(data);
    } catch { setError("Fehler beim Laden der Einträge."); }
    finally { setLoadingEntries(false); }
  };

  const handleCreate = async () => {
    if (!newName.trim()) return;
    setCreating(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/tariffs`, {
        method: "POST", headers: authHeaders,
        body: JSON.stringify({ name: newName.trim(), granularity: newGranularity }),
      });
      const data = await res.json();
      setSchedules((prev) => [data, ...prev]);
      setNewName(""); setShowCreate(false);
      await selectSchedule(data);
    } catch { setError("Fehler beim Erstellen."); }
    finally { setCreating(false); }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Tarifplan wirklich löschen?")) return;
    try {
      await fetch(`/api/eegs/${eegId}/tariffs/${id}`, { method: "DELETE", headers: authHeaders });
      setSchedules((prev) => prev.filter((s) => s.id !== id));
      if (selected?.id === id) setSelected(null);
    } catch { setError("Löschen fehlgeschlagen."); }
  };

  const handleActivate = async (s: TariffSchedule) => {
    try {
      if (s.is_active) {
        await fetch(`/api/eegs/${eegId}/tariffs/${s.id}/activate`, { method: "DELETE", headers: authHeaders });
        setSchedules((prev) => prev.map((x) => ({ ...x, is_active: x.id === s.id ? false : x.is_active })));
        if (selected?.id === s.id) setSelected((prev) => prev ? { ...prev, is_active: false } : null);
      } else {
        await fetch(`/api/eegs/${eegId}/tariffs/${s.id}/activate`, { method: "POST", headers: authHeaders });
        setSchedules((prev) => prev.map((x) => ({ ...x, is_active: x.id === s.id })));
        if (selected?.id === s.id) setSelected((prev) => prev ? { ...prev, is_active: true } : null);
      }
    } catch { setError("Fehler beim Aktivieren."); }
  };

  const handleSaveEntries = async (entries: TariffEntry[]) => {
    if (!selected) return;
    setSaving(true);
    setError(null);
    setSuccess(null);
    try {
      const res = await fetch(`/api/eegs/${eegId}/tariffs/${selected.id}/entries`, {
        method: "PUT", headers: authHeaders, body: JSON.stringify(entries),
      });
      if (!res.ok) throw new Error("Speichern fehlgeschlagen");
      const data = await res.json();
      setSuccess(`${data.saved} Einträge gespeichert.`);
      // Reload entries so editor stays in sync
      await selectSchedule(selected);
      setSchedules((prev) => prev.map((s) => s.id === selected.id ? { ...s, entry_count: data.saved } : s));
    } catch (e: unknown) { setError((e as Error).message); }
    finally { setSaving(false); }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
      {/* Left: schedule list */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Pläne</h2>
          <button onClick={() => setShowCreate(!showCreate)}
            className="px-3 py-1.5 text-xs font-medium text-white bg-blue-700 rounded-lg hover:bg-blue-800 transition-colors">
            + Neu
          </button>
        </div>

        {showCreate && (
          <div className="bg-white rounded-xl border border-slate-200 p-4 space-y-3">
            <p className="text-sm font-medium text-slate-900">Neuer Tarifplan</p>
            <input type="text" placeholder="Name (z.B. Jahrestarif 2026)" value={newName}
              onChange={(e) => setNewName(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
            <select value={newGranularity} onChange={(e) => setNewGranularity(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500">
              <option value="monthly">Monatlich</option>
              <option value="annual">Jährlich</option>
              <option value="daily">Täglich (Import)</option>
              <option value="quarter_hour">15-Minuten (Import)</option>
            </select>
            <div className="flex gap-2">
              <button onClick={handleCreate} disabled={!newName.trim() || creating}
                className="flex-1 px-3 py-1.5 text-sm font-medium text-white bg-blue-700 rounded-lg hover:bg-blue-800 disabled:opacity-50 transition-colors">
                {creating ? "Erstelle..." : "Erstellen"}
              </button>
              <button onClick={() => setShowCreate(false)}
                className="px-3 py-1.5 text-sm font-medium text-slate-600 border border-slate-300 rounded-lg hover:bg-slate-50">
                Abbrechen
              </button>
            </div>
          </div>
        )}

        {loadingList ? (
          <div className="text-sm text-slate-400 text-center py-8">Lade...</div>
        ) : schedules.length === 0 ? (
          <div className="bg-white rounded-xl border border-slate-200 px-4 py-8 text-center text-sm text-slate-400">
            Noch keine Tarifpläne. Erstellen Sie Ihren ersten Plan.
          </div>
        ) : (
          schedules.map((s) => (
            <div key={s.id}
              onClick={() => selectSchedule(s)}
              className={`bg-white rounded-xl border cursor-pointer p-4 transition-all ${
                selected?.id === s.id ? "border-blue-400 ring-2 ring-blue-100" : "border-slate-200 hover:border-slate-300"
              }`}>
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0 flex-1">
                  <p className="font-medium text-slate-900 text-sm truncate">{s.name}</p>
                  <div className="flex items-center gap-2 mt-1 flex-wrap">
                    <span className="text-xs px-1.5 py-0.5 bg-slate-100 text-slate-600 rounded">
                      {GRANULARITY_LABELS[s.granularity] ?? s.granularity}
                    </span>
                    <span className="text-xs text-slate-400">{s.entry_count ?? 0} Einträge</span>
                    {s.is_active && (
                      <span className="text-xs px-1.5 py-0.5 bg-emerald-50 text-emerald-700 border border-emerald-200 rounded font-medium">
                        Aktiv
                      </span>
                    )}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-1.5 mt-3" onClick={(e) => e.stopPropagation()}>
                <button onClick={() => handleActivate(s)}
                  className={`flex-1 px-2 py-1 text-xs font-medium rounded transition-colors ${
                    s.is_active
                      ? "bg-emerald-50 text-emerald-700 border border-emerald-200 hover:bg-emerald-100"
                      : "bg-slate-50 text-slate-600 border border-slate-200 hover:bg-slate-100"
                  }`}>
                  {s.is_active ? "Deaktivieren" : "Aktivieren"}
                </button>
                <button onClick={() => handleDelete(s.id)}
                  className="px-2 py-1 text-xs font-medium text-red-600 bg-red-50 border border-red-200 rounded hover:bg-red-100 transition-colors">
                  Löschen
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Right: entry editor */}
      <div className="lg:col-span-2">
        {/* OeMAG Panel */}
        {oemagData && (
          <div className="mb-4 bg-amber-50 border border-amber-200 rounded-xl overflow-hidden">
            <button
              onClick={() => setShowOemag((v) => !v)}
              className="w-full flex items-center justify-between px-4 py-3 text-sm font-medium text-amber-900 hover:bg-amber-100 transition-colors"
            >
              <span className="flex items-center gap-2">
                <span>OeMAG Marktpreise</span>
                <span className="text-xs font-normal text-amber-700">
                  Stand {new Date(oemagData.scraped_at).toLocaleDateString("de-AT")}
                </span>
              </span>
              <div className="flex items-center gap-2">
                <button
                  onClick={(e) => { e.stopPropagation(); refreshOemag(); }}
                  disabled={oemagRefreshing}
                  className="px-2.5 py-1 text-xs font-medium bg-white border border-amber-300 text-amber-800 rounded hover:bg-amber-50 disabled:opacity-50 transition-colors"
                >
                  {oemagRefreshing ? "..." : "↻ Aktualisieren"}
                </button>
                <span className="text-amber-500">{showOemag ? "▲" : "▼"}</span>
              </div>
            </button>
            {showOemag && (
              <div className="px-4 pb-4 overflow-x-auto">
                {[...oemagData.years].reverse().map((yp) => (
                  <div key={yp.year} className="mb-3">
                    <p className="text-xs font-semibold text-amber-800 mb-1">{yp.year}{yp.static && " (historisch)"}</p>
                    <table className="text-xs border-collapse">
                      <thead>
                        <tr className="bg-amber-100">
                          <th className="text-left px-2 py-1 font-medium text-amber-800 border border-amber-200">Monat</th>
                          <th className="text-right px-2 py-1 font-medium text-amber-800 border border-amber-200">PV & andere</th>
                          <th className="text-right px-2 py-1 font-medium text-amber-800 border border-amber-200">Wind</th>
                        </tr>
                      </thead>
                      <tbody>
                        {yp.prices.map((p) => (
                          <tr key={p.month}>
                            <td className="px-2 py-1 border border-amber-200 text-amber-900">
                              {["","Jän","Feb","Mär","Apr","Mai","Jun","Jul","Aug","Sep","Okt","Nov","Dez"][p.month]}
                            </td>
                            <td className="px-2 py-1 border border-amber-200 text-right tabular-nums text-amber-900">
                              {p.pv_price.toFixed(3).replace(".", ",")} ct
                            </td>
                            <td className="px-2 py-1 border border-amber-200 text-right tabular-nums text-amber-900">
                              {p.wind_price.toFixed(3).replace(".", ",")} ct
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {(error || success) && (
          <div className={`mb-4 p-3 rounded-lg text-sm ${error ? "bg-red-50 border border-red-200 text-red-700" : "bg-green-50 border border-green-200 text-green-700"}`}>
            {error || success}
          </div>
        )}

        {!selected && !loadingEntries && (
          <div className="bg-white rounded-xl border border-slate-200 p-12 text-center text-slate-400">
            <svg className="mx-auto w-12 h-12 mb-3 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
                d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            <p className="font-medium text-slate-600">Tarifplan auswählen</p>
            <p className="text-sm mt-1">Klicken Sie links auf einen Plan, um die Preiseinträge zu bearbeiten.</p>
          </div>
        )}

        {loadingEntries && (
          <div className="bg-white rounded-xl border border-slate-200 p-12 text-center text-slate-400">
            Lade Einträge...
          </div>
        )}

        {selected && !loadingEntries && (
          <div className="bg-white rounded-xl border border-slate-200 p-6">
            <div className="flex items-center gap-3 mb-6">
              <div>
                <div className="flex items-center gap-2">
                  <h2 className="font-semibold text-slate-900">{selected.name}</h2>
                  {selected.is_active && (
                    <span className="text-xs px-1.5 py-0.5 bg-emerald-50 text-emerald-700 border border-emerald-200 rounded font-medium">
                      Aktiv &#8212; wird für Abrechnung verwendet
                    </span>
                  )}
                </div>
                <p className="text-sm text-slate-500 mt-0.5">
                  {GRANULARITY_LABELS[selected.granularity]} &#183; Preise in ct/kWh (netto)
                </p>
              </div>
            </div>

            {selected.granularity === "monthly" && (
              <MonthlyEditor
                schedule={selected}
                eegEnergyPrice={eegEnergyPrice}
                eegProducerPrice={eegProducerPrice}
                onSave={handleSaveEntries}
                saving={saving}
                oemagData={oemagData}
              />
            )}
            {selected.granularity === "annual" && (
              <AnnualEditor
                schedule={selected}
                eegEnergyPrice={eegEnergyPrice}
                eegProducerPrice={eegProducerPrice}
                onSave={handleSaveEntries}
                saving={saving}
                oemagData={oemagData}
              />
            )}
            {(selected.granularity === "daily" || selected.granularity === "quarter_hour") && (
              <div className="rounded-lg bg-slate-50 border border-slate-200 p-6 text-center text-slate-500">
                <p className="font-medium">Import-basierte Granularität</p>
                <p className="text-sm mt-2">
                  {selected.granularity === "daily"
                    ? "Tägliche Preise werden über einen CSV-Import befüllt (in Entwicklung)."
                    : "15-Minuten Dynamikpreise (z.B. OeMAG) werden automatisch synchronisiert (in Entwicklung)."}
                </p>
                {selected.entries && selected.entries.length > 0 && (
                  <p className="text-sm mt-2 text-slate-400">
                    Aktuell {selected.entries.length} Einträge vorhanden.
                  </p>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
