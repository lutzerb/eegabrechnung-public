import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function POST(
  request: NextRequest,
  context: { params: Promise<{ eegId: string }> }
) {
  const { eegId } = await context.params;
  const session = await auth();
  if (!session?.accessToken) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  const formData = await request.formData();
  const res = await fetch(`${API}/api/v1/eegs/${eegId}/ea/belege`, {
    method: "POST",
    headers: { Authorization: `Bearer ${session.accessToken}` },
    body: formData,
  });
  return NextResponse.json(await res.json(), { status: res.status });
}
