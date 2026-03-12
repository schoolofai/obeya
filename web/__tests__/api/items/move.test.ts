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
vi.mock("node-appwrite", () => ({
  Query: {
    equal: vi.fn((field, value) => `${field}=${value}`),
    limit: vi.fn((n) => `limit=${n}`),
  },
}));

import { POST } from "@/app/api/items/[id]/move/route";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { createHistoryEntry } from "@/lib/history";

const itemDoc = {
  $id: "item-1",
  board_id: "board-1",
  status: "todo",
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

const boardDoc = {
  $id: "board-1",
  owner_id: "user-1",
  columns: JSON.stringify([
    { name: "todo", limit: 0 },
    { name: "in-progress", limit: 3 },
    { name: "done", limit: 0 },
  ]),
};

describe("POST /api/items/:id/move", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("moves item to target column and creates history entry", async () => {
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);
    const updatedDoc = { ...itemDoc, status: "in-progress" };
    const mockDb = {
      getDocument: vi.fn()
        .mockResolvedValueOnce(itemDoc)
        .mockResolvedValueOnce(boardDoc),
      listDocuments: vi.fn().mockResolvedValue({ total: 1, documents: [{}] }),
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1/move", {
      method: "POST",
      body: JSON.stringify({ status: "in-progress" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.status).toBe("in-progress");
    expect(createHistoryEntry).toHaveBeenCalledOnce();
    const historyCall = vi.mocked(createHistoryEntry).mock.calls[0][0];
    expect(historyCall.action).toBe("moved");
    expect(historyCall.detail).toBe("status: todo -> in-progress");
  });

  it("rejects move when WIP limit is reached", async () => {
    const wipBoardDoc = {
      ...boardDoc,
      columns: JSON.stringify([
        { name: "todo", limit: 0 },
        { name: "in-progress", limit: 2 },
        { name: "done", limit: 0 },
      ]),
    };
    vi.mocked(assertBoardAccess).mockResolvedValue(wipBoardDoc);
    const mockDb = {
      getDocument: vi.fn().mockResolvedValueOnce(itemDoc),
      listDocuments: vi.fn().mockResolvedValue({ total: 2, documents: [{}, {}] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1/move", {
      method: "POST",
      body: JSON.stringify({ status: "in-progress" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(400);
    expect(body.ok).toBe(false);
    expect(body.error.code).toBe("VALIDATION_ERROR");
  });

  it("rejects move to non-existent column", async () => {
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);
    const mockDb = {
      getDocument: vi.fn()
        .mockResolvedValueOnce(itemDoc)
        .mockResolvedValueOnce(boardDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1/move", {
      method: "POST",
      body: JSON.stringify({ status: "review" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(400);
    expect(body.ok).toBe(false);
  });
});
