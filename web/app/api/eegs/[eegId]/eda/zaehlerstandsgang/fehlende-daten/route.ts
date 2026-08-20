import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API_INTERNAL_URL =
  process.env.API_INTERNAL_URL || "http://eegabrechnung-api:8080";

export async function GET(
  _req: NextRequest,
  context: { params: Promise<{ eegId: string }> }
) {
  const params = await context.params;
  const session = await auth();
  if (!session?.accessToken) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const res = await fetch(
    `${API_INTERNAL_URL}/api/v1/eegs/${params.eegId}/eda/zaehlerstandsgang/fehlende-daten`,
    {
      headers: { Authorization: `Bearer ${session.accessToken}` },
    }
  );

  const data = await res.text();
  return new NextResponse(data, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}
