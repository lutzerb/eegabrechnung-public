import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import Link from "next/link";
import { getEEG } from "@/lib/api";
import { ReadingsDebugTable } from "@/components/readings-debug-table";

interface Props {
  params: Promise<{ eegId: string }>;
}

export default async function ReadingsDebugPage({ params }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId } = await params;
  const eeg = await getEEG(session.accessToken!, eegId);

  return (
    <div className="p-8 max-w-6xl">
      {/* Breadcrumb */}
      <div className="mb-6 flex items-center gap-2 text-sm text-slate-500">
        <Link href="/eegs" className="hover:text-slate-700">Energiegemeinschaften</Link>
        <span>/</span>
        <Link href={`/eegs/${eegId}/settings`} className="hover:text-slate-700">Einstellungen</Link>
        <span>/</span>
        <span className="text-slate-900 font-medium">Messwerte-Debug</span>
      </div>

      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Messwerte-Debug</h1>
        <p className="text-sm text-slate-500 mt-1">
          Rohe 15-Minuten-Messwerte für {eeg.display_name || eeg.name} — inklusive der einzelnen
          OBIS-Codes und ihrer jeweiligen Qualitätsstufe, live aus den empfangenen EDA-Nachrichten
          ermittelt.
        </p>
      </div>

      <ReadingsDebugTable eegId={eegId} />
    </div>
  );
}
