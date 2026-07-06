"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import SepaReturnDialog from "./sepa-return-dialog";
import type { Invoice } from "@/lib/api";

interface SepaReturnInvoiceActionsProps {
  eegId: string;
  invoice: Invoice;
}

function formatDate(isoString?: string): string {
  if (!isoString) return "—";
  try {
    return new Date(isoString).toLocaleDateString("de-AT", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
    });
  } catch {
    return isoString;
  }
}

export default function SepaReturnInvoiceActions({
  eegId,
  invoice,
}: SepaReturnInvoiceActionsProps) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const router = useRouter();

  const invoiceRef = invoice.invoice_number
    ? `#${invoice.invoice_number}`
    : invoice.id.slice(0, 8);

  return (
    <>
      <button
        onClick={() => setDialogOpen(true)}
        title={
          invoice.sepa_return_at
            ? `Rücklastschrift vom ${formatDate(invoice.sepa_return_at)}${invoice.sepa_return_reason ? ` (${invoice.sepa_return_reason})` : ""} – klicken zum Bearbeiten`
            : "Rücklastschrift erfassen"
        }
        className={`px-2 py-1 text-xs font-medium rounded transition-colors ${
          invoice.sepa_return_at
            ? "text-red-700 bg-red-50 border border-red-200 hover:bg-red-100"
            : "text-slate-600 bg-slate-100 hover:bg-slate-200"
        }`}
      >
        {invoice.sepa_return_at ? "Rücklastschr." : "Rücklastschr.?"}
      </button>

      {dialogOpen && (
        <SepaReturnDialog
          eegId={eegId}
          invoiceId={invoice.id}
          invoiceRef={invoiceRef}
          existingReturn={
            invoice.sepa_return_at
              ? {
                  sepa_return_at: invoice.sepa_return_at,
                  sepa_return_reason: invoice.sepa_return_reason,
                  sepa_return_note: invoice.sepa_return_note,
                }
              : undefined
          }
          onClose={() => setDialogOpen(false)}
          onSaved={() => {
            setDialogOpen(false);
            router.refresh();
          }}
        />
      )}
    </>
  );
}
