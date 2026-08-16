import type { EEG } from "@/lib/api";
import Link from "next/link";
import { eegDisplayName } from "@/lib/eeg-display-name";

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
          Keine Gemeinschaften vorhanden.
        </p>
        <p className="text-slate-400 text-sm mt-1">
          Legen Sie eine neue EEG, GEA oder BEG an, um zu beginnen.
        </p>
        <Link
          href="/eegs/new"
          className="mt-4 inline-block px-4 py-2 text-sm font-medium bg-blue-700 text-white rounded-lg hover:bg-blue-800 transition-colors"
        >
          Neue Gemeinschaft anlegen
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
              Typ
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
                  {eegDisplayName(eeg)}
                </Link>
                {eeg.display_name && eeg.display_name !== eeg.name && (
                  <div className="text-xs text-slate-400 mt-0.5">{eeg.name}</div>
                )}
              </td>
              <td className="px-6 py-4">
                <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                  eeg.gemeinschaft_typ === "GEA"
                    ? "bg-purple-100 text-purple-700"
                    : eeg.gemeinschaft_typ === "BEG"
                    ? "bg-amber-100 text-amber-700"
                    : "bg-green-100 text-green-700"
                }`}>
                  {eeg.gemeinschaft_typ || "EEG"}
                </span>
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
