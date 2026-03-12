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

vi.mock("@/lib/slugs", () => ({
  generateSlug: vi.fn().mockReturnValue("my-org"),
  ensureUniqueSlug: vi.fn().mockResolvedValue("my-org"),
}));

vi.mock("@/lib/limits", () => ({
  enforceOrgLimit: vi.fn(),
}));

import { GET, POST } from "@/app/api/orgs/route";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { enforceOrgLimit } from "@/lib/limits";
import { generateSlug, ensureUniqueSlug } from "@/lib/slugs";

describe("GET /api/orgs", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns list of user's orgs with their role", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        total: 1,
        documents: [{ $id: "mem-1", user_id: "user-1", org_id: "org-1", role: "owner" }],
      }),
      getDocument: vi.fn().mockResolvedValue({
        $id: "org-1",
        name: "Acme Corp",
        slug: "acme-corp",
        plan: "free",
        owner_id: "user-1",
        created_at: "2026-03-12T00:00:00.000Z",
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs");
    const response = await GET(request);
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(1);
    expect(body.data[0].id).toBe("org-1");
    expect(body.data[0].name).toBe("Acme Corp");
    expect(body.data[0].role).toBe("owner");
  });

  it("returns empty array when user has no orgs", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 0, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs");
    const response = await GET(request);
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data).toHaveLength(0);
  });

  it("returns 401 when not authenticated", async () => {
    vi.mocked(authenticate).mockRejectedValue(new Error("No auth"));

    const request = new Request("http://localhost/api/orgs");
    const response = await GET(request);
    expect(response.status).toBe(500);
  });
});

describe("POST /api/orgs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(generateSlug).mockReturnValue("my-org");
    vi.mocked(ensureUniqueSlug).mockResolvedValue("my-org");
    vi.mocked(enforceOrgLimit).mockResolvedValue(undefined);
  });

  it("creates an org and returns 201", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    const mockDb = {
      createDocument: vi.fn()
        .mockResolvedValueOnce({
          $id: "org-new",
          name: "My Org",
          slug: "my-org",
          plan: "free",
          owner_id: "user-1",
          created_at: "2026-03-12T00:00:00.000Z",
        })
        .mockResolvedValueOnce({
          $id: "mem-new",
          user_id: "user-1",
          org_id: "org-new",
          role: "owner",
        }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs", {
      method: "POST",
      body: JSON.stringify({ name: "My Org" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request);
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.ok).toBe(true);
    expect(body.data.id).toBe("org-new");
    expect(body.data.slug).toBe("my-org");
  });

  it("uses provided slug when specified", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(generateSlug).mockReturnValue("custom-slug");
    vi.mocked(ensureUniqueSlug).mockResolvedValue("custom-slug");

    const mockDb = {
      createDocument: vi.fn()
        .mockResolvedValueOnce({
          $id: "org-new",
          name: "My Org",
          slug: "custom-slug",
          plan: "free",
          owner_id: "user-1",
          created_at: "2026-03-12T00:00:00.000Z",
        })
        .mockResolvedValueOnce({ $id: "mem-new" }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/orgs", {
      method: "POST",
      body: JSON.stringify({ name: "My Org", slug: "custom-slug" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request);
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(ensureUniqueSlug).toHaveBeenCalledWith("custom-slug");
  });

  it("returns 400 when name is missing", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });

    const request = new Request("http://localhost/api/orgs", {
      method: "POST",
      body: JSON.stringify({}),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request);
    expect(response.status).toBe(400);
  });

  it("returns 500 when org limit is enforced (enforceOrgLimit throws regular Error)", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    vi.mocked(enforceOrgLimit).mockRejectedValue(new Error("limit reached"));

    const request = new Request("http://localhost/api/orgs", {
      method: "POST",
      body: JSON.stringify({ name: "My Org" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request);
    expect(response.status).toBe(500);
  });
});
