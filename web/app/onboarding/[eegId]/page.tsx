import OnboardingForm from "./OnboardingForm";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

interface PublicDocument {
  id: string;
  title: string;
  filename: string;
  mime_type: string;
}

interface Props {
  params: Promise<{ eegId: string }>;
  searchParams: Promise<{ ev?: string }>;
}

export default async function OnboardingPage({ params, searchParams }: Props) {
  const { eegId } = await params;
  const { ev } = await searchParams;
  let eegName = "Energiegemeinschaft";
  let eegFound = true;
  let contractText = "";
  let documents: PublicDocument[] = [];

  try {
    const res = await fetch(
      `${API}/api/v1/public/eegs/${eegId}/info`,
      { cache: "no-store" }
    );
    if (res.ok) {
      const data = await res.json();
      eegName = data.name || eegName;
      contractText = data.onboarding_contract_text || "";
      documents = data.documents || [];
    } else {
      eegFound = false;
    }
  } catch {
    // network error — still render form with default name
  }

  if (!eegFound) {
    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center px-4">
        <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-8 max-w-md w-full text-center">
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
          <h1 className="text-xl font-bold text-slate-900 mb-2">
            Energiegemeinschaft nicht gefunden
          </h1>
          <p className="text-slate-500 text-sm">
            Der angegebene Beitrittslink ist ungültig oder die
            Energiegemeinschaft existiert nicht mehr.
          </p>
        </div>
      </div>
    );
  }

  // Handle email verification token from ?ev= query param
  let verifiedEmail: string | undefined;
  let verifiedName1: string | undefined;
  let verifiedName2: string | undefined;
  let evInvalid = false;

  if (ev) {
    try {
      const verifyRes = await fetch(
        `${API}/api/v1/public/eegs/${eegId}/onboarding/verify/${ev}`,
        { method: "POST", cache: "no-store" }
      );
      if (verifyRes.ok) {
        const data = await verifyRes.json();
        verifiedEmail = data.email;
        verifiedName1 = data.name1;
        verifiedName2 = data.name2;
      } else {
        evInvalid = true;
      }
    } catch {
      evInvalid = true;
    }
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <div className="max-w-2xl mx-auto py-10 px-4">
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
          <h1 className="text-2xl font-bold text-slate-900">{eegName}</h1>
          <p className="text-slate-500 mt-1">Beitrittsantrag</p>
        </div>

        {evInvalid && (
          <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-xl text-sm text-red-700 flex items-start gap-2">
            <svg className="w-4 h-4 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <span>
              <strong>Link ungültig oder abgelaufen.</strong> Bitte starten Sie den Prozess neu und fordern Sie einen neuen Bestätigungslink an.
            </span>
          </div>
        )}

        <OnboardingForm
          eegId={eegId}
          eegName={eegName}
          contractText={contractText}
          documents={documents}
          verifiedEmail={verifiedEmail}
          verifiedName1={verifiedName1}
          verifiedName2={verifiedName2}
        />
        <p className="text-center text-xs text-slate-400 mt-6">
          Ihre Daten werden verschlüsselt übertragen und gemäß DSGVO
          verarbeitet.
        </p>
      </div>
    </div>
  );
}
