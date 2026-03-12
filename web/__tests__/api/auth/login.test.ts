import { describe, it, expect, vi, beforeEach } from "vitest";
import { ErrorCode } from "@/lib/errors";

const mockCreateEmailPasswordSession = vi.fn();
const mockAccountGet = vi.fn();

vi.mock("node-appwrite", () => {
  class MockClient {
    setEndpoint() { return this; }
    setProject() { return this; }
    setSession() { return this; }
  }

  let accountCallCount = 0;
  class MockAccount {
    createEmailPasswordSession: typeof mockCreateEmailPasswordSession;
    get: typeof mockAccountGet;
    constructor() {
      accountCallCount++;
      this.createEmailPasswordSession = mockCreateEmailPasswordSession;
      this.get = mockAccountGet;
    }
  }

  return {
    Client: MockClient,
    Account: MockAccount,
    // provide a reset hook for beforeEach
    __resetAccountCount: () => { accountCallCount = 0; },
  };
});

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test-project",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "test-db",
  }),
}));

import { POST } from "@/app/api/auth/login/route";

function jsonRequest(body: unknown): Request {
  return new Request("http://localhost/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("POST /api/auth/login", () => {
  it("returns 200 with user and session data on success", async () => {
    mockCreateEmailPasswordSession.mockResolvedValue({
      $id: "session-456",
      secret: "session-secret-xyz",
    });

    mockAccountGet.mockResolvedValue({
      $id: "user-123",
      email: "alice@example.com",
      name: "Alice",
    });

    const req = jsonRequest({
      email: "alice@example.com",
      password: "securepass123",
    });

    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(200);
    expect(body).toEqual({
      ok: true,
      data: {
        user: { id: "user-123", email: "alice@example.com", name: "Alice" },
        session: { id: "session-456", secret: "session-secret-xyz" },
      },
    });
    expect(mockCreateEmailPasswordSession).toHaveBeenCalledWith(
      "alice@example.com",
      "securepass123",
    );
  });

  it("returns 400 when email is invalid", async () => {
    const req = jsonRequest({
      email: "not-an-email",
      password: "securepass123",
    });

    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(400);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe(ErrorCode.VALIDATION_ERROR);
  });

  it("returns 400 when password is missing", async () => {
    const req = jsonRequest({
      email: "alice@example.com",
    });

    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(400);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe(ErrorCode.VALIDATION_ERROR);
  });

  it("returns 401 when credentials are invalid", async () => {
    mockCreateEmailPasswordSession.mockRejectedValue({
      code: 401,
      message: "Invalid credentials",
    });

    const req = jsonRequest({
      email: "alice@example.com",
      password: "wrongpassword",
    });

    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(401);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe(ErrorCode.INVALID_CREDENTIALS);
  });
});
