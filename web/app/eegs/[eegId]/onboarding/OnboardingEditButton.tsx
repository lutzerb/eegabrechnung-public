"use client";

import { useState } from "react";
import OnboardingEditModal from "./OnboardingEditModal";

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
}

export default function OnboardingEditButton({ eegId, req }: Props) {
  const [open, setOpen] = useState(false);

  // Only allow editing for non-final statuses
  if (req.status === "converted" || req.status === "active") return null;

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="px-2.5 py-1 border border-slate-200 bg-white text-slate-600 rounded text-xs font-medium hover:bg-slate-50 transition-colors"
      >
        Bearbeiten
      </button>
      {open && (
        <OnboardingEditModal
          eegId={eegId}
          req={req}
          onClose={() => setOpen(false)}
          onSuccess={() => setOpen(false)}
        />
      )}
    </>
  );
}
