import { auth } from "@/lib/auth";
import { NextRequest } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  _request: NextRequest,
  context: { params: Promise<{ eegId: string }> }
) {
  const params = await context.params;
  const session = await auth();
  if (!session?.accessToken)
    return new Response(JSON.stringify({ error: "Unauthorized" }), { status: 401 });
  const res = await fetch(`${API}/api/v1/eegs/${params.eegId}/forecast`, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  const body = await res.text();
  return new Response(body, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}

export async function POST(
  _request: NextRequest,
  context: { params: Promise<{ eegId: string }> }
) {
  const params = await context.params;
  const session = await auth();
  if (!session?.accessToken)
    return new Response(JSON.stringify({ error: "Unauthorized" }), { status: 401 });
  const res = await fetch(`${API}/api/v1/eegs/${params.eegId}/forecast/train`, {
    method: "POST",
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  const body = await res.text();
  return new Response(body, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}
