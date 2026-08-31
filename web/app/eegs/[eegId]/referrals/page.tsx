import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import GrantBonusButton from "./GrantBonusButton";
import CancelBonusButton from "./CancelBonusButton";

interface Props {
  params: Promise<{ eegId: string }>;
}

interface EligibleReferral {
  referrer_member_id: string;
  referrer_name: string;
  referred_member_id: string;
  referred_name: string;
  joined_at?: string;
}

interface ReferralBonus {
  id: string;
  referrer_member_id: string;
  referrer_name: string;
  referred_member_id: string;
  referred_name: string;
  amount_eur: number;
  status: "pending" | "applied" | "cancelled";
  granted_at: string;
  applied_invoice_id?: string;
  applied_at?: string;
}

function formatDate(iso?: string) {
  if (!iso) return "–";
  return new Date(iso).toLocaleDateString("de-AT");
}

function formatEur(v: number) {
  return new Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" }).format(v);
}

const STATUS_LABELS: Record<string, string> = {
  pending: "Ausstehend",
  applied: "Verrechnet",
  cancelled: "Storniert",
};

export default async function ReferralsPage({ params }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId } = await params;
  const API = process.env.API_INTERNAL_URL || "http://localhost:8080";
  const headers = { Authorization: `Bearer ${session.accessToken}` };

  const [eligibleRes, bonusesRes, eegRes] = await Promise.all([
    fetch(`${API}/api/v1/eegs/${eegId}/referrals/eligible`, { headers, cache: "no-store" }),
    fetch(`${API}/api/v1/eegs/${eegId}/referrals`, { headers, cache: "no-store" }),
    fetch(`${API}/api/v1/eegs/${eegId}`, { headers, cache: "no-store" }),
  ]);

  const eligible: EligibleReferral[] = eligibleRes.ok ? await eligibleRes.json() : [];
  const bonuses: ReferralBonus[] = bonusesRes.ok ? await bonusesRes.json() : [];
  const eeg = eegRes.ok ? await eegRes.json() : null;
  const defaultAmount = eeg?.referral_bonus_eur ?? 5;

  return (
    <div className="max-w-4xl mx-auto py-8 px-4">
      <div className="mb-6">
        <h1 className="text-xl font-bold text-slate-900">Werbeprämien</h1>
        <p className="text-sm text-slate-500 mt-1">
          Mitglieder werben Mitglieder — erfolgreiche Werbungen manuell mit einer Prämie
          belohnen. Die Prämie wird auf die nächste Rechnung des werbenden Mitglieds angewendet.
        </p>
      </div>

      <div className="bg-white rounded-xl border border-slate-200 mb-6">
        <div className="px-5 py-4 border-b border-slate-100">
          <h2 className="text-sm font-semibold text-slate-900">
            Noch nicht vergütete Werbungen {eligible.length > 0 && `(${eligible.length})`}
          </h2>
        </div>
        {eligible.length === 0 ? (
          <p className="px-5 py-6 text-sm text-slate-500">
            Keine offenen Werbungen. Mitglieder finden ihren persönlichen Einladungslink im
            Mitglieder-Portal unter &bdquo;Mitglieder werben&ldquo;.
          </p>
        ) : (
          <div className="divide-y divide-slate-100">
            {eligible.map((e) => (
              <div key={e.referred_member_id} className="px-5 py-4 flex items-center justify-between gap-4">
                <div className="min-w-0">
                  <p className="text-sm text-slate-900">
                    <span className="font-medium">{e.referrer_name}</span> hat{" "}
                    <span className="font-medium">{e.referred_name}</span> geworben
                  </p>
                  <p className="text-xs text-slate-500 mt-0.5">Beigetreten am {formatDate(e.joined_at)}</p>
                </div>
                <GrantBonusButton eegId={eegId} referredMemberId={e.referred_member_id} defaultAmount={defaultAmount} />
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="bg-white rounded-xl border border-slate-200">
        <div className="px-5 py-4 border-b border-slate-100">
          <h2 className="text-sm font-semibold text-slate-900">Vergebene Prämien</h2>
        </div>
        {bonuses.length === 0 ? (
          <p className="px-5 py-6 text-sm text-slate-500">Noch keine Prämien vergeben.</p>
        ) : (
          <div className="divide-y divide-slate-100">
            {bonuses.map((b) => (
              <div key={b.id} className="px-5 py-4 flex items-center justify-between gap-4">
                <div className="min-w-0">
                  <p className="text-sm text-slate-900">
                    <span className="font-medium">{b.referrer_name}</span> für{" "}
                    <span className="font-medium">{b.referred_name}</span> — {formatEur(b.amount_eur)}
                  </p>
                  <p className="text-xs text-slate-500 mt-0.5">
                    Vergeben am {formatDate(b.granted_at)}
                    {b.status === "applied" && ` · Verrechnet am ${formatDate(b.applied_at)}`}
                  </p>
                </div>
                <div className="flex items-center gap-3 flex-shrink-0">
                  <span
                    className={`text-xs font-medium px-2 py-1 rounded-md ${
                      b.status === "pending"
                        ? "bg-amber-50 text-amber-700"
                        : b.status === "applied"
                          ? "bg-green-50 text-green-700"
                          : "bg-slate-100 text-slate-500"
                    }`}
                  >
                    {STATUS_LABELS[b.status] || b.status}
                  </span>
                  {b.status === "pending" && (
                    <CancelBonusButton eegId={eegId} referredMemberId={b.referred_member_id} />
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
