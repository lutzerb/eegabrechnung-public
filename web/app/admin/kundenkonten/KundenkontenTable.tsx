"use client";

import { useState } from "react";
import type { EEG } from "@/lib/api";

const TYP_COLORS: Record<string, string> = {
  EEG: "bg-blue-50 text-blue-700",
  BEG: "bg-violet-50 text-violet-700",
  GEA: "bg-amber-50 text-amber-700",
};

const ASSIGN_FORM_ID = "kundenkonten-assign-form";

interface Props {
  eegs: EEG[];
  missingCounts: Record<string, number>;
  setStartAction: (formData: FormData) => Promise<void>;
  assignAction: (formData: FormData) => Promise<void>;
}

export function KundenkontenTable({ eegs, missingCounts, setStartAction, assignAction }: Props) {
  const [selected, setSelected] = useState<Set<string>>(new Set());

  function toggle(id: string, checked: boolean) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  }

  function toggleAll(checked: boolean) {
    setSelected(checked ? new Set(eegs.map((e) => e.id)) : new Set());
  }

  if (eegs.length === 0) {
    return (
      <div className="bg-white rounded-xl border border-slate-200 px-6 py-16 text-center text-slate-400 text-sm">
        Keine Energiegemeinschaften vorhanden.
      </div>
    );
  }

  return (
    <div>
      {/* Empty, detached form: the per-row "eeg_id" checkboxes and the bottom
          "jetzt vergeben" button reference it via the form="..." attribute
          (valid HTML5 — the input need not be a descendant), so it never has to
          nest inside the per-row Startnummer forms below. */}
      <form id={ASSIGN_FORM_ID} action={assignAction} />

      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm min-w-[720px]">
            <thead>
              <tr className="bg-slate-50 border-b border-slate-100">
                <th className="text-left px-5 py-3 w-10">
                  <input
                    type="checkbox"
                    checked={selected.size === eegs.length}
                    onChange={(e) => toggleAll(e.target.checked)}
                    className="rounded border-slate-300 text-blue-700 focus:ring-blue-500"
                  />
                </th>
                <th className="text-left px-5 py-3 text-xs font-medium text-slate-500">Energiegemeinschaft</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-slate-500">Ohne Kundenkonto</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-slate-500">Startnummer</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              {eegs.map((eeg) => (
                <tr key={eeg.id} className="hover:bg-slate-50 transition-colors">
                  <td className="px-5 py-3">
                    <input
                      type="checkbox"
                      name="eeg_id"
                      value={eeg.id}
                      form={ASSIGN_FORM_ID}
                      checked={selected.has(eeg.id)}
                      onChange={(e) => toggle(eeg.id, e.target.checked)}
                      className="rounded border-slate-300 text-blue-700 focus:ring-blue-500"
                    />
                  </td>
                  <td className="px-5 py-3">
                    <div className="font-medium text-slate-900">{eeg.display_name || eeg.name}</div>
                    <span
                      className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium mt-0.5 ${
                        TYP_COLORS[eeg.gemeinschaft_typ] || "bg-slate-100 text-slate-600"
                      }`}
                    >
                      {eeg.gemeinschaft_typ}
                    </span>
                  </td>
                  <td className="px-5 py-3 text-slate-600">
                    {(missingCounts[eeg.id] ?? 0) > 0 ? (
                      <span className="font-medium text-amber-700">{missingCounts[eeg.id]}</span>
                    ) : (
                      <span className="text-slate-400">0</span>
                    )}
                  </td>
                  <td className="px-5 py-3">
                    <form action={setStartAction} className="flex items-center gap-2">
                      <input type="hidden" name="eeg_id" value={eeg.id} />
                      <input
                        type="number"
                        name="kundenkonto_start"
                        min={0}
                        defaultValue={eeg.kundenkonto_start || 0}
                        className="w-24 px-2 py-1 border border-slate-300 rounded text-slate-900 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      />
                      <button
                        type="submit"
                        className="px-2.5 py-1 text-xs font-medium text-slate-700 bg-slate-100 rounded hover:bg-slate-200 transition-colors"
                      >
                        Speichern
                      </button>
                    </form>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="px-5 py-3.5 border-t border-slate-100 bg-slate-50 flex items-center justify-between">
          <p className="text-xs text-slate-500">
            {selected.size} von {eegs.length} ausgewählt
          </p>
          <button
            type="submit"
            form={ASSIGN_FORM_ID}
            disabled={selected.size === 0}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-700 rounded-lg hover:bg-blue-800 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            Kundenkonten jetzt vergeben
          </button>
        </div>
      </div>
    </div>
  );
}
