import { auth } from "@/lib/auth";
const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(request: Request, context: { params: Promise<{  eegId: string  }> }) {
  const params = await context.params;
  const session = await auth();
  if (!session?.accessToken) return Response.json({ error: "Unauthorized" }, { status: 401 });
  const res = await fetch(`${API}/api/v1/eegs/${params.eegId}/tariffs`, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  return Response.json(await res.json(), { status: res.status });
}

export async function POST(request: Request, context: { params: Promise<{  eegId: string  }> }) {
  const params = await context.params;
  const session = await auth();
  if (!session?.accessToken) return Response.json({ error: "Unauthorized" }, { status: 401 });
  const body = await request.json();
  const res = await fetch(`${API}/api/v1/eegs/${params.eegId}/tariffs`, {
    method: "POST",
    headers: { Authorization: `Bearer ${session.accessToken}`, "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return Response.json(await res.json(), { status: res.status });
}
