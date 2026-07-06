import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG, listParticipations, listMembers, type EEGMeterParticipation, type Member } from "@/lib/api";
import Link from "next/link";
import ParticipationsClient from "./ParticipationsClient";

interface Props {
  params: Promise<{ eegId: string }>;
}

export default async function ParticipationsPage({ params }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId } = await params;

  let eeg = null;
  let participations: EEGMeterParticipation[] = [];
  let members: Member[] = [];

  try {
    [eeg, participations, members] = await Promise.all([
      getEEG(session.accessToken!, eegId),
      listParticipations(session.accessToken!, eegId).catch(() => []),
      listMembers(session.accessToken!, eegId).catch(() => []),
    ]);
  } catch {
    // ignore, show empty state
  }

  // Build a map from meter_point_id → member name for display
  const mpToMember: Record<string, { memberName: string; meterId: string }> = {};
  for (const m of members) {
    for (const mp of m.meter_points || []) {
      mpToMember[mp.id] = { memberName: m.name || m.name1 || m.name2 || "—", meterId: mp.meter_id };
    }
  }

  // Flatten all meter points for the create form
  const allMeterPoints = members.flatMap((m) =>
    (m.meter_points || []).map((mp) => ({
      id: mp.id,
      label: `${m.name || m.name1 || ""} — ${mp.meter_id} (${mp.direction})`,
    }))
  );

  return (
    <div className="p-8 max-w-5xl">
      {/* Breadcrumb */}
      <div className="mb-6 text-sm text-slate-500">
        <Link href="/eegs" className="hover:text-slate-700">Energiegemeinschaften</Link>
        <span className="mx-2 text-slate-300">/</span>
        <Link href={`/eegs/${eegId}`} className="hover:text-slate-700">{eeg?.name || eegId}</Link>
        <span className="mx-2 text-slate-300">/</span>
        <span className="text-slate-900 font-medium">Mehrfachteilnahme</span>
      </div>

      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Mehrfachteilnahme</h1>
        <p className="text-sm text-slate-500 mt-1">
          Seit April 2024 (EAG) kann ein Zählpunkt gleichzeitig an bis zu 5 Energiegemeinschaften teilnehmen.
          Hier werden die Teilnahmedatensätze verwaltet (Grundlage für EDA-Anmeldungen).
        </p>
      </div>

      <ParticipationsClient
        eegId={eegId}
        initialParticipations={participations}
        mpToMember={mpToMember}
        allMeterPoints={allMeterPoints}
      />
    </div>
  );
}
