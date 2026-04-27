"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG } from "@/lib/api";
import { formatIBAN } from "@/lib/validation";
import Link from "next/link";
import OnboardingActions from "../OnboardingActions";
import OnboardingEditButton from "../OnboardingEditButton";

interface Props {
  params: Promise<{ eegId: string; requestId: string }>;
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
  meter_points: Array<{ zaehlpunkt: string; direction: string }>;
  beitritts_datum?: string;
  contract_accepted_at?: string;
  contract_ip: string;
  admin_notes: string;
  converted_member_id?: string;
  created_at: string;
  updated_at: string;
}

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { label: string; cls: string }> = {
    pending: { label: "In Prüfung", cls: "bg-yellow-100 text-yellow-800" },
    approved: { label: "Genehmigt", cls: "bg-blue-100 text-blue-800" },
    rejected: { label: "Abgelehnt", cls: "bg-red-100 text-red-800" },
    converted: { label: "Aufgenommen", cls: "bg-green-100 text-green-800" },
  };
  const cfg = map[status] || { label: status, cls: "bg-slate-100 text-slate-700" };
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded text-xs font-medium ${cfg.cls}`}>
      {cfg.label}
    </span>
  );
}

function formatDate(dateStr?: string): string {
  if (!dateStr) return "—";
  try {
    return new Date(dateStr).toLocaleDateString("de-AT", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return dateStr;
  }
}

function formatDateOnly(dateStr?: string): string {
  if (!dateStr) return "—";
  try {
    return new Date(dateStr).toLocaleDateString("de-AT", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
    });
  } catch {
    return dateStr;
  }
}

const memberTypeLabel: Record<string, string> = {
  CONSUMER: "Verbraucher (Bezug)",
  PRODUCER: "Erzeuger (Einspeisung)",
  PROSUMER: "Prosumer (Bezug & Einspeisung)",
};

const businessRoleLabel: Record<string, string> = {
  privat:               "Privatperson",
  kleinunternehmer:     "Kleinunternehmer",
  verein:               "Verein",
  landwirt_pauschaliert:"Landwirt (pauschaliert, § 22 UStG)",
  landwirt:             "Landwirt (buchführungspflichtig)",
  unternehmen:          "Unternehmen",
  gemeinde_bga:         "Gemeinde (BgA)",
  gemeinde_hoheitlich:  "Gemeinde (hoheitlich)",
};

const directionLabel: Record<string, string> = {
  CONSUMPTION: "Bezug",
  GENERATION: "Einspeisung",
};

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="grid grid-cols-3 gap-4 py-2.5 border-b border-slate-100 last:border-0">
      <dt className="text-sm text-slate-500">{label}</dt>
      <dd className="col-span-2 text-sm text-slate-900">{value || "—"}</dd>
    </div>
  );
}

export default async function OnboardingDetailPage({ params }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId, requestId } = await params;

  let eeg = null;
  let req: OnboardingRequest | null = null;

  try {
    const [eegData, res] = await Promise.all([
      getEEG(session.accessToken!, eegId),
      fetch(`${API}/api/v1/eegs/${eegId}/onboarding/${requestId}`, {
        headers: { Authorization: `Bearer ${session.accessToken}` },
        cache: "no-store",
      }),
    ]);
    eeg = eegData;
    if (res.ok) req = await res.json();
  } catch {
    // ignore
  }

  if (!req) {
    return (
      <div className="p-8">
        <p className="text-slate-500">Antrag nicht gefunden.</p>
        <Link href={`/eegs/${eegId}/onboarding`} className="text-blue-600 hover:underline text-sm mt-2 inline-block">
          ← Zurück zur Übersicht
        </Link>
      </div>
    );
  }

  const fullName = [req.name1, req.name2].filter(Boolean).join(" ");

  return (
    <div className="p-8 max-w-3xl">
      {/* Breadcrumb */}
      <div className="mb-6 flex items-center gap-2 text-sm text-slate-500">
        <Link href="/eegs" className="hover:text-slate-700">Energiegemeinschaften</Link>
        <span className="text-slate-300">/</span>
        <Link href={`/eegs/${eegId}`} className="hover:text-slate-700">{eeg?.name || eegId}</Link>
        <span className="text-slate-300">/</span>
        <Link href={`/eegs/${eegId}/onboarding`} className="hover:text-slate-700">Onboarding</Link>
        <span className="text-slate-300">/</span>
        <span className="text-slate-900 font-medium">{fullName}</span>
      </div>

      {/* Header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">{fullName}</h1>
          <p className="text-slate-500 text-sm mt-0.5">Beitrittsantrag · eingegangen {formatDate(req.created_at)}</p>
        </div>
        <div className="flex items-center gap-3">
          <StatusBadge status={req.status} />
          <OnboardingEditButton eegId={eegId} req={req} />
          {req.status !== "converted" && req.status !== "rejected" && (
            <OnboardingActions
                eegId={eegId}
                requestId={req.id}
                currentStatus={req.status}
                eegNetzbetreiberId={eeg?.eda_netzbetreiber_id}
                meterPoints={req.meter_points}
              />
          )}
        </div>
      </div>

      {/* Persönliche Daten */}
      <div className="bg-white rounded-xl border border-slate-200 p-5 mb-4">
        <h2 className="text-sm font-semibold text-slate-700 mb-3">Persönliche Daten</h2>
        <dl>
          <Row label="Name" value={fullName} />
          <Row label="E-Mail" value={<a href={`mailto:${req.email}`} className="text-blue-600 hover:underline">{req.email}</a>} />
          <Row label="Telefon" value={req.phone} />
          <Row label="Mitgliedstyp" value={memberTypeLabel[req.member_type] || req.member_type} />
          <Row label="Unternehmensart" value={businessRoleLabel[req.business_role] || req.business_role || "Privatperson"} />
          <Row label="USt-pflichtig" value={req.use_vat ? "Ja" : "Nein"} />
          {req.uid_nummer && <Row label="UID-Nummer" value={<span className="font-mono">{req.uid_nummer}</span>} />}
          <Row label="Gewünschter Beitritt" value={formatDateOnly(req.beitritts_datum)} />
        </dl>
      </div>

      {/* Adresse & Bankdaten */}
      <div className="bg-white rounded-xl border border-slate-200 p-5 mb-4">
        <h2 className="text-sm font-semibold text-slate-700 mb-3">Adresse & Bankdaten</h2>
        <dl>
          <Row label="Straße" value={req.strasse} />
          <Row label="PLZ / Ort" value={[req.plz, req.ort].filter(Boolean).join(" ")} />
          <Row label="IBAN" value={<span className="font-mono">{req.iban ? formatIBAN(req.iban) : "—"}</span>} />
          <Row label="BIC" value={req.bic} />
        </dl>
      </div>

      {/* Zählpunkte */}
      <div className="bg-white rounded-xl border border-slate-200 p-5 mb-4">
        <h2 className="text-sm font-semibold text-slate-700 mb-3">Zählpunkte</h2>
        {req.meter_points && req.meter_points.length > 0 ? (
          <div className="space-y-2">
            {req.meter_points.map((mp, i) => (
              <div key={i} className="flex items-center justify-between bg-slate-50 rounded-lg px-4 py-2.5">
                <span className="font-mono text-sm text-slate-800">{mp.zaehlpunkt || "—"}</span>
                <span className="text-xs text-slate-500 bg-slate-200 px-2 py-0.5 rounded">
                  {directionLabel[mp.direction] || mp.direction}
                </span>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm text-slate-400">Keine Zählpunkte angegeben.</p>
        )}
      </div>

      {/* Antragsstatus & Admin */}
      <div className="bg-white rounded-xl border border-slate-200 p-5">
        <h2 className="text-sm font-semibold text-slate-700 mb-3">Antragsstatus</h2>
        <dl>
          <Row label="Status" value={<StatusBadge status={req.status} />} />
          <Row label="Vertrag angenommen" value={formatDate(req.contract_accepted_at)} />
          <Row label="IP-Adresse" value={<span className="font-mono text-xs">{req.contract_ip}</span>} />
          <Row label="Admin-Notiz" value={req.admin_notes} />
          {req.converted_member_id && (
            <Row label="Mitglied" value={
              <Link href={`/eegs/${eegId}/members/${req.converted_member_id}`} className="text-blue-600 hover:underline">
                Mitglied ansehen →
              </Link>
            } />
          )}
        </dl>
      </div>

      <div className="mt-4">
        <Link href={`/eegs/${eegId}/onboarding`} className="text-sm text-slate-500 hover:text-slate-700">
          ← Zurück zur Übersicht
        </Link>
      </div>
    </div>
  );
}
