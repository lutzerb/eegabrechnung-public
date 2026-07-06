"use client";

import { useState, useRef } from "react";
import DataCoverage from "@/components/data-coverage";
import FileUploadSection from "@/components/file-upload";

export default function ImportSections({ eegId }: { eegId: string }) {
  const [coverageKey, setCoverageKey] = useState(0);
  const [importSuccess, setImportSuccess] = useState(false);
  const coverageRef = useRef<HTMLDivElement>(null);

  function handleEnergyImportComplete() {
    setCoverageKey((k) => k + 1);
    setImportSuccess(true);
    setTimeout(() => {
      coverageRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
    }, 300);
  }

  return (
    <div className="space-y-6">
      <div ref={coverageRef}>
        <DataCoverage eegId={eegId} refreshKey={coverageKey} />
      </div>

      {importSuccess && (
        <div className="p-4 bg-green-50 border border-green-200 rounded-lg flex items-center justify-between">
          <div>
            <p className="font-medium text-green-800">Import erfolgreich</p>
            <p className="text-sm text-green-700 mt-0.5">Die Datenverfügbarkeit wurde aktualisiert.</p>
          </div>
          <button
            onClick={() => coverageRef.current?.scrollIntoView({ behavior: "smooth", block: "start" })}
            className="px-3 py-1.5 text-sm font-medium text-green-700 bg-green-100 border border-green-300 rounded-lg hover:bg-green-200 transition-colors"
          >
            Zur Datenverfügbarkeit
          </button>
        </div>
      )}

      <div className="bg-white rounded-xl border border-slate-200 p-6">
        <div className="flex items-start gap-3 mb-5">
          <div className="w-10 h-10 rounded-lg bg-blue-50 flex items-center justify-center flex-shrink-0">
            <svg className="w-5 h-5 text-blue-700" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </div>
          <div>
            <h3 className="font-semibold text-slate-900">Stammdaten importieren</h3>
            <p className="text-sm text-slate-500 mt-0.5">Mitglieder und Zählpunkte aus einer XLSX-Datei importieren. Zwei Quellen werden unterstützt:</p>
          </div>
        </div>

        <div className="grid sm:grid-cols-2 gap-4">
          <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 flex flex-col gap-3">
            <div>
              <span className="inline-block text-xs font-semibold text-blue-800 bg-blue-100 border border-blue-200 rounded px-2 py-0.5 mb-1.5">eegabrechnung-Export</span>
              <p className="text-sm text-slate-700">
                Datei, die direkt aus eegabrechnung exportiert wurde — z.&nbsp;B. zur Übertragung in eine andere EEG oder nach einer Löschung.
              </p>
              <a
                href={`/api/eegs/${eegId}/import/stammdaten/template`}
                download
                className="inline-flex items-center gap-1 mt-2 text-xs font-medium text-blue-700 hover:text-blue-900"
              >
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                    d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                </svg>
                Vorlage mit Beispieldaten herunterladen
              </a>
            </div>
            <FileUploadSection
              eegId={eegId}
              type="stammdaten"
              title=""
              description=""
              acceptedFormats=".xlsx, .xls"
              compact
            />
          </div>

          <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 flex flex-col gap-3">
            <div>
              <span className="inline-block text-xs font-semibold text-slate-600 bg-slate-200 border border-slate-300 rounded px-2 py-0.5 mb-1.5">EDA-Portal (Netzbetreiber)</span>
              <p className="text-sm text-slate-700">
                Stammdaten-Export aus dem EDA-Portal des Netzbetreibers — Tabellenblatt muss <strong>„EEG Stammdaten"</strong> heißen.
              </p>
            </div>
            <FileUploadSection
              eegId={eegId}
              type="stammdaten"
              title=""
              description=""
              acceptedFormats=".xlsx, .xls"
              compact
            />
          </div>
        </div>
      </div>

      <FileUploadSection
        eegId={eegId}
        type="energiedaten"
        title="Energiedaten importieren"
        description="Verbrauchs- und Erzeugungsdaten aus XLSX-Datei importieren. Die Datei wird vor dem Import analysiert — bei Überschneidungen können Sie wählen, ob bestehende Daten überschrieben oder übersprungen werden."
        acceptedFormats=".xlsx, .xls"
        onImportComplete={handleEnergyImportComplete}
      />
    </div>
  );
}
