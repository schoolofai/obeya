import { describe, it, expect, vi, beforeEach } from "vitest";
import { authenticate } from "@/lib/auth/middleware";
import { ErrorCode } from "@/lib/errors";

vi.mock("@/lib/auth/session", () => ({
  getUserFromSession: vi.fn(),
}));

vi.mock("@/lib/auth/tokens", () => ({
  verifyToken: vi.fn(),
}));

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
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

import { getUserFromSession } from "@/lib/auth/session";
import { verifyToken } from "@/lib/auth/tokens";
import { getDatabases, getUsers } from "@/lib/appwrite/server";

const mockGetUserFromSession = vi.mocked(getUserFromSession);
const mockVerifyToken = vi.mocked(verifyToken);
const mockGetDatabases = vi.mocked(getDatabases);
const mockGetUsers = vi.mocked(getUsers);

function buildRequest(headers: Record<string, string> = {}): Request {
  return new Request("https://example.com/api/test", { headers });
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("authenticate", () => {
  it("throws UNAUTHORIZED when no auth headers are provided", async () => {
    const req = buildRequest();

    await expect(authenticate(req)).rejects.toMatchObject({
      code: ErrorCode.UNAUTHORIZED,
      message: "No authentication provided",
    });
  });

  it("returns user from valid session cookie", async () => {
    const expectedUser = { id: "user-1", email: "test@example.com", name: "Test" };
    mockGetUserFromSession.mockResolvedValue(expectedUser);

    const req = buildRequest({ cookie: "a_session=abc123" });
    const user = await authenticate(req);

    expect(user).toEqual(expectedUser);
    expect(mockGetUserFromSession).toHaveBeenCalledWith("a_session=abc123");
  });

  it("throws when session cookie is invalid (no silent fallback)", async () => {
    mockGetUserFromSession.mockRejectedValue(new Error("Session expired"));

    const req = buildRequest({ cookie: "a_session=expired" });

    await expect(authenticate(req)).rejects.toThrow("Session expired");
  });

  it("prefers Bearer token over cookie when both present", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [
          { $id: "tok-1", user_id: "user-2", token_hash: "hash-abc" },
        ],
      }),
      updateDocument: vi.fn().mockResolvedValue({}),
    };
    mockGetDatabases.mockReturnValue(mockDb as unknown as ReturnType<typeof getDatabases>);
    mockVerifyToken.mockResolvedValue(true);
    mockGetUsers.mockReturnValue({
      get: vi.fn().mockResolvedValue({ $id: "user-2", email: "token@example.com", name: "Token User" }),
    } as unknown as ReturnType<typeof getUsers>);

    const req = buildRequest({
      cookie: "a_session=something",
      authorization: "Bearer ob_tok_valid123",
    });

    const user = await authenticate(req);

    expect(user).toEqual({ id: "user-2", email: "token@example.com", name: "Token User" });
    expect(mockGetUserFromSession).not.toHaveBeenCalled();
  });

  it("returns full user data when authenticating with Bearer token", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [
          { $id: "tok-1", user_id: "user-2", token_hash: "hash-abc" },
        ],
      }),
      updateDocument: vi.fn().mockResolvedValue({}),
    };
    mockGetDatabases.mockReturnValue(mockDb as unknown as ReturnType<typeof getDatabases>);
    mockVerifyToken.mockResolvedValue(true);
    mockGetUsers.mockReturnValue({
      get: vi.fn().mockResolvedValue({ $id: "user-2", email: "alice@example.com", name: "Alice" }),
    } as unknown as ReturnType<typeof getUsers>);

    const req = buildRequest({ authorization: "Bearer ob_tok_valid123" });
    const user = await authenticate(req);

    expect(user).toEqual({ id: "user-2", email: "alice@example.com", name: "Alice" });
    expect(mockVerifyToken).toHaveBeenCalledWith("ob_tok_valid123", "hash-abc");
  });

  it("throws UNAUTHORIZED when Bearer token does not match any stored hash", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [
          { $id: "tok-1", user_id: "user-2", token_hash: "hash-abc" },
        ],
      }),
      updateDocument: vi.fn().mockResolvedValue({}),
    };
    mockGetDatabases.mockReturnValue(mockDb as unknown as ReturnType<typeof getDatabases>);
    mockVerifyToken.mockResolvedValue(false);

    const req = buildRequest({ authorization: "Bearer ob_tok_invalid" });

    await expect(authenticate(req)).rejects.toMatchObject({
      code: ErrorCode.UNAUTHORIZED,
      message: "Invalid API token",
    });
  });

  it("updates last_used_at when token matches", async () => {
    const mockUpdateDocument = vi.fn().mockResolvedValue({});
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [
          { $id: "tok-99", user_id: "user-3", token_hash: "hash-xyz" },
        ],
      }),
      updateDocument: mockUpdateDocument,
    };
    mockGetDatabases.mockReturnValue(mockDb as unknown as ReturnType<typeof getDatabases>);
    mockVerifyToken.mockResolvedValue(true);
    mockGetUsers.mockReturnValue({
      get: vi.fn().mockResolvedValue({ $id: "user-3", email: "bob@example.com", name: "Bob" }),
    } as unknown as ReturnType<typeof getUsers>);

    const req = buildRequest({ authorization: "Bearer ob_tok_good" });
    await authenticate(req);

    expect(mockUpdateDocument).toHaveBeenCalledWith(
      "test-db",
      "api_tokens",
      "tok-99",
      expect.objectContaining({ last_used_at: expect.any(String) }),
    );
  });

  it("paginates through tokens when first batch has no match", async () => {
    const batch1 = Array.from({ length: 100 }, (_, i) => ({
      $id: `tok-${i}`,
      user_id: "user-1",
      token_hash: `hash-${i}`,
    }));
    const batch2 = [
      { $id: "tok-100", user_id: "user-1", token_hash: "hash-match" },
    ];

    const mockDb = {
      listDocuments: vi.fn()
        .mockResolvedValueOnce({ documents: batch1 })
        .mockResolvedValueOnce({ documents: batch2 }),
      updateDocument: vi.fn().mockResolvedValue({}),
    };
    mockGetDatabases.mockReturnValue(mockDb as unknown as ReturnType<typeof getDatabases>);

    // All batch1 tokens don't match, batch2's single token matches
    mockVerifyToken.mockImplementation(async (_token: string, hash: string) => {
      return hash === "hash-match";
    });

    mockGetUsers.mockReturnValue({
      get: vi.fn().mockResolvedValue({ $id: "user-1", email: "page2@example.com", name: "Page2" }),
    } as unknown as ReturnType<typeof getUsers>);

    const req = buildRequest({ authorization: "Bearer ob_tok_page2" });
    const user = await authenticate(req);

    expect(user).toEqual({ id: "user-1", email: "page2@example.com", name: "Page2" });
    expect(mockDb.listDocuments).toHaveBeenCalledTimes(2);
  });
});
