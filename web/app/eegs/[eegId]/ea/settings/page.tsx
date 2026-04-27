"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

interface EASettings {
  uva_periodentyp: string;
  steuernummer: string;
  finanzamt: string;
}

export default function EASettingsPage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;

  const [form, setForm] = useState<EASettings>({ uva_periodentyp: "QUARTAL", steuernummer: "", finanzamt: "" });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    fetch(`/api/eegs/${eegId}/ea/settings`)
      .then((r) => r.ok && r.json())
      .then((d) => { if (d) setForm(d); setLoading(false); });
  }, [eegId]);

  async function handleSave() {
    setSaving(true);
    setError("");
    setSuccess(false);
    const res = await fetch(`/api/eegs/${eegId}/ea/settings`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(form),
    });
    if (res.ok) {
      setSuccess(true);
      setTimeout(() => setSuccess(false), 3000);
    } else {
      setError((await res.json()).error || "Fehler");
    }
    setSaving(false);
  }

  const inputClass = "px-3 py-2 border border-slate-300 rounded-lg text-slate-900 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 w-full";

  return (
    <div className="p-8 max-w-2xl">
      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">Übersicht</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea`} className="text-sm text-slate-500 hover:text-slate-700">E/A-Buchhaltung</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Einstellungen</span>
      </div>

      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">E/A-Buchhaltungseinstellungen</h1>
        <p className="text-slate-500 mt-1 text-sm">Steuerdaten und UVA-Periodizität konfigurieren</p>
      </div>

      {loading ? (
        <div className="p-8 text-center text-slate-500 text-sm">Wird geladen…</div>
      ) : (
        <div className="bg-white rounded-xl border border-slate-200 p-6 space-y-5">
          {error && <div className="p-3 bg-red-50 border border-red-200 rounded text-sm text-red-700">{error}</div>}
          {success && <div className="p-3 bg-green-50 border border-green-200 rounded text-sm text-green-700">Einstellungen gespeichert.</div>}

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1.5">UVA-Periodizität</label>
            <select className={inputClass} value={form.uva_periodentyp} onChange={(e) => setForm({ ...form, uva_periodentyp: e.target.value })}>
              <option value="QUARTAL">Quartal (Standard für Umsatz &lt; € 100.000)</option>
              <option value="MONAT">Monat (ab € 100.000 Umsatz Pflicht)</option>
            </select>
            <p className="text-xs text-slate-500 mt-1">
              Kleinunternehmer bis € 100.000 Jahresumsatz: Abgabe der UVA quartalsweise bis zum 15. des auf das Quartal folgenden übernächsten Monats.
              Ab € 100.000: monatliche UVA Pflicht.
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1.5">Steuernummer</label>
            <input className={inputClass} value={form.steuernummer} onChange={(e) => setForm({ ...form, steuernummer: e.target.value })} placeholder="z.B. 12/345/6789" />
            <p className="text-xs text-slate-500 mt-1">Steuernummer des Vereins (für FinanzOnline-Export)</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1.5">Finanzamt</label>
            <input className={inputClass} value={form.finanzamt} onChange={(e) => setForm({ ...form, finanzamt: e.target.value })} placeholder="z.B. FA Wien 1/23" />
            <p className="text-xs text-slate-500 mt-1">Zuständiges Finanzamt für die Steuernummer</p>
          </div>

          <div className="border-t border-slate-200 pt-4">
            <button onClick={handleSave} disabled={saving} className="px-5 py-2.5 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800 disabled:opacity-50">
              {saving ? "Wird gespeichert…" : "Einstellungen speichern"}
            </button>
          </div>
        </div>
      )}

      <div className="mt-6 p-4 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
        <strong>Hinweis zu Reverse Charge (§19 UStG):</strong> Als EEG mit Energieeinspeisung durch umsatzsteuerpflichtige Mitglieder unterliegen die Gutschriften
        an diese Mitglieder dem Übergang der Steuerschuld (Reverse Charge). Die EEG schuldet die USt (KZ 057) und kann — als
        Kleinunternehmer ohne VSt-Abzug — keinen Vorsteuerabzug geltend machen (KZ 060, KZ 066 = 0).
        Konsultieren Sie Ihren Steuerberater für die konkrete steuerliche Beurteilung.
      </div>
    </div>
  );
}
