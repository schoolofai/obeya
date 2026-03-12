import { NextResponse } from "next/server";

export async function GET() {
  return NextResponse.json({
    ok: true,
    data: {
      status: "healthy",
      version: "0.1.0",
      service: "obeya-cloud",
    },
  });
}
