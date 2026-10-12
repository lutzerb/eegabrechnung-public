"use client";

import { useState } from "react";

interface Props {
  initialSubject: string;
  initialBody: string;
}

const DEFAULT_SUBJECT = "Ihre {{dokumenttyp}} – {{eeg_name}} – {{zeitraum}}";

const DEFAULT_BODY = `Sehr geehrte/r {{name}},

anbei erhalten Sie Ihre {{dokumenttyp}} für den Abrechnungszeitraum
{{zeitraum}}.

Energiegemeinschaft:  {{eeg_name}}
Mitgliedsnummer:      {{mitglieds_nr}}
{{dokumenttyp}}nummer:       {{rechnungsnummer}}
{{bezug_zeile}}{{einspeisung_zeile}}{{gesamtbetrag_zeile}}

{{zahlungshinweis}}

Die Details entnehmen Sie bitte dem beigefügten PDF.

Im Mitgliederportal ({{portal_link}}) können Sie jederzeit Ihre Energiedaten und bisherigen Rechnungen einsehen.

Mit freundlichen Grüßen
{{eeg_name}}

--
Erstellt von eegabrechnung`;

const PLACEHOLDERS = [
  "{{name}}", "{{vorname}}", "{{nachname}}", "{{mitglieds_nr}}", "{{eeg_name}}",
  "{{dokumenttyp}}", "{{rechnungsnummer}}", "{{zeitraum}}", "{{zeitraum_von}}", "{{zeitraum_bis}}",
  "{{betrag}}", "{{bezug_kwh}}", "{{einspeisung_kwh}}", "{{bezug_zeile}}", "{{einspeisung_zeile}}",
  "{{gesamtbetrag_zeile}}", "{{zahlungshinweis}}", "{{portal_link}}",
];

const inputClass =
  "w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent";
const labelClass = "block text-sm font-medium text-slate-700 mb-1.5";
const insertButtonClass =
  "px-2.5 py-1 text-xs border border-slate-300 text-slate-700 font-medium rounded-lg hover:bg-slate-100 transition-colors whitespace-nowrap";

export default function InvoiceEmailTextFields({ initialSubject, initialBody }: Props) {
  const [subject, setSubject] = useState(initialSubject || "");
  const [body, setBody] = useState(initialBody || "");

  return (
    <div className="bg-white rounded-xl border border-slate-200 p-6">
      <h2 className="text-base font-semibold text-slate-900 mb-1">Rechnungsversand-E-Mail</h2>
      <p className="text-xs text-slate-500 mb-4">
        Betreff und Text der E-Mail, mit der Rechnungen/Gutschriften an Mitglieder verschickt werden (Massenversand
        im Abrechnungslauf und „Erneut senden“). Beide Felder leer lassen = Standardtext wird verwendet.
      </p>
      <div className="space-y-4">
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className={labelClass + " mb-0"}>Betreff</label>
            <button type="button" onClick={() => setSubject(DEFAULT_SUBJECT)} className={insertButtonClass}>
              Standardtext einfügen
            </button>
          </div>
          <input
            type="text"
            name="invoice_email_subject"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            placeholder={DEFAULT_SUBJECT}
            className={inputClass}
          />
        </div>
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className={labelClass + " mb-0"}>Text</label>
            <button type="button" onClick={() => setBody(DEFAULT_BODY)} className={insertButtonClass}>
              Standardtext einfügen
            </button>
          </div>
          <textarea
            name="invoice_email_body"
            rows={18}
            value={body}
            onChange={(e) => setBody(e.target.value)}
            placeholder="Sehr geehrte/r {{name}}, ..."
            className={inputClass + " font-mono text-sm"}
          />
        </div>
        <div className="bg-slate-50 border border-slate-200 rounded-lg p-4 text-xs text-slate-600">
          <p className="font-medium text-slate-700 mb-2">Verfügbare Platzhalter</p>
          <div className="grid grid-cols-2 gap-x-4 gap-y-1 font-mono">
            {PLACEHOLDERS.map((p) => (
              <span key={p}>{p}</span>
            ))}
          </div>
          <p className="mt-2 text-slate-500">
            {"{{bezug_zeile}}"} / {"{{einspeisung_zeile}}"} sind bereits fertige Zeilen inkl. Beschriftung und
            Zeilenumbruch — sie verschwinden komplett, wenn die jeweilige Menge 0 ist (wie im Standardtext
            verwendet). {"{{bezug_kwh}}"} / {"{{einspeisung_kwh}}"} liefern nur die reine Zahl (leer bei 0) für eigene
            Sätze. {"{{gesamtbetrag_zeile}}"} ist ebenfalls eine fertige Zeile inkl. Beschriftung und zeigt wie im
            PDF an, ob die EEG den Betrag an das Mitglied überweist ("EEG zahlt an Sie") oder umgekehrt.
          </p>
        </div>
      </div>
    </div>
  );
}
