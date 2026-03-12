import { describe, it, expect, vi } from "vitest";
import { ok, fail, handleError } from "@/lib/response";
import { AppError, ErrorCode } from "@/lib/errors";

describe("ok()", () => {
  it("returns 200 with data wrapped in envelope", async () => {
    const res = ok({ id: 1 });
    expect(res.status).toBe(200);
    const body = await res.json();
    expect(body).toEqual({ ok: true, data: { id: 1 } });
  });

  it("returns custom status when specified", async () => {
    const res = ok({ id: 2 }, { status: 201 });
    expect(res.status).toBe(201);
    const body = await res.json();
    expect(body).toEqual({ ok: true, data: { id: 2 } });
  });

  it("includes meta when provided", async () => {
    const res = ok([1, 2, 3], { meta: { total: 50, page: 1 } });
    expect(res.status).toBe(200);
    const body = await res.json();
    expect(body).toEqual({
      ok: true,
      data: [1, 2, 3],
      meta: { total: 50, page: 1 },
    });
  });
});

describe("fail()", () => {
  it("returns correct status and error envelope", async () => {
    const res = fail(ErrorCode.BOARD_NOT_FOUND, "Board xyz not found");
    expect(res.status).toBe(404);
    const body = await res.json();
    expect(body).toEqual({
      ok: false,
      error: { code: "BOARD_NOT_FOUND", message: "Board xyz not found" },
    });
  });

  it("maps error code to correct HTTP status", async () => {
    const res = fail(ErrorCode.VALIDATION_ERROR, "bad input");
    expect(res.status).toBe(400);
  });
});

describe("handleError()", () => {
  it("formats AppError into error envelope", async () => {
    const err = new AppError(ErrorCode.FORBIDDEN, "no access");
    const res = handleError(err);
    expect(res.status).toBe(403);
    const body = await res.json();
    expect(body).toEqual({
      ok: false,
      error: { code: "FORBIDDEN", message: "no access" },
    });
  });

  it("returns 500 with generic message for unknown errors", async () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    const res = handleError(new TypeError("something broke internally"));
    expect(res.status).toBe(500);
    const body = await res.json();
    expect(body).toEqual({
      ok: false,
      error: { code: "INTERNAL_ERROR", message: "Internal server error" },
    });
    expect(body.error.message).not.toContain("something broke internally");
    spy.mockRestore();
  });
});
