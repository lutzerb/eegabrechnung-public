"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

interface Props {
  eegId: string;
  referredMemberId: string;
  defaultAmount: number;
}

export default function GrantBonusButton({ eegId, referredMemberId, defaultAmount }: Props) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [amount, setAmount] = useState(String(defaultAmount));
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleGrant() {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/eegs/${eegId}/referrals/${referredMemberId}/grant-bonus`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ amount_eur: parseFloat(amount) || defaultAmount }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError((data as { error?: string }).error || "Prämie konnte nicht vergeben werden.");
        return;
      }
      router.refresh();
    } finally {
      setLoading(false);
    }
  }

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="px-3 py-1.5 bg-blue-600 text-white text-xs font-medium rounded-md hover:bg-blue-700 transition-colors"
      >
        Prämie gutschreiben
      </button>
    );
  }

  return (
    <div className="flex items-center gap-2">
      <div className="relative">
        <input
          type="number"
          step="0.01"
          min="0.01"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          className="w-24 px-2 py-1.5 pr-8 border border-slate-300 rounded-md text-xs focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <span className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 text-xs">€</span>
      </div>
      <button
        onClick={handleGrant}
        disabled={loading}
        className="px-3 py-1.5 bg-blue-600 text-white text-xs font-medium rounded-md hover:bg-blue-700 disabled:opacity-50 transition-colors"
      >
        {loading ? "…" : "Bestätigen"}
      </button>
      <button
        onClick={() => setOpen(false)}
        className="px-2 py-1.5 text-xs text-slate-500 hover:text-slate-700"
      >
        Abbrechen
      </button>
      {error && <span className="text-xs text-red-600">{error}</span>}
    </div>
  );
}
