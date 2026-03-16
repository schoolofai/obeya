import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({ authenticate: vi.fn() }));
vi.mock("@/lib/boards/permissions", () => ({ assertBoardAccess: vi.fn() }));
vi.mock("@/lib/appwrite/server", () => ({ getDatabases: vi.fn() }));
vi.mock("@/lib/env", () => ({ getEnv: () => ({ APPWRITE_DATABASE_ID: "obeya" }) }));
vi.mock("@/lib/history", () => ({ createHistoryEntry: vi.fn() }));

import { POST } from "@/app/api/boards/[id]/items/[ref]/review/route";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { createHistoryEntry } from "@/lib/history";

const mockUser = { id: "user-1", email: "a@b.com", name: "Alice" };

const itemDocWithContext = {
  $id: "item-1", board_id: "board-1", display_num: 34, type: "task",
  title: "Fix auth", description: "", status: "done", priority: "high",
  parent_id: null, assignee_id: "agent-1", blocked_by: "[]", tags: "[]",
  project: null, confidence: 45,
  review_context: JSON.stringify({ purpose: "Fix auth flow" }),
  human_review: JSON.stringify({ status: "pending" }),
  created_at: "2026-03-12T00:00:00Z", updated_at: "2026-03-12T00:00:00Z",
};

describe("POST /boards/:id/items/:ref/review", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("marks item as reviewed", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });
    const updatedDoc = {
      ...itemDocWithContext,
      human_review: JSON.stringify({ status: "reviewed", reviewed_by: "user-1" }),
    };
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: [itemDocWithContext], total: 1 }),
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/items/34/review", {
      method: "POST",
      body: JSON.stringify({ status: "reviewed" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, {
      params: Promise.resolve({ id: "board-1", ref: "34" }),
    });
    const result = await response.json();

    expect(response.status).toBe(200);
    expect(result.ok).toBe(true);
    expect(result.data.human_review.status).toBe("reviewed");
    expect(createHistoryEntry).toHaveBeenCalledWith(
      expect.objectContaining({ action: "human-review", detail: "reviewed" })
    );
  });

  it("marks item as hidden", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });
    const updatedDoc = {
      ...itemDocWithContext,
      human_review: JSON.stringify({ status: "hidden", reviewed_by: "user-1" }),
    };
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: [itemDocWithContext], total: 1 }),
      updateDocument: vi.fn().mockResolvedValue(updatedDoc),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/items/34/review", {
      method: "POST",
      body: JSON.stringify({ status: "hidden" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, {
      params: Promise.resolve({ id: "board-1", ref: "34" }),
    });

    expect(response.status).toBe(200);
  });

  it("rejects review of item without review_context", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });
    const itemWithoutContext = {
      ...itemDocWithContext,
      review_context: null,
    };
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: [itemWithoutContext], total: 1 }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/items/34/review", {
      method: "POST",
      body: JSON.stringify({ status: "reviewed" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, {
      params: Promise.resolve({ id: "board-1", ref: "34" }),
    });

    expect(response.status).toBe(400);
  });

  it("rejects invalid review status", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });

    const request = new Request("http://localhost/api/boards/board-1/items/34/review", {
      method: "POST",
      body: JSON.stringify({ status: "invalid" }),
      headers: { "Content-Type": "application/json" },
    });

    const response = await POST(request, {
      params: Promise.resolve({ id: "board-1", ref: "34" }),
    });

    expect(response.status).toBe(400);
  });
});
