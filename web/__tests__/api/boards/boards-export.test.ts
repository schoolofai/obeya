import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({ authenticate: vi.fn() }));
vi.mock("@/lib/boards/permissions", () => ({ assertBoardAccess: vi.fn() }));
vi.mock("@/lib/appwrite/server", () => ({ getDatabases: vi.fn() }));
vi.mock("@/lib/env", () => ({
  getEnv: () => ({ APPWRITE_DATABASE_ID: "obeya" }),
}));

import { GET } from "@/app/api/boards/[id]/export/route";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";

const mockUser = { id: "user-1", email: "a@b.com", name: "Alice" };

describe("GET /api/boards/:id/export", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("exports board in local board.json format with items", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({
      $id: "board-1", name: "My Board", owner_id: "user-1", org_id: null,
      display_counter: 2,
      columns: '[{"name":"todo","limit":0},{"name":"done","limit":0}]',
      display_map: '{"1":"item-1","2":"item-2"}',
      users: '{}', projects: '{}', agent_role: "worker", version: 1,
      created_at: "2026-03-12T00:00:00.000Z", updated_at: "2026-03-12T00:00:00.000Z",
    });

    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        total: 2,
        documents: [
          { $id: "item-1", board_id: "board-1", display_num: 1, type: "task",
            title: "First task", description: "", status: "todo", priority: "medium",
            parent_id: null, assignee_id: null, blocked_by: "[]",
            tags: '["important"]', project: null,
            created_at: "2026-03-12T00:00:00.000Z", updated_at: "2026-03-12T00:00:00.000Z" },
          { $id: "item-2", board_id: "board-1", display_num: 2, type: "task",
            title: "Second task", description: "desc", status: "done", priority: "low",
            parent_id: "item-1", assignee_id: null, blocked_by: '["item-1"]',
            tags: "[]", project: null,
            created_at: "2026-03-12T00:00:00.000Z", updated_at: "2026-03-12T00:00:00.000Z" },
        ],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/export");
    const response = await GET(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.name).toBe("My Board");
    expect(body.data.display_counter).toBe(2);
    expect(body.data.columns).toEqual([{ name: "todo", limit: 0 }, { name: "done", limit: 0 }]);
    expect(body.data.items).toHaveLength(2);
    expect(body.data.items[0].id).toBe("item-1");
    expect(body.data.items[0].tags).toEqual(["important"]);
    expect(body.data.items[1].blocked_by).toEqual(["item-1"]);
  });

  it("exports board with empty items list when no items exist", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({
      $id: "board-1", name: "Empty Board", owner_id: "user-1", org_id: null,
      display_counter: 0, columns: '[{"name":"todo","limit":0}]',
      display_map: '{}', users: '{}', projects: '{}',
      agent_role: "worker", version: 1,
      created_at: "2026-03-12T00:00:00.000Z", updated_at: "2026-03-12T00:00:00.000Z",
    });
    const mockDb = { listDocuments: vi.fn().mockResolvedValue({ total: 0, documents: [] }) };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/export");
    const response = await GET(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.items).toEqual([]);
  });
});
