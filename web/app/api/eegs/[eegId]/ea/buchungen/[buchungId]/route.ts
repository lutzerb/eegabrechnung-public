import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  _req: NextRequest,
  context: { params: Promise<{ eegId: string; buchungId: string }> }
) {
  const { eegId, buchungId } = await context.params;
  const session = await auth();
  if (!session?.accessToken) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  const q = new URL(_req.url).searchParams.toString();
  const res = await fetch(`${API}/api/v1/eegs/${eegId}/ea/buchungen/${buchungId}${q ? "?" + q : ""}`, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  const ct = res.headers.get("Content-Type") || "";
  if (ct.includes("spreadsheetml") || ct.includes("octet-stream")) {
    const data = await res.arrayBuffer();
    return new Response(data, {
      status: res.status,
      headers: { "Content-Type": ct, "Content-Disposition": res.headers.get("Content-Disposition") || "" },
    });
  }
  return NextResponse.json(await res.json(), { status: res.status });
}

export async function PUT(
  request: NextRequest,
  context: { params: Promise<{ eegId: string; buchungId: string }> }
) {
  const { eegId, buchungId } = await context.params;
  const session = await auth();
  if (!session?.accessToken) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  const body = await request.json();
  const res = await fetch(`${API}/api/v1/eegs/${eegId}/ea/buchungen/${buchungId}`, {
    method: "PUT",
    headers: { Authorization: `Bearer ${session.accessToken}`, "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return NextResponse.json(await res.json(), { status: res.status });
}

export async function DELETE(
  req: NextRequest,
  context: { params: Promise<{ eegId: string; buchungId: string }> }
) {
  const { eegId, buchungId } = await context.params;
  const session = await auth();
  if (!session?.accessToken) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  const body = await req.text();
  const res = await fetch(`${API}/api/v1/eegs/${eegId}/ea/buchungen/${buchungId}`, {
    method: "DELETE",
    headers: {
      Authorization: `Bearer ${session.accessToken}`,
      ...(body ? { "Content-Type": "application/json" } : {}),
    },
    body: body || undefined,
  });
  if (res.status === 204) return new Response(null, { status: 204 });
  return NextResponse.json(await res.json(), { status: res.status });
}
