import type { EffectiveTariff } from "@/lib/api";

const SOURCE_META: Record<EffectiveTariff["source"], { label: string; color: string }> = {
  individual: { label: "Individualtarif", color: "bg-amber-50 text-amber-700 border-amber-200" },
  assigned: { label: "zugewiesen", color: "bg-blue-50 text-blue-700 border-blue-200" },
  default: { label: "Standard", color: "bg-slate-100 text-slate-600 border-slate-200" },
  none: { label: "", color: "" },
};

export function EffectiveTariffCell({ tariff }: { tariff?: EffectiveTariff | null }) {
  if (!tariff || tariff.source === "none") {
    return <span className="text-slate-400 text-xs" title="Kein Tarifplan aktiv — es gelten die flachen EEG-Preise">EEG-Preise</span>;
  }
  const meta = SOURCE_META[tariff.source];
  return (
    <div className="flex flex-col items-start gap-1 min-w-0 w-full">
      <span className="text-sm text-slate-900 truncate max-w-full" title={tariff.name}>{tariff.name}</span>
      <span className={`inline-flex items-center px-1.5 py-0.5 rounded border text-xs font-medium max-w-full truncate ${meta.color}`}>
        {meta.label}
      </span>
    </div>
  );
}
