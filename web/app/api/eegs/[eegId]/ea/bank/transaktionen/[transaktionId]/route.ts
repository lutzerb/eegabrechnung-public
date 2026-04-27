import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

const API = process.env.API_INTERNAL_URL || "http://localhost:8080";

export async function DELETE(
  _req: NextRequest,
  context: { params: Promise<{ eegId: string; transaktionId: string }> }
) {
  const { eegId, transaktionId } = await context.params;
  const session = await auth();
  if (!session?.accessToken) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  const res = await fetch(`${API}/api/v1/eegs/${eegId}/ea/bank/transaktionen/${transaktionId}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  if (res.status === 204) return new Response(null, { status: 204 });
  return NextResponse.json(await res.json(), { status: res.status });
}
