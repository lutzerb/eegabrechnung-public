import { auth } from "@/lib/auth";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function POST(
  request: Request,
  context: { params: Promise<{  eegId: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken)
    return Response.json({ error: "Unauthorized" }, { status: 401 });

  const formData = await request.formData();
  const res = await fetch(
    `${API}/api/v1/eegs/${params.eegId}/import/energiedaten/preview`,
    {
      method: "POST",
      headers: { Authorization: `Bearer ${session.accessToken}` },
      body: formData,
    }
  );
  const data = await res.json();
  return Response.json(data, { status: res.status });
}
