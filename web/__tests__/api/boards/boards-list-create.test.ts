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

import { GET, POST } from "@/app/api/boards/route";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";

describe("GET /api/boards", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("returns boards owned by user and boards user is a member of", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    const mockDb = {
      listDocuments: vi.fn()
        .mockResolvedValueOnce({
          total: 1,
          documents: [{
            $id: "board-1", name: "My Board", owner_id: "user-1", org_id: null,
            display_counter: 3, columns: '[{"name":"todo","limit":0}]',
            display_map: "{}", users: "{}", projects: "{}",
            agent_role: "worker", version: 1,
            created_at: "2026-03-12T00:00:00.000Z", updated_at: "2026-03-12T00:00:00.000Z",
          }],
        })
        .mockResolvedValueOnce({ total: 0, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards");
    const response = await GET(request);
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(1);
    expect(body.data[0].name).toBe("My Board");
  });

  it("returns 401 when not authenticated", async () => {
    vi.mocked(authenticate).mockRejectedValue(new Error("No authentication"));
    const request = new Request("http://localhost/api/boards");
    const response = await GET(request);
    expect(response.status).toBe(500);
  });
});

describe("POST /api/boards", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("creates a board and returns it", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    const now = "2026-03-12T00:00:00.000Z";
    const mockDb = {
      createDocument: vi.fn().mockResolvedValue({
        $id: "board-new", name: "Sprint Board", owner_id: "user-1", org_id: null,
        display_counter: 0,
        columns: '[{"name":"todo","limit":0},{"name":"done","limit":0}]',
        display_map: "{}", users: "{}", projects: "{}",
        agent_role: "worker", version: 1, created_at: now, updated_at: now,
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards", {
      method: "POST",
      body: JSON.stringify({
        name: "Sprint Board",
        columns: [{ name: "todo", limit: 0 }, { name: "done", limit: 0 }],
      }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request);
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.ok).toBe(true);
    expect(body.data.name).toBe("Sprint Board");
    expect(body.data.id).toBe("board-new");
  });

  it("returns 400 for missing board name", async () => {
    vi.mocked(authenticate).mockResolvedValue({ id: "user-1", email: "a@b.com", name: "Alice" });
    const request = new Request("http://localhost/api/boards", {
      method: "POST",
      body: JSON.stringify({}),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request);
    expect(response.status).toBe(400);
  });
});
