import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG, getMember } from "@/lib/api";
import Link from "next/link";
import TariffManager from "@/components/tariff-manager";

interface Props {
  params: Promise<{ eegId: string; memberId: string }>;
}

export default async function MemberTariffPage({ params }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId, memberId } = await params;
  let eeg = null;
  let member = null;
  try { eeg = await getEEG(session.accessToken!, eegId); } catch {}
  try { member = await getMember(session.accessToken!, eegId, memberId); } catch {}

  const memberName = member ? `${member.name1} ${member.name2}`.trim() : memberId;

  return (
    <div className="p-8">
      <div className="mb-6">
        <Link href="/eegs" className="text-sm text-slate-500 hover:text-slate-700">Energiegemeinschaften</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">{eeg?.name || eegId}</Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}/members/${memberId}`} className="text-sm text-slate-500 hover:text-slate-700">{memberName}</Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Individualtarif</span>
      </div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Individualtarif — {memberName}</h1>
        <p className="text-slate-500 mt-1">
          Individuelle Tarifpläne und Gebühren-Overrides für dieses Mitglied. Nicht überschriebene Felder verwenden weiterhin den EEG-Standard.
        </p>
      </div>
      <TariffManager
        eegId={eegId}
        memberId={memberId}
        eegEnergyPrice={eeg?.energy_price ?? 0}
        eegProducerPrice={eeg?.producer_price ?? 0}
      />
    </div>
  );
}
