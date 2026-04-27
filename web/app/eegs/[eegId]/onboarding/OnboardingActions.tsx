"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  byMarktpartnerID,
  byZaehlpunkt,
  allNetzbetreiber,
  NetzbetreiberInfo,
} from "@/lib/netzbetreiber";

interface Props {
  eegId: string;
  requestId: string;
  currentStatus: string;
  eegNetzbetreiberId?: string;
  meterPoints?: Array<{ zaehlpunkt: string }>;
}

interface ConvertModalProps {
  eegId: string;
  requestId: string;
  eegNetzbetreiberId?: string;
  meterPoints?: Array<{ zaehlpunkt: string }>;
  onClose: () => void;
  onSuccess: () => void;
}

function ConvertModal({
  eegId,
  requestId,
  eegNetzbetreiberId,
  meterPoints,
  onClose,
  onSuccess,
}: ConvertModalProps) {
  const [loading, setLoading] = useState(false);
  const [customMessage, setCustomMessage] = useState("");
  const [adminNotes, setAdminNotes] = useState("");

  // Detect NB: first try eegNetzbetreiberId, then from meterPoints
  let autoDetected: (NetzbetreiberInfo & { id: string }) | null = null;
  if (eegNetzbetreiberId) {
    const info = byMarktpartnerID(eegNetzbetreiberId);
    if (info) autoDetected = { id: eegNetzbetreiberId.toUpperCase(), ...info };
  }
  if (!autoDetected && meterPoints) {
    for (const mp of meterPoints) {
      const info = byZaehlpunkt(mp.zaehlpunkt);
      if (info) {
        const id = mp.zaehlpunkt.substring(0, 8).toUpperCase();
        autoDetected = { id, ...info };
        break;
      }
    }
  }

  const allNBs = allNetzbetreiber();
  const [selectedNbId, setSelectedNbId] = useState(autoDetected?.id || "");

  const selectedNb = allNBs.find((nb) => nb.id === selectedNbId);

  async function handleSubmit() {
    setLoading(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/onboarding/${requestId}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          status: "approved",
          admin_notes: adminNotes,
          netzbetreiber_id: selectedNbId || undefined,
          custom_message: customMessage || undefined,
        }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        alert(data.error || "Fehler beim Aktualisieren des Status.");
        return;
      }
      onSuccess();
    } catch {
      alert("Netzwerkfehler.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="p-6 border-b border-slate-200">
          <h2 className="text-xl font-bold text-slate-900">Mitglied aufnehmen</h2>
        </div>

        <div className="p-6 space-y-6">
          {/* Section 1: Netzbetreiber */}
          <div>
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide mb-3">
              Netzbetreiber
            </h3>
            {autoDetected && (
              <div className="mb-3 p-3 bg-slate-50 rounded-lg border border-slate-200 text-sm">
                <p className="font-medium text-slate-900">{autoDetected.name}</p>
                <a
                  href={autoDetected.portalUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-600 hover:underline text-xs break-all"
                >
                  {autoDetected.portalUrl}
                </a>
              </div>
            )}
            <label className="block text-sm text-slate-600 mb-1">
              {autoDetected ? "Netzbetreiber überschreiben:" : "Netzbetreiber wählen:"}
            </label>
            <select
              value={selectedNbId}
              onChange={(e) => setSelectedNbId(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">
                {autoDetected ? "— Erkannt: " + autoDetected.name : "Nicht erkannt – bitte wählen"}
              </option>
              {allNBs.map((nb) => (
                <option key={nb.id} value={nb.id}>
                  {nb.name}
                </option>
              ))}
            </select>
            {selectedNb && selectedNbId !== autoDetected?.id && (
              <p className="mt-1 text-xs text-slate-500">
                <a
                  href={selectedNb.portalUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-600 hover:underline"
                >
                  {selectedNb.portalUrl}
                </a>
              </p>
            )}
          </div>

          {/* Section 2: Nachricht an Mitglied */}
          <div>
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide mb-2">
              Nachricht an Mitglied{" "}
              <span className="text-slate-400 font-normal normal-case">(optional)</span>
            </h3>
            <textarea
              value={customMessage}
              onChange={(e) => setCustomMessage(e.target.value)}
              placeholder="Optionale persönliche Nachricht an das Mitglied..."
              rows={4}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
            />
          </div>

          {/* Section 3: Admin-Notizen */}
          <div>
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide mb-2">
              Admin-Notizen{" "}
              <span className="text-slate-400 font-normal normal-case">(optional)</span>
            </h3>
            <textarea
              value={adminNotes}
              onChange={(e) => setAdminNotes(e.target.value)}
              placeholder="Interne Notiz (nicht sichtbar für Mitglied)..."
              rows={2}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
            />
          </div>
        </div>

        <div className="p-6 border-t border-slate-200 flex gap-3 justify-end">
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
            onClick={handleSubmit}
            disabled={loading}
            className="px-4 py-2 bg-green-600 text-white rounded-lg text-sm font-medium hover:bg-green-700 transition-colors disabled:opacity-50"
          >
            {loading ? "Wird gespeichert..." : "Mitglied aufnehmen & E-Mail senden"}
          </button>
        </div>
      </div>
    </div>
  );
}

export default function OnboardingActions({
  eegId,
  requestId,
  currentStatus,
  eegNetzbetreiberId,
  meterPoints,
}: Props) {
  const [loading, setLoading] = useState(false);
  const [showConvertModal, setShowConvertModal] = useState(false);
  const [showRejectNotes, setShowRejectNotes] = useState(false);
  const [rejectNotes, setRejectNotes] = useState("");
  const router = useRouter();

  async function handleDelete() {
    if (!window.confirm("Beitrittsantrag wirklich löschen? Diese Aktion kann nicht rückgängig gemacht werden.")) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/onboarding/${requestId}`, { method: "DELETE" });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        alert(data.error || "Löschen fehlgeschlagen.");
        return;
      }
      router.refresh();
    } catch {
      alert("Netzwerkfehler.");
    } finally {
      setLoading(false);
    }
  }

  async function updateStatus(status: string, extraBody?: Record<string, unknown>) {
    setLoading(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/onboarding/${requestId}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status, admin_notes: "", ...extraBody }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        alert(data.error || "Fehler beim Aktualisieren des Status.");
        return;
      }
      router.refresh();
    } catch {
      alert("Netzwerkfehler.");
    } finally {
      setLoading(false);
    }
  }

  function handleMarkActive() {
    if (
      window.confirm(
        "Mitglied als aktiv markieren? Dies bestätigt, dass der Netzbetreiber den Zählpunkt freigeschaltet hat."
      )
    ) {
      updateStatus("active", { admin_notes: "" });
    }
  }

  if (showRejectNotes) {
    return (
      <div className="space-y-2">
        <textarea
          value={rejectNotes}
          onChange={(e) => setRejectNotes(e.target.value)}
          placeholder="Optionaler Hinweis für den Antragsteller..."
          rows={2}
          className="w-full px-2 py-1.5 border border-slate-300 rounded text-xs focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
        />
        <div className="flex gap-1.5">
          <button
            type="button"
            onClick={() =>
              updateStatus("rejected", { admin_notes: rejectNotes }).then(() => {
                setShowRejectNotes(false);
                setRejectNotes("");
              })
            }
            disabled={loading}
            className="flex-1 px-2 py-1 bg-red-600 text-white rounded text-xs font-medium hover:bg-red-700 transition-colors disabled:opacity-50"
          >
            {loading ? "..." : "Ablehnen"}
          </button>
          <button
            type="button"
            onClick={() => {
              setShowRejectNotes(false);
              setRejectNotes("");
            }}
            disabled={loading}
            className="flex-1 px-2 py-1 border border-slate-300 text-slate-600 rounded text-xs hover:bg-slate-50 transition-colors disabled:opacity-50"
          >
            Abbrechen
          </button>
        </div>
      </div>
    );
  }

  return (
    <>
      {showConvertModal && (
        <ConvertModal
          eegId={eegId}
          requestId={requestId}
          eegNetzbetreiberId={eegNetzbetreiberId}
          meterPoints={meterPoints}
          onClose={() => setShowConvertModal(false)}
          onSuccess={() => {
            setShowConvertModal(false);
            router.refresh();
          }}
        />
      )}

      <div className="flex flex-wrap gap-1.5">
        {currentStatus === "pending" && (
          <>
            <button
              type="button"
              onClick={() => setShowConvertModal(true)}
              disabled={loading}
              className="px-2.5 py-1 bg-green-600 text-white rounded text-xs font-medium hover:bg-green-700 transition-colors disabled:opacity-50"
            >
              Genehmigen & Aufnehmen
            </button>
            <button
              type="button"
              onClick={() => setShowRejectNotes(true)}
              disabled={loading}
              className="px-2.5 py-1 bg-red-50 text-red-700 border border-red-200 rounded text-xs font-medium hover:bg-red-100 transition-colors disabled:opacity-50"
            >
              Ablehnen
            </button>
          </>
        )}

        {currentStatus === "rejected" && (
          <button
            type="button"
            onClick={() => updateStatus("pending")}
            disabled={loading}
            className="px-2.5 py-1 bg-slate-100 text-slate-700 border border-slate-200 rounded text-xs font-medium hover:bg-slate-200 transition-colors disabled:opacity-50"
          >
            Wieder öffnen
          </button>
        )}

        {(currentStatus === "eda_sent" || currentStatus === "converted") && (
          <button
            type="button"
            onClick={handleMarkActive}
            disabled={loading}
            className="px-2.5 py-1 bg-blue-600 text-white rounded text-xs font-medium hover:bg-blue-700 transition-colors disabled:opacity-50"
          >
            {loading ? "..." : "Als aktiv markieren"}
          </button>
        )}

        {currentStatus === "active" && (
          <span className="text-xs text-slate-400 italic">Aktiv</span>
        )}

        <button
          type="button"
          onClick={handleDelete}
          disabled={loading}
          title="Antrag löschen"
          className="px-2.5 py-1 bg-slate-100 text-slate-500 border border-slate-200 rounded text-xs font-medium hover:bg-red-50 hover:text-red-600 hover:border-red-200 transition-colors disabled:opacity-50"
        >
          Löschen
        </button>
      </div>
    </>
  );
}
