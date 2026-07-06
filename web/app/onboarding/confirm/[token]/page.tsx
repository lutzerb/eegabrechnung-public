import { notFound } from "next/navigation";
import ConfirmForm from "./ConfirmForm";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

interface Props {
  params: Promise<{ token: string }>;
}

export default async function OnboardingConfirmPage({ params }: Props) {
  const { token } = await params;

  const reqRes = await fetch(`${API}/api/v1/public/onboarding/status/${token}`, {
    cache: "no-store",
  });

  if (reqRes.status === 404) return notFound();
  if (reqRes.status === 410) {
    return (
      <div className="max-w-lg mx-auto px-4 py-16 text-center">
        <h1 className="text-2xl font-bold text-slate-900 mb-3">Link abgelaufen</h1>
        <p className="text-slate-500">
          Dieser Bestätigungslink ist nicht mehr gültig (30 Tage abgelaufen).
          Bitte wenden Sie sich an die Energiegemeinschaft.
        </p>
      </div>
    );
  }
  if (!reqRes.ok) return notFound();

  const req = await reqRes.json();

  if (req.status !== "admin_created") {
    return (
      <div className="max-w-lg mx-auto px-4 py-16 text-center">
        <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
          <svg className="w-8 h-8 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
        </div>
        <h1 className="text-2xl font-bold text-slate-900 mb-3">Bereits bestätigt</h1>
        <p className="text-slate-500">
          Ihre Anmeldung wurde bereits bestätigt und wird bearbeitet.
        </p>
      </div>
    );
  }

  const infoRes = await fetch(`${API}/api/v1/public/eegs/${req.eeg_id}/info`, {
    cache: "no-store",
  });
  const eeg = infoRes.ok ? await infoRes.json() : { name: "", onboarding_contract_text: "", documents: [] };

  return (
    <main className="min-h-screen bg-slate-50 py-8">
      <ConfirmForm token={token} req={req} eeg={eeg} eegId={req.eeg_id} />
    </main>
  );
}
