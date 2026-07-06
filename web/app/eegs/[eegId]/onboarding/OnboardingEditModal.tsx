"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { validateUIDNummer } from "@/lib/validation";

interface MeterPoint {
  zaehlpunkt: string;
  direction: string;
  generation_type?: string;
  participation_factor?: number;
}

interface OnboardingRequest {
  id: string;
  eeg_id: string;
  status: string;
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
  business_role: string;
  uid_nummer: string;
  use_vat: boolean;
  meter_points: MeterPoint[];
  beitritts_datum?: string;
  admin_notes: string;
}

interface Props {
  eegId: string;
  req: OnboardingRequest;
  onClose: () => void;
  onSuccess: () => void;
}

const MEMBER_TYPES = [
  { value: "CONSUMER", label: "Verbraucher (Bezug)" },
  { value: "PRODUCER", label: "Erzeuger (Einspeisung)" },
  { value: "PROSUMER", label: "Prosumer (Bezug & Einspeisung)" },
];

const BUSINESS_ROLES = [
  { value: "privat", label: "Privatperson" },
  { value: "kleinunternehmer", label: "Kleinunternehmer" },
  { value: "verein", label: "Verein" },
  { value: "landwirt_pauschaliert", label: "Landwirt (pauschaliert, § 22 UStG)" },
  { value: "landwirt", label: "Landwirt (buchführungspflichtig)" },
  { value: "unternehmen", label: "Unternehmen" },
  { value: "gemeinde_bga", label: "Gemeinde (BgA)" },
  { value: "gemeinde_hoheitlich", label: "Gemeinde (hoheitlich)" },
];

const GENERATION_TYPES = [
  "PV",
  "Windkraft",
  "Wasserkraft",
  "Biomasse",
  "Sonstige",
];

function formatDateForInput(dateStr?: string): string {
  if (!dateStr) return "";
  try {
    const d = new Date(dateStr);
    return d.toISOString().substring(0, 10);
  } catch {
    return "";
  }
}

export default function OnboardingEditModal({ eegId, req, onClose, onSuccess }: Props) {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [name1, setName1] = useState(req.name1 || "");
  const [name2, setName2] = useState(req.name2 || "");
  const [email, setEmail] = useState(req.email || "");
  const [phone, setPhone] = useState(req.phone || "");
  const [strasse, setStrasse] = useState(req.strasse || "");
  const [plz, setPlz] = useState(req.plz || "");
  const [ort, setOrt] = useState(req.ort || "");
  const [iban, setIban] = useState(req.iban || "");
  const [bic, setBic] = useState(req.bic || "");
  const [memberType, setMemberType] = useState(req.member_type || "CONSUMER");
  const [businessRole, setBusinessRole] = useState(req.business_role || "privat");
  const [uidNummer, setUidNummer] = useState(req.uid_nummer || "");
  const [useVat, setUseVat] = useState(req.use_vat || false);
  const [beitrittsDatum, setBeitrittsDatum] = useState(formatDateForInput(req.beitritts_datum));
  const [adminNotes, setAdminNotes] = useState(req.admin_notes || "");
  const [meterPoints, setMeterPoints] = useState<MeterPoint[]>(
    req.meter_points && req.meter_points.length > 0
      ? req.meter_points
      : [{ zaehlpunkt: "", direction: "CONSUMPTION" }]
  );

  function addMeterPoint() {
    setMeterPoints((prev) => [...prev, { zaehlpunkt: "", direction: "CONSUMPTION" }]);
  }

  function removeMeterPoint(idx: number) {
    setMeterPoints((prev) => prev.filter((_, i) => i !== idx));
  }

  function updateMeterPoint(idx: number, field: keyof MeterPoint, value: string) {
    setMeterPoints((prev) =>
      prev.map((mp, i) => {
        if (i !== idx) return mp;
        const updated = { ...mp, [field]: value };
        if (field === "direction" && value !== "GENERATION") {
          delete updated.generation_type;
        }
        return updated;
      })
    );
  }

  async function handleSave() {
    // Client-side validation
    if (uidNummer.trim()) {
      const uidErr = validateUIDNummer(uidNummer);
      if (uidErr) { setError(uidErr); return; }
    }

    setError(null);
    setLoading(true);
    try {
      const body: Record<string, unknown> = {
        name1,
        name2,
        email,
        phone,
        strasse,
        plz,
        ort,
        iban,
        bic,
        member_type: memberType,
        business_role: businessRole,
        uid_nummer: uidNummer,
        use_vat: useVat,
        meter_points: meterPoints.filter((mp) => mp.zaehlpunkt.trim() !== ""),
        admin_notes: adminNotes,
        beitritts_datum: beitrittsDatum || "",
      };

      const res = await fetch(`/api/eegs/${eegId}/onboarding/${req.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error || "Fehler beim Speichern.");
        return;
      }

      onSuccess();
      router.refresh();
    } catch {
      setError("Netzwerkfehler.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-2xl max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="p-6 border-b border-slate-200 flex items-center justify-between shrink-0">
          <h2 className="text-xl font-bold text-slate-900">Antrag bearbeiten</h2>
          <button
            type="button"
            onClick={onClose}
            disabled={loading}
            className="text-slate-400 hover:text-slate-600 text-xl leading-none"
          >
            ×
          </button>
        </div>

        {/* Body */}
        <div className="overflow-y-auto p-6 space-y-6">
          {error && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
              {error}
            </div>
          )}

          {/* Persönliche Daten */}
          <section>
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide mb-3">
              Persönliche Daten
            </h3>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-slate-500 mb-1">Vorname / Name 1 *</label>
                <input
                  type="text"
                  value={name1}
                  onChange={(e) => setName1(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-slate-500 mb-1">Nachname / Name 2</label>
                <input
                  type="text"
                  value={name2}
                  onChange={(e) => setName2(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-slate-500 mb-1">E-Mail *</label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-slate-500 mb-1">Telefon</label>
                <input
                  type="text"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-slate-500 mb-1">Mitgliedstyp</label>
                <select
                  value={memberType}
                  onChange={(e) => setMemberType(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {MEMBER_TYPES.map((t) => (
                    <option key={t.value} value={t.value}>{t.label}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs text-slate-500 mb-1">Unternehmensart</label>
                <select
                  value={businessRole}
                  onChange={(e) => setBusinessRole(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {BUSINESS_ROLES.map((r) => (
                    <option key={r.value} value={r.value}>{r.label}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs text-slate-500 mb-1">UID-Nummer</label>
                <input
                  type="text"
                  value={uidNummer}
                  onChange={(e) => setUidNummer(e.target.value)}
                  placeholder="ATU..."
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="flex items-center gap-2 pt-5">
                <input
                  type="checkbox"
                  id="use_vat"
                  checked={useVat}
                  onChange={(e) => setUseVat(e.target.checked)}
                  className="rounded"
                />
                <label htmlFor="use_vat" className="text-sm text-slate-700">USt-pflichtig</label>
              </div>
              <div>
                <label className="block text-xs text-slate-500 mb-1">Gewünschter Beitritt</label>
                <input
                  type="date"
                  value={beitrittsDatum}
                  onChange={(e) => setBeitrittsDatum(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>
          </section>

          {/* Adresse & Bankdaten */}
          <section>
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide mb-3">
              Adresse & Bankdaten
            </h3>
            <div className="grid grid-cols-2 gap-3">
              <div className="col-span-2">
                <label className="block text-xs text-slate-500 mb-1">Straße *</label>
                <input
                  type="text"
                  value={strasse}
                  onChange={(e) => setStrasse(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-slate-500 mb-1">PLZ *</label>
                <input
                  type="text"
                  value={plz}
                  onChange={(e) => setPlz(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-slate-500 mb-1">Ort *</label>
                <input
                  type="text"
                  value={ort}
                  onChange={(e) => setOrt(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-slate-500 mb-1">IBAN</label>
                <input
                  type="text"
                  value={iban}
                  onChange={(e) => setIban(e.target.value.replace(/\s/g, "").toUpperCase())}
                  placeholder="AT..."
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-slate-500 mb-1">BIC</label>
                <input
                  type="text"
                  value={bic}
                  onChange={(e) => setBic(e.target.value.toUpperCase())}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>
          </section>

          {/* Zählpunkte */}
          <section>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">
                Zählpunkte
              </h3>
              <button
                type="button"
                onClick={addMeterPoint}
                className="text-xs text-blue-600 hover:text-blue-700 font-medium"
              >
                + Zählpunkt hinzufügen
              </button>
            </div>
            <div className="space-y-3">
              {meterPoints.map((mp, idx) => (
                <div key={idx} className="bg-slate-50 rounded-lg p-3 space-y-2">
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={mp.zaehlpunkt}
                      onChange={(e) => updateMeterPoint(idx, "zaehlpunkt", e.target.value)}
                      placeholder="AT0030000000000000000000000XXXXX"
                      className="flex-1 px-3 py-1.5 border border-slate-300 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                    <select
                      value={mp.direction}
                      onChange={(e) => updateMeterPoint(idx, "direction", e.target.value)}
                      className="px-3 py-1.5 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="CONSUMPTION">Bezug</option>
                      <option value="GENERATION">Einspeisung</option>
                    </select>
                    {meterPoints.length > 1 && (
                      <button
                        type="button"
                        onClick={() => removeMeterPoint(idx)}
                        className="px-2 text-slate-400 hover:text-red-500 transition-colors"
                        title="Zählpunkt entfernen"
                      >
                        ×
                      </button>
                    )}
                  </div>
                  {mp.direction === "GENERATION" && (
                    <div>
                      <label className="block text-xs text-slate-500 mb-1">Erzeugungsart</label>
                      <select
                        value={mp.generation_type || ""}
                        onChange={(e) => updateMeterPoint(idx, "generation_type", e.target.value)}
                        className="w-full px-3 py-1.5 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      >
                        <option value="">— bitte wählen</option>
                        {GENERATION_TYPES.map((gt) => (
                          <option key={gt} value={gt}>{gt}</option>
                        ))}
                      </select>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </section>

          {/* Admin-Notizen */}
          <section>
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide mb-3">
              Admin-Notizen
            </h3>
            <textarea
              value={adminNotes}
              onChange={(e) => setAdminNotes(e.target.value)}
              rows={3}
              placeholder="Interne Notiz (nicht sichtbar für Mitglied)..."
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
            />
          </section>
        </div>

        {/* Footer */}
        <div className="p-6 border-t border-slate-200 flex gap-3 justify-end shrink-0">
          <button
            type="button"
            onClick={onClose}
            disabled={loading}
            className="px-4 py-2 border border-slate-300 text-slate-700 rounded-lg text-sm font-medium hover:bg-slate-50 transition-colors disabled:opacity-50"
          >
            Abbrechen
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={loading}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors disabled:opacity-50"
          >
            {loading ? "Wird gespeichert..." : "Speichern"}
          </button>
        </div>
      </div>
    </div>
  );
}
