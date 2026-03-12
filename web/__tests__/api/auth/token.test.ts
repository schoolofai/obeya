import { describe, it, expect, vi, beforeEach } from "vitest";
import { ErrorCode } from "@/lib/errors";

const mockAuthenticate = vi.fn();
const mockCreateDocument = vi.fn();
const mockGetDocument = vi.fn();
const mockDeleteDocument = vi.fn();
const mockGenerateToken = vi.fn();
const mockHashToken = vi.fn();

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: (...args: unknown[]) => mockAuthenticate(...args),
}));

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: () => ({
    createDocument: mockCreateDocument,
    getDocument: mockGetDocument,
    deleteDocument: mockDeleteDocument,
  }),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_ENDPOINT: "https://test.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "test-project",
    APPWRITE_API_KEY: "test-key",
    APPWRITE_DATABASE_ID: "test-db",
  }),
}));

vi.mock("@/lib/auth/tokens", () => ({
  generateToken: () => mockGenerateToken(),
  hashToken: (token: string) => mockHashToken(token),
}));

vi.mock("node-appwrite", () => ({
  ID: { unique: () => "unique-id" },
}));

import { POST } from "@/app/api/auth/token/route";
import { DELETE } from "@/app/api/auth/token/[id]/route";
import { AppError } from "@/lib/errors";

function jsonRequest(body: unknown, url = "http://localhost/api/auth/token"): Request {
  return new Request(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

function deleteRequest(id: string): Request {
  return new Request(`http://localhost/api/auth/token/${id}`, {
    method: "DELETE",
  });
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("POST /api/auth/token", () => {
  it("returns 201 with token starting ob_tok_", async () => {
    const rawToken = "ob_tok_abc123def456";
    mockAuthenticate.mockResolvedValue({ id: "user-1", email: "a@b.com", name: "A" });
    mockGenerateToken.mockReturnValue(rawToken);
    mockHashToken.mockResolvedValue("hashed-token");
    mockCreateDocument.mockResolvedValue({ $id: "tok-1" });

    const req = jsonRequest({ name: "My CLI token" });
    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(201);
    expect(body.ok).toBe(true);
    expect(body.data.token).toBe(rawToken);
    expect(body.data.token.startsWith("ob_tok_")).toBe(true);
    expect(body.data.id).toBe("tok-1");
    expect(body.data.name).toBe("My CLI token");
    expect(body.data.scopes).toEqual(["*"]);
    expect(mockCreateDocument).toHaveBeenCalledWith(
      "test-db",
      "api_tokens",
      "unique-id",
      expect.objectContaining({
        user_id: "user-1",
        token_hash: "hashed-token",
        name: "My CLI token",
        scopes: ["*"],
      }),
    );
  });

  it("returns 201 with custom scopes", async () => {
    mockAuthenticate.mockResolvedValue({ id: "user-1", email: "a@b.com", name: "A" });
    mockGenerateToken.mockReturnValue("ob_tok_xyz");
    mockHashToken.mockResolvedValue("hashed");
    mockCreateDocument.mockResolvedValue({ $id: "tok-2" });

    const req = jsonRequest({ name: "Read-only", scopes: ["boards:read"] });
    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(201);
    expect(body.data.scopes).toEqual(["boards:read"]);
  });

  it("returns 400 when name is missing", async () => {
    mockAuthenticate.mockResolvedValue({ id: "user-1", email: "a@b.com", name: "A" });

    const req = jsonRequest({});
    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(400);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe(ErrorCode.VALIDATION_ERROR);
  });

  it("returns 401 when not authenticated", async () => {
    mockAuthenticate.mockRejectedValue(
      new AppError(ErrorCode.UNAUTHORIZED, "No authentication provided"),
    );

    const req = jsonRequest({ name: "test" });
    const res = await POST(req);
    const body = await res.json();

    expect(res.status).toBe(401);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe(ErrorCode.UNAUTHORIZED);
  });
});

describe("DELETE /api/auth/token/[id]", () => {
  const params = Promise.resolve({ id: "tok-1" });

  it("returns 200 with revoked:true on success", async () => {
    mockAuthenticate.mockResolvedValue({ id: "user-1", email: "a@b.com", name: "A" });
    mockGetDocument.mockResolvedValue({ $id: "tok-1", user_id: "user-1" });
    mockDeleteDocument.mockResolvedValue({});

    const req = deleteRequest("tok-1");
    const res = await DELETE(req, { params });
    const body = await res.json();

    expect(res.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toEqual({ revoked: true });
    expect(mockDeleteDocument).toHaveBeenCalledWith("test-db", "api_tokens", "tok-1");
  });

  it("returns 403 when token belongs to another user", async () => {
    mockAuthenticate.mockResolvedValue({ id: "user-1", email: "a@b.com", name: "A" });
    mockGetDocument.mockResolvedValue({ $id: "tok-1", user_id: "user-other" });

    const req = deleteRequest("tok-1");
    const res = await DELETE(req, { params });
    const body = await res.json();

    expect(res.status).toBe(403);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe(ErrorCode.FORBIDDEN);
    expect(mockDeleteDocument).not.toHaveBeenCalled();
  });

  it("returns 401 when not authenticated", async () => {
    mockAuthenticate.mockRejectedValue(
      new AppError(ErrorCode.UNAUTHORIZED, "No authentication provided"),
    );

    const req = deleteRequest("tok-1");
    const res = await DELETE(req, { params });
    const body = await res.json();

    expect(res.status).toBe(401);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe(ErrorCode.UNAUTHORIZED);
  });

  it("returns 500 when token ID does not exist", async () => {
    mockAuthenticate.mockResolvedValue({ id: "user-1", email: "a@b.com", name: "A" });
    mockGetDocument.mockRejectedValue(new Error("Document not found"));

    const notFoundParams = Promise.resolve({ id: "nonexistent" });
    const req = deleteRequest("nonexistent");
    const res = await DELETE(req, { params: notFoundParams });
    const body = await res.json();

    expect(res.status).toBe(500);
    expect(body.ok).toBe(false);
    expect(mockDeleteDocument).not.toHaveBeenCalled();
  });
});
