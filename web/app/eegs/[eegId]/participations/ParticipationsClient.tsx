"use client";

import { useState } from "react";
import { type EEGMeterParticipation } from "@/lib/api";

interface MpInfo {
  memberName: string;
  meterId: string;
}

interface Props {
  eegId: string;
  initialParticipations: EEGMeterParticipation[];
  mpToMember: Record<string, MpInfo>;
  allMeterPoints: { id: string; label: string }[];
}

const SHARE_TYPE_LABELS: Record<string, string> = {
  GC: "GC — Vollzuteilung",
  RC_R: "RC_R — Residualeinspeiser",
  RC_L: "RC_L — Lastabhängig",
  CC: "CC — Konstant",
};

export default function ParticipationsClient({ eegId, initialParticipations, mpToMember, allMeterPoints }: Props) {
  const [participations, setParticipations] = useState(initialParticipations);
  const [showForm, setShowForm] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Form state
  const [mpId, setMpId] = useState(allMeterPoints[0]?.id || "");
  const [factor, setFactor] = useState("100");
  const [shareType, setShareType] = useState("GC");
  const [validFrom, setValidFrom] = useState("");
  const [validUntil, setValidUntil] = useState("");
  const [notes, setNotes] = useState("");

  function startCreate() {
    setEditId(null);
    setMpId(allMeterPoints[0]?.id || "");
    setFactor("100");
    setShareType("GC");
    setValidFrom("");
    setValidUntil("");
    setNotes("");
    setError(null);
    setShowForm(true);
  }

  function startEdit(p: EEGMeterParticipation) {
    setEditId(p.id);
    setMpId(p.meter_point_id);
    setFactor(String(p.participation_factor));
    setShareType(p.share_type);
    setValidFrom(p.valid_from.slice(0, 10));
    setValidUntil(p.valid_until ? p.valid_until.slice(0, 10) : "");
    setNotes(p.notes || "");
    setError(null);
    setShowForm(true);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const body = {
        meter_point_id: mpId,
        participation_factor: parseFloat(factor),
        share_type: shareType,
        valid_from: validFrom,
        valid_until: validUntil || undefined,
        notes,
      };

      let res: Response;
      if (editId) {
        res = await fetch(`/api/eegs/${eegId}/participations/${editId}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });
      } else {
        res = await fetch(`/api/eegs/${eegId}/participations`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });
      }

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      const data = await res.json();

      if (editId) {
        setParticipations((prev) => prev.map((p) => (p.id === editId ? data : p)));
      } else {
        setParticipations((prev) => [data, ...prev]);
      }
      setShowForm(false);
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Teilnahmedatensatz wirklich löschen?")) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/participations/${id}`, { method: "DELETE" });
      if (!res.ok && res.status !== 204) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      setParticipations((prev) => prev.filter((p) => p.id !== id));
    } catch (err: unknown) {
      alert((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      {/* Header actions */}
      <div className="flex justify-between items-center">
        <p className="text-sm text-slate-600">
          {participations.length === 0 ? "Keine Teilnahmedatensätze vorhanden." : `${participations.length} Datensatz/Datensätze`}
        </p>
        <button
          onClick={startCreate}
          disabled={allMeterPoints.length === 0}
          className="px-4 py-2 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800 disabled:opacity-50 transition-colors"
        >
          + Teilnahme hinzufügen
        </button>
      </div>

      {/* Create/Edit form */}
      {showForm && (
        <div className="bg-white rounded-xl border border-slate-200 p-6">
          <h2 className="text-base font-semibold text-slate-900 mb-4">
            {editId ? "Teilnahme bearbeiten" : "Neue Teilnahme"}
          </h2>
          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-800 text-sm">{error}</div>
          )}
          <form onSubmit={handleSubmit} className="space-y-4 max-w-lg">
            {!editId && (
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1.5">Zählpunkt *</label>
                <select
                  value={mpId}
                  onChange={(e) => setMpId(e.target.value)}
                  required
                  disabled={loading}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {allMeterPoints.map((mp) => (
                    <option key={mp.id} value={mp.id}>{mp.label}</option>
                  ))}
                </select>
              </div>
            )}

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1.5">Teilnahmefaktor (%) *</label>
                <input
                  type="number"
                  step="0.0001"
                  min="0.0001"
                  max="100"
                  value={factor}
                  onChange={(e) => setFactor(e.target.value)}
                  required
                  disabled={loading}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1.5">Anteilstyp</label>
                <select
                  value={shareType}
                  onChange={(e) => setShareType(e.target.value)}
                  disabled={loading}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {Object.entries(SHARE_TYPE_LABELS).map(([k, v]) => (
                    <option key={k} value={k}>{v}</option>
                  ))}
                </select>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1.5">Gültig ab *</label>
                <input
                  type="date"
                  value={validFrom}
                  onChange={(e) => setValidFrom(e.target.value)}
                  required
                  disabled={loading}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1.5">Gültig bis (leer = offen)</label>
                <input
                  type="date"
                  value={validUntil}
                  onChange={(e) => setValidUntil(e.target.value)}
                  disabled={loading}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">Anmerkungen</label>
              <input
                type="text"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                disabled={loading}
                className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <div className="flex gap-3">
              <button
                type="submit"
                disabled={loading}
                className="px-5 py-2.5 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800 disabled:opacity-50 transition-colors"
              >
                {loading ? "Wird gespeichert…" : editId ? "Speichern" : "Erstellen"}
              </button>
              <button
                type="button"
                onClick={() => setShowForm(false)}
                disabled={loading}
                className="px-5 py-2.5 bg-slate-100 text-slate-700 text-sm font-medium rounded-lg hover:bg-slate-200 transition-colors"
              >
                Abbrechen
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Table */}
      {participations.length > 0 && (
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wide">Mitglied / Zählpunkt</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wide">Faktor</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wide">Typ</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wide">Gültig ab</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wide">Gültig bis</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wide">Anmerkung</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {participations.map((p) => {
                const mp = mpToMember[p.meter_point_id];
                return (
                  <tr key={p.id} className="hover:bg-slate-50">
                    <td className="px-4 py-3">
                      <p className="font-medium text-slate-900">{mp?.memberName || "—"}</p>
                      <p className="text-xs text-slate-400">{mp?.meterId || p.meter_point_id.slice(0, 8)}</p>
                    </td>
                    <td className="px-4 py-3 font-mono text-slate-700">{p.participation_factor}%</td>
                    <td className="px-4 py-3 text-slate-600">{p.share_type}</td>
                    <td className="px-4 py-3 text-slate-600">{p.valid_from.slice(0, 10)}</td>
                    <td className="px-4 py-3 text-slate-600">{p.valid_until ? p.valid_until.slice(0, 10) : <span className="text-slate-400">offen</span>}</td>
                    <td className="px-4 py-3 text-slate-500 text-xs">{p.notes || "—"}</td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2 justify-end">
                        <button
                          onClick={() => startEdit(p)}
                          className="text-xs px-2 py-1 text-blue-700 hover:bg-blue-50 rounded transition-colors"
                        >
                          Bearbeiten
                        </button>
                        <button
                          onClick={() => handleDelete(p.id)}
                          className="text-xs px-2 py-1 text-red-600 hover:bg-red-50 rounded transition-colors"
                        >
                          Löschen
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {participations.length === 0 && !showForm && (
        <div className="bg-slate-50 rounded-xl border border-slate-200 p-8 text-center">
          <p className="text-slate-500 text-sm">Noch keine Mehrfachteilnahmen konfiguriert.</p>
          <p className="text-xs text-slate-400 mt-1">
            Klicken Sie auf &ldquo;Teilnahme hinzufügen&rdquo;, um einen Zählpunkt für die parallele
            Teilnahme an dieser Energiegemeinschaft zu registrieren.
          </p>
        </div>
      )}
    </div>
  );
}
