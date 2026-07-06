import { cookies } from "next/headers";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  _req: Request,
  context: { params: Promise<{  docId: string  }> }
) {
  const params = await context.params;
  const cookieStore = await cookies();
  const sessionToken = cookieStore.get("portal_session")?.value;
  if (!sessionToken) return new Response("Unauthorized", { status: 401 });

  const res = await fetch(
    `${API}/api/v1/public/portal/documents/${params.docId}`,
    {
      headers: { "X-Portal-Session": sessionToken },
    }
  );

  if (!res.ok) return new Response("Not found", { status: 404 });

  const blob = await res.arrayBuffer();
  const contentType =
    res.headers.get("Content-Type") || "application/octet-stream";
  const contentDisposition =
    res.headers.get("Content-Disposition") || `attachment; filename="document"`;

  return new Response(blob, {
    headers: {
      "Content-Type": contentType,
      "Content-Disposition": contentDisposition,
    },
  });
}
