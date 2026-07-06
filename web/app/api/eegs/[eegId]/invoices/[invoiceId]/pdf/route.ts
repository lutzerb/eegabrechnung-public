import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API_INTERNAL_URL =
  process.env.API_INTERNAL_URL || "http://eegabrechnung-api:8080";

export async function GET(
  _req: NextRequest,
  context: { params: Promise<{  eegId: string; invoiceId: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken) {
    return new NextResponse("Unauthorized", { status: 401 });
  }

  const { eegId, invoiceId } = params;
  const apiUrl = `${API_INTERNAL_URL}/api/v1/eegs/${eegId}/invoices/${invoiceId}/pdf`;

  const res = await fetch(apiUrl, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });

  if (!res.ok) {
    return new NextResponse("PDF not found", { status: res.status });
  }

  const blob = await res.blob();
  return new NextResponse(blob, {
    headers: {
      "Content-Type": "application/pdf",
      "Content-Disposition": `inline; filename="Rechnung_${invoiceId.slice(0, 8)}.pdf"`,
    },
  });
}
