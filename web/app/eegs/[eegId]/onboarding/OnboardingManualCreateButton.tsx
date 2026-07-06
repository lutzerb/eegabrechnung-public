"use client";

import { useState } from "react";
import OnboardingManualCreateModal from "./OnboardingManualCreateModal";

interface Props {
  eegId: string;
}

export default function OnboardingManualCreateButton({ eegId }: Props) {
  const [open, setOpen] = useState(false);
  const [successEmail, setSuccessEmail] = useState<string | null>(null);

  function handleSuccess(email: string) {
    setOpen(false);
    setSuccessEmail(email);
    setTimeout(() => setSuccessEmail(null), 8000);
  }

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="flex items-center gap-1.5 px-3 py-2 bg-green-600 text-white rounded-lg text-sm font-medium hover:bg-green-700 transition-colors"
      >
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
        Manuell anlegen
      </button>

      {successEmail && (
        <div className="fixed bottom-6 right-6 z-50 bg-green-50 border border-green-200 rounded-xl shadow-lg p-4 max-w-sm text-sm text-green-800">
          <strong>Bestätigungs-E-Mail gesendet</strong>
          <p className="mt-1 text-green-700">
            An <span className="font-mono">{successEmail}</span> wurde eine E-Mail mit den Daten und einem Bestätigungslink gesendet.
          </p>
        </div>
      )}

      {open && (
        <OnboardingManualCreateModal
          eegId={eegId}
          onClose={() => setOpen(false)}
          onSuccess={handleSuccess}
        />
      )}
    </>
  );
}
