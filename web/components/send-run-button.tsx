"use client";

import { useFormStatus } from "react-dom";

function SubmitButton() {
  const { pending } = useFormStatus();
  return (
    <button
      type="submit"
      disabled={pending}
      className="px-3 py-1 text-xs font-medium text-white bg-blue-600 rounded hover:bg-blue-700 transition-colors disabled:opacity-60 disabled:cursor-not-allowed flex items-center gap-1.5"
    >
      {pending && (
        <svg className="animate-spin h-3 w-3" viewBox="0 0 24 24" fill="none">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
        </svg>
      )}
      {pending ? "Wird versendet…" : "Rechnungen versenden"}
    </button>
  );
}

interface SendRunButtonProps {
  runId: string;
  sendRunAction: (formData: FormData) => Promise<void>;
}

export default function SendRunButton({ runId, sendRunAction }: SendRunButtonProps) {
  return (
    <form action={sendRunAction}>
      <input type="hidden" name="runId" value={runId} />
      <SubmitButton />
    </form>
  );
}
