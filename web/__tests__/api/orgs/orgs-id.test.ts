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

import { GET, PATCH, DELETE } from "@/app/api/orgs/[id]/route";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { requireOrgRole } from "@/lib/permissions";

type RouteContext = { params: Promise<{ id: string }> };

function makeContext(id: string): RouteContext {
  return { params: Promise.resolve({ id }) };
}

describe("GET /api/orgs/:id", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns org with member count", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockResolvedValue(undefined);
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue({
        $id: "org-1",
        name: "Acme Corp",
        slug: "acme-corp",
        plan: "free",
        owner_id: "user-1",
        created_at: "2026-03-12T00:00:00.000Z",
      }),
      listDocuments: vi.fn().mockResolvedValue({ total: 5, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs/org-1");
    const response = await GET(request, makeContext("org-1"));
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.id).toBe("org-1");
    expect(body.data.member_count).toBe(5);
  });

  it("returns 500 when not a member (regular Error from mock)", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockRejectedValue(new Error("Not a member"));

    const request = new Request("http://localhost/api/orgs/org-1");
    const response = await GET(request, makeContext("org-1"));
    expect(response.status).toBe(500);
  });
});

describe("PATCH /api/orgs/:id", () => {
  beforeEach(() => vi.clearAllMocks());

  it("updates org name", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockResolvedValue(undefined);
    const mockDb = {
      updateDocument: vi.fn().mockResolvedValue({
        $id: "org-1",
        name: "Updated Org",
        slug: "acme-corp",
        plan: "free",
        owner_id: "user-1",
        created_at: "2026-03-12T00:00:00.000Z",
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs/org-1", {
      method: "PATCH",
      body: JSON.stringify({ name: "Updated Org" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await PATCH(request, makeContext("org-1"));
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.name).toBe("Updated Org");
    expect(requireOrgRole).toHaveBeenCalledWith("user-1", "org-1", "admin");
  });

  it("returns 400 when no fields provided", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockResolvedValue(undefined);

    const request = new Request("http://localhost/api/orgs/org-1", {
      method: "PATCH",
      body: JSON.stringify({}),
      headers: { "Content-Type": "application/json" },
    });
    const response = await PATCH(request, makeContext("org-1"));
    expect(response.status).toBe(400);
  });

  it("requires admin role", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockRejectedValue(new Error("Insufficient role"));

    const request = new Request("http://localhost/api/orgs/org-1", {
      method: "PATCH",
      body: JSON.stringify({ name: "Updated" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await PATCH(request, makeContext("org-1"));
    expect(response.status).toBe(500);
  });
});

describe("DELETE /api/orgs/:id", () => {
  beforeEach(() => vi.clearAllMocks());

  it("deletes org and all its members", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockResolvedValue(undefined);
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        total: 2,
        documents: [
          { $id: "mem-1" },
          { $id: "mem-2" },
        ],
      }),
      deleteDocument: vi.fn().mockResolvedValue({}),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs/org-1", { method: "DELETE" });
    const response = await DELETE(request, makeContext("org-1"));
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.deleted).toBe(true);
    expect(requireOrgRole).toHaveBeenCalledWith("user-1", "org-1", "owner");
    expect(mockDb.deleteDocument).toHaveBeenCalledTimes(3); // 2 members + 1 org
  });

  it("requires owner role", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(requireOrgRole).mockRejectedValue(new Error("Not owner"));

    const request = new Request("http://localhost/api/orgs/org-1", { method: "DELETE" });
    const response = await DELETE(request, makeContext("org-1"));
    expect(response.status).toBe(500);
  });
});
