import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth/middleware", () => ({ authenticate: vi.fn() }));
vi.mock("@/lib/boards/permissions", () => ({ assertBoardAccess: vi.fn() }));
vi.mock("@/lib/appwrite/server", () => ({ getDatabases: vi.fn() }));
vi.mock("@/lib/env", () => ({ getEnv: () => ({ APPWRITE_DATABASE_ID: "obeya" }) }));

import { GET } from "@/app/api/boards/[id]/review-queue/route";
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

describe("GET /boards/:id/review-queue", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("returns items sorted by confidence ascending", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });

    const docs = [
      makeItemDoc({
        $id: "high", display_num: 1, confidence: 90,
        review_context: JSON.stringify({ purpose: "High" }),
        human_review: JSON.stringify({ status: "pending" }),
      }),
      makeItemDoc({
        $id: "low", display_num: 2, confidence: 20,
        review_context: JSON.stringify({ purpose: "Low" }),
        human_review: JSON.stringify({ status: "pending" }),
      }),
    ];

    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: docs, total: docs.length }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/review-queue");
    const response = await GET(request as any, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.ok).toBe(true);
    expect(body.data.length).toBe(2);
    expect(body.data[0].confidence).toBe(20);
    expect(body.data[1].confidence).toBe(90);
  });

  it("excludes hidden items", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });

    const docs = [
      makeItemDoc({
        $id: "visible", display_num: 1, confidence: 50,
        review_context: JSON.stringify({ purpose: "Visible" }),
        human_review: JSON.stringify({ status: "pending" }),
      }),
      makeItemDoc({
        $id: "hidden", display_num: 2, confidence: 30,
        review_context: JSON.stringify({ purpose: "Hidden" }),
        human_review: JSON.stringify({ status: "hidden" }),
      }),
    ];

    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: docs, total: docs.length }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/review-queue");
    const response = await GET(request as any, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(body.data.length).toBe(1);
    expect(body.data[0].id).toBe("visible");
  });

  it("excludes items without review_context", async () => {
    vi.mocked(authenticate).mockResolvedValue(mockUser);
    vi.mocked(assertBoardAccess).mockResolvedValue({ $id: "board-1" });

    const docs = [
      makeItemDoc({
        $id: "with-ctx", display_num: 1, confidence: 50,
        review_context: JSON.stringify({ purpose: "Has context" }),
      }),
      makeItemDoc({ $id: "no-ctx", display_num: 2 }),
    ];

    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ documents: docs, total: docs.length }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const request = new Request("http://localhost/api/boards/board-1/review-queue");
    const response = await GET(request as any, {
      params: Promise.resolve({ id: "board-1" }),
    });
    const body = await response.json();

    expect(body.data.length).toBe(1);
    expect(body.data[0].id).toBe("with-ctx");
  });
});
