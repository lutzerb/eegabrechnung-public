import { auth } from "@/lib/auth";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  _req: Request,
  context: { params: Promise<{  eegId: string; docId: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken) {
    return new Response("Unauthorized", { status: 401 });
  }

  const res = await fetch(
    `${API}/api/v1/eegs/${params.eegId}/documents/${params.docId}/download`,
    {
      headers: { Authorization: `Bearer ${session.accessToken}` },
    }
  );

  if (!res.ok) return new Response("Not found", { status: 404 });

  const blob = await res.arrayBuffer();
  const contentType = res.headers.get("Content-Type") || "application/octet-stream";
  const contentDisposition =
    res.headers.get("Content-Disposition") || `attachment; filename="document"`;

  return new Response(blob, {
    headers: {
      "Content-Type": contentType,
      "Content-Disposition": contentDisposition,
    },
  });
}

export async function PATCH(
  req: Request,
  context: { params: Promise<{ eegId: string; docId: string }> }
) {
  const params = await context.params;
  const session = await auth();
  if (!session?.accessToken) {
    return Response.json({ error: "Unauthorized" }, { status: 401 });
  }

  const body = await req.json().catch(() => ({}));
  const res = await fetch(
    `${API}/api/v1/eegs/${params.eegId}/documents/${params.docId}`,
    {
      method: "PATCH",
      headers: {
        Authorization: `Bearer ${session.accessToken}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body),
    }
  );

  const data = await res.json().catch(() => ({}));
  return Response.json(data, { status: res.status });
}

export async function DELETE(
  _req: Request,
  context: { params: Promise<{  eegId: string; docId: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken) {
    return Response.json({ error: "Unauthorized" }, { status: 401 });
  }

  const res = await fetch(
    `${API}/api/v1/eegs/${params.eegId}/documents/${params.docId}`,
    {
      method: "DELETE",
      headers: { Authorization: `Bearer ${session.accessToken}` },
    }
  );

  if (res.status === 204 || res.ok) {
    return new Response(null, { status: 204 });
  }

  const data = await res.json().catch(() => ({}));
  return Response.json(data, { status: res.status });
}
