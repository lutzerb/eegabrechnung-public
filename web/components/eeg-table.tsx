import type { EEG } from "@/lib/api";
import Link from "next/link";

interface EEGTableProps {
  eegs: EEG[];
}

export default function EEGTable({ eegs }: EEGTableProps) {
  if (eegs.length === 0) {
    return (
      <div className="bg-white rounded-xl border border-slate-200 px-6 py-16 text-center">
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
        <p className="text-slate-600 font-medium">
          Keine EEGs vorhanden.
        </p>
        <p className="text-slate-400 text-sm mt-1">
          Erstellen Sie eine neue Energiegemeinschaft, um zu beginnen.
        </p>
        <Link
          href="/eegs/new"
          className="mt-4 inline-block px-4 py-2 text-sm font-medium bg-blue-700 text-white rounded-lg hover:bg-blue-800 transition-colors"
        >
          Neue EEG erstellen
        </Link>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 bg-slate-50">
            <th className="text-left px-6 py-3.5 font-medium text-slate-600">
              Name
            </th>
            <th className="text-left px-6 py-3.5 font-medium text-slate-600">
              Gemeinschafts-ID
            </th>
            <th className="text-left px-6 py-3.5 font-medium text-slate-600">
              Netzbetreiber
            </th>
            <th className="text-right px-6 py-3.5 font-medium text-slate-600">
              Aktionen
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {eegs.map((eeg) => (
            <tr key={eeg.id} className="hover:bg-slate-50 transition-colors">
              <td className="px-6 py-4">
                <Link
                  href={`/eegs/${eeg.id}`}
                  className="font-medium text-slate-900 hover:text-blue-700"
                >
                  {eeg.name}
                </Link>
              </td>
              <td className="px-6 py-4 text-slate-600 font-mono text-xs">
                {eeg.gemeinschaft_id}
              </td>
              <td className="px-6 py-4 text-slate-600">{eeg.netzbetreiber}</td>
              <td className="px-6 py-4 text-right">
                <div className="flex items-center justify-end gap-2">
                  <Link
                    href={`/eegs/${eeg.id}`}
                    className="px-3 py-1.5 text-xs font-medium text-blue-700 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors"
                  >
                    Übersicht
                  </Link>
                  <Link
                    href={`/eegs/${eeg.id}/members`}
                    className="px-3 py-1.5 text-xs font-medium text-slate-600 bg-slate-100 rounded-md hover:bg-slate-200 transition-colors"
                  >
                    Mitglieder
                  </Link>
                  <Link
                    href={`/eegs/${eeg.id}/billing`}
                    className="px-3 py-1.5 text-xs font-medium text-slate-600 bg-slate-100 rounded-md hover:bg-slate-200 transition-colors"
                  >
                    Abrechnung
                  </Link>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
