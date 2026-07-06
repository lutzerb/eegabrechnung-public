"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";

interface EABeleg {
  id: string;
  dateiname: string;
  content_type: string;
  uploaded_at: string;
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
  gegenseite?: string;
  notizen?: string;
  quelle: string;
  deleted_at?: string;
  belege?: EABeleg[];
}

interface ChangelogEntry {
  id: string;
  buchung_id: string;
  operation: string;
  changed_at: string;
  changed_by: string;
  old_values?: Record<string, unknown>;
  new_values?: Record<string, unknown>;
  reason?: string;
}

function fmt(n: number): string {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(n);
}

function fmtDate(s?: string): string {
  if (!s) return "—";
  try { return new Date(s).toLocaleDateString("de-AT", { day: "2-digit", month: "2-digit", year: "numeric" }); } catch { return s; }
}

function fmtDateTime(s?: string): string {
  if (!s) return "—";
  try {
    return new Date(s).toLocaleString("de-AT", {
      day: "2-digit", month: "2-digit", year: "numeric",
      hour: "2-digit", minute: "2-digit",
    });
  } catch { return s; }
}

const QUELLE_LABELS: Record<string, string> = {
  manual: "Manuell",
  eeg_rechnung: "EEG-Rechnung",
  eeg_gutschrift: "EEG-Gutschrift",
  bankimport: "Bankimport",
};

const OP_LABELS: Record<string, string> = {
  create: "Erstellt",
  update: "Geändert",
  delete: "Gelöscht",
};

const OP_COLORS: Record<string, string> = {
  create: "bg-green-50 text-green-700",
  update: "bg-blue-50 text-blue-700",
  delete: "bg-red-50 text-red-700",
};

export default function BuchungDetailPage() {
  const params = useParams<{ eegId: string; buchungId: string }>();
  const { eegId, buchungId } = params;
  const router = useRouter();

  const [buchung, setBuchung] = useState<EABuchung | null>(null);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [zahlungDatum, setZahlungDatum] = useState("");
  const [savingZahlung, setSavingZahlung] = useState(false);

  const [changelog, setChangelog] = useState<ChangelogEntry[]>([]);
  const [changelogLoading, setChangelogLoading] = useState(false);

  const [deleteReason, setDeleteReason] = useState("");
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleting, setDeleting] = useState(false);

  async function load() {
    const res = await fetch(`/api/eegs/${eegId}/ea/buchungen/${buchungId}`);
    if (res.ok) {
      const data = await res.json();
      setBuchung(data);
      setZahlungDatum(data.zahlung_datum ? data.zahlung_datum.split("T")[0] : "");
    }
    setLoading(false);
  }

  async function loadChangelog() {
    setChangelogLoading(true);
    const res = await fetch(`/api/eegs/${eegId}/ea/buchungen/${buchungId}/changelog`);
    if (res.ok) setChangelog(await res.json());
    setChangelogLoading(false);
  }

  useEffect(() => { load(); loadChangelog(); }, [buchungId]);

  async function handleDelete() {
    if (!deleteReason.trim()) return;
    setDeleting(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/ea/buchungen/${buchungId}`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reason: deleteReason }),
      });
      if (res.ok || res.status === 204) router.push(`/eegs/${eegId}/ea/buchungen`);
      else alert("Fehler beim Löschen");
    } finally {
      setDeleting(false);
    }
  }

  function buchungPutBody(overrides: Record<string, unknown>) {
    if (!buchung) return {};
    return {
      beschreibung: buchung.beschreibung,
      beleg_datum: buchung.beleg_datum ? buchung.beleg_datum.split("T")[0] : "",
      zahlung_datum: buchung.zahlung_datum ? buchung.zahlung_datum.split("T")[0] : "",
      konto_id: buchung.konto_id,
      betrag_brutto: buchung.betrag_brutto,
      ust_code: buchung.ust_code,
      gegenseite: buchung.gegenseite ?? "",
      notizen: buchung.notizen ?? "",
      ...overrides,
    };
  }

  async function handleZahlungSave() {
    setSavingZahlung(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/ea/buchungen/${buchungId}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(buchungPutBody({ zahlung_datum: zahlungDatum })),
      });
      if (res.ok) { await load(); await loadChangelog(); }
      else alert("Fehler beim Speichern");
    } finally {
      setSavingZahlung(false);
    }
  }

  async function handleZahlungClear() {
    setSavingZahlung(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/ea/buchungen/${buchungId}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(buchungPutBody({ zahlung_datum: "" })),
      });
      if (res.ok) { setZahlungDatum(""); await load(); await loadChangelog(); }
      else alert("Fehler beim Löschen");
    } finally {
      setSavingZahlung(false);
    }
  }

  async function handleBelegUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    try {
      const fd = new FormData();
      fd.append("datei", file);
      fd.append("buchung_id", buchungId);
      const res = await fetch(`/api/eegs/${eegId}/ea/belege`, { method: "POST", body: fd });
      if (res.ok) await load();
      else alert("Upload fehlgeschlagen");
    } finally {
      setUploading(false);
      e.target.value = "";
    }
  }

  async function handleBelegDelete(belegId: string) {
    if (!confirm("Beleg löschen?")) return;
    const res = await fetch(`/api/eegs/${eegId}/ea/belege/${belegId}`, { method: "DELETE" });
    if (res.ok || res.status === 204) await load();
  }

  if (loading) return <div className="p-8 text-slate-500 text-sm">Wird geladen…</div>;
  if (!buchung) return <div className="p-8 text-red-600 text-sm">Buchung nicht gefunden.</div>;

  const isDeleted = !!buchung.deleted_at;

  return (
    <div className="p-8 max-w-2xl">
      {/* Delete modal */}
      {showDeleteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold text-slate-900 mb-1">Buchung löschen</h2>
            <p className="text-sm text-slate-500 mb-4">
              Die Buchung wird als gelöscht markiert und bleibt im Audit-Trail sichtbar (BAO §131).
            </p>
            <label className="block text-xs font-medium text-slate-700 mb-1">
              Grund <span className="text-red-500">*</span>
            </label>
            <textarea
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
              rows={3}
              placeholder="z.B. Doppelbuchung, Falscheingabe, storniert …"
              value={deleteReason}
              onChange={(e) => setDeleteReason(e.target.value)}
            />
            <div className="flex gap-3 mt-4 justify-end">
              <button onClick={() => setShowDeleteModal(false)} disabled={deleting} className="px-4 py-2 text-sm text-slate-700 border border-slate-300 rounded-lg hover:bg-slate-50 disabled:opacity-50">
                Abbrechen
              </button>
              <button
                onClick={handleDelete}
                disabled={deleting || !deleteReason.trim()}
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
        <Link href={`/eegs/${eegId}/ea/buchungen`} className="text-sm text-slate-500 hover:text-slate-700">Journal</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">{buchung.buchungsnr || buchung.id.slice(0, 8)}</span>
      </div>

      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">{buchung.buchungsnr || "Buchung"}</h1>
          <p className="text-slate-500 mt-1 text-sm">{buchung.beschreibung}</p>
          {isDeleted && (
            <span className="inline-flex mt-2 px-2 py-0.5 rounded bg-red-50 text-red-700 text-xs font-medium">
              Gelöscht am {fmtDate(buchung.deleted_at)}
            </span>
          )}
        </div>
        {!isDeleted && buchung.quelle === "manual" && (
          <button onClick={() => setShowDeleteModal(true)} className="px-3 py-2 text-sm text-red-600 border border-red-200 rounded-lg hover:bg-red-50">
            Löschen
          </button>
        )}
      </div>

      <div className="bg-white rounded-xl border border-slate-200 p-6 mb-4">
        <dl className="grid grid-cols-2 gap-x-6 gap-y-4 text-sm">
          <div>
            <dt className="text-xs text-slate-500 font-medium uppercase">Belegdatum</dt>
            <dd className="mt-0.5 text-slate-900">{fmtDate(buchung.beleg_datum)}</dd>
          </div>
          <div>
            <dt className="text-xs text-slate-500 font-medium uppercase">Zahlungsdatum</dt>
            <dd className="mt-0.5 text-slate-900">
              {buchung.zahlung_datum ? fmtDate(buchung.zahlung_datum) : <span className="text-amber-600">ausstehend</span>}
            </dd>
          </div>
          <div>
            <dt className="text-xs text-slate-500 font-medium uppercase">Konto</dt>
            <dd className="mt-0.5 text-slate-900 font-mono text-xs">
              {buchung.konto?.nummer} {buchung.konto?.name}
            </dd>
          </div>
          <div>
            <dt className="text-xs text-slate-500 font-medium uppercase">Richtung</dt>
            <dd className="mt-0.5">
              <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${buchung.richtung === "EINNAHME" ? "bg-green-50 text-green-700" : "bg-red-50 text-red-700"}`}>
                {buchung.richtung === "EINNAHME" ? "Einnahme" : "Ausgabe"}
              </span>
            </dd>
          </div>
          <div>
            <dt className="text-xs text-slate-500 font-medium uppercase">USt-Code</dt>
            <dd className="mt-0.5 text-slate-700 font-mono text-xs">{buchung.ust_code || "KEINE"}</dd>
          </div>
          <div>
            <dt className="text-xs text-slate-500 font-medium uppercase">Quelle</dt>
            <dd className="mt-0.5 text-slate-700">{QUELLE_LABELS[buchung.quelle] || buchung.quelle}</dd>
          </div>
          {buchung.gegenseite && (
            <div className="col-span-2">
              <dt className="text-xs text-slate-500 font-medium uppercase">Gegenseite</dt>
              <dd className="mt-0.5 text-slate-700">{buchung.gegenseite}</dd>
            </div>
          )}
          {buchung.notizen && (
            <div className="col-span-2">
              <dt className="text-xs text-slate-500 font-medium uppercase">Notizen</dt>
              <dd className="mt-0.5 text-slate-700">{buchung.notizen}</dd>
            </div>
          )}
          <div className="col-span-2 border-t border-slate-100 pt-3 grid grid-cols-3 gap-4">
            <div>
              <dt className="text-xs text-slate-500">Nettobetrag</dt>
              <dd className="text-base font-medium text-slate-900">{fmt(buchung.betrag_netto)}</dd>
            </div>
            <div>
              <dt className="text-xs text-slate-500">USt-Betrag</dt>
              <dd className="text-base font-medium text-slate-900">{fmt(buchung.ust_betrag)}</dd>
            </div>
            <div>
              <dt className="text-xs text-slate-500">Bruttobetrag</dt>
              <dd className={`text-base font-bold ${buchung.richtung === "EINNAHME" ? "text-green-700" : "text-red-700"}`}>{fmt(buchung.betrag_brutto)}</dd>
            </div>
          </div>
        </dl>
      </div>

      {/* Zahlungsdatum — only for non-deleted */}
      {!isDeleted && (
        <div className={`rounded-xl border p-5 mb-4 ${buchung.zahlung_datum ? "bg-green-50 border-green-200" : "bg-amber-50 border-amber-200"}`}>
          <h2 className="font-semibold text-sm mb-3 text-slate-900">
            {buchung.zahlung_datum ? "Zahlung vermerkt" : "Zahlung noch ausstehend"}
          </h2>
          <div className="flex items-center gap-3">
            <input
              type="date"
              value={zahlungDatum}
              onChange={(e) => setZahlungDatum(e.target.value)}
              className="px-3 py-2 border border-slate-300 rounded-lg text-slate-900 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              onClick={handleZahlungSave}
              disabled={savingZahlung || !zahlungDatum}
              className="px-4 py-2 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800 disabled:opacity-50"
            >
              {savingZahlung ? "Speichert…" : "Speichern"}
            </button>
            {buchung.zahlung_datum && (
              <button
                onClick={handleZahlungClear}
                disabled={savingZahlung}
                className="px-3 py-2 text-sm text-red-600 border border-red-200 rounded-lg hover:bg-red-50 disabled:opacity-50"
              >
                Zahlung löschen
              </button>
            )}
          </div>
          <p className="text-xs text-slate-500 mt-2">
            {buchung.zahlung_datum
              ? "Buchung fließt in Saldenliste und Jahresabschluss ein (IST-Prinzip)."
              : "Sobald das Zahlungsdatum gesetzt ist, erscheint die Buchung in Saldenliste und Jahresabschluss."}
          </p>
        </div>
      )}

      {/* Belege */}
      {!isDeleted && (
        <div className="bg-white rounded-xl border border-slate-200 p-6 mb-4">
          <div className="flex items-center justify-between mb-4">
            <h2 className="font-semibold text-slate-900 text-sm">Belege</h2>
            <label className={`px-3 py-1.5 text-xs border rounded-lg cursor-pointer ${uploading ? "text-slate-400 border-slate-200" : "text-blue-700 border-blue-200 hover:bg-blue-50"}`}>
              {uploading ? "Lädt hoch…" : "Beleg hochladen"}
              <input type="file" accept=".pdf,.jpg,.jpeg,.png" className="hidden" onChange={handleBelegUpload} disabled={uploading} />
            </label>
          </div>
          {!buchung.belege || buchung.belege.length === 0 ? (
            <p className="text-sm text-slate-500">Keine Belege vorhanden.</p>
          ) : (
            <ul className="divide-y divide-slate-100">
              {buchung.belege.map((b) => (
                <li key={b.id} className="flex items-center justify-between py-2">
                  <a href={`/api/eegs/${eegId}/ea/belege/${b.id}`} target="_blank" className="text-sm text-blue-600 hover:underline">{b.dateiname}</a>
                  <button onClick={() => handleBelegDelete(b.id)} className="text-xs text-red-600 hover:underline">Löschen</button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      {/* Changelog (BAO §131) */}
      <div className="bg-white rounded-xl border border-slate-200 p-6">
        <h2 className="font-semibold text-slate-900 text-sm mb-4">Änderungshistorie (BAO §131)</h2>
        {changelogLoading ? (
          <p className="text-sm text-slate-500">Wird geladen…</p>
        ) : changelog.length === 0 ? (
          <p className="text-sm text-slate-500">Keine Einträge.</p>
        ) : (
          <ol className="relative border-l border-slate-200 ml-3 space-y-4">
            {changelog.map((c) => (
              <li key={c.id} className="ml-4">
                <div className="absolute -left-1.5 w-3 h-3 rounded-full bg-slate-300 border-2 border-white" />
                <div className="flex items-center gap-2 mb-0.5">
                  <span className={`text-xs font-medium px-1.5 py-0.5 rounded ${OP_COLORS[c.operation] || "bg-slate-100 text-slate-600"}`}>
                    {OP_LABELS[c.operation] || c.operation}
                  </span>
                  <span className="text-xs text-slate-500">{fmtDateTime(c.changed_at)}</span>
                  <span className="text-xs text-slate-400 font-mono">{c.changed_by.slice(0, 8)}</span>
                </div>
                {c.reason && (
                  <p className="text-xs text-slate-600 italic">Grund: {c.reason}</p>
                )}
                {c.operation === "update" && c.old_values && c.new_values && (
                  <ChangelogDiff old={c.old_values} next={c.new_values} />
                )}
              </li>
            ))}
          </ol>
        )}
      </div>
    </div>
  );
}

function ChangelogDiff({ old: oldV, next: newV }: { old: Record<string, unknown>; next: Record<string, unknown> }) {
  const changed = Object.keys(newV).filter((k) => {
    const o = oldV[k] == null ? "" : String(oldV[k]);
    const n = newV[k] == null ? "" : String(newV[k]);
    return o !== n;
  });
  if (changed.length === 0) return null;

  const FIELD_LABELS: Record<string, string> = {
    zahlung_datum: "Zahlungsdatum",
    beleg_datum: "Belegdatum",
    betrag_brutto: "Betrag",
    beschreibung: "Beschreibung",
    konto_id: "Konto",
    ust_code: "USt-Code",
    gegenseite: "Gegenseite",
    notizen: "Notizen",
    belegnr: "Beleg-Nr.",
  };

  return (
    <ul className="mt-1 space-y-0.5">
      {changed.map((k) => (
        <li key={k} className="text-xs text-slate-500">
          <span className="font-medium text-slate-600">{FIELD_LABELS[k] || k}:</span>{" "}
          <span className="line-through text-red-400">{formatVal(oldV[k])}</span>{" → "}
          <span className="text-green-600">{formatVal(newV[k])}</span>
        </li>
      ))}
    </ul>
  );
}

function formatVal(v: unknown): string {
  if (v == null || v === "") return "—";
  if (typeof v === "string" && v.match(/^\d{4}-\d{2}-\d{2}/)) {
    try { return new Date(v).toLocaleDateString("de-AT", { day: "2-digit", month: "2-digit", year: "numeric" }); } catch { /* noop */ }
  }
  if (typeof v === "number") {
    return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(v);
  }
  return String(v);
}
