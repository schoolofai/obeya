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
  requireOrgRole: vi.fn(),
  ORG_ROLE_LEVEL: { member: 1, admin: 2, owner: 3 },
}));

vi.mock("@/lib/limits", () => ({
  enforceOrgMemberLimit: vi.fn(),
}));

import { GET, POST } from "@/app/api/orgs/[id]/members/route";
import { PATCH, DELETE } from "@/app/api/orgs/[id]/members/[uid]/route";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { requireOrgRole } from "@/lib/permissions";
import { enforceOrgMemberLimit } from "@/lib/limits";

type MembersContext = { params: Promise<{ id: string }> };
type MemberContext = { params: Promise<{ id: string; uid: string }> };

function makeMembersCtx(id: string): MembersContext {
  return { params: Promise.resolve({ id }) };
}

function makeMemberCtx(id: string, uid: string): MemberContext {
  return { params: Promise.resolve({ id, uid }) };
}

describe("GET /api/orgs/:id/members", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns list of org members", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockResolvedValue(undefined);
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        total: 2,
        documents: [
          { $id: "mem-1", user_id: "user-1", org_id: "org-1", role: "owner" },
          { $id: "mem-2", user_id: "user-2", org_id: "org-1", role: "member" },
        ],
      }),
      getDocument: vi.fn().mockResolvedValue({ plan: "free" }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs/org-1/members");
    const response = await GET(request, makeMembersCtx("org-1"));
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data).toHaveLength(2);
    expect(requireOrgRole).toHaveBeenCalledWith("user-1", "org-1", "member");
  });
});

describe("POST /api/orgs/:id/members", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(enforceOrgMemberLimit).mockResolvedValue(undefined);
  });

  it("adds a new member", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockResolvedValue(undefined);
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({ plan: "free" }),
      listDocuments: vi.fn().mockResolvedValue({ documents: [] }),
      createDocument: vi.fn().mockResolvedValue({
        $id: "mem-new",
        user_id: "user-2",
        org_id: "org-1",
        role: "member",
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs/org-1/members", {
      method: "POST",
      body: JSON.stringify({ user_id: "user-2", role: "member" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, makeMembersCtx("org-1"));
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.data.user_id).toBe("user-2");
    expect(requireOrgRole).toHaveBeenCalledWith("user-1", "org-1", "admin");
  });

  it("returns 409 when member already exists", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockResolvedValue(undefined);
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({ plan: "free" }),
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ $id: "mem-1", user_id: "user-2", org_id: "org-1", role: "member" }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs/org-1/members", {
      method: "POST",
      body: JSON.stringify({ user_id: "user-2", role: "member" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, makeMembersCtx("org-1"));
    expect(response.status).toBe(409);
  });

  it("returns 400 when role is invalid", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockResolvedValue(undefined);

    const request = new Request("http://localhost/api/orgs/org-1/members", {
      method: "POST",
      body: JSON.stringify({ user_id: "user-2", role: "owner" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, makeMembersCtx("org-1"));
    expect(response.status).toBe(400);
  });
});

describe("PATCH /api/orgs/:id/members/:uid", () => {
  beforeEach(() => vi.clearAllMocks());

  it("updates member role", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockResolvedValue(undefined);
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ $id: "mem-1", user_id: "user-2", org_id: "org-1", role: "member" }],
      }),
      updateDocument: vi.fn().mockResolvedValue({
        $id: "mem-1",
        user_id: "user-2",
        org_id: "org-1",
        role: "admin",
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs/org-1/members/user-2", {
      method: "PATCH",
      body: JSON.stringify({ role: "admin" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await PATCH(request, makeMemberCtx("org-1", "user-2"));
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.role).toBe("admin");
    expect(requireOrgRole).toHaveBeenCalledWith("user-1", "org-1", "owner");
  });

  it("returns 403 when trying to change owner role", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockResolvedValue(undefined);
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ $id: "mem-1", user_id: "user-2", org_id: "org-1", role: "owner" }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs/org-1/members/user-2", {
      method: "PATCH",
      body: JSON.stringify({ role: "admin" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await PATCH(request, makeMemberCtx("org-1", "user-2"));
    expect(response.status).toBe(403);
  });
});

describe("DELETE /api/orgs/:id/members/:uid", () => {
  beforeEach(() => vi.clearAllMocks());

  it("allows self-removal with member role", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-2", email: "b@b.com", name: "Bob" });
    vi.mocked(requireOrgRole).mockResolvedValue(undefined);
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ $id: "mem-1", user_id: "user-2", org_id: "org-1", role: "member" }],
      }),
      deleteDocument: vi.fn().mockResolvedValue({}),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs/org-1/members/user-2", {
      method: "DELETE",
    });
    const response = await DELETE(request, makeMemberCtx("org-1", "user-2"));
    expect(response.status).toBe(200);
    expect(requireOrgRole).toHaveBeenCalledWith("user-2", "org-1", "member");
  });

  it("requires admin role to remove another member", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockResolvedValue(undefined);
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ $id: "mem-1", user_id: "user-2", org_id: "org-1", role: "member" }],
      }),
      deleteDocument: vi.fn().mockResolvedValue({}),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs/org-1/members/user-2", {
      method: "DELETE",
    });
    const response = await DELETE(request, makeMemberCtx("org-1", "user-2"));
    expect(response.status).toBe(200);
    expect(requireOrgRole).toHaveBeenCalledWith("user-1", "org-1", "admin");
  });

  it("returns 403 when trying to remove owner", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockResolvedValue(undefined);
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        documents: [{ $id: "mem-1", user_id: "user-2", org_id: "org-1", role: "owner" }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs/org-1/members/user-2", {
      method: "DELETE",
    });
    const response = await DELETE(request, makeMemberCtx("org-1", "user-2"));
    expect(response.status).toBe(403);
  });
});
