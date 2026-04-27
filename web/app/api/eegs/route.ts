import { auth } from "@/lib/auth";
const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET() {
  const session = await auth();
  if (!session?.accessToken) return Response.json({ error: "Unauthorized" }, { status: 401 });
  const res = await fetch(`${API}/api/v1/eegs`, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  return Response.json(await res.json(), { status: res.status });
}
