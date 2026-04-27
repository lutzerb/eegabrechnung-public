"use client";

import { useState } from "react";
import { useSession } from "next-auth/react";
import { useRouter } from "next/navigation";
import type { BillingRun, Member } from "@/lib/api";

interface BillingRunFormProps {
  eegId: string;
  members: Member[];
}

interface OverlapInfo {
  existing_run_id: string;
  existing_period_start: string;
  existing_period_end: string;
}

function getDefaultPeriod(): { start: string; end: string } {
  const now = new Date();
  const firstDayLastMonth = new Date(now.getFullYear(), now.getMonth() - 1, 1);
  const lastDayLastMonth = new Date(now.getFullYear(), now.getMonth(), 0);

  const fmt = (d: Date) =>
    `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;

  return {
    start: fmt(firstDayLastMonth),
    end: fmt(lastDayLastMonth),
  };
}

export default function BillingRunForm({ eegId, members }: BillingRunFormProps) {
  const { data: session } = useSession();
  const router = useRouter();
  const defaultPeriod = getDefaultPeriod();

  const [periodStart, setPeriodStart] = useState(defaultPeriod.start);
  const [periodEnd, setPeriodEnd] = useState(defaultPeriod.end);
  const [selectedMonth, setSelectedMonth] = useState<string>(() => {
    const now = new Date();
    const y = now.getMonth() === 0 ? now.getFullYear() - 1 : now.getFullYear();
    const m = now.getMonth() === 0 ? 12 : now.getMonth();
    return `${y}-${String(m).padStart(2, "0")}`;
  });
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [selectedMemberIds, setSelectedMemberIds] = useState<string[]>([]);
  const [billingType, setBillingType] = useState("all");
  const [force, setForce] = useState(false);
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState<BillingRun | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [overlap, setOverlap] = useState<OverlapInfo | null>(null);
  const [previewResult, setPreviewResult] = useState<{ invoices: { member_id: string; total_amount: number; consumption_kwh: number; generation_kwh: number }[]; count: number } | null>(null);

  const toggleMember = (id: string) => {
    setSelectedMemberIds((prev) =>
      prev.includes(id) ? prev.filter((m) => m !== id) : [...prev, id]
    );
  };

  const handleSubmit = async (e: React.FormEvent, forceOverride?: boolean, preview = false) => {
    e.preventDefault();
    if (!session?.accessToken) return;

    setLoading(true);
    setError(null);
    setSuccess(null);
    setOverlap(null);
    setPreviewResult(null);

    try {
      const body: Record<string, unknown> = {
        period_start: periodStart,
        period_end: periodEnd,
        force: forceOverride ?? force,
        preview,
      };
      if (selectedMemberIds.length > 0) body.member_ids = selectedMemberIds;
      if (billingType !== "all") body.billing_type = billingType;

      const res = await fetch(`/api/eegs/${eegId}/billing/run`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${session.accessToken}`,
        },
        body: JSON.stringify(body),
      });

      if (res.status === 409) {
        const data = await res.json();
        setOverlap({
          existing_run_id: data.existing_run_id,
          existing_period_start: data.existing_period_start,
          existing_period_end: data.existing_period_end,
        });
        return;
      }

      if (!res.ok) {
        let msg = `HTTP ${res.status}`;
        try {
          const body = await res.json();
          msg = body.message || body.error || msg;
        } catch {
          // ignore
        }
        throw new Error(msg);
      }

      const data = await res.json();
      if (preview) {
        setPreviewResult({ invoices: data.invoices || [], count: data.invoices_created || 0 });
      } else {
        setSuccess(data.billing_run ?? data);
        router.refresh();
      }
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Fehler beim Starten der Abrechnung.");
    } finally {
      setLoading(false);
    }
  };

  const handleMonthSelect = (month: string) => {
    setSelectedMonth(month);
    if (!month) return;
    const [y, m] = month.split("-").map(Number);
    const fmt = (d: Date) =>
      `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
    setPeriodStart(fmt(new Date(y, m - 1, 1)));
    setPeriodEnd(fmt(new Date(y, m, 0)));
  };

  const memberLabel = (m: Member) => [m.name1, m.name2].filter(Boolean).join(" ") || m.name || m.id;
  const memberMap = new Map(members.map(m => [m.id, memberLabel(m)]));
  const fmtDate = (d: string) => new Date(d).toLocaleDateString("de-AT");

  return (
    <div className="bg-white rounded-xl border border-slate-200 p-6">
      <div className="flex items-start gap-4 mb-5">
        <div className="w-10 h-10 rounded-lg bg-amber-50 flex items-center justify-center flex-shrink-0">
          <svg className="w-5 h-5 text-amber-700" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
              d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
          </svg>
        </div>
        <div>
          <h3 className="font-semibold text-slate-900">Abrechnung starten</h3>
          <p className="text-sm text-slate-500 mt-0.5">
            Wählen Sie den Abrechnungszeitraum und starten Sie die Abrechnung.
          </p>
        </div>
      </div>

      {success && (
        <div className="mb-5 p-4 bg-green-50 border border-green-200 rounded-lg">
          <div className="flex items-start gap-2">
            <svg className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div>
              <p className="font-medium text-green-800">Abrechnung erfolgreich gestartet</p>
              <p className="text-sm text-green-700 mt-0.5">
                Abrechnungslauf ID: <span className="font-mono">{success.id}</span>
              </p>
            </div>
          </div>
        </div>
      )}

      {previewResult && (
        <div className="mb-5 p-4 bg-blue-50 border border-blue-200 rounded-lg">
          <p className="font-medium text-blue-800 mb-2">Vorschau ({previewResult.count} Rechnungen — nicht gespeichert)</p>
          <table className="w-full text-xs text-blue-900">
            <thead>
              <tr className="border-b border-blue-200">
                <th className="text-left py-1">Mitglied</th>
                <th className="text-right py-1">Bezug kWh</th>
                <th className="text-right py-1">Einspeisung kWh</th>
                <th className="text-right py-1">Betrag</th>
              </tr>
            </thead>
            <tbody>
              {previewResult.invoices.map((inv, i) => (
                <tr key={i} className="border-b border-blue-100">
                  <td className="py-1">{memberMap.get(inv.member_id) ?? inv.member_id.slice(0, 8)}</td>
                  <td className="py-1 text-right">{inv.consumption_kwh?.toFixed(2)}</td>
                  <td className="py-1 text-right">{inv.generation_kwh?.toFixed(2)}</td>
                  <td className="py-1 text-right font-medium">{inv.total_amount?.toFixed(2)} €</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr className="border-t border-blue-300 font-semibold">
                <td className="py-1">Gesamt</td>
                <td className="py-1 text-right">
                  {previewResult.invoices.reduce((s, inv) => s + (inv.consumption_kwh ?? 0), 0).toFixed(2)}
                </td>
                <td className="py-1 text-right">
                  {previewResult.invoices.reduce((s, inv) => s + (inv.generation_kwh ?? 0), 0).toFixed(2)}
                </td>
                <td className="py-1 text-right">
                  {previewResult.invoices.reduce((s, inv) => s + (inv.total_amount ?? 0), 0).toFixed(2)} €
                </td>
              </tr>
            </tfoot>
          </table>
          <p className="text-xs text-blue-600 mt-2">Klicken Sie auf &quot;Abrechnung starten&quot; um fortzufahren.</p>
        </div>
      )}

      {overlap && (
        <div className="mb-5 p-4 bg-amber-50 border border-amber-200 rounded-lg">
          <p className="font-medium text-amber-800">Zeitraumüberschneidung</p>
          <p className="text-sm text-amber-700 mt-1">
            Ein bestehender Abrechnungslauf ({fmtDate(overlap.existing_period_start)} – {fmtDate(overlap.existing_period_end)}) überschneidet sich mit dem gewählten Zeitraum.
          </p>
          <button
            onClick={(e) => { e.preventDefault(); setForce(true); setOverlap(null); handleSubmit(new Event("submit") as unknown as React.FormEvent, true); }}
            className="mt-3 px-4 py-1.5 text-sm font-medium text-white bg-amber-600 rounded hover:bg-amber-700 transition-colors"
          >
            Trotzdem ausführen (Überschneidung ignorieren)
          </button>
        </div>
      )}

      {error && (
        <div className="mb-5 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <div className="flex items-start gap-2">
            <svg className="w-5 h-5 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div>
              <p className="font-medium">Fehler bei der Abrechnung</p>
              <p className="text-sm mt-0.5">{error}</p>
            </div>
          </div>
        </div>
      )}

      <form onSubmit={handleSubmit}>
        <div className="flex flex-wrap gap-4 items-end mb-3">
          <div className="w-48">
            <label className="block text-sm font-medium text-slate-700 mb-1.5">Monat</label>
            <input
              type="month"
              value={selectedMonth}
              onChange={(e) => handleMonthSelect(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-transparent"
            />
          </div>
        </div>

        <div className="flex flex-wrap gap-4 items-end">
          <div className="flex-1 min-w-48">
            <label className="block text-sm font-medium text-slate-700 mb-1.5">Von</label>
            <input
              type="date"
              required
              value={periodStart}
              onChange={(e) => { setPeriodStart(e.target.value); setSelectedMonth(""); }}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-transparent"
            />
          </div>

          <div className="flex-1 min-w-48">
            <label className="block text-sm font-medium text-slate-700 mb-1.5">Bis</label>
            <input
              type="date"
              required
              value={periodEnd}
              min={periodStart}
              onChange={(e) => { setPeriodEnd(e.target.value); setSelectedMonth(""); }}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-transparent"
            />
          </div>

          <button
            type="button"
            disabled={loading || !periodStart || !periodEnd}
            onClick={(e) => handleSubmit(e as unknown as React.FormEvent, undefined, true)}
            className="px-5 py-2.5 bg-slate-100 text-slate-700 font-medium rounded-lg hover:bg-slate-200 disabled:opacity-50 disabled:cursor-not-allowed transition-colors whitespace-nowrap"
          >
            Vorschau
          </button>
          <button
            type="submit"
            disabled={loading || !periodStart || !periodEnd}
            className="px-6 py-2.5 bg-amber-600 text-white font-medium rounded-lg hover:bg-amber-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors focus:outline-none focus:ring-2 focus:ring-amber-500 focus:ring-offset-2 whitespace-nowrap"
          >
            {loading ? (
              <span className="flex items-center gap-2">
                <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                Abrechnung läuft...
              </span>
            ) : (
              "Abrechnung starten"
            )}
          </button>
        </div>

        {periodStart && periodEnd && (
          <p className="text-xs text-slate-500 mt-2">
            Abrechnungszeitraum: {new Date(periodStart).toLocaleDateString("de-AT")} bis{" "}
            {new Date(periodEnd).toLocaleDateString("de-AT")}
          </p>
        )}

        {/* Advanced options toggle */}
        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="mt-4 text-xs text-slate-500 hover:text-slate-700 flex items-center gap-1 transition-colors"
        >
          <svg
            className={`w-3.5 h-3.5 transition-transform ${showAdvanced ? "rotate-90" : ""}`}
            fill="none" viewBox="0 0 24 24" stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
          Erweiterte Optionen
          {(selectedMemberIds.length > 0 || billingType !== "all") && (
            <span className="ml-1 px-1.5 py-0.5 bg-amber-100 text-amber-700 rounded text-xs">aktiv</span>
          )}
        </button>

        {showAdvanced && (
          <div className="mt-4 space-y-5 border-t border-slate-100 pt-4">
            {/* Billing type */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">Abrechnungstyp</label>
              <div className="flex flex-wrap gap-3">
                {[
                  { value: "all", label: "Alle (Bezug & Einspeisung)" },
                  { value: "consumption_only", label: "Nur Bezug" },
                  { value: "production_only", label: "Nur Einspeisung" },
                ].map((opt) => (
                  <label key={opt.value} className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="billing_type"
                      value={opt.value}
                      checked={billingType === opt.value}
                      onChange={() => setBillingType(opt.value)}
                      className="text-amber-600 focus:ring-amber-500"
                    />
                    <span className="text-sm text-slate-700">{opt.label}</span>
                  </label>
                ))}
              </div>
            </div>

            {/* Member filter */}
            {members.length > 0 && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="text-sm font-medium text-slate-700">
                    Mitglieder einschränken
                    <span className="ml-1 text-xs text-slate-400 font-normal">(leer = alle)</span>
                  </label>
                  {selectedMemberIds.length > 0 && (
                    <button
                      type="button"
                      onClick={() => setSelectedMemberIds([])}
                      className="text-xs text-slate-400 hover:text-slate-600"
                    >
                      Auswahl aufheben
                    </button>
                  )}
                </div>
                <div className="space-y-1.5 max-h-40 overflow-y-auto border border-slate-200 rounded-lg p-3">
                  {members.map((m) => (
                    <label key={m.id} className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={selectedMemberIds.includes(m.id)}
                        onChange={() => toggleMember(m.id)}
                        className="w-4 h-4 rounded border-slate-300 text-amber-600 focus:ring-amber-500"
                      />
                      <span className="text-sm text-slate-700">{memberLabel(m)}</span>
                      {m.business_role && (
                        <span className="text-xs text-slate-400">{m.business_role}</span>
                      )}
                    </label>
                  ))}
                </div>
                {selectedMemberIds.length > 0 && (
                  <p className="text-xs text-amber-600 mt-1">
                    {selectedMemberIds.length} von {members.length} Mitgliedern ausgewählt
                  </p>
                )}
              </div>
            )}
          </div>
        )}
      </form>
    </div>
  );
}
