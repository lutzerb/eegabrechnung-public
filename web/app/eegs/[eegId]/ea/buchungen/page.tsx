"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

interface EAKonto {
  id: string;
  nummer: string;
  name: string;
  typ: string;
}

interface EABuchung {
  id: string;
  buchungsnr: string;
  beleg_datum?: string;
  zahlung_datum?: string;
  konto_id: string;
  konto?: { nummer: string; name: string };
  beschreibung: string;
  betrag_brutto: number;
  betrag_netto: number;
  ust_betrag: number;
  ust_code: string;
  richtung: string;
  quelle: string;
  deleted_at?: string;
}

function fmt(n: number): string {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(n);
}

function fmtDate(s?: string): string {
  if (!s) return "—";
  try { return new Date(s).toLocaleDateString("de-AT", { day: "2-digit", month: "2-digit", year: "numeric" }); } catch { return s; }
}

interface DeleteModal {
  buchungId: string;
  beschreibung: string;
  reason: string;
}

export default function BuchungenPage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;
  const curYear = new Date().getFullYear();

  const [buchungen, setBuchungen] = useState<EABuchung[]>([]);
  const [konten, setKonten] = useState<EAKonto[]>([]);
  const [loading, setLoading] = useState(true);
  const [exportLoading, setExportLoading] = useState(false);
  const [deleteModal, setDeleteModal] = useState<DeleteModal | null>(null);
  const [deleting, setDeleting] = useState(false);

  const [jahr, setJahr] = useState(curYear);
  const [kontoId, setKontoId] = useState("");
  const [richtung, setRichtung] = useState("");
  const [nurBezahlt, setNurBezahlt] = useState(false);
  const [inclDeleted, setInclDeleted] = useState(false);

  async function load() {
    setLoading(true);
    const q = new URLSearchParams({ jahr: String(jahr) });
    if (kontoId) q.set("konto_id", kontoId);
    if (richtung) q.set("richtung", richtung);
    if (nurBezahlt) q.set("bezahlt", "true");
    if (inclDeleted) q.set("incl_deleted", "true");
    const res = await fetch(`/api/eegs/${eegId}/ea/buchungen?${q}`);
    if (res.ok) setBuchungen(await res.json());
    setLoading(false);
  }

  useEffect(() => {
    fetch(`/api/eegs/${eegId}/ea/konten`).then((r) => r.ok && r.json()).then((d) => d && setKonten(d));
  }, [eegId]);

  useEffect(() => { load(); }, [eegId, jahr, kontoId, richtung, nurBezahlt, inclDeleted]);

  async function handleExport() {
    setExportLoading(true);
    try {
      const q = new URLSearchParams({ jahr: String(jahr) });
      if (kontoId) q.set("konto_id", kontoId);
      const res = await fetch(`/api/eegs/${eegId}/ea/buchungen/export?${q}`);
      if (!res.ok) { alert("Exportfehler"); return; }
      const blob = await res.blob();
      const cd = res.headers.get("Content-Disposition") || "";
      const m = cd.match(/filename="([^"]+)"/);
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = m ? m[1] : `buchungen_${jahr}.xlsx`;
      a.click();
    } finally {
      setExportLoading(false);
    }
  }

  function openDeleteModal(b: EABuchung) {
    setDeleteModal({ buchungId: b.id, beschreibung: b.beschreibung, reason: "" });
  }

  async function confirmDelete() {
    if (!deleteModal) return;
    setDeleting(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/ea/buchungen/${deleteModal.buchungId}`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reason: deleteModal.reason }),
      });
      if (res.ok || res.status === 204) {
        setDeleteModal(null);
        await load();
      } else {
        alert("Fehler beim Löschen");
      }
    } finally {
      setDeleting(false);
    }
  }

  const years = Array.from({ length: 6 }, (_, i) => curYear - i);
  const selectClass = "px-3 py-2 border border-slate-300 rounded-lg text-slate-900 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500";

  const activeBuchungen = buchungen.filter((b) => !b.deleted_at);
  const sumEin = activeBuchungen.filter((b) => b.richtung === "EINNAHME").reduce((s, b) => s + b.betrag_brutto, 0);
  const sumAus = activeBuchungen.filter((b) => b.richtung === "AUSGABE").reduce((s, b) => s + b.betrag_brutto, 0);

  return (
    <div className="p-8">
      {/* Delete modal */}
      {deleteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold text-slate-900 mb-1">Buchung löschen</h2>
            <p className="text-sm text-slate-500 mb-4">
              <span className="font-medium text-slate-700">{deleteModal.beschreibung}</span>
              <br />
              Die Buchung wird als gelöscht markiert und bleibt im Audit-Trail sichtbar (BAO §131).
            </p>
            <label className="block text-xs font-medium text-slate-700 mb-1">
              Grund <span className="text-red-500">*</span>
            </label>
            <textarea
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
              rows={3}
              placeholder="z.B. Doppelbuchung, Falscheingabe, storniert …"
              value={deleteModal.reason}
              onChange={(e) => setDeleteModal({ ...deleteModal, reason: e.target.value })}
            />
            <div className="flex gap-3 mt-4 justify-end">
              <button
                onClick={() => setDeleteModal(null)}
                disabled={deleting}
                className="px-4 py-2 text-sm text-slate-700 border border-slate-300 rounded-lg hover:bg-slate-50 disabled:opacity-50"
              >
                Abbrechen
              </button>
              <button
                onClick={confirmDelete}
                disabled={deleting || !deleteModal.reason.trim()}
                className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 disabled:opacity-50"
              >
                {deleting ? "Löscht…" : "Löschen"}
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">Übersicht</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea`} className="text-sm text-slate-500 hover:text-slate-700">E/A-Buchhaltung</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Journal</span>
      </div>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Journal</h1>
          <p className="text-slate-500 mt-1 text-sm">Alle Buchungen nach Belegdatum</p>
        </div>
        <div className="flex gap-2">
          <Link href={`/eegs/${eegId}/ea/changelog`} className="px-3 py-2 text-sm text-slate-600 border border-slate-300 rounded-lg hover:bg-slate-50">
            Audit-Trail
          </Link>
          <button onClick={handleExport} disabled={exportLoading} className="px-3 py-2 text-sm text-slate-600 border border-slate-300 rounded-lg hover:bg-slate-50 disabled:opacity-50">
            {exportLoading ? "Exportiert…" : "XLSX"}
          </button>
          <Link href={`/eegs/${eegId}/ea/buchungen/neu`} className="px-4 py-2 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800">
            + Buchung
          </Link>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-xl border border-slate-200 p-4 mb-4 flex flex-wrap gap-3 items-end">
        <div>
          <label className="block text-xs font-medium text-slate-700 mb-1">Jahr</label>
          <select className={selectClass} value={jahr} onChange={(e) => setJahr(Number(e.target.value))}>
            {years.map((y) => <option key={y} value={y}>{y}</option>)}
          </select>
        </div>
        <div>
          <label className="block text-xs font-medium text-slate-700 mb-1">Konto</label>
          <select className={selectClass} value={kontoId} onChange={(e) => setKontoId(e.target.value)}>
            <option value="">Alle Konten</option>
            {konten.map((k) => <option key={k.id} value={k.id}>{k.nummer} {k.name}</option>)}
          </select>
        </div>
        <div>
          <label className="block text-xs font-medium text-slate-700 mb-1">Richtung</label>
          <select className={selectClass} value={richtung} onChange={(e) => setRichtung(e.target.value)}>
            <option value="">Alle</option>
            <option value="EINNAHME">Einnahmen</option>
            <option value="AUSGABE">Ausgaben</option>
          </select>
        </div>
        <label className="flex items-center gap-2 text-sm text-slate-700 cursor-pointer">
          <input type="checkbox" checked={nurBezahlt} onChange={(e) => setNurBezahlt(e.target.checked)} className="rounded border-slate-300" />
          Nur bezahlte
        </label>
        <label className="flex items-center gap-2 text-sm text-slate-500 cursor-pointer">
          <input type="checkbox" checked={inclDeleted} onChange={(e) => setInclDeleted(e.target.checked)} className="rounded border-slate-300" />
          Gelöschte anzeigen
        </label>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-3 gap-3 mb-4">
        <div className="bg-green-50 border border-green-200 rounded-lg p-3 text-center">
          <p className="text-xs text-green-700 font-medium">Einnahmen</p>
          <p className="text-lg font-bold text-green-900">{fmt(sumEin)}</p>
        </div>
        <div className="bg-red-50 border border-red-200 rounded-lg p-3 text-center">
          <p className="text-xs text-red-700 font-medium">Ausgaben</p>
          <p className="text-lg font-bold text-red-900">{fmt(sumAus)}</p>
        </div>
        <div className="bg-slate-50 border border-slate-200 rounded-lg p-3 text-center">
          <p className="text-xs text-slate-700 font-medium">Saldo</p>
          <p className={`text-lg font-bold ${sumEin - sumAus >= 0 ? "text-slate-900" : "text-red-700"}`}>{fmt(sumEin - sumAus)}</p>
        </div>
      </div>

      {/* Table */}
      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-slate-500 text-sm">Wird geladen…</div>
        ) : buchungen.length === 0 ? (
          <div className="p-8 text-center text-slate-500 text-sm">Keine Buchungen gefunden.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 bg-slate-50">
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase">Beleg-Nr.</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase">Belegdatum</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase">Zahlung</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase">Konto</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase">Beschreibung</th>
                  <th className="text-right px-4 py-3 text-xs font-medium text-slate-500 uppercase">Brutto</th>
                  <th className="text-right px-4 py-3 text-xs font-medium text-slate-500 uppercase">USt</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase">Code</th>
                  <th className="px-4 py-3"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {buchungen.map((b) => {
                  const isDeleted = !!b.deleted_at;
                  return (
                    <tr key={b.id} className={isDeleted ? "opacity-40 bg-slate-50" : "hover:bg-slate-50"}>
                      <td className="px-4 py-3 font-mono text-xs text-slate-500">
                        {b.buchungsnr}
                        {isDeleted && <span className="ml-1 text-red-500 font-semibold">[GELÖSCHT]</span>}
                      </td>
                      <td className="px-4 py-3 text-slate-700">{fmtDate(b.beleg_datum)}</td>
                      <td className="px-4 py-3 text-slate-500">
                        {b.zahlung_datum ? fmtDate(b.zahlung_datum) : <span className="text-amber-600 text-xs">ausstehend</span>}
                      </td>
                      <td className="px-4 py-3 text-slate-600 font-mono text-xs">
                        {b.konto?.nummer} {b.konto?.name}
                      </td>
                      <td className="px-4 py-3 text-slate-900 max-w-xs truncate">{b.beschreibung}</td>
                      <td className={`px-4 py-3 text-right font-medium ${b.richtung === "EINNAHME" ? "text-green-700" : "text-red-700"}`}>
                        {b.richtung === "AUSGABE" ? "–" : ""}{fmt(b.betrag_brutto)}
                      </td>
                      <td className="px-4 py-3 text-right text-slate-500">{b.ust_betrag !== 0 ? fmt(b.ust_betrag) : "—"}</td>
                      <td className="px-4 py-3 text-xs text-slate-500 font-mono">{b.ust_code || "KEINE"}</td>
                      <td className="px-4 py-3 flex gap-2 justify-end">
                        <Link href={`/eegs/${eegId}/ea/buchungen/${b.id}`} className="text-xs text-blue-600 hover:underline">Detail</Link>
                        {!isDeleted && b.quelle === "manual" && (
                          <button onClick={() => openDeleteModal(b)} className="text-xs text-red-600 hover:underline">Löschen</button>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
