import { auth } from "@/lib/auth";
import { NextRequest } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  _request: NextRequest,
  context: { params: Promise<{  eegId: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken) {
    return new Response(JSON.stringify({ error: "Unauthorized" }), { status: 401 });
  }

  const res = await fetch(`${API}/api/v1/eegs/${params.eegId}/backup`, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });

  if (!res.ok) {
    const body = await res.text();
    return new Response(body, { status: res.status });
  }

  const data = await res.arrayBuffer();
  const contentDisposition = res.headers.get("Content-Disposition") || `attachment; filename="backup.json"`;

  return new Response(data, {
    status: 200,
    headers: {
      "Content-Type": "application/json",
      "Content-Disposition": contentDisposition,
    },
  });
}
