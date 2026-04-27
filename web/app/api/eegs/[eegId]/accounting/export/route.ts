import { auth } from "@/lib/auth";
import { NextRequest } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  request: NextRequest,
  context: { params: Promise<{  eegId: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken) {
    return new Response(JSON.stringify({ error: "Unauthorized" }), { status: 401 });
  }

  const { searchParams } = new URL(request.url);
  const from = searchParams.get("from") || "";
  const to = searchParams.get("to") || "";
  const format = searchParams.get("format") || "xlsx";

  const res = await fetch(
    `${API}/api/v1/eegs/${params.eegId}/accounting/export?from=${from}&to=${to}&format=${format}`,
    {
      headers: { Authorization: `Bearer ${session.accessToken}` },
    }
  );

  if (!res.ok) {
    const body = await res.text();
    return new Response(body, { status: res.status });
  }

  const data = await res.arrayBuffer();
  const contentType = res.headers.get("Content-Type") || "application/octet-stream";
  const contentDisposition = res.headers.get("Content-Disposition") || "";

  return new Response(data, {
    status: 200,
    headers: {
      "Content-Type": contentType,
      "Content-Disposition": contentDisposition,
    },
  });
}
