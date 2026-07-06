import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { getEEG } from "@/lib/api";
import Link from "next/link";
import ImportSections from "@/components/import-sections";

interface Props {
  params: Promise<{ eegId: string }>;
}

export default async function ImportPage({ params }: Props) {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  const { eegId } = await params;

  let eeg = null;
  try {
    eeg = await getEEG(session.accessToken!, eegId);
  } catch {
    // tolerate error, still show the import page
  }

  return (
    <div className="p-8">
      {/* Breadcrumb */}
      <div className="mb-6">
        <Link href="/eegs" className="text-sm text-slate-500 hover:text-slate-700">
          Energiegemeinschaften
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <Link href={`/eegs/${eegId}`} className="text-sm text-slate-500 hover:text-slate-700">
          {eeg?.name || eegId}
        </Link>
        <span className="text-slate-400 mx-2">/</span>
        <span className="text-sm text-slate-900 font-medium">Daten importieren</span>
      </div>

      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Daten importieren</h1>
        <p className="text-slate-500 mt-1">
          Laden Sie Stammdaten und Energiedaten im XLSX-Format hoch.
        </p>
      </div>

      <div className="max-w-3xl">
        <ImportSections eegId={eegId} />
      </div>
    </div>
  );
}
