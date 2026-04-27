import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  request: NextRequest,
  context: { params: Promise<{ eegId: string; type: string }> }
) {
  const { eegId, type } = await context.params;
  const session = await auth();
  if (!session?.accessToken) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  const q = new URL(request.url).searchParams.toString();
  const res = await fetch(`${API}/api/v1/eegs/${eegId}/ea/erklaerungen/${type}${q ? "?" + q : ""}`, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  const ct = res.headers.get("Content-Type") || "";
  if (ct.includes("xml") || ct.includes("octet-stream")) {
    const data = await res.arrayBuffer();
    return new Response(data, {
      status: res.status,
      headers: { "Content-Type": ct, "Content-Disposition": res.headers.get("Content-Disposition") || "" },
    });
  }
  return NextResponse.json(await res.json(), { status: res.status });
}
