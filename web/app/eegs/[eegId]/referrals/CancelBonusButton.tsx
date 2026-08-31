"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

interface Props {
  eegId: string;
  referredMemberId: string;
}

export default function CancelBonusButton({ eegId, referredMemberId }: Props) {
  const router = useRouter();
  const [loading, setLoading] = useState(false);

  async function handleCancel() {
    if (!confirm("Vergabe dieser Werbeprämie rückgängig machen?")) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/eegs/${eegId}/referrals/${referredMemberId}/grant-bonus`, {
        method: "DELETE",
      });
      if (res.ok) router.refresh();
    } finally {
      setLoading(false);
    }
  }

  return (
    <button
      onClick={handleCancel}
      disabled={loading}
      className="text-xs text-slate-400 hover:text-red-600 transition-colors disabled:opacity-50"
    >
      rückgängig
    </button>
  );
}
