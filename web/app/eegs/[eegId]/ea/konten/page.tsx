"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

interface EAKonto {
  id: string;
  nummer: string;
  name: string;
  typ: string;
  ust_relevanz: string;
  standard_ust_pct: number;
  uva_kz?: string;
  k1_kz?: string;
  sortierung: number;
  aktiv: boolean;
}

const TYP_LABELS: Record<string, string> = {
  EINNAHME: "Einnahme",
  AUSGABE: "Ausgabe",
  SONSTIG: "Sonstig",
};

const UST_LABELS: Record<string, string> = {
  KEINE: "keine",
  STEUERBAR: "steuerbar",
  VST: "Vorsteuer",
  RC: "Reverse Charge",
};

export default function KontenPage() {
  const params = useParams<{ eegId: string }>();
  const eegId = params.eegId;

  const [konten, setKonten] = useState<EAKonto[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [editKonto, setEditKonto] = useState<EAKonto | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const emptyForm = { nummer: "", name: "", typ: "EINNAHME", ust_relevanz: "KEINE", standard_ust_pct: 0, uva_kz: "", k1_kz: "", sortierung: 100 };
  const [form, setForm] = useState(emptyForm);

  async function load() {
    const res = await fetch(`/api/eegs/${eegId}/ea/konten`);
    if (res.ok) setKonten(await res.json());
    setLoading(false);
  }

  useEffect(() => { load(); }, [eegId]);

  function openCreate() {
    setForm(emptyForm);
    setEditKonto(null);
    setShowForm(true);
    setError("");
  }

  function openEdit(k: EAKonto) {
    setForm({ nummer: k.nummer, name: k.name, typ: k.typ, ust_relevanz: k.ust_relevanz, standard_ust_pct: k.standard_ust_pct, uva_kz: k.uva_kz || "", k1_kz: k.k1_kz || "", sortierung: k.sortierung });
    setEditKonto(k);
    setShowForm(true);
    setError("");
  }

  async function handleSave() {
    setSaving(true);
    setError("");
    try {
      const method = editKonto ? "PUT" : "POST";
      const url = editKonto ? `/api/eegs/${eegId}/ea/konten/${editKonto.id}` : `/api/eegs/${eegId}/ea/konten`;
      const res = await fetch(url, { method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(form) });
      if (!res.ok) { setError((await res.json()).error || "Fehler"); return; }
      setShowForm(false);
      await load();
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Konto wirklich löschen?")) return;
    const res = await fetch(`/api/eegs/${eegId}/ea/konten/${id}`, { method: "DELETE" });
    if (res.ok || res.status === 204) await load();
    else alert((await res.json()).error || "Fehler");
  }

  const inputClass = "px-3 py-2 border border-slate-300 rounded-lg text-slate-900 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 w-full";

  return (
    <div className="p-8">
      <div className="mb-6">
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">Übersicht</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/ea`} className="text-sm text-slate-500 hover:text-slate-700">E/A-Buchhaltung</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Kontenplan</span>
      </div>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Kontenplan</h1>
          <p className="text-slate-500 mt-1 text-sm">Einnahmen- und Ausgabenkonten der E/A-Buchhaltung</p>
        </div>
        <button onClick={openCreate} className="px-4 py-2 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800">
          + Konto anlegen
        </button>
      </div>

      {showForm && (
        <div className="mb-6 bg-white rounded-xl border border-slate-200 p-6">
          <h2 className="font-semibold text-slate-900 mb-4">{editKonto ? "Konto bearbeiten" : "Neues Konto"}</h2>
          {error && <p className="text-sm text-red-600 mb-3">{error}</p>}
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mb-4">
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">Kontonummer</label>
              <input className={inputClass} value={form.nummer} onChange={(e) => setForm({ ...form, nummer: e.target.value })} placeholder="z.B. 4000" />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">Bezeichnung</label>
              <input className={inputClass} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="Kontoname" />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">Typ</label>
              <select className={inputClass} value={form.typ} onChange={(e) => setForm({ ...form, typ: e.target.value })}>
                <option value="EINNAHME">Einnahme</option>
                <option value="AUSGABE">Ausgabe</option>
                <option value="SONSTIG">Sonstig</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">USt-Relevanz</label>
              <select className={inputClass} value={form.ust_relevanz} onChange={(e) => setForm({ ...form, ust_relevanz: e.target.value })}>
                <option value="KEINE">Keine</option>
                <option value="STEUERBAR">Steuerbar (USt)</option>
                <option value="VST">Vorsteuer</option>
                <option value="RC">Reverse Charge</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">Standard-USt %</label>
              <input type="number" className={inputClass} value={form.standard_ust_pct} onChange={(e) => setForm({ ...form, standard_ust_pct: Number(e.target.value) })} step="0.01" min="0" max="100" />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">UVA-Kennzahl</label>
              <input className={inputClass} value={form.uva_kz} onChange={(e) => setForm({ ...form, uva_kz: e.target.value })} placeholder="z.B. 022" />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">K1-Kennzahl (KöSt)</label>
              <input className={inputClass} value={form.k1_kz} onChange={(e) => setForm({ ...form, k1_kz: e.target.value })} placeholder="z.B. 9040" />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">Sortierung</label>
              <input type="number" className={inputClass} value={form.sortierung} onChange={(e) => setForm({ ...form, sortierung: Number(e.target.value) })} />
            </div>
          </div>
          <div className="flex gap-2">
            <button onClick={handleSave} disabled={saving} className="px-4 py-2 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800 disabled:opacity-50">
              {saving ? "Speichern…" : "Speichern"}
            </button>
            <button onClick={() => setShowForm(false)} className="px-4 py-2 text-sm text-slate-600 border border-slate-300 rounded-lg hover:bg-slate-50">
              Abbrechen
            </button>
          </div>
        </div>
      )}

      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-slate-500 text-sm">Wird geladen…</div>
        ) : konten.length === 0 ? (
          <div className="p-8 text-center text-slate-500 text-sm">Keine Konten vorhanden. Standard-Konten werden beim ersten Aufruf automatisch angelegt.</div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-200 bg-slate-50">
                <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase tracking-wide">Nr.</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase tracking-wide">Bezeichnung</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase tracking-wide">Typ</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase tracking-wide">USt-Relevanz</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase tracking-wide">USt %</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase tracking-wide">UVA-KZ</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase tracking-wide">K1-KZ</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-slate-500 uppercase tracking-wide"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {konten.map((k) => (
                <tr key={k.id} className={`hover:bg-slate-50 ${!k.aktiv ? "opacity-50" : ""}`}>
                  <td className="px-4 py-3 font-mono text-slate-700">{k.nummer}</td>
                  <td className="px-4 py-3 text-slate-900">{k.name}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${k.typ === "EINNAHME" ? "bg-green-50 text-green-700" : k.typ === "AUSGABE" ? "bg-red-50 text-red-700" : "bg-slate-100 text-slate-600"}`}>
                      {TYP_LABELS[k.typ] || k.typ}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-slate-600">{UST_LABELS[k.ust_relevanz] || k.ust_relevanz}</td>
                  <td className="px-4 py-3 text-slate-600">{k.standard_ust_pct > 0 ? `${k.standard_ust_pct} %` : "—"}</td>
                  <td className="px-4 py-3 text-slate-500 font-mono text-xs">{k.uva_kz || "—"}</td>
                  <td className="px-4 py-3 text-slate-500 font-mono text-xs">{k.k1_kz || "—"}</td>
                  <td className="px-4 py-3 flex gap-2 justify-end">
                    <button onClick={() => openEdit(k)} className="text-xs text-blue-600 hover:underline">Bearbeiten</button>
                    <button onClick={() => handleDelete(k.id)} className="text-xs text-red-600 hover:underline">Löschen</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
