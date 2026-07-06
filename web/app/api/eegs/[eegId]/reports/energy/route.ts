import { auth } from "@/lib/auth";
import { NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function GET(
  request: Request,
  context: { params: Promise<{  eegId: string  }> }
) {
  const params = await context.params;
const session = await auth();
  if (!session?.accessToken)
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });

  const { searchParams } = new URL(request.url);
  const qs = searchParams.toString();
  const res = await fetch(
    `${API}/api/v1/eegs/${params.eegId}/reports/energy${qs ? `?${qs}` : ""}`,
    { headers: { Authorization: `Bearer ${session.accessToken}` } }
  );
  return NextResponse.json(await res.json(), { status: res.status });
}
