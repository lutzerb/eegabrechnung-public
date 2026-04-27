import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  request: NextRequest,
  context: { params: Promise<{ eegId: string }> }
) {
  const { eegId } = await context.params;
  const session = await auth();
  if (!session?.accessToken) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  const { searchParams } = new URL(request.url);
  const q = searchParams.get("aktiv") ? `?aktiv=${searchParams.get("aktiv")}` : "";
  const res = await fetch(`${API}/api/v1/eegs/${eegId}/ea/konten${q}`, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  return NextResponse.json(await res.json(), { status: res.status });
}

export async function POST(
  request: NextRequest,
  context: { params: Promise<{ eegId: string }> }
) {
  const { eegId } = await context.params;
  const session = await auth();
  if (!session?.accessToken) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  const body = await request.json();
  const res = await fetch(`${API}/api/v1/eegs/${eegId}/ea/konten`, {
    method: "POST",
    headers: { Authorization: `Bearer ${session.accessToken}`, "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return NextResponse.json(await res.json(), { status: res.status });
}
