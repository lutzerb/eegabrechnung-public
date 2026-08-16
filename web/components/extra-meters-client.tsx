"use client";

import { useState, useEffect, useCallback } from "react";
import { type ExtraMeter, type ExtraMeterReading } from "@/lib/api";

interface Props {
  eegId: string;
  memberId: string;
  initialExtraMeters: ExtraMeter[];
}

function formatDate(dateStr: string): string {
  try {
    return new Date(dateStr).toLocaleDateString("de-AT", { day: "2-digit", month: "2-digit", year: "numeric" });
  } catch {
    return dateStr;
  }
}

function ReadingsSection({ eegId, memberId, meter }: { eegId: string; memberId: string; meter: ExtraMeter }) {
  const [readings, setReadings] = useState<ExtraMeterReading[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [date, setDate] = useState("");
  const [counterValue, setCounterValue] = useState("");
  const [notes, setNotes] = useState("");
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/eegs/${eegId}/members/${memberId}/extra-meters/${meter.id}/readings`);
      if (!res.ok) throw new Error(`Fehler ${res.status}`);
      setReadings(await res.json());
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [eegId, memberId, meter.id]);

  useEffect(() => {
    load();
  }, [load]);

  async function handleAddReading(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    try {
      const res = await fetch(`/api/eegs/${eegId}/members/${memberId}/extra-meters/${meter.id}/readings`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reading_date: date, counter_value: parseFloat(counterValue), notes }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      const created = await res.json();
      setReadings((prev) => [created, ...(prev || [])].sort((a, b) => (a.reading_date < b.reading_date ? 1 : -1)));
      setDate("");
      setCounterValue("");
      setNotes("");
      setShowForm(false);
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  }

  async function handleDeleteReading(readingId: string) {
    if (!confirm("Ablesung wirklich löschen?")) return;
    try {
      const res = await fetch(`/api/eegs/${eegId}/members/${memberId}/extra-meters/${meter.id}/readings/${readingId}`, {
        method: "DELETE",
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      setReadings((prev) => (prev || []).filter((r) => r.id !== readingId));
    } catch (err: unknown) {
      alert((err as Error).message);
    }
  }

  return (
    <div className="bg-slate-50 border-t border-slate-200 px-4 py-3">
      {error && <div className="mb-2 p-2 bg-red-50 border border-red-200 rounded text-red-800 text-xs">{error}</div>}
      {loading && <p className="text-xs text-slate-400">Lädt Ablesungen…</p>}
      {readings && readings.length === 0 && !showForm && (
        <p className="text-xs text-slate-400">Noch keine Ablesungen erfasst.</p>
      )}
      {readings && readings.length > 0 && (
        <table className="w-full text-xs mb-2">
          <thead>
            <tr className="text-slate-500">
              <th className="text-left py-1 font-medium">Datum</th>
              <th className="text-left py-1 font-medium">Zählerstand (kWh)</th>
              <th className="text-left py-1 font-medium">Anmerkung</th>
              <th></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200">
            {readings.map((r) => (
              <tr key={r.id}>
                <td className="py-1 text-slate-700">{formatDate(r.reading_date)}</td>
                <td className="py-1 font-mono text-slate-700">{r.counter_value}</td>
                <td className="py-1 text-slate-500">{r.notes || "—"}</td>
                <td className="py-1 text-right">
                  <button onClick={() => handleDeleteReading(r.id)} className="text-red-600 hover:underline">
                    Löschen
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {showForm ? (
        <form onSubmit={handleAddReading} className="flex flex-wrap items-end gap-2 mt-2">
          <div>
            <label className="block text-xs text-slate-500 mb-1">Datum</label>
            <input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              required
              disabled={saving}
              className="px-2 py-1 border border-slate-300 rounded text-sm text-slate-900"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Zählerstand (kWh)</label>
            <input
              type="number"
              step="0.001"
              value={counterValue}
              onChange={(e) => setCounterValue(e.target.value)}
              required
              disabled={saving}
              className="px-2 py-1 border border-slate-300 rounded text-sm text-slate-900 w-32"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Anmerkung</label>
            <input
              type="text"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              disabled={saving}
              className="px-2 py-1 border border-slate-300 rounded text-sm text-slate-900"
            />
          </div>
          <button
            type="submit"
            disabled={saving}
            className="px-3 py-1.5 bg-blue-700 text-white text-xs font-medium rounded hover:bg-blue-800 disabled:opacity-50"
          >
            {saving ? "Speichert…" : "Speichern"}
          </button>
          <button
            type="button"
            onClick={() => setShowForm(false)}
            disabled={saving}
            className="px-3 py-1.5 bg-slate-100 text-slate-700 text-xs font-medium rounded hover:bg-slate-200"
          >
            Abbrechen
          </button>
        </form>
      ) : (
        <button
          onClick={() => setShowForm(true)}
          className="text-xs px-2 py-1 text-blue-700 hover:bg-blue-50 rounded"
        >
          + Ablesung erfassen
        </button>
      )}
    </div>
  );
}

export default function ExtraMetersClient({ eegId, memberId, initialExtraMeters }: Props) {
  const [meters, setMeters] = useState(initialExtraMeters);
  const [showForm, setShowForm] = useState(false);
  const [label, setLabel] = useState("");
  const [notes, setNotes] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/eegs/${eegId}/members/${memberId}/extra-meters`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ label, notes }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      const created = await res.json();
      setMeters((prev) => [...prev, created]);
      setLabel("");
      setNotes("");
      setShowForm(false);
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  async function handleToggleStatus(meter: ExtraMeter) {
    const newStatus = meter.status === "ACTIVE" ? "INACTIVE" : "ACTIVE";
    try {
      const res = await fetch(`/api/eegs/${eegId}/members/${memberId}/extra-meters/${meter.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ label: meter.label, status: newStatus, notes: meter.notes }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      const updated = await res.json();
      setMeters((prev) => prev.map((m) => (m.id === meter.id ? updated : m)));
    } catch (err: unknown) {
      alert((err as Error).message);
    }
  }

  async function handleDelete(meter: ExtraMeter) {
    if (!confirm(`Zusatzzähler "${meter.label}" wirklich löschen? Alle Ablesungen gehen dabei verloren.`)) return;
    try {
      const res = await fetch(`/api/eegs/${eegId}/members/${memberId}/extra-meters/${meter.id}`, { method: "DELETE" });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Fehler ${res.status}`);
      }
      setMeters((prev) => prev.filter((m) => m.id !== meter.id));
    } catch (err: unknown) {
      alert((err as Error).message);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <p className="text-sm text-slate-600">
          {meters.length === 0 ? "Keine Zusatzzähler vorhanden." : `${meters.length} Zusatzzähler`}
        </p>
        <button
          onClick={() => setShowForm(true)}
          className="px-4 py-2 bg-blue-700 text-white text-sm font-medium rounded-lg hover:bg-blue-800 transition-colors"
        >
          + Zusatzzähler hinzufügen
        </button>
      </div>

      {showForm && (
        <div className="bg-white rounded-xl border border-slate-200 p-6">
          <h2 className="text-base font-semibold text-slate-900 mb-4">Neuer Zusatzzähler</h2>
          {error && <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-800 text-sm">{error}</div>}
          <form onSubmit={handleCreate} className="space-y-4 max-w-lg">
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">Bezeichnung *</label>
              <input
                type="text"
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                placeholder="z.B. Wärmepumpe Garage"
                required
                disabled={loading}
                className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
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
                {loading ? "Wird gespeichert…" : "Erstellen"}
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

      {meters.length > 0 && (
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          {meters.map((meter) => (
            <div key={meter.id} className="border-b border-slate-200 last:border-b-0">
              <div className="flex items-center justify-between px-4 py-3 hover:bg-slate-50">
                <button
                  onClick={() => setExpandedId(expandedId === meter.id ? null : meter.id)}
                  className="flex-1 text-left"
                >
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-slate-900">{meter.label}</span>
                    <span
                      className={`text-xs px-1.5 py-0.5 rounded font-medium ${
                        meter.status === "ACTIVE" ? "bg-green-50 text-green-700 border border-green-200" : "bg-slate-100 text-slate-500"
                      }`}
                    >
                      {meter.status === "ACTIVE" ? "Aktiv" : "Inaktiv"}
                    </span>
                  </div>
                  {meter.notes && <p className="text-xs text-slate-400 mt-0.5">{meter.notes}</p>}
                </button>
                <div className="flex gap-2">
                  <button
                    onClick={() => handleToggleStatus(meter)}
                    className="text-xs px-2 py-1 text-slate-600 hover:bg-slate-100 rounded transition-colors"
                  >
                    {meter.status === "ACTIVE" ? "Deaktivieren" : "Aktivieren"}
                  </button>
                  <button
                    onClick={() => handleDelete(meter)}
                    className="text-xs px-2 py-1 text-red-600 hover:bg-red-50 rounded transition-colors"
                  >
                    Löschen
                  </button>
                </div>
              </div>
              {expandedId === meter.id && <ReadingsSection eegId={eegId} memberId={memberId} meter={meter} />}
            </div>
          ))}
        </div>
      )}

      {meters.length === 0 && !showForm && (
        <div className="bg-slate-50 rounded-xl border border-slate-200 p-8 text-center">
          <p className="text-slate-500 text-sm">Noch keine Zusatzzähler erfasst.</p>
          <p className="text-xs text-slate-400 mt-1">
            Zusatzzähler sind manuell abgelesene Nebenzähler (z.B. Wärmepumpe), die keine Netzbetreiber-Smart-Meter
            sind. Der Verbrauch wird aus der Differenz zweier Zählerstände berechnet und zum normalen Bezugspreis
            des Mitglieds auf der Rechnung ausgewiesen.
          </p>
        </div>
      )}
    </div>
  );
}
