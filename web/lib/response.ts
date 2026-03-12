import { NextResponse } from "next/server";
import { AppError, ErrorCode } from "@/lib/errors";

interface OkOptions {
  status?: number;
  meta?: Record<string, unknown>;
}

export function ok(data: unknown, options: OkOptions = {}): NextResponse {
  const { status = 200, meta } = options;
  const body: Record<string, unknown> = { ok: true, data };
  if (meta) body.meta = meta;
  return NextResponse.json(body, { status });
}

export function fail(code: ErrorCode, message: string): NextResponse {
  const err = new AppError(code, message);
  return NextResponse.json(
    { ok: false, error: { code: err.code, message: err.message } },
    { status: err.statusCode }
  );
}

export function handleError(err: unknown): NextResponse {
  if (err instanceof AppError) {
    return NextResponse.json(
      { ok: false, error: { code: err.code, message: err.message } },
      { status: err.statusCode }
    );
  }
  console.error("Unhandled error:", err);
  return NextResponse.json(
    {
      ok: false,
      error: { code: ErrorCode.INTERNAL_ERROR, message: "Internal server error" },
    },
    { status: 500 }
  );
}
