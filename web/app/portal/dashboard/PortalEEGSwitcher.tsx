"use client";

import { useEffect, useRef, useState } from "react";

interface PortalEEG {
  eeg_id: string;
  eeg_name: string;
}

export default function PortalEEGSwitcher({ currentId, currentName }: {
  currentId: string;
  currentName: string;
}) {
  const [open, setOpen] = useState(false);
  const [eegs, setEegs] = useState<PortalEEG[]>([]);
  const [switching, setSwitching] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetch("/api/portal/eegs")
      .then((res) => (res.ok ? res.json() : []))
      .then((data: PortalEEG[]) => setEegs(Array.isArray(data) ? data : []))
      .catch(() => setEegs([]));
  }, []);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  async function switchTo(eegId: string) {
    if (eegId === currentId) {
      setOpen(false);
      return;
    }
    setSwitching(true);
    setError(null);
    try {
      const res = await fetch("/api/portal/switch-eeg", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ eeg_id: eegId }),
      });
      if (!res.ok) {
        setError("Umschalten fehlgeschlagen.");
        return;
      }
      window.location.href = "/portal/dashboard";
    } catch {
      setError("Netzwerkfehler.");
    } finally {
      setSwitching(false);
      setOpen(false);
    }
  }

  if (eegs.length <= 1) {
    return <p className="text-xs text-slate-500">{currentName}</p>;
  }

  return (
    <div ref={ref} className="relative flex flex-col items-center gap-1">
      <span className="text-xs text-blue-700 font-medium">Auswahl Gemeinschaft</span>
      <button
        onClick={() => setOpen((o) => !o)}
        disabled={switching}
        className="w-80 sm:w-[28rem] max-w-[90vw] flex items-center justify-between gap-3 px-4 py-2 rounded-lg bg-blue-50 border border-blue-200 hover:bg-blue-100 transition-colors disabled:opacity-60"
      >
        <span className="text-sm font-semibold text-blue-900 truncate" title={currentName}>{switching ? "…" : currentName}</span>
        <svg className={`w-4 h-4 text-blue-500 flex-shrink-0 transition-transform ${open ? "rotate-180" : ""}`}
          fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {error && <p className="text-xs text-red-600">{error}</p>}

      {open && (
        <div className="absolute left-1/2 -translate-x-1/2 top-full mt-1 w-80 sm:w-[28rem] max-w-[90vw] bg-white border border-slate-200 rounded-lg shadow-lg z-50 overflow-hidden">
          <div className="py-1 max-h-56 overflow-y-auto">
            {eegs.map((eeg) => {
              const isCurrent = eeg.eeg_id === currentId;
              return (
                <button
                  key={eeg.eeg_id}
                  onClick={() => switchTo(eeg.eeg_id)}
                  className={`w-full text-left px-3 py-2 text-sm transition-colors flex items-center gap-2 ${
                    isCurrent ? "bg-blue-50 text-blue-700 font-medium" : "text-slate-700 hover:bg-slate-50"
                  }`}
                >
                  {isCurrent && (
                    <svg className="w-3.5 h-3.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                    </svg>
                  )}
                  <span className={`truncate ${isCurrent ? "" : "ml-5"}`}>{eeg.eeg_name}</span>
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
