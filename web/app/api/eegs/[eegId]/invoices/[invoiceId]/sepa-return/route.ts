import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://eegabrechnung-api:8080";

export async function PATCH(
  req: NextRequest,
  context: { params: Promise<{ eegId: string; invoiceId: string }> }
) {
  const session = await auth();
  if (!session?.accessToken) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const { eegId, invoiceId } = await context.params;
  const body = await req.text();

  const res = await fetch(
    `${API}/api/v1/eegs/${eegId}/invoices/${invoiceId}/sepa-return`,
    {
      method: "PATCH",
      headers: {
        Authorization: `Bearer ${session.accessToken}`,
        "Content-Type": "application/json",
      },
      body,
    }
  );

  const data = await res.json();
  return NextResponse.json(data, { status: res.status });
}
