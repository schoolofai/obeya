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

import { POST } from "@/app/api/items/[id]/assign/route";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { createHistoryEntry } from "@/lib/history";

const boardDoc = { $id: "board-1", owner_id: "user-1" };

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

describe("POST /api/items/:id/assign", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("assigns a user and creates history entry", async () => {
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);
    const updatedDoc = { ...itemDoc, assignee_id: "user-2" };
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(itemDoc),
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1/assign", {
      method: "POST",
      body: JSON.stringify({ assignee_id: "user-2" }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.assignee_id).toBe("user-2");
    expect(createHistoryEntry).toHaveBeenCalledOnce();
    const historyCall = vi.mocked(createHistoryEntry).mock.calls[0][0];
    expect(historyCall.action).toBe("assigned");
    expect(historyCall.detail).toContain("user-2");
  });

  it("unassigns by passing null and creates history entry", async () => {
    vi.mocked(assertBoardAccess).mockResolvedValue(boardDoc);
    const assignedItem = { ...itemDoc, assignee_id: "user-2" };
    const updatedDoc = { ...itemDoc, assignee_id: null };
    const mockDb = {
      getDocument: vi.fn().mockResolvedValue(assignedItem),
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/items/item-1/assign", {
      method: "POST",
      body: JSON.stringify({ assignee_id: null }),
      headers: { "Content-Type": "application/json" },
    });
    const response = await POST(request, { params: Promise.resolve({ id: "item-1" }) });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.assignee_id).toBeNull();
    expect(createHistoryEntry).toHaveBeenCalledOnce();
    const historyCall = vi.mocked(createHistoryEntry).mock.calls[0][0];
    expect(historyCall.action).toBe("assigned");
    expect(historyCall.detail).toContain("unassigned");
  });
});
