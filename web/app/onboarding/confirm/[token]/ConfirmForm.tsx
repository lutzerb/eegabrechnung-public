"use client";

import { useState } from "react";

interface MeterPoint {
  zaehlpunkt: string;
  direction: string;
}

interface OnboardingRequest {
  name1: string;
  name2: string;
  email: string;
  phone: string;
  strasse: string;
  plz: string;
  ort: string;
  iban: string;
  bic: string;
  member_type: string;
  meter_points: MeterPoint[];
  beitritts_datum?: string;
}

interface EEGInfo {
  name: string;
  onboarding_contract_text: string;
  documents: Array<{ id: string; title: string }>;
}

interface Props {
  token: string;
  req: OnboardingRequest;
  eeg: EEGInfo;
  eegId: string;
}

function formatIBAN(iban: string): string {
  return iban.replace(/(.{4})/g, "$1 ").trim();
}

function formatDate(dateStr?: string): string {
  if (!dateStr) return "—";
  try {
    return new Date(dateStr).toLocaleDateString("de-AT", {
      day: "2-digit", month: "2-digit", year: "numeric",
    });
  } catch { return dateStr; }
}

export default function ConfirmForm({ token, req, eeg, eegId }: Props) {
  const [dataOk, setDataOk] = useState(false);
  const [agbOk, setAgbOk] = useState(false);
  const [sepaOk, setSepaOk] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirmed, setConfirmed] = useState(false);

  const hasIban = !!req.iban;
  const canSubmit = dataOk && agbOk && (!hasIban || sepaOk);

  const memberTypeLabel: Record<string, string> = {
    CONSUMER: "Verbraucher (Bezug)",
    PRODUCER: "Erzeuger (Einspeisung)",
    PROSUMER: "Prosumer (Bezug & Einspeisung)",
  };

  async function handleConfirm() {
    setError(null);
    setLoading(true);
    try {
      const res = await fetch(`/api/public/onboarding/confirm/${token}`, {
        method: "POST",
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error || "Fehler bei der Bestätigung.");
        return;
      }
      setConfirmed(true);
    } catch {
      setError("Netzwerkfehler. Bitte versuchen Sie es erneut.");
    } finally {
      setLoading(false);
    }
  }

  if (confirmed) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-12">
        <div className="bg-white rounded-2xl border border-green-200 shadow-sm p-8 text-center">
          <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h2 className="text-2xl font-bold text-slate-900 mb-2">Anmeldung bestätigt</h2>
          <p className="text-slate-600">
            Vielen Dank, {req.name1}! Ihre Anmeldung zur Energiegemeinschaft{" "}
            <strong>{eeg.name}</strong> wurde erfolgreich bestätigt.
          </p>
          <p className="text-slate-500 text-sm mt-3">
            Die Verwaltung wurde benachrichtigt und wird Ihre Anmeldung in Kürze abschließen.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto px-4 py-8 space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Anmeldung bestätigen</h1>
        <p className="text-slate-500 mt-1">
          Energiegemeinschaft <strong>{eeg.name}</strong>
        </p>
      </div>

      <div className="bg-blue-50 border border-blue-200 rounded-xl p-4 text-sm text-blue-800">
        Der Administrator hat Ihre Anmeldung eingetragen. Bitte prüfen Sie Ihre Daten
        und bestätigen Sie am Ende dieser Seite.
      </div>

      {/* Datenzusammenfassung */}
      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        <div className="px-5 py-4 bg-slate-50 border-b border-slate-200">
          <h2 className="font-semibold text-slate-800">Ihre eingetragenen Daten</h2>
        </div>
        <table className="w-full text-sm">
          <tbody className="divide-y divide-slate-100">
            <tr>
              <td className="px-5 py-3 text-slate-500 w-36">Name</td>
              <td className="px-5 py-3 font-medium text-slate-900">
                {[req.name1, req.name2].filter(Boolean).join(" ")}
              </td>
            </tr>
            <tr>
              <td className="px-5 py-3 text-slate-500">Adresse</td>
              <td className="px-5 py-3 text-slate-900">
                {req.strasse}<br />{req.plz} {req.ort}
              </td>
            </tr>
            <tr>
              <td className="px-5 py-3 text-slate-500">E-Mail</td>
              <td className="px-5 py-3 text-slate-900">{req.email}</td>
            </tr>
            {req.phone && (
              <tr>
                <td className="px-5 py-3 text-slate-500">Telefon</td>
                <td className="px-5 py-3 text-slate-900">{req.phone}</td>
              </tr>
            )}
            {req.iban && (
              <tr>
                <td className="px-5 py-3 text-slate-500">IBAN</td>
                <td className="px-5 py-3 font-mono text-slate-900">{formatIBAN(req.iban)}</td>
              </tr>
            )}
            <tr>
              <td className="px-5 py-3 text-slate-500">Typ</td>
              <td className="px-5 py-3 text-slate-900">
                {memberTypeLabel[req.member_type] || req.member_type}
              </td>
            </tr>
            <tr>
              <td className="px-5 py-3 text-slate-500 align-top">Zählpunkt(e)</td>
              <td className="px-5 py-3">
                <div className="space-y-1">
                  {req.meter_points.map((mp, i) => (
                    <div key={i} className="flex items-center gap-2">
                      <span className="font-mono text-slate-900 text-xs">{mp.zaehlpunkt}</span>
                      <span className="text-xs text-slate-500">
                        ({mp.direction === "GENERATION" ? "Einspeisung" : "Bezug"})
                      </span>
                    </div>
                  ))}
                </div>
              </td>
            </tr>
            {req.beitritts_datum && (
              <tr>
                <td className="px-5 py-3 text-slate-500">Beitritt</td>
                <td className="px-5 py-3 text-slate-900">{formatDate(req.beitritts_datum)}</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* AGB / Vertragstext */}
      {eeg.onboarding_contract_text && (
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          <div className="px-5 py-4 bg-slate-50 border-b border-slate-200">
            <h2 className="font-semibold text-slate-800">Einzugsermächtigung / SEPA-Lastschrift</h2>
          </div>
          <div className="px-5 py-4 text-sm text-slate-600 leading-relaxed whitespace-pre-wrap">
            {eeg.onboarding_contract_text}
          </div>
        </div>
      )}

      {/* Dokumente */}
      {eeg.documents && eeg.documents.length > 0 && (
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          <div className="px-5 py-4 bg-slate-50 border-b border-slate-200">
            <h2 className="font-semibold text-slate-800">Dokumente</h2>
          </div>
          <ul className="px-5 py-4 space-y-2">
            {eeg.documents.map((doc) => (
              <li key={doc.id}>
                <a
                  href={`/api/public/eegs/${eegId}/documents/${doc.id}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-600 hover:underline text-sm flex items-center gap-1"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                  {doc.title}
                </a>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Bestätigungs-Checkboxen */}
      <div className="bg-white rounded-xl border border-slate-200 p-5 space-y-4">
        <h2 className="font-semibold text-slate-800">Bestätigung</h2>

        {error && (
          <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
            {error}
          </div>
        )}

        <label className="flex items-start gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={dataOk}
            onChange={(e) => setDataOk(e.target.checked)}
            className="mt-0.5 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
          />
          <span className="text-sm text-slate-700">
            Die oben angezeigten Daten sind korrekt und vollständig.
          </span>
        </label>

        <label className="flex items-start gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={agbOk}
            onChange={(e) => setAgbOk(e.target.checked)}
            className="mt-0.5 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
          />
          <span className="text-sm text-slate-700">
            Ich stimme den Teilnahmebedingungen der Energiegemeinschaft{" "}
            <strong>{eeg.name}</strong> zu
            {eeg.documents && eeg.documents.length > 0
              ? " (Dokumente oben abrufbar)"
              : ""}.
          </span>
        </label>

        {hasIban && (
          <label className="flex items-start gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={sepaOk}
              onChange={(e) => setSepaOk(e.target.checked)}
              className="mt-0.5 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="text-sm text-slate-700">
              Ich erteile die SEPA-Lastschriftermächtigung für mein Konto (IBAN:{" "}
              <span className="font-mono">{formatIBAN(req.iban)}</span>).
            </span>
          </label>
        )}

        <button
          type="button"
          onClick={handleConfirm}
          disabled={!canSubmit || loading}
          className="w-full py-3 bg-green-600 text-white rounded-lg font-semibold text-sm hover:bg-green-700 transition-colors disabled:opacity-40 disabled:cursor-not-allowed mt-2"
        >
          {loading ? "Wird bestätigt..." : "Anmeldung verbindlich bestätigen"}
        </button>
      </div>
    </div>
  );
}
