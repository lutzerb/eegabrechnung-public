import { auth } from "@/lib/auth";
import { NextRequest, NextResponse } from "next/server";

interface Params {
  params: Promise<{ eegId: string; file: string }>;
}

export async function GET(req: NextRequest, context: Params) {
  const session = await auth();
  if (!session?.accessToken) {
    return new NextResponse("Unauthorized", { status: 401 });
  }

  const params = await context.params;
  const { eegId, file } = params;
  if (file !== "pain001" && file !== "pain008") {
    return new NextResponse("Not Found", { status: 404 });
  }

  const apiBase =
    process.env.API_INTERNAL_URL ||
    process.env.NEXT_PUBLIC_API_URL ||
    "http://localhost:8101";

  // Forward any query params (period_start, period_end)
  const search = req.nextUrl.searchParams.toString();
  const apiUrl = `${apiBase}/api/v1/eegs/${eegId}/sepa/${file}${search ? "?" + search : ""}`;

  const upstream = await fetch(apiUrl, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });

  if (!upstream.ok) {
    let msg = `HTTP ${upstream.status}`;
    try {
      const body = await upstream.json();
      msg = body.message || body.error || msg;
    } catch {
      // ignore
    }
    return new NextResponse(msg, { status: upstream.status });
  }

  const xml = await upstream.arrayBuffer();
  const filename =
    upstream.headers.get("Content-Disposition")?.match(/filename="([^"]+)"/)?.[1] ||
    `${file}_${eegId}.xml`;

  return new NextResponse(xml, {
    headers: {
      "Content-Type": "application/xml; charset=UTF-8",
      "Content-Disposition": `attachment; filename="${filename}"`,
    },
  });
}
