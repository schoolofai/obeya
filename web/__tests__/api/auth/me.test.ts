import { describe, it, expect, vi, beforeEach } from "vitest";
import { ErrorCode, AppError } from "@/lib/errors";

const mockAuthenticate = vi.fn();

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: (...args: unknown[]) => mockAuthenticate(...args),
}));

import { GET } from "@/app/api/auth/me/route";

function getRequest(): Request {
  return new Request("http://localhost/api/auth/me", {
    method: "GET",
  });
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("GET /api/auth/me", () => {
  it("returns 200 with user data when authenticated", async () => {
    const user = { id: "user-1", email: "alice@example.com", name: "Alice" };
    mockAuthenticate.mockResolvedValue(user);

    const req = getRequest();
    const res = await GET(req);
    const body = await res.json();

    expect(res.status).toBe(200);
    expect(body).toEqual({ ok: true, data: user });
  });

  it("returns full user data when authenticated via API token", async () => {
    const user = { id: "user-2", email: "token-user@example.com", name: "Token User" };
    mockAuthenticate.mockResolvedValue(user);

    const req = new Request("http://localhost/api/auth/me", {
      method: "GET",
      headers: { authorization: "Bearer ob_tok_abc123" },
    });

    const res = await GET(req);
    const body = await res.json();

    expect(res.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.email).toBe("token-user@example.com");
    expect(body.data.name).toBe("Token User");
  });

  it("returns 401 when not authenticated", async () => {
    mockAuthenticate.mockRejectedValue(
      new AppError(ErrorCode.UNAUTHORIZED, "No authentication provided"),
    );

    const req = getRequest();
    const res = await GET(req);
    const body = await res.json();

    expect(res.status).toBe(401);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe(ErrorCode.UNAUTHORIZED);
  });
});
