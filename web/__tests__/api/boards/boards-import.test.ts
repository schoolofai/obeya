import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({ authenticate: vi.fn() }));
vi.mock("@/lib/appwrite/server", () => ({ getDatabases: vi.fn() }));
vi.mock("@/lib/env", () => ({
  getEnv: () => ({ APPWRITE_DATABASE_ID: "obeya" }),
}));

import { POST } from "@/app/api/boards/import/route";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";

const mockUser = { id: "user-1", email: "a@b.com", name: "Alice" };

describe("POST /api/boards/import", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("imports a local board.json and returns the new board with ID mapping", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    let docCounter = 0;
    const mockDb = {
      createDocument: vi.fn().mockImplementation(() => {
        docCounter++;
        return Promise.resolve({
          $id: `cloud-${docCounter}`,
          name: "Imported Board", owner_id: "user-1", org_id: null,
          display_counter: 2,
          columns: '[{"name":"todo","limit":0},{"name":"done","limit":0}]',
          display_map: '{"1":"cloud-2","2":"cloud-3"}',
          users: "{}", projects: "{}", agent_role: "worker", version: 1,
          created_at: "2026-03-12T00:00:00.000Z", updated_at: "2026-03-12T00:00:00.000Z",
        });
      }),
      updateDocument: vi.fn().mockResolvedValue({}),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const localBoard = {
      name: "My Local Board", display_counter: 2,
      columns: [{ name: "todo", limit: 0 }, { name: "done", limit: 0 }],
      display_map: { "1": "local-item-1", "2": "local-item-2" },
      users: {}, projects: {}, agent_role: "worker", version: 1,
      items: [
        { id: "local-item-1", display_num: 1, type: "task", title: "First task",
          description: "", status: "todo", priority: "medium",
          parent_id: null, assignee_id: null, blocked_by: [], tags: [], project: null },
        { id: "local-item-2", display_num: 2, type: "task", title: "Second task",
          description: "", status: "done", priority: "low",
          parent_id: "local-item-1", blocked_by: ["local-item-1"], tags: ["bug"], project: null },
      ],
    };

    const request = new Request("http://localhost/api/boards/import", {
      method: "POST",
      body: JSON.stringify(localBoard),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request);
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.ok).toBe(true);
    expect(body.data.board_id).toBeDefined();
    expect(body.data.id_map).toBeDefined();
    expect(body.data.items_imported).toBe(2);
  });

  it("returns 400 for invalid board payload", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    const request = new Request("http://localhost/api/boards/import", {
      method: "POST",
      body: JSON.stringify({ invalid: true }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request);
    expect(response.status).toBe(400);
  });
});
