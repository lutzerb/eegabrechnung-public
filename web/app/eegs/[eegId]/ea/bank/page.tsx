"use client";

import { useEffect, useState, useRef } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

interface BankTransaktion {
  id: string;
  iban: string;
  buchungsdatum: string;
  valuta_datum: string;
  betrag: number;
  verwendungszweck: string;
  auftraggeber?: string;
  match_status: string;
  matched_buchung_id?: string;
  konfidenz?: number;
  kandidaten?: MatchKandidat[];
}

interface MatchKandidat {
  buchung_id: string;
  buchungsnr: string;
  beschreibung: string;
  betrag_brutto: number;
  buchungsdatum: string;
  konfidenz: number;
}

interface ImportResult {
  imported: number;
  auto_matched: number;
  errors: string[];
}

function fmt(n: number): string {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(n);
}

function fmtDate(s?: string): string {
  if (!s) return "—";
  try { return new Date(s).toLocaleDateString("de-AT", { day: "2-digit", month: "2-digit", year: "numeric" }); } catch { return s; }
}

const STATUS_LABEL: Record<string, { label: string; cls: string }> = {
  offen: { label: "offen", cls: "bg-amber-50 text-amber-700" },
  auto: { label: "auto", cls: "bg-blue-50 text-blue-700" },
  bestaetigt: { label: "bestätigt", cls: "bg-green-50 text-green-700" },
  ignoriert: { label: "ignoriert", cls: "bg-slate-100 text-slate-500" },
};

export default function BankPage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;

  const [transaktionen, setTransaktionen] = useState<BankTransaktion[]>([]);
  const [loading, setLoading] = useState(true);
  const [importResult, setImportResult] = useState<ImportResult | null>(null);
  const [uploading, setUploading] = useState(false);
  const [statusFilter, setStatusFilter] = useState("offen");
  const [confirming, setConfirming] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  async function load() {
    setLoading(true);
    const q = statusFilter ? `?status=${statusFilter}` : "";
    const res = await fetch(`/api/eegs/${eegId}/ea/bank/transaktionen${q}`);
    if (res.ok) setTransaktionen(await res.json());
    setLoading(false);
  }

  useEffect(() => { load(); }, [eegId, statusFilter]);

  async function handleFileUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    setImportResult(null);
    try {
      const fd = new FormData();
      fd.append("datei", file);
      const res = await fetch(`/api/eegs/${eegId}/ea/bank/import`, { method: "POST", body: fd });
      if (res.ok) {
        const r = await res.json();
        setImportResult(r);
        await load();
      } else {
        alert((await res.json()).error || "Importfehler");
      }
    } finally {
      setUploading(false);
      e.target.value = "";
    }
  }

  async function handleConfirm(t: BankTransaktion, buchungId?: string) {
    const bid = buchungId || t.kandidaten?.[0]?.buchung_id;
    if (!bid) return;
    setConfirming(t.id);
    const res = await fetch(`/api/eegs/${eegId}/ea/bank/match`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ transaktion_id: t.id, buchung_id: bid }),
    });
    if (res.ok || res.status === 204) await load();
    else alert("Fehler beim Bestätigen");
    setConfirming(null);
  }

  async function handleIgnore(id: string) {
    const res = await fetch(`/api/eegs/${eegId}/ea/bank/transaktionen/${id}`, { method: "DELETE" });
    if (res.ok || res.status === 204) await load();
  }

  return (
    <div className="p-8">
      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">Übersicht</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea`} className="text-sm text-slate-500 hover:text-slate-700">E/A-Buchhaltung</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Bankimport</span>
      </div>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Bankimport & Zuordnung</h1>
          <p className="text-slate-500 mt-1 text-sm">Kontoauszug importieren und Buchungen zuordnen (MT940 / CAMT.053)</p>
        </div>
        <div className="flex gap-2">
          <label className={`px-4 py-2 text-sm font-medium text-white rounded-lg cursor-pointer ${uploading ? "bg-blue-400" : "bg-blue-700 hover:bg-blue-800"}`}>
            {uploading ? "Lädt hoch…" : "Kontoauszug importieren"}
            <input ref={fileRef} type="file" accept=".sta,.xml,.csv" className="hidden" onChange={handleFileUpload} disabled={uploading} />
          </label>
        </div>
      </div>

      {importResult && (
        <div className="mb-4 p-4 bg-green-50 border border-green-200 rounded-lg">
          <p className="font-medium text-green-900 text-sm">Import erfolgreich</p>
          <p className="text-sm text-green-800 mt-1">{importResult.imported} Transaktionen importiert, {importResult.auto_matched} automatisch zugeordnet</p>
          {importResult.errors?.length > 0 && (
            <ul className="mt-2 text-xs text-red-700 list-disc list-inside">
              {importResult.errors.map((e, i) => <li key={i}>{e}</li>)}
            </ul>
          )}
        </div>
      )}

      {/* Filter */}
      <div className="flex gap-2 mb-4">
        {["offen", "auto", "bestaetigt", "ignoriert", ""].map((s) => (
          <button
            key={s}
            onClick={() => setStatusFilter(s)}
            className={`px-3 py-1.5 text-sm rounded-lg border font-medium transition-colors ${statusFilter === s ? "bg-blue-700 text-white border-blue-700" : "bg-white text-slate-700 border-slate-300 hover:bg-slate-50"}`}
          >
            {s === "" ? "Alle" : (STATUS_LABEL[s]?.label || s)}
          </button>
        ))}
      </div>

      {/* Table */}
      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-slate-500 text-sm">Wird geladen…</div>
        ) : transaktionen.length === 0 ? (
          <div className="p-8 text-center text-slate-500 text-sm">Keine Transaktionen{statusFilter ? ` mit Status "${STATUS_LABEL[statusFilter]?.label}"` : ""} vorhanden.</div>
        ) : (
          <div className="divide-y divide-slate-100">
            {transaktionen.map((t) => (
              <div key={t.id} className="p-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3 mb-1">
                      <span className={`text-xs px-2 py-0.5 rounded font-medium ${STATUS_LABEL[t.match_status]?.cls || "bg-slate-100 text-slate-600"}`}>
                        {STATUS_LABEL[t.match_status]?.label || t.match_status}
                      </span>
                      <span className="text-xs text-slate-500">{fmtDate(t.buchungsdatum)}</span>
                      {t.auftraggeber && <span className="text-xs text-slate-600">{t.auftraggeber}</span>}
                    </div>
                    <p className="text-sm text-slate-900 truncate">{t.verwendungszweck}</p>
                    <p className="text-xs text-slate-500 mt-0.5 font-mono">{t.iban}</p>

                    {/* Auto-match kandidaten */}
                    {(t.match_status === "offen" || t.match_status === "auto") && t.kandidaten && t.kandidaten.length > 0 && (
                      <div className="mt-2 space-y-1">
                        {t.kandidaten.map((k) => (
                          <div key={k.buchung_id} className="flex items-center gap-2 text-xs bg-blue-50 rounded p-2">
                            <span className="text-blue-700 flex-1">{k.buchungsnr} — {k.beschreibung} ({fmtDate(k.buchungsdatum)}, {fmt(k.betrag_brutto)})</span>
                            <span className="text-blue-500">{Math.round(k.konfidenz * 100)} %</span>
                            <button
                              onClick={() => handleConfirm(t, k.buchung_id)}
                              disabled={confirming === t.id}
                              className="px-2 py-0.5 bg-blue-700 text-white rounded text-xs hover:bg-blue-800 disabled:opacity-50"
                            >
                              Zuordnen
                            </button>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>

                  <div className="flex flex-col items-end gap-2 flex-shrink-0">
                    <span className={`text-base font-bold ${t.betrag >= 0 ? "text-green-700" : "text-red-700"}`}>
                      {t.betrag >= 0 ? "+" : ""}{fmt(t.betrag)}
                    </span>
                    {t.match_status === "offen" && (
                      <button onClick={() => handleIgnore(t.id)} className="text-xs text-slate-500 hover:underline">Ignorieren</button>
                    )}
                    {t.match_status === "auto" && (
                      <button
                        onClick={() => handleConfirm(t)}
                        disabled={confirming === t.id}
                        className="text-xs text-green-700 border border-green-200 px-2 py-0.5 rounded hover:bg-green-50 disabled:opacity-50"
                      >
                        Bestätigen
                      </button>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="mt-4 p-4 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-600">
        <strong>Unterstützte Formate:</strong> MT940 (.sta, SWIFT-Standard), CAMT.053 (ISO 20022 XML).
        Automatische Zuordnung bei Übereinstimmung ≥ 90 % (Betrag ±0,1 %, Datum ±14 Tage).
        Zugeordnete Buchungen erhalten automatisch das Bankbuchungsdatum als Zahlungsdatum.
      </div>
    </div>
  );
}
