import { auth } from "@/lib/auth";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  request: Request,
  context: { params: Promise<{  eegId: string; msgId: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken) {
    return new Response(JSON.stringify({ error: "Unauthorized" }), { status: 401 });
  }

  const res = await fetch(
    `${API}/api/v1/eegs/${params.eegId}/eda/messages/${params.msgId}/xml`,
    { headers: { Authorization: `Bearer ${session.accessToken}` } }
  );

  if (!res.ok) {
    const body = await res.text();
    return new Response(body, { status: res.status });
  }

  const xml = await res.text();
  return new Response(xml, {
    status: 200,
    headers: {
      "Content-Type": "application/xml; charset=utf-8",
      "Content-Disposition": `attachment; filename="eda-message-${params.msgId}.xml"`,
    },
  });
}
