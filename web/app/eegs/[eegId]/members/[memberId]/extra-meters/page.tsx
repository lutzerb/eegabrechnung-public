import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG, getMember, listExtraMeters } from "@/lib/api";
import Link from "next/link";
import ExtraMetersClient from "@/components/extra-meters-client";

interface Props {
  params: Promise<{ eegId: string; memberId: string }>;
}

export default async function MemberExtraMetersPage({ params }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId, memberId } = await params;
  let eeg = null;
  let member = null;
  try { eeg = await getEEG(session.accessToken!, eegId); } catch {}
  try { member = await getMember(session.accessToken!, eegId, memberId); } catch {}

  const memberName = member ? `${member.name1} ${member.name2}`.trim() : memberId;

  const breadcrumb = (
    <div className="mb-6">
      <Link href="/eegs" className="text-sm text-slate-500 hover:text-slate-700">Energiegemeinschaften</Link>
      <span className="text-slate-400 mx-2">/</span>
      <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">{eeg?.name || eegId}</Link>
      <span className="text-slate-400 mx-2">/</span>
      <Link href={`/eegs/${eegId}/members/${memberId}`} className="text-sm text-slate-500 hover:text-slate-700">{memberName}</Link>
      <span className="text-slate-400 mx-2">/</span>
      <span className="text-sm text-slate-900 font-medium">Zusatzzähler</span>
    </div>
  );

  if (!eeg?.extra_meters_enabled) {
    return (
      <div className="p-8">
        {breadcrumb}
        <div className="bg-slate-50 rounded-xl border border-slate-200 p-8 text-center">
          <p className="text-slate-700 font-medium">Zusatzzähler sind für diese Energiegemeinschaft nicht aktiviert.</p>
          <p className="text-sm text-slate-500 mt-1">
            Aktivierbar in den{" "}
            <Link href={`/eegs/${eegId}/settings?tab=rechnungen`} className="text-blue-600 hover:underline">
              EEG-Einstellungen
            </Link>{" "}
            unter „Rechnungen“.
          </p>
        </div>
      </div>
    );
  }

  let extraMeters: Awaited<ReturnType<typeof listExtraMeters>> = [];
  try { extraMeters = await listExtraMeters(session.accessToken!, eegId, memberId); } catch {}

  return (
    <div className="p-8">
      {breadcrumb}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Zusatzzähler — {memberName}</h1>
        <p className="text-slate-500 mt-1">
          Manuell abgelesene Nebenzähler (z.B. Wärmepumpe), die keine Netzbetreiber-Smart-Meter sind. Der Verbrauch
          zwischen zwei Zählerständen wird zum normalen Bezugspreis dieses Mitglieds auf der nächsten Rechnung
          ausgewiesen.
        </p>
      </div>
      <ExtraMetersClient eegId={eegId} memberId={memberId} initialExtraMeters={extraMeters} />
    </div>
  );
}
