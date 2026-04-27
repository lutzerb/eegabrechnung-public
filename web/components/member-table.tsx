import type { Member } from "@/lib/api";

interface MemberTableProps {
  members: Member[];
}

function DirectionBadge({ direction }: { direction: string }) {
  const isConsumer = direction.toLowerCase().includes("consume") ||
    direction.toLowerCase() === "bezug" ||
    direction === "CONSUMPTION";
  const isProducer = direction.toLowerCase().includes("produc") ||
    direction.toLowerCase() === "einspeisung" ||
    direction === "GENERATION";

  if (isConsumer) {
    return (
      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-orange-50 text-orange-700">
        Bezug
      </span>
    );
  }
  if (isProducer) {
    return (
      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-50 text-green-700">
        Einspeisung
      </span>
    );
  }
  return (
    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600">
      {direction}
    </span>
  );
}

export default function MemberTable({ members }: MemberTableProps) {
  if (members.length === 0) {
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
            d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"
          />
        </svg>
        <p className="text-slate-600 font-medium">Keine Mitglieder vorhanden.</p>
        <p className="text-slate-400 text-sm mt-1">
          Importieren Sie Stammdaten, um Mitglieder hinzuzufügen.
        </p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 bg-slate-50">
            <th className="text-left px-6 py-3.5 font-medium text-slate-600">
              Mitglied
            </th>
            <th className="text-left px-6 py-3.5 font-medium text-slate-600">
              Mitgliedsnr.
            </th>
            <th className="text-left px-6 py-3.5 font-medium text-slate-600">
              E-Mail
            </th>
            <th className="text-left px-6 py-3.5 font-medium text-slate-600">
              Zählpunkte
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {members.map((member) => (
            <tr key={member.id} className="hover:bg-slate-50 transition-colors">
              <td className="px-6 py-4">
                <p className="font-medium text-slate-900">{member.name}</p>
              </td>
              <td className="px-6 py-4 text-slate-600 font-mono text-xs">
                {member.member_number || "—"}
              </td>
              <td className="px-6 py-4 text-slate-600">
                {member.email || "—"}
              </td>
              <td className="px-6 py-4">
                {member.meter_points && member.meter_points.length > 0 ? (
                  <div className="space-y-1">
                    {member.meter_points.map((mp) => (
                      <div
                        key={mp.id}
                        className="flex items-center gap-2"
                      >
                        <DirectionBadge direction={mp.direction} />
                        <span className="font-mono text-xs text-slate-600">
                          {mp.meter_id}
                        </span>
                        {mp.name && (
                          <span className="text-xs text-slate-400">
                            ({mp.name})
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                ) : (
                  <span className="text-slate-400 text-xs">
                    Keine Zählpunkte
                  </span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="px-6 py-3 border-t border-slate-100 bg-slate-50">
        <p className="text-xs text-slate-500">
          {members.length} Mitglied{members.length !== 1 ? "er" : ""} &middot;{" "}
          {members.reduce((sum, m) => sum + (m.meter_points?.length || 0), 0)}{" "}
          Zählpunkt(e) gesamt
        </p>
      </div>
    </div>
  );
}
