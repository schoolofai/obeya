import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ErrorCode } from "@/lib/errors";

vi.mock("@/lib/appwrite/server", () => ({
  getUsers: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test-project",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "test-db",
  }),
}));

import { POST } from "@/app/api/auth/signup/route";
import { getUsers } from "@/lib/appwrite/server";

const mockCreate = vi.fn();
const mockGetUsers = vi.mocked(getUsers);
const mockFetch = vi.fn();
const originalFetch = global.fetch;

function jsonRequest(body: unknown): Request {
  return new Request("http://localhost/api/auth/signup", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

beforeEach(() => {
  vi.clearAllMocks();
  mockGetUsers.mockReturnValue({ create: mockCreate } as unknown as ReturnType<typeof getUsers>);
  global.fetch = mockFetch;
});

afterEach(() => {
  global.fetch = originalFetch;
});

describe("POST /api/auth/signup", () => {
  it("returns 201 with user and session data on successful signup", async () => {
    mockCreate.mockResolvedValue({
      $id: "user-123",
      email: "alice@example.com",
      name: "Alice",
    });

    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        $id: "session-456",
        userId: "user-123",
        secret: "session-secret",
      }),
    });

    const req = jsonRequest({
      email: "alice@example.com",
      password: "securepass123",
      name: "Alice",
    });

    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(201);
    expect(body).toEqual({
      ok: true,
      data: {
        user: { id: "user-123", email: "alice@example.com", name: "Alice" },
        session: { id: "session-456" },
      },
    });
    expect(mockCreate).toHaveBeenCalledWith(
      expect.any(String),
      "alice@example.com",
      undefined,
      "securepass123",
      "Alice",
    );
  });

  it("sets httpOnly session cookie on successful signup", async () => {
    mockCreate.mockResolvedValue({
      $id: "user-123",
      email: "alice@example.com",
      name: "Alice",
    });

    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        $id: "session-456",
        userId: "user-123",
        secret: "session-secret",
      }),
    });

    const req = jsonRequest({
      email: "alice@example.com",
      password: "securepass123",
      name: "Alice",
    });

    const res = await POST(req);
    const cookie = res.headers.get("set-cookie");

    expect(cookie).toBeTruthy();
    expect(cookie).toContain("obeya_session=user-123");
    expect(cookie).toContain("HttpOnly");
    expect(cookie).toContain("Path=/");
    expect(cookie).toContain("SameSite=lax");
  });

  it("returns 500 when session creation fails after user is created", async () => {
    mockCreate.mockResolvedValue({
      $id: "user-123",
      email: "alice@example.com",
      name: "Alice",
    });

    mockFetch.mockResolvedValue({
      ok: false,
      status: 429,
      json: async () => ({ message: "Rate limited" }),
    });

    const req = jsonRequest({
      email: "alice@example.com",
      password: "securepass123",
      name: "Alice",
    });

    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(500);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe(ErrorCode.INTERNAL_ERROR);
    expect(body.error.message).toBe("Rate limited");
  });

  it("calls Appwrite session endpoint with correct parameters", async () => {
    mockCreate.mockResolvedValue({
      $id: "user-123",
      email: "alice@example.com",
      name: "Alice",
    });

    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        $id: "session-456",
        userId: "user-123",
        secret: "session-secret",
      }),
    });

    const req = jsonRequest({
      email: "alice@example.com",
      password: "securepass123",
      name: "Alice",
    });

    await POST(req);

    expect(mockFetch).toHaveBeenCalledWith(
      "https://test.appwrite.io/v1/account/sessions/email",
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Appwrite-Project": "test-project",
        },
        body: JSON.stringify({
          email: "alice@example.com",
          password: "securepass123",
        }),
      },
    );
  });

  it("returns 400 when email is invalid", async () => {
    const req = jsonRequest({
      email: "not-an-email",
      password: "securepass123",
      name: "Alice",
    });

    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(400);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe(ErrorCode.VALIDATION_ERROR);
  });

  it("returns 400 when password is too short", async () => {
    const req = jsonRequest({
      email: "alice@example.com",
      password: "short",
      name: "Alice",
    });

    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(400);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe(ErrorCode.VALIDATION_ERROR);
  });

  it("returns 400 when name is missing", async () => {
    const req = jsonRequest({
      email: "alice@example.com",
      password: "securepass123",
    });

    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(400);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe(ErrorCode.VALIDATION_ERROR);
  });

  it("returns 409 when email already exists", async () => {
    mockCreate.mockRejectedValue({ code: 409, message: "User already exists" });

    const req = jsonRequest({
      email: "existing@example.com",
      password: "securepass123",
      name: "Alice",
    });

    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(409);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe(ErrorCode.EMAIL_ALREADY_EXISTS);
  });
});
