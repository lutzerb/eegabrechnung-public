import { auth } from "@/lib/auth";
const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function DELETE(request: Request, context: { params: Promise<{ eegId: string; memberId: string; extraMeterId: string; readingId: string }> }) {
  const params = await context.params;
  const session = await auth();
  if (!session?.accessToken) return Response.json({ error: "Unauthorized" }, { status: 401 });
  const res = await fetch(
    `${API}/api/v1/eegs/${params.eegId}/members/${params.memberId}/extra-meters/${params.extraMeterId}/readings/${params.readingId}`,
    { method: "DELETE", headers: { Authorization: `Bearer ${session.accessToken}` } },
  );
  return Response.json(await res.json(), { status: res.status });
}
