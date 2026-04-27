const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  _req: Request,
  context: { params: Promise<{ eegId: string; docId: string }> }
) {
  const params = await context.params;
  const res = await fetch(
    `${API}/api/v1/public/eegs/${params.eegId}/documents/${params.docId}`,
    { cache: "no-store" }
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
