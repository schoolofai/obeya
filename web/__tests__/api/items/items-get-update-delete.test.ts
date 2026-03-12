import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({ authenticate: vi.fn() }));
vi.mock("@/lib/boards/permissions", () => ({ assertBoardAccess: vi.fn() }));
vi.mock("@/lib/appwrite/server", () => ({ getDatabases: vi.fn() }));
vi.mock("@/lib/env", () => ({ getEnv: () => ({ APPWRITE_DATABASE_ID: "obeya" }) }));

import { GET, PATCH, DELETE } from "@/app/api/items/[id]/route";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";

const mockUser = { id: "user-1", email: "a@b.com", name: "Alice" };
const itemDoc = {
  $id: "item-1", board_id: "board-1", display_num: 3, type: "task",
  title: "Fix login", description: "Users can't log in",
  status: "in-progress", priority: "high", parent_id: null,
  assignee_id: "user-1", blocked_by: '["item-2"]', tags: '["bug"]',
  project: "web", created_at: "2026-03-12T00:00:00.000Z", updated_at: "2026-03-12T00:00:00.000Z",
};

describe("GET /api/items/:id", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("returns item with deserialized JSON fields", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });
    const mockDb = { getDocument: vi.fn().mockResolvedValue(itemDoc) };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1");
    const response = await GET(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.id).toBe("item-1");
    expect(body.data.blocked_by).toEqual(["item-2"]);
    expect(body.data.tags).toEqual(["bug"]);
  });

  it("returns 404 for non-existent item", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    const notFound = new Error("Document not found");
    (notFound as any).code = 404;
    const mockDb = { getDocument: vi.fn().mockRejectedValue(notFound) };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/missing");
    const response = await GET(request, { params: Promise.resolve({ id: "missing" }) });
    expect(response.status).toBe(404);
  });
});

describe("PATCH /api/items/:id", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("updates item title and returns updated item", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });
    const updatedDoc = { ...itemDoc, title: "Fix auth flow" };
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(itemDoc),
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1", {
      method: "PATCH",
      body: JSON.stringify({ title: "Fix auth flow" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await PATCH(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.title).toBe("Fix auth flow");
  });

  it("updates tags as serialized JSON string", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });
    const updatedDoc = { ...itemDoc, tags: '["bug","critical"]' };
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(itemDoc),
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1", {
      method: "PATCH",
      body: JSON.stringify({ tags: ["bug", "critical"] }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await PATCH(request, { params: Promise.resolve({ id: "item-1" }) });
    const updateCall = mockDb.updateDocument.mock.calls[0];
    expect(updateCall[3].tags).toBe('["bug","critical"]');
  });
});

describe("DELETE /api/items/:id", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("deletes item and returns confirmation", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(itemDoc),
      deleteDocument: vi.fn().mockResolvedValue(undefined),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1", { method: "DELETE" });
    const response = await DELETE(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.deleted).toBe(true);
  });
});
