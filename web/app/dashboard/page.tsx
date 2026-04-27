import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { listEEGs, getEEGStats, type EEG, type EEGStats } from "@/lib/api";
import Link from "next/link";

function formatDate(dateStr: string | undefined): string {
  if (!dateStr) return "—";
  try {
    return new Date(dateStr).toLocaleDateString("de-AT", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
    });
  } catch {
    return dateStr;
  }
}

export default async function DashboardPage() {
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

  // Fetch stats for all EEGs in parallel (ignore individual failures)
  const statsResults: (EEGStats | null)[] = await Promise.all(
    eegs.map((eeg) =>
      getEEGStats(session.accessToken!, eeg.id).catch(() => null)
    )
  );

  const totalMembers = statsResults.reduce(
    (sum, s) => sum + (s?.member_count || 0),
    0
  );

  const lastBillingRun = statsResults
    .map((s) => s?.last_billing_run)
    .filter(Boolean)
    .sort()
    .pop();

  return (
    <div className="p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Dashboard</h1>
        <p className="text-slate-500 mt-1">
          Willkommen, {session.user?.name || session.user?.email || "Benutzer"}
        </p>
      </div>

      {error && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <p className="font-medium">Fehler</p>
          <p className="text-sm mt-1">{error}</p>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="bg-white rounded-xl border border-slate-200 p-6">
          <p className="text-sm text-slate-500 font-medium">
            Energiegemeinschaften
          </p>
          <p className="text-3xl font-bold text-slate-900 mt-1">
            {eegs.length}
          </p>
        </div>
        <div className="bg-white rounded-xl border border-slate-200 p-6">
          <p className="text-sm text-slate-500 font-medium">
            Gesamt Mitglieder
          </p>
          <p className="text-3xl font-bold text-slate-900 mt-1">
            {eegs.length > 0 ? totalMembers : "—"}
          </p>
        </div>
        <div className="bg-white rounded-xl border border-slate-200 p-6">
          <p className="text-sm text-slate-500 font-medium">
            Letzte Abrechnung
          </p>
          <p className="text-3xl font-bold text-slate-900 mt-1">
            {eegs.length > 0 ? formatDate(lastBillingRun) : "—"}
          </p>
        </div>
      </div>

      <div className="bg-white rounded-xl border border-slate-200">
        <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between">
          <h2 className="font-semibold text-slate-800">
            Meine Energiegemeinschaften
          </h2>
          <Link
            href="/eegs/new"
            className="px-4 py-2 text-sm font-medium bg-blue-700 text-white rounded-lg hover:bg-blue-800 transition-colors"
          >
            + Neue EEG
          </Link>
        </div>

        {eegs.length === 0 && !error ? (
          <div className="px-6 py-12 text-center">
            <svg
              className="mx-auto w-12 h-12 text-slate-300 mb-3"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M13 10V3L4 14h7v7l9-11h-7z"
              />
            </svg>
            <p className="text-slate-500">
              Keine Energiegemeinschaften vorhanden.
            </p>
            <p className="text-slate-400 text-sm mt-1">
              Erstellen Sie eine neue Energiegemeinschaft, um zu beginnen.
            </p>
            <Link
              href="/eegs/new"
              className="mt-4 inline-block px-4 py-2 text-sm font-medium bg-blue-700 text-white rounded-lg hover:bg-blue-800 transition-colors"
            >
              Energiegemeinschaft erstellen
            </Link>
          </div>
        ) : (
          <div className="divide-y divide-slate-100">
            {eegs.map((eeg, idx) => {
              const stats = statsResults[idx];
              return (
                <Link
                  key={eeg.id}
                  href={`/eegs/${eeg.id}`}
                  className="flex items-center justify-between px-6 py-4 hover:bg-slate-50 transition-colors"
                >
                  <div>
                    <p className="font-medium text-slate-900">{eeg.name}</p>
                    <p className="text-sm text-slate-500 mt-0.5">
                      {eeg.gemeinschaft_id} &middot; {eeg.netzbetreiber}
                    </p>
                  </div>
                  <div className="flex items-center gap-6">
                    {stats && (
                      <div className="text-right">
                        <p className="text-sm font-medium text-slate-700">
                          {stats.member_count} Mitglieder
                        </p>
                        {stats.last_billing_run && (
                          <p className="text-xs text-slate-400">
                            Letzte Abrechnung: {formatDate(stats.last_billing_run)}
                          </p>
                        )}
                      </div>
                    )}
                    <svg
                      className="w-4 h-4 text-slate-400"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M9 5l7 7-7 7"
                      />
                    </svg>
                  </div>
                </Link>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
