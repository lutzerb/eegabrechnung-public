import { auth } from "@/lib/auth";
const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function PUT(request: Request, context: { params: Promise<{  eegId: string; scheduleId: string  }> }) {
  const params = await context.params;
  const session = await auth();
  if (!session?.accessToken) return Response.json({ error: "Unauthorized" }, { status: 401 });
  const body = await request.json();
  const res = await fetch(`${API}/api/v1/eegs/${params.eegId}/tariffs/${params.scheduleId}/entries`, {
    method: "PUT",
    headers: { Authorization: `Bearer ${session.accessToken}`, "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return Response.json(await res.json(), { status: res.status });
}
