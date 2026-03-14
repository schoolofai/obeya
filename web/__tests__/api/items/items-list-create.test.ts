import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({ authenticate: vi.fn() }));
vi.mock("@/lib/boards/permissions", () => ({ assertBoardAccess: vi.fn() }));
vi.mock("@/lib/boards/counter", () => ({ incrementDisplayCounter: vi.fn() }));
vi.mock("@/lib/appwrite/server", () => ({ getDatabases: vi.fn() }));
vi.mock("@/lib/env", () => ({ getEnv: () => ({ APPWRITE_DATABASE_ID: "obeya" }) }));

import { GET, POST } from "@/app/api/boards/[id]/items/route";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { incrementDisplayCounter } from "@/lib/boards/counter";
import { getDatabases } from "@/lib/appwrite/server";

const mockUser = { id: "user-1", email: "a@b.com", name: "Alice" };
const boardDoc = { $id: "board-1", owner_id: "user-1" };

describe("GET /api/boards/:id/items", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("returns all items for the board", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({
        total: 1,
        documents: [{
          $id: "item-1", board_id: "board-1", display_num: 1, type: "task",
          title: "First task", description: "", status: "todo", priority: "medium",
          parent_id: null, assignee_id: null, blocked_by: "[]", tags: "[]",
          project: null, created_at: "2026-03-12T00:00:00.000Z", updated_at: "2026-03-12T00:00:00.000Z",
        }],
      }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/items");
    const response = await GET(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data).toHaveLength(1);
    expect(body.data[0].id).toBe("item-1");
    expect(body.data[0].blocked_by).toEqual([]);
  });

  it("filters items by status query param", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);
    const mockDb = { listDocuments: vi.fn().mockResolvedValue({ total: 0, documents: [] }) };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/items?status=done");
    const response = await GET(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    const callArgs = mockDb.listDocuments.mock.calls[0][2];
    const hasStatusFilter = callArgs.some((q: any) => JSON.stringify(q).includes("done"));
    expect(hasStatusFilter).toBe(true);
  });
});

describe("POST /api/boards/:id/items", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("creates an item with incremented display counter", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);
    vi.mocked(incrementDisplayCounter).mockResolvedValue(5);

    const now = "2026-03-12T00:00:00.000Z";
    const mockDb = {
      createDocument: vi.fn().mockResolvedValue({
        $id: "item-new", board_id: "board-1", display_num: 5, type: "task",
        title: "New task", description: "", status: "todo", priority: "medium",
        parent_id: null, assignee_id: null, blocked_by: "[]", tags: "[]",
        project: null, created_at: now, updated_at: now,
      }),
      getDocument: vi.fn().mockResolvedValue({ $id: "board-1", owner_id: "user-1", display_map: "{}" }),
      listDocuments: vi.fn().mockResolvedValue({ total: 0, documents: [] }),
      updateDocument: vi.fn().mockResolvedValue({}),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/items", {
      method: "POST",
      body: JSON.stringify({ type: "task", title: "New task", status: "todo" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "board-1" }) });
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body.ok).toBe(true);
    expect(body.data.display_num).toBe(5);
    expect(body.data.id).toBe("item-new");
    expect(incrementDisplayCounter).toHaveBeenCalledWith("board-1");

    // Verify permissions were passed to createDocument
    const createArgs = mockDb.createDocument.mock.calls[0];
    const permissions = createArgs[4];
    expect(permissions).toContain('read("user:user-1")');
    expect(permissions).toContain('update("user:user-1")');
    expect(permissions).toContain('delete("user:user-1")');
  });

  it("returns 400 for missing required fields", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);

    const request = new Request("http://localhost/api/boards/board-1/items", {
      method: "POST",
      body: JSON.stringify({ description: "no title or type" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, { params: Promise.resolve({ id: "board-1" }) });
    expect(response.status).toBe(400);
  });
});
