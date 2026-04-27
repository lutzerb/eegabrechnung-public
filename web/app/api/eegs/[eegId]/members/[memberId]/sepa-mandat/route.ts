import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://eegabrechnung-api:8080";

export async function GET(
  _req: NextRequest,
  context: { params: Promise<{ eegId: string; memberId: string }> }
) {
  const params = await context.params;
  const session = await auth();
  if (!session?.accessToken) return new NextResponse("Unauthorized", { status: 401 });

  const res = await fetch(
    `${API}/api/v1/eegs/${params.eegId}/members/${params.memberId}/sepa-mandat`,
    {
      headers: { Authorization: `Bearer ${session.accessToken}` },
    }
  );

  if (!res.ok) {
    const text = await res.text();
    return new NextResponse(text, { status: res.status });
  }

  const pdfBytes = await res.arrayBuffer();
  return new NextResponse(pdfBytes, {
    status: 200,
    headers: {
      "Content-Type": "application/pdf",
      "Content-Disposition": `attachment; filename="sepamandat_${params.memberId.slice(0, 8)}.pdf"`,
    },
  });
}
