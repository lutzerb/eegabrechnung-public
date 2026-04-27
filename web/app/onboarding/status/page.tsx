import ResendForm from "./ResendForm";
import { byZaehlpunkt } from "@/lib/netzbetreiber";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

interface Props {
  searchParams: Promise<{ token?: string }>;
}

interface OnboardingRequest {
  id: string;
  eeg_id: string;
  status: string;
  name1: string;
  name2: string;
  email: string;
  member_type: string;
  meter_points: Array<{ zaehlpunkt: string; direction: string }>;
  admin_notes: string;
  converted_member_id?: string;
  created_at: string;
  updated_at: string;
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { label: string; cls: string }> = {
    pending: {
      label: "In Prüfung",
      cls: "bg-yellow-100 text-yellow-800 border border-yellow-200",
    },
    approved: {
      label: "Genehmigt",
      cls: "bg-blue-100 text-blue-800 border border-blue-200",
    },
    rejected: {
      label: "Abgelehnt",
      cls: "bg-red-100 text-red-800 border border-red-200",
    },
    converted: {
      label: "Aufgenommen",
      cls: "bg-green-100 text-green-800 border border-green-200",
    },
    eda_sent: {
      label: "EDA gesendet",
      cls: "bg-blue-100 text-blue-800 border border-blue-200",
    },
    active: {
      label: "Aktiv",
      cls: "bg-green-100 text-green-800 border border-green-200",
    },
  };
  const cfg = map[status] || {
    label: status,
    cls: "bg-slate-100 text-slate-700 border border-slate-200",
  };
  return (
    <span
      className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${cfg.cls}`}
    >
      {cfg.label}
    </span>
  );
}

function CheckIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2.5}
        d="M5 13l4 4L19 7"
      />
    </svg>
  );
}

function ProgressSteps({
  status,
  meterPoints,
}: {
  status: string;
  meterPoints: Array<{ zaehlpunkt: string; direction: string }>;
}) {
  const isStep2Done = ["eda_sent", "converted", "active", "approved"].includes(status);
  const isStep3Active = status === "eda_sent" || status === "converted";
  const isStep3Done = status === "active";
  const isStep4Done = status === "active";

  // Detect NB from meter points
  let nbInfo: ReturnType<typeof byZaehlpunkt> | undefined;
  for (const mp of meterPoints) {
    const found = byZaehlpunkt(mp.zaehlpunkt);
    if (found) {
      nbInfo = found;
      break;
    }
  }

  const steps = [
    {
      label: "Antrag eingereicht",
      done: true,
      active: false,
    },
    {
      label: "Von Energiegemeinschaft geprüft",
      done: isStep2Done,
      active: !isStep2Done && status === "pending",
    },
    {
      label: "Im Netzbetreiber-Portal aktivieren",
      done: isStep3Done,
      active: isStep3Active,
    },
    {
      label: "Datenaustausch aktiv",
      done: isStep4Done,
      active: false,
    },
  ];

  return (
    <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-6">
      <h3 className="font-semibold text-slate-900 mb-5">Ihr Fortschritt</h3>
      <ol className="space-y-0">
        {steps.map((step, i) => {
          const isLast = i === steps.length - 1;
          return (
            <li key={i}>
              <div className="flex gap-3">
                {/* Icon column */}
                <div className="flex flex-col items-center">
                  <div
                    className={`w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 ${
                      step.done
                        ? "bg-green-600 text-white"
                        : step.active
                          ? "bg-blue-600 text-white ring-4 ring-blue-100 animate-pulse"
                          : "bg-slate-200 text-slate-400"
                    }`}
                  >
                    {step.done ? (
                      <CheckIcon className="w-4 h-4" />
                    ) : (
                      <span className="text-xs font-bold">{i + 1}</span>
                    )}
                  </div>
                  {!isLast && (
                    <div
                      className={`w-0.5 h-6 mt-1 ${step.done ? "bg-green-300" : "bg-slate-200"}`}
                    />
                  )}
                </div>

                {/* Content column */}
                <div className={`pb-${isLast ? "0" : "4"} flex-1`}>
                  <p
                    className={`text-sm font-medium pt-1 ${
                      step.done
                        ? "text-green-700"
                        : step.active
                          ? "text-blue-700"
                          : "text-slate-400"
                    }`}
                  >
                    {step.label}
                    {step.done && i === 2 && (
                      <span className="ml-2 text-xs text-green-600 font-normal">
                        Ihr Zählpunkt ist aktiviert
                      </span>
                    )}
                  </p>

                  {/* NB portal block for step 3 when active/eda_sent */}
                  {i === 2 && isStep3Active && (
                    <div className="mt-3 mb-4 bg-blue-50 border border-blue-200 rounded-xl p-4 text-sm">
                      <p className="text-blue-900 font-medium mb-2">
                        Bitte melden Sie sich im Portal Ihres Netzbetreibers an und:
                      </p>
                      <ol className="space-y-1.5 mb-3">
                        <li className="flex gap-2 text-blue-800">
                          <span className="font-bold">①</span>
                          <span>Aktivieren Sie den 15-Minuten-Takt für Ihren Smartmeter</span>
                        </li>
                        <li className="flex gap-2 text-blue-800">
                          <span className="font-bold">②</span>
                          <span>Erteilen Sie die Datenfreigabe für die Energiegemeinschaft</span>
                        </li>
                      </ol>

                      {nbInfo && (
                        <>
                          <a
                            href={nbInfo.portalUrl}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-1.5 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors"
                          >
                            Link zu {nbInfo.portalName} öffnen
                            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                            </svg>
                          </a>
                          <p className="mt-2 text-xs text-slate-500">{nbInfo.hinweis}</p>
                        </>
                      )}
                    </div>
                  )}
                </div>
              </div>
            </li>
          );
        })}
      </ol>
    </div>
  );
}

function formatDate(dateStr: string): string {
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

export default async function OnboardingStatusPage({ searchParams }: Props) {
  const { token } = await searchParams;

  if (!token) {
    return (
      <div className="min-h-screen bg-slate-50">
        <div className="max-w-lg mx-auto py-10 px-4">
          <div className="text-center mb-8">
            <div className="w-14 h-14 bg-blue-600 rounded-xl flex items-center justify-center mx-auto mb-4">
              <svg
                className="w-7 h-7 text-white"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M13 10V3L4 14h7v7l9-11h-7z"
                />
              </svg>
            </div>
            <h1 className="text-2xl font-bold text-slate-900">
              Antragsstatus
            </h1>
            <p className="text-slate-500 mt-1 text-sm">
              Geben Sie Ihre E-Mail-Adresse ein, um einen neuen Status-Link zu
              erhalten.
            </p>
          </div>
          <ResendForm />
        </div>
      </div>
    );
  }

  let req: OnboardingRequest | null = null;
  let fetchError: string | null = null;
  let isExpired = false;

  try {
    const res = await fetch(
      `${API}/api/v1/public/onboarding/status/${token}`,
      { cache: "no-store" }
    );
    if (res.status === 410) {
      isExpired = true;
    } else if (res.status === 404) {
      fetchError = "Dieser Link ist ungültig.";
    } else if (!res.ok) {
      fetchError = "Fehler beim Laden des Antragsstatus.";
    } else {
      req = await res.json();
    }
  } catch {
    fetchError = "Netzwerkfehler. Bitte versuchen Sie es später erneut.";
  }

  if (isExpired) {
    return (
      <div className="min-h-screen bg-slate-50">
        <div className="max-w-lg mx-auto py-10 px-4">
          <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-8 text-center">
            <div className="w-14 h-14 bg-orange-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg
                className="w-7 h-7 text-orange-600"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
            <h2 className="text-xl font-bold text-slate-900 mb-2">
              Link abgelaufen
            </h2>
            <p className="text-slate-500 text-sm mb-6">
              Dieser Status-Link ist abgelaufen. Sie können einen neuen Link
              anfordern.
            </p>
            <ResendForm />
          </div>
        </div>
      </div>
    );
  }

  if (fetchError || !req) {
    return (
      <div className="min-h-screen bg-slate-50">
        <div className="max-w-lg mx-auto py-10 px-4">
          <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-8 text-center">
            <div className="w-14 h-14 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg
                className="w-7 h-7 text-red-600"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
            </div>
            <h2 className="text-xl font-bold text-slate-900 mb-2">
              Antrag nicht gefunden
            </h2>
            <p className="text-slate-500 text-sm">
              {fetchError ||
                "Dieser Link ist ungültig. Bitte überprüfen Sie den Link in Ihrer E-Mail."}
            </p>
          </div>
        </div>
      </div>
    );
  }

  const memberTypeLabel: Record<string, string> = {
    CONSUMER: "Verbraucher",
    PRODUCER: "Erzeuger",
    PROSUMER: "Prosumer",
  };

  return (
    <div className="min-h-screen bg-slate-50">
      <div className="max-w-lg mx-auto py-10 px-4">
        <div className="text-center mb-8">
          <div className="w-14 h-14 bg-blue-600 rounded-xl flex items-center justify-center mx-auto mb-4">
            <svg
              className="w-7 h-7 text-white"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M13 10V3L4 14h7v7l9-11h-7z"
              />
            </svg>
          </div>
          <h1 className="text-2xl font-bold text-slate-900">Antragsstatus</h1>
        </div>

        <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-6 mb-4">
          <div className="flex items-start justify-between mb-4">
            <div>
              <h2 className="text-lg font-semibold text-slate-900">
                {req.name1}
                {req.name2 && (
                  <span className="text-slate-500 font-normal"> {req.name2}</span>
                )}
              </h2>
              <p className="text-sm text-slate-500">{req.email}</p>
            </div>
            <StatusBadge status={req.status} />
          </div>

          <div className="grid grid-cols-2 gap-3 text-sm mb-4">
            <div>
              <p className="text-slate-500 text-xs font-medium uppercase tracking-wide mb-0.5">
                Mitgliedstyp
              </p>
              <p className="text-slate-900">
                {memberTypeLabel[req.member_type] || req.member_type}
              </p>
            </div>
            <div>
              <p className="text-slate-500 text-xs font-medium uppercase tracking-wide mb-0.5">
                Eingereicht am
              </p>
              <p className="text-slate-900">{formatDate(req.created_at)}</p>
            </div>
          </div>

          {req.meter_points && req.meter_points.length > 0 && (
            <div className="mb-4">
              <p className="text-slate-500 text-xs font-medium uppercase tracking-wide mb-1.5">
                Zählpunkte
              </p>
              <div className="space-y-1">
                {req.meter_points.map((mp, i) => (
                  <div
                    key={i}
                    className="flex items-center gap-2 text-xs font-mono bg-slate-50 px-2 py-1.5 rounded border border-slate-200"
                  >
                    <span className="text-slate-700">{mp.zaehlpunkt}</span>
                    <span className="text-slate-400">·</span>
                    <span className="text-slate-500">
                      {mp.direction === "GENERATION"
                        ? "Einspeisung"
                        : "Bezug"}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {req.admin_notes && (
            <div className="bg-blue-50 rounded-lg p-3 text-sm">
              <p className="text-xs font-medium text-blue-700 mb-1">
                Hinweis der Energiegemeinschaft
              </p>
              <p className="text-blue-900">{req.admin_notes}</p>
            </div>
          )}
        </div>

        <ProgressSteps
          status={req.status}
          meterPoints={req.meter_points || []}
        />
      </div>
    </div>
  );
}
