import { auth } from "@/lib/auth";
const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function PUT(
  request: Request,
  context: { params: Promise<{  eegId: string; id: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken) return Response.json({ error: "Unauthorized" }, { status: 401 });
  const body = await request.json();
  const res = await fetch(`${API}/api/v1/eegs/${params.eegId}/participations/${params.id}`, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${session.accessToken}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });
  const data = await res.json();
  return Response.json(data, { status: res.status });
}

export async function DELETE(
  _request: Request,
  context: { params: Promise<{  eegId: string; id: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken) return Response.json({ error: "Unauthorized" }, { status: 401 });
  const res = await fetch(`${API}/api/v1/eegs/${params.eegId}/participations/${params.id}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  if (res.status === 204) return new Response(null, { status: 204 });
  const data = await res.json();
  return Response.json(data, { status: res.status });
}
