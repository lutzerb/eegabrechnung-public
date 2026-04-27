import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  request: NextRequest,
  context: { params: Promise<{ eegId: string; kontoId: string }> }
) {
  const { eegId, kontoId } = await context.params;
  const session = await auth();
  if (!session?.accessToken) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  const q = new URL(request.url).searchParams.toString();
  const res = await fetch(`${API}/api/v1/eegs/${eegId}/ea/kontenblatt/${kontoId}${q ? "?" + q : ""}`, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  return NextResponse.json(await res.json(), { status: res.status });
}
