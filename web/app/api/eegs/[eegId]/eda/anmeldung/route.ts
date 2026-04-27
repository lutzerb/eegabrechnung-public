import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://eegabrechnung-api:8080";

export async function POST(
  req: NextRequest,
  context: { params: Promise<{  eegId: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken) return new NextResponse("Unauthorized", { status: 401 });

  const body = await req.text();
  const res = await fetch(`${API}/api/v1/eegs/${params.eegId}/eda/anmeldung`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${session.accessToken}`,
    },
    body,
  });
  const data = await res.text();
  return new NextResponse(data, { status: res.status, headers: { "Content-Type": "application/json" } });
}
