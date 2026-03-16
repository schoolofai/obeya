import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({ authenticate: vi.fn() }));
vi.mock("@/lib/boards/permissions", () => ({ assertBoardAccess: vi.fn() }));
vi.mock("@/lib/appwrite/server", () => ({ getDatabases: vi.fn() }));
vi.mock("@/lib/env", () => ({ getEnv: () => ({ APPWRITE_DATABASE_ID: "obeya" }) }));

import { GET } from "@/app/api/boards/[id]/past-reviews/route";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";

const mockUser = { id: "user-1", email: "a@b.com", name: "Alice" };

function makeItemDoc(overrides: Record<string, unknown> = {}) {
  return {
    $id: "item-1", board_id: "board-1", display_num: 1, type: "task",
    title: "Test", description: "", status: "done", priority: "medium",
    parent_id: null, assignee_id: null, blocked_by: "[]", tags: "[]",
    project: null, created_at: "2026-03-10T00:00:00Z", updated_at: "2026-03-10T00:00:00Z",
    ...overrides,
  };
}

describe("GET /boards/:id/past-reviews", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("returns reviewed items with ancestors", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });

    const docs = [
      makeItemDoc({ $id: "epic-1", display_num: 10, type: "epic", title: "Auth Rewrite" }),
      makeItemDoc({ $id: "task-1", display_num: 34, parent_id: "epic-1", title: "Fix middleware",
        review_context: JSON.stringify({ purpose: "Fix" }),
        human_review: JSON.stringify({ status: "reviewed", reviewed_by: "user-1" }),
      }),
      makeItemDoc({ $id: "task-2", display_num: 35, title: "Unrelated task" }),
    ];

    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: docs, total: docs.length }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/past-reviews");
    const response = await GET(request as any, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.length).toBe(2);

    const ids = body.data.map((i: any) => i.id);
    expect(ids).toContain("epic-1");
    expect(ids).toContain("task-1");
    expect(ids).not.toContain("task-2");
  });

  it("returns empty array when no items are reviewed", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });

    const docs = [
      makeItemDoc({ $id: "task-1", display_num: 1, title: "Not reviewed" }),
    ];

    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: docs, total: docs.length }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/past-reviews");
    const response = await GET(request as any, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(body.data.length).toBe(0);
  });

  it("includes both reviewed and hidden items", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });

    const docs = [
      makeItemDoc({
        $id: "reviewed", display_num: 1,
        review_context: JSON.stringify({ purpose: "A" }),
        human_review: JSON.stringify({ status: "reviewed" }),
      }),
      makeItemDoc({
        $id: "hidden", display_num: 2,
        review_context: JSON.stringify({ purpose: "B" }),
        human_review: JSON.stringify({ status: "hidden" }),
      }),
    ];

    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: docs, total: docs.length }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/past-reviews");
    const response = await GET(request as any, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(body.data.length).toBe(2);
  });
});
