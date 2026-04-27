import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://eegabrechnung-api:8080";

export async function POST(
  req: NextRequest,
  context: { params: Promise<{ eegId: string }> }
) {
  const session = await auth();
  if (!session?.accessToken) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const { eegId } = await context.params;

  // Forward multipart/form-data as-is
  const formData = await req.formData();
  const upstream = new FormData();
  const file = formData.get("file");
  if (!file) {
    return NextResponse.json({ error: "file field required" }, { status: 400 });
  }
  upstream.append("file", file);

  const res = await fetch(`${API}/api/v1/eegs/${eegId}/sepa/camt054`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${session.accessToken}`,
    },
    body: upstream,
  });

  const data = await res.json();
  return NextResponse.json(data, { status: res.status });
}
