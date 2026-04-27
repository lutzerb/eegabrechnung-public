"use client";

import { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

export default function AccountingExportPage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;

  const currentYear = new Date().getFullYear();
  const currentMonth = new Date().getMonth() + 1;

  const [mode, setMode] = useState<"month" | "year">("month");
  const [year, setYear] = useState(currentYear);
  const [month, setMonth] = useState(currentMonth);
  const [format, setFormat] = useState<"xlsx" | "datev">("xlsx");
  const [loading, setLoading] = useState(false);

  const monthNames = [
    "", "Jänner", "Februar", "März", "April", "Mai", "Juni",
    "Juli", "August", "September", "Oktober", "November", "Dezember",
  ];

  function buildParams() {
    let from: string;
    let to: string;
    if (mode === "month") {
      const paddedMonth = String(month).padStart(2, "0");
      const lastDay = new Date(year, month, 0).getDate();
      from = `${year}-${paddedMonth}-01`;
      to = `${year}-${paddedMonth}-${String(lastDay).padStart(2, "0")}`;
    } else {
      from = `${year}-01-01`;
      to = `${year}-12-31`;
    }
    return { from, to };
  }

  async function handleExport() {
    setLoading(true);
    try {
      const { from, to } = buildParams();
      const url = `/api/eegs/${eegId}/accounting/export?from=${from}&to=${to}&format=${format}`;
      const res = await fetch(url);
      if (!res.ok) {
        const err = await res.text();
        alert("Fehler beim Export: " + err);
        return;
      }
      const blob = await res.blob();
      const disposition = res.headers.get("Content-Disposition") || "";
      const match = disposition.match(/filename="([^"]+)"/);
      const filename = match ? match[1] : `export_${from}_${to}.${format === "datev" ? "csv" : "xlsx"}`;
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = filename;
      a.click();
      URL.revokeObjectURL(a.href);
    } finally {
      setLoading(false);
    }
  }

  const yearRange = Array.from({ length: 6 }, (_, i) => currentYear - i);

  const inputClass = "px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500";

  return (
    <div className="p-8 max-w-2xl">
      {/* Breadcrumb */}
      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">
          Übersicht
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Buchhaltungsexport</span>
      </div>

      <div className="mb-8">
        <h1 className="text-2xl font-bold text-slate-900">Buchhaltungsexport</h1>
        <p className="text-slate-500 mt-1">
          Rechnungen und Gutschriften mit MwSt-Aufschlüsselung exportieren.
        </p>
      </div>

      <div className="bg-white rounded-xl border border-slate-200 p-6 space-y-5">
        {/* Mode toggle */}
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-2">Zeitraum</label>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setMode("month")}
              className={`px-4 py-2 text-sm rounded-lg border font-medium transition-colors ${
                mode === "month"
                  ? "bg-blue-700 text-white border-blue-700"
                  : "bg-white text-slate-700 border-slate-300 hover:bg-slate-50"
              }`}
            >
              Monat
            </button>
            <button
              type="button"
              onClick={() => setMode("year")}
              className={`px-4 py-2 text-sm rounded-lg border font-medium transition-colors ${
                mode === "year"
                  ? "bg-blue-700 text-white border-blue-700"
                  : "bg-white text-slate-700 border-slate-300 hover:bg-slate-50"
              }`}
            >
              Gesamtjahr
            </button>
          </div>
        </div>

        {/* Year + month selectors */}
        <div className="flex gap-3">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1.5">Jahr</label>
            <select
              value={year}
              onChange={(e) => setYear(Number(e.target.value))}
              className={inputClass}
            >
              {yearRange.map((y) => (
                <option key={y} value={y}>{y}</option>
              ))}
            </select>
          </div>
          {mode === "month" && (
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">Monat</label>
              <select
                value={month}
                onChange={(e) => setMonth(Number(e.target.value))}
                className={inputClass}
              >
                {monthNames.slice(1).map((name, i) => (
                  <option key={i + 1} value={i + 1}>{name}</option>
                ))}
              </select>
            </div>
          )}
        </div>

        {/* Format selector */}
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-2">Format</label>
          <div className="space-y-2">
            <label className="flex items-start gap-3 cursor-pointer">
              <input
                type="radio"
                name="format"
                value="xlsx"
                checked={format === "xlsx"}
                onChange={() => setFormat("xlsx")}
                className="mt-0.5 h-4 w-4 text-blue-700 border-slate-300"
              />
              <div>
                <span className="text-sm font-medium text-slate-900">XLSX (Excel)</span>
                <p className="text-xs text-slate-500 mt-0.5">
                  Universell verwendbar — Nettobetrag, MwSt-Satz, MwSt-Betrag, Bruttobetrag je Rechnung
                </p>
              </div>
            </label>
            <label className="flex items-start gap-3 cursor-pointer">
              <input
                type="radio"
                name="format"
                value="datev"
                checked={format === "datev"}
                onChange={() => setFormat("datev")}
                className="mt-0.5 h-4 w-4 text-blue-700 border-slate-300"
              />
              <div>
                <span className="text-sm font-medium text-slate-900">DATEV Buchungsstapel (CSV)</span>
                <p className="text-xs text-slate-500 mt-0.5">
                  EXTF-Format — direkt importierbar in DATEV, BMD und kompatible Buchhaltungssoftware.
                  Kontonummern konfigurierbar in den{" "}
                  <Link href={`/eegs/${eegId}/settings`} className="text-blue-600 hover:underline">
                    Einstellungen
                  </Link>.
                </p>
              </div>
            </label>
          </div>
        </div>

        {/* Summary preview */}
        <div className="p-3 bg-slate-50 rounded-lg text-sm text-slate-600">
          {mode === "month"
            ? `Export: ${monthNames[month]} ${year} — alle nicht-stornierten Rechnungen und Gutschriften`
            : `Export: Gesamtjahr ${year} — alle nicht-stornierten Rechnungen und Gutschriften`
          }
        </div>

        {/* Export button */}
        <button
          type="button"
          onClick={handleExport}
          disabled={loading}
          className="w-full px-4 py-2.5 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? "Wird exportiert…" : `Als ${format === "datev" ? "DATEV CSV" : "XLSX"} herunterladen`}
        </button>
      </div>

      {/* Info box */}
      <div className="mt-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
        <p className="text-sm font-medium text-blue-900 mb-1">Spalten im Export</p>
        <ul className="text-xs text-blue-800 space-y-0.5 list-disc list-inside">
          <li>Belegdatum, Belegnummer, Belegtyp (Rechnung/Gutschrift)</li>
          <li>Mitglied, UID-Nummer</li>
          <li>Abrechnungszeitraum, Bezug kWh, Einspeisung kWh</li>
          <li>Nettobetrag, MwSt-Satz %, MwSt-Betrag, Bruttobetrag</li>
          <li>Status (Entwurf / Versendet / Bezahlt)</li>
        </ul>
        <p className="text-xs text-blue-700 mt-2">
          Für DATEV: Debitorenkonten werden aus Mitgliedsnummern + Basis-Kontonummer gebildet.
          Erlös- und Aufwandskonten sind in den EEG-Einstellungen konfigurierbar.
        </p>
      </div>
    </div>
  );
}
