import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  _req: NextRequest,
  context: { params: Promise<{ eegId: string; belegId: string }> }
) {
  const { eegId, belegId } = await context.params;
  const session = await auth();
  if (!session?.accessToken) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  const res = await fetch(`${API}/api/v1/eegs/${eegId}/ea/belege/${belegId}`, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  const ct = res.headers.get("Content-Type") || "";
  if (ct.includes("pdf") || ct.includes("octet-stream") || ct.includes("image")) {
    const data = await res.arrayBuffer();
    return new Response(data, {
      status: res.status,
      headers: {
        "Content-Type": ct,
        "Content-Disposition": res.headers.get("Content-Disposition") || "",
      },
    });
  }
  return NextResponse.json(await res.json(), { status: res.status });
}

export async function DELETE(
  _req: NextRequest,
  context: { params: Promise<{ eegId: string; belegId: string }> }
) {
  const { eegId, belegId } = await context.params;
  const session = await auth();
  if (!session?.accessToken) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  const res = await fetch(`${API}/api/v1/eegs/${eegId}/ea/belege/${belegId}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  if (res.status === 204) return new Response(null, { status: 204 });
  return NextResponse.json(await res.json(), { status: res.status });
}
