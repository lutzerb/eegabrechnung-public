"use client";

import { useRouter, usePathname, useSearchParams } from "next/navigation";

interface Props {
  showCancelled: boolean;
  sort: string;
}

export default function BillingRunsFilters({ showCancelled, sort }: Props) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  function update(key: string, value: string | null) {
    const params = new URLSearchParams(searchParams.toString());
    // preserve success/error params
    if (value === null) params.delete(key);
    else params.set(key, value);
    // strip success/error when changing filters
    params.delete("success");
    params.delete("error");
    router.replace(`${pathname}?${params.toString()}`);
  }

  return (
    <div className="flex items-center gap-4 flex-wrap">
      <select
        value={sort}
        onChange={(e) => update("sort", e.target.value)}
        className="text-sm border border-slate-200 rounded px-2 py-1.5 text-slate-700 bg-white"
      >
        <option value="period_desc">Zeitraum (neueste zuerst)</option>
        <option value="period_asc">Zeitraum (älteste zuerst)</option>
        <option value="status">Status</option>
        <option value="amount_desc">Betrag (höchste zuerst)</option>
        <option value="amount_asc">Betrag (niedrigste zuerst)</option>
      </select>

      <label className="flex items-center gap-2 text-sm text-slate-600 cursor-pointer select-none">
        <input
          type="checkbox"
          checked={showCancelled}
          onChange={(e) => update("show_cancelled", e.target.checked ? "1" : null)}
          className="rounded border-slate-300 text-slate-600"
        />
        Stornierte anzeigen
      </label>
    </div>
  );
}
