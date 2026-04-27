"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

interface ImportPreviewRow {
  invoice_id: string;
  invoice_nr: string;
  datum: string;
  document_type: string;
  business_role: string;
  mitglied_name: string;
  split_part: string; // "" | "bezug" | "einspeisung"
  konto_nummer: string;
  konto_name: string;
  betrag_brutto: number;
  ust_code: string;
  already_imported: boolean;
}

interface ImportResult {
  imported: number;
  skipped: number;
  errors: string[];
}

function fmt(n: number): string {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(n);
}

function fmtDate(s?: string): string {
  if (!s) return "—";
  try { return new Date(s).toLocaleDateString("de-AT", { day: "2-digit", month: "2-digit", year: "numeric" }); } catch { return s; }
}

const DOC_LABELS: Record<string, string> = { invoice: "Rechnung", credit_note: "Gutschrift" };
const ROLE_LABELS: Record<string, string> = { consumer: "Verbraucher", prosumer: "Prosumer", producer: "Erzeuger", unternehmen: "Unternehmen", landwirt_pauschaliert: "LuF pauschaliert" };

export default function EAImportPage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;
  const curYear = new Date().getFullYear();

  const [preview, setPreview] = useState<ImportPreviewRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [importing, setImporting] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [year, setYear] = useState(curYear);
  const [error, setError] = useState("");

  async function loadPreview() {
    setLoading(true);
    setError("");
    setResult(null);
    const res = await fetch(`/api/eegs/${eegId}/ea/import?von=${year}-01-01&bis=${year}-12-31`);
    if (res.ok) {
      const data = await res.json();
      setPreview(data || []);
      // Pre-select all not yet imported
      setSelected(new Set((data || []).filter((r: ImportPreviewRow) => !r.already_imported).map((r: ImportPreviewRow) => r.invoice_id)));
    } else {
      setError("Vorschau konnte nicht geladen werden.");
    }
    setLoading(false);
  }

  useEffect(() => { loadPreview(); }, [eegId, year]);

  // Unique not-yet-imported invoice IDs (split rows share the same invoice_id)
  const notImportedIds = [...new Set(preview.filter((r) => !r.already_imported).map((r) => r.invoice_id))];

  function toggleAll(checked: boolean) {
    if (checked) setSelected(new Set(notImportedIds));
    else setSelected(new Set());
  }

  function toggleRow(id: string) {
    const s = new Set(selected);
    if (s.has(id)) s.delete(id); else s.add(id);
    setSelected(s);
  }

  async function handleImport() {
    if (selected.size === 0) return;
    setImporting(true);
    setError("");
    const res = await fetch(`/api/eegs/${eegId}/ea/import`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ invoice_ids: Array.from(selected) }),
    });
    if (res.ok) {
      const r = await res.json();
      setResult(r);
      await loadPreview();
    } else {
      setError((await res.json()).error || "Importfehler");
    }
    setImporting(false);
  }

  const years = Array.from({ length: 5 }, (_, i) => curYear - i);

  return (
    <div className="p-8">
      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">Übersicht</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea`} className="text-sm text-slate-500 hover:text-slate-700">E/A-Buchhaltung</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Rechnungsimport</span>
      </div>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">EEG-Rechnungen importieren</h1>
          <p className="text-slate-500 mt-1 text-sm">Finalisierte Rechnungen automatisch als Buchungen übernehmen</p>
        </div>
        <div className="flex items-center gap-3">
          <select value={year} onChange={(e) => setYear(Number(e.target.value))} className="px-3 py-2 border border-slate-300 rounded-lg text-slate-900 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
            {years.map((y) => <option key={y} value={y}>{y}</option>)}
          </select>
        </div>
      </div>

      {error && <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-sm text-red-700">{error}</div>}

      {result && (
        <div className="mb-4 p-4 bg-green-50 border border-green-200 rounded-lg">
          <p className="font-medium text-green-900 text-sm">Import abgeschlossen</p>
          <p className="text-sm text-green-800 mt-1">{result.imported} importiert, {result.skipped} übersprungen</p>
          {result.errors?.length > 0 && (
            <ul className="mt-2 text-xs text-red-700 list-disc list-inside">
              {result.errors.map((e, i) => <li key={i}>{e}</li>)}
            </ul>
          )}
        </div>
      )}

      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        <div className="px-5 py-3 border-b border-slate-200 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <label className="flex items-center gap-2 text-sm text-slate-700 cursor-pointer">
              <input
                type="checkbox"
                checked={selected.size === notImportedIds.length && notImportedIds.length > 0}
                onChange={(e) => toggleAll(e.target.checked)}
                className="rounded border-slate-300"
              />
              Alle auswählen ({notImportedIds.length} neu)
            </label>
          </div>
          <button
            onClick={handleImport}
            disabled={importing || selected.size === 0}
            className="px-4 py-2 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800 disabled:opacity-50"
          >
            {importing ? "Importiert…" : `${selected.size} Rechnung${selected.size !== 1 ? "en" : ""} importieren`}
          </button>
        </div>

        {loading ? (
          <div className="p-8 text-center text-slate-500 text-sm">Wird geladen…</div>
        ) : preview.length === 0 ? (
          <div className="p-8 text-center text-slate-500 text-sm">Keine finalisierten Rechnungen für {year} vorhanden.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 bg-slate-50">
                  <th className="px-4 py-3 w-8"></th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500">Datum</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500">Rechnungs-Nr.</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500">Mitglied</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500">Rolle</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500">Typ</th>
                  <th className="text-right px-4 py-3 text-xs font-medium text-slate-500">Brutto</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500">USt-Code</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500">Konto</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {preview.map((r) => (
                  <tr key={`${r.invoice_id}-${r.split_part || 'main'}`} className={`hover:bg-slate-50 ${r.already_imported ? "opacity-50" : ""}`}>
                    <td className="px-4 py-3">
                      <input
                        type="checkbox"
                        checked={selected.has(r.invoice_id)}
                        disabled={r.already_imported}
                        onChange={() => toggleRow(r.invoice_id)}
                        className="rounded border-slate-300"
                      />
                    </td>
                    <td className="px-4 py-3 text-slate-600">{fmtDate(r.datum)}</td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-700">{r.invoice_nr}</td>
                    <td className="px-4 py-3 text-slate-900">{r.mitglied_name}</td>
                    <td className="px-4 py-3 text-slate-600 text-xs">{ROLE_LABELS[r.business_role] || r.business_role}</td>
                    <td className="px-4 py-3">
                      {r.split_part ? (
                        <span className={`inline-flex px-2 py-0.5 rounded text-xs ${r.split_part === "einspeisung" ? "bg-green-50 text-green-700" : "bg-blue-50 text-blue-700"}`}>
                          {r.split_part === "bezug" ? "Bezug" : "Einspeisung"}
                        </span>
                      ) : (
                        <span className={`inline-flex px-2 py-0.5 rounded text-xs ${r.document_type === "credit_note" ? "bg-green-50 text-green-700" : "bg-blue-50 text-blue-700"}`}>
                          {DOC_LABELS[r.document_type] || r.document_type}
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-right font-medium">{fmt(r.betrag_brutto)}</td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-600">{r.ust_code || "KEINE"}</td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-500">{r.konto_nummer} {r.konto_name}</td>
                    <td className="px-4 py-3">
                      {r.already_imported ? (
                        <span className="text-xs bg-slate-100 text-slate-500 px-2 py-0.5 rounded">importiert</span>
                      ) : (
                        <span className="text-xs bg-amber-50 text-amber-700 px-2 py-0.5 rounded">neu</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="mt-4 p-4 bg-blue-50 border border-blue-200 rounded-lg text-xs text-blue-800">
        <strong>Automatische USt-Code-Erkennung:</strong> Gutschriften an Unternehmen → RC_20 (Reverse Charge 20 %), Gutschriften an pauschalierte LuF → RC_13 (RC 13 %). Rechnungen an Verbraucher → KEINE.
        Die Buchungen werden im Journal aufgeführt und fließen automatisch in die UVA ein.
      </div>
    </div>
  );
}
