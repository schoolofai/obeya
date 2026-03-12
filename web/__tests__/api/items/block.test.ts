import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({
  authenticate: vi.fn().mockResolvedValue({ id: "user-1", email: "u@e.com", name: "U" }),
}));
vi.mock("@/lib/boards/permissions", () => ({ assertBoardAccess: vi.fn() }));
vi.mock("@/lib/appwrite/server", () => ({ getDatabases: vi.fn() }));
vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_DATABASE_ID: "obeya",
    APPWRITE_ENDPOINT: "https://cloud.appwrite.io/v1",
    APPWRITE_PROJECT_ID: "proj-1",
    APPWRITE_API_KEY: "key-1",
  }),
}));
vi.mock("@/lib/history", () => ({
  createHistoryEntry: vi.fn().mockResolvedValue(undefined),
}));

import { POST } from "@/app/api/items/[id]/block/route";
import { DELETE } from "@/app/api/items/[id]/block/[bid]/route";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { createHistoryEntry } from "@/lib/history";

const boardDoc = { $id: "board-1", owner_id: "user-1" };

const baseItemDoc = {
  $id: "item-1",
  board_id: "board-1",
  status: "in-progress",
  display_num: 1,
  type: "task",
  title: "Some task",
  description: "",
  priority: "medium",
  parent_id: null,
  assignee_id: null,
  blocked_by: "[]",
  tags: "[]",
  project: null,
  created_at: "2026-03-12T00:00:00.000Z",
  updated_at: "2026-03-12T00:00:00.000Z",
};

describe("POST /api/items/:id/block", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("adds a blocker to empty blocked_by and creates history", async () => {
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);
    const updatedDoc = { ...baseItemDoc, blocked_by: '["item-2"]' };
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(baseItemDoc),
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1/block", {
      method: "POST",
      body: JSON.stringify({ blocked_by_id: "item-2" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.blocked_by).toEqual(["item-2"]);
    expect(mockDb.updateDocument.mock.calls[0][3].blocked_by).toBe('["item-2"]');
    expect(createHistoryEntry).toHaveBeenCalledOnce();
    const historyCall = vi.mocked(createHistoryEntry).mock.calls[0][0];
    expect(historyCall.action).toBe("blocked");
    expect(historyCall.detail).toContain("item-2");
  });

  it("appends blocker to existing blocked_by without duplicates", async () => {
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);
    const itemWithBlocker = { ...baseItemDoc, blocked_by: '["item-3"]' };
    const updatedDoc = { ...baseItemDoc, blocked_by: '["item-3","item-4"]' };
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(itemWithBlocker),
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1/block", {
      method: "POST",
      body: JSON.stringify({ blocked_by_id: "item-4" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.data.blocked_by).toEqual(["item-3", "item-4"]);
    const updateCall = mockDb.updateDocument.mock.calls[0][3];
    expect(updateCall.blocked_by).toBe('["item-3","item-4"]');
  });

  it("rejects duplicate blocker with 400", async () => {
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);
    const itemWithBlocker = { ...baseItemDoc, blocked_by: '["item-2"]' };
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(itemWithBlocker),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1/block", {
      method: "POST",
      body: JSON.stringify({ blocked_by_id: "item-2" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(400);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe("VALIDATION_ERROR");
  });
});

describe("DELETE /api/items/:id/block/:bid", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("removes a blocker and creates history entry", async () => {
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);
    const itemWithBlocker = { ...baseItemDoc, blocked_by: '["item-2","item-3"]' };
    const updatedDoc = { ...baseItemDoc, blocked_by: '["item-3"]' };
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(itemWithBlocker),
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1/block/item-2", {
      method: "DELETE",
    });
    const response = await DELETE(request, {
      params: Promise.resolve({ id: "item-1", bid: "item-2" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.blocked_by).toEqual(["item-3"]);
    const updateCall = mockDb.updateDocument.mock.calls[0][3];
    expect(updateCall.blocked_by).toBe('["item-3"]');
    expect(createHistoryEntry).toHaveBeenCalledOnce();
    const historyCall = vi.mocked(createHistoryEntry).mock.calls[0][0];
    expect(historyCall.action).toBe("unblocked");
    expect(historyCall.detail).toContain("item-2");
  });

  it("returns 404 when blocker ID is not in the array", async () => {
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);
    const itemWithBlocker = { ...baseItemDoc, blocked_by: '["item-3"]' };
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(itemWithBlocker),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1/block/item-2", {
      method: "DELETE",
    });
    const response = await DELETE(request, {
      params: Promise.resolve({ id: "item-1", bid: "item-2" }),
    });
    const body = await response.json();

    expect(response.status).toBe(404);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe("ITEM_NOT_FOUND");
  });
});
