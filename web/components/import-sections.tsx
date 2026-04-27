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

      <FileUploadSection
        eegId={eegId}
        type="stammdaten"
        title="Stammdaten importieren"
        description="Mitgliederdaten und Zählpunkte aus XLSX-Datei importieren."
        acceptedFormats=".xlsx, .xls"
      />

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
