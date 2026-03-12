import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn(),
}));

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

vi.mock("@/lib/permissions", () => ({
  requireBoardAccess: vi.fn(),
  BOARD_ROLE_LEVEL: { viewer: 1, editor: 2, owner: 3 },
}));

import { GET, POST } from "@/app/api/boards/[id]/members/route";
import { PATCH, DELETE } from "@/app/api/boards/[id]/members/[uid]/route";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { requireBoardAccess } from "@/lib/permissions";

type MembersContext = { params: Promise<{ id: string }> };
type MemberContext = { params: Promise<{ id: string; uid: string }> };

function makeMembersCtx(id: string): MembersContext {
  return { params: Promise.resolve({ id }) };
}

function makeMemberCtx(id: string, uid: string): MemberContext {
  return { params: Promise.resolve({ id, uid }) };
}

describe("GET /api/boards/:id/members", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns list of board members", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireBoardAccess).mockResolvedValue(undefined);
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        org_id: null,
      }),
      listDocuments: vi.fn().mockResolvedValue({
        total: 2,
        documents: [
          { $id: "mem-1", user_id: "user-1", board_id: "board-1", role: "owner" },
          { $id: "mem-2", user_id: "user-2", board_id: "board-1", role: "editor" },
        ],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/members");
    const response = await GET(request, makeMembersCtx("board-1"));
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data).toHaveLength(2);
    expect(requireBoardAccess).toHaveBeenCalledWith("user-1", "board-1", null, "viewer");
  });
});

describe("POST /api/boards/:id/members", () => {
  beforeEach(() => vi.clearAllMocks());

  it("adds a new board member", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireBoardAccess).mockResolvedValue(undefined);
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        org_id: null,
      }),
      listDocuments: vi.fn().mockResolvedValue({ documents: [] }),
      createDocument: vi.fn().mockResolvedValue({
        $id: "mem-new",
        user_id: "user-2",
        board_id: "board-1",
        role: "editor",
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/members", {
      method: "POST",
      body: JSON.stringify({ user_id: "user-2", role: "editor" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, makeMembersCtx("board-1"));
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.data.user_id).toBe("user-2");
    expect(requireBoardAccess).toHaveBeenCalledWith("user-1", "board-1", null, "owner");
  });

  it("returns 409 when member already exists", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireBoardAccess).mockResolvedValue(undefined);
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        org_id: null,
      }),
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ $id: "mem-1", user_id: "user-2", board_id: "board-1", role: "editor" }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/members", {
      method: "POST",
      body: JSON.stringify({ user_id: "user-2", role: "editor" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, makeMembersCtx("board-1"));
    expect(response.status).toBe(409);
  });

  it("returns 400 when role is invalid", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireBoardAccess).mockResolvedValue(undefined);
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        org_id: null,
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/members", {
      method: "POST",
      body: JSON.stringify({ user_id: "user-2", role: "owner" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, makeMembersCtx("board-1"));
    expect(response.status).toBe(400);
  });
});

describe("PATCH /api/boards/:id/members/:uid", () => {
  beforeEach(() => vi.clearAllMocks());

  it("updates board member role", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireBoardAccess).mockResolvedValue(undefined);
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        org_id: null,
      }),
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ $id: "mem-1", user_id: "user-2", board_id: "board-1", role: "viewer" }],
      }),
      updateDocument: vi.fn().mockResolvedValue({
        $id: "mem-1",
        user_id: "user-2",
        board_id: "board-1",
        role: "editor",
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/members/user-2", {
      method: "PATCH",
      body: JSON.stringify({ role: "editor" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await PATCH(request, makeMemberCtx("board-1", "user-2"));
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.role).toBe("editor");
    expect(requireBoardAccess).toHaveBeenCalledWith("user-1", "board-1", null, "owner");
  });

  it("returns 403 when trying to change board owner role", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireBoardAccess).mockResolvedValue(undefined);
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        org_id: null,
      }),
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ $id: "mem-1", user_id: "user-2", board_id: "board-1", role: "owner" }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/members/user-2", {
      method: "PATCH",
      body: JSON.stringify({ role: "editor" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await PATCH(request, makeMemberCtx("board-1", "user-2"));
    expect(response.status).toBe(403);
  });
});

describe("DELETE /api/boards/:id/members/:uid", () => {
  beforeEach(() => vi.clearAllMocks());

  it("allows self-removal with viewer role", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-2", email: "b@b.com", name: "Bob" });
    vi.mocked(requireBoardAccess).mockResolvedValue(undefined);
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        org_id: null,
      }),
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ $id: "mem-1", user_id: "user-2", board_id: "board-1", role: "viewer" }],
      }),
      deleteDocument: vi.fn().mockResolvedValue({}),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/members/user-2", {
      method: "DELETE",
    });
    const response = await DELETE(request, makeMemberCtx("board-1", "user-2"));
    expect(response.status).toBe(200);
    expect(requireBoardAccess).toHaveBeenCalledWith("user-2", "board-1", null, "viewer");
  });

  it("requires owner access to remove another member", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireBoardAccess).mockResolvedValue(undefined);
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        org_id: null,
      }),
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ $id: "mem-1", user_id: "user-2", board_id: "board-1", role: "editor" }],
      }),
      deleteDocument: vi.fn().mockResolvedValue({}),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/members/user-2", {
      method: "DELETE",
    });
    const response = await DELETE(request, makeMemberCtx("board-1", "user-2"));
    expect(response.status).toBe(200);
    expect(requireBoardAccess).toHaveBeenCalledWith("user-1", "board-1", null, "owner");
  });

  it("returns 403 when trying to remove board owner", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireBoardAccess).mockResolvedValue(undefined);
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "board-1",
        org_id: null,
      }),
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ $id: "mem-1", user_id: "user-2", board_id: "board-1", role: "owner" }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/members/user-2", {
      method: "DELETE",
    });
    const response = await DELETE(request, makeMemberCtx("board-1", "user-2"));
    expect(response.status).toBe(403);
  });
});
