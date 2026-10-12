import type { TariffSchedule } from "@/lib/api";

// Select for assigning an EEG-wide tariff plan to a member. Empty value = Standardtarif.
export function TariffPlanSelect({
  plans,
  value,
  className,
}: {
  plans: TariffSchedule[];
  value?: string | null;
  className?: string;
}) {
  const defaultPlan = plans.find((p) => p.is_default);
  const assignable = plans.filter((p) => p.is_active && !p.is_default && !p.member_id);
  return (
    <select name="tariff_schedule_id" defaultValue={value ?? ""} className={className}>
      <option value="">
        {defaultPlan ? `Standardtarif (${defaultPlan.name})` : "Standardtarif (flache EEG-Preise)"}
      </option>
      {assignable.map((p) => (
        <option key={p.id} value={p.id}>{p.name}</option>
      ))}
    </select>
  );
}
