"use client";

import { useEffect, useState } from "react";

const STORAGE_KEY = "members-show-mp-address";
const TARGET_ID = "members-table";

function applyToTarget(on: boolean) {
  document.getElementById(TARGET_ID)?.setAttribute("data-show-address", on ? "true" : "false");
}

export function MeterPointAddressToggle() {
  const [on, setOn] = useState(false);

  useEffect(() => {
    let stored = false;
    try {
      stored = localStorage.getItem(STORAGE_KEY) === "1";
    } catch {
      // storage unavailable — keep default (off)
    }
    setOn(stored);
    applyToTarget(stored);
  }, []);

  function onChange(checked: boolean) {
    setOn(checked);
    applyToTarget(checked);
    try {
      localStorage.setItem(STORAGE_KEY, checked ? "1" : "0");
    } catch {
      // ignore
    }
  }

  return (
    <label className="flex items-center gap-2 text-sm text-slate-600 whitespace-nowrap cursor-pointer select-none">
      <input
        type="checkbox"
        checked={on}
        onChange={(e) => onChange(e.target.checked)}
        className="rounded border-slate-300 text-blue-700 focus:ring-blue-500"
      />
      Zählpunkt-Adressen anzeigen
    </label>
  );
}
