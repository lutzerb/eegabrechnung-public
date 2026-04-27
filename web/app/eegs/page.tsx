import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { listEEGs, type EEG } from "@/lib/api";
import Link from "next/link";
import EEGTable from "@/components/eeg-table";

export default async function EEGsPage() {
  const session = await auth();
  if (!session) redirect("/auth/signin");

  let eegs: EEG[] = [];
  let error: string | null = null;

  try {
    eegs = await listEEGs(session.accessToken!);
  } catch (err: unknown) {
    const apiError = err as { message?: string };
    error = apiError.message || "Fehler beim Laden der Energiegemeinschaften.";
  }

  return (
    <div className="p-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">
            Energiegemeinschaften
          </h1>
          <p className="text-slate-500 mt-1">
            Alle Energiegemeinschaften im System
          </p>
        </div>
        <Link
          href="/eegs/new"
          className="px-4 py-2 text-sm font-medium bg-blue-700 text-white rounded-lg hover:bg-blue-800 transition-colors"
        >
          + Neue EEG erstellen
        </Link>
      </div>

      {error && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <p className="font-medium">Fehler beim Laden</p>
          <p className="text-sm mt-1">{error}</p>
        </div>
      )}

      <EEGTable eegs={eegs} />
    </div>
  );
}
