"use client";

import { useRef } from "react";

interface CancelRunButtonProps {
  runId: string;
  cancelRunAction: (formData: FormData) => Promise<void>;
}

export default function CancelRunButton({ runId, cancelRunAction }: CancelRunButtonProps) {
  const formRef = useRef<HTMLFormElement>(null);

  function handleClick(e: React.MouseEvent) {
    e.preventDefault();
    if (
      window.confirm(
        "Abrechnungslauf wirklich stornieren?\n\nAlle nicht-bezahlten Rechnungen werden ebenfalls storniert. Diese Aktion kann nicht rückgängig gemacht werden."
      )
    ) {
      formRef.current?.requestSubmit();
    }
  }

  return (
    <form ref={formRef} action={cancelRunAction}>
      <input type="hidden" name="runId" value={runId} />
      <button
        type="button"
        onClick={handleClick}
        className="px-3 py-1 text-xs font-medium text-red-700 bg-red-50 border border-red-200 rounded hover:bg-red-100 transition-colors"
      >
        Lauf stornieren
      </button>
    </form>
  );
}
