import { auth } from "@/lib/auth";
const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  _request: Request,
  context: { params: Promise<{  eegId: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken) return new Response("Unauthorized", { status: 401 });
  const res = await fetch(`${API}/api/v1/eegs/${params.eegId}/logo`, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  if (!res.ok) return new Response("Not found", { status: 404 });
  const contentType = res.headers.get("Content-Type") || "image/png";
  const buffer = await res.arrayBuffer();
  return new Response(buffer, {
    headers: { "Content-Type": contentType, "Cache-Control": "no-cache" },
  });
}

export async function POST(
  request: Request,
  context: { params: Promise<{  eegId: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken) return Response.json({ error: "Unauthorized" }, { status: 401 });
  const formData = await request.formData();
  const res = await fetch(`${API}/api/v1/eegs/${params.eegId}/logo`, {
    method: "POST",
    headers: { Authorization: `Bearer ${session.accessToken}` },
    body: formData,
  });
  const data = await res.json();
  return Response.json(data, { status: res.status });
}
