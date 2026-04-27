"use server";

import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG } from "@/lib/api";
import { formatIBAN } from "@/lib/validation";
import Link from "next/link";
import CopyLinkButton from "./CopyLinkButton";
import OnboardingActions from "./OnboardingActions";

interface Props {
  params: Promise<{ eegId: string }>;
  searchParams: Promise<{ status?: string }>;
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
  meter_points: Array<{ zaehlpunkt: string; direction: string }>;
  contract_accepted_at?: string;
  admin_notes: string;
  converted_member_id?: string;
  created_at: string;
  updated_at: string;
}

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

async function fetchOnboarding(
  token: string,
  eegId: string
): Promise<OnboardingRequest[]> {
  const res = await fetch(`${API}/api/v1/eegs/${eegId}/onboarding`, {
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
  });
  if (!res.ok) return [];
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { label: string; cls: string }> = {
    pending: { label: "In Prüfung", cls: "bg-yellow-100 text-yellow-800" },
    approved: { label: "Genehmigt", cls: "bg-blue-100 text-blue-800" },
    rejected: { label: "Abgelehnt", cls: "bg-red-100 text-red-800" },
    converted: { label: "Aufgenommen", cls: "bg-green-100 text-green-800" },
    eda_sent: { label: "EDA gesendet", cls: "bg-blue-100 text-blue-800" },
    active: { label: "Aktiv", cls: "bg-green-100 text-green-800" },
  };
  const cfg = map[status] || {
    label: status,
    cls: "bg-slate-100 text-slate-700",
  };
  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${cfg.cls}`}
    >
      {cfg.label}
    </span>
  );
}

function MemberTypeBadge({ type }: { type: string }) {
  const map: Record<string, string> = {
    CONSUMER: "Verbraucher",
    PRODUCER: "Erzeuger",
    PROSUMER: "Prosumer",
  };
  return <span className="text-slate-700">{map[type] || type}</span>;
}

function formatDate(dateStr: string): string {
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

const STATUS_FILTERS = [
  { value: "", label: "Aktive" },
  { value: "pending", label: "In Prüfung" },
  { value: "approved", label: "Genehmigt" },
  { value: "converted", label: "Aufgenommen" },
  { value: "eda_sent", label: "EDA gesendet" },
  { value: "active", label: "Aktiv" },
  { value: "rejected", label: "Abgelehnt" },
  { value: "alle", label: "Alle" },
];

export default async function OnboardingAdminPage({ params, searchParams }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId } = await params;
  const { status: spStatus } = await searchParams;
  const statusFilter = spStatus || "";

  let eeg = null;
  let requests: OnboardingRequest[] = [];

  try {
    [eeg, requests] = await Promise.all([
      getEEG(session.accessToken!, eegId),
      fetchOnboarding(session.accessToken!, eegId),
    ]);
  } catch {
    // ignore
  }

  const filtered =
    statusFilter === "alle"
      ? requests
      : statusFilter
      ? requests.filter((r) => r.status === statusFilter)
      : requests.filter((r) => r.status !== "rejected"); // default: hide rejected

  const counts = requests.reduce(
    (acc, r) => {
      acc[r.status] = (acc[r.status] || 0) + 1;
      return acc;
    },
    {} as Record<string, number>
  );

  const onboardingURL =
    typeof window !== "undefined"
      ? `${window.location.origin}/onboarding/${eegId}`
      : `/onboarding/${eegId}`;
  // We use a placeholder that will be resolved client-side via CopyLinkButton
  const publicURL = `[Basis-URL]/onboarding/${eegId}`;

  return (
    <div className="p-8">
      {/* Breadcrumb */}
      <div className="mb-6 flex items-center gap-2 text-sm text-slate-500">
        <Link href="/eegs" className="hover:text-slate-700">
          Energiegemeinschaften
        </Link>
        <span className="text-slate-300">/</span>
        <Link href={`/eegs/${eegId}`} className="hover:text-slate-700">
          {eeg?.name || eegId}
        </Link>
        <span className="text-slate-300">/</span>
        <span className="text-slate-900 font-medium">Onboarding</span>
      </div>

      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Onboarding</h1>
          <p className="text-slate-500 text-sm mt-0.5">
            Beitrittsanträge verwalten und neue Mitglieder aufnehmen
          </p>
        </div>
        <div className="flex gap-2 text-xs text-slate-500">
          {Object.entries({ pending: "In Prüfung", approved: "Genehmigt", converted: "Aufgenommen", eda_sent: "EDA gesendet", active: "Aktiv" }).map(([s, l]) =>
            counts[s] ? (
              <span key={s} className="bg-slate-100 px-2 py-1 rounded">
                {l}: <strong>{counts[s]}</strong>
              </span>
            ) : null
          )}
        </div>
      </div>

      {/* Onboarding link */}
      <div className="bg-white rounded-xl border border-slate-200 p-4 mb-6">
        <p className="text-sm font-medium text-slate-700 mb-2">
          Öffentlicher Beitrittslink (für neue Mitglieder):
        </p>
        <OnboardingLinkCopy eegId={eegId} />
      </div>

      {/* Status filter */}
      <div className="flex gap-2 mb-4 flex-wrap">
        {STATUS_FILTERS.map((f) => (
          <Link
            key={f.value}
            href={
              f.value
                ? `/eegs/${eegId}/onboarding?status=${f.value}`
                : `/eegs/${eegId}/onboarding`
            }
            className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
              statusFilter === f.value
                ? "bg-blue-600 text-white"
                : "bg-white border border-slate-200 text-slate-600 hover:bg-slate-50"
            }`}
          >
            {f.label}
            {f.value === "" && requests.filter((r) => r.status !== "rejected").length > 0 && (
              <span className="ml-1.5 bg-slate-200 text-slate-600 text-xs px-1.5 py-0.5 rounded-full">
                {requests.filter((r) => r.status !== "rejected").length}
              </span>
            )}
            {f.value === "alle" && requests.length > 0 && (
              <span className="ml-1.5 bg-slate-200 text-slate-600 text-xs px-1.5 py-0.5 rounded-full">
                {requests.length}
              </span>
            )}
            {f.value && counts[f.value] ? (
              <span className="ml-1.5 bg-white/20 text-xs px-1.5 py-0.5 rounded-full">
                {counts[f.value]}
              </span>
            ) : null}
          </Link>
        ))}
      </div>

      {/* Table */}
      {filtered.length === 0 ? (
        <div className="bg-white rounded-xl border border-slate-200 p-12 text-center">
          <div className="w-12 h-12 bg-slate-100 rounded-full flex items-center justify-center mx-auto mb-3">
            <svg
              className="w-6 h-6 text-slate-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"
              />
            </svg>
          </div>
          <p className="text-slate-500 font-medium">Keine Anträge</p>
          <p className="text-slate-400 text-sm mt-1">
            {statusFilter
              ? "Keine Anträge mit diesem Status."
              : "Noch keine Beitrittsanträge eingegangen."}
          </p>
        </div>
      ) : (
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-slate-600">
                  Name
                </th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">
                  E-Mail
                </th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">
                  Typ
                </th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">
                  Zählpunkte
                </th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">
                  Status
                </th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">
                  Datum
                </th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">
                  Aktionen
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {filtered.map((req) => (
                <tr key={req.id} className="hover:bg-slate-50 transition-colors">
                  <td className="px-4 py-3">
                    <Link href={`/eegs/${eegId}/onboarding/${req.id}`} className="font-medium text-blue-700 hover:underline">
                      {req.name1}
                    </Link>
                    {req.name2 && (
                      <div className="text-xs text-slate-500">{req.name2}</div>
                    )}
                    {req.iban && (
                      <div className="text-xs text-slate-400 font-mono">
                        {formatIBAN(req.iban)}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-3 text-slate-600">{req.email}</td>
                  <td className="px-4 py-3">
                    <MemberTypeBadge type={req.member_type} />
                  </td>
                  <td className="px-4 py-3">
                    {req.meter_points && req.meter_points.length > 0 ? (
                      <div className="space-y-0.5">
                        {req.meter_points.map((mp, i) => (
                          <div
                            key={i}
                            className="text-xs font-mono text-slate-600 truncate max-w-36"
                            title={mp.zaehlpunkt}
                          >
                            {mp.zaehlpunkt || "—"}
                          </div>
                        ))}
                      </div>
                    ) : (
                      <span className="text-slate-400">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={req.status} />
                    {req.admin_notes && (
                      <div
                        className="text-xs text-slate-400 mt-0.5 truncate max-w-32"
                        title={req.admin_notes}
                      >
                        {req.admin_notes}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-3 text-slate-500">
                    {formatDate(req.created_at)}
                  </td>
                  <td className="px-4 py-3">
                    <OnboardingActions
                      eegId={eegId}
                      requestId={req.id}
                      currentStatus={req.status}
                      eegNetzbetreiberId={eeg?.eda_netzbetreiber_id}
                      meterPoints={req.meter_points}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// Small server component that renders a client copy button
function OnboardingLinkCopy({ eegId }: { eegId: string }) {
  // The URL will be shown as a relative path; CopyLinkButton handles clipboard
  // We use a data attribute so the client can reconstruct the full URL
  return (
    <div data-eeg-id={eegId}>
      <OnboardingLinkClientWrapper eegId={eegId} />
    </div>
  );
}

function OnboardingLinkClientWrapper({ eegId }: { eegId: string }) {
  // We pass a relative URL; the client component will use window.location to build full URL
  return <CopyLinkButton url={`/onboarding/${eegId}`} />;
}
