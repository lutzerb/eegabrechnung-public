"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";

interface EAKonto {
  id: string;
  nummer: string;
  name: string;
  typ: string;
  ust_relevanz: string;
  standard_ust_pct: number;
}

const UST_CODE_OPTIONS = [
  { value: "KEINE", label: "Keine (kein Umsatz/Vorsteuer)" },
  { value: "UST_20", label: "USt 20 % (brutto inkl. USt)" },
  { value: "VST_20", label: "Vorsteuer 20 % (brutto inkl. VSt)" },
  { value: "RC_20", label: "Reverse Charge 20 % (netto = brutto)" },
  { value: "RC_13", label: "Reverse Charge 13 % LuF (netto = brutto)" },
];

export default function NeueBuchungPage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;
  const router = useRouter();

  const [konten, setKonten] = useState<EAKonto[]>([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [belegFile, setBelegFile] = useState<File | null>(null);

  const today = new Date().toISOString().split("T")[0];

  const [form, setForm] = useState({
    beleg_datum: today,
    zahlung_datum: today,
    zahlung_offen: false,
    konto_id: "",
    beschreibung: "",
    betrag_brutto: "",
    ust_code: "KEINE",
    richtung: "EINNAHME",
    referenz: "",
  });

  useEffect(() => {
    fetch(`/api/eegs/${eegId}/ea/konten`).then((r) => r.ok && r.json()).then((d) => {
      if (d) {
        setKonten(d);
        if (d.length > 0) setForm((f) => ({ ...f, konto_id: d[0].id }));
      }
    });
  }, [eegId]);

  // Auto-detect ust_code from konto when konto changes
  function handleKontoChange(kontoId: string) {
    const k = konten.find((k) => k.id === kontoId);
    let ust_code = form.ust_code;
    if (k) {
      if (k.ust_relevanz === "KEINE") ust_code = "KEINE";
      else if (k.ust_relevanz === "STEUERBAR") ust_code = "UST_20";
      else if (k.ust_relevanz === "VST") ust_code = "VST_20";
      else if (k.ust_relevanz === "RC") {
        ust_code = k.standard_ust_pct === 13 ? "RC_13" : "RC_20";
      }
      const richtung = k.typ === "AUSGABE" ? "AUSGABE" : "EINNAHME";
      setForm((f) => ({ ...f, konto_id: kontoId, ust_code, richtung }));
    } else {
      setForm((f) => ({ ...f, konto_id: kontoId }));
    }
  }

  // Preview USt calculation
  const brutto = parseFloat(form.betrag_brutto) || 0;
  let netto = brutto, ust = 0;
  if (form.ust_code === "RC_20" || form.ust_code === "RC_13") {
    netto = brutto;
    ust = brutto * (form.ust_code === "RC_13" ? 0.13 : 0.20);
  } else if (form.ust_code === "UST_20" || form.ust_code === "VST_20") {
    netto = brutto / 1.20;
    ust = brutto - netto;
  }

  async function handleSave() {
    setSaving(true);
    setError("");
    try {
      const body = {
        beleg_datum: form.beleg_datum,
        zahlung_datum: form.zahlung_offen ? null : form.zahlung_datum,
        konto_id: form.konto_id,
        beschreibung: form.beschreibung,
        betrag_brutto: parseFloat(form.betrag_brutto),
        ust_code: form.ust_code,
        richtung: form.richtung,
        referenz: form.referenz || null,
      };
      const res = await fetch(`/api/eegs/${eegId}/ea/buchungen`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) { setError((await res.json()).error || "Fehler"); return; }
      const created = await res.json();
      if (belegFile) {
        const fd = new FormData();
        fd.append("datei", belegFile);
        fd.append("buchung_id", created.id);
        await fetch(`/api/eegs/${eegId}/ea/belege`, { method: "POST", body: fd });
      }
      router.push(`/eegs/${eegId}/ea/buchungen`);
    } finally {
      setSaving(false);
    }
  }

  const inputClass = "px-3 py-2 border border-slate-300 rounded-lg text-slate-900 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 w-full";

  return (
    <div className="p-8 max-w-2xl">
      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">Übersicht</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea`} className="text-sm text-slate-500 hover:text-slate-700">E/A-Buchhaltung</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea/buchungen`} className="text-sm text-slate-500 hover:text-slate-700">Journal</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Neue Buchung</span>
      </div>

      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Buchung erfassen</h1>
        <p className="text-slate-500 mt-1 text-sm">Einnahme oder Ausgabe manuell buchen</p>
      </div>

      <div className="bg-white rounded-xl border border-slate-200 p-6 space-y-5">
        {error && <div className="p-3 bg-red-50 border border-red-200 rounded text-sm text-red-700">{error}</div>}

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-medium text-slate-700 mb-1.5">Buchungsdatum</label>
            <input type="date" className={inputClass} value={form.beleg_datum} onChange={(e) => setForm({ ...form, beleg_datum: e.target.value })} />
          </div>
          <div>
            <label className="block text-xs font-medium text-slate-700 mb-1.5">Richtung</label>
            <select className={inputClass} value={form.richtung} onChange={(e) => setForm({ ...form, richtung: e.target.value })}>
              <option value="EINNAHME">Einnahme</option>
              <option value="AUSGABE">Ausgabe</option>
            </select>
          </div>
        </div>

        <div>
          <label className="block text-xs font-medium text-slate-700 mb-1.5">Konto</label>
          <select className={inputClass} value={form.konto_id} onChange={(e) => handleKontoChange(e.target.value)}>
            <option value="">Bitte wählen…</option>
            {konten.map((k) => <option key={k.id} value={k.id}>{k.nummer} – {k.name}</option>)}
          </select>
        </div>

        <div>
          <label className="block text-xs font-medium text-slate-700 mb-1.5">Beschreibung / Belegtext</label>
          <input className={inputClass} value={form.beschreibung} onChange={(e) => setForm({ ...form, beschreibung: e.target.value })} placeholder="z.B. Mitgliedsbeitrag Mai 2025" />
        </div>

        <div>
          <label className="block text-xs font-medium text-slate-700 mb-1.5">Referenz (optional)</label>
          <input className={inputClass} value={form.referenz} onChange={(e) => setForm({ ...form, referenz: e.target.value })} placeholder="Rechnungsnummer, Vertragsreferenz…" />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-medium text-slate-700 mb-1.5">Bruttobetrag (€)</label>
            <input type="number" className={inputClass} value={form.betrag_brutto} onChange={(e) => setForm({ ...form, betrag_brutto: e.target.value })} step="0.01" min="0" placeholder="0.00" />
          </div>
          <div>
            <label className="block text-xs font-medium text-slate-700 mb-1.5">USt-Code</label>
            <select className={inputClass} value={form.ust_code} onChange={(e) => setForm({ ...form, ust_code: e.target.value })}>
              {UST_CODE_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
            </select>
          </div>
        </div>

        {/* USt preview */}
        {brutto > 0 && (
          <div className="p-3 bg-slate-50 rounded-lg text-sm text-slate-600 grid grid-cols-3 gap-2">
            <div><span className="block text-xs text-slate-500">Nettobetrag</span><span className="font-medium">{new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(netto)}</span></div>
            <div><span className="block text-xs text-slate-500">USt-Betrag</span><span className="font-medium">{new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(ust)}</span></div>
            <div><span className="block text-xs text-slate-500">Bruttobetrag</span><span className="font-medium">{new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(brutto)}</span></div>
          </div>
        )}

        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className="block text-xs font-medium text-slate-700">Zahlungsdatum</label>
            <label className="flex items-center gap-1.5 text-xs text-slate-600 cursor-pointer">
              <input type="checkbox" checked={form.zahlung_offen} onChange={(e) => setForm({ ...form, zahlung_offen: e.target.checked })} className="rounded border-slate-300" />
              Zahlung noch ausstehend
            </label>
          </div>
          {!form.zahlung_offen && (
            <input type="date" className={inputClass} value={form.zahlung_datum} onChange={(e) => setForm({ ...form, zahlung_datum: e.target.value })} />
          )}
        </div>

        <div>
          <label className="block text-xs font-medium text-slate-700 mb-1.5">Beleg (optional)</label>
          <label className="flex items-center gap-2 cursor-pointer w-fit">
            <span className="px-3 py-2 text-sm border border-slate-300 rounded-lg text-slate-600 hover:bg-slate-50">
              {belegFile ? belegFile.name : "Datei wählen…"}
            </span>
            <input type="file" accept=".pdf,.jpg,.jpeg,.png" className="hidden" onChange={(e) => setBelegFile(e.target.files?.[0] ?? null)} />
            {belegFile && (
              <button type="button" onClick={() => setBelegFile(null)} className="text-xs text-slate-400 hover:text-red-600">Entfernen</button>
            )}
          </label>
        </div>

        <div className="flex gap-2 pt-2">
          <button onClick={handleSave} disabled={saving || !form.konto_id || !form.betrag_brutto || !form.beschreibung} className="px-5 py-2.5 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800 disabled:opacity-50 disabled:cursor-not-allowed">
            {saving ? "Wird gespeichert…" : "Buchung speichern"}
          </button>
          <Link href={`/eegs/${eegId}/ea/buchungen`} className="px-4 py-2.5 text-sm text-slate-600 border border-slate-300 rounded-lg hover:bg-slate-50">
            Abbrechen
          </Link>
        </div>
      </div>
    </div>
  );
}
