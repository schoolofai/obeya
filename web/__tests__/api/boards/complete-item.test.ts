import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({ authenticate: vi.fn() }));
vi.mock("@/lib/boards/permissions", () => ({ assertBoardAccess: vi.fn() }));
vi.mock("@/lib/appwrite/server", () => ({ getDatabases: vi.fn() }));
vi.mock("@/lib/env", () => ({ getEnv: () => ({ APPWRITE_DATABASE_ID: "obeya" }) }));
vi.mock("@/lib/history", () => ({ createHistoryEntry: vi.fn() }));

import { POST } from "@/app/api/boards/[id]/items/[ref]/complete/route";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { createHistoryEntry } from "@/lib/history";

const mockUser = { id: "user-1", email: "a@b.com", name: "Alice" };

const itemDoc = {
  $id: "item-1", board_id: "board-1", display_num: 34, type: "task",
  title: "Fix auth", description: "", status: "in-progress", priority: "high",
  parent_id: null, assignee_id: "agent-1", blocked_by: "[]", tags: "[]",
  project: null, created_at: "2026-03-12T00:00:00Z", updated_at: "2026-03-12T00:00:00Z",
};

describe("POST /boards/:id/items/:ref/complete", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("completes item with review context and sets confidence", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });
    const updatedDoc = {
      ...itemDoc,
      status: "done",
      confidence: 45,
      review_context: JSON.stringify({ purpose: "Fix auth flow" }),
      human_review: JSON.stringify({ status: "pending" }),
    };
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: [itemDoc], total: 1 }),
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const body = {
      confidence: 45,
      review_context: {
        purpose: "Fix auth flow",
        files_changed: [{ path: "auth.go", added: 10, removed: 5 }],
      },
    };

    const request = new Request("http://localhost/api/boards/board-1/items/34/complete", {
      method: "POST",
      body: JSON.stringify(body),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, {
      params: Promise.resolve({ id: "board-1", ref: "34" }),
    });
    const result = await response.json();

    expect(response.status).toBe(200);
    expect(result.ok).toBe(true);
    expect(result.data.status).toBe("done");
    expect(mockDb.updateDocument).toHaveBeenCalled();
    expect(createHistoryEntry).toHaveBeenCalledWith(
      expect.objectContaining({ action: "complete-with-context" })
    );
  });

  it("returns 400 when confidence is missing", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });

    const request = new Request("http://localhost/api/boards/board-1/items/34/complete", {
      method: "POST",
      body: JSON.stringify({ review_context: { purpose: "Test" } }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, {
      params: Promise.resolve({ id: "board-1", ref: "34" }),
    });

    expect(response.status).toBe(400);
  });

  it("returns 400 when purpose is missing from review_context", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });

    const request = new Request("http://localhost/api/boards/board-1/items/34/complete", {
      method: "POST",
      body: JSON.stringify({ confidence: 50, review_context: {} }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, {
      params: Promise.resolve({ id: "board-1", ref: "34" }),
    });

    expect(response.status).toBe(400);
  });

  it("returns 404 when item not found", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: [], total: 0 }),
      getDocument: vi.fn().mockRejectedValue(new Error("Not found")),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/items/999/complete", {
      method: "POST",
      body: JSON.stringify({ confidence: 50, review_context: { purpose: "Test" } }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, {
      params: Promise.resolve({ id: "board-1", ref: "999" }),
    });

    expect(response.status).toBe(404);
  });
});
